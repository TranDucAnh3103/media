package telegram

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

// TelegramClient - Client kết nối với Telegram qua MTProto
type TelegramClient struct {
	client      *telegram.Client
	api         *tg.Client
	channelID   int64
	accessHash  int64
	mu          sync.RWMutex
	connected   bool
	config      Config
	sessionPath string

	circuitBreaker *CircuitBreaker

	// Persistent connection fields
	persistentCtx    context.Context
	persistentCancel context.CancelFunc
	workChan         chan workRequest
	persistentReady  chan struct{}
	persistentErr    error
}

// SetCircuitBreaker sets the circuit breaker for this client
func (c *TelegramClient) SetCircuitBreaker(cb *CircuitBreaker) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.circuitBreaker = cb
}

// workRequest - Request struct for persistent connection work
type workRequest struct {
	ctx      context.Context
	work     func(context.Context) error
	resultCh chan error
}

// NewTelegramClient - Tạo instance mới của TelegramClient
func NewTelegramClient() (*TelegramClient, error) {
	// Load config từ environment variables
	config, err := loadConfig()
	if err != nil {
		return nil, err
	}

	// Tạo session storage
	sessionDir := config.SessionPath
	if sessionDir == "" {
		sessionDir = "./telegram_session"
	}

	// Đảm bảo thư mục session tồn tại
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}

	sessionStorage := &session.FileStorage{
		Path: filepath.Join(sessionDir, "session.json"),
	}

	// Tạo Telegram client
	client := telegram.NewClient(config.APIID, config.APIHash, telegram.Options{
		SessionStorage: sessionStorage,
	})

	return &TelegramClient{
		client:      client,
		channelID:   config.ChannelID,
		config:      config,
		sessionPath: sessionDir,
	}, nil
}

// loadConfig - Load cấu hình từ environment variables
func loadConfig() (Config, error) {
	apiIDStr := os.Getenv("TELEGRAM_API_ID")
	if apiIDStr == "" {
		return Config{}, ErrMissingAPIID
	}

	apiID, err := strconv.Atoi(apiIDStr)
	if err != nil {
		return Config{}, fmt.Errorf("invalid TELEGRAM_API_ID: %w", err)
	}

	apiHash := os.Getenv("TELEGRAM_API_HASH")
	if apiHash == "" {
		return Config{}, ErrMissingAPIHash
	}

	channelIDStr := os.Getenv("TELEGRAM_CHANNEL_ID")
	if channelIDStr == "" {
		return Config{}, ErrMissingChannel
	}

	channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
	if err != nil {
		return Config{}, fmt.Errorf("invalid TELEGRAM_CHANNEL_ID: %w", err)
	}

	return Config{
		APIID:           apiID,
		APIHash:         apiHash,
		BotToken:        os.Getenv("TELEGRAM_BOT_TOKEN"),
		PhoneNumber:     os.Getenv("TELEGRAM_PHONE"),
		SessionPath:     os.Getenv("TELEGRAM_SESSION_PATH"),
		ChannelID:       channelID,
		ChannelUsername: os.Getenv("TELEGRAM_CHANNEL_USERNAME"),
	}, nil
}

// Connect - Kết nối và xác thực với Telegram
func (c *TelegramClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	// CRITICAL: If phone number is provided, ONLY use user auth, never bot
	useUserAuth := c.config.PhoneNumber != ""
	if useUserAuth {
		fmt.Println("[Telegram] User authentication mode (MTProto)")
		fmt.Printf("[Telegram] Phone: %s\n", c.config.PhoneNumber)
	}

	return c.client.Run(ctx, func(ctx context.Context) error {
		// Lấy API client
		c.api = c.client.API()

		// Kiểm tra trạng thái đăng nhập
		status, err := c.client.Auth().Status(ctx)
		if err != nil {
			return fmt.Errorf("failed to get auth status: %w", err)
		}

		if status.Authorized {
			// Check if session type matches what we need
			if useUserAuth && status.User.Bot {
				fmt.Println("[Telegram] WARNING: Found BOT session but USER auth required!")
				fmt.Println("[Telegram] Please delete the session and restart:")
				fmt.Printf("[Telegram]   rm -rf %s\n", c.sessionPath)
				return fmt.Errorf("bot session detected but user authentication required. "+
					"Please delete the session folder '%s' and restart", c.sessionPath)
			} else if status.User.Bot {
				fmt.Printf("[Telegram] Authenticated as Bot: @%s\n", status.User.Username)
			} else {
				fmt.Printf("[Telegram] Authenticated as User: %s %s (@%s)\n",
					status.User.FirstName, status.User.LastName, status.User.Username)
			}
		}

		if !status.Authorized {
			// User authentication via phone number (MTProto)
			if useUserAuth {
				fmt.Println("[Telegram] Starting MTProto user authentication...")
				fmt.Println("[Telegram] Telegram will send OTP code to your phone/Telegram app")
				fmt.Printf("[Telegram] Write the code to file: %s\n", os.Getenv("TELEGRAM_CODE_PATH"))

				flow := auth.NewFlow(
					&codeAuth{phone: c.config.PhoneNumber},
					auth.SendCodeOptions{},
				)
				if err := c.client.Auth().IfNecessary(ctx, flow); err != nil {
					return fmt.Errorf("MTProto user authentication failed: %w", err)
				}
				fmt.Println("[Telegram] MTProto user authentication successful!")
			} else if c.config.BotToken != "" {
				// Bot authentication (only if no phone number provided)
				fmt.Println("[Telegram] Authenticating with Bot Token...")
				_, err := c.client.Auth().Bot(ctx, c.config.BotToken)
				if err != nil {
					return fmt.Errorf("bot authentication failed: %w", err)
				}
				fmt.Println("[Telegram] Bot authentication successful!")
			} else {
				return ErrAuthRequired
			}
		}

		// Load dialogs first to populate entity cache (required for channel resolution)
		fmt.Println("[Telegram] Loading dialogs to populate entity cache...")
		_, _ = c.api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
			OffsetPeer: &tg.InputPeerEmpty{},
			Limit:      100,
		})

		// Lấy thông tin channel
		if err := c.resolveChannel(ctx); err != nil {
			return fmt.Errorf("failed to resolve channel: %w", err)
		}

		c.connected = true
		return nil
	})
}

// resolveChannel - Lấy thông tin channel từ channel ID
func (c *TelegramClient) resolveChannel(ctx context.Context) error {
	// Lưu channel ID gốc (có thể có prefix -100)
	originalID := c.channelID

	// Convert marked ID format: -1003750835893 → 3750835893
	// Telegram marked IDs have -100 prefix for channels/supergroups
	rawChannelID := c.channelID
	if rawChannelID < 0 {
		// Remove -100 prefix
		channelIDStr := fmt.Sprintf("%d", -rawChannelID)
		if len(channelIDStr) > 3 && channelIDStr[:3] == "100" {
			rawChannelID, _ = strconv.ParseInt(channelIDStr[3:], 10, 64)
		} else {
			rawChannelID = -rawChannelID
		}
	}

	fmt.Printf("[Telegram] Resolving channel - Original ID: %d, Raw ID: %d\n", originalID, rawChannelID)
	c.channelID = rawChannelID

	// Method 1: Get ALL dialogs and find our channel
	fmt.Println("[Telegram] Method 1: Searching in dialogs...")
	dialogs, err := c.api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		OffsetPeer: &tg.InputPeerEmpty{},
		Limit:      200, // Lấy nhiều hơn để tìm channel
	})
	if err == nil {
		var chats []tg.ChatClass
		switch d := dialogs.(type) {
		case *tg.MessagesDialogs:
			chats = d.Chats
		case *tg.MessagesDialogsSlice:
			chats = d.Chats
		}

		fmt.Printf("[Telegram] Found %d chats in dialogs\n", len(chats))
		for _, chat := range chats {
			if channel, ok := chat.(*tg.Channel); ok {
				fmt.Printf("[Telegram] - Channel: ID=%d, Title=%s\n", channel.ID, channel.Title)
				if channel.ID == rawChannelID {
					c.accessHash = channel.AccessHash
					fmt.Printf("[Telegram] Found channel! AccessHash: %d\n", c.accessHash)
					return nil
				}
			}
		}
	} else {
		fmt.Printf("[Telegram] Method 1 error: %v\n", err)
	}

	// Method 2: Try InputPeerChannel directly (nếu đã có trong cache)
	fmt.Println("[Telegram] Method 2: Trying direct channel access...")
	inputPeer := &tg.InputPeerChannel{
		ChannelID:  rawChannelID,
		AccessHash: 0,
	}

	// Gửi một request đơn giản để lấy thông tin
	history, err := c.api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:  inputPeer,
		Limit: 1,
	})
	if err == nil {
		switch h := history.(type) {
		case *tg.MessagesChannelMessages:
			for _, chat := range h.Chats {
				if channel, ok := chat.(*tg.Channel); ok {
					if channel.ID == rawChannelID {
						c.accessHash = channel.AccessHash
						fmt.Printf("[Telegram] Found channel via history! AccessHash: %d\n", c.accessHash)
						return nil
					}
				}
			}
		}
	} else {
		fmt.Printf("[Telegram] Method 2 error: %v\n", err)
	}

	// Method 3: Try to resolve by username if provided
	if c.config.ChannelUsername != "" {
		username := strings.TrimPrefix(c.config.ChannelUsername, "@")
		fmt.Printf("[Telegram] Method 3: Resolving by username: %s\n", username)

		resolved, err := c.api.ContactsResolveUsername(ctx, username)
		if err == nil {
			for _, chat := range resolved.Chats {
				if channel, ok := chat.(*tg.Channel); ok {
					c.channelID = channel.ID
					c.accessHash = channel.AccessHash
					fmt.Printf("[Telegram] Found channel via username! ID=%d, AccessHash=%d\n", channel.ID, c.accessHash)
					return nil
				}
			}
		} else {
			fmt.Printf("[Telegram] Method 3 error: %v\n", err)
		}
	}

	// Method 4: If channel is public, try joining first
	fmt.Println("[Telegram] Method 4: Channel not found. Make sure to:")
	fmt.Println("  1. Open and view the channel in Telegram app")
	fmt.Println("  2. Join or become admin of the channel")
	fmt.Println("  3. Set TELEGRAM_CHANNEL_USERNAME in .env if channel has username")

	return fmt.Errorf("channel not found (Raw ID: %d, Original ID: %d). "+
		"Make sure you have joined or are admin of this channel. "+
		"You can also set TELEGRAM_CHANNEL_USERNAME=@your_channel in .env",
		rawChannelID, originalID)
}

// Disconnect - Ngắt kết nối với Telegram
func (c *TelegramClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.connected = false
	return nil
}

// IsConnected - Kiểm tra trạng thái kết nối
func (c *TelegramClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// GetAPI - Lấy API client
func (c *TelegramClient) GetAPI() *tg.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.api
}

// GetChannelID - Lấy channel ID
func (c *TelegramClient) GetChannelID() int64 {
	return c.channelID
}

// GetAccessHash - Lấy access hash của channel
func (c *TelegramClient) GetAccessHash() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accessHash
}

// GetInputChannel - Lấy InputChannel struct
func (c *TelegramClient) GetInputChannel() *tg.InputChannel {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return &tg.InputChannel{
		ChannelID:  c.channelID,
		AccessHash: c.accessHash,
	}
}

// RunWithClient - Chạy function với client context (DEPRECATED: use RunWithCallback instead)
func (c *TelegramClient) RunWithClient(ctx context.Context, f func(ctx context.Context) error) error {
	return c.client.Run(ctx, f)
}

// RunWithCallback - Kết nối, xác thực, và chạy callback trong lifecycle của MTProto client
// Đây là phương thức chính để thực hiện các thao tác với Telegram
// Client sẽ được giữ mở cho đến khi callback hoàn thành
func (c *TelegramClient) RunWithCallback(ctx context.Context, callback func(ctx context.Context) error) error {
	c.mu.Lock()
	useUserAuth := c.config.PhoneNumber != ""
	if useUserAuth {
		fmt.Println("[Telegram] User authentication mode (MTProto)")
		fmt.Printf("[Telegram] Phone: %s\n", c.config.PhoneNumber)
	}
	c.mu.Unlock()

	return c.client.Run(ctx, func(ctx context.Context) error {
		// Lấy API client
		c.mu.Lock()
		c.api = c.client.API()
		c.mu.Unlock()

		// Kiểm tra trạng thái đăng nhập
		status, err := c.client.Auth().Status(ctx)
		if err != nil {
			return fmt.Errorf("failed to get auth status: %w", err)
		}

		if status.Authorized {
			// Check if session type matches what we need
			if useUserAuth && status.User.Bot {
				fmt.Println("[Telegram] WARNING: Found BOT session but USER auth required!")
				fmt.Println("[Telegram] Please delete the session and restart:")
				fmt.Printf("[Telegram]   rm -rf %s\n", c.sessionPath)
				return fmt.Errorf("bot session detected but user authentication required. "+
					"Please delete the session folder '%s' and restart", c.sessionPath)
			} else if status.User.Bot {
				fmt.Printf("[Telegram] Authenticated as Bot: @%s\n", status.User.Username)
			} else {
				fmt.Printf("[Telegram] Authenticated as User: %s %s (@%s)\n",
					status.User.FirstName, status.User.LastName, status.User.Username)
			}
		}

		if !status.Authorized {
			// User authentication via phone number (MTProto)
			if useUserAuth {
				fmt.Println("[Telegram] Starting MTProto user authentication...")
				fmt.Println("[Telegram] Telegram will send OTP code to your phone/Telegram app")
				fmt.Printf("[Telegram] Write the code to file: %s\n", os.Getenv("TELEGRAM_CODE_PATH"))

				flow := auth.NewFlow(
					&codeAuth{phone: c.config.PhoneNumber},
					auth.SendCodeOptions{},
				)
				if err := c.client.Auth().IfNecessary(ctx, flow); err != nil {
					return fmt.Errorf("MTProto user authentication failed: %w", err)
				}
				fmt.Println("[Telegram] MTProto user authentication successful!")
			} else if c.config.BotToken != "" {
				// Bot authentication (only if no phone number provided)
				fmt.Println("[Telegram] Authenticating with Bot Token...")
				_, err := c.client.Auth().Bot(ctx, c.config.BotToken)
				if err != nil {
					return fmt.Errorf("bot authentication failed: %w", err)
				}
				fmt.Println("[Telegram] Bot authentication successful!")
			} else {
				return ErrAuthRequired
			}
		}

		// Load dialogs first to populate entity cache (required for channel resolution)
		fmt.Println("[Telegram] Loading dialogs to populate entity cache...")
		_, _ = c.api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
			OffsetPeer: &tg.InputPeerEmpty{},
			Limit:      100,
		})

		// Lấy thông tin channel
		if err := c.resolveChannel(ctx); err != nil {
			return fmt.Errorf("failed to resolve channel: %w", err)
		}

		c.mu.Lock()
		c.connected = true
		c.mu.Unlock()

		fmt.Println("[Telegram] Client ready, executing callback...")

		// Execute user callback - client stays open until this completes
		err = callback(ctx)

		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()

		return err
	})
}

// StartPersistentConnection - Khởi động kết nối persistent trong background
// Giữ MTProto client sống để xử lý streaming requests
func (c *TelegramClient) StartPersistentConnection(parentCtx context.Context) error {
	c.mu.Lock()
	if c.workChan != nil {
		c.mu.Unlock()
		return nil // Already started
	}

	c.workChan = make(chan workRequest, 10)
	c.persistentReady = make(chan struct{})
	c.persistentCtx, c.persistentCancel = context.WithCancel(parentCtx)
	c.mu.Unlock()

	// Start connection in background goroutine
	go func() {
		readyClosed := false

		for {
			err := c.client.Run(c.persistentCtx, func(ctx context.Context) error {
				// Setup API client
				c.mu.Lock()
				c.api = c.client.API()
				c.mu.Unlock()

			// Authenticate
			useUserAuth := c.config.PhoneNumber != ""
			status, err := c.client.Auth().Status(ctx)
			if err != nil {
				return fmt.Errorf("failed to get auth status: %w", err)
			}

			if !status.Authorized {
				if useUserAuth {
					fmt.Println("[Telegram Persistent] Starting MTProto user authentication...")
					flow := auth.NewFlow(
						&codeAuth{phone: c.config.PhoneNumber},
						auth.SendCodeOptions{},
					)
					if err := c.client.Auth().IfNecessary(ctx, flow); err != nil {
						return fmt.Errorf("MTProto authentication failed: %w", err)
					}
				} else if c.config.BotToken != "" {
					_, err := c.client.Auth().Bot(ctx, c.config.BotToken)
					if err != nil {
						return fmt.Errorf("bot authentication failed: %w", err)
					}
				} else {
					return ErrAuthRequired
				}
			}

			// Load dialogs to populate entity cache
			fmt.Println("[Telegram Persistent] Loading dialogs...")
			_, _ = c.api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
				OffsetPeer: &tg.InputPeerEmpty{},
				Limit:      100,
			})

			// Resolve channel
			if err := c.resolveChannel(ctx); err != nil {
				return fmt.Errorf("failed to resolve channel: %w", err)
			}

			c.mu.Lock()
			c.connected = true
			c.mu.Unlock()

			fmt.Println("[Telegram Persistent] Connection ready, waiting for work...")

			// Signal ready
			if !readyClosed {
				close(c.persistentReady)
				readyClosed = true
			}

			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			consecutivePingFails := 0

			// Process work requests
			for {
				select {
				case <-ctx.Done():
					fmt.Println("[Telegram Persistent] Context cancelled, shutting down...")
					return ctx.Err()
				case <-ticker.C:
					_, pingErr := c.api.HelpGetNearestDC(ctx)
					if pingErr != nil {
						log.Printf("[Heartbeat] Ping failed: %v", pingErr)
						consecutivePingFails++
						if consecutivePingFails >= 3 {
							log.Printf("[Heartbeat] Ping failed 3 times consecutively. Tripping circuit breaker.")
							c.mu.RLock()
							cb := c.circuitBreaker
							c.mu.RUnlock()
							if cb != nil {
								// Trip the circuit breaker by recording failures
								for i := 0; i < 3; i++ {
									cb.RecordFailure()
								}
							}
							return fmt.Errorf("connection lost: ping failed 3 times")
						}
					} else {
						// log.Printf("[Heartbeat] Ping OK")
						consecutivePingFails = 0
					}
				case req := <-c.workChan:
					// Execute work in connection context
					err := req.work(ctx)
					req.resultCh <- err
				}
			}
		})

		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()

		if err != nil {
			fmt.Printf("[Telegram Persistent] Connection error: %v\n", err)
		}

		// Check if we should exit permanently
		select {
		case <-c.persistentCtx.Done():
			c.mu.Lock()
			c.persistentErr = c.persistentCtx.Err()
			c.mu.Unlock()
			return
		default:
		}

		fmt.Println("[Telegram Persistent] Waiting 30s before reconnecting...")
		select {
		case <-time.After(30 * time.Second):
		case <-c.persistentCtx.Done():
			c.mu.Lock()
			c.persistentErr = c.persistentCtx.Err()
			c.mu.Unlock()
			return
		}
		} // <- close for loop
	}()

	// Wait for ready or timeout
	select {
	case <-c.persistentReady:
		fmt.Println("[Telegram Persistent] Connection established successfully")
		return nil
	case <-time.After(60 * time.Second):
		c.StopPersistentConnection()
		return fmt.Errorf("timeout waiting for persistent connection")
	case <-parentCtx.Done():
		c.StopPersistentConnection()
		return parentCtx.Err()
	}
}

// ExecuteInConnection - Thực thi function trong persistent connection
func (c *TelegramClient) ExecuteInConnection(ctx context.Context, work func(context.Context) error) error {
	c.mu.RLock()
	workChan := c.workChan
	connected := c.connected
	c.mu.RUnlock()

	if workChan == nil || !connected {
		return ErrNotConnected
	}

	resultCh := make(chan error, 1)
	req := workRequest{
		ctx:      ctx,
		work:     work,
		resultCh: resultCh,
	}

	select {
	case workChan <- req:
		// Wait for result
		select {
		case err := <-resultCh:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	case <-ctx.Done():
		return ctx.Err()
	}
}

// StopPersistentConnection - Dừng persistent connection
func (c *TelegramClient) StopPersistentConnection() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.persistentCancel != nil {
		c.persistentCancel()
		c.persistentCancel = nil
	}

	if c.workChan != nil {
		close(c.workChan)
		c.workChan = nil
	}

	c.connected = false
	fmt.Println("[Telegram Persistent] Connection stopped")
}

// codeAuth - Implementation cho phone authentication (cần user input)
type codeAuth struct {
	phone string
}

func (a *codeAuth) Phone(_ context.Context) (string, error) {
	return a.phone, nil
}

func (a *codeAuth) Password(_ context.Context) (string, error) {
	// Đọc password từ file hoặc stdin (không thể dùng interactive trong background service)
	password := os.Getenv("TELEGRAM_PASSWORD")
	if password == "" {
		return "", fmt.Errorf("2FA password required but not provided")
	}
	return password, nil
}

func (a *codeAuth) Code(_ context.Context, sentCode *tg.AuthSentCode) (string, error) {
	// Hiển thị thông tin về code
	fmt.Println("")
	fmt.Println("========================================")
	fmt.Println("  TELEGRAM VERIFICATION CODE REQUIRED")
	fmt.Println("========================================")
	fmt.Println("")
	fmt.Printf("A verification code has been sent to: %s\n", a.phone)
	fmt.Println("")
	fmt.Println("You have 2 options to enter the code:")
	fmt.Println("")

	codePath := os.Getenv("TELEGRAM_CODE_PATH")
	if codePath == "" {
		codePath = "./telegram_code.txt"
	}

	fmt.Printf("  Option 1: Create file '%s' with the code\n", codePath)
	fmt.Println("  Option 2: Enter code directly below")
	fmt.Println("")
	fmt.Println("Waiting for code (5 minute timeout)...")
	fmt.Println("")

	// Goroutine để đọc từ stdin
	codeChan := make(chan string, 1)
	go func() {
		fmt.Print("Enter code here: ")
		var code string
		fmt.Scanln(&code)
		if code != "" {
			codeChan <- code
		}
	}()

	// Đợi code từ file hoặc stdin (max 5 phút)
	for i := 0; i < 300; i++ {
		// Check file
		data, err := os.ReadFile(codePath)
		if err == nil && len(data) > 0 {
			code := strings.TrimSpace(string(data))
			if code != "" {
				os.Remove(codePath) // Xóa file sau khi đọc
				fmt.Printf("Code received from file: %s\n", code)
				return code, nil
			}
		}

		// Check stdin
		select {
		case code := <-codeChan:
			fmt.Printf("Code received from input: %s\n", code)
			return strings.TrimSpace(code), nil
		default:
		}

		time.Sleep(1 * time.Second)
	}

	return "", fmt.Errorf("verification code timeout after 5 minutes")
}

func (a *codeAuth) AcceptTermsOfService(_ context.Context, tos tg.HelpTermsOfService) error {
	return nil
}

func (a *codeAuth) SignUp(_ context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, fmt.Errorf("sign up not allowed")
}

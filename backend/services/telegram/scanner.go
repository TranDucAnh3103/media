package telegram

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gotd/td/tg"
)

// ChannelScanner - Service để quét và đồng bộ video từ Telegram channel
type ChannelScanner struct {
	client     *TelegramClient
	mu         sync.RWMutex
	isScanning bool
	status     ScanStatus
}

// ScanStatus - Trạng thái quét channel
type ScanStatus struct {
	IsRunning       bool      `json:"is_running"`
	LastSyncAt      time.Time `json:"last_sync_at"`
	TotalScanned    int       `json:"total_scanned"`
	NewVideosFound  int       `json:"new_videos_found"`
	ErrorCount      int       `json:"error_count"`
	LastError       string    `json:"last_error,omitempty"`
	CurrentProgress int       `json:"current_progress"`
	TotalMessages   int       `json:"total_messages"`
	StartedAt       time.Time `json:"started_at"`
}

// ChannelVideoMeta - Metadata của video được extract từ Telegram message
type ChannelVideoMeta struct {
	MsgID            int       `json:"message_id"`
	GrpID            int64     `json:"grouped_id"`
	ChanID           int64     `json:"channel_id"`
	Text             string    `json:"caption"`
	Dur              int       `json:"duration"`
	Size             int64     `json:"file_size"`
	W                int       `json:"width"`
	H                int       `json:"height"`
	Mime             string    `json:"mime_type"`
	Name             string    `json:"file_name"`
	DocID            int64     `json:"file_id"`
	FileRef          []byte    `json:"file_reference"`
	DocAccessHash    int64     `json:"access_hash"`
	MsgDate          time.Time `json:"message_date"`
	ThumbID          string    `json:"thumbnail_file_id"`
	Spoiler          bool      `json:"has_spoiler"`
	StreamingSupport bool      `json:"supports_streaming"`
}

// ScanOptions - Options cho việc quét channel
type ScanOptions struct {
	ChannelID int64 // Override default channel ID
	Limit     int   // Số message tối đa (0 = không giới hạn, scan tất cả)
	OffsetID  int   // Bắt đầu từ message ID này (0 = mới nhất)
	MinMsgID  int   // Message ID nhỏ nhất đã có trong database (để tránh duplicate)
}

// NewChannelScanner - Tạo instance mới của ChannelScanner
func NewChannelScanner(client *TelegramClient) *ChannelScanner {
	return &ChannelScanner{
		client: client,
		status: ScanStatus{},
	}
}

// IsScanning - Kiểm tra xem đang quét không
func (s *ChannelScanner) IsScanning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isScanning
}

// GetStatus - Lấy trạng thái quét hiện tại
func (s *ChannelScanner) GetStatus() ScanStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

// ScanChannel - Quét channel và trả về danh sách video metadata
// Đảm bảo TẤT CẢ Telegram API calls đều nằm trong ExecuteInConnection
func (s *ChannelScanner) ScanChannel(ctx context.Context, opts ScanOptions) ([]ChannelVideoMeta, error) {
	var allVideos []ChannelVideoMeta
	var scanErr error

	err := s.client.ExecuteInConnection(ctx, func(connCtx context.Context) error {
		// Check if already scanning
		s.mu.Lock()
		if s.isScanning {
			s.mu.Unlock()
			scanErr = fmt.Errorf("scan already in progress")
			return scanErr
		}
		s.isScanning = true
		s.status = ScanStatus{
			IsRunning: true,
			StartedAt: time.Now(),
		}
		s.mu.Unlock()

		// Cleanup on exit
		defer func() {
			s.mu.Lock()
			s.isScanning = false
			s.status.IsRunning = false
			s.status.LastSyncAt = time.Now()
			s.mu.Unlock()
		}()

		// Get channel info
		channelID := opts.ChannelID
		if channelID == 0 {
			channelID = s.client.GetChannelID()
		}
		accessHash := s.client.GetAccessHash()

		fmt.Printf("[Scanner] Starting channel scan - Channel ID: %d, AccessHash: %d\n", channelID, accessHash)

		offsetID := opts.OffsetID
		batchSize := 100 // Telegram API limit
		totalScanned := 0

		for {
			select {
			case <-connCtx.Done():
				scanErr = connCtx.Err()
				return scanErr
			default:
			}

			// Fetch messages batch
			fmt.Printf("[Scanner] Fetching messages batch - OffsetID: %d, Limit: %d\n", offsetID, batchSize)

			messages, err := s.fetchMessagesBatch(connCtx, channelID, accessHash, offsetID, batchSize)
			if err != nil {
				s.mu.Lock()
				s.status.ErrorCount++
				s.status.LastError = err.Error()
				s.mu.Unlock()
				scanErr = fmt.Errorf("failed to fetch messages: %w", err)
				return scanErr
			}

			if len(messages) == 0 {
				fmt.Println("[Scanner] No more messages to fetch")
				break
			}

			fmt.Printf("[Scanner] Processing %d messages\n", len(messages))

			// Process each message
			for _, msg := range messages {
				totalScanned++

				// Update progress
				s.mu.Lock()
				s.status.CurrentProgress = totalScanned
				s.mu.Unlock()

				// Extract video metadata if message contains video
				videoMeta := s.extractVideoMetadata(msg, channelID)
				if videoMeta != nil {
					// Check if we already have this video (by MinMsgID)
					if opts.MinMsgID > 0 && videoMeta.MsgID <= opts.MinMsgID {
						fmt.Printf("[Scanner] Reached existing message ID %d (min: %d), stopping\n", videoMeta.MsgID, opts.MinMsgID)
						s.mu.Lock()
						s.status.TotalScanned = totalScanned
						s.status.NewVideosFound = len(allVideos)
						s.mu.Unlock()
						return nil
					}

					allVideos = append(allVideos, *videoMeta)
					fmt.Printf("[Scanner] Found video in message %d (Duration: %ds, Size: %d bytes)\n",
						videoMeta.MsgID, videoMeta.Dur, videoMeta.Size)
				}

				// Update last message ID for next batch
				if m, ok := msg.(*tg.Message); ok {
					offsetID = m.ID
				}
			}

			// Check limit
			if opts.Limit > 0 && totalScanned >= opts.Limit {
				fmt.Printf("[Scanner] Reached scan limit: %d\n", opts.Limit)
				break
			}

			// Check if we got fewer messages than requested (end of channel)
			if len(messages) < batchSize {
				fmt.Println("[Scanner] Reached end of channel")
				break
			}
		}

		// Update final status
		s.mu.Lock()
		s.status.TotalScanned = totalScanned
		s.status.NewVideosFound = len(allVideos)
		s.mu.Unlock()

		fmt.Printf("[Scanner] Scan complete - Total scanned: %d, Videos found: %d\n", totalScanned, len(allVideos))
		return nil
	})

	if scanErr != nil {
		return allVideos, scanErr
	}
	if err != nil {
		return allVideos, err
	}
	return allVideos, nil
}

// fetchMessagesBatch - Fetch một batch messages từ channel
func (s *ChannelScanner) fetchMessagesBatch(ctx context.Context, channelID, accessHash int64, offsetID, limit int) ([]tg.MessageClass, error) {
	// FIX: khai báo api trước, không dùng trước khi khai báo
	api := s.client.GetAPI()
	if api == nil {
		return nil, fmt.Errorf("API client not available")
	}

	inputPeer := &tg.InputPeerChannel{
		ChannelID:  channelID,
		AccessHash: accessHash,
	}

	history, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:     inputPeer,
		OffsetID: offsetID,
		Limit:    limit,
	})
	if err != nil {
		return nil, err
	}

	var messages []tg.MessageClass
	switch h := history.(type) {
	case *tg.MessagesMessages:
		messages = h.Messages
	case *tg.MessagesMessagesSlice:
		messages = h.Messages
	case *tg.MessagesChannelMessages:
		messages = h.Messages
	}

	return messages, nil
}

// extractVideoMetadata - Extract video metadata từ message
func (s *ChannelScanner) extractVideoMetadata(msgClass tg.MessageClass, channelID int64) *ChannelVideoMeta {
	msg, ok := msgClass.(*tg.Message)
	if !ok {
		return nil
	}

	// Check if message has media
	if msg.Media == nil {
		return nil
	}

	// Check for video in different media types
	var doc *tg.Document
	var hasSpoiler bool

	switch media := msg.Media.(type) {
	case *tg.MessageMediaDocument:
		// Direct document media
		if d, ok := media.Document.(*tg.Document); ok {
			// Check if it's a video
			if !isVideoDocument(d) {
				return nil
			}
			doc = d
			hasSpoiler = media.Spoiler
		}

	case *tg.MessageMediaPhoto:
		// Photos are not videos
		return nil

	default:
		return nil
	}

	if doc == nil {
		return nil
	}

	// Extract video attributes
	var duration int
	var width, height int
	var fileName string
	var supportsStreaming bool

	for _, attr := range doc.Attributes {
		switch a := attr.(type) {
		case *tg.DocumentAttributeVideo:
			duration = int(a.Duration)
			width = a.W
			height = a.H
			supportsStreaming = a.SupportsStreaming
		case *tg.DocumentAttributeFilename:
			fileName = a.FileName
		}
	}

	// Build metadata
	videoMeta := &ChannelVideoMeta{
		MsgID:            msg.ID,
		GrpID:            msg.GroupedID,
		ChanID:           channelID,
		Text:             msg.Message,
		Dur:              duration,
		Size:             doc.Size,
		W:                width,
		H:                height,
		Mime:             doc.MimeType,
		Name:             fileName,
		DocID:            doc.ID,
		FileRef:          doc.FileReference,
		DocAccessHash:    doc.AccessHash,
		MsgDate:          time.Unix(int64(msg.Date), 0),
		Spoiler:          hasSpoiler,
		StreamingSupport: supportsStreaming,
	}

	// Extract thumbnail info (if available)
	if len(doc.Thumbs) > 0 {
		// Just mark that thumbnail exists, we'll use the document's thumbnail
		videoMeta.ThumbID = fmt.Sprintf("%d_thumb", doc.ID)
	}

	return videoMeta
}

// isVideoDocument - Kiểm tra xem document có phải là video không
func isVideoDocument(doc *tg.Document) bool {
	// Check MIME type
	if doc.MimeType != "" {
		if len(doc.MimeType) >= 5 && doc.MimeType[:5] == "video" {
			return true
		}
	}

	// Check attributes
	for _, attr := range doc.Attributes {
		if _, ok := attr.(*tg.DocumentAttributeVideo); ok {
			return true
		}
	}

	return false
}

// ScanChannelAsync - Quét channel bất đồng bộ (chạy trong goroutine)
// Callback được gọi khi scan hoàn thành hoặc có lỗi
func (s *ChannelScanner) ScanChannelAsync(ctx context.Context, opts ScanOptions, callback func([]ChannelVideoMeta, error)) {
	go func() {
		videos, err := s.ScanChannel(ctx, opts)
		if callback != nil {
			callback(videos, err)
		}
	}()
}

// FetchSingleMessage - Lấy thông tin một message cụ thể
func (s *ChannelScanner) FetchSingleMessage(ctx context.Context, channelID, accessHash int64, messageID int) (*ChannelVideoMeta, error) {
	api := s.client.GetAPI()
	if api == nil {
		return nil, fmt.Errorf("API client not available")
	}

	inputChannel := &tg.InputChannel{
		ChannelID:  channelID,
		AccessHash: accessHash,
	}

	result, err := api.ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
		Channel: inputChannel,
		ID:      []tg.InputMessageClass{&tg.InputMessageID{ID: messageID}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	var messages []tg.MessageClass
	switch r := result.(type) {
	case *tg.MessagesMessages:
		messages = r.Messages
	case *tg.MessagesMessagesSlice:
		messages = r.Messages
	case *tg.MessagesChannelMessages:
		messages = r.Messages
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("message not found")
	}

	return s.extractVideoMetadata(messages[0], channelID), nil
}

// RefreshFileReference - Làm mới file reference cho một video (khi reference hết hạn)
func (s *ChannelScanner) RefreshFileReference(ctx context.Context, channelID, accessHash int64, messageID int) ([]byte, error) {
	meta, err := s.FetchSingleMessage(ctx, channelID, accessHash, messageID)
	if err != nil {
		return nil, err
	}
	if meta == nil {
		return nil, fmt.Errorf("video not found in message")
	}
	return meta.FileRef, nil
}
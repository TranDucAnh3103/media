package telegram

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/gotd/td/tg"
)

// TelegramService - Service tổng hợp cho upload và stream video qua Telegram
type TelegramService struct {
	client     *TelegramClient
	uploader   *TelegramUploader
	streamer   *TelegramStreamer
	protection *ProtectionLayer
	mu         sync.RWMutex
}

// NewTelegramService - Tạo instance mới của TelegramService
func NewTelegramService() (*TelegramService, error) {
	// Tạo client
	client, err := NewTelegramClient()
	if err != nil {
		return nil, err
	}

	// Initialize protection layer with default config
	protectionConfig := DefaultProtectionConfig()
	protection := NewProtectionLayer(protectionConfig)

	log.Println("[TelegramService] Initialized with anti-ban protection layer")

	return &TelegramService{
		client:     client,
		uploader:   NewTelegramUploader(client),
		streamer:   NewTelegramStreamer(client),
		protection: protection,
	}, nil
}

// NewTelegramServiceWithConfig - Tạo instance với custom protection config
func NewTelegramServiceWithConfig(protectionConfig ProtectionConfig) (*TelegramService, error) {
	client, err := NewTelegramClient()
	if err != nil {
		return nil, err
	}

	protection := NewProtectionLayer(protectionConfig)

	log.Println("[TelegramService] Initialized with custom protection config")

	return &TelegramService{
		client:     client,
		uploader:   NewTelegramUploader(client),
		streamer:   NewTelegramStreamer(client),
		protection: protection,
	}, nil
}

// Connect - Kết nối và xác thực với Telegram
func (s *TelegramService) Connect(ctx context.Context) error {
	return s.client.Connect(ctx)
}

// Disconnect - Ngắt kết nối
func (s *TelegramService) Disconnect() error {
	// Stop protection layer first
	if s.protection != nil {
		s.protection.Stop()
	}
	// Stop persistent connection if running
	s.client.StopPersistentConnection()
	return s.client.Disconnect()
}

// IsConnected - Kiểm tra trạng thái kết nối
func (s *TelegramService) IsConnected() bool {
	return s.client.IsConnected()
}

// StartPersistentConnection - Khởi động persistent connection cho streaming
func (s *TelegramService) StartPersistentConnection(ctx context.Context) error {
	return s.client.StartPersistentConnection(ctx)
}

// StopPersistentConnection - Dừng persistent connection
func (s *TelegramService) StopPersistentConnection() {
	s.client.StopPersistentConnection()
}

// ExecuteInConnection - Thực thi function trong persistent connection
func (s *TelegramService) ExecuteInConnection(ctx context.Context, work func(context.Context) error) error {
	return s.client.ExecuteInConnection(ctx, work)
}

// ============ UPLOAD METHODS (Protected) ============

// UploadVideo - Upload video lên Telegram channel với protection
func (s *TelegramService) UploadVideo(ctx context.Context, req VideoUploadRequest) (*VideoUploadResult, error) {
	var result *VideoUploadResult
	var uploadErr error

	// Extract user ID from context or use default
	userID := extractUserID(ctx)

	err := s.protection.ExecuteUpload(ctx, userID, func() error {
		result, uploadErr = s.uploader.UploadVideo(ctx, req)
		return uploadErr
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// UploadVideoUnprotected - Upload video without protection (for internal use)
func (s *TelegramService) UploadVideoUnprotected(ctx context.Context, req VideoUploadRequest) (*VideoUploadResult, error) {
	return s.uploader.UploadVideo(ctx, req)
}

// UploadFromReader - Upload video từ io.Reader với protection
func (s *TelegramService) UploadFromReader(ctx context.Context, reader io.Reader, fileName string, size int64, caption string, progressCb func(UploadProgress)) (*VideoUploadResult, error) {
	var result *VideoUploadResult
	var uploadErr error

	userID := extractUserID(ctx)

	err := s.protection.ExecuteUpload(ctx, userID, func() error {
		result, uploadErr = s.uploader.UploadFromReader(ctx, reader, fileName, size, caption, progressCb)
		return uploadErr
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// UploadFromReaderUnprotected - Upload từ reader without protection
func (s *TelegramService) UploadFromReaderUnprotected(ctx context.Context, reader io.Reader, fileName string, size int64, caption string, progressCb func(UploadProgress)) (*VideoUploadResult, error) {
	return s.uploader.UploadFromReader(ctx, reader, fileName, size, caption, progressCb)
}

// DeleteVideo - Xóa video từ Telegram channel
func (s *TelegramService) DeleteVideo(ctx context.Context, messageID int) error {
	userID := extractUserID(ctx)

	return s.protection.ExecuteRequest(ctx, userID, func() error {
		return s.uploader.DeleteVideo(ctx, messageID)
	})
}

// GetVideoInfo - Lấy thông tin video từ message
func (s *TelegramService) GetVideoInfo(ctx context.Context, messageID int) (*MessageInfo, error) {
	var info *MessageInfo
	var infoErr error

	userID := extractUserID(ctx)

	err := s.protection.ExecuteRequest(ctx, userID, func() error {
		info, infoErr = s.uploader.GetVideoInfo(ctx, messageID)
		return infoErr
	})

	if err != nil {
		return nil, err
	}
	return info, nil
}

// ============ STREAM METHODS ============

// GetFileSize - Lấy kích thước file
func (s *TelegramService) GetFileSize(ctx context.Context, messageID int) (int64, error) {
	return s.streamer.GetFileSize(ctx, messageID)
}

// StreamVideo - Stream video từ Telegram
func (s *TelegramService) StreamVideo(ctx context.Context, req StreamRequest, writer io.Writer) error {
	return s.streamer.StreamVideo(ctx, req, writer)
}

// GetVideoMetadata - Lấy metadata của video
func (s *TelegramService) GetVideoMetadata(ctx context.Context, messageID int) (*TelegramVideoMeta, error) {
	return s.streamer.GetVideoMetadata(ctx, messageID)
}

// DownloadVideo - Tải toàn bộ video từ Telegram về memory
func (s *TelegramService) DownloadVideo(ctx context.Context, messageID int) ([]byte, error) {
	// Lấy kích thước file
	fileSize, err := s.streamer.GetFileSize(ctx, messageID)
	if err != nil {
		return nil, err
	}

	// Stream toàn bộ video vào buffer
	buf := make([]byte, 0, fileSize)
	writer := &bytesWriter{buf: &buf}

	req := StreamRequest{
		MessageID: messageID,
		Start:     0,
		End:       fileSize - 1,
	}

	if err := s.streamer.StreamVideo(ctx, req, writer); err != nil {
		return nil, err
	}

	return *writer.buf, nil
}

// DownloadVideoDirect - Download video khi đã trong RunWithCallback (không cần ExecuteInConnection)
func (s *TelegramService) DownloadVideoDirect(ctx context.Context, messageID int) ([]byte, error) {
	api := s.client.GetAPI()
	if api == nil {
		return nil, ErrNotConnected
	}

	// Lấy document info
	messages, err := api.ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
		Channel: s.client.GetInputChannel(),
		ID:      []tg.InputMessageClass{&tg.InputMessageID{ID: messageID}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	channelMessages, ok := messages.(*tg.MessagesChannelMessages)
	if !ok || len(channelMessages.Messages) == 0 {
		return nil, ErrMessageNotFound
	}

	msg, ok := channelMessages.Messages[0].(*tg.Message)
	if !ok {
		return nil, ErrMessageNotFound
	}

	media, ok := msg.Media.(*tg.MessageMediaDocument)
	if !ok {
		return nil, ErrFileNotFound
	}

	doc, ok := media.Document.(*tg.Document)
	if !ok {
		return nil, ErrFileNotFound
	}

	// Download file
	location := &tg.InputDocumentFileLocation{
		ID:            doc.ID,
		AccessHash:    doc.AccessHash,
		FileReference: doc.FileReference,
	}

	fileSize := doc.Size
	result := make([]byte, 0, fileSize)
	offset := int64(0)
	chunkSize := int64(DefaultChunkSize)

	for offset < fileSize {
		file, err := api.UploadGetFile(ctx, &tg.UploadGetFileRequest{
			Location:     location,
			Offset:       offset,
			Limit:        int(chunkSize),
			Precise:      true,
			CDNSupported: false,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to download chunk at %d: %w", offset, err)
		}

		fileResult, ok := file.(*tg.UploadFile)
		if !ok {
			return nil, ErrChunkDownloadFailed
		}

		if len(fileResult.Bytes) == 0 {
			break
		}

		result = append(result, fileResult.Bytes...)
		offset += int64(len(fileResult.Bytes))

		// Log progress for large files
		if fileSize > 10*1024*1024 && offset%(10*1024*1024) == 0 {
			log.Printf("[Download] Progress: %.1f%%", float64(offset)/float64(fileSize)*100)
		}
	}

	return result, nil
}

// bytesWriter - Simple bytes writer
type bytesWriter struct {
	buf *[]byte
}

func (w *bytesWriter) Write(p []byte) (n int, err error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}

// ============ HELPER METHODS ============

// GetChannelID - Lấy channel ID
func (s *TelegramService) GetChannelID() int64 {
	return s.client.GetChannelID()
}

// RunWithClient - Chạy function với client context (DEPRECATED)
func (s *TelegramService) RunWithClient(ctx context.Context, f func(ctx context.Context) error) error {
	return s.client.RunWithClient(ctx, f)
}

// RunWithCallback - Kết nối, xác thực, và chạy callback trong lifecycle của MTProto client
// Client sẽ được giữ mở cho đến khi callback hoàn thành
// Đây là phương thức chính để thực hiện upload/stream với Telegram
func (s *TelegramService) RunWithCallback(ctx context.Context, callback func(ctx context.Context) error) error {
	return s.client.RunWithCallback(ctx, callback)
}

// GetUploader - Lấy uploader để sử dụng trong RunWithCallback
func (s *TelegramService) GetUploader() *TelegramUploader {
	return s.uploader
}

// ============ PROTECTION LAYER METHODS ============

// GetProtectionStats - Lấy thống kê bảo vệ
func (s *TelegramService) GetProtectionStats() ProtectionStats {
	return s.protection.GetStats()
}

// ResetProtectionMetrics - Reset metrics của protection layer
func (s *TelegramService) ResetProtectionMetrics() {
	s.protection.ResetMetrics()
}

// SetSafeModeLevel - Set safe mode level manually
func (s *TelegramService) SetSafeModeLevel(level SafeModeLevel) {
	s.protection.SetSafeModeLevel(level)
}

// ResetSafeMode - Reset safe mode về normal
func (s *TelegramService) ResetSafeMode() {
	s.protection.ResetSafeMode()
}

// GetProtectionLayer - Trả về protection layer để cấu hình nâng cao
func (s *TelegramService) GetProtectionLayer() *ProtectionLayer {
	return s.protection
}

// HealthCheck - Kiểm tra sức khỏe của service
func (s *TelegramService) HealthCheck() map[string]interface{} {
	health := make(map[string]interface{})
	health["client_connected"] = s.client.IsConnected()
	health["protection"] = s.protection.HealthCheck()
	health["safe_mode_level"] = s.protection.SafeMode().CurrentLevel().String()
	return health
}

// ============ CONTEXT HELPERS ============

type contextKey string

const userIDKey contextKey = "user_id"

// WithUserID - Thêm user ID vào context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// extractUserID - Lấy user ID từ context
func extractUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(userIDKey).(string); ok {
		return userID
	}
	return "anonymous"
}

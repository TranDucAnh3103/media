package telegram

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
)

const (
	// MaxFileSize - Giới hạn file size của Telegram (2GB)
	MaxFileSize = 2 * 1024 * 1024 * 1024
	// ChunkSize - Kích thước mỗi chunk khi upload (512KB)
	ChunkSize = 512 * 1024
	// LargeFileThreshold - Ngưỡng file lớn cần chunked upload (10MB)
	LargeFileThreshold = 10 * 1024 * 1024
)

// TelegramUploader - Xử lý upload video lên Telegram
type TelegramUploader struct {
	client *TelegramClient
	mu     sync.Mutex
}

// NewTelegramUploader - Tạo instance uploader mới
func NewTelegramUploader(client *TelegramClient) *TelegramUploader {
	return &TelegramUploader{
		client: client,
	}
}

// UploadVideo - Upload video lên Telegram channel
// DEPRECATED: Use UploadVideoDirect inside RunWithCallback instead
func (u *TelegramUploader) UploadVideo(ctx context.Context, req VideoUploadRequest) (*VideoUploadResult, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	// Kiểm tra kết nối
	if !u.client.IsConnected() {
		return nil, ErrNotConnected
	}

	// Gọi UploadVideoDirect (assume đang chạy trong RunWithCallback)
	return u.UploadVideoDirect(ctx, req)
}

// UploadVideoDirect - Upload video trực tiếp (dùng trong RunWithCallback)
// Method này PHẢI được gọi trong RunWithCallback để client còn sống
func (u *TelegramUploader) UploadVideoDirect(ctx context.Context, req VideoUploadRequest) (*VideoUploadResult, error) {
	// Mở file
	file, err := os.Open(req.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Lấy thông tin file
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Kiểm tra kích thước file
	if fileInfo.Size() > MaxFileSize {
		return nil, ErrFileTooLarge
	}

	api := u.client.GetAPI()
	if api == nil {
		return nil, ErrNotConnected
	}

	// Tạo progress tracker
	progressTracker := &uploadProgressTracker{
		totalSize:  fileInfo.Size(),
		progressCb: req.ProgressCb,
	}

	// Tạo uploader với progress tracking
	up := uploader.NewUploader(api).WithProgress(progressTracker)

	// Upload file
	inputFile, err := up.FromPath(ctx, req.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// Chuẩn bị media
	fileName := filepath.Base(req.FilePath)
	if req.FileName != "" {
		fileName = req.FileName
	}

	// Trích xuất metadata nếu chưa có
	duration := req.Duration
	width := req.Width
	height := req.Height
	if duration == 0 || width == 0 || height == 0 {
		metadata, err := ExtractVideoMetadata(req.FilePath)
		if err == nil && metadata != nil {
			if duration == 0 {
				duration = metadata.Duration
			}
			if width == 0 {
				width = metadata.Width
			}
			if height == 0 {
				height = metadata.Height
			}
		}
	}

	// Tạo video attributes với metadata đầy đủ
	videoAttr := &tg.DocumentAttributeVideo{
		SupportsStreaming: true,
		Duration:          float64(duration),
		W:                 width,
		H:                 height,
	}

	media := &tg.InputMediaUploadedDocument{
		File:     inputFile,
		MimeType: getMimeType(req.FilePath),
		Attributes: []tg.DocumentAttributeClass{
			&tg.DocumentAttributeFilename{
				FileName: fileName,
			},
			videoAttr,
		},
	}

	// Upload thumbnail nếu có
	if req.ThumbnailPath != "" {
		if thumbFile, err := up.FromPath(ctx, req.ThumbnailPath); err == nil {
			media.Thumb = thumbFile
		}
	}

	// Gửi video lên channel
	inputPeer := &tg.InputPeerChannel{
		ChannelID:  u.client.GetChannelID(),
		AccessHash: u.client.GetAccessHash(),
	}

	// Generate random ID (required by Telegram MTProto)
	randomID := generateRandomID()

	updates, err := api.MessagesSendMedia(ctx, &tg.MessagesSendMediaRequest{
		Peer:     inputPeer,
		Media:    media,
		Message:  req.Caption,
		RandomID: randomID,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to send media: %w", err)
	}

	// Xử lý response để lấy message ID và file info
	result, err := u.parseUploadResult(updates, fileInfo.Size())
	if err != nil {
		return nil, fmt.Errorf("failed to parse upload result: %w", err)
	}

	return result, nil
}

// UploadFromReader - Upload video từ io.Reader
func (u *TelegramUploader) UploadFromReader(ctx context.Context, reader io.Reader, fileName string, size int64, caption string, progressCb func(UploadProgress)) (*VideoUploadResult, error) {
	// Lưu tạm file ra disk
	tempDir := os.TempDir()
	tempPath := filepath.Join(tempDir, fileName)

	tempFile, err := os.Create(tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	// Copy data vào temp file
	_, err = io.Copy(tempFile, reader)
	tempFile.Close()
	if err != nil {
		os.Remove(tempPath)
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}

	defer os.Remove(tempPath)

	// Upload temp file
	return u.UploadVideo(ctx, VideoUploadRequest{
		FilePath:   tempPath,
		FileName:   fileName,
		Caption:    caption,
		ChannelID:  u.client.GetChannelID(),
		ProgressCb: progressCb,
	})
}

// parseUploadResult - Phân tích kết quả upload để lấy thông tin
func (u *TelegramUploader) parseUploadResult(updates tg.UpdatesClass, fileSize int64) (*VideoUploadResult, error) {
	result := &VideoUploadResult{
		FileSize: fileSize,
	}

	switch v := updates.(type) {
	case *tg.Updates:
		for _, update := range v.Updates {
			switch upd := update.(type) {
			case *tg.UpdateNewChannelMessage:
				msg, ok := upd.Message.(*tg.Message)
				if !ok {
					continue
				}
				result.MessageID = msg.ID

				// Lấy thông tin media
				if media, ok := msg.Media.(*tg.MessageMediaDocument); ok {
					if doc, ok := media.Document.(*tg.Document); ok {
						result.FileID = fmt.Sprintf("%d", doc.ID)
						result.MimeType = doc.MimeType
						result.FileSize = doc.Size

						// Lấy thông tin video attributes
						for _, attr := range doc.Attributes {
							if videoAttr, ok := attr.(*tg.DocumentAttributeVideo); ok {
								result.Duration = int(videoAttr.Duration)
								result.Width = videoAttr.W
								result.Height = videoAttr.H
								break
							}
						}
					}
				}
				return result, nil
			}
		}
	case *tg.UpdateShortSentMessage:
		result.MessageID = v.ID
		return result, nil
	}

	return result, nil
}

// uploadProgressTracker - Track tiến trình upload
type uploadProgressTracker struct {
	totalSize  int64
	uploaded   int64
	progressCb func(UploadProgress)
}

func (t *uploadProgressTracker) Chunk(_ context.Context, state uploader.ProgressState) error {
	t.uploaded = state.Uploaded

	if t.progressCb != nil {
		progress := float64(state.Uploaded) / float64(t.totalSize) * 100
		t.progressCb(UploadProgress{
			BytesUploaded: state.Uploaded,
			TotalBytes:    t.totalSize,
			Percent:       progress,
			Stage:         "uploading",
		})
	}

	return nil
}

// getMimeType - Lấy MIME type từ extension
func getMimeType(filePath string) string {
	ext := filepath.Ext(filePath)
	mimeTypes := map[string]string{
		".mp4":  "video/mp4",
		".webm": "video/webm",
		".mov":  "video/quicktime",
		".avi":  "video/x-msvideo",
		".mkv":  "video/x-matroska",
		".m4v":  "video/x-m4v",
		".wmv":  "video/x-ms-wmv",
		".flv":  "video/x-flv",
	}

	if mimeType, ok := mimeTypes[ext]; ok {
		return mimeType
	}
	return "video/mp4"
}

// generateRandomID - Generate a cryptographically secure random int64 for Telegram MTProto
// This is required for messages.sendMedia to prevent RANDOM_ID_EMPTY error
func generateRandomID() int64 {
	var buf [8]byte
	_, err := rand.Read(buf[:])
	if err != nil {
		// Fallback: use time-based if crypto/rand fails (should never happen)
		return int64(binary.BigEndian.Uint64(buf[:]))
	}
	randomID := int64(binary.BigEndian.Uint64(buf[:]))
	// Ensure non-zero (extremely rare edge case)
	if randomID == 0 {
		randomID = 1
	}
	return randomID
}

// DeleteVideo - Xóa video từ Telegram channel
func (u *TelegramUploader) DeleteVideo(ctx context.Context, messageID int) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if !u.client.IsConnected() {
		return ErrNotConnected
	}

	return u.client.RunWithClient(ctx, func(ctx context.Context) error {
		api := u.client.GetAPI()
		if api == nil {
			return ErrNotConnected
		}

		_, err := api.ChannelsDeleteMessages(ctx, &tg.ChannelsDeleteMessagesRequest{
			Channel: u.client.GetInputChannel(),
			ID:      []int{messageID},
		})

		if err != nil {
			return fmt.Errorf("failed to delete message: %w", err)
		}

		return nil
	})
}

// GetVideoInfo - Lấy thông tin video từ message
func (u *TelegramUploader) GetVideoInfo(ctx context.Context, messageID int) (*MessageInfo, error) {
	if !u.client.IsConnected() {
		return nil, ErrNotConnected
	}

	var info *MessageInfo

	err := u.client.RunWithClient(ctx, func(ctx context.Context) error {
		api := u.client.GetAPI()
		if api == nil {
			return ErrNotConnected
		}

		// Lấy message
		messages, err := api.ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
			Channel: u.client.GetInputChannel(),
			ID:      []tg.InputMessageClass{&tg.InputMessageID{ID: messageID}},
		})

		if err != nil {
			return fmt.Errorf("failed to get message: %w", err)
		}

		channelMessages, ok := messages.(*tg.MessagesChannelMessages)
		if !ok || len(channelMessages.Messages) == 0 {
			return ErrMessageNotFound
		}

		msg, ok := channelMessages.Messages[0].(*tg.Message)
		if !ok {
			return ErrMessageNotFound
		}

		media, ok := msg.Media.(*tg.MessageMediaDocument)
		if !ok {
			return ErrFileNotFound
		}

		doc, ok := media.Document.(*tg.Document)
		if !ok {
			return ErrFileNotFound
		}

		info = &MessageInfo{
			ID:        messageID,
			ChannelID: u.client.GetChannelID(),
			FileID:    fmt.Sprintf("%d", doc.ID),
			FileSize:  doc.Size,
			MimeType:  doc.MimeType,
		}

		// Lấy duration từ attributes
		for _, attr := range doc.Attributes {
			if videoAttr, ok := attr.(*tg.DocumentAttributeVideo); ok {
				info.Duration = int(videoAttr.Duration)
				break
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return info, nil
}

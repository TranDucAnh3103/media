package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TelegramChannelVideo - Video được đồng bộ từ Telegram channel
// Mỗi video là một record riêng biệt, kể cả khi được gửi trong album
type TelegramChannelVideo struct {
	ID                  primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TelegramMessageID   int                `json:"telegram_message_id" bson:"telegram_message_id"`     // Telegram message ID
	TelegramGroupedID   int64              `json:"telegram_grouped_id" bson:"telegram_grouped_id"`     // Album/group ID (nếu video thuộc album)
	TelegramChannelID   int64              `json:"telegram_channel_id" bson:"telegram_channel_id"`     // Channel ID
	TelegramFileID      string             `json:"telegram_file_id" bson:"telegram_file_id"`           // Telegram file identifier
	TelegramFileRef     []byte             `json:"telegram_file_ref" bson:"telegram_file_ref"`         // Telegram file reference (có thể hết hạn)
	TelegramAccessHash  int64              `json:"telegram_access_hash" bson:"telegram_access_hash"`   // Access hash của document
	Caption             string             `json:"caption" bson:"caption"`                             // Caption của message
	Duration            int                `json:"duration" bson:"duration"`                           // Duration in seconds
	FileSize            int64              `json:"file_size" bson:"file_size"`                         // File size in bytes
	Width               int                `json:"width" bson:"width"`                                 // Video width
	Height              int                `json:"height" bson:"height"`                               // Video height
	MimeType            string             `json:"mime_type" bson:"mime_type"`                         // MIME type
	FileName            string             `json:"file_name" bson:"file_name"`                         // Original filename
	TelegramMessageDate time.Time          `json:"telegram_message_date" bson:"telegram_message_date"` // Ngày gửi message trên Telegram
	SyncedAt            time.Time          `json:"synced_at" bson:"synced_at"`                         // Ngày đồng bộ vào database
	IsPublished         bool               `json:"is_published" bson:"is_published"`                   // Đã publish thành Video chính chưa
	PublishedVideoID    primitive.ObjectID `json:"published_video_id" bson:"published_video_id"`       // ID của Video đã publish
	ThumbnailFileID     string             `json:"thumbnail_file_id" bson:"thumbnail_file_id"`         // Thumbnail file ID (nếu có)
	HasSpoiler          bool               `json:"has_spoiler" bson:"has_spoiler"`                     // Video có spoiler hay không
	SupportsStreaming   bool               `json:"supports_streaming" bson:"supports_streaming"`       // Video hỗ trợ streaming
}

// TelegramSyncStatus - Trạng thái đồng bộ
type TelegramSyncStatus struct {
	IsRunning       bool      `json:"is_running"`
	LastSyncAt      time.Time `json:"last_sync_at"`
	TotalSynced     int       `json:"total_synced"`
	NewVideosFound  int       `json:"new_videos_found"`
	ErrorCount      int       `json:"error_count"`
	LastError       string    `json:"last_error,omitempty"`
	CurrentProgress int       `json:"current_progress"` // Số message đã xử lý
	TotalMessages   int       `json:"total_messages"`   // Tổng số message cần xử lý
}

// TelegramSyncRequest - Request đồng bộ
type TelegramSyncRequest struct {
	ChannelID int64 `json:"channel_id"` // Optional: override default channel
	Limit     int   `json:"limit"`      // Số message tối đa cần scan (0 = tất cả)
	OffsetID  int   `json:"offset_id"`  // Bắt đầu từ message ID này (0 = mới nhất)
}

// TelegramSyncResponse - Response đồng bộ
type TelegramSyncResponse struct {
	Success        bool   `json:"success"`
	Message        string `json:"message"`
	NewVideosCount int    `json:"new_videos_count"`
	TotalScanned   int    `json:"total_scanned"`
	SyncDuration   string `json:"sync_duration"`
}

// TelegramChannelVideoListResponse - Response danh sách video
type TelegramChannelVideoListResponse struct {
	Videos     []TelegramChannelVideo `json:"videos"`
	Total      int64                  `json:"total"`
	Page       int                    `json:"page"`
	Limit      int                    `json:"limit"`
	TotalPages int                    `json:"total_pages"`
}

// ToVideo - Convert TelegramChannelVideo thành Video để publish
func (tcv *TelegramChannelVideo) ToVideo(title string, uploaderID primitive.ObjectID) Video {
	now := time.Now()
	return Video{
		ID:                primitive.NewObjectID(),
		Title:             title,
		Description:       tcv.Caption,
		StorageProvider:   StorageProviderTelegram,
		TelegramChannelID: tcv.TelegramChannelID,
		TelegramMessageID: tcv.TelegramMessageID,
		TelegramFileID:    tcv.TelegramFileID,
		MimeType:          tcv.MimeType,
		Duration:          tcv.Duration,
		FileSize:          tcv.FileSize,
		Width:             tcv.Width,
		Height:            tcv.Height,
		Status:            "ready",
		UploadedBy:        uploaderID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

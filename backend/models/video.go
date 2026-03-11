package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Storage provider constants
const (
	StorageProviderCloudinary = "cloudinary" // Legacy
	StorageProviderMega       = "mega"       // Legacy
	StorageProviderTelegram   = "telegram"   // New
)

// Video model - Quản lý thông tin video
type Video struct {
	ID                 primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Title              string             `json:"title" bson:"title" validate:"required"`
	Description        string             `json:"description" bson:"description"`
	Thumbnail          string             `json:"thumbnail" bson:"thumbnail"`                       // Cloudinary URL
	VideoURL           string             `json:"video_url" bson:"video_url"`                       // Cloudinary hoặc Mega URL (legacy)
	CloudinaryPublicID string             `json:"cloudinary_public_id" bson:"cloudinary_public_id"` // Cloudinary Public ID for deletion (legacy)
	MegaHash           string             `json:"mega_hash" bson:"mega_hash"`                       // Mega file hash for identification (legacy)
	StorageType        string             `json:"storage_type" bson:"storage_type"`                 // Deprecated: use StorageProvider
	StorageProvider    string             `json:"storage_provider" bson:"storage_provider"`         // cloudinary, mega, telegram

	// Telegram-specific fields
	TelegramChannelID int64  `json:"telegram_channel_id" bson:"telegram_channel_id"` // Telegram channel ID
	TelegramMessageID int    `json:"telegram_message_id" bson:"telegram_message_id"` // Telegram message ID
	TelegramFileID    string `json:"telegram_file_id" bson:"telegram_file_id"`       // Telegram file ID
	MimeType          string `json:"mime_type" bson:"mime_type"`                     // Video MIME type

	Duration     int                `json:"duration" bson:"duration"`           // Giây
	DurationType string             `json:"duration_type" bson:"duration_type"` // short (<5p), medium (5-10p), long (>10p)
	Quality      string             `json:"quality" bson:"quality"`             // 360p, 480p, 720p, 1080p
	FileSize     int64              `json:"file_size" bson:"file_size"`         // Bytes
	Width        int                `json:"width" bson:"width"`                 // Video width
	Height       int                `json:"height" bson:"height"`               // Video height
	Tags         []string           `json:"tags" bson:"tags"`
	Genres       []string           `json:"genres" bson:"genres"`
	Views        int64              `json:"views" bson:"views"`
	Likes        int64              `json:"likes" bson:"likes"`
	Dislikes     int64              `json:"dislikes" bson:"dislikes"`
	Comments     []Comment          `json:"comments" bson:"comments"`
	Status       string             `json:"status" bson:"status"` // processing, ready, error
	UploadedBy   primitive.ObjectID `json:"uploaded_by" bson:"uploaded_by"`
	CreatedAt    time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at" bson:"updated_at"`
}

// GetStreamURL - Lấy URL stream dựa trên storage provider
func (v *Video) GetStreamURL() string {
	switch v.StorageProvider {
	case StorageProviderTelegram:
		return "/api/videos/stream/" + v.ID.Hex()
	case StorageProviderMega:
		if v.MegaHash != "" {
			return "/api/videos/stream/mega/" + v.MegaHash
		}
		return v.VideoURL
	case StorageProviderCloudinary:
		return v.VideoURL
	default:
		// Legacy: check if mega_hash exists
		if v.MegaHash != "" {
			return "/api/videos/stream/mega/" + v.MegaHash
		}
		return v.VideoURL
	}
}

// Comment - Bình luận video
type Comment struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`
	Username  string             `json:"username" bson:"username"`
	Content   string             `json:"content" bson:"content"`
	Likes     int64              `json:"likes" bson:"likes"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
}

// VideoCreateRequest - Request tạo video mới
type VideoCreateRequest struct {
	Title       string   `json:"title" validate:"required"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Genres      []string `json:"genres"`
}

// VideoUpdateRequest - Request cập nhật video
type VideoUpdateRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Genres      []string `json:"genres"`
	Thumbnail   string   `json:"thumbnail"`
}

// VideoFilter - Filter cho danh sách video
type VideoFilter struct {
	Genres       []string `query:"genres"`
	DurationType string   `query:"duration_type"` // short, medium, long
	Quality      string   `query:"quality"`
	Status       string   `query:"status"`
	Search       string   `query:"search"`
	SortBy       string   `query:"sort_by"` // views, likes, created_at
	Order        string   `query:"order"`   // asc, desc
	Page         int      `query:"page"`
	Limit        int      `query:"limit"`
}

// VideoListResponse - Response danh sách video với pagination
type VideoListResponse struct {
	Videos     []Video `json:"videos"`
	Total      int64   `json:"total"`
	Page       int     `json:"page"`
	Limit      int     `json:"limit"`
	TotalPages int     `json:"total_pages"`
}

// UploadProgress - Tiến trình upload
type UploadProgress struct {
	ID       string  `json:"id"`
	Status   string  `json:"status"` // uploading, processing, completed, error
	Progress float64 `json:"progress"`
	Message  string  `json:"message"`
	VideoID  string  `json:"video_id,omitempty"`
}

package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User model - Quản lý thông tin người dùng
type User struct {
	ID           primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	Username     string               `json:"username" bson:"username" validate:"required,min=3,max=50"`
	Email        string               `json:"email" bson:"email" validate:"required,email"`
	Password     string               `json:"-" bson:"password" validate:"required,min=6"`
	Avatar       string               `json:"avatar" bson:"avatar"`
	Role         string               `json:"role" bson:"role"` // admin, user
	LikedVideos  []primitive.ObjectID `json:"liked_videos" bson:"liked_videos"`
	LikedComics  []primitive.ObjectID `json:"liked_comics" bson:"liked_comics"`
	Bookmarks    []Bookmark           `json:"bookmarks" bson:"bookmarks"`
	Playlists    []Playlist           `json:"playlists" bson:"playlists"`
	WatchHistory []WatchHistoryItem   `json:"watch_history" bson:"watch_history"`
	CreatedAt    time.Time            `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at" bson:"updated_at"`
}

// Bookmark - Lưu vị trí đánh dấu cho truyện/video
type Bookmark struct {
	ContentID   primitive.ObjectID `json:"content_id" bson:"content_id"`
	ContentType string             `json:"content_type" bson:"content_type"` // comic, video
	Page        int                `json:"page" bson:"page"`                 // Trang đang đọc (comic)
	Chapter     int                `json:"chapter" bson:"chapter"`           // Chapter (comic)
	Timestamp   float64            `json:"timestamp" bson:"timestamp"`       // Giây đang xem (video)
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
}

// Playlist - Danh sách phát của user
type Playlist struct {
	ID        primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	Name      string               `json:"name" bson:"name"`
	VideoIDs  []primitive.ObjectID `json:"video_ids" bson:"video_ids"`
	CreatedAt time.Time            `json:"created_at" bson:"created_at"`
}

// WatchHistoryItem - Lịch sử xem
type WatchHistoryItem struct {
	ContentID   primitive.ObjectID `json:"content_id" bson:"content_id"`
	ContentType string             `json:"content_type" bson:"content_type"`
	WatchedAt   time.Time          `json:"watched_at" bson:"watched_at"`
	Progress    float64            `json:"progress" bson:"progress"` // % đã xem
}

// UserResponse - Response không bao gồm password
type UserResponse struct {
	ID           primitive.ObjectID `json:"id"`
	Username     string             `json:"username"`
	Email        string             `json:"email"`
	Avatar       string             `json:"avatar"`
	Role         string             `json:"role"`
	Bookmarks    []Bookmark         `json:"bookmarks"`
	Playlists    []Playlist         `json:"playlists"`
	WatchHistory []WatchHistoryItem `json:"watch_history"`
	CreatedAt    time.Time          `json:"created_at"`
}

// LoginRequest - Request đăng nhập
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// RegisterRequest - Request đăng ký
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// ToResponse chuyển User thành UserResponse (bỏ password)
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:           u.ID,
		Username:     u.Username,
		Email:        u.Email,
		Avatar:       u.Avatar,
		Role:         u.Role,
		Bookmarks:    u.Bookmarks,
		Playlists:    u.Playlists,
		WatchHistory: u.WatchHistory,
		CreatedAt:    u.CreatedAt,
	}
}

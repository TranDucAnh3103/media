package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Comic model - Quản lý thông tin truyện tranh
type Comic struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Title       string             `json:"title" bson:"title" validate:"required"`
	Description string             `json:"description" bson:"description"`
	Author      string             `json:"author" bson:"author"`
	CoverImage  string             `json:"cover_image" bson:"cover_image"` // URL ảnh bìa từ Cloudinary
	Tags        []string           `json:"tags" bson:"tags"`
	Genres      []string           `json:"genres" bson:"genres"`
	Status      string             `json:"status" bson:"status"` // ongoing, completed
	Chapters    []Chapter          `json:"chapters" bson:"chapters"`
	Views       int64              `json:"views" bson:"views"`
	Likes       int64              `json:"likes" bson:"likes"`
	Rating      float64            `json:"rating" bson:"rating"`
	RatingCount int64              `json:"rating_count" bson:"rating_count"`
	UploadedBy  primitive.ObjectID `json:"uploaded_by" bson:"uploaded_by"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}

// Chapter - Một chapter trong truyện
type Chapter struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Number     int                `json:"number" bson:"number"`
	Title      string             `json:"title" bson:"title"`
	Images     []ComicImage       `json:"images" bson:"images"` // Danh sách ảnh từ Cloudinary
	Views      int64              `json:"views" bson:"views"`
	UploadedAt time.Time          `json:"uploaded_at" bson:"uploaded_at"`
}

// ComicImage - Thông tin một trang ảnh
type ComicImage struct {
	Page     int    `json:"page" bson:"page"`
	URL      string `json:"url" bson:"url"`             // Cloudinary URL
	PublicID string `json:"public_id" bson:"public_id"` // Cloudinary Public ID
	Width    int    `json:"width" bson:"width"`
	Height   int    `json:"height" bson:"height"`
}

// ComicCreateRequest - Request tạo truyện mới
type ComicCreateRequest struct {
	Title       string   `json:"title" validate:"required"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Tags        []string `json:"tags"`
	Genres      []string `json:"genres"`
	Status      string   `json:"status"`
}

// ComicUpdateRequest - Request cập nhật truyện
type ComicUpdateRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Tags        []string `json:"tags"`
	Genres      []string `json:"genres"`
	Status      string   `json:"status"`
	CoverImage  string   `json:"cover_image"`
}

// ComicFilter - Filter cho danh sách truyện
type ComicFilter struct {
	Genres []string `query:"genres"`
	Status string   `query:"status"`
	Author string   `query:"author"`
	Search string   `query:"search"`
	SortBy string   `query:"sort_by"` // views, rating, created_at
	Order  string   `query:"order"`   // asc, desc
	Page   int      `query:"page"`
	Limit  int      `query:"limit"`
}

// ComicListResponse - Response danh sách truyện với pagination
type ComicListResponse struct {
	Comics     []Comic `json:"comics"`
	Total      int64   `json:"total"`
	Page       int     `json:"page"`
	Limit      int     `json:"limit"`
	TotalPages int     `json:"total_pages"`
}

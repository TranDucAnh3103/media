package telegram

// VideoUploadRequest - Request để upload video lên Telegram
type VideoUploadRequest struct {
	FilePath      string               // Đường dẫn file video
	FileName      string               // Tên file
	Caption       string               // Caption cho video
	ChannelID     int64                // ID của Telegram channel
	ProgressCb    func(UploadProgress) // Callback tiến trình upload
	ThumbnailPath string               // Đường dẫn thumbnail (optional, sẽ tự extract nếu không có)
	Duration      int                  // Thời lượng video (giây, optional, sẽ tự extract)
	Width         int                  // Chiều rộng video (optional, sẽ tự extract)
	Height        int                  // Chiều cao video (optional, sẽ tự extract)
}

// VideoUploadResult - Kết quả upload video lên Telegram
type VideoUploadResult struct {
	MessageID    int    `json:"message_id"`    // ID tin nhắn trên channel
	FileID       string `json:"file_id"`       // ID file trên Telegram
	FileSize     int64  `json:"file_size"`     // Kích thước file (bytes)
	MimeType     string `json:"mime_type"`     // MIME type của video
	Duration     int    `json:"duration"`      // Thời lượng video (giây)
	Width        int    `json:"width"`         // Chiều rộng video
	Height       int    `json:"height"`        // Chiều cao video
	ThumbnailURL string `json:"thumbnail_url"` // URL thumbnail (từ Cloudinary)
}

// UploadProgress - Tiến trình upload
type UploadProgress struct {
	BytesUploaded int64   `json:"bytes_uploaded"` // Số bytes đã upload
	TotalBytes    int64   `json:"total_bytes"`    // Tổng số bytes
	Percent       float64 `json:"percent"`        // Phần trăm hoàn thành
	Stage         string  `json:"stage"`          // Giai đoạn: extracting, thumbnail, uploading, processing, completed
}

// StreamRequest - Request stream video từ Telegram
type StreamRequest struct {
	ChannelID int64  // ID channel chứa video
	MessageID int    // ID tin nhắn chứa video
	FileID    string // ID file trên Telegram
	Start     int64  // Byte offset bắt đầu
	End       int64  // Byte offset kết thúc
}

// StreamResponse - Response stream video
type StreamResponse struct {
	ContentType   string `json:"content_type"`   // MIME type
	ContentLength int64  `json:"content_length"` // Độ dài content
	TotalSize     int64  `json:"total_size"`     // Tổng kích thước file
}

// MessageInfo - Thông tin tin nhắn Telegram
type MessageInfo struct {
	ID        int    `json:"id"`
	ChannelID int64  `json:"channel_id"`
	FileID    string `json:"file_id"`
	FileRef   []byte `json:"file_ref"`
	FileSize  int64  `json:"file_size"`
	MimeType  string `json:"mime_type"`
	Duration  int    `json:"duration"`
}

// Config - Cấu hình Telegram client
type Config struct {
	APIID           int    // Telegram API ID
	APIHash         string // Telegram API Hash
	BotToken        string // Token bot (nếu dùng bot mode)
	PhoneNumber     string // Số điện thoại (nếu dùng user mode)
	SessionPath     string // Đường dẫn lưu session
	ChannelID       int64  // ID channel mặc định để lưu video
	ChannelUsername string // Username channel (vd: @my_channel) - để resolve nếu không tìm được bằng ID
}

// FileLocation - Vị trí file trên Telegram
type FileLocation struct {
	ChannelID  int64
	MessageID  int
	FileID     string
	AccessHash int64
	FileRef    []byte
}

// ChunkInfo - Thông tin chunk khi download
type ChunkInfo struct {
	Offset int64
	Limit  int
	Data   []byte
}

// TelegramVideoMeta - Metadata video từ Telegram
type TelegramVideoMeta struct {
	FileID    string `json:"file_id"`
	FileSize  int64  `json:"file_size"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Duration  int    `json:"duration"`
	MimeType  string `json:"mime_type"`
	Thumbnail string `json:"thumbnail,omitempty"`
}

// TelegramUploadProgress - Progress upload chi tiết
type TelegramUploadProgress struct {
	VideoID   string  `json:"video_id"`
	Stage     string  `json:"stage"`    // extracting, thumbnail, uploading, processing, completed
	Progress  float64 `json:"progress"` // 0-100
	BytesSent int64   `json:"bytes_sent"`
	TotalSize int64   `json:"total_size"`
	Error     string  `json:"error,omitempty"`
}

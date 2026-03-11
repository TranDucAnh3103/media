package telegram

import "errors"

// Các lỗi của Telegram service
var (
	// Connection errors
	ErrNotConnected     = errors.New("telegram client not connected")
	ErrConnectionFailed = errors.New("failed to connect to Telegram")
	ErrAuthRequired     = errors.New("authentication required")
	ErrSessionExpired   = errors.New("session expired, re-authentication required")

	// File errors
	ErrFileNotFound    = errors.New("file not found in message")
	ErrMessageNotFound = errors.New("message not found")
	ErrChannelNotFound = errors.New("channel not found")
	ErrInvalidFileID   = errors.New("invalid file ID")

	// Upload errors
	ErrUploadFailed    = errors.New("upload failed")
	ErrUploadCanceled  = errors.New("upload was canceled")
	ErrFileTooLarge    = errors.New("file size exceeds maximum limit (2GB)")
	ErrInvalidFileType = errors.New("invalid file type for video upload")
	ErrUploadTimeout   = errors.New("upload timeout")

	// Streaming errors
	ErrRangeNotSatisfiable = errors.New("requested range not satisfiable")
	ErrStreamFailed        = errors.New("stream failed")
	ErrInvalidRange        = errors.New("invalid byte range")
	ErrChunkDownloadFailed = errors.New("chunk download failed")

	// Rate limiting errors
	ErrRateLimited = errors.New("rate limited by Telegram")
	ErrFloodWait   = errors.New("flood wait - too many requests")

	// Configuration errors
	ErrMissingConfig  = errors.New("missing Telegram configuration")
	ErrInvalidConfig  = errors.New("invalid Telegram configuration")
	ErrMissingAPIID   = errors.New("missing TELEGRAM_API_ID")
	ErrMissingAPIHash = errors.New("missing TELEGRAM_API_HASH")
	ErrMissingChannel = errors.New("missing TELEGRAM_CHANNEL_ID")
)

// TelegramError - Custom error type với thông tin chi tiết
type TelegramError struct {
	Code    int
	Message string
	Err     error
}

func (e *TelegramError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *TelegramError) Unwrap() error {
	return e.Err
}

// NewTelegramError - Tạo lỗi mới
func NewTelegramError(code int, message string, err error) *TelegramError {
	return &TelegramError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Error codes
const (
	ErrCodeConnection   = 1000
	ErrCodeAuth         = 1001
	ErrCodeUpload       = 2000
	ErrCodeDownload     = 2001
	ErrCodeStream       = 2002
	ErrCodeRateLimit    = 3000
	ErrCodeConfig       = 4000
	ErrCodeFileNotFound = 5000
)

// IsRetryable - Kiểm tra lỗi có thể retry không
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Các lỗi có thể retry
	retryableErrors := []error{
		ErrConnectionFailed,
		ErrRateLimited,
		ErrFloodWait,
		ErrUploadTimeout,
		ErrChunkDownloadFailed,
	}

	for _, retryable := range retryableErrors {
		if errors.Is(err, retryable) {
			return true
		}
	}

	return false
}

// IsAuthError - Kiểm tra lỗi liên quan đến authentication
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}

	authErrors := []error{
		ErrAuthRequired,
		ErrSessionExpired,
	}

	for _, authErr := range authErrors {
		if errors.Is(err, authErr) {
			return true
		}
	}

	return false
}

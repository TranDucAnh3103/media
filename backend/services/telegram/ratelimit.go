package telegram

import (
	"context"
	"sync"
	"time"
)

// RateLimitConfig - Cấu hình rate limiting
type RateLimitConfig struct {
	// Giới hạn số upload mỗi phút
	MaxUploadsPerMinute int
	// Giới hạn số message mỗi giây cho mỗi user
	MaxMessagesPerSecondPerUser int
	// Giới hạn tổng số request API mỗi giây
	MaxGlobalRequestsPerSecond int
	// Thời gian window để tính rate (mặc định 1 phút)
	WindowDuration time.Duration
}

// DefaultRateLimitConfig - Cấu hình mặc định an toàn
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		MaxUploadsPerMinute:         20,
		MaxMessagesPerSecondPerUser: 1,
		MaxGlobalRequestsPerSecond:  30,
		WindowDuration:              time.Minute,
	}
}

// RateLimiter - Rate limiter cho Telegram API
type RateLimiter struct {
	config RateLimitConfig

	// Upload rate limiting
	uploadCount  int
	uploadWindow time.Time
	uploadMu     sync.Mutex

	// Per-user message rate limiting
	userLastMessage map[string]time.Time
	userMu          sync.RWMutex

	// Global request rate limiting (token bucket)
	globalTokens     float64
	globalLastRefill time.Time
	globalMu         sync.Mutex

	// Metrics (internal)
	metrics *rateLimitMetricsInternal
}

// RateLimitMetrics - Thống kê rate limiting (DTO - không chứa mutex, an toàn khi copy/return)
type RateLimitMetrics struct {
	TotalRequests     int64
	ThrottledRequests int64
	UploadRequests    int64
	ThrottledUploads  int64
	LastThrottleTime  time.Time
}

// rateLimitMetricsInternal - internal struct với mutex
type rateLimitMetricsInternal struct {
	RateLimitMetrics
	mu sync.RWMutex
}

// NewRateLimiter - Tạo rate limiter mới
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		config:           config,
		uploadWindow:     time.Now(),
		userLastMessage:  make(map[string]time.Time),
		globalTokens:     float64(config.MaxGlobalRequestsPerSecond),
		globalLastRefill: time.Now(),
		metrics:          &rateLimitMetricsInternal{},
	}
}

// WaitForUploadSlot - Đợi đến khi có slot upload khả dụng
// Trả về error nếu context bị cancel
func (r *RateLimiter) WaitForUploadSlot(ctx context.Context) error {
	for {
		if r.tryAcquireUploadSlot() {
			return nil
		}

		// Tính thời gian chờ đến window mới
		r.uploadMu.Lock()
		waitDuration := time.Until(r.uploadWindow.Add(r.config.WindowDuration))
		r.uploadMu.Unlock()

		if waitDuration <= 0 {
			continue
		}

		// Log throttle event
		r.recordThrottle()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitDuration):
			// Thử lại sau khi window reset
		}
	}
}

// tryAcquireUploadSlot - Thử lấy slot upload
func (r *RateLimiter) tryAcquireUploadSlot() bool {
	r.uploadMu.Lock()
	defer r.uploadMu.Unlock()

	now := time.Now()

	// Reset window nếu đã hết thời gian
	if now.Sub(r.uploadWindow) >= r.config.WindowDuration {
		r.uploadCount = 0
		r.uploadWindow = now
	}

	// Kiểm tra có còn slot không
	if r.uploadCount >= r.config.MaxUploadsPerMinute {
		return false
	}

	r.uploadCount++
	r.metrics.mu.Lock()
	r.metrics.UploadRequests++
	r.metrics.mu.Unlock()

	return true
}

// WaitForUserSlot - Đợi đến khi user có thể gửi message
func (r *RateLimiter) WaitForUserSlot(ctx context.Context, userID string) error {
	for {
		if r.tryAcquireUserSlot(userID) {
			return nil
		}

		// Tính thời gian chờ
		r.userMu.RLock()
		lastMsg := r.userLastMessage[userID]
		r.userMu.RUnlock()

		waitDuration := time.Until(lastMsg.Add(time.Second / time.Duration(r.config.MaxMessagesPerSecondPerUser)))

		if waitDuration <= 0 {
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitDuration):
		}
	}
}

// tryAcquireUserSlot - Thử lấy slot cho user
func (r *RateLimiter) tryAcquireUserSlot(userID string) bool {
	r.userMu.Lock()
	defer r.userMu.Unlock()

	now := time.Now()
	minInterval := time.Second / time.Duration(r.config.MaxMessagesPerSecondPerUser)

	lastMsg, exists := r.userLastMessage[userID]
	if exists && now.Sub(lastMsg) < minInterval {
		return false
	}

	r.userLastMessage[userID] = now
	return true
}

// WaitForGlobalSlot - Đợi slot global request
func (r *RateLimiter) WaitForGlobalSlot(ctx context.Context) error {
	for {
		if r.tryAcquireGlobalToken() {
			return nil
		}

		// Đợi để có thêm token
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// tryAcquireGlobalToken - Token bucket algorithm
func (r *RateLimiter) tryAcquireGlobalToken() bool {
	r.globalMu.Lock()
	defer r.globalMu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.globalLastRefill).Seconds()
	r.globalLastRefill = now

	// Refill tokens
	r.globalTokens += elapsed * float64(r.config.MaxGlobalRequestsPerSecond)
	if r.globalTokens > float64(r.config.MaxGlobalRequestsPerSecond) {
		r.globalTokens = float64(r.config.MaxGlobalRequestsPerSecond)
	}

	// Try to consume a token
	if r.globalTokens >= 1 {
		r.globalTokens--
		r.metrics.mu.Lock()
		r.metrics.TotalRequests++
		r.metrics.mu.Unlock()
		return true
	}

	return false
}

// recordThrottle - Ghi nhận sự kiện throttle
func (r *RateLimiter) recordThrottle() {
	r.metrics.mu.Lock()
	defer r.metrics.mu.Unlock()
	r.metrics.ThrottledRequests++
	r.metrics.LastThrottleTime = time.Now()
}

// GetMetrics - Lấy metrics hiện tại (trả về DTO không chứa mutex, an toàn)
func (r *RateLimiter) GetMetrics() RateLimitMetrics {
	r.metrics.mu.RLock()
	defer r.metrics.mu.RUnlock()
	return r.metrics.RateLimitMetrics
}

// GetCurrentUploadCount - Lấy số upload hiện tại trong window
func (r *RateLimiter) GetCurrentUploadCount() int {
	r.uploadMu.Lock()
	defer r.uploadMu.Unlock()

	// Reset nếu window đã hết
	if time.Since(r.uploadWindow) >= r.config.WindowDuration {
		return 0
	}
	return r.uploadCount
}

// GetRemainingUploadSlots - Lấy số slot upload còn lại
func (r *RateLimiter) GetRemainingUploadSlots() int {
	r.uploadMu.Lock()
	defer r.uploadMu.Unlock()

	if time.Since(r.uploadWindow) >= r.config.WindowDuration {
		return r.config.MaxUploadsPerMinute
	}
	return r.config.MaxUploadsPerMinute - r.uploadCount
}

// CleanupUserSlots - Dọn dẹp user slots cũ (gọi định kỳ)
func (r *RateLimiter) CleanupUserSlots() {
	r.userMu.Lock()
	defer r.userMu.Unlock()

	threshold := time.Now().Add(-5 * time.Minute)
	for userID, lastMsg := range r.userLastMessage {
		if lastMsg.Before(threshold) {
			delete(r.userLastMessage, userID)
		}
	}
}

// AllowUpload - Kiểm tra và cho phép upload, trả về (allowed, waitTime)
func (r *RateLimiter) AllowUpload(userID string) (bool, time.Duration) {
	// Check global rate first
	if !r.tryAcquireGlobalToken() {
		return false, 50 * time.Millisecond
	}

	// Check user rate
	if !r.tryAcquireUserSlot(userID) {
		return false, time.Second / time.Duration(r.config.MaxMessagesPerSecondPerUser)
	}

	// Check upload rate
	if !r.tryAcquireUploadSlot() {
		r.uploadMu.Lock()
		waitDuration := time.Until(r.uploadWindow.Add(r.config.WindowDuration))
		r.uploadMu.Unlock()
		return false, waitDuration
	}

	return true, 0
}

// AllowRequest - Kiểm tra và cho phép request chung
func (r *RateLimiter) AllowRequest(userID string) bool {
	// Check global rate
	if !r.tryAcquireGlobalToken() {
		return false
	}

	// Check user rate
	if !r.tryAcquireUserSlot(userID) {
		return false
	}

	return true
}

// Stats - Lấy thống kê hiện tại
func (r *RateLimiter) Stats() RateLimitMetrics {
	return r.GetMetrics()
}

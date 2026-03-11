package telegram

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// ThrottlerConfig - Cấu hình throttler
type ThrottlerConfig struct {
	// Số worker upload đồng thời tối đa
	MaxConcurrentUploads int
	// Delay tối thiểu giữa các upload (giây)
	MinDelaySeconds int
	// Delay tối đa giữa các upload (giây)
	MaxDelaySeconds int
	// Ngưỡng burst để kích hoạt slowdown
	BurstThreshold int
	// Thời gian window để phát hiện burst
	BurstWindow time.Duration
	// Hệ số slowdown khi phát hiện burst (1.0 = không slowdown)
	BurstSlowdownFactor float64
}

// DefaultThrottlerConfig - Cấu hình mặc định
func DefaultThrottlerConfig() ThrottlerConfig {
	return ThrottlerConfig{
		MaxConcurrentUploads: 2,
		MinDelaySeconds:      3,
		MaxDelaySeconds:      8,
		BurstThreshold:       10,
		BurstWindow:          time.Minute,
		BurstSlowdownFactor:  2.0,
	}
}

// UploadThrottler - Throttler cho upload operations
type UploadThrottler struct {
	config ThrottlerConfig

	// Semaphore để giới hạn concurrent uploads
	semaphore chan struct{}

	// Track upload timestamps for burst detection
	uploadTimes []time.Time
	uploadMu    sync.Mutex

	// Last upload time for delay calculation
	lastUpload time.Time
	delayMu    sync.Mutex

	// Stats
	activeUploads   int32
	totalUploads    int64
	burstDetections int64

	// Slowdown state
	slowdownActive bool
	slowdownFactor float64
	slowdownMu     sync.RWMutex
}

// NewUploadThrottler - Tạo throttler mới
func NewUploadThrottler(config ThrottlerConfig) *UploadThrottler {
	return &UploadThrottler{
		config:      config,
		semaphore:   make(chan struct{}, config.MaxConcurrentUploads),
		uploadTimes: make([]time.Time, 0, config.BurstThreshold*2),
		lastUpload:  time.Now().Add(-time.Hour), // Cho phép upload ngay lập tức
	}
}

// Acquire - Lấy slot upload, đợi nếu cần
func (t *UploadThrottler) Acquire(ctx context.Context) error {
	// Wait for semaphore
	select {
	case t.semaphore <- struct{}{}:
		// Got slot
	case <-ctx.Done():
		return ctx.Err()
	}

	atomic.AddInt32(&t.activeUploads, 1)

	// Wait for delay between uploads
	if err := t.waitForDelay(ctx); err != nil {
		// Release slot if delay was cancelled
		t.release()
		return err
	}

	return nil
}

// Release - Giải phóng slot upload
func (t *UploadThrottler) Release() {
	t.release()
	t.recordUpload()
}

func (t *UploadThrottler) release() {
	atomic.AddInt32(&t.activeUploads, -1)
	<-t.semaphore
}

// waitForDelay - Đợi delay giữa các upload
func (t *UploadThrottler) waitForDelay(ctx context.Context) error {
	t.delayMu.Lock()
	lastUpload := t.lastUpload
	t.lastUpload = time.Now()
	t.delayMu.Unlock()

	// Calculate delay
	delay := t.calculateDelay()

	// Check if we need to wait
	elapsed := time.Since(lastUpload)
	if elapsed >= delay {
		return nil
	}

	waitDuration := delay - elapsed

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitDuration):
		return nil
	}
}

// calculateDelay - Tính delay với random và slowdown
func (t *UploadThrottler) calculateDelay() time.Duration {
	minDelay := t.config.MinDelaySeconds
	maxDelay := t.config.MaxDelaySeconds

	// Random delay trong khoảng [min, max]
	delaySeconds := minDelay
	if maxDelay > minDelay {
		delaySeconds = minDelay + rand.Intn(maxDelay-minDelay+1)
	}

	delay := time.Duration(delaySeconds) * time.Second

	// Apply slowdown factor
	t.slowdownMu.RLock()
	if t.slowdownActive && t.slowdownFactor > 1.0 {
		delay = time.Duration(float64(delay) * t.slowdownFactor)
	}
	t.slowdownMu.RUnlock()

	return delay
}

// recordUpload - Ghi nhận upload và check burst
func (t *UploadThrottler) recordUpload() {
	now := time.Now()

	t.uploadMu.Lock()
	defer t.uploadMu.Unlock()

	// Add current upload
	t.uploadTimes = append(t.uploadTimes, now)
	atomic.AddInt64(&t.totalUploads, 1)

	// Cleanup old uploads
	threshold := now.Add(-t.config.BurstWindow)
	filtered := make([]time.Time, 0, len(t.uploadTimes))
	for _, ts := range t.uploadTimes {
		if ts.After(threshold) {
			filtered = append(filtered, ts)
		}
	}
	t.uploadTimes = filtered

	// Check for burst
	if len(t.uploadTimes) >= t.config.BurstThreshold {
		t.activateSlowdown()
	}
}

// activateSlowdown - Kích hoạt slowdown mode
func (t *UploadThrottler) activateSlowdown() {
	t.slowdownMu.Lock()
	defer t.slowdownMu.Unlock()

	if !t.slowdownActive {
		t.slowdownActive = true
		t.slowdownFactor = t.config.BurstSlowdownFactor
		atomic.AddInt64(&t.burstDetections, 1)
	}
}

// DeactivateSlowdown - Tắt slowdown mode
func (t *UploadThrottler) DeactivateSlowdown() {
	t.slowdownMu.Lock()
	defer t.slowdownMu.Unlock()
	t.slowdownActive = false
	t.slowdownFactor = 1.0
}

// SetSlowdownFactor - Đặt hệ số slowdown
func (t *UploadThrottler) SetSlowdownFactor(factor float64) {
	t.slowdownMu.Lock()
	defer t.slowdownMu.Unlock()
	if factor > 1.0 {
		t.slowdownActive = true
		t.slowdownFactor = factor
	}
}

// IsSlowdownActive - Kiểm tra slowdown có đang active
func (t *UploadThrottler) IsSlowdownActive() bool {
	t.slowdownMu.RLock()
	defer t.slowdownMu.RUnlock()
	return t.slowdownActive
}

// GetActiveUploads - Lấy số upload đang active
func (t *UploadThrottler) GetActiveUploads() int {
	return int(atomic.LoadInt32(&t.activeUploads))
}

// GetTotalUploads - Lấy tổng số upload
func (t *UploadThrottler) GetTotalUploads() int64 {
	return atomic.LoadInt64(&t.totalUploads)
}

// GetBurstDetections - Lấy số lần phát hiện burst
func (t *UploadThrottler) GetBurstDetections() int64 {
	return atomic.LoadInt64(&t.burstDetections)
}

// GetUploadsInWindow - Lấy số upload trong window
func (t *UploadThrottler) GetUploadsInWindow() int {
	t.uploadMu.Lock()
	defer t.uploadMu.Unlock()

	now := time.Now()
	threshold := now.Add(-t.config.BurstWindow)
	count := 0
	for _, ts := range t.uploadTimes {
		if ts.After(threshold) {
			count++
		}
	}
	return count
}

// GetAvailableSlots - Lấy số slot còn trống
func (t *UploadThrottler) GetAvailableSlots() int {
	return t.config.MaxConcurrentUploads - int(atomic.LoadInt32(&t.activeUploads))
}

// UpdateConfig - Cập nhật config (cho safe mode)
func (t *UploadThrottler) UpdateConfig(maxConcurrent int, minDelay, maxDelay int) {
	// Note: Changing semaphore size at runtime is complex
	// Just update the delay config
	t.config.MinDelaySeconds = minDelay
	t.config.MaxDelaySeconds = maxDelay
	// For concurrent limit changes, would need to recreate semaphore
}

// SetMaxConcurrent - Thay đổi số concurrent uploads
func (t *UploadThrottler) SetMaxConcurrent(max int) {
	if max < 1 {
		max = 1
	}
	// Create new semaphore
	newSem := make(chan struct{}, max)

	// Transfer existing permits
	for i := 0; i < min(max, t.config.MaxConcurrentUploads); i++ {
		select {
		case t.semaphore <- struct{}{}:
			newSem <- struct{}{}
		default:
		}
	}

	t.semaphore = newSem
	t.config.MaxConcurrentUploads = max
}

// ThrottlerStats - Thống kê throttler
type ThrottlerStats struct {
	ActiveUploads   int
	TotalUploads    int64
	BurstDetections int64
	UploadsInWindow int
	AvailableSlots  int
	SlowdownActive  bool
	SlowdownFactor  float64
}

// GetStats - Lấy thống kê
func (t *UploadThrottler) GetStats() ThrottlerStats {
	t.slowdownMu.RLock()
	slowdownActive := t.slowdownActive
	slowdownFactor := t.slowdownFactor
	t.slowdownMu.RUnlock()

	return ThrottlerStats{
		ActiveUploads:   t.GetActiveUploads(),
		TotalUploads:    t.GetTotalUploads(),
		BurstDetections: t.GetBurstDetections(),
		UploadsInWindow: t.GetUploadsInWindow(),
		AvailableSlots:  t.GetAvailableSlots(),
		SlowdownActive:  slowdownActive,
		SlowdownFactor:  slowdownFactor,
	}
}

// Stats - Alias cho GetStats (dùng trong protection layer)
func (t *UploadThrottler) Stats() ThrottlerStats {
	return t.GetStats()
}

// EnableSlowdownMode - Bật slowdown mode với delay tối thiểu
func (t *UploadThrottler) EnableSlowdownMode(minDelay time.Duration) {
	t.slowdownMu.Lock()
	defer t.slowdownMu.Unlock()

	t.slowdownActive = true
	// Calculate factor based on desired minimum delay
	currentMinDelay := time.Duration(t.config.MinDelaySeconds) * time.Second
	if currentMinDelay > 0 && minDelay > currentMinDelay {
		t.slowdownFactor = float64(minDelay) / float64(currentMinDelay)
	} else {
		t.slowdownFactor = 2.0
	}
}

// DisableSlowdownMode - Tắt slowdown mode
func (t *UploadThrottler) DisableSlowdownMode() {
	t.slowdownMu.Lock()
	defer t.slowdownMu.Unlock()

	t.slowdownActive = false
	t.slowdownFactor = 1.0
}

// Stop - Dừng throttler (cleanup)
func (t *UploadThrottler) Stop() {
	// Close semaphore to prevent new acquires
	// Note: This is a graceful stop - existing operations will complete
	t.slowdownMu.Lock()
	t.slowdownActive = false
	t.slowdownFactor = 1.0
	t.slowdownMu.Unlock()

	// Clear upload times
	t.uploadMu.Lock()
	t.uploadTimes = make([]time.Time, 0)
	t.uploadMu.Unlock()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

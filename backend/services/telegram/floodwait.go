package telegram

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// FloodWaitConfig - Cấu hình xử lý FloodWait
type FloodWaitConfig struct {
	// Số lần retry tối đa khi gặp FloodWait
	MaxRetries int
	// Thời gian chờ tối đa (nếu Telegram yêu cầu chờ quá lâu)
	MaxWaitDuration time.Duration
	// Callback khi gặp FloodWait
	OnFloodWait func(waitSeconds int, workerID string)
	// Callback khi resume sau FloodWait
	OnResume func(workerID string)
}

// DefaultFloodWaitConfig - Cấu hình mặc định
func DefaultFloodWaitConfig() FloodWaitConfig {
	return FloodWaitConfig{
		MaxRetries:      5,
		MaxWaitDuration: 30 * time.Minute,
	}
}

// FloodWaitHandler - Xử lý FloodWait từ Telegram
type FloodWaitHandler struct {
	config FloodWaitConfig

	// Track workers đang bị pause
	pausedWorkers map[string]time.Time
	mu            sync.RWMutex

	// Event log
	events    []FloodWaitEvent
	eventMu   sync.RWMutex
	maxEvents int

	// Stats (internal, protected by stats.mu)
	stats floodWaitStatsInternal
}

// FloodWaitEvent - Sự kiện FloodWait
type FloodWaitEvent struct {
	Timestamp    time.Time
	WorkerID     string
	WaitSeconds  int
	WaitDuration time.Duration
	Operation    string
	Error        string
	Resumed      bool
}

// FloodWaitStats - Thống kê FloodWait (DTO - không chứa mutex, an toàn khi copy/return)
type FloodWaitStats struct {
	TotalFloodWaits   int64
	TotalWaitTime     time.Duration
	LongestWait       time.Duration
	LastFloodWait     time.Time
	FloodWaitsLast1h  int
	FloodWaitsLast24h int
}

// floodWaitStatsInternal - internal struct với mutex, chỉ dùng trong FloodWaitHandler
type floodWaitStatsInternal struct {
	FloodWaitStats
	mu sync.RWMutex
}

// NewFloodWaitHandler - Tạo handler mới
func NewFloodWaitHandler(config FloodWaitConfig) *FloodWaitHandler {
	return &FloodWaitHandler{
		config:        config,
		pausedWorkers: make(map[string]time.Time),
		events:        make([]FloodWaitEvent, 0, 100),
		maxEvents:     1000,
	}
}

// HandleErrorWithWorkerOld - Xử lý error từ Telegram, phát hiện FloodWait (deprecated)
// Trả về (shouldRetry, waitDuration, error)
func (h *FloodWaitHandler) HandleErrorWithWorkerOld(err error, workerID string) (bool, time.Duration, error) {
	if err == nil {
		return false, 0, nil
	}

	// Parse FloodWait từ error message
	waitSeconds := h.parseFloodWait(err)
	if waitSeconds == 0 {
		// Không phải FloodWait error
		return false, 0, err
	}

	waitDuration := time.Duration(waitSeconds) * time.Second

	// Log event
	h.recordEvent(FloodWaitEvent{
		Timestamp:    time.Now(),
		WorkerID:     workerID,
		WaitSeconds:  waitSeconds,
		WaitDuration: waitDuration,
		Error:        err.Error(),
	})

	// Update stats
	h.updateStats(waitDuration)

	// Check max wait duration
	if waitDuration > h.config.MaxWaitDuration {
		return false, 0, fmt.Errorf("flood wait too long (%v > %v): %w", waitDuration, h.config.MaxWaitDuration, ErrFloodWait)
	}

	// Mark worker as paused
	h.pauseWorker(workerID, waitDuration)

	// Callback
	if h.config.OnFloodWait != nil {
		h.config.OnFloodWait(waitSeconds, workerID)
	}

	return true, waitDuration, ErrFloodWait
}

// parseFloodWait - Parse thời gian chờ từ error
func (h *FloodWaitHandler) parseFloodWait(err error) int {
	if err == nil {
		return 0
	}

	errStr := err.Error()

	// Pattern 1: FLOOD_WAIT_X hoặc FloodWait_X
	patterns := []string{
		`FLOOD_WAIT[_\s]*(\d+)`,
		`FloodWait[_\s]*(\d+)`,
		`flood_wait[_\s]*(\d+)`,
		`Too Many Requests: retry after (\d+)`,
		`retry_after[:\s]*(\d+)`,
		`retry after (\d+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(errStr)
		if len(matches) >= 2 {
			seconds, _ := strconv.Atoi(matches[1])
			return seconds
		}
	}

	// Pattern 2: HTTP 429
	if strings.Contains(errStr, "429") {
		// Default wait 60 seconds for HTTP 429 without specific time
		return 60
	}

	return 0
}

// WaitForFloodWait - Đợi FloodWait với context support
func (h *FloodWaitHandler) WaitForFloodWait(ctx context.Context, workerID string, duration time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
		h.resumeWorker(workerID)
		return nil
	}
}

// pauseWorker - Đánh dấu worker đang pause
func (h *FloodWaitHandler) pauseWorker(workerID string, duration time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pausedWorkers[workerID] = time.Now().Add(duration)
}

// resumeWorker - Resume worker sau FloodWait
func (h *FloodWaitHandler) resumeWorker(workerID string) {
	h.mu.Lock()
	delete(h.pausedWorkers, workerID)
	h.mu.Unlock()

	// Mark event as resumed
	h.eventMu.Lock()
	for i := len(h.events) - 1; i >= 0; i-- {
		if h.events[i].WorkerID == workerID && !h.events[i].Resumed {
			h.events[i].Resumed = true
			break
		}
	}
	h.eventMu.Unlock()

	// Callback
	if h.config.OnResume != nil {
		h.config.OnResume(workerID)
	}
}

// IsWorkerPaused - Kiểm tra worker có đang pause không
func (h *FloodWaitHandler) IsWorkerPaused(workerID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	resumeTime, exists := h.pausedWorkers[workerID]
	if !exists {
		return false
	}

	return time.Now().Before(resumeTime)
}

// GetPausedWorkers - Lấy danh sách workers đang pause
func (h *FloodWaitHandler) GetPausedWorkers() map[string]time.Time {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make(map[string]time.Time)
	now := time.Now()
	for id, resumeTime := range h.pausedWorkers {
		if resumeTime.After(now) {
			result[id] = resumeTime
		}
	}
	return result
}

// recordEvent - Ghi nhận event
func (h *FloodWaitHandler) recordEvent(event FloodWaitEvent) {
	h.eventMu.Lock()
	defer h.eventMu.Unlock()

	h.events = append(h.events, event)

	// Trim if too many events
	if len(h.events) > h.maxEvents {
		h.events = h.events[len(h.events)-h.maxEvents:]
	}
}

// updateStats - Cập nhật thống kê
func (h *FloodWaitHandler) updateStats(waitDuration time.Duration) {
	h.stats.mu.Lock()
	defer h.stats.mu.Unlock()

	h.stats.TotalFloodWaits++
	h.stats.TotalWaitTime += waitDuration
	h.stats.LastFloodWait = time.Now()

	if waitDuration > h.stats.LongestWait {
		h.stats.LongestWait = waitDuration
	}
}

// GetStats - Lấy thống kê (trả về DTO không chứa mutex, an toàn)
func (h *FloodWaitHandler) GetStats() FloodWaitStats {
	h.stats.mu.RLock()
	stats := h.stats.FloodWaitStats // copy data only, no mutex
	h.stats.mu.RUnlock()

	// Calculate flood waits in time windows
	h.eventMu.RLock()
	now := time.Now()
	oneHourAgo := now.Add(-time.Hour)
	oneDayAgo := now.Add(-24 * time.Hour)

	for _, event := range h.events {
		if event.Timestamp.After(oneHourAgo) {
			stats.FloodWaitsLast1h++
		}
		if event.Timestamp.After(oneDayAgo) {
			stats.FloodWaitsLast24h++
		}
	}
	h.eventMu.RUnlock()

	return stats
}

// GetRecentEvents - Lấy events gần đây
func (h *FloodWaitHandler) GetRecentEvents(limit int) []FloodWaitEvent {
	h.eventMu.RLock()
	defer h.eventMu.RUnlock()

	if limit <= 0 || limit > len(h.events) {
		limit = len(h.events)
	}

	result := make([]FloodWaitEvent, limit)
	copy(result, h.events[len(h.events)-limit:])
	return result
}

// CountFloodWaitsInWindow - Đếm số FloodWait trong khoảng thời gian
func (h *FloodWaitHandler) CountFloodWaitsInWindow(duration time.Duration) int {
	h.eventMu.RLock()
	defer h.eventMu.RUnlock()

	threshold := time.Now().Add(-duration)
	count := 0
	for _, event := range h.events {
		if event.Timestamp.After(threshold) {
			count++
		}
	}
	return count
}

// Cleanup - Dọn dẹp expired data
func (h *FloodWaitHandler) Cleanup() {
	// Cleanup paused workers
	h.mu.Lock()
	now := time.Now()
	for id, resumeTime := range h.pausedWorkers {
		if resumeTime.Before(now) {
			delete(h.pausedWorkers, id)
		}
	}
	h.mu.Unlock()

	// Cleanup old events (older than 24h)
	h.eventMu.Lock()
	threshold := time.Now().Add(-24 * time.Hour)
	filtered := make([]FloodWaitEvent, 0, len(h.events))
	for _, event := range h.events {
		if event.Timestamp.After(threshold) {
			filtered = append(filtered, event)
		}
	}
	h.events = filtered
	h.eventMu.Unlock()
}

// floodCallback được gọi khi gặp FloodWait
var floodCallback func(duration time.Duration, err error)
var floodCallbackMu sync.RWMutex

// SetFloodCallback - Đặt callback khi gặp FloodWait
func (h *FloodWaitHandler) SetFloodCallback(fn func(duration time.Duration, err error)) {
	floodCallbackMu.Lock()
	defer floodCallbackMu.Unlock()
	floodCallback = fn
}

// Stats - Alias cho GetStats
func (h *FloodWaitHandler) Stats() FloodWaitStats {
	return h.GetStats()
}

// HandleError - Context-aware error handler cho protection layer
// Trả về nil nếu đã xử lý FloodWait thành công, hoặc error gốc nếu không phải FloodWait
func (h *FloodWaitHandler) HandleError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	// Parse FloodWait từ error
	waitSeconds := h.parseFloodWait(err)
	if waitSeconds == 0 {
		// Không phải FloodWait, trả về error gốc
		return err
	}

	waitDuration := time.Duration(waitSeconds) * time.Second

	// Log event
	h.recordEvent(FloodWaitEvent{
		Timestamp:    time.Now(),
		WorkerID:     "protection",
		WaitSeconds:  waitSeconds,
		WaitDuration: waitDuration,
		Operation:    "api_call",
		Error:        err.Error(),
	})

	// Update stats
	h.updateStats(waitDuration)

	// Call global flood callback
	floodCallbackMu.RLock()
	cb := floodCallback
	floodCallbackMu.RUnlock()
	if cb != nil {
		cb(waitDuration, err)
	}

	// Check max wait duration
	if waitDuration > h.config.MaxWaitDuration {
		return fmt.Errorf("flood wait too long (%v > %v): %w", waitDuration, h.config.MaxWaitDuration, ErrFloodWait)
	}

	// Wait for FloodWait duration
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitDuration):
		return nil // Successfully waited, caller should retry
	}
}

// HandleErrorWithWorker - Xử lý error với worker ID tracking
// Trả về (shouldRetry, waitDuration, error)
func (h *FloodWaitHandler) HandleErrorWithWorker(err error, workerID string) (bool, time.Duration, error) {
	if err == nil {
		return false, 0, nil
	}

	// Parse FloodWait từ error message
	waitSeconds := h.parseFloodWait(err)
	if waitSeconds == 0 {
		// Không phải FloodWait error
		return false, 0, err
	}

	waitDuration := time.Duration(waitSeconds) * time.Second

	// Log event
	h.recordEvent(FloodWaitEvent{
		Timestamp:    time.Now(),
		WorkerID:     workerID,
		WaitSeconds:  waitSeconds,
		WaitDuration: waitDuration,
		Error:        err.Error(),
	})

	// Update stats
	h.updateStats(waitDuration)

	// Check max wait duration
	if waitDuration > h.config.MaxWaitDuration {
		return false, 0, fmt.Errorf("flood wait too long (%v > %v): %w", waitDuration, h.config.MaxWaitDuration, ErrFloodWait)
	}

	// Mark worker as paused
	h.pauseWorker(workerID, waitDuration)

	// Callback
	if h.config.OnFloodWait != nil {
		h.config.OnFloodWait(waitSeconds, workerID)
	}

	return true, waitDuration, ErrFloodWait
}

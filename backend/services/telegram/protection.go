package telegram

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// ProtectionConfig holds all protection layer configuration
type ProtectionConfig struct {
	RateLimiter RateLimitConfig
	FloodWait   FloodWaitConfig
	Throttler   ThrottlerConfig
	Backoff     BackoffConfig
	SafeMode    SafeModeConfig

	// Global settings
	EnableLogging        bool
	LogPrefix            string
	MetricsEnabled       bool
	MetricsFlushInterval time.Duration
}

// DefaultProtectionConfig returns sensible defaults for all protection layers
func DefaultProtectionConfig() ProtectionConfig {
	return ProtectionConfig{
		RateLimiter:          DefaultRateLimitConfig(),
		FloodWait:            DefaultFloodWaitConfig(),
		Throttler:            DefaultThrottlerConfig(),
		Backoff:              DefaultBackoffConfig(),
		SafeMode:             DefaultSafeModeConfig(),
		EnableLogging:        true,
		LogPrefix:            "[TelegramProtection]",
		MetricsEnabled:       true,
		MetricsFlushInterval: 1 * time.Minute,
	}
}

// ProtectionLayer wraps all anti-ban and rate-limiting components
type ProtectionLayer struct {
	mu     sync.RWMutex
	config ProtectionConfig

	// Components
	rateLimiter *RateLimiter
	floodWait   *FloodWaitHandler
	throttler   *UploadThrottler
	backoff     *ExponentialBackoff
	safeMode    *SafeModeController

	// Metrics tracking
	metrics       *ProtectionMetrics
	metricsCancel context.CancelFunc
	metricsWg     sync.WaitGroup

	// State
	isRunning bool
}

// ProtectionMetrics tracks usage statistics
type ProtectionMetrics struct {
	mu sync.RWMutex

	// Counters
	TotalRequests        int64
	BlockedByRateLimiter int64
	BlockedByThrottler   int64
	BlockedBySafeMode    int64
	FloodWaitEncountered int64
	RetriesPerformed     int64
	SuccessfulOperations int64
	FailedOperations     int64

	// Timing
	AverageWaitTime time.Duration
	MaxWaitTime     time.Duration
	TotalWaitTime   time.Duration

	// Snapshots
	LastReset time.Time
	LastFlush time.Time
}

// NewProtectionLayer creates a new protection layer with all components
func NewProtectionLayer(config ProtectionConfig) *ProtectionLayer {
	pl := &ProtectionLayer{
		config:      config,
		rateLimiter: NewRateLimiter(config.RateLimiter),
		floodWait:   NewFloodWaitHandler(config.FloodWait),
		throttler:   NewUploadThrottler(config.Throttler),
		backoff:     NewExponentialBackoff(config.Backoff),
		safeMode:    NewSafeModeController(config.SafeMode),
		metrics: &ProtectionMetrics{
			LastReset: time.Now(),
		},
	}

	// Wire up FloodWait events to SafeMode
	pl.floodWait.SetFloodCallback(func(duration time.Duration, err error) {
		pl.safeMode.RecordFloodWait(duration, "api_call")
		pl.metrics.mu.Lock()
		pl.metrics.FloodWaitEncountered++
		pl.metrics.mu.Unlock()
	})

	// Wire up SafeMode level changes
	pl.safeMode.SetLevelChangeCallback(func(oldLevel, newLevel SafeModeLevel) {
		if config.EnableLogging {
			log.Printf("%s SafeMode level changed: %s -> %s",
				config.LogPrefix, oldLevel, newLevel)
		}

		// Adjust throttler based on safe mode level
		switch newLevel {
		case SafeModeElevated:
			pl.throttler.EnableSlowdownMode(10 * time.Second)
		case SafeModeHigh:
			pl.throttler.EnableSlowdownMode(20 * time.Second)
		case SafeModeCritical:
			pl.throttler.EnableSlowdownMode(30 * time.Second)
		case SafeModeNormal:
			pl.throttler.DisableSlowdownMode()
		}
	})

	// Start metrics flushing if enabled
	if config.MetricsEnabled {
		ctx, cancel := context.WithCancel(context.Background())
		pl.metricsCancel = cancel
		pl.startMetricsFlushing(ctx)
	}

	pl.isRunning = true
	return pl
}

// startMetricsFlushing periodically logs metrics
func (pl *ProtectionLayer) startMetricsFlushing(ctx context.Context) {
	pl.metricsWg.Add(1)
	go func() {
		defer pl.metricsWg.Done()

		ticker := time.NewTicker(pl.config.MetricsFlushInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pl.flushMetrics()
			}
		}
	}()
}

// flushMetrics logs current metrics
func (pl *ProtectionLayer) flushMetrics() {
	if !pl.config.EnableLogging {
		return
	}

	stats := pl.GetStats()
	log.Printf("%s Metrics: requests=%d, blocked(rate=%d, throttle=%d, safe=%d), "+
		"floods=%d, retries=%d, success=%d, failed=%d",
		pl.config.LogPrefix,
		stats.Metrics.TotalRequests,
		stats.Metrics.BlockedByRateLimiter,
		stats.Metrics.BlockedByThrottler,
		stats.Metrics.BlockedBySafeMode,
		stats.Metrics.FloodWaitEncountered,
		stats.Metrics.RetriesPerformed,
		stats.Metrics.SuccessfulOperations,
		stats.Metrics.FailedOperations,
	)

	pl.metrics.mu.Lock()
	pl.metrics.LastFlush = time.Now()
	pl.metrics.mu.Unlock()
}

// ExecuteWithProtection wraps an operation with full protection
func (pl *ProtectionLayer) ExecuteWithProtection(
	ctx context.Context,
	userID string,
	operationType string,
	fn func() error,
) error {
	pl.metrics.mu.Lock()
	pl.metrics.TotalRequests++
	pl.metrics.mu.Unlock()

	startTime := time.Now()

	// 1. Check SafeMode - wait if in emergency
	if err := pl.safeMode.WaitIfNeeded(ctx); err != nil {
		return fmt.Errorf("safe mode wait cancelled: %w", err)
	}

	if !pl.safeMode.IsOperationAllowed() {
		pl.metrics.mu.Lock()
		pl.metrics.BlockedBySafeMode++
		pl.metrics.mu.Unlock()
		return fmt.Errorf("operation blocked: system in emergency safe mode")
	}

	// 2. Check RateLimiter
	if operationType == "upload" {
		allowed, waitTime := pl.rateLimiter.AllowUpload(userID)
		if !allowed {
			pl.metrics.mu.Lock()
			pl.metrics.BlockedByRateLimiter++
			pl.metrics.mu.Unlock()

			if waitTime > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(waitTime):
					// Retry after waiting
				}
			} else {
				return fmt.Errorf("rate limit exceeded for user %s", userID)
			}
		}
	} else {
		if !pl.rateLimiter.AllowRequest(userID) {
			pl.metrics.mu.Lock()
			pl.metrics.BlockedByRateLimiter++
			pl.metrics.mu.Unlock()
			return fmt.Errorf("rate limit exceeded for user %s", userID)
		}
	}

	// 3. Acquire throttler slot for uploads
	if operationType == "upload" {
		if err := pl.throttler.Acquire(ctx); err != nil {
			pl.metrics.mu.Lock()
			pl.metrics.BlockedByThrottler++
			pl.metrics.mu.Unlock()
			return fmt.Errorf("throttler acquisition failed: %w", err)
		}
		defer pl.throttler.Release()
	}

	// 4. Apply delay multiplier from SafeMode
	multiplier := pl.safeMode.GetDelayMultiplier()
	if multiplier > 1.0 {
		baseDelay := time.Duration(float64(2*time.Second) * multiplier)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(baseDelay):
		}
	}

	// 5. Execute with backoff and FloodWait handling
	operationID := fmt.Sprintf("%s:%s:%d", userID, operationType, time.Now().UnixNano())

	err := pl.backoff.ExecuteWithCallback(
		ctx,
		operationID,
		func() error {
			// Execute the actual operation
			opErr := fn()

			// Handle FloodWait errors specially
			if opErr != nil {
				return pl.floodWait.HandleError(ctx, opErr)
			}
			return nil
		},
		func(attempt int, delay time.Duration, err error) {
			pl.metrics.mu.Lock()
			pl.metrics.RetriesPerformed++
			pl.metrics.mu.Unlock()

			if pl.config.EnableLogging {
				log.Printf("%s Retry attempt %d for %s, waiting %s: %v",
					pl.config.LogPrefix, attempt, operationType, delay, err)
			}
		},
	)

	// Record metrics
	elapsed := time.Since(startTime)
	pl.metrics.mu.Lock()
	if err != nil {
		pl.metrics.FailedOperations++
	} else {
		pl.metrics.SuccessfulOperations++
	}
	pl.metrics.TotalWaitTime += elapsed
	if elapsed > pl.metrics.MaxWaitTime {
		pl.metrics.MaxWaitTime = elapsed
	}
	pl.metrics.mu.Unlock()

	return err
}

// ExecuteUpload wraps upload operations with full protection
func (pl *ProtectionLayer) ExecuteUpload(
	ctx context.Context,
	userID string,
	fn func() error,
) error {
	return pl.ExecuteWithProtection(ctx, userID, "upload", fn)
}

// ExecuteRequest wraps general API requests with protection
func (pl *ProtectionLayer) ExecuteRequest(
	ctx context.Context,
	userID string,
	fn func() error,
) error {
	return pl.ExecuteWithProtection(ctx, userID, "request", fn)
}

// ProtectionStats holds comprehensive statistics
type ProtectionStats struct {
	Metrics     ProtectionMetricsSnapshot `json:"metrics"`
	RateLimiter RateLimitMetrics          `json:"rate_limiter"`
	FloodWait   FloodWaitStats            `json:"flood_wait"`
	Throttler   ThrottlerStats            `json:"throttler"`
	SafeMode    SafeModeStats             `json:"safe_mode"`
	Backoff     map[string]BackoffStats   `json:"backoff"`
}

// ProtectionMetricsSnapshot is a point-in-time snapshot of metrics
type ProtectionMetricsSnapshot struct {
	TotalRequests        int64         `json:"total_requests"`
	BlockedByRateLimiter int64         `json:"blocked_by_rate_limiter"`
	BlockedByThrottler   int64         `json:"blocked_by_throttler"`
	BlockedBySafeMode    int64         `json:"blocked_by_safe_mode"`
	FloodWaitEncountered int64         `json:"flood_wait_encountered"`
	RetriesPerformed     int64         `json:"retries_performed"`
	SuccessfulOperations int64         `json:"successful_operations"`
	FailedOperations     int64         `json:"failed_operations"`
	AverageWaitTime      time.Duration `json:"average_wait_time"`
	MaxWaitTime          time.Duration `json:"max_wait_time"`
	Uptime               time.Duration `json:"uptime"`
}

// GetStats returns comprehensive protection statistics
func (pl *ProtectionLayer) GetStats() ProtectionStats {
	pl.metrics.mu.RLock()
	m := pl.metrics

	var avgWait time.Duration
	if m.TotalRequests > 0 {
		avgWait = m.TotalWaitTime / time.Duration(m.TotalRequests)
	}

	metricsSnapshot := ProtectionMetricsSnapshot{
		TotalRequests:        m.TotalRequests,
		BlockedByRateLimiter: m.BlockedByRateLimiter,
		BlockedByThrottler:   m.BlockedByThrottler,
		BlockedBySafeMode:    m.BlockedBySafeMode,
		FloodWaitEncountered: m.FloodWaitEncountered,
		RetriesPerformed:     m.RetriesPerformed,
		SuccessfulOperations: m.SuccessfulOperations,
		FailedOperations:     m.FailedOperations,
		AverageWaitTime:      avgWait,
		MaxWaitTime:          m.MaxWaitTime,
		Uptime:               time.Since(m.LastReset),
	}
	pl.metrics.mu.RUnlock()

	return ProtectionStats{
		Metrics:     metricsSnapshot,
		RateLimiter: pl.rateLimiter.Stats(),
		FloodWait:   pl.floodWait.Stats(),
		Throttler:   pl.throttler.Stats(),
		SafeMode:    pl.safeMode.Stats(false),
		Backoff:     pl.backoff.AllStats(),
	}
}

// ResetMetrics resets all metrics counters
func (pl *ProtectionLayer) ResetMetrics() {
	pl.metrics.mu.Lock()
	defer pl.metrics.mu.Unlock()

	pl.metrics.TotalRequests = 0
	pl.metrics.BlockedByRateLimiter = 0
	pl.metrics.BlockedByThrottler = 0
	pl.metrics.BlockedBySafeMode = 0
	pl.metrics.FloodWaitEncountered = 0
	pl.metrics.RetriesPerformed = 0
	pl.metrics.SuccessfulOperations = 0
	pl.metrics.FailedOperations = 0
	pl.metrics.AverageWaitTime = 0
	pl.metrics.MaxWaitTime = 0
	pl.metrics.TotalWaitTime = 0
	pl.metrics.LastReset = time.Now()
}

// SetSafeModeLevel manually sets the safe mode level
func (pl *ProtectionLayer) SetSafeModeLevel(level SafeModeLevel) {
	pl.safeMode.ForceLevel(level)
}

// ResetSafeMode resets safe mode to normal
func (pl *ProtectionLayer) ResetSafeMode() {
	pl.safeMode.Reset()
}

// Stop gracefully stops all protection layer components
func (pl *ProtectionLayer) Stop() {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	if !pl.isRunning {
		return
	}

	// Stop metrics flushing
	if pl.metricsCancel != nil {
		pl.metricsCancel()
		pl.metricsWg.Wait()
	}

	// Stop safe mode controller
	pl.safeMode.Stop()

	// Stop throttler
	pl.throttler.Stop()

	pl.isRunning = false

	if pl.config.EnableLogging {
		log.Printf("%s Protection layer stopped", pl.config.LogPrefix)
	}
}

// RateLimiter returns the rate limiter component
func (pl *ProtectionLayer) RateLimiter() *RateLimiter {
	return pl.rateLimiter
}

// FloodWaitHandler returns the flood wait handler component
func (pl *ProtectionLayer) FloodWaitHandler() *FloodWaitHandler {
	return pl.floodWait
}

// Throttler returns the upload throttler component
func (pl *ProtectionLayer) Throttler() *UploadThrottler {
	return pl.throttler
}

// Backoff returns the exponential backoff component
func (pl *ProtectionLayer) Backoff() *ExponentialBackoff {
	return pl.backoff
}

// SafeMode returns the safe mode controller component
func (pl *ProtectionLayer) SafeMode() *SafeModeController {
	return pl.safeMode
}

// HealthCheck performs a health check on all components
func (pl *ProtectionLayer) HealthCheck() map[string]bool {
	return map[string]bool{
		"rate_limiter": pl.rateLimiter != nil,
		"flood_wait":   pl.floodWait != nil,
		"throttler":    pl.throttler != nil,
		"backoff":      pl.backoff != nil,
		"safe_mode":    pl.safeMode != nil,
		"is_running":   pl.isRunning,
	}
}

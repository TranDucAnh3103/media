package telegram

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

// BackoffConfig holds exponential backoff configuration
type BackoffConfig struct {
	// BaseDelay is the initial delay before first retry
	BaseDelay time.Duration

	// MaxDelay is the maximum delay cap
	MaxDelay time.Duration

	// MaxRetries is the maximum number of retry attempts (0 = unlimited)
	MaxRetries int

	// Multiplier is the exponential factor (typically 2.0)
	Multiplier float64

	// JitterFactor adds randomness to prevent thundering herd (0.0 - 1.0)
	JitterFactor float64

	// ResetAfterSuccess resets retry count after successful operation
	ResetAfterSuccess bool
}

// DefaultBackoffConfig returns sensible defaults for Telegram API
func DefaultBackoffConfig() BackoffConfig {
	return BackoffConfig{
		BaseDelay:         1 * time.Second,
		MaxDelay:          5 * time.Minute,
		MaxRetries:        10,
		Multiplier:        2.0,
		JitterFactor:      0.3,
		ResetAfterSuccess: true,
	}
}

// BackoffState tracks the current backoff state for an operation
type BackoffState struct {
	mu           sync.RWMutex
	config       BackoffConfig
	attempts     int
	lastAttempt  time.Time
	lastSuccess  time.Time
	lastFailure  time.Time
	totalRetries int
	isInBackoff  bool
}

// NewBackoffState creates a new backoff state with given config
func NewBackoffState(config BackoffConfig) *BackoffState {
	return &BackoffState{
		config: config,
	}
}

// NextDelay calculates the next delay based on attempt number
// Formula: delay = min(base_delay * multiplier^attempt + jitter, max_delay)
func (b *BackoffState) NextDelay() time.Duration {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.calculateDelay(b.attempts)
}

// calculateDelay computes delay for given attempt number
func (b *BackoffState) calculateDelay(attempt int) time.Duration {
	if attempt == 0 {
		return 0
	}

	// Exponential calculation: base * multiplier^attempt
	delayFloat := float64(b.config.BaseDelay) * math.Pow(b.config.Multiplier, float64(attempt-1))

	// Apply jitter to prevent thundering herd
	if b.config.JitterFactor > 0 {
		jitter := delayFloat * b.config.JitterFactor * (rand.Float64()*2 - 1) // [-jitter, +jitter]
		delayFloat += jitter
	}

	delay := time.Duration(delayFloat)

	// Cap at max delay
	if delay > b.config.MaxDelay {
		delay = b.config.MaxDelay
	}

	// Ensure minimum delay
	if delay < 0 {
		delay = b.config.BaseDelay
	}

	return delay
}

// RecordAttempt increments the attempt counter
func (b *BackoffState) RecordAttempt() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.attempts++
	b.totalRetries++
	b.lastAttempt = time.Now()
	b.isInBackoff = true
}

// RecordSuccess records a successful operation
func (b *BackoffState) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.lastSuccess = time.Now()
	b.isInBackoff = false

	if b.config.ResetAfterSuccess {
		b.attempts = 0
	}
}

// RecordFailure records a failed operation
func (b *BackoffState) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.lastFailure = time.Now()
}

// Reset resets the backoff state
func (b *BackoffState) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.attempts = 0
	b.isInBackoff = false
}

// ShouldRetry returns whether another retry should be attempted
func (b *BackoffState) ShouldRetry() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.config.MaxRetries == 0 {
		return true // Unlimited retries
	}
	return b.attempts < b.config.MaxRetries
}

// Attempts returns current attempt count
func (b *BackoffState) Attempts() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.attempts
}

// IsInBackoff returns whether currently in backoff state
func (b *BackoffState) IsInBackoff() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.isInBackoff
}

// BackoffStats returns statistics about backoff state
type BackoffStats struct {
	CurrentAttempts int           `json:"current_attempts"`
	TotalRetries    int           `json:"total_retries"`
	IsInBackoff     bool          `json:"is_in_backoff"`
	LastAttempt     time.Time     `json:"last_attempt"`
	LastSuccess     time.Time     `json:"last_success"`
	LastFailure     time.Time     `json:"last_failure"`
	NextDelay       time.Duration `json:"next_delay"`
}

// Stats returns current backoff statistics
func (b *BackoffState) Stats() BackoffStats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return BackoffStats{
		CurrentAttempts: b.attempts,
		TotalRetries:    b.totalRetries,
		IsInBackoff:     b.isInBackoff,
		LastAttempt:     b.lastAttempt,
		LastSuccess:     b.lastSuccess,
		LastFailure:     b.lastFailure,
		NextDelay:       b.calculateDelay(b.attempts),
	}
}

// ExponentialBackoff provides a reusable exponential backoff mechanism
type ExponentialBackoff struct {
	mu          sync.RWMutex
	config      BackoffConfig
	states      map[string]*BackoffState // Per-operation backoff states
	globalState *BackoffState            // Global backoff state
}

// NewExponentialBackoff creates a new exponential backoff manager
func NewExponentialBackoff(config BackoffConfig) *ExponentialBackoff {
	return &ExponentialBackoff{
		config:      config,
		states:      make(map[string]*BackoffState),
		globalState: NewBackoffState(config),
	}
}

// GetState gets or creates a backoff state for an operation
func (e *ExponentialBackoff) GetState(operationID string) *BackoffState {
	e.mu.Lock()
	defer e.mu.Unlock()

	if state, exists := e.states[operationID]; exists {
		return state
	}

	state := NewBackoffState(e.config)
	e.states[operationID] = state
	return state
}

// RemoveState removes a backoff state for an operation
func (e *ExponentialBackoff) RemoveState(operationID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.states, operationID)
}

// GlobalState returns the global backoff state
func (e *ExponentialBackoff) GlobalState() *BackoffState {
	return e.globalState
}

// Wait waits for the backoff delay with context support
func (e *ExponentialBackoff) Wait(ctx context.Context, state *BackoffState) error {
	delay := state.NextDelay()
	if delay == 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Execute executes an operation with exponential backoff retry
func (e *ExponentialBackoff) Execute(ctx context.Context, operationID string, fn func() error) error {
	state := e.GetState(operationID)
	defer e.RemoveState(operationID)

	for {
		// Wait for backoff delay
		if err := e.Wait(ctx, state); err != nil {
			return err
		}

		// Record attempt
		state.RecordAttempt()

		// Execute the operation
		err := fn()
		if err == nil {
			state.RecordSuccess()
			return nil
		}

		// Record failure
		state.RecordFailure()

		// Check if we should retry
		if !state.ShouldRetry() {
			return &ErrMaxRetriesExceeded{
				Attempts: state.Attempts(),
				LastErr:  err,
			}
		}

		// Check if error is retryable
		if !isRetryableError(err) {
			return err
		}
	}
}

// ExecuteWithCallback executes with backoff and progress callback
func (e *ExponentialBackoff) ExecuteWithCallback(
	ctx context.Context,
	operationID string,
	fn func() error,
	onRetry func(attempt int, delay time.Duration, err error),
) error {
	state := e.GetState(operationID)
	defer e.RemoveState(operationID)

	for {
		// Wait for backoff delay (except first attempt)
		delay := state.NextDelay()
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}

		// Record attempt
		state.RecordAttempt()

		// Execute the operation
		err := fn()
		if err == nil {
			state.RecordSuccess()
			return nil
		}

		// Record failure
		state.RecordFailure()

		// Call retry callback
		if onRetry != nil {
			nextDelay := state.NextDelay()
			onRetry(state.Attempts(), nextDelay, err)
		}

		// Check if we should retry
		if !state.ShouldRetry() {
			return &ErrMaxRetriesExceeded{
				Attempts: state.Attempts(),
				LastErr:  err,
			}
		}

		// Check if error is retryable
		if !isRetryableError(err) {
			return err
		}
	}
}

// ErrMaxRetriesExceeded is returned when max retries are exhausted
type ErrMaxRetriesExceeded struct {
	Attempts int
	LastErr  error
}

func (e *ErrMaxRetriesExceeded) Error() string {
	return "max retries exceeded after " + string(rune(e.Attempts+'0')) + " attempts: " + e.LastErr.Error()
}

func (e *ErrMaxRetriesExceeded) Unwrap() error {
	return e.LastErr
}

// isRetryableError determines if an error is worth retrying
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// FloodWait errors should be handled by FloodWaitHandler, not backoff
	if errors.Is(err, ErrFloodWait) {
		return false // Let FloodWaitHandler deal with this
	}

	// Network timeouts are retryable
	if err.Error() == "context deadline exceeded" {
		return true
	}

	// Connection errors are generally retryable
	errStr := err.Error()
	retryablePatterns := []string{
		"connection reset",
		"connection refused",
		"no such host",
		"network is unreachable",
		"timeout",
		"temporary failure",
		"server is busy",
		"internal error",
		"bad gateway",
		"service unavailable",
	}

	for _, pattern := range retryablePatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// contains checks if s contains substr (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) && containsLower(toLower(s), toLower(substr)))
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

// AllStats returns statistics for all tracked operations
func (e *ExponentialBackoff) AllStats() map[string]BackoffStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := make(map[string]BackoffStats)
	for id, state := range e.states {
		stats[id] = state.Stats()
	}
	stats["_global"] = e.globalState.Stats()
	return stats
}

// CleanupStaleStates removes states that haven't been used recently
func (e *ExponentialBackoff) CleanupStaleStates(maxAge time.Duration) int {
	e.mu.Lock()
	defer e.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for id, state := range e.states {
		stats := state.Stats()
		if stats.LastAttempt.Before(cutoff) && !stats.IsInBackoff {
			delete(e.states, id)
			removed++
		}
	}

	return removed
}

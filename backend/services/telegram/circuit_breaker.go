package telegram

import (
	"log"
	"sync"
	"time"
)

type CircuitState int

const (
	StateClosed CircuitState = iota // Normal operation
	StateOpen                       // Failing, blocked
	StateHalfOpen                   // Testing recovery
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF-OPEN"
	default:
		return "UNKNOWN"
	}
}

type CircuitBreaker struct {
	mu           sync.RWMutex
	state        CircuitState
	failures     int
	maxFailures  int
	resetTimeout time.Duration
	lastFailure  time.Time
}

func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:        StateClosed,
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
	}
}

// AllowRequest checks if a request is allowed to proceed
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		if now.Sub(cb.lastFailure) >= cb.resetTimeout {
			log.Printf("[CircuitBreaker] Timeout reached. Transitioning from OPEN to HALF-OPEN state")
			cb.state = StateHalfOpen
			return true // Allow exactly one test request
		}
		return false

	case StateHalfOpen:
		// In HalfOpen, we already allowed the single test request. Subsequent ones are blocked
		// until the test request either calls RecordSuccess or RecordFailure.
		return false
	}

	return true
}

// RecordSuccess should be called when an operation succeeds
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateHalfOpen {
		log.Printf("[CircuitBreaker] Test request succeeded. Transitioning from HALF-OPEN to CLOSED state")
		cb.state = StateClosed
		cb.failures = 0
	} else if cb.state == StateClosed {
		cb.failures = 0 // Reset failures on success in closed state
	}
}

// RecordFailure should be called when an operation fails
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailure = time.Now()

	switch cb.state {
	case StateClosed:
		cb.failures++
		if cb.failures >= cb.maxFailures {
			log.Printf("[CircuitBreaker] Max failures (%d) reached. Transitioning from CLOSED to OPEN state. Blocked for %v",
				cb.maxFailures, cb.resetTimeout)
			cb.state = StateOpen
		} else {
			log.Printf("[CircuitBreaker] Failure recorded: %d/%d", cb.failures, cb.maxFailures)
		}

	case StateHalfOpen:
		log.Printf("[CircuitBreaker] Test request failed. Transitioning from HALF-OPEN back to OPEN state. Blocked for %v",
			cb.resetTimeout)
		cb.state = StateOpen
	}
}

// State returns the current circuit breaker state
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// IsClosed is a convenience method to check if system is operating normally
func (cb *CircuitBreaker) IsClosed() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state == StateClosed
}

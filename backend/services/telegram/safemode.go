package telegram

import (
	"context"
	"log"
	"sync"
	"time"
)

// SafeModeLevel indicates the current protection level
type SafeModeLevel int

const (
	// SafeModeNormal - normal operation, no restrictions
	SafeModeNormal SafeModeLevel = iota
	// SafeModeElevated - increased delays between operations
	SafeModeElevated
	// SafeModeHigh - significant throttling, extended delays
	SafeModeHigh
	// SafeModeCritical - minimal operations, long waits
	SafeModeCritical
	// SafeModeEmergency - stop all operations temporarily
	SafeModeEmergency
)

func (l SafeModeLevel) String() string {
	switch l {
	case SafeModeNormal:
		return "NORMAL"
	case SafeModeElevated:
		return "ELEVATED"
	case SafeModeHigh:
		return "HIGH"
	case SafeModeCritical:
		return "CRITICAL"
	case SafeModeEmergency:
		return "EMERGENCY"
	default:
		return "UNKNOWN"
	}
}

// SafeModeConfig holds safe mode controller configuration
type SafeModeConfig struct {
	// FloodWait thresholds to escalate safe mode
	ElevatedThreshold  int // FloodWaits in window to trigger ELEVATED
	HighThreshold      int // FloodWaits in window to trigger HIGH
	CriticalThreshold  int // FloodWaits in window to trigger CRITICAL
	EmergencyThreshold int // FloodWaits in window to trigger EMERGENCY

	// Time window for counting FloodWait events
	EventWindow time.Duration

	// Cooldown periods for each level before de-escalation
	ElevatedCooldown  time.Duration
	HighCooldown      time.Duration
	CriticalCooldown  time.Duration
	EmergencyCooldown time.Duration

	// Delay multipliers for each level
	ElevatedDelayMultiplier float64
	HighDelayMultiplier     float64
	CriticalDelayMultiplier float64

	// Emergency mode settings
	EmergencyPauseDuration time.Duration

	// Auto-recovery settings
	AutoRecoveryEnabled   bool
	RecoveryCheckInterval time.Duration
}

// DefaultSafeModeConfig returns sensible defaults
func DefaultSafeModeConfig() SafeModeConfig {
	return SafeModeConfig{
		ElevatedThreshold:  2,
		HighThreshold:      5,
		CriticalThreshold:  10,
		EmergencyThreshold: 15,

		EventWindow: 10 * time.Minute,

		ElevatedCooldown:  5 * time.Minute,
		HighCooldown:      15 * time.Minute,
		CriticalCooldown:  30 * time.Minute,
		EmergencyCooldown: 60 * time.Minute,

		ElevatedDelayMultiplier: 1.5,
		HighDelayMultiplier:     3.0,
		CriticalDelayMultiplier: 5.0,

		EmergencyPauseDuration: 30 * time.Minute,

		AutoRecoveryEnabled:   true,
		RecoveryCheckInterval: 1 * time.Minute,
	}
}

// SafeModeController manages automatic throttling based on FloodWait events
type SafeModeController struct {
	mu     sync.RWMutex
	config SafeModeConfig

	// Current state
	level            SafeModeLevel
	lastEscalation   time.Time
	lastDeescalation time.Time
	emergencyUntil   time.Time

	// Event tracking (uses FloodWaitEvent from floodwait.go)
	events []FloodWaitEvent

	// Callbacks
	onLevelChange func(oldLevel, newLevel SafeModeLevel)
	onEmergency   func()

	// Recovery
	ctx        context.Context
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
}

// NewSafeModeController creates a new safe mode controller
func NewSafeModeController(config SafeModeConfig) *SafeModeController {
	ctx, cancel := context.WithCancel(context.Background())

	sm := &SafeModeController{
		config:     config,
		level:      SafeModeNormal,
		events:     make([]FloodWaitEvent, 0),
		ctx:        ctx,
		cancelFunc: cancel,
	}

	// Start auto-recovery goroutine if enabled
	if config.AutoRecoveryEnabled {
		sm.startAutoRecovery()
	}

	return sm
}

// startAutoRecovery starts the background recovery checker
func (sm *SafeModeController) startAutoRecovery() {
	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()

		ticker := time.NewTicker(sm.config.RecoveryCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-sm.ctx.Done():
				return
			case <-ticker.C:
				sm.checkRecovery()
			}
		}
	}()
}

// checkRecovery checks if we can de-escalate the safe mode level
func (sm *SafeModeController) checkRecovery() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.level == SafeModeNormal {
		return
	}

	// Clean old events
	sm.cleanOldEvents()

	// Count current events
	eventCount := len(sm.events)
	now := time.Now()

	// Check if we can de-escalate
	var targetLevel SafeModeLevel
	var cooldownMet bool

	switch sm.level {
	case SafeModeEmergency:
		if now.After(sm.emergencyUntil) {
			targetLevel = SafeModeCritical
			cooldownMet = true
		}
	case SafeModeCritical:
		if eventCount < sm.config.CriticalThreshold &&
			now.Sub(sm.lastEscalation) >= sm.config.CriticalCooldown {
			targetLevel = SafeModeHigh
			cooldownMet = true
		}
	case SafeModeHigh:
		if eventCount < sm.config.HighThreshold &&
			now.Sub(sm.lastEscalation) >= sm.config.HighCooldown {
			targetLevel = SafeModeElevated
			cooldownMet = true
		}
	case SafeModeElevated:
		if eventCount < sm.config.ElevatedThreshold &&
			now.Sub(sm.lastEscalation) >= sm.config.ElevatedCooldown {
			targetLevel = SafeModeNormal
			cooldownMet = true
		}
	}

	if cooldownMet {
		oldLevel := sm.level
		sm.level = targetLevel
		sm.lastDeescalation = now

		log.Printf("[SafeMode] De-escalated from %s to %s (events: %d)",
			oldLevel, targetLevel, eventCount)

		if sm.onLevelChange != nil {
			go sm.onLevelChange(oldLevel, targetLevel)
		}
	}
}

// cleanOldEvents removes events outside the event window
func (sm *SafeModeController) cleanOldEvents() {
	cutoff := time.Now().Add(-sm.config.EventWindow)

	newEvents := make([]FloodWaitEvent, 0, len(sm.events))
	for _, event := range sm.events {
		if event.Timestamp.After(cutoff) {
			newEvents = append(newEvents, event)
		}
	}
	sm.events = newEvents
}

// RecordFloodWait records a FloodWait event and adjusts safe mode level
func (sm *SafeModeController) RecordFloodWait(waitDuration time.Duration, operation string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Record the event
	event := FloodWaitEvent{
		Timestamp:    time.Now(),
		WaitDuration: waitDuration,
		Operation:    operation,
	}
	sm.events = append(sm.events, event)

	// Clean old events
	sm.cleanOldEvents()

	// Check for escalation
	eventCount := len(sm.events)
	oldLevel := sm.level
	now := time.Now()

	if eventCount >= sm.config.EmergencyThreshold {
		sm.level = SafeModeEmergency
		sm.emergencyUntil = now.Add(sm.config.EmergencyPauseDuration)
		log.Printf("[SafeMode] EMERGENCY MODE ACTIVATED - pausing until %s",
			sm.emergencyUntil.Format(time.RFC3339))
		if sm.onEmergency != nil {
			go sm.onEmergency()
		}
	} else if eventCount >= sm.config.CriticalThreshold {
		sm.level = SafeModeCritical
	} else if eventCount >= sm.config.HighThreshold {
		sm.level = SafeModeHigh
	} else if eventCount >= sm.config.ElevatedThreshold {
		sm.level = SafeModeElevated
	}

	if sm.level != oldLevel {
		sm.lastEscalation = now
		log.Printf("[SafeMode] Escalated from %s to %s (events: %d)",
			oldLevel, sm.level, eventCount)

		if sm.onLevelChange != nil {
			go sm.onLevelChange(oldLevel, sm.level)
		}
	}
}

// CurrentLevel returns the current safe mode level
func (sm *SafeModeController) CurrentLevel() SafeModeLevel {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.level
}

// IsOperationAllowed checks if operations are currently allowed
func (sm *SafeModeController) IsOperationAllowed() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.level == SafeModeEmergency {
		return time.Now().After(sm.emergencyUntil)
	}
	return true
}

// GetDelayMultiplier returns the delay multiplier for current level
func (sm *SafeModeController) GetDelayMultiplier() float64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	switch sm.level {
	case SafeModeElevated:
		return sm.config.ElevatedDelayMultiplier
	case SafeModeHigh:
		return sm.config.HighDelayMultiplier
	case SafeModeCritical, SafeModeEmergency:
		return sm.config.CriticalDelayMultiplier
	default:
		return 1.0
	}
}

// WaitIfNeeded waits if emergency mode is active
func (sm *SafeModeController) WaitIfNeeded(ctx context.Context) error {
	sm.mu.RLock()
	level := sm.level
	emergencyUntil := sm.emergencyUntil
	sm.mu.RUnlock()

	if level != SafeModeEmergency {
		return nil
	}

	waitDuration := time.Until(emergencyUntil)
	if waitDuration <= 0 {
		return nil
	}

	log.Printf("[SafeMode] Waiting for emergency mode to end (%s remaining)", waitDuration)

	timer := time.NewTimer(waitDuration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// SetLevelChangeCallback sets callback for level changes
func (sm *SafeModeController) SetLevelChangeCallback(fn func(oldLevel, newLevel SafeModeLevel)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.onLevelChange = fn
}

// SetEmergencyCallback sets callback for emergency mode activation
func (sm *SafeModeController) SetEmergencyCallback(fn func()) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.onEmergency = fn
}

// ForceLevel forces a specific safe mode level (for testing/manual control)
func (sm *SafeModeController) ForceLevel(level SafeModeLevel) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	oldLevel := sm.level
	sm.level = level

	if level == SafeModeEmergency {
		sm.emergencyUntil = time.Now().Add(sm.config.EmergencyPauseDuration)
	}

	log.Printf("[SafeMode] Forced level change from %s to %s", oldLevel, level)

	if sm.onLevelChange != nil && oldLevel != level {
		go sm.onLevelChange(oldLevel, level)
	}
}

// Reset resets the safe mode controller to normal state
func (sm *SafeModeController) Reset() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	oldLevel := sm.level
	sm.level = SafeModeNormal
	sm.events = make([]FloodWaitEvent, 0)
	sm.emergencyUntil = time.Time{}

	log.Printf("[SafeMode] Reset from %s to NORMAL", oldLevel)

	if sm.onLevelChange != nil && oldLevel != SafeModeNormal {
		go sm.onLevelChange(oldLevel, SafeModeNormal)
	}
}

// Stop stops the safe mode controller
func (sm *SafeModeController) Stop() {
	sm.cancelFunc()
	sm.wg.Wait()
}

// SafeModeStats holds statistics about safe mode state
type SafeModeStats struct {
	CurrentLevel       SafeModeLevel    `json:"current_level"`
	LevelString        string           `json:"level_string"`
	EventCount         int              `json:"event_count"`
	LastEscalation     time.Time        `json:"last_escalation"`
	LastDeescalation   time.Time        `json:"last_deescalation"`
	EmergencyUntil     time.Time        `json:"emergency_until,omitempty"`
	IsOperationAllowed bool             `json:"is_operation_allowed"`
	DelayMultiplier    float64          `json:"delay_multiplier"`
	RecentEvents       []FloodWaitEvent `json:"recent_events,omitempty"`
}

// Stats returns current safe mode statistics
func (sm *SafeModeController) Stats(includeEvents bool) SafeModeStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stats := SafeModeStats{
		CurrentLevel:       sm.level,
		LevelString:        sm.level.String(),
		EventCount:         len(sm.events),
		LastEscalation:     sm.lastEscalation,
		LastDeescalation:   sm.lastDeescalation,
		IsOperationAllowed: sm.level != SafeModeEmergency || time.Now().After(sm.emergencyUntil),
		DelayMultiplier:    sm.GetDelayMultiplier(),
	}

	if sm.level == SafeModeEmergency {
		stats.EmergencyUntil = sm.emergencyUntil
	}

	if includeEvents {
		stats.RecentEvents = make([]FloodWaitEvent, len(sm.events))
		copy(stats.RecentEvents, sm.events)
	}

	return stats
}

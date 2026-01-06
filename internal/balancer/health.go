package balancer

import (
	"net/url"
	"sync"
	"time"
)

const (
	DefaultFailureThreshold  = 3
	DefaultUnhealthyCooldown = 15 * time.Second
)

type backendHealth struct {
	consecutiveFailures int
	unhealthyUntil      time.Time
}

// BackendStatus exposes backend health information for reporting.
type BackendStatus struct {
	ConsecutiveFailures int       `json:"consecutive_failures"`
	UnhealthyUntil      time.Time `json:"unhealthy_until"`
}

// HealthTracker tracks backend health based on consecutive failures.
type HealthTracker struct {
	mu               sync.RWMutex
	state            map[string]*backendHealth
	failureThreshold int
	cooldown         time.Duration
	now              func() time.Time
}

// NewHealthTracker creates a new tracker with the given failure threshold and cooldown.
// Zero values use DefaultFailureThreshold and DefaultUnhealthyCooldown.
func NewHealthTracker(failureThreshold int, cooldown time.Duration) *HealthTracker {
	if failureThreshold <= 0 {
		failureThreshold = DefaultFailureThreshold
	}
	if cooldown <= 0 {
		cooldown = DefaultUnhealthyCooldown
	}
	return &HealthTracker{
		state:            make(map[string]*backendHealth),
		failureThreshold: failureThreshold,
		cooldown:         cooldown,
		now:              time.Now,
	}
}

// MarkFailure records a failure for the backend and marks it unhealthy when threshold is reached.
func (h *HealthTracker) MarkFailure(target *url.URL) {
	if h == nil || target == nil {
		return
	}
	key := target.String()
	if key == "" {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	state := h.state[key]
	if state == nil {
		state = &backendHealth{}
		h.state[key] = state
	}

	state.consecutiveFailures++
	if state.consecutiveFailures >= h.failureThreshold {
		state.unhealthyUntil = h.now().Add(h.cooldown)
	}
}

// MarkSuccess records a successful response and clears failure state.
func (h *HealthTracker) MarkSuccess(target *url.URL) {
	if h == nil || target == nil {
		return
	}
	key := target.String()
	if key == "" {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	state := h.state[key]
	if state == nil {
		state = &backendHealth{}
		h.state[key] = state
	}

	state.consecutiveFailures = 0
	state.unhealthyUntil = time.Time{}
}

// IsHealthy reports whether the backend should be considered healthy.
func (h *HealthTracker) IsHealthy(target *url.URL) bool {
	if h == nil || target == nil {
		return false
	}
	key := target.String()
	if key == "" {
		return false
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	state := h.state[key]
	if state == nil {
		return true
	}
	if state.unhealthyUntil.IsZero() {
		return true
	}
	if h.now().After(state.unhealthyUntil) {
		state.unhealthyUntil = time.Time{}
		state.consecutiveFailures = 0
		return true
	}
	return false
}

// Snapshot returns a copy of the current backend health state.
func (h *HealthTracker) Snapshot() map[string]BackendStatus {
	if h == nil {
		return nil
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	snapshot := make(map[string]BackendStatus, len(h.state))
	for key, state := range h.state {
		snapshot[key] = BackendStatus{
			ConsecutiveFailures: state.consecutiveFailures,
			UnhealthyUntil:      state.unhealthyUntil,
		}
	}

	return snapshot
}

type healthAwareSelector struct {
	base    Selector
	targets []*url.URL
	tracker *HealthTracker
}

// NewHealthAwareSelector returns a selector that skips unhealthy backends.
func NewHealthAwareSelector(targets []*url.URL, algorithm string, tracker *HealthTracker) (Selector, error) {
	selector, err := NewSelector(targets, algorithm)
	if tracker == nil {
		return selector, err
	}

	return &healthAwareSelector{
		base:    selector,
		targets: targets,
		tracker: tracker,
	}, err
}

func (s *healthAwareSelector) Next() *url.URL {
	if s.tracker == nil || len(s.targets) == 0 {
		return s.base.Next()
	}

	var candidate *url.URL
	for i := 0; i < len(s.targets); i++ {
		candidate = s.base.Next()
		if s.tracker.IsHealthy(candidate) {
			return candidate
		}
	}

	return candidate
}

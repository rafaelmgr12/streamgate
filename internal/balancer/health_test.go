package balancer

import (
	"net/url"
	"testing"
	"time"
)

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("failed to parse URL %q: %v", raw, err)
	}
	return u
}

func TestHealthTracker_MarksUnhealthyAndRecovers(t *testing.T) {
	tracker := NewHealthTracker(3, 15*time.Second)
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	tracker.now = func() time.Time { return now }

	target := mustParseURL(t, "http://one")

	tracker.MarkFailure(target)
	tracker.MarkFailure(target)
	if !tracker.IsHealthy(target) {
		t.Fatalf("expected target to remain healthy before threshold")
	}

	tracker.MarkFailure(target)
	if tracker.IsHealthy(target) {
		t.Fatalf("expected target to be unhealthy after threshold")
	}

	now = now.Add(16 * time.Second)
	if !tracker.IsHealthy(target) {
		t.Fatalf("expected target to recover after cooldown")
	}

	tracker.MarkFailure(target)
	if !tracker.IsHealthy(target) {
		t.Fatalf("expected target to be healthy after reset")
	}
}

func TestHealthTracker_MarkSuccessResets(t *testing.T) {
	tracker := NewHealthTracker(1, 15*time.Second)
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	tracker.now = func() time.Time { return now }

	target := mustParseURL(t, "http://one")
	tracker.MarkFailure(target)
	if tracker.IsHealthy(target) {
		t.Fatalf("expected target to be unhealthy after failure")
	}

	tracker.MarkSuccess(target)
	if !tracker.IsHealthy(target) {
		t.Fatalf("expected target to be healthy after success")
	}
}

func TestHealthAwareSelector_SkipsUnhealthyTargets(t *testing.T) {
	tracker := NewHealthTracker(1, 15*time.Second)
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	tracker.now = func() time.Time { return now }

	targets := []*url.URL{
		mustParseURL(t, "http://one"),
		mustParseURL(t, "http://two"),
		mustParseURL(t, "http://three"),
	}

	tracker.MarkFailure(targets[1])

	selector, err := NewHealthAwareSelector(targets, DefaultAlgorithm, tracker)
	if err != nil {
		t.Fatalf("NewHealthAwareSelector() error = %v", err)
	}

	for i := 0; i < 6; i++ {
		got := selector.Next()
		if got.Host == targets[1].Host {
			t.Fatalf("expected unhealthy target to be skipped, got %q", got.Host)
		}
	}
}

func TestHealthAwareSelector_AllUnhealthyFallback(t *testing.T) {
	tracker := NewHealthTracker(1, 15*time.Second)
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	tracker.now = func() time.Time { return now }

	targets := []*url.URL{
		mustParseURL(t, "http://one"),
		mustParseURL(t, "http://two"),
	}

	for _, target := range targets {
		tracker.MarkFailure(target)
	}

	selector, err := NewHealthAwareSelector(targets, DefaultAlgorithm, tracker)
	if err != nil {
		t.Fatalf("NewHealthAwareSelector() error = %v", err)
	}

	got := selector.Next()
	if got == nil {
		t.Fatalf("expected a target even when all are unhealthy")
	}
	if !isTargetInList(got, targets) {
		t.Fatalf("expected target from list, got %q", got.Host)
	}
}

func isTargetInList(target *url.URL, targets []*url.URL) bool {
	for _, candidate := range targets {
		if target == candidate || target.String() == candidate.String() {
			return true
		}
	}
	return false
}

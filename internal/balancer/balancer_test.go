package balancer_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/rafaelmgr12/streamgate/internal/balancer"
)

type staticSelector struct {
	target *url.URL
}

func (s *staticSelector) Next() *url.URL {
	return s.target
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("failed to parse URL %q: %v", raw, err)
	}
	return u
}

func TestRegister_EmptyNameFails(t *testing.T) {
	err := balancer.Register(" ", func(targets []*url.URL) balancer.Selector {
		return &staticSelector{target: targets[0]}
	})
	if err == nil {
		t.Fatalf("expected error for empty algorithm name, got nil")
	}

	if !strings.Contains(err.Error(), "empty algorithm name") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestRegister_NilFactoryFails(t *testing.T) {
	err := balancer.Register("custom_algo", nil)
	if err == nil {
		t.Fatalf("expected error for nil factory, got nil")
	}

	if !strings.Contains(err.Error(), "nil factory") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestRegister_SuccessAndUsedByNewSelector(t *testing.T) {
	targets := []*url.URL{
		mustParseURL(t, "http://one"),
		mustParseURL(t, "http://two"),
	}

	const algo = "MyAlgo"
	err := balancer.Register(algo, func(ts []*url.URL) balancer.Selector {
		// Custom behavior: always return the last target
		if len(ts) == 0 {
			return &staticSelector{}
		}
		return &staticSelector{target: ts[len(ts)-1]}
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	selector, err := balancer.NewSelector(targets, algo)
	if err != nil {
		t.Fatalf("NewSelector() error = %v", err)
	}

	u1 := selector.Next()
	u2 := selector.Next()

	if u1.Host != "two" || u2.Host != "two" {
		t.Fatalf("expected custom selector to always return host %q, got %q and %q", "two", u1.Host, u2.Host)
	}
}

func TestNewSelector_DefaultAlgorithmRoundRobin(t *testing.T) {
	targets := []*url.URL{
		mustParseURL(t, "http://one"),
		mustParseURL(t, "http://two"),
		mustParseURL(t, "http://three"),
	}

	selector, err := balancer.NewSelector(targets, balancer.DefaultAlgorithm)
	if err != nil {
		t.Fatalf("NewSelector() error = %v", err)
	}

	wantHosts := []string{"one", "two", "three", "one", "two", "three"}
	for i, want := range wantHosts {
		got := selector.Next()
		if got.Host != want {
			t.Fatalf("call %d: expected host %q, got %q", i, want, got.Host)
		}
	}
}

func TestNewSelector_EmptyAlgorithmUsesDefault(t *testing.T) {
	targets := []*url.URL{
		mustParseURL(t, "http://one"),
		mustParseURL(t, "http://two"),
	}

	selector, err := balancer.NewSelector(targets, "   ")
	if err != nil {
		t.Fatalf("NewSelector() error = %v", err)
	}

	u1 := selector.Next()
	u2 := selector.Next()

	if u1.Host != "one" || u2.Host != "two" {
		t.Fatalf("expected default round-robin hosts %q, %q; got %q, %q", "one", "two", u1.Host, u2.Host)
	}
}

func TestNewSelector_UnknownAlgorithmFallsBackWithError(t *testing.T) {
	targets := []*url.URL{
		mustParseURL(t, "http://one"),
		mustParseURL(t, "http://two"),
		mustParseURL(t, "http://three"),
	}

	selector, err := balancer.NewSelector(targets, "does-not-exist")
	if err == nil {
		t.Fatalf("expected error for unknown algorithm, got nil")
	}
	if !strings.Contains(err.Error(), "unknown load balancing algorithm") {
		t.Fatalf("unexpected error: %v", err)
	}

	// Still should behave like the default round-robin
	wantHosts := []string{"one", "two", "three"}
	for i, want := range wantHosts {
		got := selector.Next()
		if got.Host != want {
			t.Fatalf("call %d: expected host %q, got %q", i, want, got.Host)
		}
	}
}

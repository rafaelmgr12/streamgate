package balancer

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
)

// Selector chooses the next backend URL to use.
type Selector interface {
	Next() *url.URL
}

// Factory constructs a Selector for a given backend set.
type Factory func([]*url.URL) Selector

const DefaultAlgorithm = "round_robin"

var (
	factoriesMu sync.RWMutex
	factories   = map[string]Factory{
		DefaultAlgorithm: func(targets []*url.URL) Selector {
			return &roundRobinSelector{targets: targets}
		},
	}
)

// Register adds or replaces a selector factory for an algorithm name.
func Register(name string, factory Factory) error {
	normalized := normalizeAlgorithm(name)
	if normalized == "" {
		return fmt.Errorf("empty algorithm name")
	}
	if factory == nil {
		return fmt.Errorf("nil factory for %q", normalized)
	}

	factoriesMu.Lock()
	factories[normalized] = factory
	factoriesMu.Unlock()
	return nil
}

// NewSelector returns a selector for the algorithm. Unknown algorithms fall back to DefaultAlgorithm.
func NewSelector(targets []*url.URL, algorithm string) (Selector, error) {
	normalized := normalizeAlgorithm(algorithm)
	if normalized == "" {
		normalized = DefaultAlgorithm
	}

	factoriesMu.RLock()
	factory, ok := factories[normalized]
	if !ok {
		factory = factories[DefaultAlgorithm]
	}
	factoriesMu.RUnlock()

	selector := factory(targets)
	if !ok {
		return selector, fmt.Errorf("unknown load balancing algorithm %q", normalized)
	}
	return selector, nil
}

func normalizeAlgorithm(name string) string {
	return strings.TrimSpace(strings.ToLower(name))
}

type roundRobinSelector struct {
	targets []*url.URL
	idx     uint64
}

func (s *roundRobinSelector) Next() *url.URL {
	n := atomic.AddUint64(&s.idx, 1)
	return s.targets[(n-1)%uint64(len(s.targets))]
}

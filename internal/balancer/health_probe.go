package balancer

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultProbeInterval = 10 * time.Second
	DefaultProbeTimeout  = 2 * time.Second
	DefaultHealthPath    = "/healthz"
)

// ActiveHealthChecker probes backends periodically and updates the HealthTracker.
type ActiveHealthChecker struct {
	tracker  *HealthTracker
	targets  []*url.URL
	interval time.Duration
	timeout  time.Duration
	path     string
	client   *http.Client
}

// NewActiveHealthChecker creates a new active checker.
// Zero values use defaults for interval, timeout, and path.
func NewActiveHealthChecker(tracker *HealthTracker, targets []*url.URL, interval, timeout time.Duration, path string) *ActiveHealthChecker {
	if interval <= 0 {
		interval = DefaultProbeInterval
	}
	if timeout <= 0 {
		timeout = DefaultProbeTimeout
	}
	if strings.TrimSpace(path) == "" {
		path = DefaultHealthPath
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return &ActiveHealthChecker{
		tracker:  tracker,
		targets:  targets,
		interval: interval,
		timeout:  timeout,
		path:     path,
		client:   &http.Client{Timeout: timeout},
	}
}

// Start begins probing until the context is canceled.
func (c *ActiveHealthChecker) Start(ctx context.Context) {
	if c == nil || c.tracker == nil || len(c.targets) == 0 {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	ticker := time.NewTicker(c.interval)
	go func() {
		defer ticker.Stop()
		c.checkOnce(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.checkOnce(ctx)
			}
		}
	}()
}

func (c *ActiveHealthChecker) checkOnce(ctx context.Context) {
	for _, target := range c.targets {
		c.probeTarget(ctx, target)
	}
}

func (c *ActiveHealthChecker) probeTarget(ctx context.Context, target *url.URL) {
	if target == nil {
		return
	}
	reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	checkURL := c.buildProbeURL(target)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, checkURL.String(), nil)
	if err != nil {
		c.tracker.MarkFailure(target)
		return
	}

	resp, err := c.client.Do(req)
	if err != nil {
		c.tracker.MarkFailure(target)
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusBadRequest {
		c.tracker.MarkSuccess(target)
		return
	}

	c.tracker.MarkFailure(target)
}

func (c *ActiveHealthChecker) buildProbeURL(target *url.URL) *url.URL {
	copyURL := *target
	copyURL.Path = c.path
	copyURL.RawQuery = ""
	copyURL.Fragment = ""
	return &copyURL
}

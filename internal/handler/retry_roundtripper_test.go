package handler

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRoundTripper struct {
	calls     int
	failFirst bool
}

func (f *fakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	f.calls++
	if f.failFirst && f.calls == 1 {
		return nil, errors.New("boom")
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("ok")),
		Header:     make(http.Header),
	}, nil
}

type statusRoundTripper struct {
	calls int
}

func (s *statusRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	s.calls++
	status := http.StatusOK
	if s.calls == 1 {
		status = http.StatusServiceUnavailable
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader("ok")),
		Header:     make(http.Header),
	}, nil
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return u
}

func TestRetryRoundTripper_RetriesOnceOnError(t *testing.T) {
	base := &fakeRoundTripper{failFirst: true}
	rt := &retryRoundTripper{
		base:       base,
		maxRetries: 1,
	}

	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)

	resp, err := rt.RoundTrip(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()

	assert.Equal(t, 2, base.calls)
}

func TestRetryRoundTripper_RetriesOnceOnRetryableStatus(t *testing.T) {
	base := &statusRoundTripper{}
	rt := &retryRoundTripper{
		base:       base,
		maxRetries: 1,
	}

	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)

	resp, err := rt.RoundTrip(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()

	assert.Equal(t, 2, base.calls)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRetryRoundTripper_NoRetryForNonIdempotent(t *testing.T) {
	base := &fakeRoundTripper{failFirst: true}
	rt := &retryRoundTripper{
		base:       base,
		maxRetries: 1,
	}

	req, err := http.NewRequest(http.MethodPost, "http://example.com", strings.NewReader("x"))
	require.NoError(t, err)

	_, err = rt.RoundTrip(req)
	require.Error(t, err)
	assert.Equal(t, 1, base.calls)
}

func TestRetryRoundTripper_NoRetryWhenBodyNotReplayable(t *testing.T) {
	base := &fakeRoundTripper{failFirst: true}
	rt := &retryRoundTripper{
		base:       base,
		maxRetries: 1,
	}

	req := &http.Request{
		Method: http.MethodGet,
		URL:    mustURL(t, "http://example.com"),
		Body:   io.NopCloser(strings.NewReader("x")),
	}

	_, err := rt.RoundTrip(req)
	require.Error(t, err)
	assert.Equal(t, 1, base.calls)
}

func TestIsIdempotent(t *testing.T) {
	assert.True(t, isIdempotent(http.MethodGet))
	assert.True(t, isIdempotent(http.MethodDelete))
	assert.False(t, isIdempotent(http.MethodPost))
	assert.False(t, isIdempotent("PATCH"))
}

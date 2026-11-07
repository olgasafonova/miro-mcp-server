package miro

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// recordingTransport answers every request with an empty JSON object and
// records the full URL it was sent to. A request whose path contains
// blockOn waits until release is closed, so a test can hold one call open
// while another runs.
type recordingTransport struct {
	mu      sync.Mutex
	urls    map[string]string // path -> full URL
	blockOn string
	started chan struct{}
	release chan struct{}
}

func (rt *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.mu.Lock()
	rt.urls[req.URL.Path] = req.URL.String()
	rt.mu.Unlock()

	if rt.blockOn != "" && strings.Contains(req.URL.Path, rt.blockOn) {
		close(rt.started)
		<-rt.release
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader("{}")),
		Request:    req,
	}, nil
}

func (rt *recordingTransport) url(path string) string {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.urls[path]
}

// newProductionURLClient returns a client that keeps the production BaseURL
// (so the version-routing logic is exercised) but never leaves the process.
func newProductionURLClient(rt http.RoundTripper) *Client {
	c := NewClient(testConfig(), testLogger())
	c.httpClient = &http.Client{Transport: rt, Timeout: 5 * time.Second}
	return c
}

func TestVersionedRequests_RouteToTheirBaseURL(t *testing.T) {
	rt := &recordingTransport{urls: map[string]string{}}
	c := newProductionURLClient(rt)
	ctx := context.Background()

	if _, err := c.request(ctx, http.MethodGet, "/boards/b1", nil); err != nil {
		t.Fatalf("request: %v", err)
	}
	if _, err := c.requestExperimental(ctx, http.MethodGet, "/boards/b1/mindmap_nodes", nil); err != nil {
		t.Fatalf("requestExperimental: %v", err)
	}
	if _, err := c.requestV1(ctx, http.MethodGet, "/oauth-token", nil); err != nil {
		t.Fatalf("requestV1: %v", err)
	}

	cases := map[string]string{
		"/v2/boards/b1": BaseURL + "/boards/b1",
		"/v2-experimental/boards/b1/mindmap_nodes": ExperimentalBaseURL + "/boards/b1/mindmap_nodes",
		"/v1/oauth-token":                          V1BaseURL + "/oauth-token",
	}
	for path, want := range cases {
		if got := rt.url(path); got != want {
			t.Errorf("request for %s went to %q, want %q", path, got, want)
		}
	}
}

// TestRequestExperimental_DoesNotRedirectConcurrentV2Calls reproduces the
// shared-baseURL bug: requestExperimental used to overwrite c.baseURL for the
// whole call, so a plain v2 request issued meanwhile went to v2-experimental.
func TestRequestExperimental_DoesNotRedirectConcurrentV2Calls(t *testing.T) {
	rt := &recordingTransport{
		urls:    map[string]string{},
		blockOn: "mindmap_nodes",
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	c := newProductionURLClient(rt)
	ctx := context.Background()

	done := make(chan error, 1)
	go func() {
		_, err := c.requestExperimental(ctx, http.MethodGet, "/boards/b1/mindmap_nodes", nil)
		done <- err
	}()

	<-rt.started // the experimental call is now in flight
	_, err := c.request(ctx, http.MethodGet, "/boards/b2", nil)
	close(rt.release)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("requestExperimental: %v", err)
	}

	want := BaseURL + "/boards/b2"
	if got := rt.url("/v2/boards/b2"); got != want {
		t.Fatalf("concurrent v2 request went to %q (recorded paths: %v), want %q", got, rt.urls, want)
	}
}

func TestVersionedRequests_KeepTestServerURL(t *testing.T) {
	rt := &recordingTransport{urls: map[string]string{}}
	c := newProductionURLClient(rt)
	c.baseURL = "http://127.0.0.1:9999"

	if _, err := c.requestExperimental(context.Background(), http.MethodGet, "/x", nil); err != nil {
		t.Fatalf("requestExperimental: %v", err)
	}
	if got, want := rt.url("/x"), "http://127.0.0.1:9999/x"; got != want {
		t.Fatalf("test-server request went to %q, want %q", got, want)
	}
}

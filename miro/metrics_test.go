package miro

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMetricsCollector_RecordRequest(t *testing.T) {
	m := NewMetricsCollector()

	// Record some requests
	m.RecordRequest("GET", 100*time.Millisecond, nil)
	m.RecordRequest("POST", 200*time.Millisecond, nil)
	m.RecordRequest("GET", 150*time.Millisecond, nil)

	metrics := m.GetMetrics()

	if metrics.TotalRequests != 3 {
		t.Errorf("expected 3 total requests, got %d", metrics.TotalRequests)
	}

	if metrics.RequestsByMethod["GET"] != 2 {
		t.Errorf("expected 2 GET requests, got %d", metrics.RequestsByMethod["GET"])
	}

	if metrics.RequestsByMethod["POST"] != 1 {
		t.Errorf("expected 1 POST request, got %d", metrics.RequestsByMethod["POST"])
	}
}

func TestMetricsCollector_RecordErrors(t *testing.T) {
	m := NewMetricsCollector()

	// Record requests with errors
	m.RecordRequest("GET", 100*time.Millisecond, &APIError{StatusCode: 429, Message: "Rate limited"})
	m.RecordRequest("GET", 100*time.Millisecond, &APIError{StatusCode: 401, Message: "Unauthorized"})
	m.RecordRequest("GET", 100*time.Millisecond, &APIError{StatusCode: 500, Message: "Server error"})
	m.RecordRequest("GET", 100*time.Millisecond, nil) // No error

	metrics := m.GetMetrics()

	if metrics.TotalErrors != 3 {
		t.Errorf("expected 3 total errors, got %d", metrics.TotalErrors)
	}

	if metrics.ErrorsByType["rate_limit"] != 1 {
		t.Errorf("expected 1 rate_limit error, got %d", metrics.ErrorsByType["rate_limit"])
	}

	if metrics.ErrorsByType["auth"] != 1 {
		t.Errorf("expected 1 auth error, got %d", metrics.ErrorsByType["auth"])
	}

	if metrics.ErrorsByType["server"] != 1 {
		t.Errorf("expected 1 server error, got %d", metrics.ErrorsByType["server"])
	}
}

func TestMetricsCollector_RateLimitAndRetries(t *testing.T) {
	m := NewMetricsCollector()

	m.RecordRateLimitHit()
	m.RecordRateLimitHit()
	m.RecordRetry()
	m.RecordRetry()
	m.RecordRetry()

	metrics := m.GetMetrics()

	if metrics.RateLimitHits != 2 {
		t.Errorf("expected 2 rate limit hits, got %d", metrics.RateLimitHits)
	}

	if metrics.RetryCount != 3 {
		t.Errorf("expected 3 retries, got %d", metrics.RetryCount)
	}
}

func TestMetricsCollector_Percentiles(t *testing.T) {
	m := NewMetricsCollector()

	// Record requests with varying durations
	durations := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		40 * time.Millisecond,
		50 * time.Millisecond,
		60 * time.Millisecond,
		70 * time.Millisecond,
		80 * time.Millisecond,
		90 * time.Millisecond,
		100 * time.Millisecond,
	}

	for _, d := range durations {
		m.RecordRequest("GET", d, nil)
	}

	metrics := m.GetMetrics()

	// With 10 samples: p50=50ms, p95=90ms, p99=90ms
	if metrics.LatencyP50Ms < 40 || metrics.LatencyP50Ms > 60 {
		t.Errorf("expected P50 around 50ms, got %d", metrics.LatencyP50Ms)
	}
}

func TestMetricsCollector_PrometheusHandler(t *testing.T) {
	m := NewMetricsCollector()

	// Record some data
	m.RecordRequest("GET", 100*time.Millisecond, nil)
	m.RecordRequest("POST", 200*time.Millisecond, &APIError{StatusCode: 500, Message: "error"})
	m.RecordRateLimitHit()
	m.RecordRetry()

	// Create request to the handler
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler := m.PrometheusHandler()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()

	// Check for expected metrics
	expectedMetrics := []string{
		"miro_mcp_requests_total",
		"miro_mcp_errors_total",
		"miro_mcp_rate_limit_hits_total",
		"miro_mcp_retries_total",
		"miro_mcp_uptime_seconds",
		"miro_mcp_requests_by_method",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(body, metric) {
			t.Errorf("expected metric %s in output", metric)
		}
	}

	// Check content type
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("expected text/plain content type, got %s", contentType)
	}
}

func TestMetricsCollector_UptimeTracking(t *testing.T) {
	m := NewMetricsCollector()

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	metrics := m.GetMetrics()

	if metrics.UptimeSeconds < 0 {
		t.Error("expected non-negative uptime")
	}
}

func TestCategorizeError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"nil error", nil, "none"},
		{"rate limit", &APIError{StatusCode: 429}, "rate_limit"},
		{"auth error", &APIError{StatusCode: 401}, "auth"},
		{"forbidden", &APIError{StatusCode: 403}, "forbidden"},
		{"not found", &APIError{StatusCode: 404}, "not_found"},
		{"server error 500", &APIError{StatusCode: 500}, "server"},
		{"server error 503", &APIError{StatusCode: 503}, "server"},
		{"client error 400", &APIError{StatusCode: 400}, "client"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := categorizeError(tt.err)
			if result != tt.expected {
				t.Errorf("categorizeError(%v) = %s, want %s", tt.err, result, tt.expected)
			}
		})
	}
}

func TestSortInt64s(t *testing.T) {
	tests := []struct {
		name     string
		input    []int64
		expected []int64
	}{
		{"empty", []int64{}, []int64{}},
		{"single", []int64{5}, []int64{5}},
		{"sorted", []int64{1, 2, 3}, []int64{1, 2, 3}},
		{"reverse", []int64{3, 2, 1}, []int64{1, 2, 3}},
		{"mixed", []int64{3, 1, 4, 1, 5, 9, 2, 6}, []int64{1, 1, 2, 3, 4, 5, 6, 9}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := make([]int64, len(tt.input))
			copy(input, tt.input)
			sortInt64s(input)

			for i, v := range input {
				if v != tt.expected[i] {
					t.Errorf("sortInt64s(%v) at index %d = %d, want %d", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}

package miro

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

// MetricsCollector collects and exports Prometheus-compatible metrics
type MetricsCollector struct {
	mu sync.RWMutex

	// Counters
	requestsTotal      map[string]int64 // method -> count
	requestErrorsTotal map[string]int64 // error_type -> count
	rateLimitHits      int64
	retriesTotal       int64

	// Histograms (simplified: track counts per bucket)
	requestDurations   []float64 // all durations in seconds
	requestDurationsMs []int64   // for more precise tracking

	// Gauges
	startTime     time.Time
	lastRequestAt time.Time

	// Config
	enabled bool
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		requestsTotal:      make(map[string]int64),
		requestErrorsTotal: make(map[string]int64),
		requestDurations:   make([]float64, 0, 1000),
		requestDurationsMs: make([]int64, 0, 1000),
		startTime:          time.Now(),
		enabled:            true,
	}
}

// RecordRequest records a completed API request
func (m *MetricsCollector) RecordRequest(method string, duration time.Duration, err error) {
	if !m.enabled {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Increment request counter
	m.requestsTotal[method]++

	// Record duration
	durationSec := duration.Seconds()
	m.requestDurations = append(m.requestDurations, durationSec)
	m.requestDurationsMs = append(m.requestDurationsMs, duration.Milliseconds())

	// Keep bounded (last 10000 samples)
	if len(m.requestDurations) > 10000 {
		m.requestDurations = m.requestDurations[len(m.requestDurations)-10000:]
		m.requestDurationsMs = m.requestDurationsMs[len(m.requestDurationsMs)-10000:]
	}

	m.lastRequestAt = time.Now()

	// Record errors
	if err != nil {
		errType := categorizeError(err)
		m.requestErrorsTotal[errType]++
	}
}

// RecordRateLimitHit records a rate limit encounter
func (m *MetricsCollector) RecordRateLimitHit() {
	if !m.enabled {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.rateLimitHits++
}

// RecordRetry records a retry attempt
func (m *MetricsCollector) RecordRetry() {
	if !m.enabled {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.retriesTotal++
}

// categorizeError categorizes an error for metrics
func categorizeError(err error) string {
	if err == nil {
		return "none"
	}

	// Check for API errors
	if apiErr, ok := err.(*APIError); ok {
		switch {
		case apiErr.StatusCode == 429:
			return "rate_limit"
		case apiErr.StatusCode == 401:
			return "auth"
		case apiErr.StatusCode == 403:
			return "forbidden"
		case apiErr.StatusCode == 404:
			return "not_found"
		case apiErr.StatusCode >= 500:
			return "server"
		case apiErr.StatusCode >= 400:
			return "client"
		}
	}

	return "unknown"
}

// GetMetrics returns current metrics snapshot
func (m *MetricsCollector) GetMetrics() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Calculate percentiles from durations
	p50, p95, p99 := calculatePercentiles(m.requestDurationsMs)

	// Calculate total requests
	var totalRequests int64
	for _, count := range m.requestsTotal {
		totalRequests += count
	}

	// Calculate total errors
	var totalErrors int64
	for _, count := range m.requestErrorsTotal {
		totalErrors += count
	}

	// Copy maps
	requestsByMethod := make(map[string]int64)
	for k, v := range m.requestsTotal {
		requestsByMethod[k] = v
	}

	errorsByType := make(map[string]int64)
	for k, v := range m.requestErrorsTotal {
		errorsByType[k] = v
	}

	return MetricsSnapshot{
		TotalRequests:    totalRequests,
		TotalErrors:      totalErrors,
		RateLimitHits:    m.rateLimitHits,
		RetryCount:       m.retriesTotal,
		RequestsByMethod: requestsByMethod,
		ErrorsByType:     errorsByType,
		LatencyP50Ms:     p50,
		LatencyP95Ms:     p95,
		LatencyP99Ms:     p99,
		UptimeSeconds:    int64(time.Since(m.startTime).Seconds()),
		LastRequestAt:    m.lastRequestAt,
	}
}

// MetricsSnapshot represents a point-in-time view of metrics
type MetricsSnapshot struct {
	TotalRequests    int64            `json:"total_requests"`
	TotalErrors      int64            `json:"total_errors"`
	RateLimitHits    int64            `json:"rate_limit_hits"`
	RetryCount       int64            `json:"retry_count"`
	RequestsByMethod map[string]int64 `json:"requests_by_method"`
	ErrorsByType     map[string]int64 `json:"errors_by_type"`
	LatencyP50Ms     int64            `json:"latency_p50_ms"`
	LatencyP95Ms     int64            `json:"latency_p95_ms"`
	LatencyP99Ms     int64            `json:"latency_p99_ms"`
	UptimeSeconds    int64            `json:"uptime_seconds"`
	LastRequestAt    time.Time        `json:"last_request_at"`
}

// PrometheusHandler returns an HTTP handler that serves Prometheus-format metrics
func (m *MetricsCollector) PrometheusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics := m.GetMetrics()

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		// Write metrics in Prometheus format
		writeMetric(w, "miro_mcp_requests_total", "Total number of API requests", "counter", float64(metrics.TotalRequests))
		writeMetric(w, "miro_mcp_errors_total", "Total number of errors", "counter", float64(metrics.TotalErrors))
		writeMetric(w, "miro_mcp_rate_limit_hits_total", "Total rate limit encounters", "counter", float64(metrics.RateLimitHits))
		writeMetric(w, "miro_mcp_retries_total", "Total retry attempts", "counter", float64(metrics.RetryCount))

		// Request latency percentiles
		writeMetric(w, "miro_mcp_request_duration_p50_milliseconds", "50th percentile request duration", "gauge", float64(metrics.LatencyP50Ms))
		writeMetric(w, "miro_mcp_request_duration_p95_milliseconds", "95th percentile request duration", "gauge", float64(metrics.LatencyP95Ms))
		writeMetric(w, "miro_mcp_request_duration_p99_milliseconds", "99th percentile request duration", "gauge", float64(metrics.LatencyP99Ms))

		// Uptime
		writeMetric(w, "miro_mcp_uptime_seconds", "Server uptime in seconds", "gauge", float64(metrics.UptimeSeconds))

		// Per-method request counts
		for method, count := range metrics.RequestsByMethod {
			writeMetricWithLabel(w, "miro_mcp_requests_by_method", "Requests by HTTP method", "counter", "method", method, float64(count))
		}

		// Per-type error counts
		for errType, count := range metrics.ErrorsByType {
			writeMetricWithLabel(w, "miro_mcp_errors_by_type", "Errors by type", "counter", "type", errType, float64(count))
		}
	}
}

func writeMetric(w http.ResponseWriter, name, help, metricType string, value float64) {
	w.Write([]byte("# HELP " + name + " " + help + "\n"))
	w.Write([]byte("# TYPE " + name + " " + metricType + "\n"))
	w.Write([]byte(name + " " + strconv.FormatFloat(value, 'f', -1, 64) + "\n\n"))
}

func writeMetricWithLabel(w http.ResponseWriter, name, help, metricType, labelKey, labelValue string, value float64) {
	w.Write([]byte("# HELP " + name + " " + help + "\n"))
	w.Write([]byte("# TYPE " + name + " " + metricType + "\n"))
	w.Write([]byte(name + "{" + labelKey + "=\"" + labelValue + "\"} " + strconv.FormatFloat(value, 'f', -1, 64) + "\n\n"))
}

// calculatePercentiles calculates p50, p95, p99 from a slice of durations
func calculatePercentiles(durations []int64) (p50, p95, p99 int64) {
	n := len(durations)
	if n == 0 {
		return 0, 0, 0
	}

	// Make a copy and sort
	sorted := make([]int64, n)
	copy(sorted, durations)
	sortInt64s(sorted)

	p50 = sorted[n*50/100]
	p95 = sorted[n*95/100]
	if n > 1 {
		p99 = sorted[n*99/100]
	} else {
		p99 = sorted[n-1]
	}

	return p50, p95, p99
}

// sortInt64s sorts a slice of int64 in place (simple insertion sort for small slices)
func sortInt64s(s []int64) {
	n := len(s)
	for i := 1; i < n; i++ {
		key := s[i]
		j := i - 1
		for j >= 0 && s[j] > key {
			s[j+1] = s[j]
			j--
		}
		s[j+1] = key
	}
}

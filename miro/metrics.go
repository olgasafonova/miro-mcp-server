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

// apiErrorCategories maps explicit HTTP codes to their metrics category.
var apiErrorCategories = map[int]string{
	429: "rate_limit",
	401: "auth",
	403: "forbidden",
	404: "not_found",
}

// categorizeAPIError returns the metrics category for an *APIError, falling
// back to range buckets for unmapped codes.
func categorizeAPIError(apiErr *APIError) string {
	if cat, ok := apiErrorCategories[apiErr.StatusCode]; ok {
		return cat
	}
	if apiErr.StatusCode >= 500 {
		return "server"
	}
	if apiErr.StatusCode >= 400 {
		return "client"
	}
	return "unknown"
}

// categorizeError categorizes an error for metrics
func categorizeError(err error) string {
	if err == nil {
		return "none"
	}
	if apiErr, ok := err.(*APIError); ok {
		return categorizeAPIError(apiErr)
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
		writeCounter(w, "miro_mcp_requests_total", "Total number of API requests", float64(metrics.TotalRequests))
		writeCounter(w, "miro_mcp_errors_total", "Total number of errors", float64(metrics.TotalErrors))
		writeCounter(w, "miro_mcp_rate_limit_hits_total", "Total rate limit encounters", float64(metrics.RateLimitHits))
		writeCounter(w, "miro_mcp_retries_total", "Total retry attempts", float64(metrics.RetryCount))

		// Request latency percentiles
		writeGauge(w, "miro_mcp_request_duration_p50_milliseconds", "50th percentile request duration", float64(metrics.LatencyP50Ms))
		writeGauge(w, "miro_mcp_request_duration_p95_milliseconds", "95th percentile request duration", float64(metrics.LatencyP95Ms))
		writeGauge(w, "miro_mcp_request_duration_p99_milliseconds", "99th percentile request duration", float64(metrics.LatencyP99Ms))

		// Uptime
		writeGauge(w, "miro_mcp_uptime_seconds", "Server uptime in seconds", float64(metrics.UptimeSeconds))

		// Per-method request counts
		for method, count := range metrics.RequestsByMethod {
			promMetric{
				name: "miro_mcp_requests_by_method", help: "Requests by HTTP method",
				metricType: "counter", labelKey: "method", labelValue: method,
				value: float64(count),
			}.write(w)
		}

		// Per-type error counts
		for errType, count := range metrics.ErrorsByType {
			promMetric{
				name: "miro_mcp_errors_by_type", help: "Errors by type",
				metricType: "counter", labelKey: "type", labelValue: errType,
				value: float64(count),
			}.write(w)
		}
	}
}

// promMetric is a single Prometheus-format metric line: name, help text,
// metric type, an optional label pair, and the value.
type promMetric struct {
	name       string
	help       string
	metricType string
	labelKey   string
	labelValue string
	value      float64
}

// write renders the metric's HELP/TYPE header and sample line.
func (p promMetric) write(w http.ResponseWriter) {
	series := p.name
	if p.labelKey != "" {
		series += "{" + p.labelKey + "=\"" + p.labelValue + "\"}"
	}
	w.Write([]byte("# HELP " + p.name + " " + p.help + "\n"))
	w.Write([]byte("# TYPE " + p.name + " " + p.metricType + "\n"))
	w.Write([]byte(series + " " + strconv.FormatFloat(p.value, 'f', -1, 64) + "\n\n"))
}

// writeCounter renders an unlabeled counter metric.
func writeCounter(w http.ResponseWriter, name, help string, value float64) {
	promMetric{name: name, help: help, metricType: "counter", value: value}.write(w)
}

// writeGauge renders an unlabeled gauge metric.
func writeGauge(w http.ResponseWriter, name, help string, value float64) {
	promMetric{name: name, help: help, metricType: "gauge", value: value}.write(w)
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

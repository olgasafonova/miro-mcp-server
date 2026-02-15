package desirepath

import (
	"log/slog"
	"sync"
	"time"
)

// Logger records desire path normalization events in a thread-safe ring buffer.
// Mirrors the audit.MemoryLogger pattern.
type Logger struct {
	mu       sync.RWMutex
	events   []Event
	maxSize  int
	writePos int
	count    int
	config   Config
	slogger  *slog.Logger
}

// NewLogger creates a desire path logger with the given configuration.
func NewLogger(config Config, logger *slog.Logger) *Logger {
	maxSize := config.MaxEvents
	if maxSize <= 0 {
		maxSize = 500
	}
	return &Logger{
		events:  make([]Event, maxSize),
		maxSize: maxSize,
		config:  config,
		slogger: logger,
	}
}

// Log records a normalization event.
func (l *Logger) Log(event Event) {
	if !l.config.Enabled {
		return
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Emit to slog for stderr visibility
	if l.slogger != nil {
		l.slogger.Info("Desire path normalization",
			"tool", event.Tool,
			"param", event.Parameter,
			"rule", event.Rule,
			"raw", event.RawValue,
			"normalized", event.NormalizedTo,
		)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.events[l.writePos] = event
	l.writePos = (l.writePos + 1) % l.maxSize
	if l.count < l.maxSize {
		l.count++
	}
}

// QueryOptions specifies filters for querying desire path events.
type QueryOptions struct {
	Tool  string    // Filter by tool name
	Rule  string    // Filter by normalizer rule
	Since time.Time // Events after this time
	Limit int       // Max events to return (0 = all)
}

// Query retrieves events matching the specified criteria.
func (l *Logger) Query(opts QueryOptions) []Event {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var matches []Event
	for i := 0; i < l.count; i++ {
		idx := (l.writePos - l.count + i + l.maxSize) % l.maxSize
		event := l.events[idx]

		if opts.Tool != "" && event.Tool != opts.Tool {
			continue
		}
		if opts.Rule != "" && event.Rule != opts.Rule {
			continue
		}
		if !opts.Since.IsZero() && event.Timestamp.Before(opts.Since) {
			continue
		}

		matches = append(matches, event)
	}

	// Reverse to most-recent-first
	for i, j := 0, len(matches)-1; i < j; i, j = i+1, j-1 {
		matches[i], matches[j] = matches[j], matches[i]
	}

	if opts.Limit > 0 && len(matches) > opts.Limit {
		matches = matches[:opts.Limit]
	}

	return matches
}

// Report generates a grouped summary of all recorded normalizations.
type Report struct {
	TotalEvents int              `json:"total_events"`
	ByRule      map[string]int   `json:"by_rule"`
	ByTool      map[string]int   `json:"by_tool"`
	ByParam     map[string]int   `json:"by_param"`
	TopPatterns []PatternSummary `json:"top_patterns"`
	RecentOnes  []Event          `json:"recent,omitempty"`
}

// PatternSummary groups identical normalization patterns.
type PatternSummary struct {
	Rule      string `json:"rule"`
	Tool      string `json:"tool"`
	Parameter string `json:"parameter"`
	Example   string `json:"example"`
	Count     int    `json:"count"`
}

// Report generates a summary of all recorded desire path events.
func (l *Logger) Report() Report {
	l.mu.RLock()
	defer l.mu.RUnlock()

	report := Report{
		TotalEvents: l.count,
		ByRule:      make(map[string]int),
		ByTool:      make(map[string]int),
		ByParam:     make(map[string]int),
	}

	// Pattern key: rule|tool|param
	type patternKey struct {
		rule, tool, param string
	}
	patternCounts := make(map[patternKey]*PatternSummary)

	for i := 0; i < l.count; i++ {
		idx := (l.writePos - l.count + i + l.maxSize) % l.maxSize
		event := l.events[idx]

		report.ByRule[event.Rule]++
		report.ByTool[event.Tool]++
		report.ByParam[event.Parameter]++

		key := patternKey{event.Rule, event.Tool, event.Parameter}
		if ps, ok := patternCounts[key]; ok {
			ps.Count++
		} else {
			patternCounts[key] = &PatternSummary{
				Rule:      event.Rule,
				Tool:      event.Tool,
				Parameter: event.Parameter,
				Example:   event.RawValue + " -> " + event.NormalizedTo,
				Count:     1,
			}
		}
	}

	// Sort patterns by count (descending), take top 10
	patterns := make([]PatternSummary, 0, len(patternCounts))
	for _, ps := range patternCounts {
		patterns = append(patterns, *ps)
	}
	// Simple insertion sort (small N)
	for i := 1; i < len(patterns); i++ {
		for j := i; j > 0 && patterns[j].Count > patterns[j-1].Count; j-- {
			patterns[j], patterns[j-1] = patterns[j-1], patterns[j]
		}
	}
	if len(patterns) > 10 {
		patterns = patterns[:10]
	}
	report.TopPatterns = patterns

	// Add 5 most recent events
	recent := make([]Event, 0, 5)
	for i := l.count - 1; i >= 0 && len(recent) < 5; i-- {
		idx := (l.writePos - l.count + i + l.maxSize) % l.maxSize
		recent = append(recent, l.events[idx])
	}
	report.RecentOnes = recent

	return report
}

// Count returns the total number of events recorded.
func (l *Logger) Count() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.count
}

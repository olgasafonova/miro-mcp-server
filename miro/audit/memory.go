package audit

import (
	"context"
	"sync"
	"time"
)

// MemoryLogger is an in-memory audit logger using a ring buffer.
// Useful for development, testing, and short-lived sessions.
type MemoryLogger struct {
	mu       sync.RWMutex
	events   []Event
	maxSize  int
	writePos int
	count    int
	config   Config
}

// NewMemoryLogger creates a new in-memory audit logger.
// The maxSize parameter limits the number of events stored (ring buffer).
func NewMemoryLogger(maxSize int, config Config) *MemoryLogger {
	if maxSize <= 0 {
		maxSize = 1000
	}
	return &MemoryLogger{
		events:  make([]Event, maxSize),
		maxSize: maxSize,
		config:  config,
	}
}

// Log records an audit event to the ring buffer.
func (l *MemoryLogger) Log(ctx context.Context, event Event) error {
	if !l.config.Enabled {
		return nil
	}

	// Sanitize input if configured
	if l.config.SanitizeInput && event.Input != nil {
		event.Input = SanitizeInput(event.Input)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.events[l.writePos] = event
	l.writePos = (l.writePos + 1) % l.maxSize
	if l.count < l.maxSize {
		l.count++
	}

	return nil
}

// Query retrieves audit events matching the specified criteria.
func (l *MemoryLogger) Query(ctx context.Context, opts QueryOptions) (*QueryResult, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	matches := l.collectMatches(opts)
	reverseEvents(matches) // most recent first

	total := len(matches)
	page, hasMore := paginate(matches, opts.Offset, opts.Limit)

	return &QueryResult{
		Events:  page,
		Total:   total,
		HasMore: hasMore,
	}, nil
}

// collectMatches walks the ring buffer in oldest-to-newest order and returns
// the events that satisfy opts. Caller must hold the read lock.
func (l *MemoryLogger) collectMatches(opts QueryOptions) []Event {
	var matches []Event
	for i := 0; i < l.count; i++ {
		idx := (l.writePos - l.count + i + l.maxSize) % l.maxSize
		event := l.events[idx]
		if matchesQuery(event, opts) {
			matches = append(matches, event)
		}
	}
	return matches
}

// reverseEvents reverses events in place.
func reverseEvents(events []Event) {
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}
}

// paginate applies offset and limit to events. Returns the page slice and
// whether more matches exist beyond the page.
func paginate(events []Event, offset, limit int) ([]Event, bool) {
	if offset > 0 {
		if offset >= len(events) {
			return nil, false
		}
		events = events[offset:]
	}
	if limit > 0 && len(events) > limit {
		return events[:limit], true
	}
	return events, false
}

// Flush is a no-op for MemoryLogger (events are written synchronously).
func (l *MemoryLogger) Flush(ctx context.Context) error {
	return nil
}

// Close is a no-op for MemoryLogger.
func (l *MemoryLogger) Close() error {
	return nil
}

// GetAllEvents returns all events in the buffer (for testing).
func (l *MemoryLogger) GetAllEvents() []Event {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]Event, l.count)
	for i := 0; i < l.count; i++ {
		idx := (l.writePos - l.count + i + l.maxSize) % l.maxSize
		result[i] = l.events[idx]
	}
	return result
}

// Clear removes all events from the buffer (for testing).
func (l *MemoryLogger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.events = make([]Event, l.maxSize)
	l.writePos = 0
	l.count = 0
}

// matchesQuery checks if an event matches the query options.
func matchesQuery(event Event, opts QueryOptions) bool {
	return matchesTimeRange(event, opts) && matchesFieldFilters(event, opts)
}

// matchesTimeRange checks the Since / Until bounds.
func matchesTimeRange(event Event, opts QueryOptions) bool {
	if !opts.Since.IsZero() && event.Timestamp.Before(opts.Since) {
		return false
	}
	if !opts.Until.IsZero() && event.Timestamp.After(opts.Until) {
		return false
	}
	return true
}

// matchesFieldFilters checks the exact-match field filters (tool, method,
// user, board, action, success).
func matchesFieldFilters(event Event, opts QueryOptions) bool {
	if opts.Tool != "" && event.Tool != opts.Tool {
		return false
	}
	if opts.Method != "" && event.Method != opts.Method {
		return false
	}
	if opts.UserID != "" && event.UserID != opts.UserID {
		return false
	}
	if opts.BoardID != "" && event.BoardID != opts.BoardID {
		return false
	}
	if opts.Action != "" && event.Action != opts.Action {
		return false
	}
	if opts.Success != nil && event.Success != *opts.Success {
		return false
	}
	return true
}

// Stats returns statistics about the audit log.
type Stats struct {
	TotalEvents   int            `json:"total_events"`
	SuccessCount  int            `json:"success_count"`
	ErrorCount    int            `json:"error_count"`
	ByTool        map[string]int `json:"by_tool"`
	ByAction      map[Action]int `json:"by_action"`
	OldestEvent   time.Time      `json:"oldest_event,omitempty"`
	NewestEvent   time.Time      `json:"newest_event,omitempty"`
	AvgDurationMs float64        `json:"avg_duration_ms"`
}

// GetStats returns statistics about the logged events.
func (l *MemoryLogger) GetStats() Stats {
	l.mu.RLock()
	defer l.mu.RUnlock()

	stats := Stats{
		TotalEvents: l.count,
		ByTool:      make(map[string]int),
		ByAction:    make(map[Action]int),
	}

	var totalDuration int64
	for i := 0; i < l.count; i++ {
		idx := (l.writePos - l.count + i + l.maxSize) % l.maxSize
		event := l.events[idx]
		accumulateEventIntoStats(&stats, event)
		totalDuration += event.DurationMs
	}

	if l.count > 0 {
		stats.AvgDurationMs = float64(totalDuration) / float64(l.count)
	}

	return stats
}

// accumulateEventIntoStats folds a single event into the running Stats:
// success/error tally, by-tool / by-action counts, and oldest/newest bounds.
func accumulateEventIntoStats(stats *Stats, event Event) {
	if event.Success {
		stats.SuccessCount++
	} else {
		stats.ErrorCount++
	}
	stats.ByTool[event.Tool]++
	stats.ByAction[event.Action]++
	updateTimestampRange(stats, event.Timestamp)
}

// updateTimestampRange expands the oldest/newest window to include ts.
func updateTimestampRange(stats *Stats, ts time.Time) {
	if stats.OldestEvent.IsZero() || ts.Before(stats.OldestEvent) {
		stats.OldestEvent = ts
	}
	if stats.NewestEvent.IsZero() || ts.After(stats.NewestEvent) {
		stats.NewestEvent = ts
	}
}

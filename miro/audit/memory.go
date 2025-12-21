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

	// Collect matching events
	var matches []Event
	for i := 0; i < l.count; i++ {
		// Read from oldest to newest
		idx := (l.writePos - l.count + i + l.maxSize) % l.maxSize
		event := l.events[idx]

		if matchesQuery(event, opts) {
			matches = append(matches, event)
		}
	}

	// Sort by timestamp descending (most recent first)
	for i, j := 0, len(matches)-1; i < j; i, j = i+1, j-1 {
		matches[i], matches[j] = matches[j], matches[i]
	}

	total := len(matches)

	// Apply offset
	if opts.Offset > 0 {
		if opts.Offset >= len(matches) {
			matches = nil
		} else {
			matches = matches[opts.Offset:]
		}
	}

	// Apply limit
	hasMore := false
	if opts.Limit > 0 && len(matches) > opts.Limit {
		matches = matches[:opts.Limit]
		hasMore = true
	}

	return &QueryResult{
		Events:  matches,
		Total:   total,
		HasMore: hasMore,
	}, nil
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
	// Time range
	if !opts.Since.IsZero() && event.Timestamp.Before(opts.Since) {
		return false
	}
	if !opts.Until.IsZero() && event.Timestamp.After(opts.Until) {
		return false
	}

	// Field filters
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

		if event.Success {
			stats.SuccessCount++
		} else {
			stats.ErrorCount++
		}

		stats.ByTool[event.Tool]++
		stats.ByAction[event.Action]++
		totalDuration += event.DurationMs

		if stats.OldestEvent.IsZero() || event.Timestamp.Before(stats.OldestEvent) {
			stats.OldestEvent = event.Timestamp
		}
		if stats.NewestEvent.IsZero() || event.Timestamp.After(stats.NewestEvent) {
			stats.NewestEvent = event.Timestamp
		}
	}

	if l.count > 0 {
		stats.AvgDurationMs = float64(totalDuration) / float64(l.count)
	}

	return stats
}

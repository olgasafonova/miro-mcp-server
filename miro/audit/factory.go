package audit

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// NewLogger creates an audit logger based on the provided configuration.
// If config.Path is empty, returns an in-memory logger.
// Otherwise, returns a file-based logger.
func NewLogger(config Config) (Logger, error) {
	if !config.Enabled {
		return NewNoopLogger(), nil
	}

	if config.Path == "" {
		return NewMemoryLogger(1000, config), nil
	}

	return NewFileLogger(config)
}

// envBoolean parses a boolean-style env value ("true" / "1"). Empty input
// leaves dst untouched.
func envBoolean(name string, dst *bool) {
	val := os.Getenv(name)
	if val == "" {
		return
	}
	*dst = strings.ToLower(val) == "true" || val == "1"
}

// envString assigns a non-empty env value to dst.
func envString(name string, dst *string) {
	if val := os.Getenv(name); val != "" {
		*dst = val
	}
}

// envRetentionDays parses MIRO_AUDIT_RETENTION as a day-suffixed duration.
func envRetentionDays(name string, dst *int) {
	val := os.Getenv(name)
	if val == "" {
		return
	}
	if days := parseDuration(val); days > 0 {
		*dst = days
	}
}

// envMaxSize parses MIRO_AUDIT_MAX_SIZE as a K/M/G-suffixed byte size.
func envMaxSize(name string, dst *int64) {
	val := os.Getenv(name)
	if val == "" {
		return
	}
	if size := parseSize(val); size > 0 {
		*dst = size
	}
}

// envNonNegativeInt parses a non-negative integer env value.
func envNonNegativeInt(name string, dst *int) {
	val := os.Getenv(name)
	if val == "" {
		return
	}
	if size, err := strconv.Atoi(val); err == nil && size >= 0 {
		*dst = size
	}
}

// LoadConfigFromEnv loads audit configuration from environment variables.
func LoadConfigFromEnv() Config {
	config := DefaultConfig()
	envBoolean("MIRO_AUDIT_ENABLED", &config.Enabled)
	envString("MIRO_AUDIT_PATH", &config.Path)
	envRetentionDays("MIRO_AUDIT_RETENTION", &config.RetentionDays)
	envMaxSize("MIRO_AUDIT_MAX_SIZE", &config.MaxSizeBytes)
	envNonNegativeInt("MIRO_AUDIT_BUFFER_SIZE", &config.BufferSize)
	envBoolean("MIRO_AUDIT_SANITIZE", &config.SanitizeInput)
	return config
}

// parseDuration parses a duration string like "30d", "7d", "90d".
func parseDuration(s string) int {
	s = strings.TrimSpace(strings.ToLower(s))
	if strings.HasSuffix(s, "d") {
		if days, err := strconv.Atoi(strings.TrimSuffix(s, "d")); err == nil {
			return days
		}
	}
	// Try parsing as integer days
	if days, err := strconv.Atoi(s); err == nil {
		return days
	}
	return 0
}

// parseSize parses a size string like "100M", "1G", "500K".
func parseSize(s string) int64 {
	s = strings.TrimSpace(strings.ToUpper(s))

	multiplier := int64(1)
	switch {
	case strings.HasSuffix(s, "K"):
		multiplier = 1024
		s = strings.TrimSuffix(s, "K")
	case strings.HasSuffix(s, "M"):
		multiplier = 1024 * 1024
		s = strings.TrimSuffix(s, "M")
	case strings.HasSuffix(s, "G"):
		multiplier = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "G")
	}

	if size, err := strconv.ParseInt(s, 10, 64); err == nil {
		return size * multiplier
	}
	return 0
}

// =============================================================================
// Event Builder
// =============================================================================

// EventBuilder helps construct audit events with a fluent API.
type EventBuilder struct {
	event Event
}

// NewEvent creates a new EventBuilder with required fields.
func NewEvent(tool, method string, action Action) *EventBuilder {
	return &EventBuilder{
		event: Event{
			ID:        uuid.New().String(),
			Timestamp: time.Now().UTC(),
			Tool:      tool,
			Method:    method,
			Action:    action,
		},
	}
}

// WithUser sets the user information.
func (b *EventBuilder) WithUser(userID, email string) *EventBuilder {
	b.event.UserID = userID
	b.event.UserEmail = email
	return b
}

// WithBoard sets the board ID.
func (b *EventBuilder) WithBoard(boardID string) *EventBuilder {
	b.event.BoardID = boardID
	return b
}

// WithItem sets the item ID and type.
func (b *EventBuilder) WithItem(itemID, itemType string) *EventBuilder {
	b.event.ItemID = itemID
	b.event.ItemType = itemType
	return b
}

// WithItemCount sets the number of items affected.
func (b *EventBuilder) WithItemCount(count int) *EventBuilder {
	b.event.ItemCount = count
	return b
}

// WithInput sets the input arguments.
func (b *EventBuilder) WithInput(input map[string]interface{}) *EventBuilder {
	b.event.Input = input
	return b
}

// WithDuration sets the operation duration.
func (b *EventBuilder) WithDuration(d time.Duration) *EventBuilder {
	b.event.DurationMs = d.Milliseconds()
	return b
}

// Success marks the event as successful.
func (b *EventBuilder) Success() *EventBuilder {
	b.event.Success = true
	return b
}

// Failure marks the event as failed with an error.
func (b *EventBuilder) Failure(err error) *EventBuilder {
	b.event.Success = false
	if err != nil {
		b.event.Error = err.Error()
	}
	return b
}

// Build returns the constructed event.
func (b *EventBuilder) Build() Event {
	return b.event
}

// =============================================================================
// Noop Logger
// =============================================================================

// NoopLogger is a no-operation logger that discards all events.
// Used when audit logging is disabled.
type NoopLogger struct{}

// NewNoopLogger creates a new no-op logger.
func NewNoopLogger() *NoopLogger {
	return &NoopLogger{}
}

// Log discards the event.
func (l *NoopLogger) Log(ctx context.Context, event Event) error {
	return nil
}

// Query returns an empty result.
func (l *NoopLogger) Query(ctx context.Context, opts QueryOptions) (*QueryResult, error) {
	return &QueryResult{Events: []Event{}}, nil
}

// Flush is a no-op.
func (l *NoopLogger) Flush(ctx context.Context) error {
	return nil
}

// Close is a no-op.
func (l *NoopLogger) Close() error {
	return nil
}

// Compile-time interface checks
var (
	_ Logger = (*MemoryLogger)(nil)
	_ Logger = (*FileLogger)(nil)
	_ Logger = (*NoopLogger)(nil)
)

// =============================================================================
// Action Detection
// =============================================================================

// actionPrefixes maps each Action to the method-name prefixes that signal it.
// Order matters only between rows; within a row, any prefix triggers the row's
// Action.
var actionPrefixes = []struct {
	action   Action
	prefixes []string
}{
	{ActionCreate, []string{"create", "bulk"}},
	{ActionRead, []string{"list", "get", "search", "find"}},
	{ActionUpdate, []string{"update"}},
	{ActionDelete, []string{"delete", "ungroup", "detach"}},
	{ActionExport, []string{"export"}},
	{ActionAuth, []string{"validate", "share"}},
}

// hasAnyPrefix reports whether s starts with any of the given prefixes.
func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// DetectAction infers the action type from the method name.
func DetectAction(method string) Action {
	method = strings.ToLower(method)
	for _, row := range actionPrefixes {
		if hasAnyPrefix(method, row.prefixes) {
			return row.action
		}
	}
	return ActionRead
}

// FormatDuration returns a human-readable duration.
func FormatDuration(ms int64) string {
	d := time.Duration(ms) * time.Millisecond

	if d < time.Second {
		return fmt.Sprintf("%dms", ms)
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}

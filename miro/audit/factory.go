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

// applyEnvSwitches overrides the boolean switches and the log path from
// MIRO_AUDIT_ENABLED, MIRO_AUDIT_SANITIZE ("true" / "1"), and
// MIRO_AUDIT_PATH. Unset variables leave the config untouched.
func applyEnvSwitches(config *Config) {
	if val := os.Getenv("MIRO_AUDIT_ENABLED"); val != "" {
		config.Enabled = envFlag(val).isTrue()
	}
	if val := os.Getenv("MIRO_AUDIT_SANITIZE"); val != "" {
		config.SanitizeInput = envFlag(val).isTrue()
	}
	if val := os.Getenv("MIRO_AUDIT_PATH"); val != "" {
		config.Path = val
	}
}

// envFlag is a raw boolean-style env value ("true" / "1").
type envFlag string

// isTrue reports whether the value spells a true boolean.
func (v envFlag) isTrue() bool {
	val := string(v)
	return strings.ToLower(val) == "true" || val == "1"
}

// applyEnvLimits overrides the numeric limits from MIRO_AUDIT_RETENTION
// (day-suffixed duration), MIRO_AUDIT_MAX_SIZE (K/M/G-suffixed byte size),
// and MIRO_AUDIT_BUFFER_SIZE (non-negative integer). Unset or invalid
// variables leave the config untouched.
func applyEnvLimits(config *Config) {
	if days := envDuration(os.Getenv("MIRO_AUDIT_RETENTION")).days(); days > 0 {
		config.RetentionDays = days
	}
	if size := envSize(os.Getenv("MIRO_AUDIT_MAX_SIZE")).bytes(); size > 0 {
		config.MaxSizeBytes = size
	}
	if size, err := strconv.Atoi(os.Getenv("MIRO_AUDIT_BUFFER_SIZE")); err == nil && size >= 0 {
		config.BufferSize = size
	}
}

// LoadConfigFromEnv loads audit configuration from environment variables.
func LoadConfigFromEnv() Config {
	config := DefaultConfig()
	applyEnvSwitches(&config)
	applyEnvLimits(&config)
	return config
}

// envDuration is a raw duration env value like "30d", "7d", "90d".
type envDuration string

// days parses the value as a day count, returning 0 when invalid.
func (v envDuration) days() int {
	s := strings.TrimSpace(strings.ToLower(string(v)))
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

// envSize is a raw size env value like "100M", "1G", "500K".
type envSize string

// bytes parses the value as a byte count, returning 0 when invalid.
func (v envSize) bytes() int64 {
	s := strings.TrimSpace(strings.ToUpper(string(v)))

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

// DetectAction infers the action type from the method name.
func DetectAction(method string) Action {
	method = strings.ToLower(method)
	for _, row := range actionPrefixes {
		for _, prefix := range row.prefixes {
			if strings.HasPrefix(method, prefix) {
				return row.action
			}
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

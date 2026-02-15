package desirepath

import (
	"log/slog"
	"os"
	"testing"
	"time"
)

// =============================================================================
// URL-to-ID Normalizer Tests
// =============================================================================

func TestURLToIDNormalizer(t *testing.T) {
	n := NewURLToIDNormalizer(MiroURLPatterns())

	tests := []struct {
		name      string
		param     string
		input     any
		wantVal   any
		wantRule  string
		wantMatch bool
	}{
		{
			name:      "full board URL",
			param:     "board_id",
			input:     "https://miro.com/app/board/uXjVN123=/",
			wantVal:   "uXjVN123=",
			wantRule:  "url_to_id",
			wantMatch: true,
		},
		{
			name:      "board URL without trailing slash",
			param:     "board_id",
			input:     "https://miro.com/app/board/uXjVN456=",
			wantVal:   "uXjVN456=",
			wantRule:  "url_to_id",
			wantMatch: true,
		},
		{
			name:      "board URL with query params",
			param:     "board_id",
			input:     "https://miro.com/app/board/uXjVN789=?moveToWidget=123",
			wantVal:   "uXjVN789=",
			wantRule:  "url_to_id",
			wantMatch: true,
		},
		{
			name:      "plain board ID unchanged",
			param:     "board_id",
			input:     "uXjVN123=",
			wantVal:   "uXjVN123=",
			wantMatch: false,
		},
		{
			name:      "item URL",
			param:     "item_id",
			input:     "https://miro.com/app/board/uXjVN123=/item/3458764529000000123",
			wantVal:   "3458764529000000123",
			wantRule:  "url_to_id",
			wantMatch: true,
		},
		{
			name:      "plain item ID unchanged",
			param:     "item_id",
			input:     "3458764529000000123",
			wantVal:   "3458764529000000123",
			wantMatch: false,
		},
		{
			name:      "non-string value ignored",
			param:     "board_id",
			input:     42,
			wantVal:   42,
			wantMatch: false,
		},
		{
			name:      "wrong param name ignored",
			param:     "other_param",
			input:     "https://miro.com/app/board/uXjVN123=/",
			wantVal:   "https://miro.com/app/board/uXjVN123=/",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, result := n.Normalize(tt.param, tt.input)
			if got != tt.wantVal {
				t.Errorf("value = %v, want %v", got, tt.wantVal)
			}
			if result.Changed != tt.wantMatch {
				t.Errorf("changed = %v, want %v", result.Changed, tt.wantMatch)
			}
			if tt.wantMatch && result.Rule != tt.wantRule {
				t.Errorf("rule = %q, want %q", result.Rule, tt.wantRule)
			}
		})
	}
}

// =============================================================================
// CamelCase-to-snake_case Tests
// =============================================================================

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"boardId", "board_id"},
		{"itemID", "item_id"},
		{"board_id", "board_id"},
		{"BoardID", "board_id"},
		{"dryRun", "dry_run"},
		{"x", "x"},
		{"", ""},
		{"createStickyNote", "create_sticky_note"},
		{"getHTTPResponse", "get_httpresponse"}, // consecutive uppercase collapses
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := camelToSnake(tt.input)
			if got != tt.want {
				t.Errorf("camelToSnake(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCamelToSnakeNormalizer_ConvertKey(t *testing.T) {
	n := &CamelToSnakeNormalizer{}

	tests := []struct {
		key         string
		wantKey     string
		wantChanged bool
	}{
		{"boardId", "board_id", true},
		{"board_id", "board_id", false},
		{"dryRun", "dry_run", true},
		{"limit", "limit", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			gotKey, gotChanged := n.ConvertKey(tt.key)
			if gotKey != tt.wantKey {
				t.Errorf("key = %q, want %q", gotKey, tt.wantKey)
			}
			if gotChanged != tt.wantChanged {
				t.Errorf("changed = %v, want %v", gotChanged, tt.wantChanged)
			}
		})
	}
}

// =============================================================================
// String-to-Numeric Tests
// =============================================================================

func TestStringToNumericNormalizer(t *testing.T) {
	n := NewStringToNumericNormalizer(nil) // Use defaults

	tests := []struct {
		name      string
		param     string
		input     any
		wantVal   any
		wantMatch bool
	}{
		{
			name:      "string int to number",
			param:     "limit",
			input:     "42",
			wantVal:   float64(42),
			wantMatch: true,
		},
		{
			name:      "string float to number",
			param:     "x",
			input:     "3.14",
			wantVal:   3.14,
			wantMatch: true,
		},
		{
			name:      "already a number",
			param:     "limit",
			input:     float64(42),
			wantVal:   float64(42),
			wantMatch: false,
		},
		{
			name:      "non-numeric param ignored",
			param:     "board_id",
			input:     "42",
			wantVal:   "42",
			wantMatch: false,
		},
		{
			name:      "non-numeric string ignored",
			param:     "limit",
			input:     "abc",
			wantVal:   "abc",
			wantMatch: false,
		},
		{
			name:      "string with spaces trimmed",
			param:     "offset",
			input:     " 10 ",
			wantVal:   float64(10),
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, result := n.Normalize(tt.param, tt.input)
			if got != tt.wantVal {
				t.Errorf("value = %v (%T), want %v (%T)", got, got, tt.wantVal, tt.wantVal)
			}
			if result.Changed != tt.wantMatch {
				t.Errorf("changed = %v, want %v", result.Changed, tt.wantMatch)
			}
		})
	}
}

// =============================================================================
// Whitespace Normalizer Tests
// =============================================================================

func TestWhitespaceNormalizer(t *testing.T) {
	n := &WhitespaceNormalizer{}

	tests := []struct {
		name      string
		input     any
		wantVal   any
		wantMatch bool
	}{
		{
			name:      "leading and trailing spaces",
			input:     "  uXjVN123  ",
			wantVal:   "uXjVN123",
			wantMatch: true,
		},
		{
			name:      "surrounding double quotes",
			input:     `"uXjVN123"`,
			wantVal:   "uXjVN123",
			wantMatch: true,
		},
		{
			name:      "surrounding single quotes",
			input:     `'uXjVN123'`,
			wantVal:   "uXjVN123",
			wantMatch: true,
		},
		{
			name:      "clean string unchanged",
			input:     "uXjVN123",
			wantVal:   "uXjVN123",
			wantMatch: false,
		},
		{
			name:      "non-string ignored",
			input:     42,
			wantVal:   42,
			wantMatch: false,
		},
		{
			name:      "spaces inside quotes",
			input:     `" uXjVN123 "`,
			wantVal:   "uXjVN123",
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, result := n.Normalize("board_id", tt.input)
			if got != tt.wantVal {
				t.Errorf("value = %v, want %v", got, tt.wantVal)
			}
			if result.Changed != tt.wantMatch {
				t.Errorf("changed = %v, want %v", result.Changed, tt.wantMatch)
			}
		})
	}
}

// =============================================================================
// Boolean Coercion Tests
// =============================================================================

func TestBooleanCoercionNormalizer(t *testing.T) {
	n := NewBooleanCoercionNormalizer(nil) // Use defaults

	tests := []struct {
		name      string
		param     string
		input     any
		wantVal   any
		wantMatch bool
	}{
		{
			name:      "string true",
			param:     "dry_run",
			input:     "true",
			wantVal:   true,
			wantMatch: true,
		},
		{
			name:      "string false",
			param:     "dry_run",
			input:     "false",
			wantVal:   false,
			wantMatch: true,
		},
		{
			name:      "string yes",
			param:     "dry_run",
			input:     "yes",
			wantVal:   true,
			wantMatch: true,
		},
		{
			name:      "string 1",
			param:     "dry_run",
			input:     "1",
			wantVal:   true,
			wantMatch: true,
		},
		{
			name:      "string 0",
			param:     "dry_run",
			input:     "0",
			wantVal:   false,
			wantMatch: true,
		},
		{
			name:      "already bool",
			param:     "dry_run",
			input:     true,
			wantVal:   true,
			wantMatch: false,
		},
		{
			name:      "non-boolean param ignored",
			param:     "board_id",
			input:     "true",
			wantVal:   "true",
			wantMatch: false,
		},
		{
			name:      "case insensitive",
			param:     "dry_run",
			input:     "TRUE",
			wantVal:   true,
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, result := n.Normalize(tt.param, tt.input)
			if got != tt.wantVal {
				t.Errorf("value = %v, want %v", got, tt.wantVal)
			}
			if result.Changed != tt.wantMatch {
				t.Errorf("changed = %v, want %v", result.Changed, tt.wantMatch)
			}
		})
	}
}

// =============================================================================
// Logger Tests
// =============================================================================

func testSlogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestLoggerBasic(t *testing.T) {
	l := NewLogger(Config{Enabled: true, MaxEvents: 10}, testSlogger())

	l.Log(Event{
		Tool:         "miro_get_board",
		Parameter:    "board_id",
		Rule:         "url_to_id",
		RawValue:     "https://miro.com/app/board/uXjVN123=/",
		NormalizedTo: "uXjVN123=",
	})

	if l.Count() != 1 {
		t.Fatalf("count = %d, want 1", l.Count())
	}

	events := l.Query(QueryOptions{})
	if len(events) != 1 {
		t.Fatalf("query returned %d events, want 1", len(events))
	}
	if events[0].Tool != "miro_get_board" {
		t.Errorf("tool = %q, want %q", events[0].Tool, "miro_get_board")
	}
}

func TestLoggerDisabled(t *testing.T) {
	l := NewLogger(Config{Enabled: false, MaxEvents: 10}, testSlogger())

	l.Log(Event{Tool: "test", Rule: "test"})

	if l.Count() != 0 {
		t.Errorf("disabled logger should not record events, got count = %d", l.Count())
	}
}

func TestLoggerRingBuffer(t *testing.T) {
	l := NewLogger(Config{Enabled: true, MaxEvents: 3}, testSlogger())

	// Write 5 events; only last 3 should survive
	for i := 0; i < 5; i++ {
		l.Log(Event{
			Tool:         "tool",
			Parameter:    "p",
			Rule:         "r",
			RawValue:     string(rune('A' + i)),
			NormalizedTo: string(rune('a' + i)),
		})
	}

	if l.Count() != 3 {
		t.Fatalf("count = %d, want 3", l.Count())
	}

	events := l.Query(QueryOptions{})
	if len(events) != 3 {
		t.Fatalf("query returned %d events, want 3", len(events))
	}

	// Most recent first
	if events[0].RawValue != "E" {
		t.Errorf("most recent event raw = %q, want %q", events[0].RawValue, "E")
	}
	if events[2].RawValue != "C" {
		t.Errorf("oldest event raw = %q, want %q", events[2].RawValue, "C")
	}
}

func TestLoggerQueryFilters(t *testing.T) {
	l := NewLogger(Config{Enabled: true, MaxEvents: 100}, testSlogger())

	now := time.Now().UTC()
	l.Log(Event{Tool: "tool_a", Rule: "url_to_id", Timestamp: now.Add(-2 * time.Minute)})
	l.Log(Event{Tool: "tool_b", Rule: "whitespace", Timestamp: now.Add(-1 * time.Minute)})
	l.Log(Event{Tool: "tool_a", Rule: "whitespace", Timestamp: now})

	// Filter by tool
	events := l.Query(QueryOptions{Tool: "tool_a"})
	if len(events) != 2 {
		t.Errorf("tool filter: got %d events, want 2", len(events))
	}

	// Filter by rule
	events = l.Query(QueryOptions{Rule: "whitespace"})
	if len(events) != 2 {
		t.Errorf("rule filter: got %d events, want 2", len(events))
	}

	// Filter by time
	events = l.Query(QueryOptions{Since: now.Add(-90 * time.Second)})
	if len(events) != 2 {
		t.Errorf("since filter: got %d events, want 2", len(events))
	}

	// Filter with limit
	events = l.Query(QueryOptions{Limit: 1})
	if len(events) != 1 {
		t.Errorf("limit filter: got %d events, want 1", len(events))
	}
}

func TestLoggerReport(t *testing.T) {
	l := NewLogger(Config{Enabled: true, MaxEvents: 100}, testSlogger())

	l.Log(Event{Tool: "tool_a", Parameter: "board_id", Rule: "url_to_id", RawValue: "url1", NormalizedTo: "id1"})
	l.Log(Event{Tool: "tool_a", Parameter: "board_id", Rule: "url_to_id", RawValue: "url2", NormalizedTo: "id2"})
	l.Log(Event{Tool: "tool_b", Parameter: "limit", Rule: "string_to_numeric", RawValue: "10", NormalizedTo: "10"})

	report := l.Report()

	if report.TotalEvents != 3 {
		t.Errorf("total = %d, want 3", report.TotalEvents)
	}
	if report.ByRule["url_to_id"] != 2 {
		t.Errorf("by_rule[url_to_id] = %d, want 2", report.ByRule["url_to_id"])
	}
	if report.ByTool["tool_a"] != 2 {
		t.Errorf("by_tool[tool_a] = %d, want 2", report.ByTool["tool_a"])
	}
	if report.ByParam["board_id"] != 2 {
		t.Errorf("by_param[board_id] = %d, want 2", report.ByParam["board_id"])
	}
	if len(report.TopPatterns) != 2 {
		t.Errorf("top_patterns count = %d, want 2", len(report.TopPatterns))
	}
	// Top pattern should be url_to_id (count 2)
	if report.TopPatterns[0].Rule != "url_to_id" {
		t.Errorf("top pattern rule = %q, want %q", report.TopPatterns[0].Rule, "url_to_id")
	}
	if report.TopPatterns[0].Count != 2 {
		t.Errorf("top pattern count = %d, want 2", report.TopPatterns[0].Count)
	}
}

// =============================================================================
// Config Tests
// =============================================================================

func TestDefaultConfig(t *testing.T) {
	c := DefaultConfig()
	if !c.Enabled {
		t.Error("default should be enabled")
	}
	if c.MaxEvents != 500 {
		t.Errorf("max events = %d, want 500", c.MaxEvents)
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("MIRO_DESIRE_PATHS", "false")
	t.Setenv("MIRO_DESIRE_PATHS_MAX_EVENTS", "1000")

	c := LoadConfigFromEnv()
	if c.Enabled {
		t.Error("should be disabled")
	}
	if c.MaxEvents != 1000 {
		t.Errorf("max events = %d, want 1000", c.MaxEvents)
	}
}

// =============================================================================
// Normalizer Name Tests
// =============================================================================

func TestNormalizerNames(t *testing.T) {
	normalizers := []Normalizer{
		NewURLToIDNormalizer(MiroURLPatterns()),
		&CamelToSnakeNormalizer{},
		NewStringToNumericNormalizer(nil),
		&WhitespaceNormalizer{},
		NewBooleanCoercionNormalizer(nil),
	}

	expectedNames := []string{"url_to_id", "camel_to_snake", "string_to_numeric", "whitespace", "boolean_coercion"}

	for i, n := range normalizers {
		if n.Name() != expectedNames[i] {
			t.Errorf("normalizer %d name = %q, want %q", i, n.Name(), expectedNames[i])
		}
	}
}

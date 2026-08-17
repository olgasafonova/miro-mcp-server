package audit

import (
	"os"
	"testing"
)

// newLoggerFromConfig calls NewLogger and registers cleanup, failing on error.
func newLoggerFromConfig(t *testing.T, config Config) Logger {
	t.Helper()
	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	t.Cleanup(func() { logger.Close() })
	return logger
}

func TestNewLogger_Memory(t *testing.T) {
	logger := newLoggerFromConfig(t, Config{Enabled: true, Path: ""}) // Empty path = memory logger

	// Should be memory logger
	if _, ok := logger.(*MemoryLogger); !ok {
		t.Error("expected MemoryLogger for empty path")
	}
}

func TestNewLogger_Disabled(t *testing.T) {
	logger := newLoggerFromConfig(t, Config{Enabled: false})

	// Should be noop logger
	if _, ok := logger.(*NoopLogger); !ok {
		t.Error("expected NoopLogger when disabled")
	}
}

func TestNewLogger_FileLoggerWithPath(t *testing.T) {
	config := Config{
		Enabled:      true,
		Path:         t.TempDir(),
		MaxSizeBytes: 10 * 1024 * 1024,
	}
	logger := newLoggerFromConfig(t, config)

	// Should be file logger
	if _, ok := logger.(*FileLogger); !ok {
		t.Error("expected FileLogger for non-empty path")
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	// Set test env vars
	os.Setenv("MIRO_AUDIT_ENABLED", "true")
	os.Setenv("MIRO_AUDIT_PATH", "/tmp/audit")
	os.Setenv("MIRO_AUDIT_RETENTION", "7d")
	os.Setenv("MIRO_AUDIT_MAX_SIZE", "50M")
	os.Setenv("MIRO_AUDIT_BUFFER_SIZE", "50")
	defer func() {
		os.Unsetenv("MIRO_AUDIT_ENABLED")
		os.Unsetenv("MIRO_AUDIT_PATH")
		os.Unsetenv("MIRO_AUDIT_RETENTION")
		os.Unsetenv("MIRO_AUDIT_MAX_SIZE")
		os.Unsetenv("MIRO_AUDIT_BUFFER_SIZE")
	}()

	config := LoadConfigFromEnv()

	if !config.Enabled {
		t.Error("expected Enabled=true")
	}
	if config.Path != "/tmp/audit" {
		t.Errorf("expected Path=/tmp/audit, got %s", config.Path)
	}
	if config.RetentionDays != 7 {
		t.Errorf("expected RetentionDays=7, got %d", config.RetentionDays)
	}
	if config.MaxSizeBytes != 50*1024*1024 {
		t.Errorf("expected MaxSizeBytes=50MB, got %d", config.MaxSizeBytes)
	}
	if config.BufferSize != 50 {
		t.Errorf("expected BufferSize=50, got %d", config.BufferSize)
	}
}

func TestLoadConfigFromEnv_SanitizeOption(t *testing.T) {
	os.Setenv("MIRO_AUDIT_ENABLED", "1")
	os.Setenv("MIRO_AUDIT_SANITIZE", "true")
	defer func() {
		os.Unsetenv("MIRO_AUDIT_ENABLED")
		os.Unsetenv("MIRO_AUDIT_SANITIZE")
	}()

	config := LoadConfigFromEnv()

	if !config.Enabled {
		t.Error("expected Enabled=true")
	}
	if !config.SanitizeInput {
		t.Error("expected SanitizeInput=true")
	}
}

// =============================================================================
// parseDuration Tests
// =============================================================================

type parseCase[N int | int64] struct {
	input    string
	expected N
}

// runParseCases checks a parse function against each case as a subtest.
func runParseCases[N int | int64](t *testing.T, name string, parse func(string) N, tests []parseCase[N]) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if result := parse(tt.input); result != tt.expected {
				t.Errorf("%s(%q) = %d, want %d", name, tt.input, result, tt.expected)
			}
		})
	}
}

// runParseDurationCases checks envDuration.days against each case as a subtest.
func runParseDurationCases(t *testing.T, tests []parseCase[int]) {
	t.Helper()
	runParseCases(t, "envDuration.days", func(s string) int { return envDuration(s).days() }, tests)
}

func TestParseDuration_DaysSuffix(t *testing.T) {
	runParseDurationCases(t, []parseCase[int]{
		{"30d", 30},
		{"7d", 7},
		{"90d", 90},
		{"1d", 1},
		{"365d", 365},
		{"  30d  ", 30}, // whitespace
		{"30D", 30},     // uppercase
		{"  30D  ", 30}, // mixed
	})
}

func TestParseDuration_IntegerOnly(t *testing.T) {
	runParseDurationCases(t, []parseCase[int]{
		{"30", 30},
		{"7", 7},
		{"90", 90},
		{"  30  ", 30},
	})
}

func TestParseDuration_Invalid(t *testing.T) {
	runParseDurationCases(t, []parseCase[int]{
		{"", 0},
		{"abc", 0},
		{"abcd", 0},
		{"30x", 0},
	})
}

// =============================================================================
// parseSize Tests
// =============================================================================

// runParseSizeCases checks envSize.bytes against each case as a subtest.
func runParseSizeCases(t *testing.T, tests []parseCase[int64]) {
	t.Helper()
	runParseCases(t, "envSize.bytes", func(s string) int64 { return envSize(s).bytes() }, tests)
}

func TestParseSize_KilobytesSuffix(t *testing.T) {
	runParseSizeCases(t, []parseCase[int64]{
		{"1K", 1024},
		{"10K", 10 * 1024},
		{"100K", 100 * 1024},
		{"500k", 500 * 1024}, // lowercase
		{"  1K  ", 1024},     // whitespace
	})
}

func TestParseSize_MegabytesSuffix(t *testing.T) {
	runParseSizeCases(t, []parseCase[int64]{
		{"1M", 1024 * 1024},
		{"10M", 10 * 1024 * 1024},
		{"100M", 100 * 1024 * 1024},
		{"500m", 500 * 1024 * 1024}, // lowercase
	})
}

func TestParseSize_GigabytesSuffix(t *testing.T) {
	runParseSizeCases(t, []parseCase[int64]{
		{"1G", 1024 * 1024 * 1024},
		{"2G", 2 * 1024 * 1024 * 1024},
		{"1g", 1024 * 1024 * 1024}, // lowercase
	})
}

func TestParseSize_PlainBytes(t *testing.T) {
	runParseSizeCases(t, []parseCase[int64]{
		{"1024", 1024},
		{"1000000", 1000000},
		{"  1024  ", 1024}, // whitespace
	})
}

func TestParseSize_Invalid(t *testing.T) {
	runParseSizeCases(t, []parseCase[int64]{
		{"", 0},
		{"abc", 0},
		{"abcM", 0},
	})
}

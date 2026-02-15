// Package desirepath instruments agent behavior normalization for MCP servers.
//
// Agents (LLMs) frequently send slightly wrong input: full URLs instead of IDs,
// camelCase instead of snake_case, string numbers instead of integers. Rather than
// rejecting these, the desire path layer normalizes them silently and logs what
// happened as a signal for permanent schema fixes.
//
// Named after the urban planning concept: watch where people walk, then pave those paths.
package desirepath

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Event records a single normalization that was applied to agent input.
type Event struct {
	Timestamp    time.Time `json:"timestamp"`
	Tool         string    `json:"tool"`
	Parameter    string    `json:"parameter"`
	Rule         string    `json:"rule"`
	RawValue     string    `json:"raw_value"`
	NormalizedTo string    `json:"normalized_to"`
}

// Normalizer transforms raw parameter values into their expected form.
type Normalizer interface {
	// Name returns a human-readable identifier for this normalizer (e.g., "url_to_id").
	Name() string

	// Normalize inspects a raw value and optionally transforms it.
	// Returns the (possibly modified) value and a result describing what changed.
	Normalize(paramName string, rawValue any) (any, NormalizationResult)
}

// NormalizationResult describes what a normalizer did to a value.
type NormalizationResult struct {
	Changed  bool   // Whether the value was modified
	Rule     string // Normalizer rule name that fired
	Original string // Original value (as string)
	New      string // New value (as string)
}

// Config controls the desire path instrumentation behavior.
type Config struct {
	Enabled   bool // Whether normalization and logging are active
	MaxEvents int  // Ring buffer capacity (default 500)
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:   true,
		MaxEvents: 500,
	}
}

// LoadConfigFromEnv loads desire path configuration from environment variables.
// Follows the same pattern as audit.LoadConfigFromEnv.
func LoadConfigFromEnv() Config {
	config := DefaultConfig()

	if val := os.Getenv("MIRO_DESIRE_PATHS"); val != "" {
		config.Enabled = strings.EqualFold(val, "true") || val == "1"
	}

	if val := os.Getenv("MIRO_DESIRE_PATHS_MAX_EVENTS"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			config.MaxEvents = n
		}
	}

	return config
}

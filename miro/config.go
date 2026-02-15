package miro

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Configuration validation constants.
const (
	// MinTimeout is the minimum allowed request timeout.
	MinTimeout = 5 * time.Second

	// MaxTimeout is the maximum allowed request timeout.
	MaxTimeout = 5 * time.Minute
)

// LoadConfigFromEnv creates a Config from environment variables.
// Required: MIRO_ACCESS_TOKEN
// Optional: MIRO_TIMEOUT, MIRO_USER_AGENT, MIRO_TEAM_ID
func LoadConfigFromEnv() (*Config, error) {
	token := os.Getenv("MIRO_ACCESS_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("MIRO_ACCESS_TOKEN environment variable is required. Get one at https://miro.com/app/settings/user-profile/apps — Still stuck? https://github.com/olgasafonova/miro-mcp-server/issues/new?template=bug_report.yml")
	}

	timeout := DefaultTimeout
	if t := os.Getenv("MIRO_TIMEOUT"); t != "" {
		if d, err := time.ParseDuration(t); err == nil {
			timeout = d
		}
	}

	userAgent := os.Getenv("MIRO_USER_AGENT")
	if userAgent == "" {
		userAgent = "miro-mcp-server/1.0"
	}

	// Try to get TeamID from env first, then from tokens file
	teamID := os.Getenv("MIRO_TEAM_ID")
	if teamID == "" {
		teamID = loadTeamIDFromTokensFile()
	}

	return &Config{
		AccessToken: token,
		TeamID:      teamID,
		Timeout:     timeout,
		UserAgent:   userAgent,
	}, nil
}

// loadTeamIDFromTokensFile attempts to read the team_id from the OAuth tokens file.
func loadTeamIDFromTokensFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	tokensPath := filepath.Join(home, ".miro", "tokens.json")
	data, err := os.ReadFile(tokensPath)
	if err != nil {
		return ""
	}

	var tokens struct {
		TeamID string `json:"team_id"`
	}
	if err := json.Unmarshal(data, &tokens); err != nil {
		return ""
	}

	return tokens.TeamID
}

// Validate checks if the Config is valid and applies defaults for optional fields.
// Returns an error describing the first validation failure found.
func (c *Config) Validate() error {
	if c == nil {
		return &ValidationError{
			Field:   "Config",
			Message: "configuration is nil",
		}
	}

	// Validate access token (required)
	if c.AccessToken == "" {
		return &ValidationError{
			Field:   "AccessToken",
			Message: "access token is required. Get one at https://miro.com/app/settings/user-profile/apps",
		}
	}

	// Basic token format validation (should look like a JWT or API token)
	if !isValidTokenFormat(c.AccessToken) {
		return &ValidationError{
			Field:   "AccessToken",
			Message: "access token format appears invalid. Expected JWT or API token format",
		}
	}

	// Validate timeout range
	if c.Timeout < 0 {
		return &ValidationError{
			Field:   "Timeout",
			Message: fmt.Sprintf("timeout cannot be negative: %v", c.Timeout),
		}
	}
	if c.Timeout == 0 {
		c.Timeout = DefaultTimeout
	} else if c.Timeout < MinTimeout {
		return &ValidationError{
			Field:   "Timeout",
			Message: fmt.Sprintf("timeout %v is below minimum %v", c.Timeout, MinTimeout),
		}
	} else if c.Timeout > MaxTimeout {
		return &ValidationError{
			Field:   "Timeout",
			Message: fmt.Sprintf("timeout %v exceeds maximum %v", c.Timeout, MaxTimeout),
		}
	}

	// Apply default user agent if not set
	if c.UserAgent == "" {
		c.UserAgent = "miro-mcp-server/1.0"
	}

	// TeamID is optional but validate format if provided
	if c.TeamID != "" && !isValidTeamID(c.TeamID) {
		return &ValidationError{
			Field:   "TeamID",
			Message: "team ID format appears invalid. Expected numeric string",
		}
	}

	return nil
}

// ValidateConfig checks if the configuration is valid.
// Deprecated: Use Config.Validate() method instead.
func ValidateConfig(cfg *Config) error {
	return cfg.Validate()
}

// isValidTokenFormat performs basic validation on the token format.
// Accepts JWT-like tokens (xxx.yyy.zzz) or Miro API tokens (eyJ... prefixed).
func isValidTokenFormat(token string) bool {
	token = strings.TrimSpace(token)
	if len(token) < 20 {
		return false
	}

	// JWT-like format: three base64 segments separated by dots
	if strings.Count(token, ".") == 2 {
		parts := strings.Split(token, ".")
		for _, part := range parts {
			if len(part) == 0 {
				return false
			}
		}
		return true
	}

	// Miro API tokens typically start with "eyJ" (base64 encoded JSON header)
	if strings.HasPrefix(token, "eyJ") {
		return true
	}

	// Accept any non-empty token with reasonable length for flexibility
	return len(token) >= 20 && len(token) <= 4096
}

// isValidTeamID validates the team ID format.
// Miro team IDs are numeric strings.
func isValidTeamID(teamID string) bool {
	teamID = strings.TrimSpace(teamID)
	if len(teamID) == 0 || len(teamID) > 50 {
		return false
	}

	// Team IDs are numeric
	for _, r := range teamID {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

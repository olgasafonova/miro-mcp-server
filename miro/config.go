package miro

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LoadConfigFromEnv creates a Config from environment variables.
// Required: MIRO_ACCESS_TOKEN
// Optional: MIRO_TIMEOUT, MIRO_USER_AGENT, MIRO_TEAM_ID
func LoadConfigFromEnv() (*Config, error) {
	token := os.Getenv("MIRO_ACCESS_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("MIRO_ACCESS_TOKEN environment variable is required. Get one at https://miro.com/app/settings/user-profile/apps")
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

// ValidateConfig checks if the configuration is valid.
func ValidateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if cfg.AccessToken == "" {
		return fmt.Errorf("access token is required")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeout
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "miro-mcp-server/1.0"
	}
	return nil
}

package miro

import (
	"fmt"
	"os"
	"time"
)

func init() {
	// Override the lookupEnv function with actual os.Getenv
	lookupEnv = os.Getenv
}

// LoadConfigFromEnv creates a Config from environment variables.
// Required: MIRO_ACCESS_TOKEN
// Optional: MIRO_TIMEOUT, MIRO_USER_AGENT
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

	return &Config{
		AccessToken: token,
		Timeout:     timeout,
		UserAgent:   userAgent,
	}, nil
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

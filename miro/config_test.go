package miro

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfigFromEnv(t *testing.T) {
	// Save original env and restore after test
	originalToken := os.Getenv("MIRO_ACCESS_TOKEN")
	originalTimeout := os.Getenv("MIRO_TIMEOUT")
	originalUserAgent := os.Getenv("MIRO_USER_AGENT")
	defer func() {
		os.Setenv("MIRO_ACCESS_TOKEN", originalToken)
		os.Setenv("MIRO_TIMEOUT", originalTimeout)
		os.Setenv("MIRO_USER_AGENT", originalUserAgent)
	}()

	t.Run("missing token returns error", func(t *testing.T) {
		os.Unsetenv("MIRO_ACCESS_TOKEN")
		_, err := LoadConfigFromEnv()
		if err == nil {
			t.Error("expected error when token is missing")
		}
	})

	t.Run("valid token returns config", func(t *testing.T) {
		os.Setenv("MIRO_ACCESS_TOKEN", "test-token")
		os.Unsetenv("MIRO_TIMEOUT")
		os.Unsetenv("MIRO_USER_AGENT")

		cfg, err := LoadConfigFromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.AccessToken != "test-token" {
			t.Errorf("AccessToken = %q, want %q", cfg.AccessToken, "test-token")
		}
		if cfg.Timeout != DefaultTimeout {
			t.Errorf("Timeout = %v, want %v", cfg.Timeout, DefaultTimeout)
		}
		if cfg.UserAgent != "miro-mcp-server/1.0" {
			t.Errorf("UserAgent = %q, want %q", cfg.UserAgent, "miro-mcp-server/1.0")
		}
	})

	t.Run("custom timeout", func(t *testing.T) {
		os.Setenv("MIRO_ACCESS_TOKEN", "test-token")
		os.Setenv("MIRO_TIMEOUT", "60s")

		cfg, err := LoadConfigFromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Timeout != 60*time.Second {
			t.Errorf("Timeout = %v, want %v", cfg.Timeout, 60*time.Second)
		}
	})

	t.Run("custom user agent", func(t *testing.T) {
		os.Setenv("MIRO_ACCESS_TOKEN", "test-token")
		os.Setenv("MIRO_USER_AGENT", "custom-agent/2.0")

		cfg, err := LoadConfigFromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.UserAgent != "custom-agent/2.0" {
			t.Errorf("UserAgent = %q, want %q", cfg.UserAgent, "custom-agent/2.0")
		}
	})

	t.Run("invalid timeout uses default", func(t *testing.T) {
		os.Setenv("MIRO_ACCESS_TOKEN", "test-token")
		os.Setenv("MIRO_TIMEOUT", "invalid")

		cfg, err := LoadConfigFromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Timeout != DefaultTimeout {
			t.Errorf("Timeout = %v, want %v", cfg.Timeout, DefaultTimeout)
		}
	})
}

func TestValidateConfig(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		err := ValidateConfig(nil)
		if err == nil {
			t.Error("expected error for nil config")
		}
	})

	t.Run("empty token", func(t *testing.T) {
		cfg := &Config{}
		err := ValidateConfig(cfg)
		if err == nil {
			t.Error("expected error for empty token")
		}
	})

	t.Run("valid config", func(t *testing.T) {
		cfg := &Config{
			AccessToken: "test-token",
			Timeout:     30 * time.Second,
			UserAgent:   "test-agent",
		}
		err := ValidateConfig(cfg)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("sets defaults for missing values", func(t *testing.T) {
		cfg := &Config{
			AccessToken: "test-token",
			Timeout:     0,
			UserAgent:   "",
		}
		err := ValidateConfig(cfg)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if cfg.Timeout != DefaultTimeout {
			t.Errorf("Timeout = %v, want %v", cfg.Timeout, DefaultTimeout)
		}
		if cfg.UserAgent != "miro-mcp-server/1.0" {
			t.Errorf("UserAgent = %q, want %q", cfg.UserAgent, "miro-mcp-server/1.0")
		}
	})
}

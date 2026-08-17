package miro

import (
	"os"
	"testing"
	"time"
)

// setConfigEnv pins the three config env vars for one test case. An empty
// value means "unset"; t.Setenv registers restoration either way.
func setConfigEnv(t *testing.T, token, timeout, userAgent string) {
	t.Helper()
	envs := map[string]string{
		"MIRO_ACCESS_TOKEN": token,
		"MIRO_TIMEOUT":      timeout,
		"MIRO_USER_AGENT":   userAgent,
	}
	for key, value := range envs {
		t.Setenv(key, value)
		if value == "" {
			os.Unsetenv(key)
		}
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		timeout     string
		userAgent   string
		wantErr     bool
		wantTimeout time.Duration
		wantUA      string
	}{
		{name: "missing token returns error", wantErr: true},
		{
			name: "valid token returns config", token: "test-token",
			wantTimeout: DefaultTimeout, wantUA: "miro-mcp-server/1.0",
		},
		{
			name: "custom timeout", token: "test-token", timeout: "60s",
			wantTimeout: 60 * time.Second, wantUA: "miro-mcp-server/1.0",
		},
		{
			name: "custom user agent", token: "test-token", userAgent: "custom-agent/2.0",
			wantTimeout: DefaultTimeout, wantUA: "custom-agent/2.0",
		},
		{
			name: "invalid timeout uses default", token: "test-token", timeout: "invalid",
			wantTimeout: DefaultTimeout, wantUA: "miro-mcp-server/1.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setConfigEnv(t, tt.token, tt.timeout, tt.userAgent)
			cfg, err := LoadConfigFromEnv()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error when token is missing")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.AccessToken != tt.token {
				t.Errorf("AccessToken = %q, want %q", cfg.AccessToken, tt.token)
			}
			if cfg.Timeout != tt.wantTimeout {
				t.Errorf("Timeout = %v, want %v", cfg.Timeout, tt.wantTimeout)
			}
			if cfg.UserAgent != tt.wantUA {
				t.Errorf("UserAgent = %q, want %q", cfg.UserAgent, tt.wantUA)
			}
		})
	}
}

// validTestJWT is a well-formed JWT accepted by the token format check.
const validTestJWT = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{name: "nil config", cfg: nil, wantErr: true},
		{name: "empty token", cfg: &Config{}, wantErr: true},
		{
			name: "valid config",
			cfg: &Config{
				AccessToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
				Timeout:     30 * time.Second,
				UserAgent:   "test-agent",
			},
		},
		{
			name:    "invalid token format",
			cfg:     &Config{AccessToken: "too-short"},
			wantErr: true,
		},
		{
			name:    "timeout below minimum",
			cfg:     &Config{AccessToken: validTestJWT, Timeout: 1 * time.Second},
			wantErr: true,
		},
		{
			name:    "timeout above maximum",
			cfg:     &Config{AccessToken: validTestJWT, Timeout: 10 * time.Minute},
			wantErr: true,
		},
		{
			name:    "invalid team ID format",
			cfg:     &Config{AccessToken: validTestJWT, TeamID: "invalid-team-id"},
			wantErr: true,
		},
		{
			name: "valid numeric team ID",
			cfg:  &Config{AccessToken: validTestJWT, TeamID: "3458764653228771705"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.cfg)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateConfig_SetsDefaults(t *testing.T) {
	cfg := &Config{
		AccessToken: validTestJWT,
		Timeout:     0,
		UserAgent:   "",
	}
	if err := ValidateConfig(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if cfg.Timeout != DefaultTimeout {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, DefaultTimeout)
	}
	if cfg.UserAgent != "miro-mcp-server/1.0" {
		t.Errorf("UserAgent = %q, want %q", cfg.UserAgent, "miro-mcp-server/1.0")
	}
}

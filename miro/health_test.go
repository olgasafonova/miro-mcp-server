package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// testLogger is defined in client_test.go

func TestHealthChecker_Check(t *testing.T) {
	t.Run("shallow check returns report", func(t *testing.T) {
		// Create mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{},
			})
		}))
		defer server.Close()

		config := &Config{
			AccessToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
			Timeout:     30 * time.Second,
		}
		client := NewClient(config, testLogger())
		client.baseURL = server.URL

		healthChecker := NewHealthChecker(client, "test-server", "1.0.0")

		report := healthChecker.Check(context.Background(), false)

		if report.Status != HealthStatusHealthy {
			t.Errorf("expected status %s, got %s", HealthStatusHealthy, report.Status)
		}
		if report.Server != "test-server" {
			t.Errorf("expected server 'test-server', got '%s'", report.Server)
		}
		if report.Version != "1.0.0" {
			t.Errorf("expected version '1.0.0', got '%s'", report.Version)
		}
		if report.Uptime == "" {
			t.Error("expected non-empty uptime")
		}

		// Config component should be present
		configComp, ok := report.Components["config"]
		if !ok {
			t.Error("expected config component in report")
		} else if configComp.Status != HealthStatusHealthy {
			t.Errorf("expected config status %s, got %s", HealthStatusHealthy, configComp.Status)
		}
	})

	t.Run("deep check tests API connectivity", func(t *testing.T) {
		// Create mock server that responds to token validation via /boards
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":   "board123",
						"name": "Test Board",
						"owner": map[string]interface{}{
							"id":   "owner123",
							"name": "Test User",
						},
					},
				},
			})
		}))
		defer server.Close()

		config := &Config{
			AccessToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
			Timeout:     30 * time.Second,
		}
		client := NewClient(config, testLogger())
		client.baseURL = server.URL

		healthChecker := NewHealthChecker(client, "test-server", "1.0.0")

		report := healthChecker.Check(context.Background(), true)

		if report.Status != HealthStatusHealthy {
			t.Errorf("expected status %s, got %s", HealthStatusHealthy, report.Status)
		}

		// API component should be present and healthy
		apiComp, ok := report.Components["miro_api"]
		if !ok {
			t.Error("expected miro_api component in report")
		} else {
			if apiComp.Status != HealthStatusHealthy {
				t.Errorf("expected miro_api status %s, got %s", HealthStatusHealthy, apiComp.Status)
			}
			if apiComp.Latency == "" {
				t.Error("expected non-empty latency for deep check")
			}
		}
	})

	t.Run("API failure results in unhealthy status", func(t *testing.T) {
		// Create mock server that returns error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  401,
				"message": "Unauthorized",
			})
		}))
		defer server.Close()

		config := &Config{
			AccessToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
			Timeout:     30 * time.Second,
		}
		client := NewClient(config, testLogger())
		client.baseURL = server.URL

		healthChecker := NewHealthChecker(client, "test-server", "1.0.0")

		report := healthChecker.Check(context.Background(), true)

		if report.Status != HealthStatusUnhealthy {
			t.Errorf("expected status %s, got %s", HealthStatusUnhealthy, report.Status)
		}

		apiComp := report.Components["miro_api"]
		if apiComp.Status != HealthStatusUnhealthy {
			t.Errorf("expected miro_api status %s, got %s", HealthStatusUnhealthy, apiComp.Status)
		}
	})

	t.Run("nil client results in unhealthy config", func(t *testing.T) {
		healthChecker := NewHealthChecker(nil, "test-server", "1.0.0")

		report := healthChecker.Check(context.Background(), false)

		configComp := report.Components["config"]
		if configComp.Status != HealthStatusUnhealthy {
			t.Errorf("expected config status %s, got %s", HealthStatusUnhealthy, configComp.Status)
		}
	})
}

func TestHealthReport_ToJSON(t *testing.T) {
	report := HealthReport{
		Status:    HealthStatusHealthy,
		Server:    "test-server",
		Version:   "1.0.0",
		Uptime:    "1h30m",
		StartedAt: time.Now().Add(-90 * time.Minute),
		Timestamp: time.Now(),
		Components: map[string]ComponentHealth{
			"config": {
				Status:      HealthStatusHealthy,
				Message:     "configuration valid",
				LastChecked: time.Now(),
			},
		},
	}

	jsonBytes, err := report.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if parsed["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got '%v'", parsed["status"])
	}
	if parsed["server"] != "test-server" {
		t.Errorf("expected server 'test-server', got '%v'", parsed["server"])
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "2m0s"},
		{30 * time.Minute, "30m0s"},
		{90 * time.Minute, "1h30m0s"},
		{25 * time.Hour, "1 day 1h"},
		{48 * time.Hour, "2 days"},
		{50 * time.Hour, "2 days 2h"},
	}

	for _, tt := range tests {
		t.Run(tt.duration.String(), func(t *testing.T) {
			got := formatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

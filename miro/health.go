package miro

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"time"
)

// HealthStatus represents the overall health of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// ComponentHealth represents the health of a single component
type ComponentHealth struct {
	Status      HealthStatus `json:"status"`
	Message     string       `json:"message,omitempty"`
	LastChecked time.Time    `json:"last_checked"`
	Latency     string       `json:"latency,omitempty"`
}

// HealthReport represents the overall health check response
type HealthReport struct {
	Status     HealthStatus               `json:"status"`
	Server     string                     `json:"server"`
	Version    string                     `json:"version"`
	Uptime     string                     `json:"uptime"`
	StartedAt  time.Time                  `json:"started_at"`
	Timestamp  time.Time                  `json:"timestamp"`
	Components map[string]ComponentHealth `json:"components"`
}

// HealthChecker provides health checking capabilities
type HealthChecker struct {
	client     *Client
	serverName string
	version    string
	startTime  time.Time

	mu             sync.RWMutex
	lastAPICheck   time.Time
	lastAPILatency time.Duration
	lastAPIStatus  HealthStatus
	lastAPIError   string
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(client *Client, serverName, version string) *HealthChecker {
	return &HealthChecker{
		client:        client,
		serverName:    serverName,
		version:       version,
		startTime:     time.Now(),
		lastAPIStatus: HealthStatusHealthy,
	}
}

// Check performs a health check and returns a report
// If deep is true, it also tests the Miro API connectivity
func (h *HealthChecker) Check(ctx context.Context, deep bool) HealthReport {
	report := HealthReport{
		Status:     HealthStatusHealthy,
		Server:     h.serverName,
		Version:    h.version,
		StartedAt:  h.startTime,
		Uptime:     formatDuration(time.Since(h.startTime)),
		Timestamp:  time.Now(),
		Components: make(map[string]ComponentHealth),
	}

	// Config component check
	report.Components["config"] = h.checkConfig()

	// API component check (deep check)
	if deep {
		report.Components["miro_api"] = h.checkAPI(ctx)
	} else {
		// Report last known status for shallow checks
		h.mu.RLock()
		if h.lastAPICheck.IsZero() {
			report.Components["miro_api"] = ComponentHealth{
				Status:  HealthStatusHealthy,
				Message: "not yet checked (use ?deep=true)",
			}
		} else {
			report.Components["miro_api"] = ComponentHealth{
				Status:      h.lastAPIStatus,
				Message:     h.lastAPIError,
				LastChecked: h.lastAPICheck,
				Latency:     h.lastAPILatency.String(),
			}
		}
		h.mu.RUnlock()
	}

	// Determine overall status
	for _, comp := range report.Components {
		if comp.Status == HealthStatusUnhealthy {
			report.Status = HealthStatusUnhealthy
			break
		}
		if comp.Status == HealthStatusDegraded && report.Status == HealthStatusHealthy {
			report.Status = HealthStatusDegraded
		}
	}

	return report
}

// checkConfig verifies the configuration is valid
func (h *HealthChecker) checkConfig() ComponentHealth {
	if h.client == nil || h.client.config == nil {
		return ComponentHealth{
			Status:      HealthStatusUnhealthy,
			Message:     "client or config is nil",
			LastChecked: time.Now(),
		}
	}

	if err := h.client.config.Validate(); err != nil {
		return ComponentHealth{
			Status:      HealthStatusUnhealthy,
			Message:     err.Error(),
			LastChecked: time.Now(),
		}
	}

	return ComponentHealth{
		Status:      HealthStatusHealthy,
		Message:     "configuration valid",
		LastChecked: time.Now(),
	}
}

// checkAPI tests connectivity to the Miro API
func (h *HealthChecker) checkAPI(ctx context.Context) ComponentHealth {
	start := time.Now()

	// Use ValidateToken as a lightweight API check
	_, err := h.client.ValidateToken(ctx)
	latency := time.Since(start)

	h.mu.Lock()
	h.lastAPICheck = time.Now()
	h.lastAPILatency = latency

	var comp ComponentHealth
	if err != nil {
		h.lastAPIStatus = HealthStatusUnhealthy
		h.lastAPIError = err.Error()
		comp = ComponentHealth{
			Status:      HealthStatusUnhealthy,
			Message:     err.Error(),
			LastChecked: time.Now(),
			Latency:     latency.String(),
		}
	} else {
		// Check latency thresholds
		if latency > 5*time.Second {
			h.lastAPIStatus = HealthStatusDegraded
			h.lastAPIError = "high latency"
			comp = ComponentHealth{
				Status:      HealthStatusDegraded,
				Message:     "API responding but with high latency",
				LastChecked: time.Now(),
				Latency:     latency.String(),
			}
		} else {
			h.lastAPIStatus = HealthStatusHealthy
			h.lastAPIError = ""
			comp = ComponentHealth{
				Status:      HealthStatusHealthy,
				Message:     "API responding normally",
				LastChecked: time.Now(),
				Latency:     latency.String(),
			}
		}
	}
	h.mu.Unlock()

	return comp
}

// ToJSON returns the health report as JSON bytes
func (r HealthReport) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// formatDuration formats a duration into a human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return d.Round(time.Second).String()
	}
	if d < time.Hour {
		return d.Round(time.Minute).String()
	}
	hours := int(d.Hours())
	if hours < 24 {
		return d.Round(time.Minute).String()
	}
	days := hours / 24
	remainingHours := hours % 24
	return formatDaysHours(days, remainingHours)
}

func formatDaysHours(days, hours int) string {
	if days == 1 {
		if hours == 0 {
			return "1 day"
		}
		return "1 day " + formatHours(hours)
	}
	if hours == 0 {
		return formatDaysOnly(days)
	}
	return formatDaysOnly(days) + " " + formatHours(hours)
}

func formatDaysOnly(days int) string {
	if days == 1 {
		return "1 day"
	}
	return formatInt(days) + " days"
}

func formatHours(hours int) string {
	if hours == 1 {
		return "1h"
	}
	return formatInt(hours) + "h"
}

func formatInt(n int) string {
	return strconv.Itoa(n)
}

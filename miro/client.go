// Package miro provides a client for the Miro REST API v2.
//
// This package implements a high-level client for interacting with Miro boards,
// including creating, reading, updating, and deleting board items like sticky notes,
// shapes, text, connectors, frames, cards, images, documents, and embeds.
//
// # Features
//
//   - Rate limiting with configurable concurrency
//   - Response caching with TTL
//   - Automatic retry with exponential backoff for rate limit errors
//   - Input validation for IDs and content
//   - Board name resolution (find boards by name, not just ID)
//
// # Usage
//
//	config, err := miro.LoadConfigFromEnv()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	client := miro.NewClient(config, logger)
//
//	// Validate token on startup
//	user, err := client.ValidateToken(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// List boards
//	result, err := client.ListBoards(ctx, miro.ListBoardsArgs{})
package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// =============================================================================
// Constants
// =============================================================================

// API and client configuration constants.
const (
	// BaseURL is the Miro REST API v2 base URL.
	BaseURL = "https://api.miro.com/v2"

	// ExperimentalBaseURL is the Miro REST API v2-experimental base URL.
	ExperimentalBaseURL = "https://api.miro.com/v2-experimental"

	// DefaultTimeout is the default HTTP request timeout.
	DefaultTimeout = 30 * time.Second

	// MaxConcurrentRequests limits parallel API calls to prevent rate limiting.
	MaxConcurrentRequests = 5

	// DefaultCacheTTL is the default cache time-to-live for board data.
	DefaultCacheTTL = 2 * time.Minute

	// MaxRetries is the maximum number of retry attempts for retriable errors.
	MaxRetries = 3

	// BaseRetryDelay is the initial delay for exponential backoff.
	BaseRetryDelay = 1 * time.Second

	// MaxRetryDelay caps the exponential backoff delay.
	MaxRetryDelay = 10 * time.Second
)

// =============================================================================
// Configuration
// =============================================================================

// Config holds Miro client configuration.
type Config struct {
	// AccessToken is the OAuth access token (required if not using TokenRefresher).
	// Get one at https://miro.com/app/settings/user-profile/apps
	AccessToken string

	// TeamID is the Miro team ID for board operations.
	// If set, ListBoards will filter by this team.
	// Can be read from MIRO_TEAM_ID env or from OAuth tokens file.
	TeamID string

	// Timeout for HTTP requests (default 30s).
	Timeout time.Duration

	// UserAgent identifies this client in API requests.
	UserAgent string
}

// TokenRefresher provides automatic token refresh for OAuth.
type TokenRefresher interface {
	// GetAccessToken returns a valid access token, refreshing if needed.
	GetAccessToken(ctx context.Context) (string, error)
}

// =============================================================================
// Client
// =============================================================================

// Client handles communication with the Miro API.
// It provides rate limiting, caching, and retry capabilities.
type Client struct {
	config     *Config
	httpClient *http.Client
	logger     *slog.Logger
	baseURL    string

	// semaphore limits concurrent API requests.
	semaphore chan struct{}

	// rateLimiter provides adaptive rate limiting based on API response headers.
	rateLimiter *AdaptiveRateLimiter

	// cache stores API responses with TTL and invalidation.
	cache       *Cache
	cacheConfig CacheConfig

	// circuitBreakers manages circuit breakers for API endpoints.
	circuitBreakers *CircuitBreakerRegistry

	// tokenRefresher provides automatic OAuth token refresh.
	// If nil, uses config.AccessToken (static token mode).
	tokenRefresher TokenRefresher
	tokenMu        sync.RWMutex
}

// UserInfo contains authenticated user information.
type UserInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// NewClient creates a new Miro API client with the given configuration.
// The client is safe for concurrent use by multiple goroutines.
func NewClient(config *Config, logger *slog.Logger) *Client {
	cacheConfig := DefaultCacheConfig()
	cbConfig := DefaultCircuitBreakerConfig()
	rlConfig := DefaultRateLimiterConfig()
	return &Client{
		config:  config,
		baseURL: BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		logger:          logger,
		semaphore:       make(chan struct{}, MaxConcurrentRequests),
		rateLimiter:     NewAdaptiveRateLimiter(rlConfig),
		cache:           NewCache(cacheConfig),
		cacheConfig:     cacheConfig,
		circuitBreakers: NewCircuitBreakerRegistry(cbConfig),
	}
}

// CacheStats returns cache performance statistics.
func (c *Client) CacheStats() CacheStats {
	return c.cache.Stats()
}

// InvalidateCache clears all cached data.
func (c *Client) InvalidateCache() {
	c.cache.Clear()
}

// CircuitBreakerStats returns statistics for all circuit breakers.
func (c *Client) CircuitBreakerStats() map[string]CircuitBreakerStats {
	return c.circuitBreakers.AllStats()
}

// ResetCircuitBreakers resets all circuit breakers to closed state.
func (c *Client) ResetCircuitBreakers() {
	c.circuitBreakers.Reset()
}

// RateLimiterStats returns rate limiter statistics.
func (c *Client) RateLimiterStats() RateLimiterStats {
	return c.rateLimiter.Stats()
}

// ResetRateLimiter resets the rate limiter to its initial state.
func (c *Client) ResetRateLimiter() {
	c.rateLimiter.Reset()
}

// WithTokenRefresher sets an OAuth token refresher for automatic token management.
// When set, the client will use the refresher to get valid access tokens instead
// of using the static AccessToken from config.
func (c *Client) WithTokenRefresher(refresher TokenRefresher) *Client {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	c.tokenRefresher = refresher
	return c
}

// getAccessToken returns the current access token, refreshing if needed.
func (c *Client) getAccessToken(ctx context.Context) (string, error) {
	c.tokenMu.RLock()
	refresher := c.tokenRefresher
	c.tokenMu.RUnlock()

	if refresher != nil {
		return refresher.GetAccessToken(ctx)
	}
	return c.config.AccessToken, nil
}

// =============================================================================
// Token Validation
// =============================================================================

// ValidateToken verifies the access token by calling /v2/boards?limit=1.
// Note: We use /boards instead of /users/me because Miro's /users/me endpoint
// has a bug returning "Invalid parameter type: long is required" for OAuth tokens.
// Call this on startup to fail fast with a clear error message.
func (c *Client) ValidateToken(ctx context.Context) (*UserInfo, error) {
	// Check cache first (valid for 5 minutes)
	if cached, ok := c.getCached("token:userinfo"); ok {
		return cached.(*UserInfo), nil
	}

	// Use /boards?limit=1 to validate token since /users/me is broken for OAuth
	respBody, err := c.request(ctx, http.MethodGet, "/boards?limit=1", nil)
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	// Parse response to extract team info if available
	var boardsResp struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Team struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"team"`
			Owner struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"owner"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &boardsResp); err != nil {
		return nil, fmt.Errorf("failed to parse boards response: %w", err)
	}

	// Create UserInfo from available data
	user := &UserInfo{
		ID:   "validated",
		Name: "Token Valid",
	}
	if len(boardsResp.Data) > 0 && boardsResp.Data[0].Owner.ID != "" {
		user.ID = boardsResp.Data[0].Owner.ID
		user.Name = boardsResp.Data[0].Owner.Name
	}

	// Cache for 5 minutes
	c.setCacheWithTTL("token:userinfo", user, 5*time.Minute)

	return user, nil
}

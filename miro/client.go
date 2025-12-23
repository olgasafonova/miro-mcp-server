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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
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

	// MaxRetries is the maximum number of retry attempts for rate-limited requests.
	MaxRetries = 3

	// BaseRetryDelay is the initial delay for exponential backoff.
	BaseRetryDelay = 1 * time.Second
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
		logger:             logger,
		semaphore:          make(chan struct{}, MaxConcurrentRequests),
		rateLimiter:        NewAdaptiveRateLimiter(rlConfig),
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
// Input Validation
// =============================================================================

var (
	// validIDPattern matches safe Miro IDs (alphanumeric, underscore, hyphen, equals)
	validIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_=\-]+$`)

	// maxContentLen is the maximum allowed content length
	maxContentLen = 10000

	// maxIDLen is the maximum allowed ID length
	maxIDLen = 100
)

// ValidateBoardID ensures board ID is safe and well-formed.
func ValidateBoardID(id string) error {
	if id == "" {
		return fmt.Errorf("board_id is required")
	}
	if len(id) > maxIDLen {
		return fmt.Errorf("board_id too long (max %d characters)", maxIDLen)
	}
	if !validIDPattern.MatchString(id) {
		return fmt.Errorf("board_id contains invalid characters")
	}
	return nil
}

// ValidateItemID ensures item ID is safe and well-formed.
func ValidateItemID(id string) error {
	if id == "" {
		return fmt.Errorf("item_id is required")
	}
	if len(id) > maxIDLen {
		return fmt.Errorf("item_id too long (max %d characters)", maxIDLen)
	}
	if !validIDPattern.MatchString(id) {
		return fmt.Errorf("item_id contains invalid characters")
	}
	return nil
}

// ValidateContent ensures content is within allowed limits.
func ValidateContent(content string) error {
	if len(content) > maxContentLen {
		return fmt.Errorf("content exceeds maximum length of %d characters", maxContentLen)
	}
	return nil
}

// =============================================================================
// HTTP Request Handling
// =============================================================================

// knownPathSegments are API path segments that should NOT be treated as IDs.
var knownPathSegments = map[string]bool{
	"boards": true, "items": true, "sticky_notes": true, "shapes": true,
	"text": true, "connectors": true, "frames": true, "cards": true,
	"images": true, "documents": true, "embeds": true, "tags": true,
	"groups": true, "members": true, "mindmaps": true, "nodes": true,
	"export": true, "jobs": true, "picture": true, "copy": true,
	"orgs": true, "users": true, "me": true, "teams": true,
}

// extractEndpoint extracts a normalized endpoint from a path for circuit breaker.
// For example: /boards/abc123/items/xyz -> /boards/{id}/items/{id}
func extractEndpoint(path string) string {
	parts := make([]string, 0)
	for _, part := range splitPath(path) {
		// Skip query strings
		if idx := indexOf(part, "?"); idx != -1 {
			part = part[:idx]
		}
		if part == "" {
			continue
		}
		// Check if this is a known path segment
		if knownPathSegments[part] {
			parts = append(parts, part)
		} else {
			// This is likely an ID - replace with placeholder
			// Avoid consecutive {id} entries
			if len(parts) == 0 || parts[len(parts)-1] != "{id}" {
				parts = append(parts, "{id}")
			}
		}
	}
	return "/" + joinPath(parts)
}

func splitPath(path string) []string {
	if path == "" {
		return nil
	}
	if path[0] == '/' {
		path = path[1:]
	}
	result := make([]string, 0)
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func indexOf(s string, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func joinPath(parts []string) string {
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += "/"
		}
		result += part
	}
	return result
}

// request makes an authenticated request to the Miro API.
func (c *Client) request(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	// Check circuit breaker
	endpoint := extractEndpoint(path)
	cb := c.circuitBreakers.Get(endpoint)
	if err := cb.Allow(); err != nil {
		c.logger.Warn("Circuit breaker blocked request",
			"endpoint", endpoint,
			"state", cb.State().String(),
		)
		return nil, fmt.Errorf("circuit breaker open for %s: %w", endpoint, err)
	}

	// Acquire semaphore slot (concurrency limiting)
	select {
	case c.semaphore <- struct{}{}:
		defer func() { <-c.semaphore }()
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled while waiting for rate limiter: %w", ctx.Err())
	}

	// Apply adaptive rate limiting based on previous response headers
	if delay, err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("context cancelled during rate limit wait: %w", err)
	} else if delay > 0 {
		c.logger.Debug("Adaptive rate limiter applied delay",
			"delay", delay,
			"state", c.rateLimiter.State(),
		)
	}

	// Get access token (may refresh if using OAuth)
	token, err := c.getAccessToken(ctx)
	if err != nil {
		cb.RecordFailure()
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Build request URL
	reqURL := c.baseURL + path

	// Prepare request body
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", c.config.UserAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		cb.RecordFailure()
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		cb.RecordFailure()
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Update rate limiter from response headers (do this for ALL responses)
	c.rateLimiter.UpdateFromResponse(resp)

	// Check for errors
	if resp.StatusCode >= 400 {
		apiErr := ParseAPIError(resp, respBody)
		// Record failure for server errors (5xx) - don't trip circuit for client errors
		if resp.StatusCode >= 500 {
			cb.RecordFailure()
		}
		return nil, apiErr
	}

	// Record success
	cb.RecordSuccess()

	c.logger.Debug("API request completed",
		"method", method,
		"path", path,
		"status", resp.StatusCode,
	)

	return respBody, nil
}

// requestExperimental makes a request to the v2-experimental API endpoints.
// Used for features that are not yet in the stable v2 API (e.g., mindmaps).
func (c *Client) requestExperimental(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	// Only swap to experimental if using production URL (preserve test server URLs)
	if c.baseURL == BaseURL {
		originalBaseURL := c.baseURL
		c.baseURL = ExperimentalBaseURL
		defer func() { c.baseURL = originalBaseURL }()
	}

	return c.request(ctx, method, path, body)
}


// =============================================================================
// Caching
// =============================================================================

// getCached retrieves a cached value if valid.
func (c *Client) getCached(key string) (interface{}, bool) {
	return c.cache.Get(key)
}

// setCache stores a value in the cache with default TTL (BoardTTL).
func (c *Client) setCache(key string, data interface{}) {
	c.cache.Set(key, data, c.cacheConfig.BoardTTL)
}

// setCacheWithTTL stores a value in the cache with custom TTL.
func (c *Client) setCacheWithTTL(key string, data interface{}, ttl time.Duration) {
	c.cache.Set(key, data, ttl)
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

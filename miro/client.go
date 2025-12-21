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
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/olgasafonova/miro-mcp-server/miro/webhooks"
)

// =============================================================================
// Constants
// =============================================================================

// API and client configuration constants.
const (
	// BaseURL is the Miro REST API v2 base URL.
	BaseURL = "https://api.miro.com/v2"

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

	// cache stores API responses with TTL.
	cache    sync.Map
	cacheTTL time.Duration

	// tokenRefresher provides automatic OAuth token refresh.
	// If nil, uses config.AccessToken (static token mode).
	tokenRefresher TokenRefresher
	tokenMu        sync.RWMutex

	// webhookMgr handles webhook subscription CRUD.
	webhookMgr *webhooks.Manager
	// webhookCallbackURL is the default callback URL for webhooks.
	webhookCallbackURL string
	// mu protects lazy-initialized fields.
	mu sync.Mutex
}

// cacheEntry holds cached data with expiration.
type cacheEntry struct {
	data      interface{}
	expiresAt time.Time
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
		cacheTTL:           DefaultCacheTTL,
		webhookCallbackURL: os.Getenv("MIRO_WEBHOOKS_CALLBACK_URL"),
	}
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

// request makes an authenticated request to the Miro API.
func (c *Client) request(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	// Acquire semaphore slot (rate limiting)
	select {
	case c.semaphore <- struct{}{}:
		defer func() { <-c.semaphore }()
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled while waiting for rate limiter: %w", ctx.Err())
	}

	// Get access token (may refresh if using OAuth)
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Build request URL
	reqURL := BaseURL + path

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
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.Message != "" {
			return nil, fmt.Errorf("API error [%d %s]: %s", resp.StatusCode, apiErr.Code, apiErr.Message)
		}
		return nil, fmt.Errorf("API error [%d]: %s", resp.StatusCode, string(respBody))
	}

	c.logger.Debug("API request completed",
		"method", method,
		"path", path,
		"status", resp.StatusCode,
	)

	return respBody, nil
}

// requestWithRetry wraps request with exponential backoff for rate limit errors (HTTP 429).
// It will retry up to MaxRetries times with exponentially increasing delays.
func (c *Client) requestWithRetry(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		respBody, err := c.request(ctx, method, path, body)
		if err == nil {
			return respBody, nil
		}

		// Check if rate limited (429)
		if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "rate limit") {
			delay := BaseRetryDelay * time.Duration(1<<uint(attempt)) // Exponential: 1s, 2s, 4s, 8s
			c.logger.Warn("Rate limited, retrying",
				"attempt", attempt+1,
				"max_retries", MaxRetries,
				"delay", delay,
				"path", path,
			)
			select {
			case <-time.After(delay):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		lastErr = err
		break // Don't retry non-rate-limit errors
	}

	return nil, lastErr
}

// =============================================================================
// Caching
// =============================================================================

// getCached retrieves a cached value if valid.
func (c *Client) getCached(key string) (interface{}, bool) {
	if entry, ok := c.cache.Load(key); ok {
		ce := entry.(*cacheEntry)
		if time.Now().Before(ce.expiresAt) {
			return ce.data, true
		}
		c.cache.Delete(key)
	}
	return nil, false
}

// setCache stores a value in the cache.
func (c *Client) setCache(key string, data interface{}) {
	c.cache.Store(key, &cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(c.cacheTTL),
	})
}

// =============================================================================
// Token Validation
// =============================================================================

// ValidateToken verifies the access token by calling /v2/users/me.
// Call this on startup to fail fast with a clear error message.
func (c *Client) ValidateToken(ctx context.Context) (*UserInfo, error) {
	// Check cache first (valid for 5 minutes)
	if cached, ok := c.getCached("token:userinfo"); ok {
		return cached.(*UserInfo), nil
	}

	respBody, err := c.request(ctx, http.MethodGet, "/users/me", nil)
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	var user UserInfo
	if err := json.Unmarshal(respBody, &user); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	// Cache for 5 minutes
	c.cache.Store("token:userinfo", &cacheEntry{
		data:      &user,
		expiresAt: time.Now().Add(5 * time.Minute),
	})

	return &user, nil
}

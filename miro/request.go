package miro

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// =============================================================================
// HTTP Request Handling
// =============================================================================

// request makes an authenticated request to the Miro API with retry support.
func (c *Client) request(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	if !c.config.IsConfigured() {
		return nil, fmt.Errorf("MIRO_ACCESS_TOKEN is not configured. Set the MIRO_ACCESS_TOKEN environment variable. Get one at https://miro.com/app/settings/user-profile/apps")
	}

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

	// Marshal body once for potential retries
	var bodyBytes []byte
	if body != nil {
		var marshalErr error
		bodyBytes, marshalErr = json.Marshal(body)
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", marshalErr)
		}
	}

	// Retry loop
	var lastErr error
	var lastStatusCode int
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate delay for retry
			retryAfter := GetRetryAfter(lastErr)
			delay := calculateRetryDelay(attempt-1, retryAfter)

			c.logger.Debug("Retrying request after transient error",
				"attempt", attempt,
				"delay", delay,
				"last_status", lastStatusCode,
				"path", path,
			)

			// Wait before retry, respecting context cancellation
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled during retry wait: %w", ctx.Err())
			}
		}

		// Build request URL
		reqURL := c.baseURL + path

		// Prepare request body (fresh reader for each attempt)
		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
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
			lastErr = err
			lastStatusCode = 0
			// Check if network error is retriable
			if isRetriableError(0, err) && attempt < MaxRetries {
				cb.RecordFailure()
				continue
			}
			cb.RecordFailure()
			return nil, fmt.Errorf("request failed: %w", err)
		}

		// Read response body
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			cb.RecordFailure()
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		// Update rate limiter from response headers (do this for ALL responses)
		c.rateLimiter.UpdateFromResponse(resp)

		lastStatusCode = resp.StatusCode

		// Check for errors
		if resp.StatusCode >= 400 {
			apiErr := ParseAPIError(resp, respBody)
			lastErr = apiErr

			// Check if retriable
			if isRetriableError(resp.StatusCode, nil) && attempt < MaxRetries {
				if resp.StatusCode >= 500 {
					cb.RecordFailure()
				}
				continue
			}

			// Not retriable or max retries exceeded
			if resp.StatusCode >= 500 {
				cb.RecordFailure()
			}
			return nil, apiErr
		}

		// Success - record and return
		cb.RecordSuccess()

		c.logger.Debug("API request completed",
			"method", method,
			"path", path,
			"status", resp.StatusCode,
			"attempts", attempt+1,
		)

		return respBody, nil
	}

	// Should not reach here, but return last error if we do
	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// requestMultipart makes a multipart form request to the Miro API.
// Used for file upload endpoints that require multipart/form-data.
func (c *Client) requestMultipart(ctx context.Context, method, path, contentType string, body io.Reader) ([]byte, error) {
	// Check circuit breaker
	endpoint := extractEndpoint(path)
	cb := c.circuitBreakers.Get(endpoint)
	if err := cb.Allow(); err != nil {
		return nil, fmt.Errorf("circuit breaker open for %s: %w", endpoint, err)
	}

	// Acquire semaphore slot
	select {
	case c.semaphore <- struct{}{}:
		defer func() { <-c.semaphore }()
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled while waiting for rate limiter: %w", ctx.Err())
	}

	// Apply adaptive rate limiting
	if delay, err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("context cancelled during rate limit wait: %w", err)
	} else if delay > 0 {
		c.logger.Debug("Adaptive rate limiter applied delay", "delay", delay)
	}

	// Get access token
	token, err := c.getAccessToken(ctx)
	if err != nil {
		cb.RecordFailure()
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	reqURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", c.config.UserAgent)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		cb.RecordFailure()
		return nil, fmt.Errorf("request failed: %w", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		cb.RecordFailure()
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	c.rateLimiter.UpdateFromResponse(resp)

	if resp.StatusCode >= 400 {
		apiErr := ParseAPIError(resp, respBody)
		if resp.StatusCode >= 500 {
			cb.RecordFailure()
		}
		return nil, apiErr
	}

	cb.RecordSuccess()
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
// Caching helpers
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

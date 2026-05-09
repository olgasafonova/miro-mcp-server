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

// requestState carries the per-call inputs that the retry loop reuses across
// attempts. Marshaled body bytes live here so retries don't re-serialize.
type requestState struct {
	method      string
	path        string
	bodyBytes   []byte
	hasJSONBody bool
	token       string
	cb          *CircuitBreaker
}

// request makes an authenticated request to the Miro API with retry support.
func (c *Client) request(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	if !c.config.IsConfigured() {
		return nil, fmt.Errorf("MIRO_ACCESS_TOKEN is not configured. Set the MIRO_ACCESS_TOKEN environment variable. Get one at https://miro.com/app/settings/user-profile/apps")
	}

	cb, err := c.checkCircuitBreaker(path)
	if err != nil {
		return nil, err
	}

	if err := c.acquireSlot(ctx); err != nil {
		return nil, err
	}
	defer c.releaseSlot()

	if err := c.waitForRateLimit(ctx); err != nil {
		return nil, err
	}

	token, err := c.getAccessToken(ctx)
	if err != nil {
		cb.RecordFailure()
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	bodyBytes, err := marshalRequestBody(body)
	if err != nil {
		return nil, err
	}

	return c.runRetryLoop(ctx, requestState{
		method:      method,
		path:        path,
		bodyBytes:   bodyBytes,
		hasJSONBody: body != nil,
		token:       token,
		cb:          cb,
	})
}

// checkCircuitBreaker returns the breaker for path's endpoint and rejects when
// it's open.
func (c *Client) checkCircuitBreaker(path string) (*CircuitBreaker, error) {
	endpoint := extractEndpoint(path)
	cb := c.circuitBreakers.Get(endpoint)
	if err := cb.Allow(); err != nil {
		c.logger.Warn("Circuit breaker blocked request",
			"endpoint", endpoint,
			"state", cb.State().String(),
		)
		return nil, fmt.Errorf("circuit breaker open for %s: %w", endpoint, err)
	}
	return cb, nil
}

// acquireSlot reserves a concurrency slot. Pair with releaseSlot via defer.
func (c *Client) acquireSlot(ctx context.Context) error {
	select {
	case c.semaphore <- struct{}{}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled while waiting for rate limiter: %w", ctx.Err())
	}
}

// releaseSlot returns the slot acquired by acquireSlot.
func (c *Client) releaseSlot() {
	<-c.semaphore
}

// waitForRateLimit applies the adaptive rate limiter's current delay.
func (c *Client) waitForRateLimit(ctx context.Context) error {
	delay, err := c.rateLimiter.Wait(ctx)
	if err != nil {
		return fmt.Errorf("context cancelled during rate limit wait: %w", err)
	}
	if delay > 0 {
		c.logger.Debug("Adaptive rate limiter applied delay",
			"delay", delay,
			"state", c.rateLimiter.State(),
		)
	}
	return nil
}

// marshalRequestBody serializes body to JSON if non-nil.
func marshalRequestBody(body interface{}) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	return bodyBytes, nil
}

// retryContext carries the per-iteration backoff inputs to waitBeforeRetry.
type retryContext struct {
	attempt        int
	lastErr        error
	lastStatusCode int
	path           string
}

// runRetryLoop performs up to MaxRetries+1 attempts of st against the Miro API,
// honoring backoff with optional Retry-After hints between attempts.
func (c *Client) runRetryLoop(ctx context.Context, st requestState) ([]byte, error) {
	var lastErr error
	var lastStatusCode int
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			if err := c.waitBeforeRetry(ctx, retryContext{
				attempt:        attempt,
				lastErr:        lastErr,
				lastStatusCode: lastStatusCode,
				path:           st.path,
			}); err != nil {
				return nil, err
			}
		}

		respBody, statusCode, retriable, err := c.tryOnce(ctx, st)
		if err == nil {
			c.logger.Debug("API request completed",
				"method", st.method,
				"path", st.path,
				"status", statusCode,
				"attempts", attempt+1,
			)
			return respBody, nil
		}

		lastErr = err
		lastStatusCode = statusCode
		if !retriable || attempt >= MaxRetries {
			return nil, err
		}
	}
	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// waitBeforeRetry sleeps with backoff, respecting context cancellation.
func (c *Client) waitBeforeRetry(ctx context.Context, rc retryContext) error {
	retryAfter := GetRetryAfter(rc.lastErr)
	delay := calculateRetryDelay(rc.attempt-1, retryAfter)

	c.logger.Debug("Retrying request after transient error",
		"attempt", rc.attempt,
		"delay", delay,
		"last_status", rc.lastStatusCode,
		"path", rc.path,
	)

	select {
	case <-time.After(delay):
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled during retry wait: %w", ctx.Err())
	}
}

// tryOnce performs a single HTTP attempt and reports the outcome. Network
// errors are returned wrapped as "request failed: %w" but remain unwrappable
// via errors.As, so GetRetryAfter still finds rate-limit hints. Caller decides
// retry based on retriable.
func (c *Client) tryOnce(ctx context.Context, st requestState) (respBody []byte, statusCode int, retriable bool, err error) {
	req, err := c.buildRequest(ctx, st)
	if err != nil {
		return nil, 0, false, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		st.cb.RecordFailure()
		return nil, 0, isRetriableError(0, err), fmt.Errorf("request failed: %w", err)
	}

	respBody, err = c.readAndCheckResponse(resp, st.cb)
	if err == nil {
		return respBody, resp.StatusCode, false, nil
	}
	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, isRetriableError(resp.StatusCode, nil), err
	}
	return nil, resp.StatusCode, false, err
}

// readAndCheckResponse reads resp.Body, updates the adaptive rate limiter,
// and converts >=400 status into a structured APIError. Records circuit
// breaker outcomes (failure on 5xx and read errors, success on 2xx-3xx).
func (c *Client) readAndCheckResponse(resp *http.Response, cb *CircuitBreaker) ([]byte, error) {
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

// buildRequest constructs a fresh HTTP request from st. Called once per attempt
// because bytes.Reader is not replayable across retries.
func (c *Client) buildRequest(ctx context.Context, st requestState) (*http.Request, error) {
	var reqBody io.Reader
	if st.bodyBytes != nil {
		reqBody = bytes.NewReader(st.bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, st.method, c.baseURL+st.path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	setMiroHeaders(req, st.token, c.config.UserAgent)
	if st.hasJSONBody {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// setMiroHeaders sets Authorization, User-Agent, and Accept on req.
func setMiroHeaders(req *http.Request, token, userAgent string) {
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
}

// multipartRequest carries the inputs to requestMultipart.
type multipartRequest struct {
	method      string
	path        string
	contentType string
	body        io.Reader
}

// requestMultipart makes a multipart form request to the Miro API. Multipart
// streams cannot be replayed cleanly, so this path does not retry.
func (c *Client) requestMultipart(ctx context.Context, mr multipartRequest) ([]byte, error) {
	cb, err := c.checkCircuitBreaker(mr.path)
	if err != nil {
		return nil, err
	}

	if err := c.acquireSlot(ctx); err != nil {
		return nil, err
	}
	defer c.releaseSlot()

	if err := c.waitForRateLimit(ctx); err != nil {
		return nil, err
	}

	token, err := c.getAccessToken(ctx)
	if err != nil {
		cb.RecordFailure()
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, mr.method, c.baseURL+mr.path, mr.body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	setMiroHeaders(req, token, c.config.UserAgent)
	req.Header.Set("Content-Type", mr.contentType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		cb.RecordFailure()
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return c.readAndCheckResponse(resp, cb)
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

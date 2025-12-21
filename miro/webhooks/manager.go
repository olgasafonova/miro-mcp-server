package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// Miro experimental webhooks API base path
	webhooksBasePath = "/v2-experimental/webhooks/board_subscriptions"
)

// Manager handles webhook subscription CRUD operations via Miro API.
type Manager struct {
	baseURL    string
	httpClient *http.Client
	getToken   func() string
}

// NewManager creates a new webhook manager.
// getToken is a function that returns the current access token.
func NewManager(baseURL string, getToken func() string) *Manager {
	return &Manager{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		getToken: getToken,
	}
}

// Create creates a new webhook subscription for a board.
func (m *Manager) Create(ctx context.Context, req CreateSubscriptionRequest) (*Subscription, error) {
	if req.BoardID == "" {
		return nil, fmt.Errorf("board_id is required")
	}
	if req.CallbackURL == "" {
		return nil, fmt.Errorf("callback_url is required")
	}

	// Default to enabled status
	if req.Status == "" {
		req.Status = StatusEnabled
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+webhooksBasePath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	m.setHeaders(httpReq)

	resp, err := m.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := m.checkResponse(resp); err != nil {
		return nil, err
	}

	var subscription Subscription
	if err := json.NewDecoder(resp.Body).Decode(&subscription); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &subscription, nil
}

// Get retrieves a webhook subscription by ID.
func (m *Manager) Get(ctx context.Context, subscriptionID string) (*Subscription, error) {
	if subscriptionID == "" {
		return nil, fmt.Errorf("subscription_id is required")
	}

	url := fmt.Sprintf("%s%s/%s", m.baseURL, webhooksBasePath, subscriptionID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	m.setHeaders(httpReq)

	resp, err := m.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := m.checkResponse(resp); err != nil {
		return nil, err
	}

	var subscription Subscription
	if err := json.NewDecoder(resp.Body).Decode(&subscription); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &subscription, nil
}

// Delete removes a webhook subscription.
func (m *Manager) Delete(ctx context.Context, subscriptionID string) error {
	if subscriptionID == "" {
		return fmt.Errorf("subscription_id is required")
	}

	url := fmt.Sprintf("%s%s/%s", m.baseURL, webhooksBasePath, subscriptionID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	m.setHeaders(httpReq)

	resp, err := m.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// 204 No Content is expected for successful deletion
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
		return nil
	}

	return m.checkResponse(resp)
}

// List retrieves all webhook subscriptions.
// Note: Miro's experimental API may not support listing all subscriptions.
// This implementation filters by board if boardID is provided.
func (m *Manager) List(ctx context.Context, boardID string) ([]Subscription, error) {
	url := m.baseURL + webhooksBasePath
	if boardID != "" {
		url += "?board_id=" + boardID
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	m.setHeaders(httpReq)

	resp, err := m.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := m.checkResponse(resp); err != nil {
		return nil, err
	}

	var listResp ListSubscriptionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		// Try parsing as array directly
		resp.Body.Close()
		httpReq, _ = http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		m.setHeaders(httpReq)
		resp, err = m.httpClient.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("retry request failed: %w", err)
		}
		defer resp.Body.Close()

		var subs []Subscription
		if err := json.NewDecoder(resp.Body).Decode(&subs); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return subs, nil
	}

	return listResp.Data, nil
}

// setHeaders sets the required headers for Miro API requests.
func (m *Manager) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+m.getToken())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}

// checkResponse checks the HTTP response for errors.
func (m *Manager) checkResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("unauthorized: check your access token")
	case http.StatusForbidden:
		return fmt.Errorf("forbidden: insufficient permissions for webhooks")
	case http.StatusNotFound:
		return fmt.Errorf("not found: subscription or endpoint does not exist")
	case http.StatusConflict:
		return fmt.Errorf("conflict: subscription may already exist for this board")
	case http.StatusTooManyRequests:
		return fmt.Errorf("rate limited: too many requests")
	default:
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}
}

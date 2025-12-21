package miro

import (
	"context"
	"fmt"

	"github.com/olgasafonova/miro-mcp-server/miro/webhooks"
)

// webhookManager lazily initializes and returns the webhook manager.
func (c *Client) webhookManager() *webhooks.Manager {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.webhookMgr == nil {
		c.webhookMgr = webhooks.NewManager(c.baseURL, func() string {
			return c.config.AccessToken
		})
	}
	return c.webhookMgr
}

// CreateWebhook creates a webhook subscription for a board.
func (c *Client) CreateWebhook(ctx context.Context, args CreateWebhookArgs) (CreateWebhookResult, error) {
	if args.BoardID == "" {
		return CreateWebhookResult{}, fmt.Errorf("board_id is required")
	}

	callbackURL := args.CallbackURL
	if callbackURL == "" {
		// Use configured webhook callback URL
		callbackURL = c.webhookCallbackURL
		if callbackURL == "" {
			return CreateWebhookResult{}, fmt.Errorf("callback_url is required (either in args or via MIRO_WEBHOOKS_CALLBACK_URL)")
		}
	}

	req := webhooks.CreateSubscriptionRequest{
		BoardID:     args.BoardID,
		CallbackURL: callbackURL,
		Status:      webhooks.StatusEnabled,
	}

	sub, err := c.webhookManager().Create(ctx, req)
	if err != nil {
		return CreateWebhookResult{}, fmt.Errorf("failed to create webhook: %w", err)
	}

	return CreateWebhookResult{
		ID:          sub.ID,
		BoardID:     sub.BoardID,
		CallbackURL: sub.CallbackURL,
		Status:      string(sub.Status),
		Message:     fmt.Sprintf("Webhook created for board %s", sub.BoardID),
	}, nil
}

// ListWebhooks lists webhook subscriptions, optionally filtered by board.
func (c *Client) ListWebhooks(ctx context.Context, args ListWebhooksArgs) (ListWebhooksResult, error) {
	subs, err := c.webhookManager().List(ctx, args.BoardID)
	if err != nil {
		return ListWebhooksResult{}, fmt.Errorf("failed to list webhooks: %w", err)
	}

	webhookInfos := make([]WebhookInfo, len(subs))
	for i, sub := range subs {
		webhookInfos[i] = WebhookInfo{
			ID:          sub.ID,
			BoardID:     sub.BoardID,
			CallbackURL: sub.CallbackURL,
			Status:      string(sub.Status),
			CreatedAt:   sub.CreatedAt,
		}
	}

	message := fmt.Sprintf("Found %d webhook(s)", len(subs))
	if args.BoardID != "" {
		message = fmt.Sprintf("Found %d webhook(s) for board %s", len(subs), args.BoardID)
	}

	return ListWebhooksResult{
		Webhooks: webhookInfos,
		Count:    len(subs),
		Message:  message,
	}, nil
}

// DeleteWebhook removes a webhook subscription.
func (c *Client) DeleteWebhook(ctx context.Context, args DeleteWebhookArgs) (DeleteWebhookResult, error) {
	if args.WebhookID == "" {
		return DeleteWebhookResult{}, fmt.Errorf("webhook_id is required")
	}

	err := c.webhookManager().Delete(ctx, args.WebhookID)
	if err != nil {
		return DeleteWebhookResult{
			Success:   false,
			WebhookID: args.WebhookID,
			Message:   fmt.Sprintf("Failed to delete webhook: %v", err),
		}, err
	}

	return DeleteWebhookResult{
		Success:   true,
		WebhookID: args.WebhookID,
		Message:   fmt.Sprintf("Webhook %s deleted successfully", args.WebhookID),
	}, nil
}

// GetWebhook retrieves a webhook subscription by ID.
func (c *Client) GetWebhook(ctx context.Context, args GetWebhookArgs) (GetWebhookResult, error) {
	if args.WebhookID == "" {
		return GetWebhookResult{}, fmt.Errorf("webhook_id is required")
	}

	sub, err := c.webhookManager().Get(ctx, args.WebhookID)
	if err != nil {
		return GetWebhookResult{}, fmt.Errorf("failed to get webhook: %w", err)
	}

	return GetWebhookResult{
		ID:          sub.ID,
		BoardID:     sub.BoardID,
		CallbackURL: sub.CallbackURL,
		Status:      string(sub.Status),
		CreatedAt:   sub.CreatedAt,
		ModifiedAt:  sub.ModifiedAt,
		Message:     fmt.Sprintf("Webhook %s for board %s", sub.ID, sub.BoardID),
	}, nil
}

package miro

import "time"

// CreateWebhookArgs represents the arguments for creating a webhook subscription.
type CreateWebhookArgs struct {
	// BoardID is the board to monitor for events
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"ID of the board to monitor for events"`
	// CallbackURL is where Miro will send events (optional, uses server's callback URL if not specified)
	CallbackURL string `json:"callback_url,omitempty" jsonschema_description:"URL where Miro will send webhook events. If not provided, uses the server's configured callback URL."`
}

// CreateWebhookResult represents the result of creating a webhook subscription.
type CreateWebhookResult struct {
	// ID is the webhook subscription ID
	ID string `json:"id"`
	// BoardID is the monitored board
	BoardID string `json:"board_id"`
	// CallbackURL is where events are sent
	CallbackURL string `json:"callback_url"`
	// Status is the subscription status
	Status string `json:"status"`
	// Message is a human-readable summary
	Message string `json:"message"`
}

// ListWebhooksArgs represents the arguments for listing webhook subscriptions.
type ListWebhooksArgs struct {
	// BoardID filters subscriptions by board (optional)
	BoardID string `json:"board_id,omitempty" jsonschema_description:"Filter webhooks by board ID. If not provided, lists all webhooks."`
}

// WebhookInfo represents a single webhook in the list.
type WebhookInfo struct {
	// ID is the webhook subscription ID
	ID string `json:"id"`
	// BoardID is the monitored board
	BoardID string `json:"board_id"`
	// CallbackURL is where events are sent
	CallbackURL string `json:"callback_url"`
	// Status is the subscription status
	Status string `json:"status"`
	// CreatedAt is when the subscription was created
	CreatedAt time.Time `json:"created_at"`
}

// ListWebhooksResult represents the result of listing webhook subscriptions.
type ListWebhooksResult struct {
	// Webhooks is the list of active webhook subscriptions
	Webhooks []WebhookInfo `json:"webhooks"`
	// Count is the total number of webhooks
	Count int `json:"count"`
	// Message is a human-readable summary
	Message string `json:"message"`
}

// DeleteWebhookArgs represents the arguments for deleting a webhook subscription.
type DeleteWebhookArgs struct {
	// WebhookID is the ID of the webhook subscription to delete
	WebhookID string `json:"webhook_id" jsonschema:"required" jsonschema_description:"ID of the webhook subscription to delete"`
}

// DeleteWebhookResult represents the result of deleting a webhook subscription.
type DeleteWebhookResult struct {
	// Success indicates if the deletion was successful
	Success bool `json:"success"`
	// WebhookID is the deleted webhook ID
	WebhookID string `json:"webhook_id"`
	// Message is a human-readable summary
	Message string `json:"message"`
}

// GetWebhookArgs represents the arguments for getting a webhook subscription.
type GetWebhookArgs struct {
	// WebhookID is the ID of the webhook subscription to retrieve
	WebhookID string `json:"webhook_id" jsonschema:"required" jsonschema_description:"ID of the webhook subscription to retrieve"`
}

// GetWebhookResult represents the result of getting a webhook subscription.
type GetWebhookResult struct {
	// ID is the webhook subscription ID
	ID string `json:"id"`
	// BoardID is the monitored board
	BoardID string `json:"board_id"`
	// CallbackURL is where events are sent
	CallbackURL string `json:"callback_url"`
	// Status is the subscription status
	Status string `json:"status"`
	// CreatedAt is when the subscription was created
	CreatedAt time.Time `json:"created_at"`
	// ModifiedAt is when the subscription was last modified
	ModifiedAt time.Time `json:"modified_at,omitempty"`
	// Message is a human-readable summary
	Message string `json:"message"`
}

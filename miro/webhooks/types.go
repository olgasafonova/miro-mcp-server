// Package webhooks provides webhook subscription management for Miro boards.
// Uses Miro's experimental webhooks API for real-time board event notifications.
package webhooks

import (
	"os"
	"time"
)

// Config holds webhook configuration from environment variables.
type Config struct {
	// Enabled determines if webhooks feature is active
	Enabled bool
	// CallbackURL is the public URL where Miro will send webhook events
	CallbackURL string
	// Secret is used to sign webhook payloads for verification
	Secret string
}

// LoadConfigFromEnv loads webhook configuration from environment variables.
func LoadConfigFromEnv() Config {
	return Config{
		Enabled:     os.Getenv("MIRO_WEBHOOKS_ENABLED") == "true",
		CallbackURL: os.Getenv("MIRO_WEBHOOKS_CALLBACK_URL"),
		Secret:      os.Getenv("MIRO_WEBHOOKS_SECRET"),
	}
}

// IsConfigured returns true if the minimum webhook config is set.
func (c Config) IsConfigured() bool {
	return c.CallbackURL != ""
}

// EventType represents the type of board event.
type EventType string

const (
	// EventItemCreate is triggered when an item is added to a board.
	EventItemCreate EventType = "board.item.create"
	// EventItemUpdate is triggered when an item is modified (except position changes).
	EventItemUpdate EventType = "board.item.update"
	// EventItemDelete is triggered when an item is removed from a board.
	EventItemDelete EventType = "board.item.delete"
)

// AllEventTypes returns all supported webhook event types.
func AllEventTypes() []EventType {
	return []EventType{EventItemCreate, EventItemUpdate, EventItemDelete}
}

// Status represents the webhook subscription status.
type Status string

const (
	StatusEnabled  Status = "enabled"
	StatusDisabled Status = "disabled"
	StatusPending  Status = "pending"
)

// Subscription represents a webhook subscription for a board.
type Subscription struct {
	// ID is the unique identifier for this subscription
	ID string `json:"id"`
	// BoardID is the board being monitored
	BoardID string `json:"boardId"`
	// CallbackURL is where Miro sends events
	CallbackURL string `json:"callbackUrl"`
	// Status indicates if the subscription is active
	Status Status `json:"status"`
	// CreatedAt is when the subscription was created
	CreatedAt time.Time `json:"createdAt"`
	// ModifiedAt is when the subscription was last modified
	ModifiedAt time.Time `json:"modifiedAt"`
	// CreatedBy is the user ID who created the subscription
	CreatedBy string `json:"createdBy"`
}

// CreateSubscriptionRequest is the payload for creating a new webhook subscription.
type CreateSubscriptionRequest struct {
	// BoardID is the board to monitor for events
	BoardID string `json:"boardId"`
	// CallbackURL is where Miro will send events
	CallbackURL string `json:"callbackUrl"`
	// Status controls if the subscription is active
	Status Status `json:"status,omitempty"`
}

// Event represents a webhook event received from Miro.
type Event struct {
	// Type is the event type (board.item.create, etc.)
	Type EventType `json:"type"`
	// Timestamp is when the event occurred
	Timestamp time.Time `json:"timestamp"`
	// BoardID is the board where the event occurred
	BoardID string `json:"boardId"`
	// ItemID is the affected item (if applicable)
	ItemID string `json:"itemId,omitempty"`
	// ItemType is the type of item (sticky_note, shape, etc.)
	ItemType string `json:"itemType,omitempty"`
	// UserID is who triggered the event
	UserID string `json:"userId,omitempty"`
	// Raw contains the full event payload as received
	Raw map[string]interface{} `json:"raw,omitempty"`
}

// WebhookPayload represents the incoming webhook request body from Miro.
type WebhookPayload struct {
	// Event contains the event details
	Event *EventPayload `json:"event,omitempty"`
	// Challenge is set when Miro validates the callback URL
	Challenge string `json:"challenge,omitempty"`
	// BoardID is included in the payload
	BoardID string `json:"boardId,omitempty"`
}

// EventPayload represents the event data within a webhook payload.
type EventPayload struct {
	// Type is the event type
	Type string `json:"type"`
	// Item contains the affected item details
	Item *EventItem `json:"item,omitempty"`
	// Board contains board info
	Board *EventBoard `json:"board,omitempty"`
	// User is who triggered the event
	User *EventUser `json:"user,omitempty"`
}

// EventItem represents item data in a webhook event.
type EventItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// EventBoard represents board data in a webhook event.
type EventBoard struct {
	ID string `json:"id"`
}

// EventUser represents user data in a webhook event.
type EventUser struct {
	ID string `json:"id"`
}

// ChallengeResponse is the response format for webhook challenge validation.
type ChallengeResponse struct {
	Challenge string `json:"challenge"`
}

// ListSubscriptionsResponse is the API response for listing subscriptions.
type ListSubscriptionsResponse struct {
	Data   []Subscription `json:"data"`
	Total  int            `json:"total"`
	Size   int            `json:"size"`
	Offset int            `json:"offset"`
	Limit  int            `json:"limit"`
}

package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Handler handles incoming webhook requests from Miro.
type Handler struct {
	config     Config
	logger     *slog.Logger
	eventBus   *EventBus
	mu         sync.RWMutex
	eventCount int64
}

// NewHandler creates a new webhook handler.
func NewHandler(config Config, logger *slog.Logger) *Handler {
	return &Handler{
		config:   config,
		logger:   logger,
		eventBus: NewEventBus(),
	}
}

// EventBus returns the event bus for subscribing to events.
func (h *Handler) EventBus() *EventBus {
	return h.eventBus
}

// EventCount returns the total number of events processed.
func (h *Handler) EventCount() int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.eventCount
}

// ServeHTTP implements http.Handler for webhook callbacks.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Failed to read webhook body", "error", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Verify signature if secret is configured
	if h.config.Secret != "" {
		signature := r.Header.Get("X-Miro-Signature")
		if !h.verifySignature(body, signature) {
			h.logger.Warn("Invalid webhook signature", "remote", r.RemoteAddr)
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// Parse payload
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.Error("Failed to parse webhook payload", "error", err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Handle challenge validation (Miro verifies the callback URL)
	if payload.Challenge != "" {
		h.handleChallenge(w, payload.Challenge)
		return
	}

	// Process the event
	if payload.Event != nil {
		h.processEvent(payload)
	}

	// Acknowledge receipt
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// handleChallenge responds to Miro's webhook URL validation request.
func (h *Handler) handleChallenge(w http.ResponseWriter, challenge string) {
	h.logger.Info("Received webhook challenge", "challenge", challenge[:min(10, len(challenge))]+"...")

	response := ChallengeResponse{Challenge: challenge}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// processEvent handles an incoming webhook event.
func (h *Handler) processEvent(payload WebhookPayload) {
	event := h.parseEvent(payload)
	if event == nil {
		return
	}

	// Increment counter
	h.mu.Lock()
	h.eventCount++
	h.mu.Unlock()

	h.logger.Info("Received webhook event",
		"type", event.Type,
		"board", event.BoardID,
		"item", event.ItemID,
	)

	// Publish to event bus
	h.eventBus.Publish(*event)
}

// parseEvent converts a webhook payload to an Event.
func (h *Handler) parseEvent(payload WebhookPayload) *Event {
	if payload.Event == nil {
		return nil
	}

	event := &Event{
		Type:      EventType(payload.Event.Type),
		Timestamp: time.Now(),
		Raw:       make(map[string]interface{}),
	}

	// Extract board ID
	if payload.Event.Board != nil {
		event.BoardID = payload.Event.Board.ID
	} else if payload.BoardID != "" {
		event.BoardID = payload.BoardID
	}

	// Extract item details
	if payload.Event.Item != nil {
		event.ItemID = payload.Event.Item.ID
		event.ItemType = payload.Event.Item.Type
	}

	// Extract user
	if payload.Event.User != nil {
		event.UserID = payload.Event.User.ID
	}

	return event
}

// verifySignature validates the HMAC signature of the webhook payload.
func (h *Handler) verifySignature(body []byte, signature string) bool {
	if signature == "" {
		return false
	}

	// Remove "sha256=" prefix if present
	signature = strings.TrimPrefix(signature, "sha256=")

	// Calculate expected signature
	mac := hmac.New(sha256.New, []byte(h.config.Secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	// Constant-time comparison to prevent timing attacks
	return hmac.Equal([]byte(expected), []byte(signature))
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

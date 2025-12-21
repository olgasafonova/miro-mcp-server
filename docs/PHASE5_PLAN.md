# Phase 5: Enterprise Features Implementation Plan

> **Goal**: Add OAuth 2.1, Webhooks, and Audit Logging to the Miro MCP Server
> **Created**: 2025-12-21

---

## Overview

Phase 5 introduces three enterprise-grade features:

| Feature | Complexity | Impact | Priority |
|---------|------------|--------|----------|
| Audit Logging (Local) | Medium | High | 1 |
| OAuth 2.1 Flow | High | High | 2 |
| Webhooks Support | High | Medium | 3 |

---

## 1. Audit Logging (Local)

**Purpose**: Track all MCP tool executions for debugging, compliance, and analytics.

### 1.1 Design

```
miro/
├── audit/
│   ├── logger.go      # Core audit logger interface
│   ├── file.go        # File-based audit log implementation
│   ├── memory.go      # In-memory ring buffer (dev/testing)
│   └── types.go       # AuditEvent, Config types
```

### 1.2 AuditEvent Structure

```go
type AuditEvent struct {
    ID        string                 `json:"id"`         // UUID
    Timestamp time.Time              `json:"timestamp"`
    Tool      string                 `json:"tool"`       // e.g., "miro_create_sticky"
    Method    string                 `json:"method"`     // e.g., "CreateSticky"
    UserID    string                 `json:"user_id"`    // Miro user ID
    UserEmail string                 `json:"user_email"` // From token validation
    BoardID   string                 `json:"board_id,omitempty"`
    ItemID    string                 `json:"item_id,omitempty"`
    Action    string                 `json:"action"`     // create, read, update, delete
    Input     map[string]interface{} `json:"input"`      // Sanitized input args
    Success   bool                   `json:"success"`
    Error     string                 `json:"error,omitempty"`
    Duration  time.Duration          `json:"duration_ms"`
}
```

### 1.3 AuditLogger Interface

```go
type AuditLogger interface {
    Log(ctx context.Context, event AuditEvent) error
    Query(ctx context.Context, opts QueryOptions) ([]AuditEvent, error)
    Close() error
}

type QueryOptions struct {
    Since     time.Time
    Until     time.Time
    Tool      string
    UserID    string
    BoardID   string
    Action    string
    Limit     int
}
```

### 1.4 Integration Points

1. **Handler wrapper** in `tools/handlers.go`:
   - Wrap each handler to capture timing, input, output
   - Log on success/failure

2. **New MCP tool** `miro_get_audit_log`:
   - Query recent audit events
   - Filter by tool, board, time range

### 1.5 Configuration

```bash
# Environment variables
MIRO_AUDIT_ENABLED=true          # Enable audit logging (default: true)
MIRO_AUDIT_PATH=/var/log/miro/   # Directory for audit logs
MIRO_AUDIT_RETENTION=30d         # How long to keep logs
```

### 1.6 Implementation Tasks

- [ ] Create `miro/audit/types.go` - Event and config types
- [ ] Create `miro/audit/logger.go` - Interface and factory
- [ ] Create `miro/audit/file.go` - JSON Lines file logger
- [ ] Create `miro/audit/memory.go` - In-memory ring buffer
- [ ] Add audit middleware to `tools/handlers.go`
- [ ] Add `miro_get_audit_log` tool
- [ ] Add tests for audit package

---

## 2. OAuth 2.1 Flow

**Purpose**: Replace static access tokens with full OAuth authorization code flow.

### 2.1 Miro OAuth Endpoints

| Endpoint | URL |
|----------|-----|
| Authorization | `https://miro.com/oauth/authorize` |
| Token Exchange | `https://api.miro.com/v1/oauth/token` |
| Revoke | `https://api.miro.com/v1/oauth/revoke` |

### 2.2 Token Lifecycle

```
Access Token:  60 minutes validity
Refresh Token: 60 days validity
```

### 2.3 New File Structure

```
miro/
├── oauth/
│   ├── provider.go    # OAuth flow implementation
│   ├── tokens.go      # Token storage/refresh
│   ├── server.go      # Local callback server
│   └── types.go       # OAuthConfig, TokenSet
```

### 2.4 OAuthProvider Interface

```go
type OAuthProvider interface {
    // GetAuthorizationURL returns the URL to redirect users for authorization
    GetAuthorizationURL(state string) string

    // ExchangeCode trades an authorization code for tokens
    ExchangeCode(ctx context.Context, code string) (*TokenSet, error)

    // RefreshToken gets a new access token using the refresh token
    RefreshToken(ctx context.Context, refreshToken string) (*TokenSet, error)

    // RevokeToken invalidates the current token
    RevokeToken(ctx context.Context, token string) error
}

type TokenSet struct {
    AccessToken  string    `json:"access_token"`
    RefreshToken string    `json:"refresh_token"`
    ExpiresAt    time.Time `json:"expires_at"`
    UserID       string    `json:"user_id"`
    TeamID       string    `json:"team_id"`
    Scope        string    `json:"scope"`
}
```

### 2.5 Token Storage

```go
type TokenStore interface {
    Save(ctx context.Context, tokens *TokenSet) error
    Load(ctx context.Context) (*TokenSet, error)
    Delete(ctx context.Context) error
}
```

Implementations:
- **FileTokenStore**: JSON file in user's home directory (`~/.miro/tokens.json`)
- **KeychainTokenStore**: macOS Keychain (optional, secure)

### 2.6 Auto-Refresh Mechanism

```go
// Client modification: check token before each request
func (c *Client) ensureValidToken(ctx context.Context) error {
    if c.tokenSet.ExpiresAt.Before(time.Now().Add(5 * time.Minute)) {
        newTokens, err := c.oauth.RefreshToken(ctx, c.tokenSet.RefreshToken)
        if err != nil {
            return fmt.Errorf("token refresh failed: %w", err)
        }
        c.tokenSet = newTokens
        c.tokenStore.Save(ctx, newTokens)
    }
    return nil
}
```

### 2.7 CLI Commands

```bash
# Initiate OAuth flow (opens browser)
./miro-mcp-server auth login

# Check current auth status
./miro-mcp-server auth status

# Logout (revoke and delete tokens)
./miro-mcp-server auth logout
```

### 2.8 Implementation Tasks

- [ ] Create `miro/oauth/types.go` - Config and token types
- [ ] Create `miro/oauth/provider.go` - OAuth flow
- [ ] Create `miro/oauth/tokens.go` - Token storage implementations
- [ ] Create `miro/oauth/server.go` - Local callback server
- [ ] Add `auth` subcommand to `main.go`
- [ ] Modify `miro/client.go` for auto-refresh
- [ ] Update config to support both static token and OAuth
- [ ] Add tests for OAuth package

---

## 3. Webhooks Support

**Purpose**: Enable real-time notifications when board items change.

### 3.1 Miro Webhook API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v2-experimental/webhooks/board_subscriptions` | POST | Create subscription |
| `/v2-experimental/webhooks/board_subscriptions/{id}` | GET | Get subscription |
| `/v2-experimental/webhooks/board_subscriptions/{id}` | DELETE | Delete subscription |

### 3.2 Event Types

| Event | Trigger |
|-------|---------|
| `board.item.create` | New item added to board |
| `board.item.update` | Item modified (except position) |
| `board.item.delete` | Item removed from board |

**Limitations**: Tags, connectors, and comments are not supported. Position changes don't trigger events.

### 3.3 New File Structure

```
miro/
├── webhooks/
│   ├── handler.go     # HTTP handler for callbacks
│   ├── manager.go     # Subscription management
│   ├── events.go      # Event types and parsing
│   └── types.go       # WebhookConfig, Subscription
```

### 3.4 Webhook Handler

```go
type WebhookHandler struct {
    subscriptions map[string]*Subscription
    eventChan     chan WebhookEvent
    logger        *slog.Logger
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Handle challenge validation
    // Parse event payload
    // Send to event channel
}
```

### 3.5 New MCP Tools

| Tool | Description |
|------|-------------|
| `miro_create_webhook` | Subscribe to board events |
| `miro_list_webhooks` | List active subscriptions |
| `miro_delete_webhook` | Remove subscription |

### 3.6 Event Delivery

For MCP clients to receive events:
1. **HTTP mode**: POST events to callback URL
2. **SSE endpoint**: `/events` for streaming

### 3.7 Implementation Tasks

- [ ] Create `miro/webhooks/types.go` - Event and subscription types
- [ ] Create `miro/webhooks/handler.go` - HTTP callback handler
- [ ] Create `miro/webhooks/manager.go` - Subscription CRUD
- [ ] Add webhook tools to `tools/definitions.go`
- [ ] Add `/webhooks` and `/events` endpoints in HTTP mode
- [ ] Add tests for webhook package

---

## Implementation Order

### Sprint 1: Audit Logging (1-2 days)
1. Create audit package structure
2. Implement file logger
3. Add handler middleware
4. Add query tool
5. Write tests

### Sprint 2: OAuth 2.1 (2-3 days)
1. Create OAuth package structure
2. Implement token exchange
3. Add token storage
4. Implement auto-refresh
5. Add CLI commands
6. Write tests

### Sprint 3: Webhooks (2-3 days)
1. Create webhooks package structure
2. Implement subscription management
3. Add callback handler
4. Add MCP tools
5. Add SSE endpoint
6. Write tests

---

## Testing Strategy

### Audit Logging
- Unit tests for each logger implementation
- Integration test for handler middleware
- Mock clock for time-based queries

### OAuth 2.1
- Mock OAuth server for unit tests
- Integration test with test Miro app
- Token refresh timing tests

### Webhooks
- Mock callback server
- Event parsing tests
- Challenge validation tests

---

## Documentation Updates

After implementation:
- [ ] Update README.md with new features
- [ ] Update CLAUDE.md with new patterns
- [ ] Update ROADMAP.md to mark Phase 5 complete
- [ ] Add CHANGELOG.md entry

---

## References

- [Miro OAuth 2.0 Guide](https://developers.miro.com/docs/getting-started-with-oauth)
- [Miro Webhooks Guide](https://developers.miro.com/docs/getting-started-with-webhooks)
- [Miro Audit Logs API](https://developers.miro.com/reference/enterprise-get-audit-logs)

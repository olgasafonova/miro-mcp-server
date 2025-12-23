# Claude Code Instructions for miro-mcp-server

This file provides context for Claude Code sessions working on this repository.

## Project Overview

**Goal**: Build the most comprehensive, performant, secure, and user-friendly Miro MCP server in Go.

**Current Status**: 68 tools implemented. Phases 1-7 complete, plus batch update/delete. (Webhooks removed - Miro sunset Dec 2025)

## Quick Start

```bash
# Build
go build -o miro-mcp-server .

# Run (stdio mode) with static token
MIRO_ACCESS_TOKEN=your_token ./miro-mcp-server

# Run (HTTP mode)
MIRO_ACCESS_TOKEN=your_token ./miro-mcp-server -http :8080

# OAuth login (alternative to static token)
MIRO_CLIENT_ID=xxx MIRO_CLIENT_SECRET=yyy ./miro-mcp-server auth login

# Test
go test ./...
```

## Architecture

```
miro-mcp-server/
├── main.go                    # Entry point, transport setup, auth CLI
├── miro/
│   ├── client.go              # Base client (HTTP, retry, caching, token refresh)
│   ├── interfaces.go          # MiroClient interface + service interfaces
│   ├── config.go              # Environment config
│   │
│   │   # Domain implementations (one file per domain)
│   ├── boards.go              # Board operations (list, get, create, copy, delete)
│   ├── items.go               # Item CRUD (list, get, update, delete, search)
│   ├── create.go              # Create operations (sticky, shape, text, etc.)
│   ├── tags.go                # Tag operations (create, attach, detach)
│   ├── groups.go              # Group operations (create, ungroup)
│   ├── members.go             # Member operations (list, share)
│   ├── mindmaps.go            # Mindmap operations (create, get, list, delete)
│   ├── frames.go              # Frame operations (get, update, delete, get items)
│   ├── export.go              # Export operations (picture, export jobs)
│   ├── diagrams.go            # Diagram generation from Mermaid
│   │
│   │   # Domain types (one file per domain)
│   ├── types_boards.go        # Board-related types
│   ├── types_items.go         # Item-related types
│   ├── types_operations.go    # CRUD operation types
│   ├── types_tags.go          # Tag types
│   ├── types_groups.go        # Group types
│   ├── types_members.go       # Member types
│   ├── types_mindmaps.go      # Mindmap types
│   ├── types_frames.go        # Frame types
│   ├── types_export.go        # Export types
│   ├── types_diagrams.go      # Diagram types
│   │
│   ├── audit/                 # Audit logging package
│   │   ├── types.go           # Event types and config
│   │   ├── file.go            # File-based JSON Lines logger
│   │   ├── memory.go          # In-memory ring buffer logger
│   │   └── factory.go         # Logger factory and helpers
│   │
│   ├── oauth/                 # OAuth 2.1 package
│   │   ├── types.go           # Config, TokenSet, errors
│   │   ├── provider.go        # OAuth flow (PKCE, token exchange)
│   │   ├── tokens.go          # Token storage (file, memory)
│   │   ├── server.go          # Local callback server
│   │   └── auth.go            # AuthFlow orchestration
│   │
│   ├── webhooks/              # Webhooks package
│   │   ├── types.go           # Config, Subscription, Event types
│   │   ├── handler.go         # HTTP callback + signature verification
│   │   ├── manager.go         # Subscription CRUD via Miro API
│   │   └── events.go          # EventBus, RingBuffer, SSEHandler
│   │
│   └── diagrams/              # Diagram parsing and layout
│       ├── types.go           # Diagram, Node, Edge types
│       ├── mermaid.go         # Mermaid flowchart parser
│       ├── layout.go          # Auto-layout algorithm (Sugiyama-style)
│       └── converter.go       # Convert to Miro API items
│
└── tools/
    ├── definitions.go         # Tool specs (add new tools here)
    ├── handlers.go            # Map-based handler registration + audit
    ├── mock_client_test.go    # Mock implementation for testing
    └── handlers_test.go       # Handler unit tests
```

### Key Interfaces (miro/interfaces.go)

```go
// Service interfaces enable mock-based testing
type BoardService interface { ... }
type ItemService interface { ... }
type CreateService interface { ... }
type TagService interface { ... }
type GroupService interface { ... }
type MemberService interface { ... }
type MindmapService interface { ... }
type FrameService interface { ... }
type TokenService interface { ... }
type ExportService interface { ... }
type WebhookService interface { ... }
type DiagramService interface { ... }
type ConnectorService interface { ... }

// MiroClient embeds all service interfaces
type MiroClient interface {
    BoardService
    ItemService
    CreateService
    TagService
    GroupService
    MemberService
    MindmapService
    FrameService
    TokenService
    ExportService
    WebhookService
    DiagramService
}
```

## Adding a New Tool (4 Steps)

### 1. Add types in domain-specific type file

Add to the appropriate `miro/types_*.go` file (e.g., `types_operations.go` for new create operations):

```go
type NewFeatureArgs struct {
    BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
    // Add fields...
}

type NewFeatureResult struct {
    ID      string `json:"id"`
    Message string `json:"message"`
}
```

### 2. Add method to service interface and implementation

First, add to the appropriate interface in `miro/interfaces.go`:
```go
type CreateService interface {
    // ... existing methods ...
    NewFeature(ctx context.Context, args NewFeatureArgs) (NewFeatureResult, error)
}
```

Then implement in the domain file (e.g., `miro/create.go`):
```go
func (c *Client) NewFeature(ctx context.Context, args NewFeatureArgs) (NewFeatureResult, error) {
    if args.BoardID == "" {
        return NewFeatureResult{}, fmt.Errorf("board_id is required")
    }
    // Implementation...
}
```

### 3. Add tool spec in `tools/definitions.go`

```go
{
    Name:     "miro_new_feature",
    Method:   "NewFeature",
    Title:    "New Feature",
    Category: "create",
    Description: `Short description.

USE WHEN: User says "..."

PARAMETERS:
- param: Description

VOICE-FRIENDLY: "Created X"`,
},
```

### 4. Register in `tools/handlers.go`

Add one line to `buildHandlerMap()`:
```go
"NewFeature": makeHandler(h, h.client.NewFeature),
```

That's it! The generic `makeHandler` function handles type-safe registration automatically.

## Implementation Status

See `ROADMAP.md` for full details.

### Phase 1: Complete ✅ (26 tools)
- **Boards**: list, get, create, copy, delete
- **Create**: sticky, shape, text, connector, frame, card, image, document, embed, bulk
- **Read**: list items, list all items, get item, search
- **Tags**: create, list, attach, detach, get item tags
- **Modify**: update item, delete item

### Phase 2: Complete ✅ (+3 tools, +4 enhancements)
- **New Tools**: `miro_find_board`, `miro_get_board_summary`, `miro_create_sticky_grid`
- **Enhancements**:
  - Token validation on startup (fails fast with clear error)
  - Board name resolution (find by name, not just ID)
  - Input sanitization (validates IDs and content)
  - Retry with exponential backoff (handles rate limits)

### Phase 3: Complete ✅ (+5 tools)
- **Groups**: `miro_create_group`, `miro_ungroup`
- **Board Members**: `miro_list_board_members`, `miro_share_board`
- **Mindmaps**: `miro_create_mindmap_node`

### Phase 4: Complete ✅ (+4 tools)
- **Export**: `miro_get_board_picture`, `miro_create_export_job`, `miro_get_export_job_status`, `miro_get_export_job_results`
- **Note**: Export jobs require Enterprise plan; board picture works for all plans

### Phase 5: Complete ✅ (+1 tool)
- **Audit**: `miro_get_audit_log` (file/memory loggers)
- **Webhooks**: ~~REMOVED~~ - Miro sunset experimental webhooks Dec 5, 2025
- **OAuth 2.1**: PKCE flow with auto-refresh

### Phase 6: Complete ✅ (+1 tool)
- **Diagrams**: `miro_generate_diagram` (Mermaid flowchart → Miro shapes)
- **Parser**: flowchart/graph keywords, TB/LR/BT/RL directions, 5 node shapes
- **Layout**: Sugiyama-style layered algorithm with barycenter ordering

### Phase 7: Complete ✅ (+7 tools)
- **Frames**: `miro_get_frame`, `miro_update_frame`, `miro_delete_frame`, `miro_get_frame_items`
- **Mindmaps**: `miro_get_mindmap_node`, `miro_list_mindmap_nodes`, `miro_delete_mindmap_node`
- **Distribution**: Homebrew tap, Docker image, ARM64 Linux binary, install script

## Miro API Quick Reference

Base: `https://api.miro.com/v2`

| Endpoint | Use For |
|----------|---------|
| `POST /boards/{id}/cards` | Cards with due dates |
| `POST /boards/{id}/images` | Images from URL |
| `GET/POST /boards/{id}/tags` | Tag management |
| `POST /boards/{id}/items/{id}/tags/{id}` | Attach tag |
| `POST /boards` | Create board |
| `POST /boards/{id}/copy` | Copy board |
| `GET /users/me` | Token validation |
| `POST /orgs/{id}/boards/export/jobs` | Create export job (Enterprise) |
| `GET /orgs/{id}/boards/export/jobs/{id}` | Get export status (Enterprise) |

## Code Style

- **Validation first**: Check required fields at method start
- **Wrap errors**: Use `fmt.Errorf("context: %w", err)`
- **Log execution**: Add to `logExecution()` in handlers.go
- **Voice-friendly**: Tool descriptions should be speakable

## Testing

```bash
# Unit tests (uses MockClient, no API calls)
go test ./...

# With coverage
go test -cover ./...

# Integration (needs real token)
MIRO_TEST_TOKEN=xxx go test -tags=integration ./...
```

### Mock-Based Testing

The `tools/` package includes a complete mock implementation:

- `mock_client_test.go` - MockClient implementing MiroClient interface
- `handlers_test.go` - Unit tests for all handlers

MockClient features:
- **Function injection** - Override any method with custom behavior
- **Call tracking** - Verify which methods were called with what arguments
- **Default responses** - Returns sensible defaults for quick testing

```go
// Example: Test with custom behavior
mock := &MockClient{
    CreateStickyFn: func(ctx context.Context, args miro.CreateStickyArgs) (miro.CreateStickyResult, error) {
        return miro.CreateStickyResult{}, errors.New("API error")
    },
}

// Example: Verify calls
if !mock.WasCalled("CreateSticky") {
    t.Error("expected CreateSticky to be called")
}
```

## Key Files to Review First

1. `miro/interfaces.go` - All service interfaces and MiroClient composite
2. `miro/create.go` - See `CreateSticky` as implementation template
3. `miro/types_operations.go` - Common Args/Result types
4. `tools/definitions.go` - Tool description format
5. `tools/handlers.go` - Map-based registration with generics
6. `ROADMAP.md` - Full implementation plan

## Current Advantages Over Competitors

1. **Only Go-based Miro MCP** - faster, smaller, single binary
2. **Rate limiting** - semaphore-based (5 concurrent)
3. **Caching** - 2min TTL for boards
4. **Voice-optimized** - tool descriptions for voice assistants
5. **Panic recovery** - production-safe handlers
6. **Dual transport** - stdio + HTTP
7. **Token validation** - fails fast on startup with clear error
8. **Board name resolution** - find boards by name, not just ID
9. **Input sanitization** - validates all IDs and content
10. **Retry with backoff** - handles rate limits gracefully
11. **Composite tools** - efficient multi-step operations
12. **Interface-based design** - enables mock-based unit testing
13. **Generic handler registration** - type-safe, single-line tool registration
14. **Audit logging** - track all tool executions with file/memory loggers
15. **OAuth 2.1 with PKCE** - secure authentication with auto-refresh

## What NOT to Change

- Tool naming convention: `miro_verb_noun`
- JSON field naming: snake_case
- jsonschema tags format
- Voice-friendly description format

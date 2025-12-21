# Claude Code Instructions for miro-mcp-server

This file provides context for Claude Code sessions working on this repository.

## Project Overview

**Goal**: Build the most comprehensive, performant, secure, and user-friendly Miro MCP server in Go.

**Current Status**: 34 tools implemented. Phase 1, 2, and 3 complete.

## Quick Start

```bash
# Build
go build -o miro-mcp-server .

# Run (stdio mode)
MIRO_ACCESS_TOKEN=your_token ./miro-mcp-server

# Run (HTTP mode)
MIRO_ACCESS_TOKEN=your_token ./miro-mcp-server -http :8080

# Test
go test ./...
```

## Architecture

```
miro-mcp-server/
├── main.go              # Entry point, transport setup
├── miro/
│   ├── client.go        # API client (add new API methods here)
│   ├── config.go        # Environment config
│   └── types.go         # All types (Args/Result structs)
└── tools/
    ├── definitions.go   # Tool specs (add new tools here)
    └── handlers.go      # Handler registration
```

## Adding a New Tool (4 Steps)

### 1. Add types in `miro/types.go`

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

### 2. Add method in `miro/client.go`

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

Add to `registerByName()` switch:
```go
case "NewFeature":
    h.register(server, tool, spec, h.client.NewFeature)
```

Add to `register()` switch:
```go
case func(context.Context, miro.NewFeatureArgs) (miro.NewFeatureResult, error):
    register(h, server, tool, spec, m)
```

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

## Code Style

- **Validation first**: Check required fields at method start
- **Wrap errors**: Use `fmt.Errorf("context: %w", err)`
- **Log execution**: Add to `logExecution()` in handlers.go
- **Voice-friendly**: Tool descriptions should be speakable

## Testing

```bash
# Unit tests
go test ./...

# With coverage
go test -cover ./...

# Integration (needs real token)
MIRO_TEST_TOKEN=xxx go test -tags=integration ./...
```

## Key Files to Review First

1. `miro/client.go` - See `CreateSticky` as template
2. `miro/types.go` - Many types already defined (Cards, Images, Tags)
3. `tools/definitions.go` - Tool description format
4. `ROADMAP.md` - Full implementation plan

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

## What NOT to Change

- Tool naming convention: `miro_verb_noun`
- JSON field naming: snake_case
- jsonschema tags format
- Voice-friendly description format

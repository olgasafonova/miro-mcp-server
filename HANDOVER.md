# Session Handover - Miro MCP Server

> **Date**: 2025-12-22 (v1.6.0 Release)
> **Project**: miro-mcp-server
> **Location**: `/Users/olgasafonova/go/src/miro-mcp-server`
> **Version**: v1.6.0
> **Repo**: https://github.com/olgasafonova/miro-mcp-server

---

## Current State

**58 MCP tools** for Miro whiteboard control. Build passes, all tests pass.

```bash
# Verify build
cd /Users/olgasafonova/go/src/miro-mcp-server
go build -o miro-mcp-server .
go test ./...
```

**MCP is configured in Claude Code** at user level with correct team_id.

---

## What Was Done This Session (Session 6) - Documentation & Competitive Analysis

### 1. Comprehensive Competitive Analysis

Analyzed all Miro MCP competitors:

| Server | Tools | Stars | Language | Status |
|--------|-------|-------|----------|--------|
| **Miro Official** (mcp.miro.com) | ~5 | N/A | Cloud | Beta, limited |
| **k-jarzyna/mcp-miro** | **119** | 59 | TypeScript | Most comprehensive |
| **evalstate/mcp-miro** | 3 | 101 | TypeScript | Very limited |
| **LuotoCompany/mcp-server-miro** | ~40 | 14 | TypeScript | Experimental |
| **Ours** | **58** | 0 | Go | Fast, unique features, closing gap |

### 2. Feature Gap Analysis (Updated v1.6.0)

**Added in v1.6.0 (12 tools):**

| Category | Tools Added |
|----------|-------------|
| **Board** | ✅ `update_board` |
| **Members** | ✅ `get_board_member`, `remove_board_member`, `update_board_member` |
| **Groups** | ✅ `list_groups`, `get_group`, `get_group_items`, `delete_group` |
| **App Cards** | ✅ `create_app_card`, `get_app_card`, `update_app_card`, `delete_app_card` |

**Still missing (4):**

| Category | Tools Missing |
|----------|---------------|
| **Mindmaps** | `get_mindmap_node`, `list_mindmap_nodes`, `delete_mindmap_node` (API experimental, requires testing) |
| **Groups** | `update_group` (not in Miro API) |

**Our unique advantages (they don't have):**
- Single binary (no Node.js)
- Performance optimizations (caching, circuit breaker, rate limiting)
- Mermaid diagram generation with auto-layout
- `find_board` (search by name)
- `search_board` (content search)
- `create_sticky_grid`
- `get_board_summary`
- Local audit logging
- Voice-optimized descriptions

### 3. Documentation Created

**SETUP.md** - Comprehensive setup guide with:
- One-line downloads for all platforms
- Token acquisition instructions
- Copy-paste configs for: Claude Code, Claude Desktop (macOS/Windows/Linux), Cursor, VS Code + Copilot, Windsurf, Replit, n8n
- Troubleshooting section

**README.md** - Simplified and cleaned up:
- Quick 3-step start
- Collapsible tool categories
- Points to SETUP.md for detailed guides

---

## Competitive Intelligence (Internal - DO NOT publish)

### Miro Official MCP (https://mcp.miro.com/)
- Cloud-hosted, OAuth 2.1 with dynamic registration
- Beta status, limited tools (~5 main operations)
- Focused on diagram generation and code generation
- Enterprise admin controls
- Supported clients: Cursor, Claude Code, VS Code, Replit, Windsurf, etc.

### k-jarzyna/mcp-miro (Main Competitor)
- **119 tools** - most comprehensive
- Full enterprise features (legal holds, cases, org management)
- Per-item-type CRUD (separate update/delete for each type)
- Requires Node.js
- No performance optimizations
- No unique composite tools

### evalstate/mcp-miro
- Only 3 tools but 101 stars
- Easy Smithery install
- Good for photo-to-Miro workflow

---

## Next Session Priority: Add Missing Tools

### Phase 1: Board & Member Tools (4 tools)

```go
// In miro/boards.go
func (c *Client) UpdateBoard(ctx context.Context, args UpdateBoardArgs) (UpdateBoardResult, error)

// In miro/members.go
func (c *Client) GetBoardMember(ctx context.Context, args GetBoardMemberArgs) (GetBoardMemberResult, error)
func (c *Client) RemoveBoardMember(ctx context.Context, args RemoveBoardMemberArgs) (RemoveBoardMemberResult, error)
func (c *Client) UpdateBoardMember(ctx context.Context, args UpdateBoardMemberArgs) (UpdateBoardMemberResult, error)
```

### Phase 2: Group Tools (5 tools)

```go
// In miro/groups.go
func (c *Client) ListGroups(ctx context.Context, args ListGroupsArgs) (ListGroupsResult, error)
func (c *Client) GetGroup(ctx context.Context, args GetGroupArgs) (GetGroupResult, error)
func (c *Client) GetGroupItems(ctx context.Context, args GetGroupItemsArgs) (GetGroupItemsResult, error)
func (c *Client) UpdateGroup(ctx context.Context, args UpdateGroupArgs) (UpdateGroupResult, error)
func (c *Client) DeleteGroup(ctx context.Context, args DeleteGroupArgs) (DeleteGroupResult, error)
```

### Phase 3: Mindmap Tools (3 tools)

```go
// In miro/mindmaps.go - NOTE: Check API status first, may return 405
func (c *Client) GetMindmapNode(ctx context.Context, args GetMindmapNodeArgs) (GetMindmapNodeResult, error)
func (c *Client) ListMindmapNodes(ctx context.Context, args ListMindmapNodesArgs) (ListMindmapNodesResult, error)
func (c *Client) DeleteMindmapNode(ctx context.Context, args DeleteMindmapNodeArgs) (DeleteMindmapNodeResult, error)
```

### Phase 4: App Card Tools (4 tools)

```go
// In miro/create.go or new miro/appcards.go
func (c *Client) CreateAppCard(ctx context.Context, args CreateAppCardArgs) (CreateAppCardResult, error)
func (c *Client) GetAppCard(ctx context.Context, args GetAppCardArgs) (GetAppCardResult, error)
func (c *Client) UpdateAppCard(ctx context.Context, args UpdateAppCardArgs) (UpdateAppCardResult, error)
func (c *Client) DeleteAppCard(ctx context.Context, args DeleteAppCardArgs) (DeleteAppCardResult, error)
```

---

## Miro API Endpoints for New Tools

| Tool | Endpoint | Method |
|------|----------|--------|
| UpdateBoard | `/v2/boards/{id}` | PATCH |
| GetBoardMember | `/v2/boards/{id}/members/{member_id}` | GET |
| RemoveBoardMember | `/v2/boards/{id}/members/{member_id}` | DELETE |
| UpdateBoardMember | `/v2/boards/{id}/members/{member_id}` | PATCH |
| ListGroups | `/v2/boards/{id}/groups` | GET |
| GetGroup | `/v2/boards/{id}/groups/{group_id}` | GET |
| GetGroupItems | `/v2/boards/{id}/groups/{group_id}/items` | GET |
| UpdateGroup | `/v2/boards/{id}/groups/{group_id}` | PATCH |
| DeleteGroup | `/v2/boards/{id}/groups/{group_id}` | DELETE |
| GetMindmapNode | `/v2/boards/{id}/mindmap_nodes/{node_id}` | GET |
| ListMindmapNodes | `/v2/boards/{id}/mindmap_nodes` | GET |
| DeleteMindmapNode | `/v2/boards/{id}/mindmap_nodes/{node_id}` | DELETE |
| CreateAppCard | `/v2/boards/{id}/app_cards` | POST |
| GetAppCard | `/v2/boards/{id}/app_cards/{item_id}` | GET |
| UpdateAppCard | `/v2/boards/{id}/app_cards/{item_id}` | PATCH |
| DeleteAppCard | `/v2/boards/{id}/app_cards/{item_id}` | DELETE |

---

## Test Board

**URL**: https://miro.com/app/board/uXjVOXQCe5c=
**Name**: "All tests"
**Board ID**: `uXjVOXQCe5c=`

---

## MCP Server Configuration

**User-level config** in `~/.claude.json`:
```json
{
  "mcpServers": {
    "miro": {
      "type": "stdio",
      "command": "/Users/olgasafonova/go/src/miro-mcp-server/miro-mcp-server",
      "args": [],
      "env": {
        "MIRO_ACCESS_TOKEN": "eyJtaXJvLm9yaWdpbiI6ImV1MDEifQ_LUIBL31IVOjKuoLn6HoWVwjx-sg",
        "MIRO_TEAM_ID": "3458764516184293832"
      }
    }
  }
}
```

---

## Testing Status: 45/58 Tools Verified

v1.6.0 adds 12 new tools (58 total).

### Known Issues

| Tool | Issue | Cause |
|------|-------|-------|
| `miro_get_board_picture` | Returns empty | May need specific conditions |
| `miro_copy_board` | 500 on complex boards | Miro API limitation |
| Export tools | Not tested | Require Enterprise plan |

### Fixed in v1.6.0

| Tool | Issue | Fix |
|------|-------|-----|
| `miro_create_mindmap_node` | 405 error | Changed to v2-experimental endpoint with `mindmap_nodes` path |

---

## Quick Commands

```bash
# Build
go build -o miro-mcp-server .

# Test
go test ./...

# Benchmarks
go test ./miro/... -bench=. -benchmem

# Coverage
go test -cover ./...

# Run
MIRO_ACCESS_TOKEN=xxx MIRO_TEAM_ID=3458764516184293832 ./miro-mcp-server

# Build all platforms
GOOS=darwin GOARCH=arm64 go build -o dist/miro-mcp-server-darwin-arm64 .
GOOS=darwin GOARCH=amd64 go build -o dist/miro-mcp-server-darwin-amd64 .
GOOS=linux GOARCH=amd64 go build -o dist/miro-mcp-server-linux-amd64 .
GOOS=windows GOARCH=amd64 go build -o dist/miro-mcp-server-windows-amd64.exe .

# Create release
gh release create v1.6.0 dist/* --title "v1.6.0 - Documentation & Tools" --notes "..."
```

---

## Architecture Overview

```
miro-mcp-server/
├── main.go                    # Entry point
├── README.md                  # Quick start (simplified)
├── SETUP.md                   # Detailed setup for all IDEs
├── HANDOVER.md                # This file - session continuity
├── miro/
│   ├── client.go              # HTTP client + rate limiter + circuit breaker
│   ├── cache.go               # Item-level caching
│   ├── circuitbreaker.go      # Per-endpoint circuit breakers
│   ├── ratelimit.go           # Adaptive rate limiting
│   │
│   ├── boards.go, items.go, create.go, tags.go, ...  # Domain logic
│   ├── types_*.go             # Type definitions
│   │
│   ├── audit/                 # Audit logging
│   ├── oauth/                 # OAuth 2.1 PKCE
│   └── diagrams/              # Mermaid parser + layout
│
└── tools/
    ├── definitions.go         # 46 tool specs
    ├── handlers.go            # Generic handler registration
    └── handlers_test.go       # Unit tests with MockClient
```

---

## v1.6.0 Release Checklist

- [x] Performance optimizations (caching, circuit breaker, rate limiting)
- [x] Documentation (SETUP.md, README.md simplified)
- [ ] Add missing tools (~15)
- [ ] Fix mindmap API issue
- [ ] Test all platforms
- [ ] Create GitHub release

---

## Session 7 Priority

1. **Add missing tools** - Start with UpdateBoard, then member tools, then groups
2. **Test mindmap API** - May need to check Miro docs for endpoint changes
3. **Build and test all platforms** - Ensure Windows works
4. **Release v1.6.0**

---

**Ready for tool implementation!**

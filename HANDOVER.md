# Session Handover - Miro MCP Server

> **Date**: 2025-12-22 (Documentation Update Session)
> **Project**: miro-mcp-server
> **Location**: `/Users/olgasafonova/go/src/miro-mcp-server`
> **Version**: v1.6.1
> **Repo**: https://github.com/olgasafonova/miro-mcp-server
> **Release**: https://github.com/olgasafonova/miro-mcp-server/releases/tag/v1.6.1

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

## What Was Done This Session

### Documentation Update - Competitive Analysis & README Fix

**Problem**: README was outdated - showed "46 tools" when we have 58. Tool lists were incomplete.

**Actions Taken**:

1. **Competitive Analysis** against:
   - k-jarzyna/mcp-miro (80 tools, TypeScript, 59 stars)
   - Miro Official MCP Server (Public Beta, ~5 tools, cloud-hosted)
   - LuotoCompany/mcp-server-miro (experimental)

2. **Fixed README.md**:
   - Updated tool count from 46 → 58
   - Reorganized tool categories (8 sections)
   - Added all v1.6.0 tools (groups, members, app cards)
   - Added "Why This Server?" comparison table
   - Added Account Compatibility section
   - Added sequence diagram example
   - Added AI tool status table

3. **Fixed SETUP.md**:
   - Corrected n8n section - was misleading about MCP support
   - Added link to n8n's native Miro integration
   - Clarified that n8n doesn't support MCP natively

4. **Updated CLAUDE.md**: 46 → 58 tools

**Competitive Position**:
| Server | Tools | Our Advantage |
|--------|-------|---------------|
| k-jarzyna | 80 | We have: single binary, Mermaid diagrams, no Node.js |
| Miro Official | ~5 | We have: 58 tools, self-hosted, works offline |

**Gap Analysis** (for future sessions):
- Missing: Frame CRUD (+4), Mindmap get/list/delete (+3), Data Tables API (+5)
- Missing: Docker image, Homebrew tap
- Need: Complete test coverage (39/58 tested)

---

### v1.6.1 - Mindmap Request Body Fix

**Problem**: `miro_create_mindmap_node` was returning 400 Invalid Parameters even after v1.6.0 endpoint fix.

**Root Cause**: Miro's v2-experimental mindmap API uses a nested structure for content:
- Wrong: `data.content`
- Correct: `data.nodeView.data.content`

**Fix** in `miro/mindmaps.go`:
```go
// Before (wrong):
reqBody := map[string]interface{}{
    "data": map[string]interface{}{
        "content": args.Content,
    },
}

// After (correct):
nodeViewData := map[string]interface{}{
    "content": args.Content,
}
nodeView := map[string]interface{}{
    "data": nodeViewData,
}
reqBody := map[string]interface{}{
    "data": map[string]interface{}{
        "nodeView": nodeView,
    },
}
```

**Verified**: Created 6 mindmap nodes on test board, including root and child nodes.

---

### v1.6.0 - Added 12 New Tools (46 → 58)

| Category | Tools Added | File |
|----------|-------------|------|
| **Board** | `miro_update_board` | `miro/boards.go` |
| **Members** | `miro_get_board_member`, `miro_remove_board_member`, `miro_update_board_member` | `miro/members.go` |
| **Groups** | `miro_list_groups`, `miro_get_group`, `miro_get_group_items`, `miro_delete_group` | `miro/groups.go` |
| **App Cards** | `miro_create_app_card`, `miro_get_app_card`, `miro_update_app_card`, `miro_delete_app_card` | `miro/appcards.go` (new) |

### 2. Fixed Mindmap API (405 Error)

**Problem**: `miro_create_mindmap_node` was returning 405 Method Not Allowed.

**Root Cause**: Miro's mindmap API uses:
- `v2-experimental` base URL (not `v2`)
- `mindmap_nodes` path (not `mind_map_nodes`)

**Fix**:
1. Added `ExperimentalBaseURL` constant in `miro/client.go`
2. Added `requestExperimental()` method that uses the experimental base URL
3. Updated `miro/mindmaps.go` to use `requestExperimental()` with correct path

```go
// miro/client.go
const ExperimentalBaseURL = "https://api.miro.com/v2-experimental"

func (c *Client) requestExperimental(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
    if c.baseURL == BaseURL {
        originalBaseURL := c.baseURL
        c.baseURL = ExperimentalBaseURL
        defer func() { c.baseURL = originalBaseURL }()
    }
    return c.request(ctx, method, path, body)
}

// miro/mindmaps.go
respBody, err := c.requestExperimental(ctx, http.MethodPost, "/boards/"+args.BoardID+"/mindmap_nodes", reqBody)
```

### 3. Created Documentation

- **SETUP.md**: Comprehensive setup guide for all IDEs/platforms
- Updated **HANDOVER.md**: This file
- Updated **TESTING.md**: Added new tools and fixed issues

### 4. Released v1.6.0

```bash
gh release create v1.6.0 dist/* --title "v1.6.0 - 12 New Tools & Mindmap Fix"
```

---

## Files Changed This Session

### v1.6.1 Changes
| File | Changes |
|------|---------|
| `main.go` | Updated ServerVersion to 1.6.1 |
| `miro/mindmaps.go` | Fixed request body structure + response parsing |
| `TESTING.md` | Added v1.6.1 fix, moved mindmap to tested |
| `HANDOVER.md` | Added v1.6.1 section |

### New Files (v1.6.0)
| File | Purpose |
|------|---------|
| `miro/appcards.go` | App card CRUD operations |
| `miro/types_appcards.go` | App card type definitions |
| `SETUP.md` | Comprehensive IDE setup guide |

### Modified Files
| File | Changes |
|------|---------|
| `miro/client.go` | Added `ExperimentalBaseURL`, `requestExperimental()` |
| `miro/boards.go` | Added `UpdateBoard()` |
| `miro/members.go` | Added `GetBoardMember()`, `RemoveBoardMember()`, `UpdateBoardMember()` |
| `miro/groups.go` | Added `ListGroups()`, `GetGroup()`, `GetGroupItems()`, `DeleteGroup()` |
| `miro/mindmaps.go` | Fixed to use `requestExperimental()` with correct path |
| `miro/interfaces.go` | Added `AppCardService` interface |
| `miro/types_boards.go` | Added `UpdateBoardArgs`, `UpdateBoardResult` |
| `miro/types_members.go` | Added get/remove/update member types |
| `miro/types_groups.go` | Added list/get/delete group types |
| `tools/definitions.go` | Added 12 new tool definitions |
| `tools/handlers.go` | Added 12 new handlers |
| `tools/definitions_test.go` | Updated expected count to 58, added categories |
| `tools/mock_client_test.go` | Added mock methods for all new tools |
| `miro/client_test.go` | Fixed mindmap test path assertion |

---

## Tool Categories (58 Total)

| Category | Count | Tools |
|----------|-------|-------|
| **boards** | 6 | list, find, get, create, copy, delete, update |
| **create** | 11 | sticky, shape, text, connector, frame, card, image, document, embed, bulk, sticky_grid, app_card, mindmap_node |
| **read** | 6 | list_items, list_all_items, get_item, search, get_board_summary, get_app_card |
| **update** | 5 | update_item, update_tag, update_connector, update_board_member, update_app_card |
| **delete** | 5 | delete_item, delete_tag, delete_connector, delete_group, delete_app_card |
| **tags** | 5 | create, list, attach, detach, get_item_tags |
| **connectors** | 4 | create, list, get, update, delete |
| **groups** | 6 | create, ungroup, list, get, get_items, delete |
| **members** | 5 | list, share, get, remove, update |
| **diagrams** | 1 | generate_diagram |
| **export** | 4 | get_board_picture, create_export_job, get_export_job_status, get_export_job_results |
| **audit** | 1 | get_audit_log |

---

## Competitive Analysis

| Server | Tools | Language | Status |
|--------|-------|----------|--------|
| **k-jarzyna/mcp-miro** | **119** | TypeScript | Most comprehensive |
| **Ours** | **58** | Go | Fast, unique features |
| **LuotoCompany/mcp-server-miro** | ~40 | TypeScript | Experimental |
| **Miro Official** | ~5 | Cloud | Beta, limited |
| **evalstate/mcp-miro** | 3 | TypeScript | Very limited |

**Our unique advantages:**
- Single binary (no Node.js required)
- Performance optimizations (caching, circuit breaker, rate limiting)
- Mermaid diagram generation with auto-layout
- Voice-optimized tool descriptions
- Local audit logging

---

## What's Still Missing (vs k-jarzyna)

| Category | Tools | Priority | Notes |
|----------|-------|----------|-------|
| **Mindmaps** | get, list, delete | Medium | API is experimental |
| **Frames** | get, update, delete, get_items | Medium | Basic frame support exists |
| **Images** | get, update, delete | Low | Basic create exists |
| **Shapes** | get, update, delete | Low | Basic create exists |
| **Stickies** | get, update, delete | Low | Generic item ops work |
| **Organization** | org management | Low | Enterprise only |

---

## Test Board

- **URL**: https://miro.com/app/board/uXjVOXQCe5c=
- **Name**: "All tests"
- **Board ID**: `uXjVOXQCe5c=`

---

## MCP Configuration

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

## Quick Commands

```bash
# Build
go build -o miro-mcp-server .

# Test
go test ./...

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
gh release create v1.7.0 dist/* --title "v1.7.0" --notes "..."
```

---

## Architecture Overview

```
miro-mcp-server/
├── main.go                    # Entry point, transport setup
├── README.md                  # Quick start
├── SETUP.md                   # Detailed IDE setup guides
├── HANDOVER.md                # This file
├── TESTING.md                 # Test status and results
│
├── miro/
│   ├── client.go              # HTTP client + rate limiter + circuit breaker
│   ├── interfaces.go          # MiroClient interface + all service interfaces
│   │
│   ├── boards.go              # Board operations
│   ├── items.go               # Item CRUD
│   ├── create.go              # Create operations (sticky, shape, etc.)
│   ├── tags.go                # Tag operations
│   ├── groups.go              # Group operations
│   ├── members.go             # Member operations
│   ├── mindmaps.go            # Mindmap operations (v2-experimental)
│   ├── appcards.go            # App card operations (NEW)
│   ├── export.go              # Export operations
│   │
│   ├── types_*.go             # Type definitions per domain
│   │
│   ├── audit/                 # Audit logging
│   ├── oauth/                 # OAuth 2.1 PKCE
│   └── diagrams/              # Mermaid parser + layout
│
└── tools/
    ├── definitions.go         # 58 tool specs
    ├── definitions_test.go    # Tool validation tests
    ├── handlers.go            # Generic handler registration
    ├── handlers_test.go       # Handler unit tests
    └── mock_client_test.go    # MockClient for testing
```

---

## Known Issues

| Tool | Issue | Status |
|------|-------|--------|
| `miro_get_board_picture` | Returns empty | May need board activity |
| `miro_copy_board` | 500 on complex boards | Miro API limitation |
| Export tools | Not tested | Require Enterprise plan |

---

## Next Session Suggestions

### Priority 1: Complete Core CRUD (Close Gap with k-jarzyna)

**Frame Tools (+4 tools → 62 total)**:
```go
func (c *Client) GetFrame(ctx context.Context, args GetFrameArgs) (GetFrameResult, error)
func (c *Client) UpdateFrame(ctx context.Context, args UpdateFrameArgs) (UpdateFrameResult, error)
func (c *Client) DeleteFrame(ctx context.Context, args DeleteFrameArgs) (DeleteFrameResult, error)
func (c *Client) GetFrameItems(ctx context.Context, args GetFrameItemsArgs) (GetFrameItemsResult, error)
```

**Mindmap Tools (+3 tools → 65 total)**:
```go
// Use v2-experimental endpoint
func (c *Client) GetMindmapNode(ctx context.Context, args GetMindmapNodeArgs) (GetMindmapNodeResult, error)
func (c *Client) ListMindmapNodes(ctx context.Context, args ListMindmapNodesArgs) (ListMindmapNodesResult, error)
func (c *Client) DeleteMindmapNode(ctx context.Context, args DeleteMindmapNodeArgs) (DeleteMindmapNodeResult, error)
```

### Priority 2: New Miro APIs (Differentiation)

**Data Tables API (+5 tools → 70 total)** - New Miro feature (5 months old):
- `miro_create_table`
- `miro_get_table`
- `miro_update_table`
- `miro_delete_table`
- `miro_list_tables`

### Priority 3: Improve Distribution

- **Homebrew tap**: `brew install miro-mcp-server`
- **Docker image**: For containerized deployment
- **Windows installer**: Or better PowerShell script

### Priority 4: Complete Testing

Run through remaining untested tools:
- `miro_create_board`, `miro_copy_board`, `miro_delete_board`
- All v1.6.0 tools (groups, members, app cards)
- Export tools (need Enterprise account)

### Competitive Target

| Current | With Frames+Mindmaps | With Data Tables |
|---------|---------------------|------------------|
| 58 tools | 65 tools | 70 tools |

k-jarzyna has 80 tools, but many are Enterprise-only. At 70 tools we'd be competitive for most users.

---

## Session Summary

### This Session (Documentation Update)
✅ Completed competitive analysis (vs k-jarzyna, Miro Official, others)
✅ Fixed README.md - updated from 46 to 58 tools, reorganized categories
✅ Fixed SETUP.md - corrected misleading n8n section
✅ Updated CLAUDE.md - correct tool count
✅ Created prioritized roadmap to reach 70 tools
✅ Documented gap analysis and competitive position

### Previous Session (v1.6.1)
✅ Fixed mindmap request body structure (400 error)
✅ Released v1.6.1

### Session Before (v1.6.0)
✅ Added 12 new tools (boards, members, groups, app cards)
✅ Fixed mindmap API 405 error (v2-experimental + correct path)
✅ Created SETUP.md documentation
✅ Released v1.6.0

**Total tools: 58**

**Competitive Status**:
- k-jarzyna: 80 tools (TypeScript, 59 stars)
- **Us: 58 tools** (Go, single binary, Mermaid diagrams)
- Miro Official: ~5 tools (cloud-only, beta)

---

**Ready for next session!**

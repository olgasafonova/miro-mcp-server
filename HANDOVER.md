# Session Handover - Miro MCP Server

> **Date**: 2025-12-22
> **Project**: miro-mcp-server
> **Location**: `/Users/olgasafonova/go/src/miro-mcp-server`
> **Version**: v1.7.0
> **Repo**: https://github.com/olgasafonova/miro-mcp-server
> **Release**: https://github.com/olgasafonova/miro-mcp-server/releases/tag/v1.7.0

---

## Current State

**66 MCP tools** for Miro whiteboard control. Build passes, all tests pass.

```bash
# Verify build
cd /Users/olgasafonova/go/src/miro-mcp-server
go build -o miro-mcp-server .
go test ./...
```

**MCP is configured in Claude Code** at user level with correct team_id.

---

## Version History

### v1.7.0 - Frame & Mindmap Tools + Distribution (Current)

**New Tools (+7, total 66)**:
- `miro_get_frame` - Get frame details
- `miro_update_frame` - Update frame title/color/size
- `miro_delete_frame` - Delete a frame
- `miro_get_frame_items` - List items inside a frame
- `miro_get_mindmap_node` - Get node details
- `miro_list_mindmap_nodes` - List all mindmap nodes
- `miro_delete_mindmap_node` - Delete a mindmap node

**Distribution**:
- Homebrew tap: `brew tap olgasafonova/tap && brew install miro-mcp-server`
- Docker: `ghcr.io/olgasafonova/miro-mcp-server:latest`
- Install script: `curl -fsSL https://...install.sh | sh`
- Linux ARM64 binary added

### v1.6.1 - Mindmap Request Body Fix

Fixed `miro_create_mindmap_node` 400 error - content must go in `data.nodeView.data.content`.

### v1.6.0 - 12 New Tools

Added: `miro_update_board`, member tools (get, remove, update), group tools (list, get, get_items, delete), app card tools (create, get, update, delete).

Fixed mindmap API 405 error (v2-experimental endpoint).

---

## Tool Categories (66 Total)

| Category | Count | Tools |
|----------|-------|-------|
| **Boards** | 8 | list, find, get, create, copy, update, delete, get_summary |
| **Members** | 5 | list, get, share, update, remove |
| **Create** | 14 | sticky, sticky_grid, shape, text, frame, card, app_card, image, document, embed, connector, group, mindmap_node, bulk |
| **Frames** | 4 | get, update, delete, get_items |
| **Mindmaps** | 4 | create, get, list, delete |
| **Read** | 5 | list_items, list_all_items, get_item, get_app_card, search |
| **Update** | 5 | update_item, update_app_card, update_connector, update_tag, update_frame |
| **Delete** | 6 | delete_item, delete_app_card, delete_connector, delete_tag, delete_frame, delete_group |
| **Tags** | 6 | create, list, get, attach, detach, get_item_tags |
| **Connectors** | 4 | create, list, get, update, delete |
| **Groups** | 5 | create, ungroup, list, get, get_items |
| **Export** | 4 | get_board_picture, create_export_job, get_export_job_status, get_export_job_results |
| **Diagrams** | 1 | generate_diagram |
| **Audit** | 1 | get_audit_log |

---

## Competitive Analysis

| Server | Tools | Language | Our Advantage |
|--------|-------|----------|---------------|
| **k-jarzyna/mcp-miro** | 81 | TypeScript | Single binary, Mermaid diagrams, no Node.js, rate limiting, caching |
| **Miro Official** | ~10 | Cloud | Self-hosted, 66 tools, works offline, full control |
| **LuotoCompany** | ~15 | TypeScript | More comprehensive, production-ready |

**Our Unique Features**:
- Single Go binary (~10MB, no dependencies)
- Mermaid diagram generation (flowcharts + sequence diagrams)
- Rate limiting with exponential backoff
- Response caching (2-min TTL)
- Circuit breaker for failing endpoints
- Voice-optimized tool descriptions
- Local audit logging
- OAuth 2.1 with PKCE

---

## Supported Platforms

| Platform | Binary | Status |
|----------|--------|--------|
| macOS (Apple Silicon) | `miro-mcp-server-darwin-arm64` | Tested |
| macOS (Intel) | `miro-mcp-server-darwin-amd64` | Tested |
| Linux (x64) | `miro-mcp-server-linux-amd64` | Tested |
| Linux (ARM64) | `miro-mcp-server-linux-arm64` | New in v1.7.0 |
| Windows (x64) | `miro-mcp-server-windows-amd64.exe` | Tested |
| Docker | `ghcr.io/olgasafonova/miro-mcp-server` | Available |

## Supported AI Tools

| Tool | Status | Config |
|------|--------|--------|
| Claude Code | Tested | `claude mcp add miro -- miro-mcp-server` |
| Claude Desktop | Tested | JSON config |
| Cursor | Tested | JSON config |
| VS Code + Copilot | Supported | MCP extension |
| Windsurf | Supported | JSON config |
| Replit | Supported | Download binary |

**Note**: n8n has a native Miro integration and doesn't use MCP.

---

## Test Board

- **URL**: https://miro.com/app/board/uXjVOXQCe5c=
- **Name**: "All tests"
- **Board ID**: `uXjVOXQCe5c=`

---

## Architecture

```
miro-mcp-server/
├── main.go                    # Entry point, transport setup
├── miro/
│   ├── client.go              # HTTP client + rate limiter + circuit breaker
│   ├── interfaces.go          # MiroClient interface + all service interfaces
│   ├── boards.go, items.go, create.go, tags.go, groups.go, members.go
│   ├── mindmaps.go, frames.go, export.go
│   ├── types_*.go             # Type definitions per domain
│   ├── audit/                 # Audit logging
│   ├── oauth/                 # OAuth 2.1 PKCE
│   └── diagrams/              # Mermaid parser + layout
└── tools/
    ├── definitions.go         # 66 tool specs
    └── handlers.go            # Generic handler registration
```

---

## Known Issues

| Issue | Status | Notes |
|-------|--------|-------|
| `miro_get_board_picture` returns empty | Open | May need board activity |
| `miro_copy_board` 500 on complex boards | Open | Miro API limitation |
| Export tools untested | Open | Require Enterprise plan |

---

## Gap Analysis (vs k-jarzyna 81 tools)

| Missing | Priority | Notes |
|---------|----------|-------|
| Comments API (+3) | Medium | create, list, delete |
| Item-specific CRUD | Low | Generic update_item works |
| Organization tools | Low | Enterprise only |
| Teams/Projects | Low | Enterprise only |

---

## Quick Commands

```bash
# Build
go build -o miro-mcp-server .

# Test
go test ./...

# Run
MIRO_ACCESS_TOKEN=xxx ./miro-mcp-server

# Build all platforms
GOOS=darwin GOARCH=arm64 go build -o dist/miro-mcp-server-darwin-arm64 .
GOOS=darwin GOARCH=amd64 go build -o dist/miro-mcp-server-darwin-amd64 .
GOOS=linux GOARCH=amd64 go build -o dist/miro-mcp-server-linux-amd64 .
GOOS=linux GOARCH=arm64 go build -o dist/miro-mcp-server-linux-arm64 .
GOOS=windows GOARCH=amd64 go build -o dist/miro-mcp-server-windows-amd64.exe .
```

---

## This Session Summary (Dec 22, 2025 - continued)

**New Tool (+1, total 66):**
- `miro_get_tag` - Get tag details by ID (was implemented internally, now exposed)

**New Files:**
- `.github/workflows/ci.yml` - CI workflow (tests, lint, build on PRs)
- `QUICKSTART.md` - 60-second setup guide

**All docs updated to 66 tools.**

---

## Awesome MCP Servers PR

To add to [punkpeye/awesome-mcp-servers](https://github.com/punkpeye/awesome-mcp-servers), submit PR with:

```markdown
- [olgasafonova/miro-mcp-server](https://github.com/olgasafonova/miro-mcp-server) 🏎️ ☁️ - Miro whiteboard control (66 tools). Single Go binary, Mermaid diagram generation, rate limiting, caching.
```

Add under "Whiteboard & Design" or similar category, after existing Miro entries.

---

## Next Session Suggestions

1. **Submit to Awesome MCP Servers** - Use PR format above
2. **Add Comments API** - Research if Miro v2 has endpoints
3. **More test coverage** - Currently 40/66 tools tested
4. **Docker image verification** - Test ghcr.io image works
5. **Tag v1.7.1 release** - Include new get_tag tool

**Completed This Session:**
- Added miro_get_tag tool (66 total)
- Added GitHub Actions CI workflow
- Created QUICKSTART.md

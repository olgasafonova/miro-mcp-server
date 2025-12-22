# Miro MCP Server

Control Miro whiteboards with AI. Built in Go for speed and simplicity.

**66 tools** | **Single binary** | **All platforms** | **All major AI tools**

[![CI](https://github.com/olgasafonova/miro-mcp-server/actions/workflows/ci.yml/badge.svg)](https://github.com/olgasafonova/miro-mcp-server/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/olgasafonova/miro-mcp-server)](https://goreportcard.com/report/github.com/olgasafonova/miro-mcp-server)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## Documentation

| Document | Description |
|----------|-------------|
| [QUICKSTART.md](QUICKSTART.md) | Get running in 2 minutes |
| [SETUP.md](SETUP.md) | Full setup for all AI tools |
| [CONFIG.md](CONFIG.md) | Configuration reference |
| [COMPARISON.md](COMPARISON.md) | Competitive analysis |
| [PERFORMANCE.md](PERFORMANCE.md) | Optimization guide |
| [ROADMAP.md](ROADMAP.md) | Development roadmap |
| [CHANGELOG.md](CHANGELOG.md) | Version history |

---

## Quick Start

### 1. Install

**Homebrew (macOS/Linux):**
```bash
brew tap olgasafonova/tap && brew install miro-mcp-server
```

**One-liner (macOS/Linux):**
```bash
curl -fsSL https://raw.githubusercontent.com/olgasafonova/miro-mcp-server/main/install.sh | sh
```

**Docker:**
```bash
docker pull ghcr.io/olgasafonova/miro-mcp-server:latest
```

**Manual download:** See [SETUP.md](SETUP.md) for all platforms

### 2. Get a Miro Token

1. Go to [miro.com/app/settings/user-profile/apps](https://miro.com/app/settings/user-profile/apps)
2. Create an app with `boards:read` and `boards:write` permissions
3. Install to your team and copy the token

### 3. Configure Your AI Tool

**Claude Code:**
```bash
claude mcp add miro -e MIRO_ACCESS_TOKEN=your-token -- miro-mcp-server
```

**Claude Desktop / Cursor / VS Code:** See [SETUP.md](SETUP.md)

---

## What You Can Do

| Category | Examples |
|----------|----------|
| **Boards** | Create, copy, delete, update, share, list members |
| **Items** | Sticky notes, shapes, text, cards, app cards, frames |
| **Diagrams** | Generate flowcharts and sequence diagrams from Mermaid |
| **Mindmaps** | Create mindmap nodes with parent-child relationships |
| **Bulk Ops** | Create multiple items at once, sticky grids |
| **Tags** | Create, attach, update, and organize with tags |
| **Groups** | Group, ungroup, list, and manage item groups |
| **Connectors** | Connect items with styled arrows |
| **Export** | Board thumbnails, PDF/SVG export (Enterprise) |

### Voice Examples

- *"Add a yellow sticky saying 'Review PRs'"*
- *"Create a flowchart: Start → Decision → End"*
- *"What boards do I have?"*
- *"Share the Design board with jane@example.com"*
- *"Create a mindmap with 'Project Ideas' as root"*

---

## All 66 Tools

<details>
<summary><b>Board Management (8)</b></summary>

| Tool | Description |
|------|-------------|
| `miro_list_boards` | List accessible boards |
| `miro_find_board` | Find board by name |
| `miro_get_board` | Get board details |
| `miro_get_board_summary` | Get board stats and item counts |
| `miro_create_board` | Create a new board |
| `miro_copy_board` | Copy an existing board |
| `miro_update_board` | Update board name/description |
| `miro_delete_board` | Delete a board |

</details>

<details>
<summary><b>Board Members (5)</b></summary>

| Tool | Description |
|------|-------------|
| `miro_list_board_members` | List users with access |
| `miro_get_board_member` | Get member details |
| `miro_share_board` | Share board via email |
| `miro_update_board_member` | Update member role |
| `miro_remove_board_member` | Remove member from board |

</details>

<details>
<summary><b>Create Items (14)</b></summary>

| Tool | Description |
|------|-------------|
| `miro_create_sticky` | Create a sticky note |
| `miro_create_sticky_grid` | Create stickies in a grid layout |
| `miro_create_shape` | Create a shape (rectangle, circle, etc.) |
| `miro_create_text` | Create text element |
| `miro_create_frame` | Create a frame container |
| `miro_create_card` | Create a card with due date |
| `miro_create_app_card` | Create app card with custom fields |
| `miro_create_image` | Add image from URL |
| `miro_create_document` | Add document from URL |
| `miro_create_embed` | Embed YouTube, Figma, etc. |
| `miro_create_connector` | Connect two items with arrow |
| `miro_create_group` | Group items together |
| `miro_create_mindmap_node` | Create mindmap node |
| `miro_bulk_create` | Create multiple items at once |

</details>

<details>
<summary><b>Frames (4)</b></summary>

| Tool | Description |
|------|-------------|
| `miro_get_frame` | Get frame details |
| `miro_update_frame` | Update frame title/color/size |
| `miro_delete_frame` | Delete a frame |
| `miro_get_frame_items` | List items inside a frame |

</details>

<details>
<summary><b>Mindmaps (4)</b></summary>

| Tool | Description |
|------|-------------|
| `miro_create_mindmap_node` | Create mindmap node |
| `miro_get_mindmap_node` | Get node details |
| `miro_list_mindmap_nodes` | List all mindmap nodes |
| `miro_delete_mindmap_node` | Delete a mindmap node |

</details>

<details>
<summary><b>Read Items (5)</b></summary>

| Tool | Description |
|------|-------------|
| `miro_list_items` | List items on a board |
| `miro_list_all_items` | Get ALL items with auto-pagination |
| `miro_get_item` | Get item details |
| `miro_get_app_card` | Get app card details |
| `miro_search_board` | Search items by content |

</details>

<details>
<summary><b>Update & Delete Items (4)</b></summary>

| Tool | Description |
|------|-------------|
| `miro_update_item` | Update item content/position/color |
| `miro_update_app_card` | Update app card fields |
| `miro_delete_item` | Delete an item |
| `miro_delete_app_card` | Delete an app card |

</details>

<details>
<summary><b>Tags (8)</b></summary>

| Tool | Description |
|------|-------------|
| `miro_create_tag` | Create a tag |
| `miro_list_tags` | List all tags on board |
| `miro_get_tag` | Get tag details by ID |
| `miro_attach_tag` | Attach tag to item |
| `miro_detach_tag` | Remove tag from item |
| `miro_get_item_tags` | Get tags on an item |
| `miro_update_tag` | Update tag name/color |
| `miro_delete_tag` | Delete a tag |

</details>

<details>
<summary><b>Connectors (4)</b></summary>

| Tool | Description |
|------|-------------|
| `miro_list_connectors` | List all connectors |
| `miro_get_connector` | Get connector details |
| `miro_update_connector` | Update connector style/caption |
| `miro_delete_connector` | Delete a connector |

</details>

<details>
<summary><b>Groups (5)</b></summary>

| Tool | Description |
|------|-------------|
| `miro_list_groups` | List all groups on board |
| `miro_get_group` | Get group details |
| `miro_get_group_items` | List items in a group |
| `miro_ungroup` | Ungroup items |
| `miro_delete_group` | Delete a group |

</details>

<details>
<summary><b>Export (4)</b></summary>

| Tool | Description |
|------|-------------|
| `miro_get_board_picture` | Get board thumbnail |
| `miro_create_export_job` | Export to PDF/SVG (Enterprise) |
| `miro_get_export_job_status` | Check export progress |
| `miro_get_export_job_results` | Get download links |

</details>

<details>
<summary><b>Diagrams & Audit (2)</b></summary>

| Tool | Description |
|------|-------------|
| `miro_generate_diagram` | Create diagram from Mermaid syntax |
| `miro_get_audit_log` | Query local execution log |

</details>

---

## Diagram Generation

Create flowcharts and sequence diagrams from Mermaid syntax:

**Flowchart:**
```
flowchart TB
    A[Start] --> B{Decision}
    B -->|Yes| C[Success]
    B -->|No| D[Retry]
    D --> B
```

**Sequence Diagram:**
```
sequenceDiagram
    Alice->>Bob: Hello Bob!
    Bob-->>Alice: Hi Alice!
```

**Supported:** `flowchart`/`graph`, `sequenceDiagram`, directions (TB/LR/BT/RL), shapes (`[]` rectangle, `{}` diamond, `(())` circle), labeled edges.

---

## Why This Server?

| Feature | This Server | TypeScript alternatives |
|---------|-------------|------------------------|
| **Runtime** | Single binary | Requires Node.js |
| **Size** | ~14MB | 100MB+ with node_modules |
| **Startup** | ~50ms | 500ms-2s |
| **Memory** | ~10MB idle | ~50MB idle |
| **Diagram generation** | Built-in Mermaid parser | Not available |
| **Rate limiting** | Automatic with backoff | Manual |
| **Caching** | 2-minute TTL | None |
| **Circuit breaker** | Yes | No |

See [COMPARISON.md](COMPARISON.md) for detailed competitive analysis.

---

## Performance

- **Caching:** 2-minute TTL reduces API calls
- **Rate limiting:** Adapts to Miro's rate limit headers
- **Circuit breaker:** Isolates failing endpoints
- **Parallel bulk ops:** Creates items concurrently
- **Token validation:** Fails fast on startup with clear error

See [PERFORMANCE.md](PERFORMANCE.md) for optimization tips and benchmarks.

---

## Supported Platforms

| Platform | Binary |
|----------|--------|
| macOS (Apple Silicon) | `miro-mcp-server-darwin-arm64` |
| macOS (Intel) | `miro-mcp-server-darwin-amd64` |
| Linux (x64) | `miro-mcp-server-linux-amd64` |
| Linux (ARM64) | `miro-mcp-server-linux-arm64` |
| Windows (x64) | `miro-mcp-server-windows-amd64.exe` |
| Docker | `ghcr.io/olgasafonova/miro-mcp-server` |

---

## Supported AI Tools

| Tool | Status |
|------|--------|
| Claude Code | Tested |
| Claude Desktop | Tested |
| Cursor | Tested |
| VS Code + GitHub Copilot | Supported |
| Windsurf | Supported |
| Replit | Supported |
| Any MCP-compatible client | Supported |

See [SETUP.md](SETUP.md) for configuration guides.

---

## Account Compatibility

| Account Type | Support |
|--------------|---------|
| Free | Full access to all 66 tools |
| Team | Full access to all 66 tools |
| Business | Full access to all 66 tools |
| Enterprise | Full access + export to PDF/SVG |

---

## License

MIT

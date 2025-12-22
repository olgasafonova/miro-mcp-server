# Miro MCP Server

Control Miro whiteboards with AI. Built in Go for speed and simplicity.

**46 tools** | **Single binary** | **All platforms** | **All major AI tools**

---

## Quick Start

### 1. Download

**macOS (Apple Silicon):**
```bash
curl -L -o miro-mcp-server https://github.com/olgasafonova/miro-mcp-server/releases/latest/download/miro-mcp-server-darwin-arm64
chmod +x miro-mcp-server
sudo mv miro-mcp-server /usr/local/bin/
```

**Other platforms:** See [SETUP.md](SETUP.md)

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
| **Boards** | Create, copy, delete, share, list members |
| **Items** | Sticky notes, shapes, text, cards, frames |
| **Diagrams** | Generate flowcharts from Mermaid syntax |
| **Bulk Ops** | Create multiple items at once, sticky grids |
| **Tags** | Create, attach, and organize with tags |
| **Groups** | Group and ungroup items |
| **Connectors** | Connect items with arrows |
| **Export** | Board thumbnails, PDF/SVG (Enterprise) |

### Voice Examples

- *"Add a yellow sticky saying 'Review PRs'"*
- *"Create a flowchart: Start → Decision → End"*
- *"What boards do I have?"*
- *"Share the Design board with jane@example.com"*

---

## All 46 Tools

<details>
<summary><b>Board Tools (9)</b></summary>

| Tool | Description |
|------|-------------|
| `miro_list_boards` | List accessible boards |
| `miro_get_board` | Get board details |
| `miro_create_board` | Create a new board |
| `miro_copy_board` | Copy an existing board |
| `miro_delete_board` | Delete a board |
| `miro_find_board` | Find board by name |
| `miro_get_board_summary` | Get board stats and item counts |
| `miro_share_board` | Share board via email |
| `miro_list_board_members` | List users with access |

</details>

<details>
<summary><b>Create Tools (13)</b></summary>

| Tool | Description |
|------|-------------|
| `miro_create_sticky` | Create a sticky note |
| `miro_create_sticky_grid` | Create stickies in a grid |
| `miro_create_shape` | Create a shape |
| `miro_create_text` | Create text |
| `miro_create_connector` | Connect two items |
| `miro_create_frame` | Create a frame container |
| `miro_create_card` | Create a card with due date |
| `miro_create_image` | Add image from URL |
| `miro_create_document` | Add document from URL |
| `miro_create_embed` | Embed YouTube, Figma, etc. |
| `miro_bulk_create` | Create multiple items |
| `miro_create_group` | Group items together |
| `miro_create_mindmap_node` | Create mindmap node |

</details>

<details>
<summary><b>Read Tools (4)</b></summary>

| Tool | Description |
|------|-------------|
| `miro_list_items` | List items on a board |
| `miro_list_all_items` | Get ALL items (auto-pagination) |
| `miro_get_item` | Get item details |
| `miro_search_board` | Search items by content |

</details>

<details>
<summary><b>Tag Tools (7)</b></summary>

| Tool | Description |
|------|-------------|
| `miro_create_tag` | Create a tag |
| `miro_list_tags` | List all tags |
| `miro_attach_tag` | Attach tag to item |
| `miro_detach_tag` | Remove tag from item |
| `miro_get_item_tags` | Get tags on an item |
| `miro_update_tag` | Update tag name/color |
| `miro_delete_tag` | Delete a tag |

</details>

<details>
<summary><b>Connector Tools (4)</b></summary>

| Tool | Description |
|------|-------------|
| `miro_list_connectors` | List all connectors |
| `miro_get_connector` | Get connector details |
| `miro_update_connector` | Update connector style |
| `miro_delete_connector` | Delete a connector |

</details>

<details>
<summary><b>Modify Tools (3)</b></summary>

| Tool | Description |
|------|-------------|
| `miro_update_item` | Update item content/position |
| `miro_delete_item` | Delete an item |
| `miro_ungroup` | Ungroup items |

</details>

<details>
<summary><b>Export Tools (4)</b></summary>

| Tool | Description |
|------|-------------|
| `miro_get_board_picture` | Get board thumbnail |
| `miro_create_export_job` | Export to PDF/SVG (Enterprise) |
| `miro_get_export_job_status` | Check export progress |
| `miro_get_export_job_results` | Get download links |

</details>

<details>
<summary><b>Diagram & Audit Tools (2)</b></summary>

| Tool | Description |
|------|-------------|
| `miro_generate_diagram` | Create flowchart from Mermaid |
| `miro_get_audit_log` | Query local execution log |

</details>

---

## Diagram Generation

Create flowcharts from Mermaid syntax:

```
flowchart TB
    A[Start] --> B{Decision}
    B -->|Yes| C[Success]
    B -->|No| D[Retry]
    D --> B
```

**Supported:** `flowchart`/`graph`, directions (TB/LR/BT/RL), shapes (`[]` rectangle, `{}` diamond, `(())` circle), labeled edges.

---

## Performance

- **Caching:** 2-minute TTL reduces API calls
- **Rate limiting:** Adapts to Miro's rate limit headers
- **Circuit breaker:** Isolates failing endpoints
- **Parallel bulk ops:** Creates items concurrently

---

## Supported Platforms

| Platform | Binary |
|----------|--------|
| macOS (Apple Silicon) | `miro-mcp-server-darwin-arm64` |
| macOS (Intel) | `miro-mcp-server-darwin-amd64` |
| Linux (x64) | `miro-mcp-server-linux-amd64` |
| Windows (x64) | `miro-mcp-server-windows-amd64.exe` |

---

## Supported AI Tools

- Claude Code
- Claude Desktop
- Cursor
- VS Code + GitHub Copilot
- Windsurf
- Replit
- Any MCP-compatible client

See [SETUP.md](SETUP.md) for configuration guides.

---

## License

MIT

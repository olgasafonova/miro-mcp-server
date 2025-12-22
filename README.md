# Miro MCP Server

A Model Context Protocol (MCP) server for controlling Miro whiteboards with AI assistants. Built in Go for performance and single-binary deployment.

## Features

- **46 tools** for complete Miro control
- **AI Diagram Generation**: Create flowcharts from Mermaid syntax with auto-layout
- **Board management**: Create, copy, delete, find by name, share with users
- **Create items**: Sticky notes, shapes, text, connectors, frames, cards, images, documents, embeds, mindmap nodes
- **Bulk operations**: Create multiple items at once, sticky grids
- **Groups**: Group and ungroup items
- **Tags**: Create, attach, and manage tags
- **Export**: Board pictures (all plans) and PDF/SVG/HTML export (Enterprise)
- **Audit logging**: Track all tool executions
- **OAuth 2.1**: PKCE flow with auto-refresh
- **Token validation**: Fails fast with clear error if token is invalid
- **Rate limiting**: Semaphore-based (5 concurrent requests)
- **Caching**: 2-minute TTL for board data
- **Retry with backoff**: Handles rate limits gracefully
- **Dual transport**: stdio (default) + HTTP
- **Voice-optimized**: Short, speakable responses

## Installation

### Option 1: Download Pre-built Binary (Recommended)

Download the latest release for your platform from [GitHub Releases](https://github.com/olgasafonova/miro-mcp-server/releases):

| Platform | Binary |
|----------|--------|
| macOS (Apple Silicon) | `miro-mcp-server-darwin-arm64` |
| macOS (Intel) | `miro-mcp-server-darwin-amd64` |
| Linux | `miro-mcp-server-linux-amd64` |
| Windows | `miro-mcp-server-windows-amd64.exe` |

```bash
# macOS/Linux: Make executable after download
chmod +x miro-mcp-server-*

# Move to a location in your PATH (optional)
sudo mv miro-mcp-server-darwin-arm64 /usr/local/bin/miro-mcp-server
```

### Option 2: Build from Source

Requires Go 1.21 or later.

```bash
git clone https://github.com/olgasafonova/miro-mcp-server.git
cd miro-mcp-server
go build -o miro-mcp-server .
```

### Option 3: Go Install

```bash
go install github.com/olgasafonova/miro-mcp-server@latest
```

## Quick Start

### 1. Get a Miro Access Token

1. Go to [Miro Developer Settings](https://miro.com/app/settings/user-profile/apps)
2. Create a new app or use an existing one
3. Install the app to your team with required scopes:
   - `boards:read` - Read board data
   - `boards:write` - Create and modify items
   - `boards:export` - Export boards (Enterprise only)
4. Copy the access token

### 2. Run

**Stdio mode** (for Claude Desktop, Cursor):
```bash
MIRO_ACCESS_TOKEN="your-token" ./miro-mcp-server
```

**HTTP mode** (for remote clients):
```bash
MIRO_ACCESS_TOKEN="your-token" ./miro-mcp-server -http :8080
```

## Claude Desktop Configuration

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "miro": {
      "command": "/path/to/miro-mcp-server",
      "env": {
        "MIRO_ACCESS_TOKEN": "your-token-here"
      }
    }
  }
}
```

## Available Tools (46 total)

### Board Tools
| Tool | Description |
|------|-------------|
| `miro_list_boards` | List accessible boards |
| `miro_get_board` | Get board details |
| `miro_create_board` | Create a new board |
| `miro_copy_board` | Copy an existing board |
| `miro_delete_board` | Delete a board (destructive) |
| `miro_find_board` | Find board by name (fuzzy match) |
| `miro_get_board_summary` | Get board with item counts and stats |
| `miro_share_board` | Share a board with someone by email |
| `miro_list_board_members` | List users with access to a board |

### Create Tools
| Tool | Description |
|------|-------------|
| `miro_create_sticky` | Create a sticky note |
| `miro_create_shape` | Create a shape (rectangle, circle, etc.) |
| `miro_create_text` | Create a text element |
| `miro_create_connector` | Connect two items with a line |
| `miro_create_frame` | Create a frame container |
| `miro_create_card` | Create a card with title, description, due date |
| `miro_create_image` | Add an image from URL |
| `miro_create_document` | Add a document (PDF, etc.) from URL |
| `miro_create_embed` | Embed external content (YouTube, Figma, etc.) |
| `miro_bulk_create` | Create multiple items at once |
| `miro_create_sticky_grid` | Create stickies in a grid layout |
| `miro_create_group` | Group multiple items together |
| `miro_create_mindmap_node` | Create a mindmap node (root or child) |

### Read Tools
| Tool | Description |
|------|-------------|
| `miro_list_items` | List items on a board (paginated) |
| `miro_list_all_items` | Retrieve ALL items with automatic pagination |
| `miro_get_item` | Get full details of a specific item |
| `miro_search_board` | Search for items by content |

### Tag Tools
| Tool | Description |
|------|-------------|
| `miro_create_tag` | Create a new tag on a board |
| `miro_list_tags` | List all tags on a board |
| `miro_attach_tag` | Attach a tag to a sticky note |
| `miro_detach_tag` | Remove a tag from a sticky note |
| `miro_get_item_tags` | List tags on a specific item |

### Modify Tools
| Tool | Description |
|------|-------------|
| `miro_update_item` | Update an item's content or position |
| `miro_delete_item` | Delete an item (destructive) |
| `miro_ungroup` | Ungroup items (release from a group) |

### Export Tools
| Tool | Description |
|------|-------------|
| `miro_get_board_picture` | Get board thumbnail image URL (all plans) |
| `miro_create_export_job` | Create PDF/SVG/HTML export job (Enterprise) |
| `miro_get_export_job_status` | Check export job progress (Enterprise) |
| `miro_get_export_job_results` | Get download links for exported boards (Enterprise) |

### Diagram Tools
| Tool | Description |
|------|-------------|
| `miro_generate_diagram` | Create flowcharts from Mermaid syntax with auto-layout |

### Webhook Tools (Removed)

> ⚠️ **Webhook tools have been removed.** Miro is [discontinuing experimental webhooks](https://community.miro.com/developer-platform-and-apis-57/miro-webhooks-4281) on December 5, 2025. The `/v2-experimental/webhooks/board_subscriptions` endpoints no longer function reliably.

### Audit Tools
| Tool | Description |
|------|-------------|
| `miro_get_audit_log` | Query local audit log of tool executions |

## Example Usage

### Voice Commands
- "Add a yellow sticky saying 'Review PRs'"
- "Create 5 stickies for our action items"
- "Draw a rectangle for the header"
- "Connect the first box to the second"
- "What boards do I have?"

### Programmatic

```
User: Add a sticky note to my Design board saying "MVP feature"
Assistant: [Uses miro_list_boards to find Design board]
         [Uses miro_create_sticky with content "MVP feature"]
         Created yellow sticky "MVP feature" on Design board
```

## Sticky Note Colors

| Color | Name |
|-------|------|
| 🟡 | yellow (default) |
| 🟢 | green |
| 🔵 | blue |
| 🩷 | pink |
| 🟠 | orange |
| 🔴 | red |
| ⚫ | gray |
| 🩵 | cyan |
| 🟣 | purple |

## Shape Types

**Basic**: rectangle, round_rectangle, circle, triangle, rhombus

**Extended**: parallelogram, trapezoid, pentagon, hexagon, octagon, star

**Flowchart**: flow_chart_predefined_process, wedge_round_rectangle_callout

## Tag Colors

| Color | Name |
|-------|------|
| 🔴 | red |
| 🩷 | magenta |
| 🟣 | violet |
| 🔵 | blue |
| 🩵 | cyan |
| 🟢 | green |
| 🟡 | yellow |
| 🟠 | orange |
| ⚫ | gray |

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `MIRO_ACCESS_TOKEN` | Yes | Your Miro OAuth access token |
| `MIRO_TIMEOUT` | No | Request timeout (default: 30s) |
| `MIRO_USER_AGENT` | No | Custom user agent string |

## AI Diagram Generation

Generate flowcharts from Mermaid syntax with automatic layout:

```
miro_generate_diagram board_id="xxx" diagram="flowchart TB
    A[Start] --> B{Decision}
    B -->|Yes| C[Success]
    B -->|No| D[Retry]
    D --> B"
```

### Supported Mermaid Features

| Feature | Syntax | Example |
|---------|--------|---------|
| **Keywords** | `flowchart`, `graph` | `flowchart TB` |
| **Directions** | TB, LR, BT, RL | `flowchart LR` (left to right) |
| **Rectangle** | `[text]` | `A[Start]` |
| **Diamond** | `{text}` | `B{Decision}` |
| **Circle** | `((text))` | `C((End))` |
| **Stadium** | `(text)` | `D(Process)` |
| **Hexagon** | `{{text}}` | `E{{Prepare}}` |
| **Arrow** | `-->` | `A --> B` |
| **Labeled edge** | `--\|text\|-->` | `A --\|yes\|--> B` |
| **Chain** | `-->` | `A --> B --> C` |
| **Subgraph** | `subgraph...end` | `subgraph Group ... end` |
| **Comment** | `%%` | `%% This is ignored` |

### Diagram Examples

**Simple flow:**
```
flowchart LR
    A[Input] --> B[Process] --> C[Output]
```

**Decision tree:**
```
flowchart TB
    Start[Start] --> Check{Valid?}
    Check -->|Yes| Success[Continue]
    Check -->|No| Error[Handle Error]
    Error --> Start
```

**With subgroups:**
```
flowchart TB
    subgraph Frontend
        A[React] --> B[API Call]
    end
    subgraph Backend
        C[Server] --> D[Database]
    end
    B --> C
```

### Layout Algorithm

The diagram generator uses a Sugiyama-style layered layout:
- **Topological ordering**: Nodes arranged by dependency
- **Layer assignment**: Nodes grouped into horizontal/vertical layers
- **Barycenter ordering**: Minimizes edge crossings
- **Configurable spacing**: Adjust node width, height, and gaps

## Voice Demo Use Case

This server is designed for "manage your Miro with your voice" scenarios:

1. **Brainstorming**: "Add stickies for each of these ideas: A, B, C, D"
2. **Retrospectives**: "Create a frame called 'What went well'"
3. **Diagramming**: "Create a flowchart: Start goes to Process, Process goes to End"
4. **Organization**: "Move all the red stickies to the Done frame"

## License

MIT
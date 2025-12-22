# Session Handover - Miro MCP Server

> **Date**: 2025-12-22 (Session 3)
> **Project**: miro-mcp-server
> **Location**: `/Users/olgasafonova/go/src/miro-mcp-server`
> **Version**: v1.4.2 (with post-release fixes)
> **Latest Session**: Fixed embed geometry bug, tag colors documentation

---

## Current State

**50 MCP tools** for Miro whiteboard control. Build passes, tests pass.

```bash
# Verify build
cd /Users/olgasafonova/go/src/miro-mcp-server
go build -o miro-mcp-server .
go test ./...
```

**MCP is configured in Claude Code** at user level with correct team_id.

---

## What Was Fixed This Session (Session 3)

### Fix 1: CreateEmbed sends both width and height

**Problem**: `miro_create_embed` failed for YouTube videos with:
```
Only height or width should be passed for widgets with fixed aspect ratio
```

**Root Cause**: Code always sent both `width` and `height` in geometry.

**Solution**: Only send width OR height, not both. Let Miro calculate the other dimension for fixed aspect ratio embeds.

**File**: `miro/create.go` lines 735-746
```go
// For embeds with fixed aspect ratio (like YouTube), only send width
// Miro will calculate height automatically. Sending both causes an error.
if args.Width > 0 {
    reqBody["geometry"] = map[string]interface{}{
        "width": args.Width,
    }
} else if args.Height > 0 {
    reqBody["geometry"] = map[string]interface{}{
        "height": args.Height,
    }
}
```

### Fix 2: Tag color "orange" documented but API rejects

**Problem**: Tool description said "orange" is valid tag color, but API returns:
```
Unexpected value [orange], expected one of: [red, light_green, cyan, yellow, magenta, green, blue, gray, violet, dark_gray, dark_green, dark_blue, black]
```

**Solution**: Updated valid colors in documentation.

**Files Changed**:
- `tools/definitions.go` lines 403, 482
- `miro/types_tags.go` lines 22, 115

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

**OAuth tokens** at `~/.miro/tokens.json`:
- `team_id`: `3458764516184293832` (corrected!)
- Access token auto-refreshes

---

## Test Board

**URL**: https://miro.com/app/board/uXjVOXQCe5c=
**Name**: "All tests"
**Board ID**: `uXjVOXQCe5c=`

---

## Testing Progress

### Verified Working (39/50 tools)

**Boards**: list_boards, find_board, get_board, get_board_summary
**Items**: list_items, list_all_items, get_item, search_board, update_item, delete_item
**Create**: create_sticky, create_sticky_grid, create_shape, create_text, create_frame, create_card, create_connector, create_image, bulk_create
**Tags**: create_tag, list_tags, attach_tag, detach_tag, get_item_tags, update_tag, delete_tag
**Groups**: create_group, ungroup
**Connectors**: list_connectors, get_connector, update_connector, delete_connector
**Members**: list_board_members
**Diagrams**: generate_diagram (flowchart), generate_diagram (sequence)
**Audit**: get_audit_log

### Known Issues

| Tool | Issue |
|------|-------|
| `miro_create_mindmap_node` | API returns 405 - endpoint may have changed |
| `miro_get_board_picture` | Returns empty - may need board activity |
| `miro_create_webhook` | "subscription or endpoint does not exist" |
| `miro_list_webhooks` | Same error - may need app setup |

### Still Need Testing

- [ ] `miro_create_board`
- [ ] `miro_copy_board`
- [ ] `miro_delete_board` (destructive)
- [ ] `miro_create_document`
- [ ] `miro_create_embed` (after restart - bug fixed)
- [ ] `miro_share_board` (needs email)
- [ ] Export tools (Enterprise only)

---

## All Fixes Summary

| Version | Date | Issue | Fix |
|---------|------|-------|-----|
| v1.4.2+ | 2025-12-22 | CreateEmbed geometry error | Only send width OR height |
| v1.4.2+ | 2025-12-22 | Tag color "orange" invalid | Updated valid colors list |
| v1.4.2+ | 2025-12-22 | CreateGroup fails | Request body needs `{data:{items:[...]}}` |
| v1.4.2+ | 2025-12-22 | ListConnectors limit < 10 fails | Enforce minimum limit of 10 |
| v1.4.2+ | 2025-12-22 | ListBoards empty | Wrong team_id in tokens.json |
| v1.4.2+ | 2025-12-22 | CreateTag fails without color | Default to "blue" |
| v1.4.2+ | 2025-12-22 | ListItems limit 100 fails | Changed to max 50 |
| v1.4.2+ | 2025-12-22 | GetItemTags returns null | Return empty array `[]` |

---

## Quick Commands

```bash
# Build
go build -o miro-mcp-server .

# Test all
go test ./...

# Run with token
MIRO_ACCESS_TOKEN=xxx MIRO_TEAM_ID=3458764516184293832 ./miro-mcp-server
```

---

## Next Session

1. **Restart Claude Code** to pick up rebuilt binary
2. Test `miro_create_embed` - should work now with geometry fix
3. Test remaining tools: create_board, copy_board, create_document
4. Investigate webhook API issues
5. Consider releasing v1.4.3 with all fixes

---

**Binary rebuilt with fixes - restart Claude Code to continue testing!**

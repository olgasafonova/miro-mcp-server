# Session Handover - Miro MCP Server

> **Date**: 2025-12-22
> **Project**: miro-mcp-server
> **Location**: `/Users/olgasafonova/go/src/miro-mcp-server`
> **Version**: v1.4.2 (with post-release fixes)
> **Latest Session**: Fixed team_id, tag color, and limit bugs

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

## What Was Fixed This Session (2025-12-22)

### Fix 1: Wrong Team ID (ListBoards returning empty)

**Problem**: `miro_list_boards` and `miro_find_board` returned empty results.

**Root Cause**: The `~/.miro/tokens.json` had wrong team_id:
- Wrong: `3458764653228607818`
- Correct: `3458764516184293832` (where the boards actually live)

**Solution**:
1. Updated `~/.miro/tokens.json` with correct team_id
2. Added `MIRO_TEAM_ID` env var to MCP config in `~/.claude.json`

**Files Changed**:
- `~/.miro/tokens.json` - corrected team_id
- `~/.claude.json` - added MIRO_TEAM_ID to miro server env

### Fix 2: CreateTag fails without color

**Problem**: `miro_create_tag` without color parameter failed with:
```
Field [fillColor] of type [String] is required
```

**Solution**: Added default color "blue" when not specified.

**File**: `miro/tags.go` lines 26-35
```go
// Color is required by Miro API, default to "blue" if not specified
color := args.Color
if color == "" {
    color = "blue"
}

reqBody := map[string]interface{}{
    "title":     args.Title,
    "fillColor": normalizeTagColor(color),
}
```

### Fix 3: ListItems limit says max 100 but Miro only allows 50

**Problem**: Tool description said "max 100" but Miro API returns error for limit > 50.

**Solution**:
1. Fixed code to cap at 50: `miro/items.go` line 28
2. Fixed tool description: `tools/definitions.go` line 268

### Fix 4: AttachTag unclear about taggable item types

**Problem**: Claude tried to tag a shape, got API error, then explained limitation.

**Solution**: Made tool description more emphatic about sticky_note/card only.

**File**: `tools/definitions.go` lines 427-436
```
IMPORTANT: Tags can ONLY be attached to sticky_note or card items.
Shapes, text, frames, images, and other item types CANNOT be tagged.
```

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

Current contents (as of this session):
- 5 yellow stars (shapes)
- Several sticky notes with tags
- Various test items from previous sessions

---

## Testing Progress

### Verified Working
- [x] `miro_list_boards` - Returns 3 boards after team_id fix
- [x] `miro_find_board` - Finds "All tests" board
- [x] `miro_get_board` - Gets board details
- [x] `miro_list_items` - Lists items (with correct 50 limit)
- [x] `miro_get_item` - Gets item details
- [x] `miro_create_tag` - Creates tags (with default blue color)
- [x] `miro_attach_tag` - Attaches tags to sticky notes
- [x] `miro_create_sticky` - Creates sticky notes
- [x] `miro_bulk_create` - Creates multiple items

### Still Need Testing
- [ ] `miro_get_board_summary`
- [ ] `miro_create_board`
- [ ] `miro_copy_board`
- [ ] `miro_delete_board`
- [ ] `miro_generate_diagram` - flowchart
- [ ] `miro_generate_diagram` - sequence diagram
- [ ] `miro_create_shape`
- [ ] `miro_create_text`
- [ ] `miro_create_frame`
- [ ] `miro_create_card`
- [ ] `miro_create_image`
- [ ] `miro_create_document`
- [ ] `miro_create_embed`
- [ ] `miro_create_sticky_grid`
- [ ] `miro_create_connector`
- [ ] `miro_list_all_items`
- [ ] `miro_search_board`
- [ ] `miro_update_item`
- [ ] `miro_delete_item`
- [ ] `miro_list_tags`
- [ ] `miro_detach_tag`
- [ ] `miro_get_item_tags`
- [ ] `miro_update_tag`
- [ ] `miro_delete_tag`
- [ ] `miro_create_group`
- [ ] `miro_ungroup`
- [ ] `miro_list_board_members`
- [ ] `miro_share_board`
- [ ] `miro_create_mindmap_node`
- [ ] `miro_get_board_picture`
- [ ] `miro_create_export_job` (Enterprise only)
- [ ] `miro_get_export_job_status` (Enterprise only)
- [ ] `miro_get_export_job_results` (Enterprise only)
- [ ] `miro_get_audit_log`
- [ ] `miro_create_webhook`
- [ ] `miro_list_webhooks`
- [ ] `miro_get_webhook`
- [ ] `miro_delete_webhook`
- [ ] `miro_list_connectors`
- [ ] `miro_get_connector`
- [ ] `miro_update_connector`
- [ ] `miro_delete_connector`

---

## Permission Prompts

Claude Code asks for permission for each MCP tool. To reduce prompts:
1. Select option 2 "Yes, and don't ask again..." when prompted
2. Or use `/permissions` command to add rules like `mcp__miro__*`

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

## Architecture Summary

```
miro-mcp-server/
├── main.go                 # Entry point + --verbose flag
├── miro/
│   ├── client.go           # HTTP client, Config (with TeamID), token validation
│   ├── config.go           # LoadConfigFromEnv, loadTeamIDFromTokensFile
│   ├── boards.go           # ListBoards (uses TeamID), GetBoard, CreateBoard, etc.
│   ├── items.go            # ListItems (max 50), GetItem, UpdateItem, DeleteItem
│   ├── tags.go             # CreateTag (default blue), AttachTag, DetachTag
│   ├── members.go          # ListBoardMembers, ShareBoard
│   ├── diagrams.go         # Diagram generation (skip layout for sequence!)
│   ├── diagrams/           # Mermaid parsers + Sugiyama layout
│   ├── oauth/              # OAuth 2.1 + PKCE
│   └── webhooks/           # Webhook subscriptions + SSE
└── tools/
    ├── definitions.go      # Tool specs (50 tools) - updated descriptions
    └── handlers.go         # Handler registration
```

---

## All Fixes Summary

| Version | Date | Issue | Fix |
|---------|------|-------|-----|
| v1.4.2+ | 2025-12-22 | CreateGroup fails | Request body needs `{data:{items:[...]}}` not `{items:[...]}` |
| v1.4.2+ | 2025-12-22 | ListConnectors limit < 10 fails | Enforce minimum limit of 10 |
| v1.4.2+ | 2025-12-22 | ListBoards empty | Wrong team_id in tokens.json |
| v1.4.2+ | 2025-12-22 | CreateTag fails without color | Default to "blue" |
| v1.4.2+ | 2025-12-22 | ListItems limit 100 fails | Changed to max 50 |
| v1.4.2+ | 2025-12-22 | AttachTag unclear | Emphatic description about sticky_note/card only |
| v1.4.2 | 2025-12-21 | Token validation broken | Use /boards?limit=1 instead of /users/me |
| v1.4.2 | 2025-12-21 | JSON unmarshal offset | Changed offset from string to int |
| v1.4.1 | 2025-12-21 | Sequence diagram visuals | Fixed lifeline width, anchor colors |
| v1.4.0 | 2025-12-21 | Sequence diagrams | Added rendering support |

---

## Session 2 Progress (2025-12-22 afternoon)

### Verified Working This Session
- [x] `miro_list_boards` - Returns 3 boards
- [x] `miro_generate_diagram` - Flowchart (simple and complex)
- [x] `miro_generate_diagram` - Sequence diagram
- [x] `miro_create_shape` - Circle with color
- [x] `miro_create_text` - Heading with font size
- [x] `miro_create_frame` - Container with title
- [x] `miro_create_card` - Card with due date
- [x] `miro_create_connector` - Curved with caption
- [x] `miro_list_connectors` - Works with limit >= 10
- [x] `miro_get_connector` - Gets details
- [x] `miro_update_connector` - Updates caption/color

### Bugs Found & Fixed This Session
1. **CreateGroup**: Wrong request body format - fixed in `miro/groups.go`
2. **ListConnectors**: No minimum limit validation - fixed in `miro/create.go`

### Still Need Testing (after restart)
- [ ] `miro_create_group` - Fixed, needs retest
- [ ] `miro_ungroup`
- [ ] `miro_delete_connector`
- [ ] `miro_search_board`
- [ ] `miro_update_item`
- [ ] `miro_delete_item`
- [ ] `miro_list_board_members`
- [ ] `miro_share_board`
- [ ] `miro_create_mindmap_node`
- [ ] `miro_get_board_picture`
- [ ] Webhook tools
- [ ] Audit log tool

## Next Session

1. **Restart Claude Code** to pick up the rebuilt binary
2. Test `miro_create_group` - should work now with fixed request body
3. Continue testing remaining tools from checklist
4. Consider releasing v1.4.3 with all fixes

---

**Binary rebuilt with fixes - restart Claude Code to continue testing!**

# Miro MCP Server - Testing Status

Last updated: 2025-12-22 (Session 3)

## Test Board

- **Board Name**: All tests
- **Board ID**: `uXjVOXQCe5c=`
- **View Link**: https://miro.com/app/board/uXjVOXQCe5c=

## Test Results Summary

### Passed (39/50 tools tested)

| Tool | Status | Notes |
|------|--------|-------|
| `miro_list_boards` | ✅ | Returns 3 boards |
| `miro_find_board` | ✅ | Finds by name |
| `miro_get_board` | ✅ | Returns board details |
| `miro_get_board_summary` | ✅ | Shows 50 items |
| `miro_list_items` | ✅ | Lists all item types |
| `miro_list_all_items` | ✅ | Pagination works (11 stickies) |
| `miro_list_tags` | ✅ | Found 3 tags |
| `miro_list_connectors` | ✅ | Found 9+ connectors |
| `miro_create_sticky` | ✅ | Created cyan sticky |
| `miro_create_sticky_grid` | ✅ | Created 6 stickies in 3x2 grid |
| `miro_create_shape` | ✅ | Created round_rectangle |
| `miro_create_text` | ✅ | Created 36pt text |
| `miro_create_frame` | ✅ | Created 600x400 frame |
| `miro_create_card` | ✅ | Created card with due date |
| `miro_create_connector` | ✅ | Works with correct caps |
| `miro_create_image` | ✅ | Added image from Wikipedia URL |
| `miro_create_tag` | ✅ | Created magenta tag |
| `miro_attach_tag` | ✅ | Attached to sticky |
| `miro_detach_tag` | ✅ | Removed from sticky |
| `miro_update_tag` | ✅ | Changed title and color |
| `miro_delete_tag` | ✅ | Deleted temp tag |
| `miro_get_item_tags` | ✅ | Returns empty array for no tags |
| `miro_bulk_create` | ✅ | Created 3 stickies |
| `miro_search_board` | ✅ | Found matching items |
| `miro_create_group` | ✅ | Grouped 3 items |
| `miro_ungroup` | ✅ | Ungrouped successfully |
| `miro_update_item` | ✅ | Updated content/color |
| `miro_get_item` | ✅ | Returns full details |
| `miro_get_connector` | ✅ | Returns connector details |
| `miro_update_connector` | ✅ | Updated caption/style |
| `miro_delete_item` | ✅ | Deleted bulk sticky |
| `miro_delete_connector` | ✅ | Deleted test connector |
| `miro_list_board_members` | ✅ | Found 1 owner |
| `miro_get_audit_log` | ✅ | Shows 25+ events |
| `miro_generate_diagram` (flowchart) | ✅ | Created 4 nodes, 4 connectors |
| `miro_generate_diagram` (sequence) | ✅ | Created 14 nodes, 4 connectors |

### Known Issues

| Tool | Status | Issue |
|------|--------|-------|
| `miro_create_mindmap_node` | ❌ | API returns 405 - endpoint may have changed |
| `miro_create_embed` | ⚠️ | BUG FIXED - was sending both width and height |
| `miro_get_board_picture` | ⚠️ | Returns empty - may need board activity to generate |
| `miro_create_webhook` | ❌ | API error "subscription or endpoint does not exist" |
| `miro_list_webhooks` | ❌ | Same error - may need app setup or permissions |

### Fixed Issues (Session 3 - 2025-12-22)

| Issue | Fix | File |
|-------|-----|------|
| `miro_create_embed` sends both width and height | Only send width OR height, not both | `miro/create.go` |
| Tag color "orange" documented but API rejects | Updated valid colors list | `tools/definitions.go`, `miro/types_tags.go` |

### Valid Tag Colors

The Miro API accepts these tag colors (NOT orange):
- `red`, `magenta`, `violet`, `blue`, `cyan`, `green`, `yellow`, `gray`
- `light_green`, `dark_green`, `dark_blue`, `dark_gray`, `black`

### Valid Connector Cap Values

- `none`, `arrow`, `stealth`, `rounded_stealth`
- `diamond`, `filled_diamond`, `oval`, `filled_oval`
- `triangle`, `filled_triangle`
- ERD: `erd_one`, `erd_many`, `erd_only_one`, `erd_zero_or_one`, `erd_one_or_many`, `erd_zero_or_many`

## Tools Not Yet Tested

| Tool | Category | Notes |
|------|----------|-------|
| `miro_create_board` | boards | |
| `miro_copy_board` | boards | |
| `miro_delete_board` | boards | Destructive |
| `miro_create_document` | create | Needs public PDF URL |
| `miro_share_board` | members | Needs email |
| `miro_create_export_job` | export | Enterprise only |
| `miro_get_export_job_status` | export | Enterprise only |
| `miro_get_export_job_results` | export | Enterprise only |
| `miro_get_webhook` | webhooks | Needs working webhook |
| `miro_delete_webhook` | webhooks | Needs working webhook |

## Test Items Created (Session 3)

| Type | ID | Content/Title | Position |
|------|----|---------------|----------|
| sticky_note grid | 3458764653399468846+ | "Idea 1-6" | (3500, 100) |
| image | 3458764653400040795 | "Test Ant Image" | (4500, 300) |

## How to Continue Testing

1. **Restart Claude Code** to pick up the rebuilt binary with fixes

2. **Test the embed fix**:
   ```
   miro_create_embed board_id=uXjVOXQCe5c= url=https://www.youtube.com/watch?v=dQw4w9WgXcQ x=5000 y=100
   ```

3. **Test remaining tools** from the "Not Yet Tested" list

4. **Investigate webhook API** - may need different app permissions or setup

## Running Unit Tests

```bash
cd /Users/olgasafonova/go/src/miro-mcp-server
go test ./...
```

All 6 test packages pass as of 2025-12-22.

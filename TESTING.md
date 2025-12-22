# Miro MCP Server - Testing Status

Last updated: 2025-12-22

## Test Board

- **Board Name**: All tests
- **Board ID**: `uXjVOXQCe5c=`
- **View Link**: https://miro.com/app/board/uXjVOXQCe5c=

## Test Results Summary

### Passed (29/31 tools tested)

| Tool | Status | Notes |
|------|--------|-------|
| `miro_list_boards` | ✅ | Returns 3 boards |
| `miro_get_board_summary` | ✅ | Shows item counts |
| `miro_list_items` | ✅ | Lists all item types |
| `miro_list_tags` | ✅ | Found 2 tags |
| `miro_list_connectors` | ✅ | Found 9+ connectors |
| `miro_create_sticky` | ✅ | Created cyan sticky |
| `miro_create_shape` | ✅ | Created round_rectangle |
| `miro_create_text` | ✅ | Created 36pt text |
| `miro_create_frame` | ✅ | Created 600x400 frame |
| `miro_create_card` | ✅ | Created card with due date |
| `miro_create_connector` | ✅ | Works with correct caps |
| `miro_create_tag` | ✅ | Created magenta tag |
| `miro_attach_tag` | ✅ | Attached to sticky |
| `miro_detach_tag` | ✅ | Removed from sticky |
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
| `miro_find_board` | ⚪ | Not tested this session |

### Fixed Issues (committed 2025-12-22)

| Issue | Fix | Commit |
|-------|-----|--------|
| `miro_get_item_tags` schema validation error | Return empty array `[]` instead of `null` | 1a09e84 |
| `miro_create_connector` invalid cap values | Updated docs: removed `filled_arrow`, added correct values | 1a09e84 |

### Valid Connector Cap Values

The Miro API accepts these cap values (not `filled_arrow`):
- `none`
- `arrow`
- `stealth`
- `rounded_stealth`
- `diamond`
- `filled_diamond`
- `oval`
- `filled_oval`
- `triangle`
- `filled_triangle`
- `erd_one`, `erd_many`, `erd_only_one`, `erd_zero_or_one`, `erd_one_or_many`, `erd_zero_or_many`

## Tools Not Yet Tested

These tools exist but weren't tested in this session:

| Tool | Category | Notes |
|------|----------|-------|
| `miro_get_board` | boards | |
| `miro_create_board` | boards | |
| `miro_copy_board` | boards | |
| `miro_delete_board` | boards | Destructive |
| `miro_find_board` | boards | |
| `miro_create_image` | create | Needs public URL |
| `miro_create_document` | create | Needs public URL |
| `miro_create_embed` | create | Needs embed URL |
| `miro_create_sticky_grid` | create | |
| `miro_create_mindmap_node` | create | |
| `miro_list_all_items` | read | Pagination test |
| `miro_update_tag` | tags | |
| `miro_delete_tag` | tags | Destructive |
| `miro_share_board` | members | Needs email |
| `miro_get_board_picture` | export | |
| `miro_create_export_job` | export | Enterprise only |
| `miro_get_export_job_status` | export | Enterprise only |
| `miro_get_export_job_results` | export | Enterprise only |
| `miro_create_webhook` | webhooks | |
| `miro_list_webhooks` | webhooks | |
| `miro_get_webhook` | webhooks | |
| `miro_delete_webhook` | webhooks | |

## Test Items Created

Items created during testing (on board uXjVOXQCe5c=):

| Type | ID | Content/Title | Position |
|------|----|---------------|----------|
| sticky_note | 3458764653397810331 | "Updated Sticky - Dec 22 ✓" | (1000, 100) |
| shape | 3458764653397810338 | "Target Shape" | (1000, 300) |
| text | 3458764653397810344 | "MCP Server Test Suite" | (1000, -50) |
| frame | 3458764653398073026 | "Test Frame - Dec 22" | (2600, 100) |
| card | 3458764653398073036 | "Test Card" | (2650, 150) |
| sticky_note | 3458764653398073057 | "Bulk 2" | (2900, 250) |
| sticky_note | 3458764653398073060 | "Bulk 3" | (3100, 250) |
| tag | 3458764653396050500 | "Dec22-Test" (magenta) | - |
| flowchart | 3458764653397810524+ | 4 nodes | (1400, 100) |
| sequence diagram | 3458764653397951724+ | 14 nodes | (2000, 100) |

## How to Continue Testing

1. **Restart MCP server** to load the bug fixes:
   ```bash
   # The fixes are in the binary, just restart Claude Code
   # or restart the MCP server process
   ```

2. **Verify fixes work**:
   ```
   # Test null tags handling (should return empty array, not error)
   miro_get_item_tags board_id=uXjVOXQCe5c= item_id=3458764653397810331

   # Test connector with correct cap
   miro_create_connector board_id=uXjVOXQCe5c= start_item_id=X end_item_id=Y end_cap=filled_triangle
   ```

3. **Test remaining tools** from the "Not Yet Tested" list above

4. **Clean up test items** if needed:
   ```
   miro_delete_item board_id=uXjVOXQCe5c= item_id=<id>
   ```

## Running Unit Tests

```bash
cd /Users/olgasafonova/go/src/miro-mcp-server
go test ./...
```

All 6 test packages pass as of 2025-12-22.

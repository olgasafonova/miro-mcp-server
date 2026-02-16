# Miro API Gap Analysis

**Date:** 2026-02-16 (updated after v1.14.0 release)
**Spec source:** [miroapp/api-clients](https://github.com/miroapp/api-clients) `packages/generator/spec.json`
**Spec version:** v2.0 (last updated 2026-02-16)
**Our version:** 86 tools (v1.14.0)

## Summary

| Category | Miro API Endpoints | Our Tools | Coverage |
|----------|-------------------|-----------|----------|
| Standard (all plans) | 85 | 83 tools | ~97% |
| Experimental | 13 | 5 (mindmap + flowchart) | ~38% |
| Enterprise | 92 | 4 (export) | ~4% |

## Gaps: Standard API (High Priority)

These endpoints exist in the stable v2 API, are available to all plans, and we don't cover them.

### 1. ~~Doc Formats API~~ DONE in v1.14.0

Rich text documents created from Markdown. This is a new item type on Miro boards.

| Endpoint | Tool | Status |
|----------|------|--------|
| `POST /v2/boards/{board_id}/docs` | `miro_create_doc` | Implemented |
| `GET /v2/boards/{board_id}/docs/{item_id}` | `miro_get_doc` | Implemented |
| `DELETE /v2/boards/{board_id}/docs/{item_id}` | `miro_delete_doc` | Implemented |

### 2. ~~Get Items By Tag~~ DONE in v1.14.0

| Endpoint | Tool | Status |
|----------|------|--------|
| `GET /v2/boards/{board_id}/items?tag_id=...` | `miro_get_items_by_tag` | Implemented |

### 3. File Upload Endpoints (partially done)

| Endpoint | Tool | Status |
|----------|------|--------|
| `POST /v2/boards/{board_id}/images` (multipart) | `miro_upload_image` | Implemented in v1.14.0 |
| `PATCH /v2/boards/{board_id}/images/{item_id}` (multipart) | - | Not implemented |
| `POST /v2/boards/{board_id}/documents` (multipart) | - | Not implemented |
| `PATCH /v2/boards/{board_id}/documents/{item_id}` (multipart) | - | Not implemented |
| `POST /v2/boards/{board_id}/items/bulk` (multipart) | - | Not implemented |

**Remaining gaps:**
- `miro_upload_document` - Upload document from local file
- Update image/document from local file (PATCH multipart)
- Bulk create from local files (multipart)

### 4. Type-Specific Delete Endpoints

The API has dedicated delete endpoints per item type. We use the generic `DELETE /v2/boards/{board_id}/items/{item_id}` for everything, which works. However, specific endpoints exist:

| Endpoint | Status |
|----------|--------|
| `DELETE .../sticky_notes/{item_id}` | We use generic delete |
| `DELETE .../shapes/{item_id}` | We use generic delete |
| `DELETE .../texts/{item_id}` | We use generic delete |
| `DELETE .../cards/{item_id}` | We use generic delete |
| `DELETE .../images/{item_id}` | We use generic delete |
| `DELETE .../documents/{item_id}` | We use generic delete |
| `DELETE .../embeds/{item_id}` | We use generic delete |

**Impact:** Low. Generic delete works fine. Type-specific deletes add no user value.

**Action:** No change needed.

### 5. Type-Specific Get Endpoints

Similar to deletes, the API has type-specific GET endpoints (GET .../sticky_notes/{id}, GET .../shapes/{id}, etc.) that return richer type-specific data. We use generic `GET /v2/boards/{board_id}/items/{item_id}`.

**Impact:** Low-Medium. Type-specific gets return more detailed fields (e.g., sticky shape, card due_date). Our generic get handles most cases.

**Action:** Optional enhancement for power users.

### 6. OAuth Token Revoke v2

| Endpoint | What it does |
|----------|-------------|
| `POST /v2/oauth/revoke` | Revoke an OAuth token (v2 endpoint) |

**Impact:** Low. Useful for OAuth flow cleanup. We already support OAuth login.

**Missing tool (optional):**
- `miro_revoke_token` - Revoke the current OAuth token

## Gaps: Experimental API (Medium Priority)

### 7. Flowchart Shapes (Experimental) - partially done

| Endpoint | Tool | Status |
|----------|------|--------|
| `POST /v2-experimental/boards/{board_id}/shapes` | `miro_create_flowchart_shape` | Implemented in v1.14.0 |
| `GET /v2-experimental/boards/{board_id}/shapes/{item_id}` | - | Not implemented |
| `PATCH /v2-experimental/boards/{board_id}/shapes/{item_id}` | - | Not implemented |
| `DELETE /v2-experimental/boards/{board_id}/shapes/{item_id}` | - | Not implemented (generic delete works) |
| `GET /v2-experimental/boards/{board_id}/items` | - | Not implemented |
| `GET /v2-experimental/boards/{board_id}/items/{item_id}` | - | Not implemented |
| `DELETE /v2-experimental/boards/{board_id}/items/{item_id}` | - | Not implemented |

**Remaining gaps:** Get, update, and delete for experimental shapes. Low priority; generic endpoints work for get/delete.

### 8. App Metrics (Experimental)

| Endpoint | What it does |
|----------|-------------|
| `GET /v2-experimental/apps/{app_id}/metrics` | Get app usage metrics |
| `GET /v2-experimental/apps/{app_id}/metrics-total` | Get total app metrics |

**Impact:** Low for end users. Useful for us as maintainers to track adoption.

**Action:** Skip for now.

## Gaps: Enterprise API (Low Priority)

Enterprise endpoints require `org_id` and enterprise-tier accounts. We cover the board export subset (4 tools). Major uncovered areas:

| Area | Endpoints | Notes |
|------|-----------|-------|
| Legal Holds | 12 | Compliance feature for enterprise |
| Board Export (extended) | 3 more | Export job tasks, update status, create links |
| SCIM User/Group Management | ~25 | Identity provisioning |
| Organizations & Teams | ~30 | Org structure management |
| Projects | ~12 | Project management |
| Board Classification | 6 | Data classification labels |
| Audit Logs | 1 | Enterprise audit trail |
| Board Content Logs | 1 | Content change tracking |

**Impact:** Low for our target audience (individual devs and small teams). Enterprise customers typically use Miro's official tooling.

**Action:** Skip unless user demand emerges. Document as "Enterprise features available via Miro API but not exposed in this server."

## Webhook Status

Miro sunset experimental webhooks on December 5, 2025. The webhook schemas still exist in the OpenAPI spec (`BoardSubscription`, etc.) but the endpoints are gone from the paths. Our removal was correct.

## What We Have That the API Doesn't

Our MCP server adds value-add tools not in the raw API:

| Our Tool | What it does |
|----------|-------------|
| `miro_find_board` | Search boards by name (composite) |
| `miro_get_board_summary` | Board stats with item counts (composite) |
| `miro_get_board_content` | Full board content for AI analysis (composite) |
| `miro_create_sticky_grid` | Grid layout of stickies (composite) |
| `miro_search_board` | Text search across board items (composite) |
| `miro_generate_diagram` | Mermaid-to-Miro diagram generation (custom) |
| `miro_get_audit_log` | Local execution audit trail (custom) |
| `miro_desire_path_report` | Agent behavior normalization insights (custom) |
| `miro_bulk_create/update/delete` | Batched operations with dry_run (enhanced) |
| Type-specific updates (8) | `update_sticky`, `update_shape`, `update_text`, `update_card`, `update_image`, `update_document`, `update_embed`, `update_group` (enhanced) |
| Type-specific reads (2) | `get_image`, `get_document` (enhanced with richer fields) |

## Recommended Priorities (updated post v1.14.0)

### Completed in v1.14.0
- ~~**Doc Formats API** (3 new tools)~~ `miro_create_doc`, `miro_get_doc`, `miro_delete_doc`
- ~~**Get Items By Tag** (1 new tool)~~ `miro_get_items_by_tag`
- ~~**File Upload (images)** (1 new tool)~~ `miro_upload_image`
- ~~**Flowchart Shapes (experimental)** (1 new tool)~~ `miro_create_flowchart_shape`

### P1 - Do Soon (medium value, medium effort)
1. **Upload document from file** - Complete the multipart upload set
2. **Update image/document from file** - PATCH multipart for existing items

### P2 - Monitor
3. **Experimental flowchart get/update/delete** - Generic endpoints cover most cases
4. **OAuth Token Revoke v2** - Minor utility
5. **Type-specific get endpoints** - Richer fields for power users

### P3 - Skip
6. Enterprise APIs - Not our target audience
7. App Metrics - Maintainer-only value
8. Type-specific delete endpoints - Generic delete works fine

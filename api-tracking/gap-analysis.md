# Miro API Gap Analysis

**Date:** 2026-02-16
**Spec source:** [miroapp/api-clients](https://github.com/miroapp/api-clients) `packages/generator/spec.json`
**Spec version:** v2.0 (last updated 2026-02-16)
**Our version:** 77 tools (v1.11.1)

## Summary

| Category | Miro API Endpoints | Our Tools | Coverage |
|----------|-------------------|-----------|----------|
| Standard (all plans) | 85 | 77 tools | ~90% |
| Experimental | 13 | 4 (mindmap) | ~30% |
| Enterprise | 92 | 4 (export) | ~4% |

## Gaps: Standard API (High Priority)

These endpoints exist in the stable v2 API, are available to all plans, and we don't cover them.

### 1. Doc Formats API (NEW - ~2 months old)

Rich text documents created from Markdown. This is a new item type on Miro boards.

| Endpoint | What it does |
|----------|-------------|
| `POST /v2/boards/{board_id}/docs` | Create doc from Markdown |
| `GET /v2/boards/{board_id}/docs/{item_id}` | Get doc item |
| `DELETE /v2/boards/{board_id}/docs/{item_id}` | Delete doc item |

**Schema:** Takes `{ data: { contentType: "markdown", content: "# Hello" }, position, parent }`

**Impact:** High. This is a brand new item type. LLMs sending Markdown to Miro is a natural fit.

**Missing tools to add:**
- `miro_create_doc` - Create a doc format item from Markdown
- `miro_get_doc` - Get doc format item details
- `miro_delete_doc` - Delete a doc format item

### 2. Get Items By Tag

| Endpoint | What it does |
|----------|-------------|
| `GET /v2/boards/{board_id}/items?tag_id=...` | Filter items by tag ID |

**Impact:** Medium. We have tag CRUD and attach/detach, but can't query "show me all items tagged Urgent". Useful for board analysis workflows.

**Missing tool:**
- `miro_get_items_by_tag` - List all items with a specific tag

### 3. File Upload Endpoints

| Endpoint | What it does |
|----------|-------------|
| `POST /v2/boards/{board_id}/images` (multipart) | Upload image from local file |
| `PATCH /v2/boards/{board_id}/images/{item_id}` (multipart) | Update image from local file |
| `POST /v2/boards/{board_id}/documents` (multipart) | Upload document from local file |
| `PATCH /v2/boards/{board_id}/documents/{item_id}` (multipart) | Update document from local file |
| `POST /v2/boards/{board_id}/items/bulk` (multipart) | Bulk create from local files |

**Impact:** Medium. We only support URL-based image/document creation. File upload would enable local screenshot uploads, generated images, etc.

**Note:** MCP protocol has limited binary file support via stdio. This would primarily benefit HTTP mode users. Consider implementing for images first (most common use case).

**Missing tools (stretch):**
- `miro_upload_image` - Upload image from local file
- `miro_upload_document` - Upload document from local file

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

### 7. Flowchart Shapes (Experimental)

New experimental endpoints for flowchart-specific shapes with richer properties.

| Endpoint | What it does |
|----------|-------------|
| `POST /v2-experimental/boards/{board_id}/shapes` | Create flowchart shape |
| `GET /v2-experimental/boards/{board_id}/shapes/{item_id}` | Get flowchart shape |
| `PATCH /v2-experimental/boards/{board_id}/shapes/{item_id}` | Update flowchart shape |
| `DELETE /v2-experimental/boards/{board_id}/shapes/{item_id}` | Delete flowchart shape |
| `GET /v2-experimental/boards/{board_id}/items` | Get items (experimental) |
| `GET /v2-experimental/boards/{board_id}/items/{item_id}` | Get specific item (experimental) |
| `DELETE /v2-experimental/boards/{board_id}/items/{item_id}` | Delete item (experimental) |

**Impact:** Medium. We already have `CreateShapeExperimental` in our client code but don't expose it as a tool. These may offer improved flowchart shape types for our diagram generation feature.

**Action:** Monitor. Expose when/if it moves to GA. We already have the client method.

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
| `miro_bulk_create/update/delete` | Batched operations with dry_run (enhanced) |

## Recommended Priorities

### P0 - Do Now (high value, low effort)
1. **Doc Formats API** (3 new tools) - New item type, Markdown input is perfect for LLMs
2. **Get Items By Tag** (1 new tool) - Completes tag workflow

### P1 - Do Soon (medium value, medium effort)
3. **File Upload (images)** - Enable local screenshot/generated image uploads

### P2 - Monitor
4. **Flowchart Shapes (experimental)** - Already have client code; expose when GA
5. **OAuth Token Revoke v2** - Minor utility

### P3 - Skip
6. Enterprise APIs - Not our target audience
7. App Metrics - Maintainer-only value
8. Type-specific get/delete - Generic endpoints work fine

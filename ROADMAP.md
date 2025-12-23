# Miro MCP Server - Development Roadmap

> **Goal**: Build the most comprehensive, performant, secure, and user-friendly Miro MCP server.
> **Language**: Go (unique differentiator - only Go-based Miro MCP server)
> **Status**: 76 tools implemented. Phases 1-7 complete, plus batch operations and type-specific updates.
> **Last Updated**: 2025-12-23

---

## Table of Contents

1. [Current State](#current-state)
2. [Competitive Analysis](#competitive-analysis)
3. [Gap Analysis](#gap-analysis)
4. [Implementation Roadmap](#implementation-roadmap)
5. [Technical Specifications](#technical-specifications)
6. [Code Patterns](#code-patterns)
7. [Testing Strategy](#testing-strategy)

---

## Current State

### Architecture

```
miro-mcp-server/
├── main.go                 # Entry point, dual transport (stdio/HTTP)
├── miro/
│   ├── client.go          # API client with rate limiting, caching
│   ├── config.go          # Environment-based configuration
│   └── types.go           # All request/response types
└── tools/
    ├── definitions.go     # Tool specifications (voice-optimized)
    └── handlers.go        # Generic handler registration
```

### Implemented Tools (76 total)

| Category | Tool | Method |
|----------|------|--------|
| **Boards (8)** | `miro_list_boards` | ListBoards |
| | `miro_get_board` | GetBoard |
| | `miro_create_board` | CreateBoard |
| | `miro_copy_board` | CopyBoard |
| | `miro_update_board` | UpdateBoard |
| | `miro_delete_board` | DeleteBoard |
| | `miro_find_board` | FindBoardByNameTool |
| | `miro_get_board_summary` | GetBoardSummary |
| **Members (5)** | `miro_list_board_members` | ListBoardMembers |
| | `miro_get_board_member` | GetBoardMember |
| | `miro_share_board` | ShareBoard |
| | `miro_update_board_member` | UpdateBoardMember |
| | `miro_remove_board_member` | RemoveBoardMember |
| **Create (13)** | `miro_create_sticky` | CreateSticky |
| | `miro_create_sticky_grid` | CreateStickyGrid |
| | `miro_create_shape` | CreateShape |
| | `miro_create_text` | CreateText |
| | `miro_create_frame` | CreateFrame |
| | `miro_create_card` | CreateCard |
| | `miro_create_app_card` | CreateAppCard |
| | `miro_create_image` | CreateImage |
| | `miro_create_document` | CreateDocument |
| | `miro_create_embed` | CreateEmbed |
| | `miro_create_connector` | CreateConnector |
| | `miro_create_group` | CreateGroup |
| | `miro_create_mindmap_node` | CreateMindmapNode |
| **Frames (4)** | `miro_get_frame` | GetFrame |
| | `miro_update_frame` | UpdateFrame |
| | `miro_delete_frame` | DeleteFrame |
| | `miro_get_frame_items` | GetFrameItems |
| **Mindmaps (4)** | `miro_create_mindmap_node` | CreateMindmapNode |
| | `miro_get_mindmap_node` | GetMindmapNode |
| | `miro_list_mindmap_nodes` | ListMindmapNodes |
| | `miro_delete_mindmap_node` | DeleteMindmapNode |
| **Read (5)** | `miro_list_items` | ListItems |
| | `miro_list_all_items` | ListAllItems |
| | `miro_get_item` | GetItem |
| | `miro_get_app_card` | GetAppCard |
| | `miro_search_board` | SearchBoard |
| **Update (13)** | `miro_update_item` | UpdateItem |
| | `miro_update_app_card` | UpdateAppCard |
| | `miro_update_connector` | UpdateConnector |
| | `miro_update_tag` | UpdateTag |
| | `miro_update_frame` | UpdateFrame |
| | `miro_update_sticky` | UpdateSticky |
| | `miro_update_shape` | UpdateShape |
| | `miro_update_text` | UpdateText |
| | `miro_update_card` | UpdateCard |
| | `miro_update_group` | UpdateGroup |
| | `miro_update_image` | UpdateImage |
| | `miro_update_document` | UpdateDocument |
| | `miro_update_embed` | UpdateEmbed |
| **Bulk (3)** | `miro_bulk_create` | BulkCreate |
| | `miro_bulk_update` | BulkUpdate |
| | `miro_bulk_delete` | BulkDelete |
| **Delete (6)** | `miro_delete_item` | DeleteItem |
| | `miro_delete_app_card` | DeleteAppCard |
| | `miro_delete_connector` | DeleteConnector |
| | `miro_delete_tag` | DeleteTag |
| | `miro_delete_frame` | DeleteFrame |
| | `miro_delete_group` | DeleteGroup |
| **Tags (6)** | `miro_create_tag` | CreateTag |
| | `miro_list_tags` | ListTags |
| | `miro_get_tag` | GetTagTool |
| | `miro_attach_tag` | AttachTag |
| | `miro_detach_tag` | DetachTag |
| | `miro_get_item_tags` | GetItemTags |
| **Connectors (4)** | `miro_create_connector` | CreateConnector |
| | `miro_list_connectors` | ListConnectors |
| | `miro_get_connector` | GetConnector |
| | `miro_delete_connector` | DeleteConnector |
| **Groups (5)** | `miro_create_group` | CreateGroup |
| | `miro_ungroup` | Ungroup |
| | `miro_list_groups` | ListGroups |
| | `miro_get_group` | GetGroup |
| | `miro_get_group_items` | GetGroupItems |
| **Export (4)** | `miro_get_board_picture` | GetBoardPicture |
| | `miro_create_export_job` | CreateExportJob |
| | `miro_get_export_job_status` | GetExportJobStatus |
| | `miro_get_export_job_results` | GetExportJobResults |
| **Diagrams (1)** | `miro_generate_diagram` | GenerateDiagram |
| **Audit (1)** | `miro_get_audit_log` | GetAuditLog |

### Existing Strengths

- **Rate Limiting**: Semaphore-based (5 concurrent requests)
- **Caching**: 2-minute TTL for board data
- **Connection Pooling**: 100 max idle, 10 per host
- **Panic Recovery**: Catches and logs panics in handlers
- **Structured Logging**: slog with context
- **Dual Transport**: stdio (default) + HTTP with health endpoint
- **Voice-Optimized**: Tool descriptions designed for voice interaction
- **Token Validation**: Validates MIRO_ACCESS_TOKEN on startup with clear error messages
- **Board Name Resolution**: Find boards by name, not just ID (`miro_find_board`)
- **Input Sanitization**: Validates board IDs and content to prevent injection
- **Retry with Backoff**: Exponential backoff for rate-limited requests
- **Composite Tools**: `miro_get_board_summary`, `miro_create_sticky_grid`

---

## Competitive Analysis

### Competitor Overview

| Server | Language | Stars | Tools | Last Update | License |
|--------|----------|-------|-------|-------------|---------|
| **Official Miro MCP** | Hosted | N/A | ~10 | Active | Proprietary |
| **evalstate/mcp-miro** | TypeScript | 101 | ~8 | Nov 2024 | - |
| **k-jarzyna/mcp-miro** | TypeScript | 59 | 81 | Active | Apache 2.0 |
| **LuotoCompany/mcp-server-miro** | TypeScript | 14 | ~15 | Apr 2025 | MIT |
| **Ours** | **Go** | - | **76** | Active | MIT |

### Feature Comparison Matrix

| Feature | Ours | Official | evalstate | k-jarzyna | LuotoCompany |
|---------|------|----------|-----------|-----------|--------------|
| **Board list/get** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Board create/delete** | ✅ | ? | ❌ | ✅ | ❌ |
| **Board copy** | ✅ | ? | ❌ | ✅ | ❌ |
| **Sticky notes** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Shapes** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Text** | ✅ | ✅ | ? | ✅ | ✅ |
| **Connectors** | ✅ | ✅ | ? | ✅ | ✅ |
| **Frames** | ✅ | ✅ | ✅ | ✅ | ? |
| **Cards** | ✅ | ? | ? | ✅ | ✅ |
| **Images** | ✅ | ? | ? | ✅ | ✅ |
| **Documents** | ✅ | ? | ? | ✅ | ✅ |
| **Embeds** | ✅ | ? | ? | ✅ | ✅ |
| **Tags** | ✅ | ❌ | ❌ | ✅ | ❌ |
| **Groups** | ✅ | ❌ | ❌ | ✅ | ❌ |
| **Members/sharing** | ✅ | ❌ | ❌ | ✅ | ❌ |
| **Mindmaps** | ✅ | ❌ | ❌ | ✅ | ❌ |
| **Export** | ✅ | ❌ | ❌ | ✅ | ❌ |
| **Bulk operations** | ✅ | ? | ✅ | ✅ | ? |
| **Rate limiting** | ✅ | ✅ | ❌ | ❌ | ❌ |
| **Caching** | ✅ | ? | ❌ | ❌ | ❌ |
| **Dual transport** | ✅ | ❌ | ❌ | ❌ | ✅ (SSE) |
| **Voice-optimized** | ✅ | ❌ | ❌ | ❌ | ❌ |
| **Diagram generation** | ✅ | ✅ | ❌ | ❌ | ❌ |
| **Code generation** | ❌ | ✅ | ❌ | ❌ | ❌ |

### Our Unique Advantages

1. **Go Language**: Faster, lower memory, single binary deployment
2. **Rate Limiting**: Built-in protection against API limits
3. **Response Caching**: Reduces redundant API calls
4. **Voice-Optimized Descriptions**: Better for voice assistants
5. **Panic Recovery**: Production-safe error handling
6. **Dual Transport**: Works with any MCP client

---

## Gap Analysis

### Tier 1: High Priority (Must Have)

These features are commonly used and provided by competitors.

| Feature | Miro API Endpoint | Complexity | Impact |
|---------|-------------------|------------|--------|
| Cards | `POST /v2/boards/{id}/cards` | Medium | High |
| Images | `POST /v2/boards/{id}/images` | Low | High |
| Tags (CRUD) | `POST /v2/boards/{id}/tags` | Medium | High |
| Tag attach/detach | `POST /v2/boards/{id}/items/{id}/tags` | Low | High |
| Board create | `POST /v2/boards` | Low | Medium |
| Board copy | `POST /v2/boards/{id}/copy` | Low | Medium |
| Board delete | `DELETE /v2/boards/{id}` | Low | Low |

### Tier 2: Competitive Parity

| Feature | Miro API Endpoint | Complexity | Impact |
|---------|-------------------|------------|--------|
| Documents | `POST /v2/boards/{id}/documents` | Medium | Medium |
| Embeds | `POST /v2/boards/{id}/embeds` | Medium | Medium |
| Groups | `POST /v2/boards/{id}/groups` | Medium | Medium |
| Board members | `GET /v2/boards/{id}/members` | Low | Medium |
| Share board | `POST /v2/boards/{id}/members` | Medium | Medium |
| Mindmap nodes | `POST /v2/boards/{id}/mind_map_nodes` | Medium | Low |

### Tier 3: Differentiation (Beat Everyone)

| Feature | Description | Complexity | Impact |
|---------|-------------|------------|--------|
| Token validation | Verify token on startup | Low | High |
| Board name resolution | Find board by name, not just ID | Low | High |
| Composite tools | Single tool for common workflows | Medium | High |
| Retry with backoff | Handle rate limits gracefully | Medium | Medium |
| Input sanitization | Prevent injection attacks | Low | High |
| Fuzzy search | Typo-tolerant board/item search | Medium | Medium |

### Tier 4: Enterprise Features

| Feature | Description | Complexity | Impact |
|---------|-------------|------------|--------|
| OAuth 2.1 flow | Full OAuth instead of static token | High | High |
| Webhooks | Real-time event notifications | High | Medium |
| Audit logging | Track all operations | Medium | Low |
| Multi-board ops | Operations across multiple boards | High | Medium |

---

## Implementation Roadmap

> **Note:** All phases 1-8 are complete. Code examples removed as implementation is done.
> See `CLAUDE.md` for current code patterns and `tools/definitions.go` for tool specs.

### Phase 1: Core Completeness ✅
- Cards, Images, Tags (CRUD), Board Management
- 27 tools implemented

### Phase 2: Differentiation ✅
- Token validation on startup
- Board name resolution (`miro_find_board`)
- Input sanitization for IDs and content
- Composite tools (`miro_get_board_summary`, `miro_create_sticky_grid`)
- Retry with exponential backoff for rate limits

### Phase 3: Additional Features ✅
- Documents, Embeds, Groups, Board Members, Mindmap nodes
- 5 new tools implemented

---

## Technical Reference

See [Miro REST API Documentation](https://developers.miro.com/reference/api-reference) for complete endpoint reference.

See `CLAUDE.md` for code patterns and how to add new tools.

---

## Testing

```bash
# Unit tests
go test ./...

# With coverage
go test -cover ./...

# Integration tests (requires real token)
MIRO_TEST_TOKEN=your_token go test -tags=integration ./...
```

See `tools/mock_client_test.go` for mock implementation and `miro/client_test.go` for test patterns.

---

## Appendix: Full Tool List Target

### Phase 1 Tools (27 implemented)

| Tool | Status |
|------|--------|
| `miro_list_boards` | ✅ Done |
| `miro_get_board` | ✅ Done |
| `miro_create_board` | ✅ Done |
| `miro_copy_board` | ✅ Done |
| `miro_delete_board` | ✅ Done |
| `miro_create_sticky` | ✅ Done |
| `miro_create_shape` | ✅ Done |
| `miro_create_text` | ✅ Done |
| `miro_create_connector` | ✅ Done |
| `miro_create_frame` | ✅ Done |
| `miro_create_card` | ✅ Done |
| `miro_create_image` | ✅ Done |
| `miro_create_document` | ✅ Done |
| `miro_create_embed` | ✅ Done |
| `miro_bulk_create` | ✅ Done |
| `miro_list_items` | ✅ Done |
| `miro_list_all_items` | ✅ Done |
| `miro_get_item` | ✅ Done |
| `miro_search_board` | ✅ Done |
| `miro_update_item` | ✅ Done |
| `miro_delete_item` | ✅ Done |
| `miro_list_tags` | ✅ Done |
| `miro_create_tag` | ✅ Done |
| `miro_get_tag` | ✅ Done |
| `miro_attach_tag` | ✅ Done |
| `miro_detach_tag` | ✅ Done |
| `miro_get_item_tags` | ✅ Done |

### Phase 2 Tools (Differentiation)

| Tool | Status |
|------|--------|
| `miro_get_board_summary` | ✅ Done |
| `miro_create_sticky_grid` | ✅ Done |
| `miro_find_board` | ✅ Done |

### Phase 2 Enhancements

| Feature | Status |
|---------|--------|
| Token validation on startup | ✅ Done |
| Board name resolution | ✅ Done |
| Input sanitization | ✅ Done |
| Retry with exponential backoff | ✅ Done |

### Phase 3 Tools (Additional Features)

| Tool | Status |
|------|--------|
| `miro_create_group` | ✅ Done |
| `miro_ungroup` | ✅ Done |
| `miro_list_board_members` | ✅ Done |
| `miro_share_board` | ✅ Done |
| `miro_create_mindmap_node` | ✅ Done |

### Phase 4 Tools (Export)

| Tool | Status | Notes |
|------|--------|-------|
| `miro_get_board_picture` | ✅ Done | All plans - gets board thumbnail URL |
| `miro_create_export_job` | ✅ Done | Enterprise only - PDF/SVG/HTML export |
| `miro_get_export_job_status` | ✅ Done | Enterprise only - check progress |
| `miro_get_export_job_results` | ✅ Done | Enterprise only - get download links |

### Phase 5: Enterprise Features (Complete ✅)

| Feature | Status | Notes |
|---------|--------|-------|
| Audit Logging (Local) | ✅ Done | File/memory logger, middleware integration, query tool |
| OAuth 2.1 Flow | ✅ Done | Full OAuth with PKCE, auto-refresh, CLI commands |
| Webhooks Support | ❌ Removed | Miro sunset experimental webhooks Dec 2025 |

#### Phase 5 Tools

| Tool | Status | Notes |
|------|--------|-------|
| `miro_get_audit_log` | ✅ Done | Query local audit log for MCP tool executions |

#### Phase 5 Enhancements

| Feature | Status |
|---------|--------|
| Audit event logging for all tool calls | ✅ Done |
| File-based audit logger with rotation | ✅ Done |
| Memory-based audit logger (dev/testing) | ✅ Done |
| Sensitive input sanitization | ✅ Done |
| Event builder with fluent API | ✅ Done |
| OAuth 2.1 with PKCE support | ✅ Done |
| OAuth token auto-refresh | ✅ Done |
| OAuth CLI commands (login/status/logout) | ✅ Done |
| Secure token storage (~/.miro/tokens.json) | ✅ Done |

### Phase 6: Extended Features (Complete ✅)

| Feature | Status | Notes |
|---------|--------|-------|
| Diagram Generation | ✅ Done | Mermaid flowchart and sequence diagrams → Miro shapes |
| Connector List/Get | ✅ Done | Full CRUD for connectors |
| Tag Update/Delete | ✅ Done | Complete tag management |

#### Phase 6 Tools

| Tool | Status | Notes |
|------|--------|-------|
| `miro_generate_diagram` | ✅ Done | Convert Mermaid to Miro shapes/connectors |
| `miro_list_connectors` | ✅ Done | List all connectors on a board |
| `miro_get_connector` | ✅ Done | Get full connector details |
| `miro_update_connector` | ✅ Done | Update connector style/endpoints |
| `miro_delete_connector` | ✅ Done | Delete a connector |
| `miro_update_tag` | ✅ Done | Update tag title/color |
| `miro_delete_tag` | ✅ Done | Delete a tag |

#### Phase 6 Enhancements

| Feature | Status |
|---------|--------|
| Mermaid flowchart parser | ✅ Done |
| Mermaid sequence diagram parser | ✅ Done |
| Sugiyama-style layout algorithm | ✅ Done |
| Auto-layout for diagrams | ✅ Done |
| Support for 7 node shapes | ✅ Done |
| Support for 4 edge styles | ✅ Done |

### Phase 7: Frame & Mindmap Tools + Distribution (Complete ✅)

| Feature | Status | Notes |
|---------|--------|-------|
| Frame CRUD | ✅ Done | Get, update, delete, list items in frame |
| Mindmap CRUD | ✅ Done | Get, list, delete mindmap nodes |
| App Card CRUD | ✅ Done | Create, get, update, delete app cards |
| Member Management | ✅ Done | Get, update, remove board members |
| Group Management | ✅ Done | List, get, get items, delete groups |
| Board Update | ✅ Done | Update board name/description |
| Distribution | ✅ Done | Homebrew tap, Docker, install script |

#### Phase 7 Tools (+12 new, 66 total including miro_get_tag)

| Tool | Status | Notes |
|------|--------|-------|
| `miro_get_frame` | ✅ Done | Get frame details |
| `miro_update_frame` | ✅ Done | Update frame title/color/size |
| `miro_delete_frame` | ✅ Done | Delete a frame |
| `miro_get_frame_items` | ✅ Done | List items inside a frame |
| `miro_get_mindmap_node` | ✅ Done | Get node details (v2-experimental API) |
| `miro_list_mindmap_nodes` | ✅ Done | List all mindmap nodes |
| `miro_delete_mindmap_node` | ✅ Done | Delete a mindmap node |
| `miro_create_app_card` | ✅ Done | Create app card with custom fields |
| `miro_get_app_card` | ✅ Done | Get app card details |
| `miro_update_app_card` | ✅ Done | Update app card fields |
| `miro_delete_app_card` | ✅ Done | Delete an app card |
| `miro_update_board` | ✅ Done | Update board name/description |
| `miro_get_board_member` | ✅ Done | Get member details |
| `miro_update_board_member` | ✅ Done | Update member role |
| `miro_remove_board_member` | ✅ Done | Remove member from board |
| `miro_list_groups` | ✅ Done | List all groups on board |
| `miro_get_group` | ✅ Done | Get group details |
| `miro_get_group_items` | ✅ Done | List items in a group |
| `miro_delete_group` | ✅ Done | Delete a group |

#### Phase 7 Distribution

| Platform | Status | Notes |
|----------|--------|-------|
| Homebrew tap | ✅ Done | `brew tap olgasafonova/tap && brew install miro-mcp-server` |
| Docker image | ✅ Done | `ghcr.io/olgasafonova/miro-mcp-server:latest` |
| Install script | ✅ Done | `curl -fsSL https://...install.sh | sh` |
| Linux ARM64 | ✅ Done | New binary for ARM64 Linux |
| GitHub Release | ✅ Done | v1.7.0 with all binaries |

### Phase 8: Batch Operations & Type-Specific Updates (Complete ✅)

| Feature | Status | Notes |
|---------|--------|-------|
| Batch Update/Delete | ✅ Done | Efficient multi-item operations |
| Type-Specific Updates | ✅ Done | Dedicated endpoints for all item types |

#### Phase 8 Tools (+10 new, 76 total)

| Tool | Status | Notes |
|------|--------|-------|
| `miro_bulk_update` | ✅ Done | Update up to 20 items at once |
| `miro_bulk_delete` | ✅ Done | Delete up to 20 items at once |
| `miro_update_sticky` | ✅ Done | Update sticky via dedicated endpoint |
| `miro_update_shape` | ✅ Done | Update shape via dedicated endpoint |
| `miro_update_text` | ✅ Done | Update text via dedicated endpoint |
| `miro_update_card` | ✅ Done | Update card via dedicated endpoint |
| `miro_update_group` | ✅ Done | Update group members |
| `miro_update_image` | ✅ Done | Update image title, URL, position |
| `miro_update_document` | ✅ Done | Update document title, URL, position |
| `miro_update_embed` | ✅ Done | Update embed URL, mode, dimensions |

#### Phase 8 Coverage

| Package | Coverage |
|---------|----------|
| miro | 88.9% |
| audit | 92.8% |
| diagrams | 92.6% |
| oauth | 72.5% |
| tools | 99.1% |

---

## Notes for Future Claude Code Sessions

1. **Types are already defined** for Cards, Images, Documents, Embeds, and Tags in `miro/types.go` - just need client methods and tool registration.

2. **Follow the existing pattern** - see `CreateSticky` as the template for all create operations.

3. **Tool descriptions are voice-optimized** - keep them short, action-oriented, with "USE WHEN" and "VOICE-FRIENDLY" sections.

4. **Test with real Miro account** - get a token at https://miro.com/app/settings/user-profile/apps

5. **Rate limits** - Miro allows 100k credits/minute, but our semaphore limits to 5 concurrent requests for safety.

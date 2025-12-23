# Miro MCP Server Improvement Plan

## Executive Summary

This plan addresses verified improvement opportunities identified through competitive analysis and MCP ecosystem best practices review (December 2024).

**Current State:** Production-ready with 76 tools, best-in-class performance, comprehensive API coverage.

**Key Finding:** The server already has proper input schemas via `jsonschema` struct tags, complete installation options, and full Miro REST API coverage (non-enterprise).

**Last Updated:** 2025-12-23

---

## Phase 1: Quick Wins (1-2 hours) ✅ COMPLETE

### 1.1 Fix Tool Count Inconsistency ✅
- **Issue:** README mentions 68, 73, and 76 tools in different places
- **Actual count:** 76 tools
- **Fix:** Updated all references to 76

### 1.2 Add MCP Inspector Documentation ✅
- **Issue:** No guidance on debugging with MCP Inspector
- **Solution:** Added comprehensive section to README and SETUP.md
- **Content:** MCP Inspector, manual JSON-RPC testing, HTTP mode debugging

### 1.3 Add Test Coverage Badge ✅
- **Issue:** Coverage not visible in CI
- **Solution:** Added Codecov badge to README (CI already uploads coverage)

### 1.4 Add Demo GIF
- **Issue:** No visual demonstration
- **Status:** Pending (requires manual screen recording)
- **Solution:** Create 30-second GIF showing basic workflow

---

## Phase 2: Code Quality (4-6 hours) ✅ COMPLETE

### 2.1 Extract Validation Helpers ✅
- **Issue:** Duplicated validation pattern in 44+ methods
- **Solution:** Added to `miro/errors.go`:
  - Predefined errors: `ErrBoardIDRequired`, `ErrItemIDRequired`, etc. (16 constants)
  - Helper functions: `RequireBoardID()`, `RequireItemID()`, `RequireNonEmpty()`, `RequireNonEmptySlice()`, `RequireMinItems()`
- **Tests:** Full test coverage in `miro/errors_test.go`
- **Migration:** New code can use helpers immediately; existing code can migrate gradually

### 2.2 Add Typed Error Constants ✅
- **Issue:** String-based error matching is fragile
- **Solution:** Combined with 2.1 - predefined error constants allow `errors.Is()` comparison
- **Note:** Rate limit errors already use `APIError` type with `IsRateLimited()` method

### 2.3 Split Large Test Files
- **Issue:**
  - `miro/client_test.go` - 9,617 lines
  - `miro/audit/audit_test.go` - 36,446 lines
- **Status:** Pending (low priority - tests work fine)
- **Solution:** Split by domain (boards_test.go, items_test.go, etc.)

### 2.4 Define Config Constants ✅
- **Issue:** Magic numbers scattered throughout code
- **Status:** Complete
- **Solution:** Created `miro/constants.go` with centralized constants:
  - API pagination limits (DefaultBoardLimit, MaxBoardLimit, etc.)
  - Bulk operation limits (MaxBulkItems, MinGroupItems)
  - HTTP server timeouts (HTTPReadTimeout, HTTPWriteTimeout, HTTPIdleTimeout)
  - Cache configuration (BoardCacheTTL, ItemCacheTTL, TagCacheTTL, CacheMaxEntries)
  - Rate limiting config (RateLimitMaxDelay, IdleConnTimeout)
  - Circuit breaker config (CircuitBreakerTimeout, thresholds)
  - OAuth config (OAuthServerReadTimeout, TokenRefreshBuffer)
  - Diagram config (MaxDiagramNodes, DefaultNodeWidth/Height/Spacing)
  - Audit log config (DefaultAuditMaxSizeBytes, DefaultMemoryRingSize)
- **Files updated:** main.go, cache.go, constants.go

---

## Phase 3: MCP Compliance (4-8 hours) ✅ COMPLETE

### 3.1 Add Dry-Run Mode for Destructive Operations ✅
- **Issue:** Delete operations execute immediately without preview
- **Solution:** Added `dry_run` parameter to all 9 delete operations:
  - `miro_delete_item` ✅
  - `miro_delete_board` ✅
  - `miro_bulk_delete` ✅
  - `miro_delete_tag` ✅
  - `miro_delete_frame` ✅
  - `miro_delete_group` ✅
  - `miro_delete_connector` ✅
  - `miro_delete_mindmap_node` ✅
  - `miro_delete_app_card` ✅
- **Behavior:** When `dry_run: true`, returns `[DRY RUN] Would delete...` message without calling API
- **Tool descriptions:** Updated all 9 tool descriptions to document `dry_run` parameter

### 3.2 Add Preview for Bulk Operations ✅
- **Issue:** Bulk delete doesn't show what will happen
- **Solution:** `miro_bulk_delete` returns item count and IDs in preview when `dry_run: true`

---

## Phase 4: Optional Enhancements

### 4.1 MCP Resources ✅ COMPLETE
- **Benefit:** Expose board content as MCP resources for direct access
- **Implementation:** `miro://board/{id}` resource URIs
- **Resources added:**
  - `miro://board/{board_id}` - Board summary with metadata and item counts
  - `miro://board/{board_id}/items` - All items on a board
  - `miro://board/{board_id}/frames` - All frames on a board
- **Tests:** Full test coverage in `resources/resources_test.go`

### 4.2 MCP Prompts ✅ COMPLETE
- **Benefit:** Pre-built prompt templates for common workflows
- **Prompts added:**
  - `create-sprint-board` - Sprint planning board with standard columns
  - `create-retrospective` - Retrospective with What Went Well/Could Improve/Action Items
  - `create-brainstorm` - Brainstorming session with central topic
  - `create-story-map` - User story mapping board
  - `create-kanban` - Kanban board with customizable columns
- **Tests:** Full test coverage in `prompts/prompts_test.go`

### 4.3 Demo GIF
- **Benefit:** Visual demonstration in README
- **Status:** Requires manual screen recording
- **Effort:** 1 hour

### 4.4 Smithery Registry Listing
- **Benefit:** Discoverability for Claude users
- **Effort:** 2 hours
- **Priority:** Low (existing installation methods work well)

### 4.5 Architecture Diagram
- **Benefit:** Better documentation
- **Effort:** 1 hour

### 4.6 Enterprise Compliance Tools
- **Tools:** Legal holds, cases, content logs, classifications
- **Requirement:** Miro Enterprise plan
- **Priority:** Only if Enterprise customers need it

### 4.7 Split Large Test Files (Low Priority)
- **Issue:** `miro/client_test.go` (9,617 lines), `miro/audit/audit_test.go` (36,446 lines)
- **Solution:** Split by domain (boards_test.go, items_test.go, etc.)
- **Priority:** Low - tests work fine as-is

### 4.8 Integration Test Improvements
- **Current:** Require MIRO_TEST_TOKEN
- **Enhancement:** Record/replay HTTP for offline testing
- **Priority:** Low

---

## What NOT To Build

| Don't Build | Reason |
|-------------|--------|
| Comments API | Does not exist in Miro REST API |
| Templates API | Does not exist in Miro REST API |
| npm/npx wrapper | Homebrew/Docker/binaries sufficient |
| Output schemas | Already have via Result struct types |

---

## Competitive Analysis Summary

### vs k-jarzyna/mcp-miro (98 tools)
- Their extra tools are **Enterprise compliance features** (legal holds, cases)
- Performance: Your Go server is 10-20x faster startup, 1/7th memory
- Built-in: Rate limiting, caching, circuit breaker (they have none)
- Unique: Mermaid diagram generation (they don't have)

### vs Official Miro MCP
- Their server is hosted (requires internet)
- Your server is self-hosted (works offline after token setup)
- Both have OAuth 2.1 support

---

## Success Metrics

| Metric | Before | After | Status |
|--------|--------|-------|--------|
| Tool count accuracy | Inconsistent | 76 everywhere | ✅ Done |
| Test coverage visibility | Hidden | Badge in README | ✅ Done |
| MCP Inspector docs | None | Complete guide | ✅ Done |
| Validation duplication | 20+ copies | Helper functions | ✅ Done |
| Magic numbers | Scattered | Centralized constants | ✅ Done |
| Destructive operations | No preview | dry_run parameter | ✅ Done |
| MCP Resources | None | 3 resource templates | ✅ Done |
| MCP Prompts | None | 5 workflow templates | ✅ Done |

### Test Coverage (as of Dec 2025)

| Package | Coverage |
|---------|----------|
| miro | 88.9% |
| audit | 82.1% |
| diagrams | 92.6% |
| oauth | 72.5% (browser flows not testable) |
| tools | 85.0% |

---

## Timeline

| Phase | Duration | Status |
|-------|----------|--------|
| Phase 1: Quick Wins | 1-2 hours | ✅ Complete (except demo GIF) |
| Phase 2: Code Quality | 4-6 hours | ✅ Complete |
| Phase 3: MCP Compliance | 4-8 hours | ✅ Complete |
| Phase 4: Optional | As needed | ✅ Partial (Resources + Prompts complete) |

---

## Miro Developer Terms Compliance

Verified compliance with [Miro Developer Terms of Use](https://miro.com/legal/developer-terms-of-use/):

| Requirement | Status | Notes |
|-------------|--------|-------|
| Use only published APIs | ✅ | All endpoints from official v2 API docs |
| No undocumented features | ✅ | No reverse engineering or hidden endpoints |
| Rate limiting | ✅ | 5-concurrent semaphore + exponential backoff |
| No permanent data storage | ✅ | 2-minute cache TTL for performance only |
| User-controlled auth | ✅ | Users provide their own tokens |
| Branding compliance | ✅ | Named "miro-mcp-server" (acknowledges compatibility, no endorsement claim) |
| No data selling | ✅ | Open source, no commercial data use |

**Notes:**
- Audit logs store execution metadata (timestamps, tool names), not user content
- OAuth uses standard PKCE flow per Miro's documentation
- Users control which boards they access; data handling responsibility falls on them

---

## References

- [MCP Inspector](https://modelcontextprotocol.io/docs/tools/inspector)
- [MCP Best Practices](https://modelcontextprotocol.info/docs/best-practices/)
- [Miro REST API Reference](https://developers.miro.com/reference/api-reference)
- [Miro Developer Terms of Use](https://miro.com/legal/developer-terms-of-use/)
- [Miro Comments API Request](https://community.miro.com/ideas/access-to-comments-via-rest-api-webhooks-6965) (not available)

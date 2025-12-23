# Miro MCP Server Improvement Plan

## Executive Summary

This plan addresses verified improvement opportunities identified through competitive analysis and MCP ecosystem best practices review (December 2024).

**Current State:** Production-ready with 76 tools, best-in-class performance, comprehensive API coverage.

**Key Finding:** The server already has proper input schemas via `jsonschema` struct tags, complete installation options, and full Miro REST API coverage (non-enterprise).

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

## Phase 2: Code Quality (4-6 hours)

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

### 2.4 Define Config Constants
- **Issue:** Magic numbers scattered throughout code
- **Status:** Pending
- **Solution:** Centralize in `miro/constants.go`

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

### 4.1 Smithery Registry Listing
- **Benefit:** Discoverability for Claude users
- **Effort:** 2 hours
- **Priority:** Low (existing installation methods work well)

### 4.2 Architecture Diagram
- **Benefit:** Better documentation
- **Effort:** 1 hour

### 4.3 Enterprise Compliance Tools
- **Tools:** Legal holds, cases, content logs, classifications
- **Requirement:** Miro Enterprise plan
- **Priority:** Only if Enterprise customers need it

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

| Metric | Current | Target |
|--------|---------|--------|
| Tool count accuracy | Inconsistent | 76 everywhere |
| Test coverage visibility | Hidden | Badge in README |
| MCP Inspector docs | None | Complete guide |
| Validation duplication | 20+ copies | 1 helper function |
| Magic numbers | Scattered | Centralized constants |

---

## Timeline

| Phase | Duration | Status |
|-------|----------|--------|
| Phase 1: Quick Wins | 1-2 hours | ✅ Complete (except demo GIF) |
| Phase 2: Code Quality | 4-6 hours | ✅ Partial (validation helpers done) |
| Phase 3: MCP Compliance | 4-8 hours | ✅ Complete |
| Phase 4: Optional | As needed | Optional |

---

## References

- [MCP Inspector](https://modelcontextprotocol.io/docs/tools/inspector)
- [MCP Best Practices](https://modelcontextprotocol.info/docs/best-practices/)
- [Miro REST API Reference](https://developers.miro.com/reference/api-reference)
- [Miro Comments API Request](https://community.miro.com/ideas/access-to-comments-via-rest-api-webhooks-6965) (not available)

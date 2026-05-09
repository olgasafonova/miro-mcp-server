# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Text alignment on shapes**: `miro_create_shape` and `miro_update_shape` now accept `text_align` (left/center/right) and `text_align_vertical` (top/middle/bottom). Previously, text on shapes rendered with the API's default alignment, which was particularly visible on triangles, hexagons, and other non-rectangular shapes where the bounding-box center is not the visual centroid. The new fields are validated against the allowed enums; invalid values fail at the SDK boundary with a clear error.
- **`text_color` + alignment on `miro_bulk_create` shape items**: the bulk schema now accepts `text_color`, `text_align`, and `text_align_vertical` on shape items, closing the gap where these fields worked on single-create but were rejected by `bulk_create` validation.
- **Mindmap child node positions**: `miro_create_mindmap_node` now accepts explicit `x`/`y` for child nodes (previously root-only). Without this, multiple siblings created via the API stacked at the same default position. Supplying explicit coordinates lets agents lay out children spatially.

### Fixed
- **`miro_bulk_delete` and `miro_delete_item` now work transparently on mindmap node IDs**: when the generic `/items/{id}` endpoint returns 400 or 404, the client retries via the experimental `/mindmap_nodes/{id}` endpoint before giving up. Previously, mindmap nodes had to be deleted through `miro_delete_mindmap_node`; bulk delete on a mixed-type list of IDs would fail entirely on any mindmap node. The fallback is gated to 4xx so that 5xx errors don't trigger silent endpoint-swapping.
- **`miro_create_mindmap_node` description corrected**: removed the misleading "bubble" `node_view` value (the underlying API rejects it with 400). Documented that only `text` is reliably supported and noted the new explicit-positioning workflow for child nodes.
- **`miro_create_frame` color description clarified**: explicitly states that frames use a smaller palette than stickies. Sticky-only names like `light_yellow` and `light_green` now have a clear "not valid for frames" hint instead of producing a generic palette error.

### Known limitations
- **`miro_bulk_update` does not yet support shape-specific style fields** (`text_color`, `text_align`, `text_align_vertical`) because it dispatches through the generic `/items/{id}` PATCH endpoint, not the type-specific `/shapes/{id}` PATCH. Use `miro_update_shape` directly for these properties; bulk update remains correct for content, position, geometry, color, and parent.

## [1.20.1] - 2026-05-09

### Changed

- **Internal code-health pass: 8 files lifted from Yellow to Green/Optimal on the CodeScene scale.** No behavior change for MCP callers; same tool list, same response shapes, same error wording. Release exists so existing users on `@latest` pick up the cleaner code on their next reinstall. Files: `main.go` (8.30 → 10.0), `miro/appcards.go` (8.34 → 10.0), `miro/items.go` (8.62 → 10.0), `miro/bulk.go` (8.26 → 9.38), `miro/diagrams/mermaid.go` (7.91 → 9.38), `miro/tags.go` (8.47 → 9.38), `miro/shape.go` (8.82 → 9.38), `miro/upload.go` (8.33 → 9.09).
- **Auth-subcommand CLI error prefixes lowercased** (`Login failed:` → `login failed:`, `Logout failed:` → `logout failed:`) to satisfy Go's staticcheck ST1005 convention. CLI-only; never reaches MCP clients or LLM transcripts.

### Decomposition recipes applied
- Brain `Update*` methods (cc 19–24) → per-section body builders + shared `buildUpdatePosition` / `buildUpdateGeometry` / `updateParentPayload` helpers (preserving the empty-string-nulls-parent semantic).
- Sibling `Create*` / `Update*` methods → shared validation, shared body skeletons, type-specific result wrappers preserved at the public-API boundary.
- Inline anonymous JSON structs → named `raw*` types so parse helpers can be split out (`parseItemSummary` 81 LoC → `minimalItemSummary` + `addItemFullDetails`).
- Per-call regex compilation and per-call map literals promoted to package-level (`edgeLabelPattern`, `tagColorAliases`).

## [1.20.0] - 2026-05-03

### Security

- **Panic in any tool now surfaces as a structured error to the MCP caller.** Previously, if a tool handler panicked, the deferred recover only logged — the MCP caller received what looked like a successful empty response, with no error signal and no audit-log entry. After: the caller gets a clear error with a correlation ID; the panic value is logged server-side only and never reaches the agent. **Behavior change**: clients that observed the old "empty success on panic" should re-review their error handling.
- **API errors no longer include the raw HTTP response body.** During Miro incidents or CDN/edge errors that return HTML pages, `ParseAPIError` previously echoed up to several KB of response body verbatim into the caller-facing error string (internal hostnames, request IDs, `nginx/X.Y.Z` strings could leak into agent transcripts). Now non-JSON or malformed-JSON responses fall back to `http.StatusText(StatusCode)`. JSON errors with a usable `message` field are unchanged.
- **`board_id`, `item_id`, and `org_id` now validated at every Miro API call site.** A prompt-injected agent could previously send `board_id="valid?team_id=victim"` and Go's URL parser would silently split it into a path-pivot to a different endpoint with attacker-controlled query params. Validators now reject any ID outside `^[a-zA-Z0-9_=\-]+$` (max 100 chars) before the request is constructed. Real Miro IDs match this regex; no legitimate workflow regresses. Invalid IDs now get `board_id contains invalid characters` instead of an opaque Miro 4xx after a wasted request. Resource handlers (`miro://board/...`) also validate.

These three fixes close regressions of hard gates graduated 2026-04-25 (`rules/review-patterns.md`: HG-1 dispatcher panic recovery, HG-2 error sanitization, path-injection class). Found by an autonomous-vulnerability-research sweep across the MCP portfolio.

## [1.19.0] - 2026-04-26

### Added
- **`miro_tool_search` discovery tool**. Server-side keyword search across all tool names, titles, categories, and descriptions. Returns up to 50 matches with short description excerpts so the agent can pick a tool to call. Use when you don't know which tool to reach for, or to scope to a category before browsing. Tool count: 91 → 92.
- **`MIRO_TOOLS_PROFILE=full|essentials` env var**. Default `full` registers all 92 tools (preserves existing behavior). `essentials` registers `miro_tool_search` plus 14 high-frequency tools (boards, list/find, search, sticky/text/frame/connector creation, list/get/update/delete items); agents reach the long tail via the discovery tool. Saves ~13K tokens of preload (~84.5% reduction, measured). See [CONFIG.md](CONFIG.md) for details. Unknown values fall back to `full` with a logged warning.

### Fixed
- Translate color names to hex in write paths (#41). Some Miro APIs accept named colors (e.g. `yellow`), others require hex (`#FFEB3B`); this normalizes at the SDK boundary so agents don't need to know the difference.

## [1.18.0] - 2026-04-25

### Added
- **Companion skill**: `skills/miro-workflow/` ships alongside the MCP server. Five workflows (sprint board, retrospective, brainstorm, story map, kanban) compose the 91 atomic tools into common board layouts with documented spatial defaults and color conventions. Skill files organized as SKILL.md + workflows/ + references/ per the agentskills.io spec convention. Description includes negative triggers to disambiguate against `/diagram` and `/feature-scoping`. README updated with a Companion Skill section.

### Fixed
- **Tool descriptions corrected**: `parent_id`, `x`, `y`, and `color` jsonschema descriptions for `miro_create_sticky`, `miro_create_shape`, and `miro_bulk_create` were misleading. Coordinates are frame-**TOP-LEFT** origin with item-**CENTER** placement (previously documented as "frame center", which produced overflowing layouts). Sticky `color` now enumerates the named-only enum; shape `color` clarifies hex requirement. No behavior change at the API level; only the docs clients see.

## [1.17.0] - 2026-04-25

### Security
- **Action required for `miro_share_board` users:** the tool now enforces a server-side allowlist via `MIRO_SHARE_ALLOWED_DOMAINS` (comma-separated). When unset, the server falls back to the domain of the authenticated user's email; when neither is available, all sharing is rejected with a clear error. This prevents prompt-injected agents from exfiltrating board access by inviting attacker-controlled emails. See `SECURITY.md` and `CONFIG.md`.
- `miro_share_board` marked `Destructive: true` so MCP clients prompt before invocation. USE WHEN / DO NOT USE clauses in the tool description constrain agent triggering to direct user instructions.
- `miro_update_board_member` marked `Destructive: true` to prevent silent role escalation (viewer → editor) from prompt-injected agents processing board content.

### Changed
- Bumped `go.opentelemetry.io/otel` to v1.43.0.
- Bumped `github.com/modelcontextprotocol/go-sdk`.

### Documentation
- Documented `MIRO_SHARE_ALLOWED_DOMAINS` in `CONFIG.md` (env-var table) and `SECURITY.md` (Board Sharing Allowlist section).
- Footnoted the destructive sharing tools in the README Board Members table.
- Recorded scope decision: allowlist enforces at the MCP-handler boundary; direct library consumers of `miro.Client.ShareBoard` bypass it intentionally.
- Improved parent-relative coordinate guidance in schema tags.

### Infrastructure
- Added deslop baseline for cloud-routine code-health regression detection.
- Added `CODEOWNERS` to protect workflow files from spam PRs.
- Added `bd` merge driver for `.beads/issues.jsonl` to prevent JSONL conflicts on concurrent PRs.

## [1.16.2] - 2026-04-05

### Changed
- Tool description quality audit: added RETURNS to 55 tools, USE WHEN to 30 tools, FAILS WHEN to 10 tools
- Rewrote `miro_generate_diagram` and `miro_get_export_job_status` descriptions with full USE WHEN / RETURNS / FAILS WHEN / PARAMETERS sections

## [1.16.1] - 2026-04-04

### Fixed
- Server now starts in inspection mode when `MIRO_ACCESS_TOKEN` is not set, allowing MCP registries (Glama, Smithery) to enumerate tool definitions. Tool calls return a clear configuration error instead of crashing at startup.

## [1.16.0] - 2026-03-23

### Added
- **New Tools (3)**: 91 tools total (up from 88)
  - `miro_update_doc` - Update doc format item content (full replacement or find-and-replace)
  - `miro_list_tables` - List tables (data_table_format) on a board
  - `miro_get_table` - Get table metadata by ID

### Notes
- Doc update uses delete+recreate internally (Miro REST API does not support PATCH on doc_format). Item ID changes after update; position is preserved.
- Table tools return metadata only (ID, position, size, timestamps). The Miro REST API does not expose table column definitions or row data; full table content requires the Miro UI or Miro's hosted MCP server.

## [1.15.2] - 2026-03-03

### Fixed
- Suppress pre-initialize `notifications/tools/list_changed` from go-sdk, preventing intermittent connection failures when many MCP servers start simultaneously

## [1.14.1] - 2026-02-16

### Fixed
- Release workflow: eliminated race condition where 5 parallel `softprops/action-gh-release` calls collided during release finalization
- Release workflow: separated build matrix (upload-artifact) from release upload (single `gh release upload` job)
- Release workflow: added `fail-fast: false` so one build failure doesn't cancel all platforms
- MCP Registry: republished with correct SHA256 hashes (v1.14.0 hashes were stale from failed builds)

## [1.14.0] - 2026-02-16

### Added
- **New Tools (6)**: 86 tools total (up from 80)
  - `miro_create_doc` - Create Markdown documents on boards (Doc Format API)
  - `miro_get_doc` - Get doc format item content
  - `miro_delete_doc` - Delete doc format items (with dry_run support)
  - `miro_get_items_by_tag` - Get all items with a specific tag
  - `miro_upload_image` - Upload local image files via multipart form
  - `miro_create_flowchart_shape` - Create flowchart shapes (experimental API)
- **API Tracking**: Weekly GitHub Action to diff Miro's OpenAPI spec and open issues on changes
  - `api-tracking/diff-spec.py` - Python script to diff two OpenAPI specs
  - `api-tracking/spec-baseline.json` - Pinned baseline spec
  - `.github/workflows/api-tracking.yml` - Runs every Monday 09:00 UTC
- **Tool Descriptions**: Added RELATED cross-references between tools for better LLM tool selection
- **Multipart Upload**: New `requestMultipart` client method for file-based endpoints

### Changed
- Updated official Miro MCP comparison table with February 2026 data (15 tools, DSL diagrams, AI context)

## [1.11.1] - 2026-01-05

### Added
- **MCP Registry**: Server now listed on official MCP Registry
- `server.json` metadata for registry integration
- GitHub Actions workflow for automatic registry publishing on release
- MCP label in Docker image for OCI validation
- Support for both Docker/OCI and MCPB binary distribution

### Fixed
- Release workflow homebrew job syntax error

## [1.11.0] - 2026-01-05

### Added
- **New Tool**: `miro_get_board_content` - Get comprehensive board data for AI analysis (77 tools total)
  - Returns structured content with frames, items, connectors, and tags
  - Designed for AI agents to analyze and document boards
- **Diagrams**: `output_mode` parameter to return created items for compound diagrams
- **Diagrams**: `use_stencils` parameter for professional flowchart shapes
  - Uses Miro's flowchart stencil shapes (`flow_chart_terminator`, `flow_chart_decision`, `flow_chart_process`, etc.)
  - Professional color coding with matching border colors
- **Responses**: `detail_level` parameter for rich response mode across tools
- **Responses**: Deep links added to all create operation responses
- **Developer**: Inline examples in tool descriptions for better LLM understanding
- **Developer**: CLAUDE.md for Claude Code guidance

### Fixed
- Lint warning for bool comparisons in GetBoardContent (staticcheck S1002)

### Changed
- Updated comparison with official Miro MCP server (now has ~5 tools with AI-based diagram generation)

## [1.8.0] - 2025-12-23

### Added
- **Reliability**: Transient error retry (502, 503, 504) with exponential backoff
- **Security**: ReDoS protection for Mermaid diagram parser
- **Validation**: `Config.Validate()` method with token/timeout/team ID validation
- **Bulk Operations**: Enhanced error recovery with categorized errors and retriable IDs
- **Health Check**: Enhanced `/health` endpoint with component status and `/health?deep=true` for API connectivity test
- **Observability**: Prometheus metrics endpoint (`/metrics`) with request counts, latencies, error rates
- **DevOps**: Dockerfile with multi-stage build (final image ~15MB)
- **DevOps**: docker-compose.yml with health checks and resource limits template
- **DevOps**: Makefile with 20+ targets (build, test, lint, docker, etc.)

### Changed
- **Dependencies**: Updated MCP SDK v1.1.0 → v1.2.0
- **Dependencies**: Updated jsonschema-go v0.3.0 → v0.4.2
- **Dependencies**: Updated golang.org/x/oauth2 v0.30.0 → v0.34.0
- **Dependencies**: Updated golang-jwt/jwt v5.2.1 → v5.3.0
- **Dependencies**: Updated Go version 1.23.0 → 1.24.0
- **Internal**: Consolidated duplicate caching mechanism (sync.Map → unified *Cache)

### Removed
- **Dead webhook code**: Removed webhook endpoints from HTTP mode (Miro sunset Dec 5, 2025)

## [1.7.0] - 2025-12-22

### Added
- **Distribution**: Homebrew tap (`brew install olgasafonova/tap/miro-mcp-server`)
- **Distribution**: Docker image (`ghcr.io/olgasafonova/miro-mcp-server`)
- **Distribution**: Linux ARM64 binary
- **Distribution**: Install script for macOS/Linux
- **66 tools total**: Complete feature set

### Changed
- Improved installation documentation
- Enhanced platform compatibility

## [1.6.0] - 2025-12-22

### Added
- **Mindmaps**: `miro_get_mindmap_node` - Get node details
- **Mindmaps**: `miro_list_mindmap_nodes` - List all mindmap nodes on board
- **Mindmaps**: `miro_delete_mindmap_node` - Delete a mindmap node
- **Frames**: `miro_get_frame` - Get frame details
- **Frames**: `miro_update_frame` - Update frame title/color/size
- **Frames**: `miro_delete_frame` - Delete a frame
- **Frames**: `miro_get_frame_items` - List items inside a frame

### Changed
- Enhanced mindmap API support with v2-experimental endpoints
- Improved frame management capabilities

## [1.5.0] - 2025-12-21

### Added
- **App Cards**: `miro_create_app_card` - Create app cards with custom fields
- **App Cards**: `miro_get_app_card` - Get app card details
- **App Cards**: `miro_update_app_card` - Update app card fields/status
- **App Cards**: `miro_delete_app_card` - Delete an app card
- **Tags**: `miro_update_tag` - Update tag name/color
- **Tags**: `miro_delete_tag` - Delete a tag
- **Connectors**: `miro_list_connectors` - List all connectors on board
- **Connectors**: `miro_get_connector` - Get connector details
- **Connectors**: `miro_update_connector` - Update connector style/caption
- **Connectors**: `miro_delete_connector` - Delete a connector
- **Groups**: `miro_list_groups` - List all groups on board
- **Groups**: `miro_get_group` - Get group details
- **Groups**: `miro_get_group_items` - List items in a group
- **Groups**: `miro_delete_group` - Delete a group
- **Members**: `miro_get_board_member` - Get member details
- **Members**: `miro_update_board_member` - Update member role
- **Members**: `miro_remove_board_member` - Remove member from board

### Changed
- Expanded from 50 to 66 tools
- Enhanced CRUD coverage across all domains

## [1.4.2] - 2025-12-21

### Fixed
- **CRITICAL: Sequence diagram layout**: Fixed major bug where flowchart layout algorithm was being applied to sequence diagrams, destroying participant positions and causing chaotic rendering
- Sequence diagrams now correctly preserve parser-set positions (participants horizontal, messages vertical)
- Added support for `startX`/`startY` offset in sequence diagrams

## [1.4.1] - 2025-12-21

### Fixed
- **Sequence diagram visibility**: Lifelines now visible at 10px width (was 4px)
- **Anchor appearance**: Anchors now match lifeline color (#90CAF9) instead of white
- **Miro API compliance**: Anchor size increased to 8px (Miro minimum requirement)

## [1.4.0] - 2025-12-21

### Added
- **Sequence diagram rendering**: Full Miro output for sequence diagrams
  - Participant headers (rectangle or circle for actors)
  - Vertical lifelines below each participant
  - Horizontal message arrows with labels
  - Support for sync (`->>`) and async (`-->>`) messages
  - Proper Y positioning for message ordering
- **New sequence converter tests**: 10 comprehensive tests for sequence diagram rendering

### Changed
- `ConvertToMiro` now auto-detects diagram type and uses appropriate converter
- Edge struct extended with Y position for sequence message placement
- Improved diagrams package coverage: 71.2% → 73.4%

### Technical
- Sequence messages rendered via anchor shapes + connectors
- Lifelines as thin vertical rectangle shapes
- Maintains flowchart compatibility (no breaking changes)

## [1.3.0] - 2025-12-21

### Added
- **Verbose logging**: Added `--verbose` flag for debug logging
- **Diagram benchmarks**: Comprehensive performance benchmarks for Mermaid parsing and layout algorithms
- **Improved error messages**: Diagram parsing errors now include helpful suggestions and hints
- **New diagram error types**: Structured DiagramError with error codes, line numbers, and actionable suggestions

### Changed
- Updated to 50 tools total
- Enhanced diagram validation with early input checking
- Improved test coverage across packages:
  - miro/: 71.9%
  - miro/audit: 78.2%
  - miro/diagrams: 71.2%
  - miro/oauth: 46.6% (up from 31.3%)
  - miro/webhooks: 53.2% (up from 40.8%)

### Fixed
- Various test compilation errors in client_test.go

## [1.2.0] - 2025-12-XX

### Added
- Mermaid diagram generation with `miro_generate_diagram` tool
- Flowchart and sequence diagram parsing
- Sugiyama-style auto-layout algorithm
- Support for multiple node shapes (rectangle, rounded, diamond, circle, hexagon)

## [1.1.0] - 2025-12-XX

### Added
- OAuth 2.1 with PKCE flow
- Token auto-refresh
- Audit logging (file and memory backends)
- Webhook support with SSE streaming
- Export tools for Enterprise plans

## [1.0.0] - 2025-12-XX

### Added
- Initial release with 48 core tools
- Board management (list, create, copy, delete)
- Item creation (sticky notes, shapes, text, connectors, frames, cards, images)
- Bulk operations
- Tag management
- Rate limiting and caching
- Dual transport (stdio and HTTP)

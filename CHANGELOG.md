# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- **Connector reads work against the live API.** `miro_list_connectors` and `miro_get_connector` parsed the connected item's id from a `"item"` key the API never sends (endpoint ids came back empty), and typed the endpoint's relative anchor as numbers when the wire carries percentage strings (`"x": "100%"`) — so any connector attached to an item failed the whole call with a parse error. Both found 18-08-2026 while porting the SVG dialect to miro-cli: the unit fixtures encoded the imagined wire shape, so tests passed while every live read was broken. Fixtures now mirror a captured live response. Consequence fixed along the way: `miro_read_board_svg` and the board-summary connector enrichment swallow connector errors as best-effort, so connectors silently never appeared in either — they render now.

- **Page-size caps corrected to the API's real maximum of 50.** Connectors, frame items, and mindmap nodes all clamped requested limits to 100, but the live endpoints answer 400 to anything above 50 (`limit=51` and `limit=100` both verified 18-08-2026). Affected paths either errored outright (`miro_get_frame_items`, `miro_list_mindmap_nodes` with a large limit) or silently degraded (the frame-scoped SVG read's child listing, the connector fetches above). All clamps now cap at 50, matching the `MaxItemLimit` the plain items listing already enforced.

- **`miro_update_from_svg` accepts minimal deletion markers and deletes connectors.** A bare `<rect data-miro-id="X" data-deleted="true"/>` was skipped as degenerate geometry (an empty `<text>` marker likewise as empty content) — deletion needs identity, not geometry, so markers now bypass those checks. And a `<line>` deletion routed to the generic items endpoint, which answers 404 for connectors; it now routes to the connectors endpoint. Both found by the miro-cli port's live smoke test and fixed in both codebases.

## [1.24.0] - 2026-08-18

### Added

- **`miro_list_diagrams` + `miro_get_diagram` (2)**: read native diagram items via `GET /v2/boards/{id}/diagrams`, an endpoint that answers real data on a plain personal access token but appears in neither Miro's OpenAPI spec baseline nor the docs — a live probe on 15-08-2026 is the evidence. The surface is read-only: `POST` returns 405 `methodNotSupported`, so diagram creation stays with Miro's UI and hosted tooling (the official connector's `diagram_create_mermaid`). Returns item metadata (id, title, position, size, parent frame, timestamps); the diagram's internal nodes and edges are not exposed by the API.

  Distinct from `miro_generate_diagram`, which parses Mermaid locally and draws regular shapes and connectors — those never show up as native diagram items, and both tool descriptions now say so. Tool count: 106 → 108.

- **`miro_get_org_audit_logs` (1)**: wraps Miro's organization audit log, `GET /v2/audit/logs` — who did what across the Miro workspace, including actions taken outside this server. Enterprise plan and the `auditlogs:read` scope; 403 and 404 carry a hint saying so, because on this endpoint a non-Enterprise org, a missing scope, and a genuinely absent resource are indistinguishable from the status alone. A 400 deliberately does not get the hint: that is what a malformed time window returns, and a plan hint would send the caller after the wrong problem.

  `created_after` and `created_before` are both required and validated locally, since the API has no default window and answers a missing bound with an opaque 400. The nested `context` object is flattened, so `team_id`, `team_name`, `organization_id` and `ip` sit directly on each event rather than making callers walk it. Miro retains 90 days; older events are only available via the CSV export in the admin UI, which this does not wrap. Tool count: 105 → 106.

- **`miro_who_am_i` (1)**: token introspection via `GET /v1/oauth-token` — whose token this server is running on, which team and organization it is scoped to, and which scopes it carries. The REST twin of the official connector's `user_who_am_i`, and the missing first step when debugging a 403: a token without the needed scope and a genuinely forbidden resource are indistinguishable from the status alone, so check the scopes before blaming the endpoint.

  The endpoint is the one piece of token introspection Miro never ported to v2, so the client gains a narrow v1 base-URL path (`V1BaseURL` + `requestV1`) alongside the existing v2 and v2-experimental ones. Deliberately uncached: the point of the tool is a live answer. Wire shape captured by live probe 18-08-2026 — a plain personal access token answers 200 with user, team, organization, application and scopes. Tool count: 109 → 110.

- **SVG canvas round trip: frame-scoped read, update-from-svg diff, richer create dialect.** Grounded in David Balkind's canvas-composer probe campaign (Aug 2026), which mapped the official Miro MCP's SVG surface against this server:

  - `miro_read_board_svg` gains `frame_id`: render one frame with frame-relative child coordinates (the official composer's reads are whole-board only).
  - New `miro_update_from_svg`: a diff keyed on `data-miro-id` — update in place, remove via `data-deleted`, create additively. Two failure classes: malformed XML fails the whole request; semantic errors land in a `failed` list while the rest of the batch applies. Read output is re-submittable by construction (the official composer's reads are re-escaped and are not). Tool count: 108 → 109.
  - `miro_create_from_svg` dialect extended: `data-type` sticky/frame hints, 3-point polygon → triangle, `image href`, and `line` + `data-start`/`data-end` → connector via two-pass authored-id resolution.
  - Eventual-consistency caveats added to `miro_get_board_summary` and `miro_list_items`: the REST index lags composer-created items, and interactive widgets never appear in the REST enumeration.

### Changed

- **CodeScene Code Health raised to 10.0 across the repo** (measured per file with the CodeScene CLI/MCP). 63 files refactored — structure only: test setup/assertion boilerplate extracted into local helpers, duplicated test pairs merged into table-driven tests with every case preserved as a named subtest, and five oversized files split by responsibility (`client_test.go` → utils/export/bulk/resilience, `handlers_test.go` and `mock_client_test.go` → per-area files, `oauth_test.go` → provider/store/server/flow, `audit_test.go` → memory/file/factory). No exported signature, error string, JSON tag, or API path changed. Five files stay Green at 9.38–9.68 rather than 10.0 because their remaining finding is the exported API's own string parameters (`miro/cache.go`, `tools/share_allowlist.go`, `miro/diagrams/errors.go`, `miro/diagrams/mermaid.go`, `miro/diagrams/sequence.go`) — reshaping those surfaces for a score is contortion, not health.

- **`miro_bulk_create` now issues one request instead of one per item.** It called the typed single-create endpoint N times in parallel; it now posts to Miro's native `POST /v2/boards/{board_id}/items/bulk`. `MaxBulkItems` was already 20, the endpoint's own cap, so every request this server accepts fits in a single call — 20 stickies cost 1 request rather than 20, which is the rate-limit pressure the change is for.

  The endpoint is **transactional**: if one item fails, none are created. That conflicts with this server's per-item contract, where 19 of 20 may land, so the fallback is deliberately narrow. A `400` means the batch was rejected on validation, and transactionality proves nothing was created — so it falls back to the old per-item fan-out, which cannot double-create and is the only way to report *which* item was bad. Any other failure (429, 5xx, network, timeout) leaves the outcome unknown, because the transaction may have committed with the response lost; there it does **not** fall back, and reports every item failed and retriable with a message saying the outcome is not provable. Falling back on those would be the bug that duplicates a whole batch.

  Item bodies are built by the same helpers the typed `Create*` methods use, so a bulk-created item is byte-identical to a singly-created one. That matters most for sticky notes, whose fill colours are a named enum the API rejects hex for; a hand-rolled bulk mapping is exactly where that would drift. `buildStickyCreateBody` was extracted from `CreateSticky` for this, guarded by the existing sticky tests.

- **`miro_get_audit_log`'s description now says which audit log it means.** It reads this server's local execution log, not Miro's — a distinction the name alone does not carry, and one that got easier to trip over now that both exist. Both descriptions name the other tool. The name itself is unchanged; renaming it would break existing callers for a problem that sharper wording fixes.

- **README account-compatibility table corrected.** It claimed every plan had "full access to all tools", which was already untrue for the three PDF/SVG export tools and would now also be untrue for the org audit log. It now gives 106 tools on Free/Team/Business (of 110 at release) and names the four Enterprise-only tools. `miro_get_board_picture` works on every plan and is not one of them.

### Fixed

- **A client that omits `params` on `notifications/initialized` no longer crashes the server.** MCP makes `params` optional on notifications, so this was a plain interop crash: a spec-compliant client sending `{"jsonrpc":"2.0","method":"notifications/initialized"}` took the process down with a SIGSEGV before it could answer anything else. Sending `params:{}` avoided it, which is why it went unnoticed — the Go SDK's own client always populates the field.

  The fault was in `mcp-otel-go`, fixed in v0.2.1 and picked up here. `ServerRequest[P].GetParams()` returns its typed field verbatim, so an absent `params` arrives as a non-nil interface wrapping a nil pointer; the `== nil` guard was false against it and the accessor dereferenced the nil receiver. Verified against this binary: v0.2.0 exits 2 with a SIGSEGV after answering only `initialize`, v0.2.1 exits 0 and serves `tools/list` afterwards.

- **Cache hints now cover every cacheable method, not just `tools/list`.** SEP-2549 requires `ttlMs` on cacheable results, and the SDK ships the field as an `int` with no `omitempty`, so any method nobody configured serialised `ttlMs: 0` — which the spec reads as "immediately stale". Measured on the v1.23.0 binary, `prompts/list`, `resources/list` and `resources/templates/list` all advertised `0`, so a compliant client re-fetched them every turn. All three now advertise 30 minutes, matching `tools/list`.

  `resources/read` advertises 1 minute, matching `miro.ItemCacheTTL` — the shortest cache behind its handlers. Within that window the server would serve the same bytes from its own cache anyway, so the client saves a round trip; past it the server refetches, and a longer hint would leave the client holding content the server had already replaced.

  A completeness test now fails if any method `mcp-cache-go` knows to be cacheable has no TTL configured, so a seventh cacheable result type in a future SDK cannot ship as `ttlMs: 0` unnoticed.

## [1.23.0] - 2026-08-13

### Added

- **Comment tools (5, v2-experimental)**: `miro_create_comment`, `miro_list_comments`, `miro_get_comment`, `miro_reply_comment`, `miro_resolve_comment` — comment threads on boards and items. The endpoints are live on `/v2-experimental/boards/{board_id}/comments` but absent from Miro's OpenAPI spec, so the wire shapes were captured by live probe (13-08-2026): a comment is a thread with `messages[]`, replies POST to `.../comments/{id}/messages`, resolve is `PATCH {"resolved":bool}` and works in both directions. The API ignores position coordinates on create and DELETE returns 405, so the tools expose neither. `item_id` attaches a thread to an item (`position.type` becomes `attached`). 403/404 responses carry the same experimental-availability hint as the code widget tools. Tool count: 98 → 103.

- **Canvas SVG tools (2, local transform)**: `miro_read_board_svg` renders a board's items as an SVG document computed locally from item geometry — frames as dashed outlines (drawn first, so they sit under their children), shapes and stickies as filled rects/ellipses, text as text, connectors as lines between item centers, every element tagged with `data-miro-id`/`data-miro-type`. `miro_create_from_svg` parses a constrained SVG subset (rect, circle, ellipse, text, nested `g transform="translate"`) and creates matching shapes and text items; unsupported elements are itemized as skipped with reasons, never silently dropped, and a mid-batch failure still reports every item that landed. Caps: 1 MiB source, 200 elements per call. Same local-transform philosophy as the Mermaid parser — no export job, no external service. Tool count: 103 → 105.

### Fixed

- **Shape kind now populated in full-detail item listings.** The API carries a shape's kind (`rectangle`, `circle`, ...) in `data.shape`, which the list-item parser never read, so `ItemStyleInfo.Shape` was declared but always empty. Found by the SVG round-trip test (a circle rendered as a rect); folded into `style.shape` in `detail_level=full` responses.

- **Board listings now carry owner, team and timestamps.** `miro_list_boards` and `miro_find_board` return `team_id`, `team_name`, `owner` (`id` + `name`), `created_at` and `modified_at` alongside the existing `id`, `name`, `description` and `view_link`. `GET /v2/boards` already returns all of these on every board in the page, so nothing here costs an extra request — the projection was simply dropping them. Segmenting boards by team or owner, and sorting by recency, no longer needs a `miro_get_board` per row, and `team_id` feeds straight back into the `team_id` filter.

  Absent values stay absent: a board with no owner, team or timestamps omits those keys entirely rather than emitting `null` or a zero date like `0001-01-01T00:00:00Z`.

## [1.22.0] - 2026-07-21

### Added

- **Code widget tools (6, v2-experimental)**: `miro_create_code_widget`, `miro_get_code_widget`, `miro_list_code_widgets`, `miro_update_code_widget`, `miro_move_code_widget`, `miro_delete_code_widget` — syntax-highlighted code snippets on boards, mapped 1:1 to Miro's six `/v2-experimental/boards/{board_id}/code_widgets` endpoints (detected by the api-tracking workflow on 27-04-2026). Field caps validated client-side per the spec (code ≤ 6000 chars, title ≤ 100). List responses return 80-char code previews with cursor pagination (default 50, max 100); full source via the get tool. 403/404 responses from these endpoints carry an added hint that the experimental API may be unavailable for the account or plan, since a plain 404 is ambiguous there. Delete supports `dry_run=true` and is annotated `Destructive`. Tool count: 92 → 98. Closes bead `miro-mcp-server-yl6`.

## [1.21.1] - 2026-05-09

### Changed

- **Internal code-health pass: 6 files lifted from Yellow to Green on the CodeScene scale.** No behavior change for MCP callers; same tool list, same response shapes, same error wording. Release exists so existing users on `@latest` pick up the cleaner code on their next reinstall. Files: `miro/shape.go` (8.54 → 9.38), `miro/cache.go` (8.47 → 9.38), `miro/desirepath/normalizers.go` (8.81 → 9.68), `miro/audit/memory.go` (8.72 → 9.68), `miro/boards_summary.go` (8.53 → 9.61), `miro/diagrams.go` (8.62 → 9.68), `miro/diagrams/sequence.go` (8.74 → 9.09).

### Decomposition recipes applied
- Brain `Parse` / `GenerateDiagram` / `GetBoardContent` methods (cc 35–38, 150–206 LoC) decomposed into per-phase helpers: argument normalization → input aggregation → result assembly → optional enrichment → message build. Same shape as the v1.20.1 `mermaid.go` lift.
- Five-argument `applyOptionalEnum` / `applyOptionalEnumPtr` collapsed to three by bundling `{styleKey, errorTag, allowed}` into an `enumField` value object; `shapeTextAlignField` / `shapeTextAlignVerticalField` promoted to package-level enums so call sites no longer pass primitive triplets.
- Duplicate `args → shapeCoreBody` mapping between `CreateShape` and `CreateShapeExperimental` replaced by per-args `toCoreBody()` methods plus a shared `validateShapeCreateArgs` validator.
- Brain Method `evictOldest` (cc 12, depth 4) split into `findExpiredKeys` / `findOldestAccessKeys` / `evictionTargetCount` / `removeEntries` / `rebuildAccessList`, preserving the expired-first then ~10%-of-oldest-accessed eviction order.
- Sequence-diagram `Parse` line dispatch extracted to `parseState` + `dispatchLine` + per-pattern `tryParse*` handlers, with output-side helpers (`buildParticipantNode`, `buildMessageEdge`, `messageStyleForArrow`, `applyArrowStyleForMessage`, `setSequenceDimensions`) split out of the parser.

### Documentation

- README: added `miro_get_desire_paths` to the **All 92 Tools** category listing (the tool was registered in `tools/definitions.go` but missing from the README); health-check JSON example bumped from `1.15.2` to current.

After this pass, only `miro/diagrams/layout.go` (8.18) remains in Yellow; the Sugiyama-style layered-graph layout algorithm is intrinsically high-cyclomatic and was deferred until golden-file tests can guard the output against drift.

## [1.21.0] - 2026-05-09

### Added
- **Text alignment on shapes**: `miro_create_shape` and `miro_update_shape` now accept `text_align` (left/center/right) and `text_align_vertical` (top/middle/bottom). Previously, text on shapes rendered with the API's default alignment, which was particularly visible on triangles, hexagons, and other non-rectangular shapes where the bounding-box center is not the visual centroid. The new fields are validated against the allowed enums; invalid values fail at the SDK boundary with a clear error.
- **`text_color` + alignment on `miro_bulk_create` shape items**: the bulk schema now accepts `text_color`, `text_align`, and `text_align_vertical` on shape items, closing the gap where these fields worked on single-create but were rejected by `bulk_create` validation.
- **Mindmap child node positions**: `miro_create_mindmap_node` now accepts explicit `x`/`y` for child nodes (previously root-only). Without this, multiple siblings created via the API stacked at the same default position. Supplying explicit coordinates lets agents lay out children spatially.
- **`miro_bulk_update` type-dispatch**: bulk updates now accept an optional `type` field on each item (`shape`, `sticky_note`, `text`). When set, the update routes to the type-specific endpoint (`UpdateShape` / `UpdateSticky` / `UpdateText`) and accepts type-specific fields like `text_align`, `text_color`, and sticky color names. Backward-compatible: items without `type` still go through the generic `/items/{id}` endpoint with no behavior change.

### Fixed
- **`miro_bulk_delete` and `miro_delete_item` now work transparently on mindmap node IDs**: when the generic `/items/{id}` endpoint returns 400 or 404, the client retries via the experimental `/mindmap_nodes/{id}` endpoint before giving up. Previously, mindmap nodes had to be deleted through `miro_delete_mindmap_node`; bulk delete on a mixed-type list of IDs would fail entirely on any mindmap node. The fallback is gated to 4xx so that 5xx errors don't trigger silent endpoint-swapping.
- **`miro_create_mindmap_node` description corrected**: removed the misleading "bubble" `node_view` value (the underlying API rejects it with 400). Documented that only `text` is reliably supported and noted the new explicit-positioning workflow for child nodes.
- **`miro_create_frame` color description clarified**: explicitly states that frames use a smaller palette than stickies. Sticky-only names like `light_yellow` and `light_green` now have a clear "not valid for frames" hint instead of producing a generic palette error.

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

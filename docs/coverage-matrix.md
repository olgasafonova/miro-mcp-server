# Miro Family Coverage Matrix

Three-surface audit of the Miro tooling family, produced 15-08-2026 (bead 8de).
Columns: **server** = miro-mcp-server (106 tools: 105 in `tools/definitions.go` +
`miro_tool_search` in `tools/search.go`), **cli** = miro-cli (23 command modules),
**apps** = miro-mcp-apps (7 shipped views).

Per-surface meaning of *full*:

- **server** — every public REST domain has tools. Hosted-only connector features
  are not gaps.
- **cli** — verb-level parity with the server.
- **apps** — every read domain with a visual payoff has a view. Not 1:1.

Cell values: `covered` / `partial` / `missing` / `n/a`.

## Matrix

| API domain | server | cli | apps | Notes |
|---|---|---|---|---|
| Boards (CRUD, copy, find, summary, content) | covered | covered | covered | Views: recent-boards, board-summary |
| Board members + sharing | covered | covered | missing | View candidate (bead 6wp) |
| Items generic (list, get, search, bulk, update, delete) | covered | covered | covered | View: list-items |
| Item creation (sticky, shape, text, card, image, document, embed, frame, flowchart, grid) | covered | covered | covered | View: sticky-clusters covers the read side |
| App cards | covered | covered | n/a | No visual payoff beyond list-items |
| Tags | covered | covered | missing | View candidate: tag-usage map (bead 6wp) |
| Groups | covered | covered | n/a | Low visual payoff |
| Connectors | covered | covered | covered | View: connectors |
| Frames | covered | covered | covered | View: frame-overview |
| Mindmaps (v2-experimental) | covered | covered | missing | View candidate: tree layout (bead 6wp) |
| Code widgets (v2-experimental) | covered | **partial** | n/a | CLI has `list` only; 5 verbs missing (miro-cli bead 7fm) |
| Comments (v2-experimental, undocumented) | covered | **missing** | covered | CLI port scoped as hpv part 2 (P1). View: comments |
| Canvas SVG (local transform) | covered | **missing** | covered | CLI port scoped as hpv part 3; apps view shipped 15-08-2026 as miro-mcp-apps PR #4 (merge + Claude Desktop pass pending, bead 06g) |
| Doc formats (`/docs`, Markdown) | covered | **partial** | n/a | CLI has create-doc only; get/update/delete hit `/documents`, the wrong family (miro-cli bead 194) |
| Data tables (`data_table_formats`, read) | covered | covered | missing | Public /v2, confirmed live. View candidate (bead 6wp) |
| Data table rows / sync | n/a | n/a | n/a | Hosted-only: rows paths 404 (probe log below) |
| Native diagrams (read) | **missing** | **missing** | missing | New gap found by probe: GET `/v2/boards/{id}/diagrams` answers 200 (bead x71). Apps view blocked on server tools |
| Mermaid diagram generation (local) | covered | covered | n/a | Local mermaid-to-shapes transform on both; hosted mermaid create is connector-only (POST diagrams = 405) |
| Uploads (multipart image/document) | covered | covered | n/a | |
| Export (board picture, export jobs) | covered | covered | missing | View candidate: export status (deferred; low frequency) |
| Org audit logs (Enterprise) | covered | covered | n/a | |
| Local execution audit log | covered | n/a | n/a | Server-side telemetry concept |
| Desire paths report | covered | n/a | n/a | Server-side telemetry concept |
| Token introspection (`/v1/oauth-token`) | missing | missing | n/a | Public, answers 200. Small who-am-i tool (bead epr, P4) |
| Spaces | n/a | n/a | n/a | Hosted-only: all path shapes 404 (probe log) |
| Sections | n/a (watch) | n/a | n/a | Route exists (400, not 404) but unusable with plain PAT — watchlist (bead u9a) |
| Prototypes | n/a (watch) | n/a | n/a | Same signature as sections — watchlist (bead u9a) |
| Layout DSL | n/a | n/a | n/a | Hosted-only: 404 |

## Probe log (15-08-2026)

Plain PAT against `api.miro.com`, board `uXjVHWiWdK4=` (AI Playground) where
board-scoped. Spec-baseline contains none of these paths; live probes are the
evidence, per bead 7rh. Differential control: a garbage path returns
`404 "No endpoint GET …"`, so a 400 means the route exists.

| Path | Status | Reading |
|---|---|---|
| `GET /v2/boards/{id}/diagrams` | 200 (1 item) | Public read endpoint, server gap (x71) |
| `POST /v2/boards/{id}/diagrams` | 405 | Creation hosted-only |
| `GET /v2/boards/{id}/data_table_formats` | 200 | Already covered (plain /v2, not experimental) |
| `GET …/data_table_formats/{id}/rows` | 404 | Row sync hosted-only |
| `GET /v2/boards/{id}/sections` (list, by-id, ?limit) | 400 Validation failure | Route present, feature-gated — watch (u9a) |
| `GET /v2/boards/{id}/prototypes` (list, by-id, ?limit) | 400 Validation failure | Route present, feature-gated — watch (u9a) |
| `GET /v2/spaces`, `/v2-experimental/spaces`, `/v2/teams/{team}/spaces` | 404 | Hosted-only |
| `GET /v2-experimental/boards/{id}/layout`, `/layouts` | 404 | Hosted-only |
| `GET /v1/oauth-token` | 200 | Public token introspection (epr) |
| `GET /v2-experimental/boards/{id}/comments` | 200 | Control: known-good undocumented endpoint |
| `GET /v2/boards/{id}/zzznotreal` | 404 "No endpoint" | Differential control |

## Ecosystem context

MCP Apps (SEP-1865) went Final in the MCP 2026-07-28 release and renders in
Claude, ChatGPT, VS Code, Goose, and Postman. Miro ships no official MCP Apps
views as of 15-08-2026, so the apps column competes with nobody: every view in
bead 6wp extends the only MCP Apps surface in this niche. Claude Desktop
rendering requires the JSON Schema 2020-12 dialect fix (miro-mcp-apps
`5f350a2`, `withDialectFix` in main.ts).

## Execution queue spawned from this matrix

| Bead | Repo | Priority | Gap |
|---|---|---|---|
| x71 | miro-mcp-server | P2 | Native diagram read tools (list + get) |
| u9a | miro-mcp-server | P3 | api-tracking probe watchlist (sections, prototypes, spaces, layout, rows) |
| 6wp | miro-mcp-server | P3 | Apps view candidates: tags map, data-table, mindmap tree, board members |
| epr | miro-mcp-server | P4 | who-am-i token introspection tool |
| hpv | miro-cli | P1 | (pre-existing) comments 5 verbs + canvas SVG 2 verbs |
| 7fm | miro-cli | P2 | codewidgets create/get/update/move/delete |
| 194 | miro-cli | P3 | doc-format get/update/delete verbs |
| 06g | miro-mcp-server | — | apps SVG view shipped as miro-mcp-apps PR #4; merge + host pass remain |

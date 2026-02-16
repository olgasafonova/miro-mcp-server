# Miro MCP Server

## Project
Go MCP server for Miro REST API v2. 89 tools across boards, items, diagrams, mindmaps, tags, groups, connectors, export, and audit. OAuth2 + token auth. Single binary with stdio + HTTP transport.

## Architecture
- `main.go` — entry point, dual stdio/HTTP transport, health/metrics endpoints, token validation
- `miro/client.go` — HTTP client with connection pooling (100 max idle, 10 per host)
- `miro/cache.go` — LRU cache with 2-minute TTL
- `miro/circuitbreaker.go` — circuit breaker isolating failing endpoints
- `miro/ratelimit.go` — adaptive rate limiting using Miro response headers
- `miro/errors.go` — structured errors with retryable flag
- `miro/diagrams/` — Mermaid parser for flowchart/sequence diagram generation
- `miro/audit/` — local execution audit log (JSON file-based)
- `miro/oauth/` — OAuth2 authentication flow
- `tools/definitions.go` — ToolSpec declarations with annotations (ReadOnly, Destructive, Idempotent)
- `tools/handlers.go` — HandlerRegistry with generic makeHandler[Args, Result] + panic recovery + audit logging
- `prompts/prompts.go` — MCP workflow prompts (sprint board, retro, brainstorm, story map, kanban)
- `resources/resources.go` — MCP resource URIs (board summary, items, frames)
- `evals/` — tool selection + argument correctness benchmarks

## Key Patterns
- `makeHandler[Args, Result]` generic wraps every tool handler with: panic recovery, timing, audit event creation
- `HandlerRegistry.buildHandlerMap()` maps Method name -> registration function; adding a tool = one map entry
- `ToolSpec` annotations drive MCP tool hints (ReadOnly, Destructive, Idempotent, OpenWorld)
- All API methods live on the `MiroClient` interface for testability (mock in `tools/mock_client_test.go`)
- Mermaid diagrams parsed locally (no external service), supporting flowchart + sequenceDiagram

## Tool Categories (89 total)
- **Board Management** (9): list, find, get, create, copy, update, delete, summary, content
- **Board Members** (5): list, get, share, update, remove
- **Create Items** (15): sticky, shape, flowchart shape, text, connector, frame, card, app card, image, document, embed, bulk create/update/delete, sticky grid
- **Read Items** (7): list, list all (paginated), get, search, get image/document/app card
- **Update/Delete** (9): update/delete for sticky, shape, text, card, image, document, embed, generic item
- **Tags** (9): create, list, get, attach, detach, get item tags, get items by tag, update, delete
- **Connectors** (4): list, get, update, delete
- **Groups** (7): create, ungroup, list, get, get items, update, delete
- **Mindmaps** (4): create, get, list, delete
- **Frames** (4): get, update, delete, get items
- **Doc Formats** (3): create, get, delete (Markdown input)
- **Upload** (4): upload image, upload document, update image from file, update document from file (multipart)
- **App Cards** (2): update, delete (create and get counted above)
- **Export** (4): board picture, create job, status, results
- **Diagrams** (1): generate from Mermaid
- **Audit** (1): query local execution log
- **Desire Paths** (1): report tool usage patterns

## Build & Test
```bash
make check     # fmt-check + vet + lint + test
make test      # go test -v ./...
make lint      # golangci-lint run
make build     # build binary
```

## Decision Log
`~/Documents/remote-v/Projects/Miro MCP Server - Decision Log.md` — rationale for API tracking, release workflow design, and other architectural choices. Check before making operational decisions.

## Cross-References (sister MCP servers)
- ToolSpec "USE WHEN" descriptions: ~/Projects/gleif-mcp-server/tools/definitions.go
- LRU cache with TTL: ~/Projects/productplan-mcp-server/pkg/productplan/cache.go
- Evals framework: ~/Projects/productplan-mcp-server/evals/runner.go
- Circuit breaker + deduplicator: ~/Projects/nordic-registry-mcp-server/internal/infra/resilience.go
- Incremental scan state: ~/Projects/code-to-arch-mcp/internal/infra/persist.go
- Dual transport pattern: ~/Projects/figjam-mcp/main.go (lines 311-442)

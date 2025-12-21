# Session Handover - Miro MCP Server

> **Date**: 2025-12-21
> **Project**: miro-mcp-server
> **Location**: `/Users/olgasafonova/go/src/miro-mcp-server`
> **Version**: v1.2.0-dev (unreleased)
> **Latest Commit**: `084d51b`

---

## Current State

**44 MCP tools** for Miro whiteboard control. Phases 1-6 complete.

### Phase 6: AI Diagram Generation (NEW)
- `miro_generate_diagram`: Parse Mermaid flowchart syntax and create visual diagrams on Miro boards
- Supports: flowchart/graph keywords, TB/LR/BT/RL directions, 5 node shapes, edge labels, subgraphs
- Auto-layout: Sugiyama-style layered algorithm with topological ordering

**Release**: https://github.com/olgasafonova/miro-mcp-server/releases/tag/v1.1.0

---

## Quick Start

```bash
# Build
go build -o miro-mcp-server .

# Run (stdio)
MIRO_ACCESS_TOKEN=xxx ./miro-mcp-server

# Run (HTTP with webhooks)
MIRO_ACCESS_TOKEN=xxx MIRO_WEBHOOKS_ENABLED=true ./miro-mcp-server -http :8080

# Test
go test ./...
```

---

## Key Docs

| File | Purpose |
|------|---------|
| `CLAUDE.md` | Architecture, patterns, how to add tools |
| `ROADMAP.md` | Implementation status, future plans |
| `docs/PHASE5_PLAN.md` | Phase 5 design details |

---

## What's Next?

Possible Phase 7 ideas:
- Multi-board operations (bulk actions across boards)
- Board templates (create from template)
- Advanced search (fuzzy matching, filters)
- Presentation mode tools
- Sequence diagrams (extend Mermaid parser)
- Class diagrams (extend Mermaid parser)

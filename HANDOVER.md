# Session Handover - Miro MCP Server

> **Date**: 2025-12-21
> **Project**: miro-mcp-server
> **Location**: `/Users/olgasafonova/go/src/miro-mcp-server`
> **Version**: v1.4.2 (ready for release)
> **Latest Session**: CRITICAL Sequence Diagram Layout Fix

---

## Current State

**50 MCP tools** for Miro whiteboard control. Phases 1-6 complete. **Sequence diagram rendering NOW WORKING.**

All tests passing. Build works. v1.4.2 ready for release.

```bash
# Verify
go build -o miro-mcp-server .
go test ./...
```

---

## Just Completed

### v1.4.2 - CRITICAL Layout Fix (Ready for Release)

**Root Cause**: The `Layout()` function in `diagrams.go` was being applied to ALL diagrams, including sequence diagrams. The flowchart Sugiyama layout algorithm was destroying the carefully-set positions from the sequence parser.

**Symptoms Observed**:
- Participants (Alice, Bob) scattered randomly instead of horizontal row
- Multiple duplicate-looking boxes (lifelines + anchors at wrong positions)
- Message connectors curved/chaotic instead of straight horizontal

**Fix**: Skip `Layout()` call for sequence diagrams since they already have correct positions from the parser.

```go
// In miro/diagrams.go - line 55-68
if diagram.Type != diagrams.TypeSequence {
    diagrams.Layout(diagram, config)
} else {
    // Apply startX/startY offset if provided
    ...
}
```

### v1.4.1 - Visual Fixes (Released)

| Issue | Before | After |
|-------|--------|-------|
| Lifelines invisible | 4px | 10px |
| Anchors visible as white dots | #FFFFFF | #90CAF9 (matches lifeline) |
| Anchors rejected by API | 6px | 8px (Miro minimum) |

### What Should Render Correctly Now

```
┌─────────┐          ┌─────────┐
│  Alice  │          │   Bob   │
└────┬────┘          └────┬────┘
     │                    │
     █ ←-- lifelines --→  █   (10px wide, #90CAF9)
     │                    │
   ──●────────────────────●──  ← message with arrow (straight!)
     │   "Hello Bob!"     │
```

---

## Files Changed This Session

| File | Changes |
|------|---------|
| `miro/diagrams.go` | **CRITICAL FIX**: Skip Layout() for sequence diagrams, add startX/startY offset support |
| `CHANGELOG.md` | Added v1.4.2 entry |
| `HANDOVER.md` | This file |

---

## OAuth Setup (Working)

Token stored at `~/.miro/tokens.json`. Credentials:
- Client ID: `3458764653228771705`
- Redirect URI: `http://localhost:8089/callback`

To re-authenticate:
```bash
MIRO_CLIENT_ID=3458764653228771705 MIRO_CLIENT_SECRET=xxx ./miro-mcp-server auth login
```

Test board: https://miro.com/app/board/uXjVOXQCe5c=/

---

## What's Next? (Recommendations)

### Priority 1: Visual Polish
The sequence diagram works but could look better:
- Add dashed line style for async messages (`-->>`)
- Consider using text labels instead of connectors for messages
- Add activation boxes (tall thin rectangles on lifelines)

### Priority 2: Additional Diagram Types

| Diagram Type | Complexity | Value |
|--------------|------------|-------|
| Class diagrams | Medium | High |
| State diagrams | Medium | High |
| ER diagrams | High | Medium |

### Priority 3: CI/CD Pipeline
- GitHub Actions for automated testing
- Automated release builds on tag push

---

## Architecture Summary

```
miro-mcp-server/
├── main.go                 # Entry point + --verbose flag
├── miro/
│   ├── client.go           # HTTP client with retry/caching
│   ├── diagrams.go         # Diagram generation + validation
│   ├── diagrams/
│   │   ├── types.go        # Diagram, Node, Edge (+ Y field)
│   │   ├── mermaid.go      # Flowchart parser
│   │   ├── sequence.go     # Sequence diagram parser
│   │   ├── converter.go    # ConvertToMiro + ConvertSequenceToMiro
│   │   └── layout.go       # Sugiyama-style algorithm
│   ├── oauth/              # OAuth 2.1 + PKCE
│   └── webhooks/           # Webhook subscriptions + SSE
└── tools/
    ├── definitions.go      # Tool specs (50 tools)
    └── handlers.go         # Handler registration
```

---

## Quick Reference

```bash
# Build
go build -o miro-mcp-server .

# Test
go test -cover ./...

# Test sequence rendering specifically
go test -v ./miro/diagrams/... -run Sequence

# Run with token
MIRO_ACCESS_TOKEN=xxx ./miro-mcp-server

# Run with OAuth
MIRO_CLIENT_ID=xxx MIRO_CLIENT_SECRET=yyy ./miro-mcp-server
```

---

## Competitive Position

| Server | Tools | Flowchart | Sequence Output | Language |
|--------|-------|-----------|-----------------|----------|
| **This server** | 50 | ✅ | ✅ | Go |
| k-jarzyna/miro-mcp | 87 | ❌ | ❌ | TypeScript |
| Official Miro MCP | ~10 | ✅ | ❌ | TypeScript |

**Unique advantages:**
- **Only MCP with sequence diagram rendering**
- Only Go-based Miro MCP (single binary, fast)
- Rate limiting + caching built-in
- Voice-optimized tool descriptions
- 73.4% test coverage on diagrams package

---

## Session Notes

- Miro API minimum shape size is 8px (discovered during testing)
- Shapes can't be truly "invisible" - use matching colors to blend
- Connector captions work but positioning is automatic
- OAuth tokens expire after ~1 hour, stored in ~/.miro/tokens.json

# Session Handover - Miro MCP Server

> **Date**: 2025-12-21
> **Project**: miro-mcp-server
> **Location**: `/Users/olgasafonova/go/src/miro-mcp-server`
> **Version**: v1.4.2 (released)
> **Latest Session**: CRITICAL Sequence Diagram Layout Fix

---

## ⚠️ FIRST THING TOMORROW

**TEST ALL 50 TOOLS ON A LIVE BOARD!**

We fixed a critical bug but may have created a monster. Need to verify:
1. All diagram tools still work (flowcharts, sequence diagrams)
2. All shape creation tools work
3. All connector tools work
4. Board operations work

Test board: https://miro.com/app/board/uXjVOXQCe5c=/

---

## Current State

**50 MCP tools** for Miro whiteboard control. Phases 1-6 complete.

- v1.4.2 released on GitHub with binaries
- All unit tests passing
- Sequence diagrams NOW rendering correctly (verified on live board)

```bash
# Verify build
cd /Users/olgasafonova/go/src/miro-mcp-server
go build -o miro-mcp-server .
go test ./...
```

---

## What Was Fixed (v1.4.2)

### The Bug
The `Layout()` function in `miro/diagrams.go` was being applied to ALL diagrams, including sequence diagrams. This Sugiyama flowchart algorithm was **destroying** the carefully-set positions from the sequence parser.

### Symptoms (Before Fix)
- Participants (Alice, Bob) scattered randomly instead of horizontal row
- Multiple duplicate-looking boxes everywhere
- Message connectors curved/chaotic instead of straight horizontal
- Complete visual chaos

### The Fix (One Line Change)
```go
// miro/diagrams.go line 55-68
if diagram.Type != diagrams.TypeSequence {
    diagrams.Layout(diagram, config)
} else {
    // For sequence diagrams, apply startX/startY offset if provided
    if config.StartX != 0 || config.StartY != 0 {
        for _, node := range diagram.Nodes {
            node.X += config.StartX
            node.Y += config.StartY
        }
        for _, edge := range diagram.Edges {
            edge.Y += config.StartY
        }
    }
}
```

### Result (After Fix)
- Participants horizontally aligned at top
- Vertical lifelines below each participant
- Straight horizontal message arrows
- Proper sequence diagram layout!

---

## Known Issues

### OAuth Token Validation Failing
The `/v2/users/me` endpoint returns a weird error:
```
"user_id": "Invalid parameter type: long is required"
```

This blocks the MCP server from starting with OAuth. **Workaround**: The token itself works fine for board operations - just the validation endpoint is broken. May need to:
1. Change validation to use `/v2/boards?limit=1` instead
2. Or skip validation entirely
3. Or investigate if Miro API changed

### Token Expires Immediately
OAuth tokens show `expires_at` within seconds of being issued. This seems wrong - investigate if it's a parsing issue or actual Miro behavior.

---

## Files Changed This Session

| File | Changes |
|------|---------|
| `miro/diagrams.go` | **CRITICAL FIX**: Skip Layout() for sequence diagrams, add startX/startY offset support |
| `CHANGELOG.md` | Added v1.4.2 entry |
| `HANDOVER.md` | This file |

---

## Test Checklist for Tomorrow

### Diagram Tools
- [ ] `miro_generate_diagram` with flowchart
- [ ] `miro_generate_diagram` with sequence diagram
- [ ] Verify flowchart layout still works (wasn't broken by fix)

### Shape Creation Tools
- [ ] `miro_create_sticky`
- [ ] `miro_create_shape`
- [ ] `miro_create_text`
- [ ] `miro_create_frame`
- [ ] `miro_create_card`
- [ ] `miro_create_image`
- [ ] `miro_create_document`
- [ ] `miro_create_embed`
- [ ] `miro_create_sticky_grid`

### Connector Tools
- [ ] `miro_create_connector`

### Board Tools
- [ ] `miro_list_boards`
- [ ] `miro_get_board`
- [ ] `miro_find_board`
- [ ] `miro_get_board_summary`
- [ ] `miro_create_board`
- [ ] `miro_copy_board`
- [ ] `miro_delete_board`

### Item Tools
- [ ] `miro_list_items`
- [ ] `miro_list_all_items`
- [ ] `miro_get_item`
- [ ] `miro_search_items`
- [ ] `miro_update_item`
- [ ] `miro_delete_item`
- [ ] `miro_bulk_create_items`

### Tag Tools
- [ ] `miro_create_tag`
- [ ] `miro_list_tags`
- [ ] `miro_attach_tag`
- [ ] `miro_detach_tag`
- [ ] `miro_get_item_tags`

### Group Tools
- [ ] `miro_create_group`
- [ ] `miro_ungroup`

### Member Tools
- [ ] `miro_list_board_members`
- [ ] `miro_share_board`

### Mindmap Tools
- [ ] `miro_create_mindmap_node`

### Export Tools (Enterprise only)
- [ ] `miro_get_board_picture`
- [ ] `miro_create_export_job`
- [ ] `miro_get_export_job_status`
- [ ] `miro_get_export_job_results`

### Audit Tools
- [ ] `miro_get_audit_log`

### Webhook Tools
- [ ] `miro_create_webhook`
- [ ] `miro_list_webhooks`
- [ ] `miro_get_webhook`
- [ ] `miro_delete_webhook`

---

## OAuth Setup

Token stored at `~/.miro/tokens.json`. Credentials:
- Client ID: `3458764653228771705`
- Client Secret: `4NkQBjdTFmzYvRoUFolOZIOi0OyaxbSH`
- Redirect URI: `http://localhost:8089/callback`

To authenticate:
```bash
MIRO_CLIENT_ID=3458764653228771705 MIRO_CLIENT_SECRET=4NkQBjdTFmzYvRoUFolOZIOi0OyaxbSH ./miro-mcp-server auth login
```

---

## Architecture Summary

```
miro-mcp-server/
├── main.go                 # Entry point + --verbose flag
├── miro/
│   ├── client.go           # HTTP client with retry/caching
│   ├── diagrams.go         # Diagram generation + THE FIX IS HERE
│   ├── diagrams/
│   │   ├── types.go        # Diagram, Node, Edge (+ Y field)
│   │   ├── mermaid.go      # Flowchart parser
│   │   ├── sequence.go     # Sequence diagram parser (sets positions)
│   │   ├── converter.go    # ConvertToMiro + ConvertSequenceToMiro
│   │   └── layout.go       # Sugiyama-style algorithm (SKIP for sequence!)
│   ├── oauth/              # OAuth 2.1 + PKCE
│   └── webhooks/           # Webhook subscriptions + SSE
└── tools/
    ├── definitions.go      # Tool specs (50 tools)
    └── handlers.go         # Handler registration
```

---

## Quick Commands

```bash
# Build
go build -o miro-mcp-server .

# Test all
go test ./...

# Test sequence specifically
go test -v ./miro/diagrams/... -run Sequence

# Run with static token (if you have one)
MIRO_ACCESS_TOKEN=xxx ./miro-mcp-server

# Build release binaries
GOOS=darwin GOARCH=arm64 go build -o dist/miro-mcp-server-darwin-arm64 .
GOOS=darwin GOARCH=amd64 go build -o dist/miro-mcp-server-darwin-amd64 .
GOOS=linux GOARCH=amd64 go build -o dist/miro-mcp-server-linux-amd64 .
GOOS=windows GOARCH=amd64 go build -o dist/miro-mcp-server-windows-amd64.exe .
```

---

## Recommended Next Steps

### Priority 1: Test Everything
Run through the checklist above on a live board to make sure nothing is broken.

### Priority 2: Fix Token Validation
The `/users/me` endpoint issue needs investigation. Options:
- Use different endpoint for validation
- Make validation optional with a flag
- Debug the actual API response

### Priority 3: Visual Polish (Future)
- Dashed lines for async messages (`-->>`)
- Activation boxes on lifelines
- Better message label positioning

### Priority 4: More Diagram Types (Future)
- Class diagrams
- State diagrams
- ER diagrams

### Priority 5: CI/CD Pipeline (Future)
- GitHub Actions for automated testing
- Automated release builds on tag push

---

## Session Notes

- Miro API `/v2/users/me` returning weird error - may have changed
- OAuth tokens expire very quickly (seconds?) - needs investigation
- The sequence diagram fix was a one-liner but had massive impact
- Test board has various test items from debugging - can be cleaned up
- "We might have created a monster" - thorough testing needed!

---

## Release History

| Version | Date | Changes |
|---------|------|---------|
| v1.4.2 | 2025-12-21 | **CRITICAL**: Fixed sequence diagram layout bug |
| v1.4.1 | 2025-12-21 | Visual fixes (lifeline width, anchor colors) |
| v1.4.0 | 2025-12-21 | Sequence diagram rendering |
| v1.3.0 | 2025-12-21 | Verbose logging, benchmarks, error messages |

---

**Good luck tomorrow! Test everything!** 🧪

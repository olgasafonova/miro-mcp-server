# miro-mcp-server Improvements Roadmap

**Last Updated:** January 5, 2026
**Source:** Comparison testing with official-miro-mcp
**Miro Board:** https://miro.com/app/board/uXjVGVokv2A=

---

## Executive Summary

After comprehensive comparison testing between `miro-mcp-server` and Miro's `official-miro-mcp`, we identified several features worth implementing. The official MCP trades tool breadth for AI-powered intelligence; we can adopt their best ideas while maintaining our comprehensive API coverage.

### Key Metrics from Testing

| Metric | miro-mcp-server | official-miro-mcp |
|--------|-----------------|-------------------|
| Total Tools | 60+ | 5 |
| Token Usage | ~460/session | ~2,450/session |
| Response Style | Minimal JSON | Rich + AI analysis |
| Diagram Output | Discrete shapes | Compound items |

---

## Roadmap Items

### Q1 2026 - Quick Wins

#### 1. Deep Links in Responses
**Priority:** P1
**Effort:** Low
**Status:** ✅ DONE (January 2026)

Add `item_url` field to all create operation responses for direct navigation.

**Current:**
```json
{
  "id": "3458764653995431573",
  "message": "Created sticky note"
}
```

**Proposed:**
```json
{
  "id": "3458764653995431573",
  "message": "Created sticky note",
  "item_url": "https://miro.com/app/board/uXjVGVokv2A=/?focusWidget=3458764653995431573"
}
```

**Implementation:**
- Add `item_url` field to all create tool responses
- Format: `https://miro.com/app/board/{boardId}/?focusWidget={itemId}`
- Apply to: create_sticky, create_shape, create_frame, create_connector, bulk_create, generate_diagram

---

#### 2. Rich Response Mode
**Priority:** P2
**Effort:** Low
**Status:** ✅ DONE (January 2026)

Add optional `detail_level` parameter to list/get operations.

**Current (minimal):**
```json
{
  "id": "x",
  "type": "sticky_note",
  "content": "text",
  "x": 0,
  "y": 0
}
```

**Proposed (with detail_level=full):**
```json
{
  "id": "x",
  "type": "sticky_note",
  "content": "text",
  "x": 0,
  "y": 0,
  "style": {
    "fillColor": "yellow",
    "textAlign": "center",
    "textAlignVertical": "middle"
  },
  "geometry": {
    "width": 199,
    "height": 228
  },
  "parent": null,
  "created_at": "2025-12-30T23:43:50Z",
  "modified_at": "2025-12-30T23:43:50Z"
}
```

**Implementation:**
- Add `detail_level` parameter: `"minimal"` (default) | `"full"`
- Apply to: list_items, get_item, list_all_items, get_frame_items
- Full mode includes: style, geometry, timestamps, parent info

---

### Q2 2026 - Medium Effort

#### 3. Natural Language Diagram Input
**Priority:** P3
**Effort:** Medium
**Status:** ⏭️ SKIPPED

Accept plain English descriptions in addition to Mermaid syntax.

**Why skipped:** Redundant. When used with any AI agent (Claude Code, Cursor, etc.), the agent already converts natural language to Mermaid before calling the tool. Embedding an LLM inside the MCP server would add API costs, latency, and complexity for zero benefit.

**Original proposal:**
```
miro_generate_diagram(
  description="Create a flowchart showing: Start leads to Decision...",
  input_type="natural"
)
```

**Reality:** Users say "create a flowchart with Start, Decision, End" and the AI agent generates Mermaid automatically. No server-side LLM needed.

---

#### 4. Professional Flowchart Stencils
**Priority:** P4
**Effort:** Medium
**Status:** ✅ DONE (January 2026)

Use Miro's official flowchart stencils instead of basic shapes.

**Current output:**
- Basic rectangles and rhombuses
- No color coding
- Generic styling

**Proposed output:**
- `flow_chart_terminator` for Start/End (green)
- `flow_chart_decision` for decisions (yellow)
- `flow_chart_process` for actions (blue)
- Proper border colors and styling

**Implementation:**
- Added `use_stencils` parameter to `GenerateDiagramArgs`
- Mapped Mermaid node types to Miro stencil shapes
- Applied professional color coding with matching borders
- Uses v2-experimental API endpoint for flowchart shapes

---

#### 5. Compound Diagram Items
**Priority:** P5
**Effort:** Medium
**Status:** ✅ DONE (January 2026)

Create compound diagram for easier manipulation instead of discrete shapes.

**Current behavior:**
- 5 nodes = 5 separate shape items
- 5 edges = 5 separate connector items
- Total: 10 items on board (hard to move/delete together)

**Implemented behavior:**
- Added `output_mode` parameter with three modes:
  - `"discrete"` (default): Individual shapes and connectors
  - `"grouped"`: All items grouped together via Miro Groups API
  - `"framed"`: All items contained in a frame
- Cleaner board organization
- Easier to move/delete as a unit

**Implementation:**
- Researched Miro API; no native "diagram widget" exists
- Solution: Use Groups API to combine all diagram items
- Alternative: Frame mode wraps diagram in a container
- New response fields: `output_mode`, `diagram_id`, `diagram_url`, `diagram_type`, `total_items`
- Graceful degradation: falls back to discrete if grouping fails

---

### Q3 2026 - High Effort

#### 6. AI Board Analysis
**Priority:** P6
**Effort:** High
**Status:** ⏭️ SKIPPED

Generate intelligent summaries from board content.

**Why skipped:** Redundant. The AI agent (Claude Code, Cursor, etc.) already has full access to board content via `miro_list_all_items` and `miro_get_board_summary`. The agent can analyze, summarize, and generate insights directly without embedding a separate LLM in the server.

**Proof:** Tested on board `uXjVGVokv2A=` - Claude Code produced complete board analysis using only existing tools.

**What works today:**
```
User: "Analyze this Miro board"
AI Agent: calls miro_list_all_items → interprets content → returns analysis
```

No `miro_analyze_board` tool needed. No Claude API integration in server.

---

#### 7. Document Generation
**Priority:** P7
**Effort:** High
**Status:** ⏭️ SKIPPED

Generate formal documents from board content.

**Why skipped:** Same reason as P6. The AI agent can generate any document format (tech specs, requirements, etc.) from board content fetched via existing tools. No server-side LLM needed.

**What works today:**
```
User: "Generate a technical specification from this Miro board"
AI Agent: calls miro_list_all_items → generates tech spec document
```

The agent already knows document formats and can output markdown, HTML, or any structure requested.

---

### Q3+ 2026 - Very High Effort

#### 8. Auto-Expand Diagrams
**Priority:** P8
**Effort:** Very High
**Status:** ⏭️ SKIPPED

AI automatically expands diagrams with contextually relevant branches.

**Why skipped:** Same pattern. When user says "create a mindmap about MCP Features", the AI agent already expands the concept, generates detailed Mermaid code, and calls `miro_generate_diagram`. No server-side LLM needed.

**What works today:**
```
User: "Create a mindmap about MCP Features with Read, Write, AI branches - expand each"
AI Agent: generates expanded Mermaid → calls miro_generate_diagram
```

---

## Implementation Status

```
✅ DONE
├── P1: Deep Links in Responses
├── P2: Rich Response Mode
├── P4: Professional Flowchart Stencils
└── P5: Compound Diagram Items

⏭️ SKIPPED (redundant - AI agent handles these natively)
├── P3: Natural Language Diagrams
├── P6: AI Board Analysis
├── P7: Document Generation
└── P8: Auto-Expand Diagrams
```

**Roadmap complete.** All valuable features implemented. Remaining items were found to be redundant since AI agents (Claude Code, Cursor, etc.) already provide these capabilities when given access to board data via existing tools.

---

## Dependencies

| Feature | External Dependency |
|---------|---------------------|
| Natural Language Diagrams | ⏭️ Skipped (redundant with AI agents) |
| AI Board Analysis | ⏭️ Skipped (AI agent does this natively) |
| Document Generation | ⏭️ Skipped (AI agent does this natively) |
| Auto-Expand Diagrams | ⏭️ Skipped (AI agent does this natively) |
| Professional Stencils | Miro Stencil API access (v2-experimental) |
| Compound Diagrams | ✅ None (uses existing Groups/Frames API) |

---

## Success Metrics

| Feature | Success Criteria |
|---------|------------------|
| Deep Links | 100% of create operations return item_url |
| Rich Response | <10% token increase when using full mode |
| Natural Language | 90%+ diagram accuracy from plain English |
| Stencils | Visual parity with official-miro-mcp diagrams |
| AI Analysis | Useful summaries for 80%+ of boards |

---

## References

- [Comparison Report](./docs/miro-mcp-comparison-report.md)
- [Official Miro MCP](https://github.com/miroapp/mcp)
- [Miro REST API](https://developers.miro.com/reference)
- [Test Board](https://miro.com/app/board/uXjVGVokv2A=)

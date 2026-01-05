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
**Status:** Planned

Accept plain English descriptions in addition to Mermaid syntax.

**Current (Mermaid only):**
```
miro_generate_diagram(
  diagram="flowchart TD\n  A[Start] --> B{Decision}\n  B -->|Yes| C[End]"
)
```

**Proposed (natural language):**
```
miro_generate_diagram(
  description="Create a flowchart showing: Start leads to a Decision diamond, if Yes go to End, if No loop back to Start",
  input_type="natural"  // or "mermaid" (default)
)
```

**Implementation:**
- Add `input_type` parameter: `"mermaid"` (default) | `"natural"`
- When natural, use LLM to convert to Mermaid before processing
- Requires: Claude API integration or local LLM

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
**Status:** Planned

Generate intelligent summaries from board content.

**New tool: `miro_analyze_board`**

```
miro_analyze_board(
  board_id="xxx",
  analysis_type="project_summary"  // or "technical_spec", "requirements"
)
```

**Output example:**
```markdown
# Project Summary

## Overview
- **Project Type**: Sprint planning board
- **Primary Purpose**: Q1 feature development
- **Key Features**: Authentication, Analytics, Notifications
- **Design Maturity**: Planning stage

## Recommendations
- Prioritize HIGH PRIORITY items first
- Create technical specs for API work
...
```

**Implementation:**
- Integrate Claude API
- Fetch all board items
- Build prompt with board context
- Generate structured analysis
- Support multiple analysis types:
  - `project_summary`
  - `technical_specification`
  - `functional_requirements`
  - `non_functional_requirements`

---

#### 7. Document Generation
**Priority:** P7
**Effort:** High
**Status:** Planned

Generate formal documents from board content.

**New tool: `miro_generate_docs`**

```
miro_generate_docs(
  board_id="xxx",
  doc_types=["technical_spec", "functional_requirements"],
  format="markdown"  // or "html"
)
```

**Output:** Full technical specifications with:
- System architecture
- Service definitions
- Data models
- API contracts
- Deployment notes

**Implementation:**
- Build on AI analysis tool
- Add document templates
- Support multiple output formats
- Include Miro board reference links

---

### Q3+ 2026 - Very High Effort

#### 8. Auto-Expand Diagrams
**Priority:** P8
**Effort:** Very High
**Status:** Future

AI automatically expands diagrams with contextually relevant branches.

**Example:**
```
Input: "Create mindmap with central topic 'MCP Features' and branches: Read, Write, AI"

Output: Auto-expanded to include:
- Read Operations
  - Data Retrieval
  - Search Functionality
  - Content Filtering
- Write Operations
  - Data Entry
  - Content Modification
- AI Analysis
  - Pattern Recognition
  - Sentiment Analysis
```

**Implementation:**
- Deep LLM integration
- Context-aware expansion
- Domain knowledge for relevant sub-topics
- User confirmation before expansion

---

## Implementation Order

```
Q1 2026
├── Week 1-2: Deep Links in Responses
└── Week 3-4: Rich Response Mode

Q2 2026
├── Week 1-3: Natural Language Diagrams
├── Week 4-6: Professional Stencils
└── Week 7-9: Compound Diagram Items

Q3 2026
├── Week 1-4: AI Board Analysis
└── Week 5-8: Document Generation

Q3+ 2026
└── Auto-Expand Diagrams (scope TBD)
```

---

## Dependencies

| Feature | External Dependency |
|---------|---------------------|
| Natural Language Diagrams | Claude API or local LLM |
| AI Board Analysis | Claude API |
| Document Generation | Claude API |
| Auto-Expand Diagrams | Claude API |
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

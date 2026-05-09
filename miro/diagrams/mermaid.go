package diagrams

import (
	"fmt"
	"regexp"
	"strings"
)

// MermaidParser parses Mermaid diagram syntax.
type MermaidParser struct {
	directionPattern *regexp.Regexp
	nodePattern      *regexp.Regexp
	edgePattern      *regexp.Regexp
	subgraphStart    *regexp.Regexp
	subgraphEnd      *regexp.Regexp
}

// NewMermaidParser creates a new Mermaid parser.
func NewMermaidParser() *MermaidParser {
	return &MermaidParser{
		// flowchart TB, graph LR, etc.
		directionPattern: regexp.MustCompile(`(?i)^(flowchart|graph)\s+(TB|TD|BT|LR|RL)\s*$`),

		// Match node definitions with various shapes:
		// A, A[text], A(text), A{text}, A((text)), A>text], A[/text/], A[\text\], A{{text}}
		nodePattern: regexp.MustCompile(`^([A-Za-z0-9_]+)(\[["']?([^"\]]+)["']?\]|\(["']?([^"\)]+)["']?\)|\{["']?([^"\}]+)["']?\}|\(\(["']?([^"\)]+)["']?\)\)|\[\/([^\/]+)\/\]|\[\\([^\\]+)\\\]|\{\{([^\}]+)\}\}|>([^\]]+)\])?$`),

		// Match edges with labels: A --> B, A -- text --> B, A -->|text| B, A -.-> B, A ==> B
		edgePattern: regexp.MustCompile(`([A-Za-z0-9_]+)\s*(-->|---|-\.->|-.->|==>|--)\s*(?:\|([^|]*)\|\s*)?([A-Za-z0-9_]+)(?:\s*(?:--|-->)\s*["']?([^"'\n]+)["']?)?`),

		subgraphStart: regexp.MustCompile(`(?i)^subgraph\s+(\S+)(?:\s*\[([^\]]+)\])?$`),
		subgraphEnd:   regexp.MustCompile(`(?i)^end$`),
	}
}

// directionMap resolves the textual direction marker (TB/TD/BT/LR/RL) to its enum.
var directionMap = map[string]Direction{
	"TB": TopToBottom,
	"TD": TopToBottom,
	"BT": BottomToTop,
	"LR": LeftToRight,
	"RL": RightToLeft,
}

// parserState holds the mutable subgraph-nesting state threaded through line handlers.
type parserState struct {
	currentSubgraph string
	subgraphStack   []string
}

// pushSubgraph remembers the current subgraph and descends into a new one.
func (s *parserState) pushSubgraph(id string) {
	s.subgraphStack = append(s.subgraphStack, s.currentSubgraph)
	s.currentSubgraph = id
}

// popSubgraph ascends to the parent subgraph (or empty when the stack is exhausted).
func (s *parserState) popSubgraph() {
	if len(s.subgraphStack) == 0 {
		s.currentSubgraph = ""
		return
	}
	s.currentSubgraph = s.subgraphStack[len(s.subgraphStack)-1]
	s.subgraphStack = s.subgraphStack[:len(s.subgraphStack)-1]
}

// Parse parses Mermaid syntax and returns a Diagram.
// Performs input validation including size limits to prevent ReDoS attacks.
func (p *MermaidParser) Parse(input string) (*Diagram, error) {
	if err := ValidateDiagramInput(input); err != nil {
		return nil, err
	}

	diagram := NewDiagram(TypeFlowchart)
	state := &parserState{}

	for _, line := range strings.Split(input, "\n") {
		p.handleLine(diagram, state, strings.TrimSpace(line))
	}

	if len(diagram.Nodes) == 0 {
		return nil, fmt.Errorf("no nodes found in diagram")
	}
	return diagram, nil
}

// handleLine dispatches a single trimmed line through the recognizer chain.
// Each helper returns true once it has consumed the line; the chain stops at
// the first match. Unknown lines are silently skipped.
func (p *MermaidParser) handleLine(diagram *Diagram, state *parserState, line string) {
	if isEmptyOrComment(line) {
		return
	}
	if p.handleDirection(diagram, line) {
		return
	}
	if p.handleSubgraphStart(diagram, state, line) {
		return
	}
	if p.handleSubgraphEnd(state, line) {
		return
	}
	if p.parseEdgeLine(diagram, line, state.currentSubgraph) {
		return
	}
	if p.parseNodeLine(diagram, line, state.currentSubgraph) {
		return
	}
	// Reserved-but-unparsed keywords (`flowchart`, `graph` without a direction)
	// are silently skipped, matching the prior behavior.
	_ = isReservedKeyword(line)
}

// isEmptyOrComment reports whether a line should be skipped without further parsing.
func isEmptyOrComment(line string) bool {
	return line == "" || strings.HasPrefix(line, "%%")
}

// isReservedKeyword reports whether the line begins with a Mermaid keyword we
// recognize but choose not to parse further (e.g. a bare `flowchart` declaration).
func isReservedKeyword(line string) bool {
	lower := strings.ToLower(line)
	return strings.HasPrefix(lower, "flowchart") || strings.HasPrefix(lower, "graph")
}

// handleDirection sets the diagram direction when the line is a `flowchart <DIR>`
// or `graph <DIR>` declaration. Returns true if the line was a direction line.
func (p *MermaidParser) handleDirection(diagram *Diagram, line string) bool {
	matches := p.directionPattern.FindStringSubmatch(line)
	if matches == nil {
		return false
	}
	if dir, ok := directionMap[strings.ToUpper(matches[2])]; ok {
		diagram.Direction = dir
	}
	return true
}

// handleSubgraphStart records a new subgraph on the diagram and descends into it.
// Returns true if the line was a `subgraph` opener.
func (p *MermaidParser) handleSubgraphStart(diagram *Diagram, state *parserState, line string) bool {
	matches := p.subgraphStart.FindStringSubmatch(line)
	if matches == nil {
		return false
	}
	sgID := matches[1]
	sgLabel := sgID
	if len(matches) > 2 && matches[2] != "" {
		sgLabel = matches[2]
	}
	diagram.AddSubGraph(&SubGraph{
		ID:       sgID,
		Label:    sgLabel,
		NodeIDs:  []string{},
		ParentID: state.currentSubgraph,
	})
	state.pushSubgraph(sgID)
	return true
}

// handleSubgraphEnd ascends out of the current subgraph when the line is `end`.
// Returns true if the line was a subgraph terminator.
func (p *MermaidParser) handleSubgraphEnd(state *parserState, line string) bool {
	if !p.subgraphEnd.MatchString(line) {
		return false
	}
	state.popSubgraph()
	return true
}

// parseEdgeLine parses a line that contains an edge definition.
// Handles chained edges (A --> B --> C) by splitting on edge operators and
// connecting consecutive parts.
func (p *MermaidParser) parseEdgeLine(diagram *Diagram, line string, subgraph string) bool {
	parts := p.splitEdges(line)
	if len(parts) < 2 {
		return false
	}

	prevNodeID := ""
	for i := range parts {
		nodeID := p.consumeEdgePart(diagram, edgePartCtx{
			parts:      parts,
			i:          i,
			subgraph:   subgraph,
			prevNodeID: prevNodeID,
		})
		if nodeID == "" {
			continue
		}
		prevNodeID = nodeID
	}
	return prevNodeID != ""
}

// edgePartCtx captures the state needed to consume one segment of a chained edge.
type edgePartCtx struct {
	parts      []string
	i          int
	subgraph   string
	prevNodeID string
}

// consumeEdgePart processes a single segment of a chained-edge line: extracts
// the edge label, parses the node, registers it if new, and connects it to the
// previous node when this is not the first segment. Returns the node ID for
// the next iteration's prev-pointer (or "" when the segment was empty).
func (p *MermaidParser) consumeEdgePart(diagram *Diagram, ctx edgePartCtx) string {
	part := strings.TrimSpace(ctx.parts[ctx.i])
	if part == "" {
		return ""
	}

	edgeLabel, part := extractEdgeLabel(part)

	nodeID, nodeLabel, nodeShape := p.extractNode(part)
	if nodeID == "" {
		return ""
	}

	addNodeIfMissing(diagram, nodeRecord{
		id:       nodeID,
		label:    nodeLabel,
		shape:    nodeShape,
		subgraph: ctx.subgraph,
	})

	if ctx.prevNodeID != "" && ctx.i > 0 {
		style, startCap, endCap := p.detectEdgeStyle(ctx.parts, ctx.i-1, ctx.i)
		diagram.AddEdge(&Edge{
			ID:       fmt.Sprintf("edge_%s_%s", ctx.prevNodeID, nodeID),
			FromID:   ctx.prevNodeID,
			ToID:     nodeID,
			Label:    edgeLabel,
			Style:    style,
			StartCap: startCap,
			EndCap:   endCap,
		})
	}
	return nodeID
}

// edgeLabelPattern matches inline edge labels of the form |label|.
var edgeLabelPattern = regexp.MustCompile(`\|([^|]*)\|`)

// extractEdgeLabel pulls a `|label|` segment out of an edge part if present and
// returns (label, partWithoutLabel). When no label is present, returns ("", part).
func extractEdgeLabel(part string) (string, string) {
	if !strings.Contains(part, "|") {
		return "", part
	}
	matches := edgeLabelPattern.FindStringSubmatch(part)
	if matches == nil {
		return "", part
	}
	return matches[1], edgeLabelPattern.ReplaceAllString(part, "")
}

// nodeRecord bundles the fields used to register a new node in a diagram.
type nodeRecord struct {
	id       string
	label    string
	shape    NodeShape
	subgraph string
}

// addNodeIfMissing registers a node in the diagram and (when applicable) in its
// subgraph. No-op when the node is already known.
func addNodeIfMissing(diagram *Diagram, n nodeRecord) {
	if _, exists := diagram.Nodes[n.id]; exists {
		return
	}
	diagram.AddNode(&Node{
		ID:       n.id,
		Label:    n.label,
		Shape:    n.shape,
		SubGraph: n.subgraph,
		Width:    150,
		Height:   60,
	})
	if n.subgraph == "" {
		return
	}
	if sg, ok := diagram.SubGraphs[n.subgraph]; ok {
		sg.NodeIDs = append(sg.NodeIDs, n.id)
	}
}

// splitEdges splits a line by edge operators.
func (p *MermaidParser) splitEdges(line string) []string {
	// Match edge operators: -->, --->, -.->,-.->, ==>, -- text -->, etc.
	edgeOps := regexp.MustCompile(`\s*(-->|--[^>]*-->|-\.->|-.->|==>|---)\s*`)
	parts := edgeOps.Split(line, -1)
	if len(parts) == 1 {
		return parts
	}
	return parts
}

// detectEdgeStyle determines edge style from the original text between nodes.
func (p *MermaidParser) detectEdgeStyle(parts []string, fromIdx, toIdx int) (EdgeStyle, ArrowType, ArrowType) {
	return EdgeSolid, ArrowNone, ArrowNormal
}

// extractNode extracts node ID, label, and shape from a node reference.
func (p *MermaidParser) extractNode(text string) (id, label string, shape NodeShape) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", "", ShapeRectangle
	}

	// Shape brackets:
	// [text] = rectangle, (text) = stadium, {text} = diamond, ((text)) = circle,
	// [/text/] = parallelogram, [\text\] = trapezoid, {{text}} = hexagon,
	// >text] = flag/asymmetric (treated as parallelogram)
	patterns := []struct {
		regex *regexp.Regexp
		shape NodeShape
	}{
		{regexp.MustCompile(`^([A-Za-z0-9_]+)\(\(["']?(.+?)["']?\)\)$`), ShapeCircle},
		{regexp.MustCompile(`^([A-Za-z0-9_]+)\{\{["']?(.+?)["']?\}\}$`), ShapeHexagon},
		{regexp.MustCompile(`^([A-Za-z0-9_]+)\{["']?(.+?)["']?\}$`), ShapeDiamond},
		{regexp.MustCompile(`^([A-Za-z0-9_]+)\(["']?(.+?)["']?\)$`), ShapeStadium},
		{regexp.MustCompile(`^([A-Za-z0-9_]+)\[\/["']?(.+?)["']?\/\]$`), ShapeParallelogram},
		{regexp.MustCompile(`^([A-Za-z0-9_]+)\[\\["']?(.+?)["']?\\\]$`), ShapeTrapezoid},
		{regexp.MustCompile(`^([A-Za-z0-9_]+)>["']?(.+?)["']?\]$`), ShapeParallelogram},
		{regexp.MustCompile(`^([A-Za-z0-9_]+)\[["']?(.+?)["']?\]$`), ShapeRectangle},
	}

	for _, p := range patterns {
		if matches := p.regex.FindStringSubmatch(text); matches != nil {
			return matches[1], matches[2], p.shape
		}
	}

	if match := regexp.MustCompile(`^([A-Za-z0-9_]+)$`).FindStringSubmatch(text); match != nil {
		return match[1], match[1], ShapeRectangle
	}

	return "", "", ShapeRectangle
}

// parseNodeLine parses a standalone node definition.
func (p *MermaidParser) parseNodeLine(diagram *Diagram, line string, subgraph string) bool {
	nodeID, nodeLabel, nodeShape := p.extractNode(line)
	if nodeID == "" {
		return false
	}
	addNodeIfMissing(diagram, nodeRecord{
		id:       nodeID,
		label:    nodeLabel,
		shape:    nodeShape,
		subgraph: subgraph,
	})
	return true
}

// ParseMermaid is a convenience function to parse Mermaid syntax.
// It auto-detects the diagram type (flowchart or sequence).
// Performs input validation including size limits to prevent ReDoS attacks.
func ParseMermaid(input string) (*Diagram, error) {
	if err := ValidateDiagramInput(input); err != nil {
		return nil, err
	}
	if isSequenceDiagram(input) {
		return NewSequenceParser().Parse(input)
	}
	return NewMermaidParser().Parse(input)
}

// isSequenceDiagram checks if the input is a sequence diagram.
func isSequenceDiagram(input string) bool {
	for _, line := range strings.Split(input, "\n") {
		line = strings.TrimSpace(strings.ToLower(line))
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}
		return line == "sequencediagram"
	}
	return false
}

package diagrams

import (
	"fmt"
	"regexp"
	"strings"
)

// MermaidParser parses Mermaid diagram syntax.
type MermaidParser struct {
	// Patterns for parsing
	directionPattern  *regexp.Regexp
	nodePattern       *regexp.Regexp
	edgePattern       *regexp.Regexp
	subgraphStart     *regexp.Regexp
	subgraphEnd       *regexp.Regexp
}

// NewMermaidParser creates a new Mermaid parser.
func NewMermaidParser() *MermaidParser {
	return &MermaidParser{
		// Match: flowchart TB, graph LR, etc.
		directionPattern: regexp.MustCompile(`(?i)^(flowchart|graph)\s+(TB|TD|BT|LR|RL)\s*$`),

		// Match node definitions with various shapes:
		// A, A[text], A(text), A{text}, A((text)), A>text], A[/text/], A[\text\], A{{text}}
		nodePattern: regexp.MustCompile(`^([A-Za-z0-9_]+)(\[["']?([^"\]]+)["']?\]|\(["']?([^"\)]+)["']?\)|\{["']?([^"\}]+)["']?\}|\(\(["']?([^"\)]+)["']?\)\)|\[\/([^\/]+)\/\]|\[\\([^\\]+)\\\]|\{\{([^\}]+)\}\}|>([^\]]+)\])?$`),

		// Match edges with labels:
		// A --> B, A -- text --> B, A -->|text| B, A -.-> B, A ==> B
		edgePattern: regexp.MustCompile(`([A-Za-z0-9_]+)\s*(-->|---|-\.->|-.->|==>|--)\s*(?:\|([^|]*)\|\s*)?([A-Za-z0-9_]+)(?:\s*(?:--|-->)\s*["']?([^"'\n]+)["']?)?`),

		// Match subgraph start
		subgraphStart: regexp.MustCompile(`(?i)^subgraph\s+(\S+)(?:\s*\[([^\]]+)\])?$`),

		// Match subgraph end
		subgraphEnd: regexp.MustCompile(`(?i)^end$`),
	}
}

// Parse parses Mermaid syntax and returns a Diagram.
func (p *MermaidParser) Parse(input string) (*Diagram, error) {
	diagram := NewDiagram(TypeFlowchart)
	lines := strings.Split(input, "\n")

	var currentSubgraph string
	subgraphStack := []string{}

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}

		// Check for direction declaration
		if matches := p.directionPattern.FindStringSubmatch(line); matches != nil {
			dir := strings.ToUpper(matches[2])
			switch dir {
			case "TB", "TD":
				diagram.Direction = TopToBottom
			case "BT":
				diagram.Direction = BottomToTop
			case "LR":
				diagram.Direction = LeftToRight
			case "RL":
				diagram.Direction = RightToLeft
			}
			continue
		}

		// Check for subgraph start
		if matches := p.subgraphStart.FindStringSubmatch(line); matches != nil {
			sgID := matches[1]
			sgLabel := matches[1]
			if len(matches) > 2 && matches[2] != "" {
				sgLabel = matches[2]
			}

			sg := &SubGraph{
				ID:       sgID,
				Label:    sgLabel,
				NodeIDs:  []string{},
				ParentID: currentSubgraph,
			}
			diagram.AddSubGraph(sg)

			subgraphStack = append(subgraphStack, currentSubgraph)
			currentSubgraph = sgID
			continue
		}

		// Check for subgraph end
		if p.subgraphEnd.MatchString(line) {
			if len(subgraphStack) > 0 {
				currentSubgraph = subgraphStack[len(subgraphStack)-1]
				subgraphStack = subgraphStack[:len(subgraphStack)-1]
			} else {
				currentSubgraph = ""
			}
			continue
		}

		// Try to parse as edge (may contain inline node definitions)
		if p.parseEdgeLine(diagram, line, currentSubgraph) {
			continue
		}

		// Try to parse as standalone node definition
		if p.parseNodeLine(diagram, line, currentSubgraph) {
			continue
		}

		// If we can't parse the line, check if it's a valid keyword we should skip
		lowerLine := strings.ToLower(line)
		if strings.HasPrefix(lowerLine, "flowchart") || strings.HasPrefix(lowerLine, "graph") {
			continue
		}

		// Unknown line - could warn but continue parsing
		_ = lineNum // suppress unused warning
	}

	if len(diagram.Nodes) == 0 {
		return nil, fmt.Errorf("no nodes found in diagram")
	}

	return diagram, nil
}

// parseEdgeLine parses a line that contains an edge definition.
func (p *MermaidParser) parseEdgeLine(diagram *Diagram, line string, subgraph string) bool {
	// Try to find edges in the line
	// Handle chained edges: A --> B --> C
	parts := p.splitEdges(line)
	if len(parts) < 2 {
		return false
	}

	prevNodeID := ""
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Extract edge info if present
		edgeLabel := ""
		var edgeStyle EdgeStyle
		var startCap, endCap ArrowType

		// Check for edge modifiers in part
		if strings.Contains(part, "|") {
			// Extract label from |label|
			labelMatch := regexp.MustCompile(`\|([^|]*)\|`).FindStringSubmatch(part)
			if labelMatch != nil {
				edgeLabel = labelMatch[1]
				part = regexp.MustCompile(`\|[^|]*\|`).ReplaceAllString(part, "")
			}
		}

		// Determine node ID and shape from part
		nodeID, nodeLabel, nodeShape := p.extractNode(part)
		if nodeID == "" {
			continue
		}

		// Create or update node
		if _, exists := diagram.Nodes[nodeID]; !exists {
			node := &Node{
				ID:       nodeID,
				Label:    nodeLabel,
				Shape:    nodeShape,
				SubGraph: subgraph,
				Width:    150,
				Height:   60,
			}
			diagram.AddNode(node)

			// Add to subgraph if in one
			if subgraph != "" {
				if sg, ok := diagram.SubGraphs[subgraph]; ok {
					sg.NodeIDs = append(sg.NodeIDs, nodeID)
				}
			}
		}

		// Create edge from previous node
		if prevNodeID != "" && i > 0 {
			// Check what kind of edge connects them
			edgeStyle, startCap, endCap = p.detectEdgeStyle(parts, i-1, i)

			edge := &Edge{
				ID:       fmt.Sprintf("edge_%s_%s", prevNodeID, nodeID),
				FromID:   prevNodeID,
				ToID:     nodeID,
				Label:    edgeLabel,
				Style:    edgeStyle,
				StartCap: startCap,
				EndCap:   endCap,
			}
			diagram.AddEdge(edge)
		}

		prevNodeID = nodeID
	}

	return prevNodeID != ""
}

// splitEdges splits a line by edge operators.
func (p *MermaidParser) splitEdges(line string) []string {
	// Match edge operators: -->, --->, -.->,-.->, ==>, -- text -->, etc.
	edgeOps := regexp.MustCompile(`\s*(-->|--[^>]*-->|-\.->|-.->|==>|---)\s*`)

	// Split but keep the operators for style detection
	parts := edgeOps.Split(line, -1)

	// If no splits happened, check if it's a node-only line
	if len(parts) == 1 {
		return parts
	}

	return parts
}

// detectEdgeStyle determines edge style from the original text between nodes.
func (p *MermaidParser) detectEdgeStyle(parts []string, fromIdx, toIdx int) (EdgeStyle, ArrowType, ArrowType) {
	style := EdgeSolid
	startCap := ArrowNone
	endCap := ArrowNormal

	// Default arrow style
	return style, startCap, endCap
}

// extractNode extracts node ID, label, and shape from a node reference.
func (p *MermaidParser) extractNode(text string) (id, label string, shape NodeShape) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", "", ShapeRectangle
	}

	// Check for shape brackets
	// [text] = rectangle
	// (text) = rounded rectangle
	// {text} = diamond
	// ((text)) = circle
	// [/text/] = parallelogram
	// [\text\] = trapezoid
	// {{text}} = hexagon
	// >text] = flag/asymmetric

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
		{regexp.MustCompile(`^([A-Za-z0-9_]+)>["']?(.+?)["']?\]$`), ShapeParallelogram}, // flag shape
		{regexp.MustCompile(`^([A-Za-z0-9_]+)\[["']?(.+?)["']?\]$`), ShapeRectangle},
	}

	for _, p := range patterns {
		if matches := p.regex.FindStringSubmatch(text); matches != nil {
			return matches[1], matches[2], p.shape
		}
	}

	// Just a bare ID
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

	// Only add if not already defined
	if _, exists := diagram.Nodes[nodeID]; !exists {
		node := &Node{
			ID:       nodeID,
			Label:    nodeLabel,
			Shape:    nodeShape,
			SubGraph: subgraph,
			Width:    150,
			Height:   60,
		}
		diagram.AddNode(node)

		if subgraph != "" {
			if sg, ok := diagram.SubGraphs[subgraph]; ok {
				sg.NodeIDs = append(sg.NodeIDs, nodeID)
			}
		}
	}

	return true
}

// ParseMermaid is a convenience function to parse Mermaid syntax.
// It auto-detects the diagram type (flowchart or sequence).
func ParseMermaid(input string) (*Diagram, error) {
	// Check for sequence diagram
	if isSequenceDiagram(input) {
		parser := NewSequenceParser()
		return parser.Parse(input)
	}

	// Default to flowchart
	parser := NewMermaidParser()
	return parser.Parse(input)
}

// isSequenceDiagram checks if the input is a sequence diagram.
func isSequenceDiagram(input string) bool {
	lines := strings.Split(input, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(strings.ToLower(line))
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}
		return line == "sequencediagram"
	}
	return false
}

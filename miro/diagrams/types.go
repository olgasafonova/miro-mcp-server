// Package diagrams provides diagram parsing, layout, and Miro conversion.
package diagrams

// DiagramType represents the type of diagram.
type DiagramType string

const (
	TypeFlowchart DiagramType = "flowchart"
	TypeSequence  DiagramType = "sequence"
	TypeMindmap   DiagramType = "mindmap"
)

// Direction represents the layout direction.
type Direction string

const (
	TopToBottom Direction = "TB"
	BottomToTop Direction = "BT"
	LeftToRight Direction = "LR"
	RightToLeft Direction = "RL"
)

// NodeShape represents the shape of a node in the diagram.
type NodeShape string

const (
	ShapeRectangle        NodeShape = "rectangle"
	ShapeRoundedRectangle NodeShape = "rounded_rectangle"
	ShapeDiamond          NodeShape = "rhombus"
	ShapeCircle           NodeShape = "circle"
	ShapeStadium          NodeShape = "pill"
	ShapeCylinder         NodeShape = "can"
	ShapeParallelogram    NodeShape = "parallelogram"
	ShapeHexagon          NodeShape = "hexagon"
	ShapeTrapezoid        NodeShape = "trapezoid"
)

// EdgeStyle represents the style of a connector.
type EdgeStyle string

const (
	EdgeSolid  EdgeStyle = "solid"
	EdgeDotted EdgeStyle = "dotted"
	EdgeThick  EdgeStyle = "thick"
)

// ArrowType represents arrow head types.
type ArrowType string

const (
	ArrowNone   ArrowType = "none"
	ArrowNormal ArrowType = "arrow"
	ArrowCircle ArrowType = "filled_circle"
	ArrowCross  ArrowType = "diamond"
)

// Node represents a node in the diagram.
type Node struct {
	ID       string    // Unique identifier
	Label    string    // Display text
	Shape    NodeShape // Visual shape
	SubGraph string    // Parent subgraph ID (empty if root level)

	// Layout positions (computed)
	X      float64
	Y      float64
	Width  float64
	Height float64

	// Styling
	Color string // Fill color
}

// Edge represents a connection between nodes.
type Edge struct {
	ID              string    // Unique identifier
	FromID          string    // Source node ID
	ToID            string    // Target node ID
	Label           string    // Edge label/caption
	Style           EdgeStyle // Line style
	StartCap        ArrowType // Arrow at start
	EndCap          ArrowType // Arrow at end
	IsBidirectional bool

	// Sequence diagram specific
	Y float64 // Y position for sequence diagram messages (0 for flowcharts)
}

// SubGraph represents a grouping of nodes.
type SubGraph struct {
	ID       string   // Unique identifier
	Label    string   // Display title
	NodeIDs  []string // Nodes in this subgraph
	ParentID string   // Parent subgraph ID (for nesting)

	// Layout positions (computed)
	X      float64
	Y      float64
	Width  float64
	Height float64
}

// Diagram represents a parsed diagram ready for layout.
type Diagram struct {
	Type      DiagramType
	Direction Direction
	Title     string
	Nodes     map[string]*Node
	Edges     []*Edge
	SubGraphs map[string]*SubGraph

	// Computed layout bounds
	Width  float64
	Height float64
}

// NewDiagram creates a new empty diagram.
func NewDiagram(dtype DiagramType) *Diagram {
	return &Diagram{
		Type:      dtype,
		Direction: TopToBottom,
		Nodes:     make(map[string]*Node),
		Edges:     make([]*Edge, 0),
		SubGraphs: make(map[string]*SubGraph),
	}
}

// AddNode adds a node to the diagram.
func (d *Diagram) AddNode(node *Node) {
	d.Nodes[node.ID] = node
}

// AddEdge adds an edge to the diagram.
func (d *Diagram) AddEdge(edge *Edge) {
	d.Edges = append(d.Edges, edge)
}

// AddSubGraph adds a subgraph to the diagram.
func (d *Diagram) AddSubGraph(sg *SubGraph) {
	d.SubGraphs[sg.ID] = sg
}

// GetNodeOrder returns nodes in topological order for layout, using Kahn's
// algorithm. Nodes on cycles (never reaching zero in-degree) are appended at
// the end in map order.
func (d *Diagram) GetNodeOrder() []string {
	incoming, outgoing := d.buildAdjacency()
	order := kahnSort(incoming, outgoing)
	return d.appendCycleNodes(order)
}

// buildAdjacency computes in-degree counts and outgoing adjacency lists for
// edges whose endpoints both exist in the node set.
func (d *Diagram) buildAdjacency() (map[string]int, map[string][]string) {
	incoming := make(map[string]int)
	outgoing := make(map[string][]string)

	for id := range d.Nodes {
		incoming[id] = 0
		outgoing[id] = []string{}
	}

	for _, edge := range d.Edges {
		if _, ok := d.Nodes[edge.FromID]; !ok {
			continue
		}
		if _, ok := d.Nodes[edge.ToID]; !ok {
			continue
		}
		incoming[edge.ToID]++
		outgoing[edge.FromID] = append(outgoing[edge.FromID], edge.ToID)
	}
	return incoming, outgoing
}

// kahnSort runs Kahn's algorithm over the adjacency data, consuming the
// incoming map as it goes.
func kahnSort(incoming map[string]int, outgoing map[string][]string) []string {
	queue := zeroInDegreeNodes(incoming)

	var order []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		order = append(order, node)
		queue = append(queue, releaseNeighbors(incoming, outgoing[node])...)
	}
	return order
}

// zeroInDegreeNodes collects the nodes with no incoming edges.
func zeroInDegreeNodes(incoming map[string]int) []string {
	var queue []string
	for id, count := range incoming {
		if count == 0 {
			queue = append(queue, id)
		}
	}
	return queue
}

// releaseNeighbors decrements each neighbor's in-degree and returns those
// that reach zero.
func releaseNeighbors(incoming map[string]int, neighbors []string) []string {
	var released []string
	for _, neighbor := range neighbors {
		incoming[neighbor]--
		if incoming[neighbor] == 0 {
			released = append(released, neighbor)
		}
	}
	return released
}

// appendCycleNodes appends any node missing from the sorted order (nodes on
// cycles never reach zero in-degree during Kahn's algorithm).
func (d *Diagram) appendCycleNodes(order []string) []string {
	seen := make(map[string]bool, len(order))
	for _, id := range order {
		seen[id] = true
	}
	for id := range d.Nodes {
		if !seen[id] {
			order = append(order, id)
		}
	}
	return order
}

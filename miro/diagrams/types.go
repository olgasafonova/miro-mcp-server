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
	ID        string    // Unique identifier
	FromID    string    // Source node ID
	ToID      string    // Target node ID
	Label     string    // Edge label/caption
	Style     EdgeStyle // Line style
	StartCap  ArrowType // Arrow at start
	EndCap    ArrowType // Arrow at end
	IsBidirectional bool
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

// GetNodeOrder returns nodes in topological order for layout.
func (d *Diagram) GetNodeOrder() []string {
	// Build adjacency list
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

	// Kahn's algorithm for topological sort
	var queue []string
	for id, count := range incoming {
		if count == 0 {
			queue = append(queue, id)
		}
	}

	var order []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		order = append(order, node)

		for _, neighbor := range outgoing[node] {
			incoming[neighbor]--
			if incoming[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Add any remaining nodes (cycles)
	for id := range d.Nodes {
		found := false
		for _, ordered := range order {
			if ordered == id {
				found = true
				break
			}
		}
		if !found {
			order = append(order, id)
		}
	}

	return order
}

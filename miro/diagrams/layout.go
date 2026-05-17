package diagrams

import (
	"sort"
)

// LayoutConfig contains configuration for the layout algorithm.
type LayoutConfig struct {
	NodeWidth    float64 // Default node width
	NodeHeight   float64 // Default node height
	NodeSpacingX float64 // Horizontal spacing between nodes
	NodeSpacingY float64 // Vertical spacing between layers
	StartX       float64 // Starting X position
	StartY       float64 // Starting Y position
	Padding      float64 // Padding around subgraphs
}

// DefaultLayoutConfig returns sensible defaults.
func DefaultLayoutConfig() LayoutConfig {
	return LayoutConfig{
		NodeWidth:    180,
		NodeHeight:   70,
		NodeSpacingX: 80,
		NodeSpacingY: 120,
		StartX:       0,
		StartY:       0,
		Padding:      40,
	}
}

// Layout applies automatic layout to a diagram.
func Layout(diagram *Diagram, config LayoutConfig) {
	if len(diagram.Nodes) == 0 {
		return
	}

	// Assign layers to nodes
	layers := assignLayers(diagram)

	// Order nodes within each layer to minimize edge crossings
	orderLayers(diagram, layers)

	// Calculate positions based on layers and order
	positionNodes(diagram, layers, config)

	// Calculate diagram bounds
	calculateBounds(diagram)
}

// adjacency holds outgoing/incoming neighbor lists for a diagram's nodes.
type adjacency struct {
	outgoing map[string][]string
	incoming map[string][]string
}

// buildAdjacency constructs outgoing/incoming adjacency maps, skipping edges
// that reference unknown nodes.
func buildAdjacency(diagram *Diagram) adjacency {
	adj := adjacency{
		outgoing: make(map[string][]string),
		incoming: make(map[string][]string),
	}

	for id := range diagram.Nodes {
		adj.outgoing[id] = []string{}
		adj.incoming[id] = []string{}
	}

	for _, edge := range diagram.Edges {
		if !edgeRefersToKnownNodes(diagram, edge) {
			continue
		}
		adj.outgoing[edge.FromID] = append(adj.outgoing[edge.FromID], edge.ToID)
		adj.incoming[edge.ToID] = append(adj.incoming[edge.ToID], edge.FromID)
	}

	return adj
}

func edgeRefersToKnownNodes(diagram *Diagram, edge *Edge) bool {
	if _, ok := diagram.Nodes[edge.FromID]; !ok {
		return false
	}
	if _, ok := diagram.Nodes[edge.ToID]; !ok {
		return false
	}
	return true
}

// findRoots returns nodes with no incoming edges. If none exist, returns one
// arbitrary node so layer assignment can still proceed.
func findRoots(diagram *Diagram, incoming map[string][]string) []string {
	var roots []string
	for id := range diagram.Nodes {
		if len(incoming[id]) == 0 {
			roots = append(roots, id)
		}
	}
	if len(roots) > 0 {
		return roots
	}
	for id := range diagram.Nodes {
		return []string{id}
	}
	return roots
}

// bfsLayers walks the graph from each root, assigning longest-path layer
// numbers along outgoing edges.
func bfsLayers(roots []string, outgoing map[string][]string) map[string]int {
	nodeLayer := make(map[string]int)
	queue := make([]string, 0, len(roots))
	for _, root := range roots {
		nodeLayer[root] = 0
		queue = append(queue, root)
	}

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		layer := nodeLayer[node]
		for _, neighbor := range outgoing[node] {
			newLayer := layer + 1
			if existing, ok := nodeLayer[neighbor]; !ok || existing < newLayer {
				nodeLayer[neighbor] = newLayer
				queue = append(queue, neighbor)
			}
		}
	}

	// Disconnected components default to layer 0.
	for id := range nodeLayer {
		_ = id
	}
	return nodeLayer
}

// groupByLayer collapses a per-node layer map into a layer-indexed slice of
// node IDs.
func groupByLayer(diagram *Diagram, nodeLayer map[string]int) map[int][]string {
	layers := make(map[int][]string)
	for id := range diagram.Nodes {
		layer, ok := nodeLayer[id]
		if !ok {
			layer = 0
		}
		layers[layer] = append(layers[layer], id)
	}
	return layers
}

// assignLayers assigns each node to a layer using longest path.
func assignLayers(diagram *Diagram) map[int][]string {
	adj := buildAdjacency(diagram)
	roots := findRoots(diagram, adj.incoming)
	nodeLayer := bfsLayers(roots, adj.outgoing)
	return groupByLayer(diagram, nodeLayer)
}

// sortedLayerNums returns layer indices in ascending order.
func sortedLayerNums(layers map[int][]string) []int {
	layerNums := make([]int, 0, len(layers))
	for l := range layers {
		layerNums = append(layerNums, l)
	}
	sort.Ints(layerNums)
	return layerNums
}

// initialBarycenter assigns each node its current position within its layer.
func initialBarycenter(layers map[int][]string, layerNums []int) map[string]float64 {
	nodePos := make(map[string]float64)
	for _, l := range layerNums {
		for i, id := range layers[l] {
			nodePos[id] = float64(i)
		}
	}
	return nodePos
}

// reorderLayer recomputes each node's barycenter as the mean of its neighbors'
// positions, sorts the layer, and snaps positions to integer indices.
func reorderLayer(layer []string, neighbors map[string][]string, nodePos map[string]float64) {
	for _, id := range layer {
		ns := neighbors[id]
		if len(ns) == 0 {
			continue
		}
		sum := 0.0
		for _, n := range ns {
			sum += nodePos[n]
		}
		nodePos[id] = sum / float64(len(ns))
	}

	sort.Slice(layer, func(a, b int) bool {
		return nodePos[layer[a]] < nodePos[layer[b]]
	})

	for j, id := range layer {
		nodePos[id] = float64(j)
	}
}

// orderLayers orders nodes within each layer to minimize crossings.
func orderLayers(diagram *Diagram, layers map[int][]string) {
	adj := buildAdjacency(diagram)
	layerNums := sortedLayerNums(layers)
	nodePos := initialBarycenter(layers, layerNums)

	const barycenterIterations = 4
	for iter := 0; iter < barycenterIterations; iter++ {
		// Forward pass: order by predecessors.
		for i := 1; i < len(layerNums); i++ {
			reorderLayer(layers[layerNums[i]], adj.incoming, nodePos)
		}
		// Backward pass: order by successors.
		for i := len(layerNums) - 2; i >= 0; i-- {
			reorderLayer(layers[layerNums[i]], adj.outgoing, nodePos)
		}
	}
}

// maxLayerSize returns the largest layer size, used to center smaller layers.
func maxLayerSize(layers map[int][]string) int {
	maxN := 0
	for _, nodes := range layers {
		if len(nodes) > maxN {
			maxN = len(nodes)
		}
	}
	return maxN
}

// positionParams bundles values reused across per-node positioning.
type positionParams struct {
	config        LayoutConfig
	isHorizontal  bool
	isReversed    bool
	layerCount    int
	maxLayerWidth int
}

func makePositionParams(diagram *Diagram, layers map[int][]string, config LayoutConfig) positionParams {
	return positionParams{
		config:        config,
		isHorizontal:  diagram.Direction == LeftToRight || diagram.Direction == RightToLeft,
		isReversed:    diagram.Direction == BottomToTop || diagram.Direction == RightToLeft,
		layerCount:    len(layers),
		maxLayerWidth: maxLayerSize(layers),
	}
}

// nodePlacement captures where a node should land within its layer.
type nodePlacement struct {
	layerIndex int
	nodeIndex  int
	layerSize  int
}

// placeNode writes X/Y/Width/Height onto a single node.
func placeNode(node *Node, np nodePlacement, p positionParams) {
	node.Width = p.config.NodeWidth
	node.Height = p.config.NodeHeight

	layerWidth := float64(np.layerSize) * (p.config.NodeWidth + p.config.NodeSpacingX)
	maxWidth := float64(p.maxLayerWidth) * (p.config.NodeWidth + p.config.NodeSpacingX)
	offset := (maxWidth - layerWidth) / 2

	li := float64(np.layerIndex)
	ni := float64(np.nodeIndex)

	if p.isHorizontal {
		node.X = p.config.StartX + li*(p.config.NodeWidth+p.config.NodeSpacingY)
		node.Y = p.config.StartY + offset + ni*(p.config.NodeHeight+p.config.NodeSpacingX)
		return
	}
	node.X = p.config.StartX + offset + ni*(p.config.NodeWidth+p.config.NodeSpacingX)
	node.Y = p.config.StartY + li*(p.config.NodeHeight+p.config.NodeSpacingY)
}

// effectiveLayerIndex flips the layer order when the diagram direction is
// reversed (BottomToTop / RightToLeft).
func effectiveLayerIndex(rawLayer int, p positionParams) int {
	if p.isReversed {
		return p.layerCount - 1 - rawLayer
	}
	return rawLayer
}

// positionLayer places every node in a single layer.
func positionLayer(diagram *Diagram, layerKey int, nodes []string, p positionParams) {
	layerIndex := effectiveLayerIndex(layerKey, p)
	for i, nodeID := range nodes {
		node := diagram.Nodes[nodeID]
		if node == nil {
			continue
		}
		placeNode(node, nodePlacement{
			layerIndex: layerIndex,
			nodeIndex:  i,
			layerSize:  len(nodes),
		}, p)
	}
}

// positionNodes calculates actual x,y positions for nodes.
func positionNodes(diagram *Diagram, layers map[int][]string, config LayoutConfig) {
	layerNums := sortedLayerNums(layers)
	p := makePositionParams(diagram, layers, config)
	for _, l := range layerNums {
		positionLayer(diagram, l, layers[l], p)
	}
}

// calculateBounds calculates the overall diagram bounds.
func calculateBounds(diagram *Diagram) {
	if len(diagram.Nodes) == 0 {
		return
	}

	minX, minY := float64(1e9), float64(1e9)
	maxX, maxY := float64(-1e9), float64(-1e9)

	for _, node := range diagram.Nodes {
		if node.X < minX {
			minX = node.X
		}
		if node.Y < minY {
			minY = node.Y
		}
		if node.X+node.Width > maxX {
			maxX = node.X + node.Width
		}
		if node.Y+node.Height > maxY {
			maxY = node.Y + node.Height
		}
	}

	diagram.Width = maxX - minX
	diagram.Height = maxY - minY
}

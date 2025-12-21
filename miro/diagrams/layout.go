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

// assignLayers assigns each node to a layer using longest path.
func assignLayers(diagram *Diagram) map[int][]string {
	// Build adjacency list
	outgoing := make(map[string][]string)
	incoming := make(map[string][]string)

	for id := range diagram.Nodes {
		outgoing[id] = []string{}
		incoming[id] = []string{}
	}

	for _, edge := range diagram.Edges {
		if _, ok := diagram.Nodes[edge.FromID]; !ok {
			continue
		}
		if _, ok := diagram.Nodes[edge.ToID]; !ok {
			continue
		}
		outgoing[edge.FromID] = append(outgoing[edge.FromID], edge.ToID)
		incoming[edge.ToID] = append(incoming[edge.ToID], edge.FromID)
	}

	// Find root nodes (no incoming edges)
	var roots []string
	for id := range diagram.Nodes {
		if len(incoming[id]) == 0 {
			roots = append(roots, id)
		}
	}

	// If no roots, pick first node
	if len(roots) == 0 {
		for id := range diagram.Nodes {
			roots = append(roots, id)
			break
		}
	}

	// Assign layers using BFS from roots
	nodeLayer := make(map[string]int)
	queue := make([]string, 0)
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
			if existingLayer, ok := nodeLayer[neighbor]; !ok || existingLayer < newLayer {
				nodeLayer[neighbor] = newLayer
				queue = append(queue, neighbor)
			}
		}
	}

	// Handle nodes not reached (disconnected components)
	for id := range diagram.Nodes {
		if _, ok := nodeLayer[id]; !ok {
			nodeLayer[id] = 0
		}
	}

	// Group nodes by layer
	layers := make(map[int][]string)
	for id, layer := range nodeLayer {
		layers[layer] = append(layers[layer], id)
	}

	return layers
}

// orderLayers orders nodes within each layer to minimize crossings.
func orderLayers(diagram *Diagram, layers map[int][]string) {
	// Build adjacency for ordering
	outgoing := make(map[string][]string)
	incoming := make(map[string][]string)

	for id := range diagram.Nodes {
		outgoing[id] = []string{}
		incoming[id] = []string{}
	}

	for _, edge := range diagram.Edges {
		if _, ok := diagram.Nodes[edge.FromID]; !ok {
			continue
		}
		if _, ok := diagram.Nodes[edge.ToID]; !ok {
			continue
		}
		outgoing[edge.FromID] = append(outgoing[edge.FromID], edge.ToID)
		incoming[edge.ToID] = append(incoming[edge.ToID], edge.FromID)
	}

	// Get sorted layer indices
	layerNums := make([]int, 0, len(layers))
	for l := range layers {
		layerNums = append(layerNums, l)
	}
	sort.Ints(layerNums)

	// Simple barycenter ordering
	nodePos := make(map[string]float64)

	// Initial positions
	for _, l := range layerNums {
		for i, id := range layers[l] {
			nodePos[id] = float64(i)
		}
	}

	// Iterate to improve ordering
	for iter := 0; iter < 4; iter++ {
		// Forward pass
		for i := 1; i < len(layerNums); i++ {
			l := layerNums[i]
			for _, id := range layers[l] {
				preds := incoming[id]
				if len(preds) > 0 {
					sum := 0.0
					for _, pred := range preds {
						sum += nodePos[pred]
					}
					nodePos[id] = sum / float64(len(preds))
				}
			}

			// Sort layer by barycenter
			sort.Slice(layers[l], func(a, b int) bool {
				return nodePos[layers[l][a]] < nodePos[layers[l][b]]
			})

			// Update positions
			for j, id := range layers[l] {
				nodePos[id] = float64(j)
			}
		}

		// Backward pass
		for i := len(layerNums) - 2; i >= 0; i-- {
			l := layerNums[i]
			for _, id := range layers[l] {
				succs := outgoing[id]
				if len(succs) > 0 {
					sum := 0.0
					for _, succ := range succs {
						sum += nodePos[succ]
					}
					nodePos[id] = sum / float64(len(succs))
				}
			}

			// Sort layer by barycenter
			sort.Slice(layers[l], func(a, b int) bool {
				return nodePos[layers[l][a]] < nodePos[layers[l][b]]
			})

			// Update positions
			for j, id := range layers[l] {
				nodePos[id] = float64(j)
			}
		}
	}
}

// positionNodes calculates actual x,y positions for nodes.
func positionNodes(diagram *Diagram, layers map[int][]string, config LayoutConfig) {
	// Get sorted layer indices
	layerNums := make([]int, 0, len(layers))
	for l := range layers {
		layerNums = append(layerNums, l)
	}
	sort.Ints(layerNums)

	// Calculate max width of any layer
	maxLayerWidth := 0
	for _, nodes := range layers {
		if len(nodes) > maxLayerWidth {
			maxLayerWidth = len(nodes)
		}
	}

	// Position based on direction
	isHorizontal := diagram.Direction == LeftToRight || diagram.Direction == RightToLeft
	isReversed := diagram.Direction == BottomToTop || diagram.Direction == RightToLeft

	for _, l := range layerNums {
		nodes := layers[l]
		layerIndex := l
		if isReversed {
			layerIndex = len(layerNums) - 1 - l
		}

		for i, nodeID := range nodes {
			node := diagram.Nodes[nodeID]
			if node == nil {
				continue
			}

			// Set node dimensions
			node.Width = config.NodeWidth
			node.Height = config.NodeHeight

			// Calculate position
			nodeIndex := float64(i)

			// Center the layer
			layerWidth := float64(len(nodes)) * (config.NodeWidth + config.NodeSpacingX)
			offset := (float64(maxLayerWidth)*(config.NodeWidth+config.NodeSpacingX) - layerWidth) / 2

			if isHorizontal {
				node.X = config.StartX + float64(layerIndex)*(config.NodeWidth+config.NodeSpacingY)
				node.Y = config.StartY + offset + nodeIndex*(config.NodeHeight+config.NodeSpacingX)
			} else {
				node.X = config.StartX + offset + nodeIndex*(config.NodeWidth+config.NodeSpacingX)
				node.Y = config.StartY + float64(layerIndex)*(config.NodeHeight+config.NodeSpacingY)
			}
		}
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

package diagrams

import (
	"testing"
)

func TestLayout_SimpleChain(t *testing.T) {
	diagram := NewDiagram(TypeFlowchart)
	diagram.Direction = TopToBottom

	diagram.AddNode(&Node{ID: "A", Label: "Start"})
	diagram.AddNode(&Node{ID: "B", Label: "Process"})
	diagram.AddNode(&Node{ID: "C", Label: "End"})

	diagram.AddEdge(&Edge{FromID: "A", ToID: "B"})
	diagram.AddEdge(&Edge{FromID: "B", ToID: "C"})

	config := DefaultLayoutConfig()
	Layout(diagram, config)

	// Check that nodes are positioned
	for id, node := range diagram.Nodes {
		if node.Width == 0 || node.Height == 0 {
			t.Errorf("Node %s has no dimensions", id)
		}
	}

	// Check that A is above B, B is above C (TB direction)
	if diagram.Nodes["A"].Y >= diagram.Nodes["B"].Y {
		t.Error("Node A should be above B in TB layout")
	}
	if diagram.Nodes["B"].Y >= diagram.Nodes["C"].Y {
		t.Error("Node B should be above C in TB layout")
	}
}

func TestLayout_LeftToRight(t *testing.T) {
	diagram := NewDiagram(TypeFlowchart)
	diagram.Direction = LeftToRight

	diagram.AddNode(&Node{ID: "A", Label: "Start"})
	diagram.AddNode(&Node{ID: "B", Label: "End"})

	diagram.AddEdge(&Edge{FromID: "A", ToID: "B"})

	config := DefaultLayoutConfig()
	Layout(diagram, config)

	// Check that A is left of B (LR direction)
	if diagram.Nodes["A"].X >= diagram.Nodes["B"].X {
		t.Error("Node A should be left of B in LR layout")
	}
}

func TestLayout_ParallelBranches(t *testing.T) {
	diagram := NewDiagram(TypeFlowchart)
	diagram.Direction = TopToBottom

	// A branches to B and C, which converge at D
	diagram.AddNode(&Node{ID: "A", Label: "Start"})
	diagram.AddNode(&Node{ID: "B", Label: "Branch1"})
	diagram.AddNode(&Node{ID: "C", Label: "Branch2"})
	diagram.AddNode(&Node{ID: "D", Label: "End"})

	diagram.AddEdge(&Edge{FromID: "A", ToID: "B"})
	diagram.AddEdge(&Edge{FromID: "A", ToID: "C"})
	diagram.AddEdge(&Edge{FromID: "B", ToID: "D"})
	diagram.AddEdge(&Edge{FromID: "C", ToID: "D"})

	config := DefaultLayoutConfig()
	Layout(diagram, config)

	// B and C should be on the same layer (same Y)
	if diagram.Nodes["B"].Y != diagram.Nodes["C"].Y {
		t.Errorf("Parallel branches B and C should be on same Y level, got B=%v, C=%v",
			diagram.Nodes["B"].Y, diagram.Nodes["C"].Y)
	}

	// B and C should be on different X positions
	if diagram.Nodes["B"].X == diagram.Nodes["C"].X {
		t.Error("Parallel branches B and C should have different X positions")
	}
}

func TestLayout_DiagramBounds(t *testing.T) {
	diagram := NewDiagram(TypeFlowchart)

	diagram.AddNode(&Node{ID: "A", Label: "A"})
	diagram.AddNode(&Node{ID: "B", Label: "B"})
	diagram.AddEdge(&Edge{FromID: "A", ToID: "B"})

	config := DefaultLayoutConfig()
	Layout(diagram, config)

	if diagram.Width <= 0 {
		t.Error("Diagram width should be > 0")
	}
	if diagram.Height <= 0 {
		t.Error("Diagram height should be > 0")
	}
}

func TestLayout_CustomConfig(t *testing.T) {
	diagram := NewDiagram(TypeFlowchart)
	diagram.AddNode(&Node{ID: "A", Label: "A"})

	config := LayoutConfig{
		NodeWidth:    300,
		NodeHeight:   100,
		NodeSpacingX: 50,
		NodeSpacingY: 80,
		StartX:       100,
		StartY:       200,
	}
	Layout(diagram, config)

	node := diagram.Nodes["A"]
	if node.Width != 300 {
		t.Errorf("Expected node width 300, got %v", node.Width)
	}
	if node.Height != 100 {
		t.Errorf("Expected node height 100, got %v", node.Height)
	}
}

func TestGetNodeOrder_Topological(t *testing.T) {
	diagram := NewDiagram(TypeFlowchart)

	diagram.AddNode(&Node{ID: "A", Label: "A"})
	diagram.AddNode(&Node{ID: "B", Label: "B"})
	diagram.AddNode(&Node{ID: "C", Label: "C"})

	diagram.AddEdge(&Edge{FromID: "A", ToID: "B"})
	diagram.AddEdge(&Edge{FromID: "B", ToID: "C"})

	order := diagram.GetNodeOrder()

	// A should come before B, B before C
	aIdx, bIdx, cIdx := -1, -1, -1
	for i, id := range order {
		switch id {
		case "A":
			aIdx = i
		case "B":
			bIdx = i
		case "C":
			cIdx = i
		}
	}

	if aIdx >= bIdx || bIdx >= cIdx {
		t.Errorf("Expected topological order A < B < C, got order: %v", order)
	}
}

func TestGetNodeOrder_DisconnectedNodes(t *testing.T) {
	diagram := NewDiagram(TypeFlowchart)

	diagram.AddNode(&Node{ID: "A", Label: "A"})
	diagram.AddNode(&Node{ID: "B", Label: "B"})
	diagram.AddNode(&Node{ID: "C", Label: "C"})

	// A -> B, but C is disconnected
	diagram.AddEdge(&Edge{FromID: "A", ToID: "B"})

	order := diagram.GetNodeOrder()

	if len(order) != 3 {
		t.Errorf("Expected 3 nodes in order, got %d", len(order))
	}
}

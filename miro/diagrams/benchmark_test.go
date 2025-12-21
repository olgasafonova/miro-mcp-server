package diagrams

import (
	"fmt"
	"strings"
	"testing"
)

// =============================================================================
// Benchmark: Flowchart Parsing
// =============================================================================

// BenchmarkParseMermaid_SmallFlowchart benchmarks parsing a 5-node flowchart.
func BenchmarkParseMermaid_SmallFlowchart(b *testing.B) {
	input := `flowchart TB
    A[Start] --> B[Process 1]
    B --> C{Decision}
    C -->|Yes| D[Action]
    C -->|No| E[End]`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseMermaid(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseMermaid_MediumFlowchart benchmarks parsing a 20-node flowchart.
func BenchmarkParseMermaid_MediumFlowchart(b *testing.B) {
	input := generateFlowchart(20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseMermaid(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseMermaid_LargeFlowchart benchmarks parsing a 100-node flowchart.
func BenchmarkParseMermaid_LargeFlowchart(b *testing.B) {
	input := generateFlowchart(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseMermaid(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseMermaid_HugeFlowchart benchmarks parsing a 500-node flowchart.
func BenchmarkParseMermaid_HugeFlowchart(b *testing.B) {
	input := generateFlowchart(500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseMermaid(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// =============================================================================
// Benchmark: Sequence Diagram Parsing
// =============================================================================

// BenchmarkParseSequence_Small benchmarks parsing a 5-message sequence diagram.
func BenchmarkParseSequence_Small(b *testing.B) {
	input := `sequenceDiagram
    participant A as Alice
    participant B as Bob
    A->>B: Hello
    B-->>A: Hi
    A->>B: How are you?
    B-->>A: Fine thanks
    A->>B: Bye`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseMermaid(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseSequence_Medium benchmarks parsing a 50-message sequence diagram.
func BenchmarkParseSequence_Medium(b *testing.B) {
	input := generateSequenceDiagram(5, 50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseMermaid(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseSequence_Large benchmarks parsing a 200-message sequence diagram.
func BenchmarkParseSequence_Large(b *testing.B) {
	input := generateSequenceDiagram(10, 200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseMermaid(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// =============================================================================
// Benchmark: Layout Algorithm
// =============================================================================

// BenchmarkLayout_SmallGraph benchmarks layout for a 10-node graph.
func BenchmarkLayout_SmallGraph(b *testing.B) {
	diagram := generateTestDiagram(10, 15)
	config := DefaultLayoutConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Layout(diagram, config)
	}
}

// BenchmarkLayout_MediumGraph benchmarks layout for a 50-node graph.
func BenchmarkLayout_MediumGraph(b *testing.B) {
	diagram := generateTestDiagram(50, 80)
	config := DefaultLayoutConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Layout(diagram, config)
	}
}

// BenchmarkLayout_LargeGraph benchmarks layout for a 200-node graph.
func BenchmarkLayout_LargeGraph(b *testing.B) {
	diagram := generateTestDiagram(200, 350)
	config := DefaultLayoutConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Layout(diagram, config)
	}
}

// BenchmarkLayout_VeryLargeGraph benchmarks layout for a 500-node graph.
func BenchmarkLayout_VeryLargeGraph(b *testing.B) {
	diagram := generateTestDiagram(500, 800)
	config := DefaultLayoutConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Layout(diagram, config)
	}
}

// =============================================================================
// Benchmark: Memory Allocation
// =============================================================================

// BenchmarkParseMermaid_Allocs measures allocations for flowchart parsing.
func BenchmarkParseMermaid_Allocs(b *testing.B) {
	input := generateFlowchart(50)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseMermaid(input)
	}
}

// BenchmarkLayout_Allocs measures allocations for layout algorithm.
func BenchmarkLayout_Allocs(b *testing.B) {
	diagram := generateTestDiagram(100, 150)
	config := DefaultLayoutConfig()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Layout(diagram, config)
	}
}

// =============================================================================
// Benchmark: End-to-End (Parse + Layout)
// =============================================================================

// BenchmarkEndToEnd_SmallFlowchart benchmarks full diagram processing.
func BenchmarkEndToEnd_SmallFlowchart(b *testing.B) {
	input := generateFlowchart(10)
	config := DefaultLayoutConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		diagram, err := ParseMermaid(input)
		if err != nil {
			b.Fatal(err)
		}
		Layout(diagram, config)
	}
}

// BenchmarkEndToEnd_LargeFlowchart benchmarks full diagram processing for large graphs.
func BenchmarkEndToEnd_LargeFlowchart(b *testing.B) {
	input := generateFlowchart(100)
	config := DefaultLayoutConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		diagram, err := ParseMermaid(input)
		if err != nil {
			b.Fatal(err)
		}
		Layout(diagram, config)
	}
}

// =============================================================================
// Test Helpers
// =============================================================================

// generateFlowchart creates a flowchart with n nodes in a linear chain.
func generateFlowchart(n int) string {
	var sb strings.Builder
	sb.WriteString("flowchart TB\n")

	// Create a mix of shapes for realistic parsing
	shapes := []string{"[%s]", "{%s}", "((%s))", "(%s)", "{{%s}}"}

	for i := 0; i < n; i++ {
		shape := shapes[i%len(shapes)]
		label := fmt.Sprintf("Node %d", i)
		nodeText := fmt.Sprintf(shape, label)

		if i == 0 {
			sb.WriteString(fmt.Sprintf("    N%d%s\n", i, nodeText))
		} else {
			sb.WriteString(fmt.Sprintf("    N%d --> N%d%s\n", i-1, i, nodeText))
		}
	}

	// Add some branching for more realistic graph
	if n > 5 {
		sb.WriteString(fmt.Sprintf("    N2 --> N%d\n", n-1))
	}
	if n > 10 {
		sb.WriteString(fmt.Sprintf("    N5 --> N%d\n", n-2))
	}

	return sb.String()
}

// generateSequenceDiagram creates a sequence diagram with p participants and m messages.
func generateSequenceDiagram(participants, messages int) string {
	var sb strings.Builder
	sb.WriteString("sequenceDiagram\n")

	// Declare participants
	for i := 0; i < participants; i++ {
		sb.WriteString(fmt.Sprintf("    participant P%d as Participant %d\n", i, i))
	}

	// Generate messages in a round-robin pattern
	messageTypes := []string{"->>", "-->>", "-)", "-x"}
	for i := 0; i < messages; i++ {
		from := i % participants
		to := (i + 1) % participants
		msgType := messageTypes[i%len(messageTypes)]
		sb.WriteString(fmt.Sprintf("    P%d%sP%d: Message %d\n", from, msgType, to, i))
	}

	return sb.String()
}

// generateTestDiagram creates a Diagram with n nodes and e edges.
func generateTestDiagram(nodes, edges int) *Diagram {
	diagram := NewDiagram(TypeFlowchart)

	// Create nodes
	for i := 0; i < nodes; i++ {
		diagram.AddNode(&Node{
			ID:     fmt.Sprintf("N%d", i),
			Label:  fmt.Sprintf("Node %d", i),
			Shape:  ShapeRectangle,
			Width:  150,
			Height: 60,
		})
	}

	// Create edges (connecting nodes in a semi-random pattern)
	edgeCount := 0
	for i := 0; i < nodes && edgeCount < edges; i++ {
		// Each node connects to 1-3 subsequent nodes
		for j := 1; j <= 3 && i+j < nodes && edgeCount < edges; j++ {
			diagram.AddEdge(&Edge{
				ID:     fmt.Sprintf("E%d", edgeCount),
				FromID: fmt.Sprintf("N%d", i),
				ToID:   fmt.Sprintf("N%d", i+j),
				Style:  EdgeSolid,
			})
			edgeCount++
		}
	}

	return diagram
}

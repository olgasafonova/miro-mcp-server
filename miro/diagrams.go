package miro

import (
	"context"
	"fmt"
	"strings"

	"github.com/olgasafonova/miro-mcp-server/miro/diagrams"
)

// =============================================================================
// Diagram Generation
// =============================================================================

// GenerateDiagram parses diagram code and creates shapes/connectors on a board.
func (c *Client) GenerateDiagram(ctx context.Context, args GenerateDiagramArgs) (GenerateDiagramResult, error) {
	if args.BoardID == "" {
		return GenerateDiagramResult{}, fmt.Errorf("board_id is required")
	}
	if args.Diagram == "" {
		return GenerateDiagramResult{}, fmt.Errorf("diagram code is required")
	}

	// Clean up the diagram input
	diagramCode := strings.TrimSpace(args.Diagram)

	// Parse the Mermaid diagram
	parser := diagrams.NewMermaidParser()
	diagram, err := parser.Parse(diagramCode)
	if err != nil {
		return GenerateDiagramResult{}, fmt.Errorf("failed to parse diagram: %w", err)
	}

	// Configure layout
	config := diagrams.DefaultLayoutConfig()
	if args.StartX != 0 {
		config.StartX = args.StartX
	}
	if args.StartY != 0 {
		config.StartY = args.StartY
	}
	if args.NodeWidth > 0 {
		config.NodeWidth = args.NodeWidth
	}

	// Apply layout
	diagrams.Layout(diagram, config)

	// Convert to Miro items
	miroOutput := diagrams.ConvertToMiro(diagram)

	// Create all items on the board
	var nodeIDs []string
	var connectorIDs []string
	var frameIDs []string

	// First create frames (so shapes can be inside them)
	for _, frame := range miroOutput.Frames {
		result, err := c.CreateFrame(ctx, CreateFrameArgs{
			BoardID: args.BoardID,
			Title:   frame.Title,
			X:       frame.X,
			Y:       frame.Y,
			Width:   frame.Width,
			Height:  frame.Height,
			Color:   frame.Color,
		})
		if err != nil {
			c.logger.Warn("failed to create frame", "title", frame.Title, "error", err)
			continue
		}
		frameIDs = append(frameIDs, result.ID)
	}

	// Create shapes
	shapeIDMap := make(map[int]string) // Index to created ID
	for i, shape := range miroOutput.Shapes {
		result, err := c.CreateShape(ctx, CreateShapeArgs{
			BoardID:  args.BoardID,
			Shape:    shape.Shape,
			Content:  shape.Content,
			X:        shape.X,
			Y:        shape.Y,
			Width:    shape.Width,
			Height:   shape.Height,
			Color:    shape.Color,
			ParentID: args.ParentID,
		})
		if err != nil {
			c.logger.Warn("failed to create shape", "content", shape.Content, "error", err)
			continue
		}
		shapeIDMap[i] = result.ID
		nodeIDs = append(nodeIDs, result.ID)
	}

	// Create connectors
	for _, conn := range miroOutput.Connectors {
		startID, ok1 := shapeIDMap[conn.StartItemIndex]
		endID, ok2 := shapeIDMap[conn.EndItemIndex]

		if !ok1 || !ok2 {
			continue
		}

		result, err := c.CreateConnector(ctx, CreateConnectorArgs{
			BoardID:     args.BoardID,
			StartItemID: startID,
			EndItemID:   endID,
			Caption:     conn.Caption,
			Style:       conn.Style,
			StartCap:    conn.StartCap,
			EndCap:      conn.EndCap,
		})
		if err != nil {
			c.logger.Warn("failed to create connector", "error", err)
			continue
		}
		connectorIDs = append(connectorIDs, result.ID)
	}

	// Build summary message
	var parts []string
	if len(nodeIDs) > 0 {
		parts = append(parts, fmt.Sprintf("%d nodes", len(nodeIDs)))
	}
	if len(connectorIDs) > 0 {
		parts = append(parts, fmt.Sprintf("%d connectors", len(connectorIDs)))
	}
	if len(frameIDs) > 0 {
		parts = append(parts, fmt.Sprintf("%d frames", len(frameIDs)))
	}

	message := "Created diagram"
	if len(parts) > 0 {
		message = fmt.Sprintf("Created diagram with %s", strings.Join(parts, ", "))
	}

	return GenerateDiagramResult{
		NodesCreated:      len(nodeIDs),
		ConnectorsCreated: len(connectorIDs),
		FramesCreated:     len(frameIDs),
		NodeIDs:           nodeIDs,
		ConnectorIDs:      connectorIDs,
		FrameIDs:          frameIDs,
		DiagramWidth:      diagram.Width,
		DiagramHeight:     diagram.Height,
		Message:           message,
	}, nil
}

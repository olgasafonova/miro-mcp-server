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

	// Validate input before parsing
	if err := diagrams.ValidateDiagramInput(diagramCode); err != nil {
		return GenerateDiagramResult{}, err
	}

	// Parse the Mermaid diagram (auto-detects flowchart vs sequence)
	diagram, err := diagrams.ParseMermaid(diagramCode)
	if err != nil {
		// Wrap with helpful context
		hint := diagrams.DiagramTypeHint(diagramCode)
		if hint != "" {
			return GenerateDiagramResult{}, fmt.Errorf("failed to parse diagram: %w. Hint: %s", err, hint)
		}
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

	// Apply layout only for flowcharts - sequence diagrams already have positions set by parser
	if diagram.Type != diagrams.TypeSequence {
		diagrams.Layout(diagram, config)
	} else {
		// For sequence diagrams, apply startX/startY offset if provided
		if config.StartX != 0 || config.StartY != 0 {
			for _, node := range diagram.Nodes {
				node.X += config.StartX
				node.Y += config.StartY
			}
			for _, edge := range diagram.Edges {
				edge.Y += config.StartY
			}
		}
	}

	// Convert to Miro items (with optional stencil shapes)
	miroOutput := diagrams.ConvertToMiroWithOptions(diagram, args.UseStencils)

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

	// Create shapes (use experimental API for stencil shapes)
	shapeIDMap := make(map[int]string) // Index to created ID
	for i, shape := range miroOutput.Shapes {
		var result CreateShapeResult
		var err error

		if shape.IsStencil {
			// Use experimental API for flowchart stencil shapes
			result, err = c.CreateShapeExperimental(ctx, CreateShapeExperimentalArgs{
				BoardID:     args.BoardID,
				Shape:       shape.Shape,
				Content:     shape.Content,
				X:           shape.X,
				Y:           shape.Y,
				Width:       shape.Width,
				Height:      shape.Height,
				FillColor:   shape.Color,
				BorderColor: shape.BorderColor,
				ParentID:    args.ParentID,
			})
		} else {
			// Use standard API for basic shapes
			result, err = c.CreateShape(ctx, CreateShapeArgs{
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
		}

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
		NodeURLs:          BuildItemURLs(args.BoardID, nodeIDs),
		ConnectorIDs:      connectorIDs,
		ConnectorURLs:     BuildItemURLs(args.BoardID, connectorIDs),
		FrameIDs:          frameIDs,
		FrameURLs:         BuildItemURLs(args.BoardID, frameIDs),
		DiagramWidth:      diagram.Width,
		DiagramHeight:     diagram.Height,
		Message:           message,
	}, nil
}

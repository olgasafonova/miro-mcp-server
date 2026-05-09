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
	if err := ValidateBoardID(args.BoardID); err != nil {
		return GenerateDiagramResult{}, err
	}
	if args.Diagram == "" {
		return GenerateDiagramResult{}, fmt.Errorf("diagram code is required")
	}

	diagram, err := parseAndLayoutDiagram(args)
	if err != nil {
		return GenerateDiagramResult{}, err
	}

	miroOutput := diagrams.ConvertToMiroWithOptions(diagram, args.UseStencils)

	frameIDs := c.createDiagramFrames(ctx, args.BoardID, miroOutput.Frames)
	nodeIDs, shapeIDMap := c.createDiagramShapes(ctx, args, miroOutput.Shapes)
	connectorIDs := c.createDiagramConnectors(ctx, args.BoardID, miroOutput.Connectors, shapeIDMap)

	allItemIDs := make([]string, 0, len(nodeIDs)+len(connectorIDs))
	allItemIDs = append(allItemIDs, nodeIDs...)
	allItemIDs = append(allItemIDs, connectorIDs...)
	totalItems := len(allItemIDs)

	result := GenerateDiagramResult{
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
		TotalItems:        totalItems,
		OutputMode:        normalizeOutputMode(args.OutputMode),
	}

	switch result.OutputMode {
	case "grouped":
		c.finalizeGroupedDiagram(ctx, args.BoardID, allItemIDs, totalItems, &result)
	case "framed":
		c.finalizeFramedDiagram(ctx, args, diagram, totalItems, &result)
	default:
		result.Message = buildDiscreteDiagramMessage(len(nodeIDs), len(connectorIDs), len(frameIDs))
	}

	return result, nil
}

// parseAndLayoutDiagram validates input, parses Mermaid, configures layout,
// and runs the layout pass (or applies the sequence-diagram offset).
func parseAndLayoutDiagram(args GenerateDiagramArgs) (*diagrams.Diagram, error) {
	diagramCode := strings.TrimSpace(args.Diagram)

	if err := diagrams.ValidateDiagramInput(diagramCode); err != nil {
		return nil, err
	}

	diagram, err := diagrams.ParseMermaid(diagramCode)
	if err != nil {
		hint := diagrams.DiagramTypeHint(diagramCode)
		if hint != "" {
			return nil, fmt.Errorf("failed to parse diagram: %w. Hint: %s", err, hint)
		}
		return nil, fmt.Errorf("failed to parse diagram: %w", err)
	}

	config := buildLayoutConfig(args)

	if diagram.Type == diagrams.TypeSequence {
		applySequenceDiagramOffset(diagram, config)
	} else {
		diagrams.Layout(diagram, config)
	}

	return diagram, nil
}

// buildLayoutConfig starts from the package default and applies non-zero
// overrides from args.
func buildLayoutConfig(args GenerateDiagramArgs) diagrams.LayoutConfig {
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
	return config
}

// applySequenceDiagramOffset shifts pre-positioned sequence-diagram nodes
// by the configured StartX/StartY (the parser already runs the layout for
// sequences; only the offset remains).
func applySequenceDiagramOffset(diagram *diagrams.Diagram, config diagrams.LayoutConfig) {
	if config.StartX == 0 && config.StartY == 0 {
		return
	}
	for _, node := range diagram.Nodes {
		node.X += config.StartX
		node.Y += config.StartY
	}
	for _, edge := range diagram.Edges {
		edge.Y += config.StartY
	}
}

// createDiagramFrames creates each frame and returns the IDs of those that
// succeeded. Failures are logged and skipped.
func (c *Client) createDiagramFrames(ctx context.Context, boardID string, frames []diagrams.MiroFrame) []string {
	ids := make([]string, 0, len(frames))
	for _, frame := range frames {
		result, err := c.CreateFrame(ctx, CreateFrameArgs{
			BoardID: boardID,
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
		ids = append(ids, result.ID)
	}
	return ids
}

// createDiagramShapes creates each shape via the standard or experimental API
// (based on IsStencil) and returns the successful IDs plus an index→ID map
// used to wire connectors. Failures are logged and skipped.
func (c *Client) createDiagramShapes(ctx context.Context, args GenerateDiagramArgs, shapes []diagrams.MiroShape) ([]string, map[int]string) {
	nodeIDs := make([]string, 0, len(shapes))
	shapeIDMap := make(map[int]string, len(shapes))
	for i, shape := range shapes {
		result, err := c.createOneDiagramShape(ctx, args, shape)
		if err != nil {
			c.logger.Warn("failed to create shape", "content", shape.Content, "error", err)
			continue
		}
		shapeIDMap[i] = result.ID
		nodeIDs = append(nodeIDs, result.ID)
	}
	return nodeIDs, shapeIDMap
}

// createOneDiagramShape dispatches a single shape to the standard or
// experimental endpoint based on IsStencil.
func (c *Client) createOneDiagramShape(ctx context.Context, args GenerateDiagramArgs, shape diagrams.MiroShape) (CreateShapeResult, error) {
	if shape.IsStencil {
		return c.CreateShapeExperimental(ctx, CreateShapeExperimentalArgs{
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
	}
	return c.CreateShape(ctx, CreateShapeArgs{
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

// createDiagramConnectors creates connectors using the shape index map.
// Connectors whose endpoints didn't make it into shapeIDMap are silently
// skipped (the corresponding shape failed earlier).
func (c *Client) createDiagramConnectors(ctx context.Context, boardID string, connectors []diagrams.MiroConnector, shapeIDMap map[int]string) []string {
	ids := make([]string, 0, len(connectors))
	for _, conn := range connectors {
		startID, ok1 := shapeIDMap[conn.StartItemIndex]
		endID, ok2 := shapeIDMap[conn.EndItemIndex]
		if !ok1 || !ok2 {
			continue
		}
		result, err := c.CreateConnector(ctx, CreateConnectorArgs{
			BoardID:     boardID,
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
		ids = append(ids, result.ID)
	}
	return ids
}

// normalizeOutputMode lowercases args.OutputMode and defaults to "discrete".
func normalizeOutputMode(mode string) string {
	mode = strings.ToLower(mode)
	if mode == "" {
		return "discrete"
	}
	return mode
}

// finalizeGroupedDiagram bundles all created items into a single Miro group
// (when at least two items exist).
func (c *Client) finalizeGroupedDiagram(ctx context.Context, boardID string, allItemIDs []string, totalItems int, result *GenerateDiagramResult) {
	if len(allItemIDs) < 2 {
		result.Message = fmt.Sprintf("Created diagram with %d items (too few items to group)", totalItems)
		return
	}
	groupResult, err := c.CreateGroup(ctx, CreateGroupArgs{
		BoardID: boardID,
		ItemIDs: allItemIDs,
	})
	if err != nil {
		c.logger.Warn("failed to group diagram items", "error", err)
		result.Message = fmt.Sprintf("Created diagram with %d items (grouping failed: %v)", totalItems, err)
		return
	}
	result.DiagramID = groupResult.ID
	result.DiagramURL = groupResult.ItemURL
	result.DiagramType = "group"
	result.Message = fmt.Sprintf("Created grouped diagram with %d items", totalItems)
}

// finalizeFramedDiagram wraps the created items in a containing frame sized
// to the diagram bounds plus padding.
func (c *Client) finalizeFramedDiagram(ctx context.Context, args GenerateDiagramArgs, diagram *diagrams.Diagram, totalItems int, result *GenerateDiagramResult) {
	const padding = 40.0
	frameWidth := diagram.Width + padding*2
	frameHeight := diagram.Height + padding*2
	frameCenterX := args.StartX + diagram.Width/2
	frameCenterY := args.StartY + diagram.Height/2

	frameResult, err := c.CreateFrame(ctx, CreateFrameArgs{
		BoardID: args.BoardID,
		Title:   "Diagram",
		X:       frameCenterX,
		Y:       frameCenterY,
		Width:   frameWidth,
		Height:  frameHeight,
	})
	if err != nil {
		c.logger.Warn("failed to create diagram frame", "error", err)
		result.Message = fmt.Sprintf("Created diagram with %d items (framing failed: %v)", totalItems, err)
		return
	}
	result.DiagramID = frameResult.ID
	result.DiagramURL = frameResult.ItemURL
	result.DiagramType = "frame"
	result.FrameIDs = append(result.FrameIDs, frameResult.ID)
	result.FrameURLs = append(result.FrameURLs, frameResult.ItemURL)
	result.FramesCreated++
	result.Message = fmt.Sprintf("Created framed diagram with %d items", totalItems)
}

// buildDiscreteDiagramMessage assembles the "Created diagram with N nodes,
// M connectors, K frames" line for the default discrete output mode.
func buildDiscreteDiagramMessage(nodes, connectors, frames int) string {
	var parts []string
	if nodes > 0 {
		parts = append(parts, fmt.Sprintf("%d nodes", nodes))
	}
	if connectors > 0 {
		parts = append(parts, fmt.Sprintf("%d connectors", connectors))
	}
	if frames > 0 {
		parts = append(parts, fmt.Sprintf("%d frames", frames))
	}
	if len(parts) == 0 {
		return "Created diagram"
	}
	return fmt.Sprintf("Created diagram with %s", strings.Join(parts, ", "))
}

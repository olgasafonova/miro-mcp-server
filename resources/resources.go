// Package resources provides MCP resource handlers for Miro boards.
// Resources allow LLMs to directly access board content via miro:// URIs.
package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	miroclient "github.com/olgasafonova/miro-mcp-server/miro"
)

// ResourceClient defines the minimal interface needed for resource handlers.
// This allows for easier testing with mock implementations.
type ResourceClient interface {
	GetBoardSummary(ctx context.Context, args miroclient.GetBoardSummaryArgs) (miroclient.GetBoardSummaryResult, error)
	ListAllItems(ctx context.Context, args miroclient.ListAllItemsArgs) (miroclient.ListAllItemsResult, error)
	ListItems(ctx context.Context, args miroclient.ListItemsArgs) (miroclient.ListItemsResult, error)
}

// Registry manages MCP resource registration.
type Registry struct {
	client ResourceClient
}

// NewRegistry creates a new resource registry.
func NewRegistry(client ResourceClient) *Registry {
	return &Registry{client: client}
}

// RegisterAll registers all Miro resources with the MCP server.
func (r *Registry) RegisterAll(server *mcp.Server) {
	// Board resource template - miro://board/{board_id}
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "miro-board",
		URITemplate: "miro://board/{board_id}",
		Title:       "Miro Board",
		Description: "Access a Miro board's content including metadata and items. Returns board summary with item counts and recent items.",
		MIMEType:    "application/json",
	}, r.handleBoardResource)

	// Board items resource template - miro://board/{board_id}/items
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "miro-board-items",
		URITemplate: "miro://board/{board_id}/items",
		Title:       "Miro Board Items",
		Description: "Access all items on a Miro board. Returns a complete list of stickies, shapes, text, frames, and other items.",
		MIMEType:    "application/json",
	}, r.handleBoardItemsResource)

	// Board frames resource template - miro://board/{board_id}/frames
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "miro-board-frames",
		URITemplate: "miro://board/{board_id}/frames",
		Title:       "Miro Board Frames",
		Description: "Access all frames on a Miro board. Returns frame titles, positions, and item counts.",
		MIMEType:    "application/json",
	}, r.handleBoardFramesResource)
}

// handleBoardResource handles requests for miro://board/{board_id}
func (r *Registry) handleBoardResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	boardID, err := extractBoardID(req.Params.URI, "miro://board/")
	if err != nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	// Remove any trailing path segments
	if idx := strings.Index(boardID, "/"); idx != -1 {
		boardID = boardID[:idx]
	}

	// Get board summary (includes metadata and item counts)
	summary, err := r.client.GetBoardSummary(ctx, miroclient.GetBoardSummaryArgs{
		BoardID: boardID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get board: %w", err)
	}

	// Convert to JSON
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal board data: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(data),
			},
		},
	}, nil
}

// handleBoardItemsResource handles requests for miro://board/{board_id}/items
func (r *Registry) handleBoardItemsResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	boardID, err := extractBoardID(req.Params.URI, "miro://board/")
	if err != nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	// Remove /items suffix
	boardID = strings.TrimSuffix(boardID, "/items")

	// Get all items on the board
	items, err := r.client.ListAllItems(ctx, miroclient.ListAllItemsArgs{
		BoardID:  boardID,
		MaxItems: 500, // Reasonable limit
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}

	// Convert to JSON
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal items: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(data),
			},
		},
	}, nil
}

// handleBoardFramesResource handles requests for miro://board/{board_id}/frames
func (r *Registry) handleBoardFramesResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	boardID, err := extractBoardID(req.Params.URI, "miro://board/")
	if err != nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	// Remove /frames suffix
	boardID = strings.TrimSuffix(boardID, "/frames")

	// Get all items and filter for frames
	items, err := r.client.ListItems(ctx, miroclient.ListItemsArgs{
		BoardID: boardID,
		Type:    "frame",
		Limit:   100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list frames: %w", err)
	}

	// Convert to JSON
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal frames: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(data),
			},
		},
	}, nil
}

// extractBoardID extracts the board ID from a resource URI.
func extractBoardID(uri, prefix string) (string, error) {
	if !strings.HasPrefix(uri, prefix) {
		return "", fmt.Errorf("invalid URI format: %s", uri)
	}
	boardID := strings.TrimPrefix(uri, prefix)
	if boardID == "" {
		return "", fmt.Errorf("missing board ID in URI: %s", uri)
	}
	return boardID, nil
}

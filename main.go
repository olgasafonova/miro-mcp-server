// Miro MCP Server - A Model Context Protocol server for Miro whiteboards
// Provides tools for creating and managing Miro board content via voice or text
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/olgasafonova/miro-mcp-server/miro"
	"github.com/olgasafonova/miro-mcp-server/tools"
)

const (
	ServerName    = "miro-mcp-server"
	ServerVersion = "1.0.0"
)

func main() {
	// Parse command-line flags
	httpAddr := flag.String("http", "", "HTTP address to listen on (e.g., :8080). If empty, uses stdio transport.")
	flag.Parse()

	// Configure logging to stderr (stdout is used for MCP protocol in stdio mode)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Load configuration from environment
	config, err := miro.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Validate config
	if err := miro.ValidateConfig(config); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Create Miro client
	client := miro.NewClient(config, logger)

	// Create MCP server with instructions for LLMs
	server := mcp.NewServer(&mcp.Implementation{
		Name:    ServerName,
		Version: ServerVersion,
	}, &mcp.ServerOptions{
		Logger:       logger,
		Instructions: serverInstructions,
	})

	// Register all Miro tools
	registry := tools.NewHandlerRegistry(client, logger)
	registry.RegisterAll(server)

	ctx := context.Background()

	// Choose transport based on flags
	if *httpAddr != "" {
		runHTTPServer(server, logger, *httpAddr)
	} else {
		// stdio transport mode (default)
		logger.Info("Starting Miro MCP Server (stdio mode)",
			"name", ServerName,
			"version", ServerVersion,
		)

		if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
}

// runHTTPServer starts the MCP server with HTTP transport
func runHTTPServer(server *mcp.Server, logger *slog.Logger, addr string) {
	// Create the Streamable HTTP handler
	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, nil)

	// Create mux for routing
	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","server":"%s","version":"%s"}`, ServerName, ServerVersion)
	})

	// MCP endpoint
	mux.Handle("/", mcpHandler)

	// Create HTTP server with timeouts
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	logger.Info("Starting Miro MCP Server (HTTP mode)",
		"name", ServerName,
		"version", ServerVersion,
		"address", addr,
	)

	// Security warning
	if !strings.HasPrefix(addr, "127.0.0.1") && !strings.HasPrefix(addr, "localhost") {
		logger.Warn("Server binding to external interface. Ensure you're behind HTTPS proxy in production.")
	}

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}
}

// serverInstructions provides guidance for LLMs on how to use the Miro tools
const serverInstructions = `# Miro MCP Server - Voice-Friendly Whiteboard Control

## Quick Reference

### Creating Items
- miro_create_sticky: Add sticky notes ("add a yellow sticky saying Action Items")
- miro_create_shape: Add shapes ("create a rectangle for the header")
- miro_create_text: Add text labels ("write a title 'Sprint Goals'")
- miro_create_connector: Connect items ("draw an arrow from box A to box B")
- miro_create_frame: Create containers ("make a section for brainstorming")
- miro_bulk_create: Add multiple items ("add these 5 sticky notes")

### Reading
- miro_list_boards: Find boards ("show my boards", "find Design Sprint board")
- miro_get_board: Get board details
- miro_list_items: List items on a board ("what's on this board", "show all stickies")

### Modifying
- miro_update_item: Change items ("update sticky text", "move this shape")
- miro_delete_item: Remove items ("delete that sticky")

## Workflow

1. ALWAYS start with miro_list_boards to get board IDs
2. Use the board ID in all subsequent operations
3. When creating items, save the returned item IDs for connections

## Voice Interaction Tips

- Give SHORT confirmations: "Added 3 stickies to Design board"
- When listing, be concise: "Found 5 boards: Design, Retro, Planning..."
- For errors, explain simply: "Couldn't find that board. Try 'list my boards' first"

## Colors for Stickies

yellow, green, blue, pink, orange, red, gray, cyan, purple, dark_green, dark_blue, black

## Common Shapes

rectangle, round_rectangle, circle, triangle, rhombus, star, hexagon, pentagon

## Example Voice Commands -> Tool Mapping

| User Says | Use This Tool |
|-----------|---------------|
| "Add a sticky saying Review PRs" | miro_create_sticky |
| "Create 5 stickies for action items" | miro_bulk_create |
| "Draw a box around these ideas" | miro_create_frame |
| "Connect the first box to the second" | miro_create_connector |
| "What boards do I have?" | miro_list_boards |
| "What's on the Design board?" | miro_list_items |
| "Delete that last sticky" | miro_delete_item |

## Authentication

Requires MIRO_ACCESS_TOKEN environment variable.
Get a token at: https://miro.com/app/settings/user-profile/apps
`

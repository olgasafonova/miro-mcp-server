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
	"github.com/olgasafonova/miro-mcp-server/miro/audit"
	"github.com/olgasafonova/miro-mcp-server/miro/oauth"
	"github.com/olgasafonova/miro-mcp-server/miro/webhooks"
	"github.com/olgasafonova/miro-mcp-server/tools"
)

const (
	ServerName    = "miro-mcp-server"
	ServerVersion = "1.6.1"
)

func main() {
	// Check for auth subcommand first
	if len(os.Args) > 1 && os.Args[1] == "auth" {
		runAuthCommand(os.Args[2:])
		return
	}

	// Parse command-line flags
	httpAddr := flag.String("http", "", "HTTP address to listen on (e.g., :8080). If empty, uses stdio transport.")
	verbose := flag.Bool("verbose", false, "Enable verbose debug logging")
	flag.Parse()

	// Configure logging to stderr (stdout is used for MCP protocol in stdio mode)
	logLevel := slog.LevelInfo
	if *verbose {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
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

	// Validate token on startup
	user, err := client.ValidateToken(context.Background())
	if err != nil {
		log.Fatalf("Token validation failed: %v\nGet a valid token at https://miro.com/app/settings/user-profile/apps", err)
	}
	logger.Info("Token validated successfully", "user", user.Name, "email", user.Email)

	// Initialize audit logger
	auditConfig := audit.LoadConfigFromEnv()
	auditLogger, err := audit.NewLogger(auditConfig)
	if err != nil {
		logger.Warn("Failed to initialize audit logger, using in-memory", "error", err)
		auditLogger = audit.NewMemoryLogger(1000, auditConfig)
	}
	defer auditLogger.Close()

	if auditConfig.Enabled {
		logger.Info("Audit logging enabled", "path", auditConfig.Path)
	}

	// Create MCP server with instructions for LLMs
	server := mcp.NewServer(&mcp.Implementation{
		Name:    ServerName,
		Version: ServerVersion,
	}, &mcp.ServerOptions{
		Logger:       logger,
		Instructions: serverInstructions,
	})

	// Register all Miro tools with audit logging
	registry := tools.NewHandlerRegistry(client, logger).
		WithAuditLogger(auditLogger).
		WithUser(user.ID, user.Email)
	registry.RegisterAll(server)

	ctx := context.Background()

	// Choose transport based on flags
	if *httpAddr != "" {
		runHTTPServer(server, logger, *httpAddr, *verbose)
	} else {
		// stdio transport mode (default)
		logger.Info("Starting Miro MCP Server (stdio mode)",
			"name", ServerName,
			"version", ServerVersion,
			"verbose", *verbose,
		)

		if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
}

// runHTTPServer starts the MCP server with HTTP transport
func runHTTPServer(server *mcp.Server, logger *slog.Logger, addr string, verbose bool) {
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

	// Webhook endpoints (if enabled)
	webhookConfig := webhooks.LoadConfigFromEnv()
	if webhookConfig.Enabled {
		webhookHandler := webhooks.NewHandler(webhookConfig, logger)
		sseHandler := webhooks.NewSSEHandler(webhookHandler.EventBus())

		// Miro sends webhook events here
		mux.Handle("/webhooks", webhookHandler)
		mux.Handle("/webhooks/", webhookHandler)

		// SSE endpoint for real-time event streaming to clients
		mux.Handle("/events", sseHandler)

		logger.Info("Webhook endpoints enabled",
			"callback", "/webhooks",
			"sse", "/events",
		)
	}

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
		"verbose", verbose,
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

Or use OAuth: ./miro-mcp-server auth login
`

// runAuthCommand handles auth subcommands (login, status, logout)
func runAuthCommand(args []string) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	oauthConfig := oauth.LoadConfigFromEnv()

	if !oauthConfig.IsConfigured() {
		fmt.Println("OAuth not configured. Set MIRO_CLIENT_ID and MIRO_CLIENT_SECRET.")
		fmt.Println("\nTo get OAuth credentials:")
		fmt.Println("1. Go to https://miro.com/app/settings/user-profile/apps")
		fmt.Println("2. Create a new app or edit an existing one")
		fmt.Println("3. Copy Client ID and Client Secret")
		os.Exit(1)
	}

	authFlow := oauth.NewAuthFlow(oauthConfig, logger)
	ctx := context.Background()

	if len(args) == 0 {
		printAuthUsage()
		return
	}

	switch args[0] {
	case "login":
		fmt.Println("Starting OAuth login flow...")
		tokens, err := authFlow.Login(ctx)
		if err != nil {
			log.Fatalf("Login failed: %v", err)
		}
		fmt.Printf("\nLogged in successfully!\n")
		fmt.Printf("User ID: %s\n", tokens.UserID)
		fmt.Printf("Team ID: %s\n", tokens.TeamID)
		fmt.Printf("Token expires: %s\n", tokens.ExpiresAt.Format(time.RFC3339))
		fmt.Printf("\nTokens saved to: %s\n", oauthConfig.TokenStorePath)

	case "status":
		tokens, err := authFlow.Status(ctx)
		if err != nil {
			fmt.Printf("Not logged in: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Authentication status: Logged in")
		fmt.Printf("User ID: %s\n", tokens.UserID)
		fmt.Printf("Team ID: %s\n", tokens.TeamID)
		if tokens.IsExpired() {
			fmt.Println("Token status: Expired (will refresh on next use)")
		} else {
			fmt.Printf("Token expires: %s\n", tokens.ExpiresAt.Format(time.RFC3339))
		}

	case "logout":
		if err := authFlow.Logout(ctx); err != nil {
			log.Fatalf("Logout failed: %v", err)
		}
		fmt.Println("Logged out successfully.")

	default:
		fmt.Printf("Unknown auth command: %s\n\n", args[0])
		printAuthUsage()
		os.Exit(1)
	}
}

func printAuthUsage() {
	fmt.Println("Usage: miro-mcp-server auth <command>")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  login   Start OAuth login flow (opens browser)")
	fmt.Println("  status  Show current authentication status")
	fmt.Println("  logout  Revoke tokens and log out")
	fmt.Println("")
	fmt.Println("Environment variables:")
	fmt.Println("  MIRO_CLIENT_ID      OAuth client ID (required)")
	fmt.Println("  MIRO_CLIENT_SECRET  OAuth client secret (required)")
	fmt.Println("  MIRO_REDIRECT_URI   Callback URL (default: http://localhost:8089/callback)")
	fmt.Println("  MIRO_TOKEN_PATH     Token storage path (default: ~/.miro/tokens.json)")
}

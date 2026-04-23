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
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/olgasafonova/mcp-otel-go/mcpotel"
	"github.com/olgasafonova/mcp-servercard-go/servercard"
	"github.com/olgasafonova/miro-mcp-server/miro"
	"github.com/olgasafonova/miro-mcp-server/miro/audit"
	"github.com/olgasafonova/miro-mcp-server/miro/desirepath"
	"github.com/olgasafonova/miro-mcp-server/miro/oauth"
	"github.com/olgasafonova/miro-mcp-server/prompts"
	"github.com/olgasafonova/miro-mcp-server/resources"
	"github.com/olgasafonova/miro-mcp-server/tools"
)

const (
	ServerName    = "miro-mcp-server"
	ServerVersion = "1.16.1"
)

func main() {
	// Check for auth subcommand first
	if len(os.Args) > 1 && os.Args[1] == "auth" {
		runAuthCommand(os.Args[2:])
		return
	}

	// Parse command-line flags
	httpAddr := flag.String("http", "", "HTTP address to listen on (e.g., :8080). If empty, uses stdio transport.")
	bearerToken := flag.String("bearer-token", "", "Bearer token for HTTP mode authentication. If empty, HTTP endpoints are unauthenticated.")
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

	// Load configuration from environment.
	// Uses LoadConfigOrUnconfigured so the server starts even without MIRO_ACCESS_TOKEN,
	// allowing MCP registries (Glama, Smithery) to inspect tool definitions.
	// Tool calls will return a clear error if the token is not configured.
	config, err := miro.LoadConfigOrUnconfigured()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	if config.IsConfigured() {
		// Validate config only when token is present
		if err := miro.ValidateConfig(config); err != nil {
			log.Fatalf("Invalid configuration: %v", err)
		}
	} else {
		logger.Warn("MIRO_ACCESS_TOKEN not set. Server will start in inspection mode: tools are listed but calls will fail until configured.")
	}

	// Create Miro client
	client := miro.NewClient(config, logger)

	// Validate token on startup (non-fatal to support containerized deployments)
	var user *miro.UserInfo
	if config.IsConfigured() {
		user, err = client.ValidateToken(context.Background())
		if err != nil {
			logger.Warn("Token validation failed; tools will return errors until a valid token is provided",
				"error", err,
				"help", "https://miro.com/app/settings/user-profile/apps")
		} else {
			logger.Info("Token validated successfully", "user", user.Name, "email", user.Email)
		}
	}

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
		// Suppress pre-initialize notifications/tools/list_changed from go-sdk.
		// Without this, AddTool triggers a notification before the client completes
		// the initialize handshake, causing intermittent connection failures.
		Capabilities: &mcp.ServerCapabilities{Tools: &mcp.ToolCapabilities{}},
	})

	// Add OpenTelemetry instrumentation middleware
	server.AddReceivingMiddleware(mcpotel.Middleware(mcpotel.Config{
		ServiceName:    ServerName,
		ServiceVersion: ServerVersion,
	}))

	// Initialize desire path logger for agent behavior normalization
	dpConfig := desirepath.LoadConfigFromEnv()
	dpLogger := desirepath.NewLogger(dpConfig, logger)

	normalizers := []desirepath.Normalizer{
		&desirepath.WhitespaceNormalizer{},
		desirepath.NewURLToIDNormalizer(desirepath.MiroURLPatterns()),
		&desirepath.CamelToSnakeNormalizer{},
		desirepath.NewStringToNumericNormalizer(nil),
		desirepath.NewBooleanCoercionNormalizer(nil),
	}

	if dpConfig.Enabled {
		logger.Info("Desire path normalization enabled", "max_events", dpConfig.MaxEvents)
	}

	// Build the miro_share_board domain allowlist. Operators set
	// MIRO_SHARE_ALLOWED_DOMAINS to an explicit comma-separated list. When the
	// env var is not set, the allowlist falls back to the authenticated user's
	// own email domain (fail-safe default for single-user deployments). If
	// neither is available, every share invitation is rejected with a clear
	// error. See tools/share_allowlist.go for details.
	fallbackEmail := ""
	if user != nil {
		fallbackEmail = user.Email
	}
	shareAllowlist := tools.LoadShareAllowlistFromEnv(fallbackEmail)
	if shareAllowlist.IsEmpty() {
		logger.Warn("miro_share_board allowlist is empty; all share invitations will be rejected",
			"fix", "set MIRO_SHARE_ALLOWED_DOMAINS to a comma-separated list of permitted domains",
			"source", shareAllowlist.Source())
	} else {
		logger.Info("miro_share_board allowlist configured",
			"domains", shareAllowlist.Domains(),
			"source", shareAllowlist.Source())
	}

	// Register all Miro tools with audit logging and desire path normalization
	registry := tools.NewHandlerRegistry(client, logger).
		WithAuditLogger(auditLogger).
		WithShareAllowlist(shareAllowlist).
		WithDesirePathLogger(dpLogger, normalizers)
	if user != nil {
		registry = registry.WithUser(user.ID, user.Email)
	}
	registry.RegisterAll(server)

	// Register MCP Resources (miro://board/{id} URIs)
	resourceRegistry := resources.NewRegistry(client)
	resourceRegistry.RegisterAll(server)
	logger.Debug("Registered MCP resources", "count", 3)

	// Register MCP Prompts (workflow templates)
	promptRegistry := prompts.NewRegistry()
	promptRegistry.RegisterAll(server)
	logger.Debug("Registered MCP prompts", "count", 5)

	// Register SEP-2127 Server Card as MCP resource
	cardOpts := servercard.Options{
		Name:        "io.github.olgasafonova/miro-mcp-server",
		Version:     ServerVersion,
		Description: "MCP server for Miro whiteboards. 91 tools for boards, items, diagrams, mindmaps, tags, groups, connectors, export, and audit. Voice-friendly.",
		Title:       "Miro MCP Server",
		WebsiteURL:  "https://github.com/olgasafonova/miro-mcp-server",
		Repository: &servercard.Repository{
			URL:    "https://github.com/olgasafonova/miro-mcp-server",
			Source: "github",
		},
		Provider: &servercard.Provider{
			Name: "Olga Safonova",
			URL:  "https://github.com/olgasafonova",
		},
	}
	serverCard, err := servercard.Build(cardOpts)
	if err != nil {
		log.Fatalf("Server card error: %v", err)
	}
	servercard.RegisterResource(server, serverCard)
	logger.Debug("Registered Server Card resource", "uri", servercard.ResourceURI)

	ctx := context.Background()

	// Create health checker for HTTP mode
	healthChecker := miro.NewHealthChecker(client, ServerName, ServerVersion)

	// Create metrics collector
	metricsCollector := miro.NewMetricsCollector()

	// Choose transport based on flags
	if *httpAddr != "" {
		runHTTPServer(server, logger, *httpAddr, *bearerToken, *verbose, healthChecker, metricsCollector, serverCard)
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

// bearerTokenMiddleware returns middleware that validates Bearer token authentication.
func bearerTokenMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" || auth != "Bearer "+token {
				w.Header().Set("WWW-Authenticate", `Bearer`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// runHTTPServer starts the MCP server with HTTP transport and graceful shutdown
func runHTTPServer(server *mcp.Server, logger *slog.Logger, addr, bearerToken string, verbose bool, healthChecker *miro.HealthChecker, metrics *miro.MetricsCollector, card *servercard.ServerCard) {
	// Create the Streamable HTTP handler
	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, nil)

	// Create mux for routing
	mux := http.NewServeMux()

	// Enhanced health endpoint
	// Use ?deep=true to perform API connectivity check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		deep := r.URL.Query().Get("deep") == "true"
		report := healthChecker.Check(r.Context(), deep)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")

		// Set appropriate status code based on health
		switch report.Status {
		case miro.HealthStatusHealthy:
			w.WriteHeader(http.StatusOK)
		case miro.HealthStatusDegraded:
			w.WriteHeader(http.StatusOK) // Still OK but with warnings
		case miro.HealthStatusUnhealthy:
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		jsonBytes, err := report.ToJSON()
		if err != nil {
			logger.Error("Failed to marshal health report", "error", err)
			fmt.Fprintf(w, `{"status":"unhealthy","error":"failed to generate report"}`)
			return
		}
		w.Write(jsonBytes)
	})

	// SEP-2127 Server Card endpoint (unauthenticated, for pre-connect discovery)
	// Add remote transport info now that we know the HTTP address.
	card.Remotes = []servercard.Remote{{
		Type:                      "streamable-http",
		URL:                       "/",
		SupportedProtocolVersions: []string{"2025-06-18"},
	}}
	mux.Handle(servercard.WellKnownPath, servercard.Handler(card))

	// Prometheus metrics endpoint
	var metricsHandler http.Handler = http.HandlerFunc(metrics.PrometheusHandler())

	// MCP endpoint with cache control
	var mcpRootHandler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		mcpHandler.ServeHTTP(w, r)
	})

	// Wrap protected endpoints with bearer token middleware if configured
	if bearerToken != "" {
		authMiddleware := bearerTokenMiddleware(bearerToken)
		metricsHandler = authMiddleware(metricsHandler)
		mcpRootHandler = authMiddleware(mcpRootHandler)
	}

	mux.Handle("/metrics", metricsHandler)
	mux.Handle("/", mcpRootHandler)

	// Create HTTP server with timeouts
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  miro.HTTPReadTimeout,
		WriteTimeout: miro.HTTPWriteTimeout,
		IdleTimeout:  miro.HTTPIdleTimeout,
	}

	logger.Info("Starting Miro MCP Server (HTTP mode)",
		"name", ServerName,
		"version", ServerVersion,
		"address", addr,
		"verbose", verbose,
	)

	// Security warnings
	if bearerToken == "" {
		logger.Warn("HTTP mode without --bearer-token. All tools accessible without authentication.")
	}
	if !strings.HasPrefix(addr, "127.0.0.1") && !strings.HasPrefix(addr, "localhost") {
		logger.Warn("Server binding to external interface. Ensure you're behind HTTPS proxy in production.")
	}

	// Graceful shutdown handling
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigChan
		logger.Info("Received shutdown signal", "signal", sig)

		// Give outstanding requests 10 seconds to complete
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("Graceful shutdown failed", "error", err)
		} else {
			logger.Info("Server shutdown complete")
		}
	}()

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

## MCP Resources

Access board content directly via resource URIs:
- miro://board/{board_id} - Get board summary with metadata and item counts
- miro://board/{board_id}/items - Get all items on a board
- miro://board/{board_id}/frames - Get all frames on a board

## MCP Prompts (Workflow Templates)

Use prompts for common workflows:
- create-sprint-board: Create sprint planning board with standard columns
- create-retrospective: Create retrospective with What Went Well/Could Improve/Action Items
- create-brainstorm: Set up brainstorming session with central topic
- create-story-map: Create user story mapping board
- create-kanban: Create kanban board with customizable columns

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

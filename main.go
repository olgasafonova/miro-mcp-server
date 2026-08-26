// Miro MCP Server - A Model Context Protocol server for Miro whiteboards
// Provides tools for creating and managing Miro board content via voice or text
package main

import (
	"context"
	"crypto/subtle"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/olgasafonova/mcp-cache-go/mcpcache"
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
	ServerVersion = "1.24.0"
)

// SEP-2549 cache hints. Two tiers, split by what actually changes underneath.
const (
	// staticListTTL covers the descriptor lists — tools, prompts, resource
	// templates. These are built from code at startup and cannot change while
	// the process lives, so the only thing that invalidates them is a release.
	// Thirty minutes rather than hours because this is an actively developed
	// surface and a stale client should not be pinned across a deploy.
	staticListTTL = 30 * time.Minute

	// liveReadTTL covers resources/read, which returns live board content.
	//
	// It matches miro.ItemCacheTTL, the SHORTEST of the caches sitting behind
	// those handlers (items 1m, boards 2m), because ttlMs is a promise about
	// how long the answer stays good. Within the window the server would serve
	// the same bytes from its own cache anyway, so the client saves a round
	// trip and gives up nothing; past it the server refetches, and a longer
	// hint would have the client holding content the server had already
	// replaced. Advertise the floor, never the average.
	liveReadTTL = miro.ItemCacheTTL
)

// runtimeFlags bundles parsed CLI flags and the configured logger.
type runtimeFlags struct {
	httpAddr    string
	bearerToken string
	verbose     bool
	logger      *slog.Logger
}

// registryDeps groups the optional registry inputs that the tools registry consumes.
type registryDeps struct {
	auditLogger    audit.Logger
	shareAllowlist *tools.ShareAllowlist
	dpLogger       *desirepath.Logger
	normalizers    []desirepath.Normalizer
	user           *miro.UserInfo
}

// httpServerOpts bundles the dependencies passed to runHTTPServer.
type httpServerOpts struct {
	server        *mcp.Server
	logger        *slog.Logger
	addr          string
	bearerToken   string
	verbose       bool
	healthChecker *miro.HealthChecker
	metrics       *miro.MetricsCollector
	card          *servercard.ServerCard
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "auth" {
		runAuthCommand(os.Args[2:])
		return
	}

	rt := parseFlags()
	config := loadMiroConfig(rt.logger)
	client := miro.NewClient(config, rt.logger)
	user := validateMiroToken(client, config, rt.logger)

	auditLogger := initAuditLogger(rt.logger)
	defer auditLogger.Close()

	server := createMCPServer(rt.logger)
	dpLogger, normalizers := initDesirePath(rt.logger)
	shareAllowlist := loadShareAllowlist(user, rt.logger)

	registerTools(server, client, rt.logger, registryDeps{
		auditLogger:    auditLogger,
		shareAllowlist: shareAllowlist,
		dpLogger:       dpLogger,
		normalizers:    normalizers,
		user:           user,
	})
	registerResourcesAndPrompts(server, client, rt.logger)

	serverCard := buildServerCard()

	if rt.httpAddr != "" {
		runHTTPServer(httpServerOpts{
			server:        server,
			logger:        rt.logger,
			addr:          rt.httpAddr,
			bearerToken:   rt.bearerToken,
			verbose:       rt.verbose,
			healthChecker: miro.NewHealthChecker(client, ServerName, ServerVersion),
			metrics:       miro.NewMetricsCollector(),
			card:          serverCard,
		})
		return
	}

	runStdioServer(server, rt)
}

// parseFlags parses CLI flags and constructs the structured logger.
func parseFlags() runtimeFlags {
	httpAddr := flag.String("http", "", "HTTP address to listen on (e.g., :8080). If empty, uses stdio transport.")
	bearerToken := flag.String("bearer-token", "", "Bearer token for HTTP mode authentication. If empty, HTTP endpoints are unauthenticated.")
	verbose := flag.Bool("verbose", false, "Enable verbose debug logging")
	flag.Parse()

	// stdout is reserved for MCP protocol in stdio mode; logs go to stderr.
	logLevel := slog.LevelInfo
	if *verbose {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))

	return runtimeFlags{
		httpAddr:    *httpAddr,
		bearerToken: *bearerToken,
		verbose:     *verbose,
		logger:      logger,
	}
}

// loadMiroConfig loads the miro config and validates it when a token is present.
// LoadConfigOrUnconfigured lets the server start without a token so MCP registries
// can inspect tool definitions; tool calls return a clear error in that mode.
func loadMiroConfig(logger *slog.Logger) *miro.Config {
	config, err := miro.LoadConfigOrUnconfigured()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}
	if !config.IsConfigured() {
		logger.Warn("MIRO_ACCESS_TOKEN not set. Server will start in inspection mode: tools are listed but calls will fail until configured.")
		return config
	}
	if err := miro.ValidateConfig(config); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}
	return config
}

// validateMiroToken performs a non-fatal token check and returns user info if successful.
// Returns nil when the server starts unconfigured or the token check fails.
func validateMiroToken(client *miro.Client, config *miro.Config, logger *slog.Logger) *miro.UserInfo {
	if !config.IsConfigured() {
		return nil
	}
	user, err := client.ValidateToken(context.Background())
	if err != nil {
		logger.Warn("Token validation failed; tools will return errors until a valid token is provided",
			"error", err,
			"help", "https://miro.com/app/settings/user-profile/apps")
		return nil
	}
	logger.Info("Token validated successfully", "user", user.Name, "email", user.Email)
	return user
}

// initAuditLogger initializes the audit logger, falling back to in-memory on failure.
func initAuditLogger(logger *slog.Logger) audit.Logger {
	auditConfig := audit.LoadConfigFromEnv()
	auditLogger, err := audit.NewLogger(auditConfig)
	if err != nil {
		logger.Warn("Failed to initialize audit logger, using in-memory", "error", err)
		auditLogger = audit.NewMemoryLogger(1000, auditConfig)
	}
	if auditConfig.Enabled {
		logger.Info("Audit logging enabled", "path", auditConfig.Path)
	}
	return auditLogger
}

// createMCPServer constructs the MCP server with OTel and cache-TTL middleware
// attached. Capabilities is set explicitly to suppress pre-initialize
// tools/list_changed notifications from go-sdk.
//
// Attached here rather than in runStdioServer or the HTTP handler because main
// builds one *mcp.Server and hands it to whichever transport is selected.
// Attaching in a transport branch would leave the other one unstamped.
func createMCPServer(logger *slog.Logger) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    ServerName,
		Version: ServerVersion,
	}, &mcp.ServerOptions{
		Logger:       logger,
		Instructions: serverInstructions,
		Capabilities: &mcp.ServerCapabilities{Tools: &mcp.ToolCapabilities{}},
	})

	// Both middlewares go in ONE call, and the order is deliberate.
	//
	// AddReceivingMiddleware applies right-to-left, so this composes as
	// mcpotel(mcpcache(handler)): OTel is outermost. Its span therefore covers
	// the TTL stamp as well as the handler.
	//
	// Two separate calls would NOT do this. Each call wraps whatever handler is
	// already installed (shared.go:113), so adding mcpcache in a second call
	// would compose as mcpcache(mcpotel(handler)) and stamp the TTL after the
	// OTel span had already ended. Harmless today, because mcpotel never reads
	// ttlMs, but it is the wrong nesting to leave in place.
	//
	// mcpcache: SEP-2549 requires ttlMs and cacheScope on every cacheable
	// result, but the SDK's setDefaultCacheableValues() sets cacheScope only and
	// leaves ttlMs at 0, which the spec reads as "immediately stale". There is
	// no ServerOptions knob for it.
	//
	// Every method the SDK marks cacheable is listed here, and every one this
	// server answers. Config.Default is deliberately left unset: a blanket
	// default would also cover resources/read, which returns live board content
	// rather than a descriptor list, and would silently pick up any seventh
	// cacheable method a future SDK adds. An omission here ships ttlMs:0, which
	// is spec-compliant and useless, so the list is the safer shape.
	server.AddReceivingMiddleware(
		mcpotel.Middleware(mcpotel.Config{
			ServiceName:    ServerName,
			ServiceVersion: ServerVersion,
		}),
		mcpcache.Middleware(cacheConfig()),
	)

	return server
}

// cacheConfig returns the SEP-2549 TTL policy. Extracted from createMCPServer
// so a test can assert the resources/read entry, which cannot be exercised
// end-to-end without a live board behind it.
func cacheConfig() mcpcache.Config {
	return mcpcache.Config{
		TTLs: map[string]time.Duration{
			mcpcache.MethodListTools:             staticListTTL,
			mcpcache.MethodDiscover:              staticListTTL,
			mcpcache.MethodListPrompts:           staticListTTL,
			mcpcache.MethodListResources:         staticListTTL,
			mcpcache.MethodListResourceTemplates: staticListTTL,
			mcpcache.MethodReadResource:          liveReadTTL,
		},
	}
}

// initDesirePath initializes the desire-path logger and the standard normalizer chain.
func initDesirePath(logger *slog.Logger) (*desirepath.Logger, []desirepath.Normalizer) {
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

	return dpLogger, normalizers
}

// loadShareAllowlist builds the miro_share_board domain allowlist with status logging.
// Operators set MIRO_SHARE_ALLOWED_DOMAINS to an explicit comma-separated list. When
// unset, the allowlist falls back to the authenticated user's own email domain (a
// fail-safe default for single-user deployments). With neither, every share invitation
// is rejected with a clear error.
func loadShareAllowlist(user *miro.UserInfo, logger *slog.Logger) *tools.ShareAllowlist {
	fallbackEmail := ""
	if user != nil {
		fallbackEmail = user.Email
	}
	shareAllowlist := tools.LoadShareAllowlistFromEnv(fallbackEmail)
	switch {
	case shareAllowlist.EmailSource() != "" && len(shareAllowlist.Emails()) == 0:
		// Operator set MIRO_SHARE_ALLOWED_EMAILS to a value that normalized to
		// nothing (e.g. "," or whitespace). Fail closed loudly rather than letting
		// the weaker domain layer take over silently.
		logger.Warn("miro_share_board MIRO_SHARE_ALLOWED_EMAILS is set but has no valid addresses after normalization; all share invitations will be rejected (fail-closed)",
			"fix", "provide at least one valid email, or unset MIRO_SHARE_ALLOWED_EMAILS to use the domain allowlist",
			"source", shareAllowlist.EmailSource())
	case shareAllowlist.IsEmpty():
		logger.Warn("miro_share_board allowlist is empty; all share invitations will be rejected",
			"fix", "set MIRO_SHARE_ALLOWED_EMAILS (exact-email, tighter) or MIRO_SHARE_ALLOWED_DOMAINS (domain)",
			"source", shareAllowlist.Source())
	case len(shareAllowlist.Emails()) > 0:
		// Exact-email layer is authoritative; the domain layer is not consulted.
		logger.Info("miro_share_board allowlist configured (exact-email, authoritative)",
			"emails", shareAllowlist.Emails(),
			"source", shareAllowlist.EmailSource())
	default:
		logger.Info("miro_share_board allowlist configured (domain)",
			"domains", shareAllowlist.Domains(),
			"source", shareAllowlist.Source())
	}
	return shareAllowlist
}

// registerTools registers all Miro tools using the configured profile.
// Unknown MIRO_TOOLS_PROFILE values fall back to full with a warning so a typo
// never silently strips tools the operator was relying on.
func registerTools(server *mcp.Server, client *miro.Client, logger *slog.Logger, deps registryDeps) {
	profileRaw := os.Getenv("MIRO_TOOLS_PROFILE")
	profile, ok := tools.ParseProfile(profileRaw)
	if !ok {
		logger.Warn("Unknown MIRO_TOOLS_PROFILE; falling back to full",
			"value", profileRaw,
			"valid", []string{string(tools.ProfileFull), string(tools.ProfileEssentials)})
	}

	registry := tools.NewHandlerRegistry(client, logger).
		WithAuditLogger(deps.auditLogger).
		WithShareAllowlist(deps.shareAllowlist).
		WithDesirePathLogger(deps.dpLogger, deps.normalizers)
	if deps.user != nil {
		registry = registry.WithUser(deps.user.ID, deps.user.Email)
	}
	registry.RegisterProfile(server, profile)
}

// registerResourcesAndPrompts registers MCP resources and prompts on the server.
func registerResourcesAndPrompts(server *mcp.Server, client *miro.Client, logger *slog.Logger) {
	resources.NewRegistry(client).RegisterAll(server)
	logger.Debug("Registered MCP resources", "count", 3)

	prompts.NewRegistry().RegisterAll(server)
	logger.Debug("Registered MCP prompts", "count", 5)
}

// buildServerCard constructs the SEP-2127 server card for HTTP discovery.
func buildServerCard() *servercard.ServerCard {
	cardOpts := servercard.Options{
		Name:    "io.github.olgasafonova/miro-mcp-server",
		Version: ServerVersion,
		// Tool count is derived from AllTools, never written down. This string is
		// what a client reads before it connects, and a hardcoded number here goes
		// stale the next time a tool is added, silently.
		Description: fmt.Sprintf("MCP server for Miro whiteboards. %d tools for boards, items, diagrams, mindmaps, tags, groups, connectors, export, and audit, including miro_tool_search for discovery. Voice-friendly.", len(tools.AllTools)),
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
	card, err := servercard.Build(cardOpts)
	if err != nil {
		log.Fatalf("Server card error: %v", err)
	}
	return card
}

// runStdioServer runs the MCP server over stdio transport.
func runStdioServer(server *mcp.Server, rt runtimeFlags) {
	rt.logger.Info("Starting Miro MCP Server (stdio mode)",
		"name", ServerName,
		"version", ServerVersion,
		"verbose", rt.verbose,
	)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// bearerTokenMiddleware returns middleware that validates Bearer token authentication.
func bearerTokenMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Constant-time compare: a plain != short-circuits on the first
			// differing byte, which leaks the token prefix to a remote prober.
			// mediawiki-mcp-server and nordic-registry-mcp-server both already
			// use subtle.ConstantTimeCompare here; miro was the outlier.
			auth := r.Header.Get("Authorization")
			expected := "Bearer " + token
			if auth == "" || subtle.ConstantTimeCompare([]byte(auth), []byte(expected)) != 1 {
				w.Header().Set("WWW-Authenticate", `Bearer`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// runHTTPServer starts the MCP server with HTTP transport and graceful shutdown.
func runHTTPServer(opts httpServerOpts) {
	if err := validateHTTPSecurity(opts.addr, opts.bearerToken); err != nil {
		log.Fatalf("HTTP security validation failed: %v", err)
	}

	mux := buildHTTPMux(opts)

	httpServer := &http.Server{
		Addr:         opts.addr,
		Handler:      mux,
		ReadTimeout:  miro.HTTPReadTimeout,
		WriteTimeout: miro.HTTPWriteTimeout,
		IdleTimeout:  miro.HTTPIdleTimeout,
	}

	opts.logger.Info("Starting Miro MCP Server (HTTP mode)",
		"name", ServerName,
		"version", ServerVersion,
		"address", opts.addr,
		"verbose", opts.verbose,
	)
	logHTTPSecurityWarnings(opts)

	go gracefulShutdown(opts.logger, httpServer)

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}
}

// buildHTTPMux assembles the HTTP routing tree for runHTTPServer, including
// health, server-card discovery, metrics, and the MCP root handler. Bearer-token
// middleware wraps protected endpoints when a token is configured.
func buildHTTPMux(opts httpServerOpts) *http.ServeMux {
	// Stateless is required to serve protocol revision 2026-07-28: without it
	// the transport rejects every request at that version with HTTP 400, with
	// server/discover the only exemption. Safe here because the same
	// *mcp.Server is returned for every request, so there is no per-session
	// state to lose. Session IDs are ignored and DELETE returns 405, per the
	// spec (SEP-2567).
	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return opts.server
	}, &mcp.StreamableHTTPOptions{Stateless: true})

	mux := http.NewServeMux()
	mux.HandleFunc("/health", buildHealthHandler(opts.healthChecker, opts.logger))

	// SEP-2127 server card endpoint (unauthenticated, for pre-connect discovery).
	opts.card.Remotes = []servercard.Remote{{
		Type:                      "streamable-http",
		URL:                       "/",
		SupportedProtocolVersions: []string{"2026-07-28", "2025-11-25", "2025-06-18"},
	}}
	mux.Handle(servercard.WellKnownPath, servercard.Handler(opts.card))

	var metricsHandler http.Handler = http.HandlerFunc(opts.metrics.PrometheusHandler())
	var mcpRootHandler http.Handler = wrapNoStore(mcpHandler)

	if opts.bearerToken != "" {
		auth := bearerTokenMiddleware(opts.bearerToken)
		metricsHandler = auth(metricsHandler)
		mcpRootHandler = auth(mcpRootHandler)
	}

	mux.Handle("/metrics", metricsHandler)
	mux.Handle("/", mcpRootHandler)

	return mux
}

// healthStatusCodes maps a HealthStatus to its HTTP response code.
// HealthStatusDegraded still returns 200 because the service is reachable.
var healthStatusCodes = map[miro.HealthStatus]int{
	miro.HealthStatusHealthy:   http.StatusOK,
	miro.HealthStatusDegraded:  http.StatusOK,
	miro.HealthStatusUnhealthy: http.StatusServiceUnavailable,
}

// buildHealthHandler returns the /health handler. Use ?deep=true to include
// an API connectivity probe in the response.
func buildHealthHandler(checker *miro.HealthChecker, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deep := r.URL.Query().Get("deep") == "true"
		report := checker.Check(r.Context(), deep)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(healthStatusCodes[report.Status])

		jsonBytes, err := report.ToJSON()
		if err != nil {
			logger.Error("Failed to marshal health report", "error", err)
			if _, werr := fmt.Fprintf(w, `{"status":"unhealthy","error":"failed to generate report"}`); werr != nil {
				logger.Error("Failed to write health fallback response", "error", werr)
			}
			return
		}
		if _, werr := w.Write(jsonBytes); werr != nil {
			logger.Error("Failed to write health response", "error", werr)
		}
	}
}

// wrapNoStore wraps a handler so responses carry Cache-Control: no-store.
func wrapNoStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// isLoopbackAddr reports whether addr binds only the local loopback interface.
// A bare port (":8080") or "0.0.0.0" binds all interfaces and is treated as
// externally reachable. A hostname that is not exactly "localhost" and does not
// parse as an IP is treated as non-loopback, which is the safe default.
//
// The host is parsed rather than prefix-matched. The previous implementation
// used strings.HasPrefix(addr, "localhost"), which also matched hostnames that
// merely START with it: "-http localhost.corp.example:8080" was read as a
// loopback bind, so validateHTTPSecurity skipped the mandatory-token check and
// the server could bind an externally-resolvable interface with no auth at all.
// "127.0.0.1.evil.example" defeated the other prefix the same way. This is the
// implementation mediawiki-mcp-server (isLoopbackBind) and
// nordic-registry-mcp-server (isLoopbackAddr) already use; miro was the only
// one of the three still on the prefix check.
func isLoopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		// No port present; treat the whole value as the host.
		host = addr
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

// validateHTTPSecurity refuses to start an externally-reachable HTTP server that
// has no authentication configured. Binding to a non-loopback interface without
// a bearer token would expose every tool unauthenticated, so it is a startup
// error rather than a warning. Loopback binds keep the softer warn-and-start
// behavior for local development (see logHTTPSecurityWarnings).
func validateHTTPSecurity(addr, bearerToken string) error {
	if isLoopbackAddr(addr) || bearerToken != "" {
		return nil
	}
	return fmt.Errorf("refusing to start: HTTP server binds to non-loopback address %q without --bearer-token; "+
		"set --bearer-token to require authentication, or bind to 127.0.0.1/localhost for local-only access", addr)
}

// logHTTPSecurityWarnings emits warnings about exposed-but-unauthenticated
// configurations that are permitted but still worth flagging. The fail-closed
// non-loopback + no-token case is rejected earlier by validateHTTPSecurity, so
// a missing token here only ever means a loopback bind.
func logHTTPSecurityWarnings(opts httpServerOpts) {
	if opts.bearerToken == "" {
		opts.logger.Warn("HTTP mode without --bearer-token on a loopback bind. Tools are accessible without authentication to local processes.")
	}
	if !isLoopbackAddr(opts.addr) {
		opts.logger.Warn("Server binding to external interface. Ensure you're behind an HTTPS proxy in production.")
	}
}

// gracefulShutdown blocks until SIGINT/SIGTERM, then shuts down the HTTP server
// allowing 10 seconds for in-flight requests to complete.
func gracefulShutdown(logger *slog.Logger, httpServer *http.Server) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan
	logger.Info("Received shutdown signal", "signal", sig)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Graceful shutdown failed", "error", err)
		return
	}
	logger.Info("Server shutdown complete")
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

// authCommand is the signature for an auth subcommand handler.
type authCommand func(ctx context.Context, flow *oauth.AuthFlow, cfg *oauth.Config) error

// authCommands dispatches the auth subcommand name to its handler.
var authCommands = map[string]authCommand{
	"login":  runAuthLogin,
	"status": runAuthStatus,
	"logout": runAuthLogout,
}

// runAuthCommand handles auth subcommands (login, status, logout).
func runAuthCommand(args []string) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	oauthConfig := oauth.LoadConfigFromEnv()

	if !oauthConfig.IsConfigured() {
		printOAuthSetupHelp()
		os.Exit(1)
	}

	authFlow := oauth.NewAuthFlow(oauthConfig, logger)
	ctx := context.Background()

	if len(args) == 0 {
		printAuthUsage()
		return
	}

	handler, ok := authCommands[args[0]]
	if !ok {
		fmt.Printf("Unknown auth command: %s\n\n", args[0])
		printAuthUsage()
		os.Exit(1)
	}
	if err := handler(ctx, authFlow, oauthConfig); err != nil {
		log.Fatal(err)
	}
}

// runAuthLogin starts an interactive OAuth login flow.
func runAuthLogin(ctx context.Context, flow *oauth.AuthFlow, cfg *oauth.Config) error {
	fmt.Println("Starting OAuth login flow...")
	tokens, err := flow.Login(ctx)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	fmt.Printf("\nLogged in successfully!\n")
	fmt.Printf("User ID: %s\n", tokens.UserID)
	fmt.Printf("Team ID: %s\n", tokens.TeamID)
	fmt.Printf("Token expires: %s\n", tokens.ExpiresAt.Format(time.RFC3339))
	fmt.Printf("\nTokens saved to: %s\n", cfg.TokenStorePath)
	return nil
}

// runAuthStatus prints the current authentication status.
func runAuthStatus(ctx context.Context, flow *oauth.AuthFlow, _ *oauth.Config) error {
	tokens, err := flow.Status(ctx)
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
	return nil
}

// runAuthLogout revokes saved tokens and reports success.
func runAuthLogout(ctx context.Context, flow *oauth.AuthFlow, _ *oauth.Config) error {
	if err := flow.Logout(ctx); err != nil {
		return fmt.Errorf("logout failed: %w", err)
	}
	fmt.Println("Logged out successfully.")
	return nil
}

func printOAuthSetupHelp() {
	fmt.Println("OAuth not configured. Set MIRO_CLIENT_ID and MIRO_CLIENT_SECRET.")
	fmt.Println("\nTo get OAuth credentials:")
	fmt.Println("1. Go to https://miro.com/app/settings/user-profile/apps")
	fmt.Println("2. Create a new app or edit an existing one")
	fmt.Println("3. Copy Client ID and Client Secret")
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

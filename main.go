// Miro MCP Server - A Model Context Protocol server for Miro whiteboards
// Provides tools for creating and managing Miro board content via voice or text
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/olgasafonova/mcp-cache-go/mcpcache"
	"github.com/olgasafonova/mcp-otel-go/mcpotel"
	"github.com/olgasafonova/mcp-servercard-go/servercard"
	"github.com/olgasafonova/miro-mcp-server/miro"
	"github.com/olgasafonova/miro-mcp-server/miro/audit"
	"github.com/olgasafonova/miro-mcp-server/miro/desirepath"
	"github.com/olgasafonova/miro-mcp-server/prompts"
	"github.com/olgasafonova/miro-mcp-server/resources"
	"github.com/olgasafonova/miro-mcp-server/tools"
)

const (
	ServerName    = "miro-mcp-server"
	ServerVersion = "1.25.0"
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

package main

import (
	"context"
	"crypto/subtle"
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
	"github.com/olgasafonova/mcp-servercard-go/servercard"
	"github.com/olgasafonova/miro-mcp-server/miro"
)

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

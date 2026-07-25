package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/olgasafonova/miro-mcp-server/miro"
)

// =============================================================================
// Wiring tests for the main package
//
// main(), the auth subcommands, the serve loops, gracefulShutdown, and
// parseFlags are deliberately absent: they block on a listener, launch a
// browser, call os.Exit, or read global flag state.
// =============================================================================

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

// captureStdout runs fn with os.Stdout redirected and returns what it printed.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var sb strings.Builder
		io.Copy(&sb, r)
		done <- sb.String()
	}()

	fn()

	w.Close()
	os.Stdout = orig
	return <-done
}

// testHTTPOpts builds a minimal-but-real httpServerOpts for mux tests.
func testHTTPOpts(t *testing.T, bearerToken string) httpServerOpts {
	t.Helper()

	logger := quietLogger()
	client := miro.NewClient(&miro.Config{AccessToken: "test-token"}, logger)

	return httpServerOpts{
		server:        createMCPServer(logger),
		logger:        logger,
		addr:          "127.0.0.1:0",
		bearerToken:   bearerToken,
		healthChecker: miro.NewHealthChecker(client, ServerName, ServerVersion),
		metrics:       miro.NewMetricsCollector(),
		card:          buildServerCard(),
	}
}

// =============================================================================
// Server card
// =============================================================================

func TestBuildServerCard(t *testing.T) {
	card := buildServerCard()

	if card == nil {
		t.Fatal("buildServerCard returned nil")
	}
	if card.Version != ServerVersion {
		t.Errorf("Version = %q, want %q", card.Version, ServerVersion)
	}
	if !strings.Contains(card.Name, "miro-mcp-server") {
		t.Errorf("Name = %q, want it to identify the server", card.Name)
	}
	if card.Repository == nil || card.Repository.URL == "" {
		t.Error("Repository should be populated")
	}
}

// =============================================================================
// HTTP wiring
// =============================================================================

func TestBuildHTTPMux_RoutesUnauthenticated(t *testing.T) {
	mux := buildHTTPMux(testHTTPOpts(t, ""))

	t.Run("health is served", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

		if rec.Code != http.StatusOK && rec.Code != http.StatusServiceUnavailable {
			t.Errorf("status = %d, want 200 or 503", rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
			t.Errorf("Cache-Control = %q, want no-store", cc)
		}

		var body map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Errorf("health body is not JSON: %v", err)
		}
	})

	t.Run("metrics is reachable without a token", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rec.Code)
		}
	})

	t.Run("server card is served unauthenticated with remotes filled in", func(t *testing.T) {
		opts := testHTTPOpts(t, "secret")
		mux := buildHTTPMux(opts)

		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/.well-known/mcp-server-card", nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 even when a bearer token is set", rec.Code)
		}
		if len(opts.card.Remotes) != 1 {
			t.Fatalf("Remotes = %d, want 1", len(opts.card.Remotes))
		}
		if opts.card.Remotes[0].Type != "streamable-http" {
			t.Errorf("Remotes[0].Type = %q, want streamable-http", opts.card.Remotes[0].Type)
		}
	})
}

func TestBuildHTTPMux_BearerTokenGuardsMetricsAndRoot(t *testing.T) {
	mux := buildHTTPMux(testHTTPOpts(t, "secret"))

	for _, path := range []string{"/metrics", "/"} {
		t.Run("unauthenticated "+path+" is rejected", func(t *testing.T) {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", rec.Code)
			}
		})
	}

	t.Run("health stays open when a token is set", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

		if rec.Code == http.StatusUnauthorized {
			t.Error("health must remain reachable for liveness probes")
		}
	})

	t.Run("a valid token reaches metrics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		req.Header.Set("Authorization", "Bearer secret")

		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rec.Code)
		}
	})
}

func TestBuildHealthHandler_DeepQueryParam(t *testing.T) {
	opts := testHTTPOpts(t, "")
	handler := buildHealthHandler(opts.healthChecker, opts.logger)

	for _, q := range []string{"/health", "/health?deep=true"} {
		rec := httptest.NewRecorder()
		handler(rec, httptest.NewRequest(http.MethodGet, q, nil))

		if rec.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("%s: Cache-Control = %q, want no-store", q, rec.Header().Get("Cache-Control"))
		}
		var body map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Errorf("%s: body is not JSON: %v", q, err)
		}
		if _, ok := body["status"]; !ok {
			t.Errorf("%s: body has no status field", q)
		}
	}
}

func TestHealthStatusCodes(t *testing.T) {
	// Degraded deliberately maps to 200: the service is still reachable.
	tests := []struct {
		status miro.HealthStatus
		want   int
	}{
		{miro.HealthStatusHealthy, http.StatusOK},
		{miro.HealthStatusDegraded, http.StatusOK},
		{miro.HealthStatusUnhealthy, http.StatusServiceUnavailable},
	}
	for _, tt := range tests {
		if got := healthStatusCodes[tt.status]; got != tt.want {
			t.Errorf("healthStatusCodes[%v] = %d, want %d", tt.status, got, tt.want)
		}
	}
}

func TestWrapNoStore(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("body"))
	})

	rec := httptest.NewRecorder()
	wrapNoStore(inner).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", rec.Header().Get("Cache-Control"))
	}
	if rec.Code != http.StatusTeapot {
		t.Errorf("status = %d, want the inner handler's 418", rec.Code)
	}
	if rec.Body.String() != "body" {
		t.Errorf("body = %q, want the inner handler's body", rec.Body.String())
	}
}

func TestLogHTTPSecurityWarnings(t *testing.T) {
	// No assertion on log content; this pins the branch matrix (token set or
	// not, loopback or not) so a panic or nil-deref in either arm surfaces.
	for _, tt := range []struct {
		name  string
		addr  string
		token string
	}{
		{"loopback without token", "127.0.0.1:8080", ""},
		{"loopback with token", "127.0.0.1:8080", "secret"},
		{"external without token", "0.0.0.0:8080", ""},
		{"external with token", "0.0.0.0:8080", "secret"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			logHTTPSecurityWarnings(httpServerOpts{
				logger:      quietLogger(),
				addr:        tt.addr,
				bearerToken: tt.token,
			})
		})
	}
}

// =============================================================================
// Startup wiring
// =============================================================================

func TestLoadMiroConfig_Unconfigured(t *testing.T) {
	t.Setenv("MIRO_ACCESS_TOKEN", "")

	config := loadMiroConfig(quietLogger())
	if config == nil {
		t.Fatal("loadMiroConfig returned nil")
	}
	if config.IsConfigured() {
		t.Error("config should report unconfigured with no token set")
	}
}

func TestLoadMiroConfig_Configured(t *testing.T) {
	// The token must pass isValidTokenFormat: loadMiroConfig calls log.Fatalf
	// on a malformed one, which would kill the test binary rather than fail.
	// JWT-shaped, 20+ chars, three non-empty dot-separated segments.
	t.Setenv("MIRO_ACCESS_TOKEN", "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.c2lnbmF0dXJl")

	config := loadMiroConfig(quietLogger())
	if config == nil {
		t.Fatal("loadMiroConfig returned nil")
	}
	if !config.IsConfigured() {
		t.Error("config should report configured when a token is set")
	}
}

func TestValidateMiroToken_UnconfiguredReturnsNil(t *testing.T) {
	logger := quietLogger()
	config := &miro.Config{}
	client := miro.NewClient(config, logger)

	if user := validateMiroToken(client, config, logger); user != nil {
		t.Errorf("user = %+v, want nil when unconfigured", user)
	}
}

func TestInitAuditLogger(t *testing.T) {
	logger := initAuditLogger(quietLogger())
	if logger == nil {
		t.Fatal("initAuditLogger returned nil")
	}
	defer logger.Close()
}

func TestCreateMCPServer(t *testing.T) {
	if server := createMCPServer(quietLogger()); server == nil {
		t.Fatal("createMCPServer returned nil")
	}
}

func TestInitDesirePath(t *testing.T) {
	dpLogger, normalizers := initDesirePath(quietLogger())

	if dpLogger == nil {
		t.Fatal("desire path logger is nil")
	}
	if len(normalizers) != 5 {
		t.Errorf("normalizers = %d, want the 5-strong standard chain", len(normalizers))
	}
	for i, n := range normalizers {
		if n == nil {
			t.Errorf("normalizers[%d] is nil", i)
		}
	}
}

func TestLoadShareAllowlist(t *testing.T) {
	t.Run("empty when nothing is set and no user", func(t *testing.T) {
		t.Setenv("MIRO_SHARE_ALLOWED_DOMAINS", "")
		t.Setenv("MIRO_SHARE_ALLOWED_EMAILS", "")

		allowlist := loadShareAllowlist(nil, quietLogger())
		if allowlist == nil {
			t.Fatal("loadShareAllowlist returned nil")
		}
		if !allowlist.IsEmpty() {
			t.Error("allowlist should be empty, so every share is rejected")
		}
	})

	t.Run("falls back to the user's own email domain", func(t *testing.T) {
		t.Setenv("MIRO_SHARE_ALLOWED_DOMAINS", "")
		t.Setenv("MIRO_SHARE_ALLOWED_EMAILS", "")

		allowlist := loadShareAllowlist(&miro.UserInfo{Email: "olga@example.com"}, quietLogger())
		if allowlist.IsEmpty() {
			t.Error("allowlist should fall back to the user's email domain")
		}
	})

	t.Run("explicit domains are used", func(t *testing.T) {
		t.Setenv("MIRO_SHARE_ALLOWED_EMAILS", "")
		t.Setenv("MIRO_SHARE_ALLOWED_DOMAINS", "example.com,other.org")

		allowlist := loadShareAllowlist(nil, quietLogger())
		if len(allowlist.Domains()) != 2 {
			t.Errorf("domains = %v, want 2", allowlist.Domains())
		}
	})

	t.Run("exact emails are authoritative", func(t *testing.T) {
		t.Setenv("MIRO_SHARE_ALLOWED_DOMAINS", "example.com")
		t.Setenv("MIRO_SHARE_ALLOWED_EMAILS", "someone@example.com")

		allowlist := loadShareAllowlist(nil, quietLogger())
		if len(allowlist.Emails()) != 1 {
			t.Errorf("emails = %v, want 1", allowlist.Emails())
		}
	})

	t.Run("an email var that normalizes to nothing fails closed", func(t *testing.T) {
		t.Setenv("MIRO_SHARE_ALLOWED_DOMAINS", "example.com")
		t.Setenv("MIRO_SHARE_ALLOWED_EMAILS", " , ")

		allowlist := loadShareAllowlist(nil, quietLogger())
		if len(allowlist.Emails()) != 0 {
			t.Errorf("emails = %v, want none", allowlist.Emails())
		}
	})
}

func TestRegisterTools_ProfileHandling(t *testing.T) {
	for _, tt := range []struct {
		name    string
		profile string
	}{
		{"unset falls back to full", ""},
		{"explicit full", "full"},
		{"essentials", "essentials"},
		{"an unknown value falls back to full", "not-a-real-profile"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MIRO_TOOLS_PROFILE", tt.profile)

			logger := quietLogger()
			server := createMCPServer(logger)
			client := miro.NewClient(&miro.Config{AccessToken: "test-token"}, logger)

			auditLogger := initAuditLogger(logger)
			defer auditLogger.Close()
			dpLogger, normalizers := initDesirePath(logger)

			// Registration must not panic for any profile value.
			registerTools(server, client, logger, registryDeps{
				auditLogger:    auditLogger,
				shareAllowlist: loadShareAllowlist(nil, logger),
				dpLogger:       dpLogger,
				normalizers:    normalizers,
			})
		})
	}
}

func TestRegisterTools_WithUser(t *testing.T) {
	t.Setenv("MIRO_TOOLS_PROFILE", "essentials")

	logger := quietLogger()
	server := createMCPServer(logger)
	client := miro.NewClient(&miro.Config{AccessToken: "test-token"}, logger)

	auditLogger := initAuditLogger(logger)
	defer auditLogger.Close()
	dpLogger, normalizers := initDesirePath(logger)

	registerTools(server, client, logger, registryDeps{
		auditLogger:    auditLogger,
		shareAllowlist: loadShareAllowlist(nil, logger),
		dpLogger:       dpLogger,
		normalizers:    normalizers,
		user:           &miro.UserInfo{ID: "user-123", Email: "olga@example.com"},
	})
}

func TestRegisterResourcesAndPrompts(t *testing.T) {
	logger := quietLogger()
	server := createMCPServer(logger)
	client := miro.NewClient(&miro.Config{AccessToken: "test-token"}, logger)

	registerResourcesAndPrompts(server, client, logger)
}

// =============================================================================
// CLI help output
// =============================================================================

func TestPrintAuthUsage(t *testing.T) {
	out := captureStdout(t, printAuthUsage)

	for _, want := range []string{"login", "status", "logout", "MIRO_CLIENT_ID", "MIRO_CLIENT_SECRET"} {
		if !strings.Contains(out, want) {
			t.Errorf("auth usage is missing %q\n%s", want, out)
		}
	}
}

func TestPrintOAuthSetupHelp(t *testing.T) {
	out := captureStdout(t, printOAuthSetupHelp)

	if !strings.Contains(out, "MIRO_CLIENT_ID") {
		t.Errorf("setup help should name MIRO_CLIENT_ID\n%s", out)
	}
	if !strings.Contains(out, "miro.com") {
		t.Errorf("setup help should link to the Miro app settings\n%s", out)
	}
}

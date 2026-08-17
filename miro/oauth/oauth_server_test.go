package oauth

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// Server Tests
// =============================================================================

// startCallbackServer builds and starts a callback server on a random port,
// stopping it when the test finishes.
func startCallbackServer(t *testing.T) *CallbackServer {
	t.Helper()
	server, err := NewCallbackServer(":0", testLogger())
	if err != nil {
		t.Fatalf("NewCallbackServer() error = %v", err)
	}
	t.Cleanup(func() { server.Stop(context.Background()) })
	server.Start()
	return server
}

// getCallback issues a GET against the running server and checks the status code.
func getCallback(t *testing.T, server *CallbackServer, pathQuery string, wantStatus int) {
	t.Helper()
	resp, err := http.Get("http://" + server.Addr() + pathQuery)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != wantStatus {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, wantStatus)
	}
}

// awaitCallbackResult waits for the server to report the callback result.
func awaitCallbackResult(t *testing.T, server *CallbackServer) *CallbackResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := server.WaitForCallback(ctx, 2*time.Second)
	if err != nil {
		t.Fatalf("WaitForCallback() error = %v", err)
	}
	return result
}

func TestGetCallbackPort(t *testing.T) {
	tests := []struct {
		name        string
		redirectURI string
		expected    string
		shouldError bool
	}{
		{
			name:        "with explicit port",
			redirectURI: "http://localhost:8089/callback",
			expected:    "127.0.0.1:8089",
		},
		{
			name:        "without port (http)",
			redirectURI: "http://localhost/callback",
			expected:    "127.0.0.1:80",
		},
		{
			name:        "without port (https)",
			redirectURI: "https://localhost/callback",
			expected:    "127.0.0.1:443",
		},
		{
			name:        "custom port",
			redirectURI: "http://127.0.0.1:9999/oauth",
			expected:    "127.0.0.1:9999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port, err := GetCallbackPort(tt.redirectURI)
			if tt.shouldError {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetCallbackPort() error = %v", err)
			}
			if port != tt.expected {
				t.Errorf("GetCallbackPort() = %q, want %q", port, tt.expected)
			}
		})
	}
}

func TestGetCallbackPort_BindsToLocalhost(t *testing.T) {
	tests := []struct {
		name        string
		redirectURI string
	}{
		{"http with port", "http://localhost:8089/callback"},
		{"https without port", "https://example.com/callback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := GetCallbackPort(tt.redirectURI)
			if err != nil {
				t.Fatalf("GetCallbackPort() error = %v", err)
			}
			if !strings.HasPrefix(addr, "127.0.0.1:") {
				t.Errorf("GetCallbackPort() = %q, should start with '127.0.0.1:'", addr)
			}
		})
	}
}

// =============================================================================
// CallbackServer Tests
// =============================================================================

func TestNewCallbackServer(t *testing.T) {
	// Test successful creation (:0 picks a random available port)
	server, err := NewCallbackServer(":0", testLogger())
	if err != nil {
		t.Fatalf("NewCallbackServer() error = %v", err)
	}
	defer server.Stop(context.Background())

	if server.Addr() == "" {
		t.Error("server address should not be empty")
	}
}

func TestNewCallbackServer_InvalidAddress(t *testing.T) {
	// Try to bind to an invalid address
	_, err := NewCallbackServer("invalid:address:format", testLogger())
	if err == nil {
		t.Error("expected error for invalid address")
	}
}

func TestCallbackServer_HandleCallback_Success(t *testing.T) {
	server := startCallbackServer(t)

	// Make a request to the callback endpoint with code and state
	getCallback(t, server, "/callback?code=test-auth-code&state=test-state", http.StatusOK)

	// Wait for the result
	result := awaitCallbackResult(t, server)

	if result.Code != "test-auth-code" {
		t.Errorf("Code = %q, want %q", result.Code, "test-auth-code")
	}
	if result.State != "test-state" {
		t.Errorf("State = %q, want %q", result.State, "test-state")
	}
	if result.Error != nil {
		t.Errorf("Error should be nil, got %v", result.Error)
	}
}

func TestCallbackServer_HandleCallback_Error(t *testing.T) {
	server := startCallbackServer(t)

	// Make a request with an error
	getCallback(t, server, "/callback?error=access_denied&error_description=User+denied+access", http.StatusBadRequest)

	// Wait for the result
	result := awaitCallbackResult(t, server)

	if result.Error == nil {
		t.Fatal("Error should not be nil")
	}
	if result.Error.Code != "access_denied" {
		t.Errorf("Error.Code = %q, want %q", result.Error.Code, "access_denied")
	}
	if result.Error.Description != "User denied access" {
		t.Errorf("Error.Description = %q, want %q", result.Error.Description, "User denied access")
	}
}

func TestCallbackServer_HandleCallback_MissingCode(t *testing.T) {
	server := startCallbackServer(t)

	// Make a request without a code
	getCallback(t, server, "/callback?state=test-state", http.StatusBadRequest)

	// Wait for the result
	result := awaitCallbackResult(t, server)

	if result.Error == nil {
		t.Fatal("Error should not be nil")
	}
	if result.Error.Code != "missing_code" {
		t.Errorf("Error.Code = %q, want %q", result.Error.Code, "missing_code")
	}
}

func TestCallbackServer_HandleRoot(t *testing.T) {
	server := startCallbackServer(t)

	// Make a request to root
	resp, err := http.Get("http://" + server.Addr() + "/")
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", contentType, "text/html; charset=utf-8")
	}
}

func TestCallbackServer_WaitForCallback_Timeout(t *testing.T) {
	server := startCallbackServer(t)

	// Wait with a very short timeout (no callback will come)
	ctx := context.Background()
	_, err := server.WaitForCallback(ctx, 10*time.Millisecond)
	if err == nil {
		t.Error("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("error should mention timeout, got %q", err.Error())
	}
}

// =============================================================================
// XSS Prevention Tests
// =============================================================================

func TestWriteErrorPage_XSSPrevention(t *testing.T) {
	server, err := NewCallbackServer("127.0.0.1:0", testLogger())
	if err != nil {
		t.Fatalf("NewCallbackServer() error = %v", err)
	}
	defer server.Stop(context.Background())

	server.Start()

	// Send a callback with XSS payload in error params
	addr := "http://" + server.Addr()
	xssPayload := `<script>alert('xss')</script>`
	resp, err := http.Get(addr + "/callback?error=" + xssPayload + "&error_description=" + xssPayload)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// The raw script tag should NOT appear in the response
	if strings.Contains(bodyStr, "<script>") {
		t.Error("response contains unescaped <script> tag; XSS vulnerability present")
	}

	// The escaped version should appear
	if !strings.Contains(bodyStr, "&lt;script&gt;") {
		t.Error("response should contain HTML-escaped script tag")
	}
}

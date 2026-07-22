package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// =============================================================================
// Bearer Token Middleware Tests
// =============================================================================

func TestBearerTokenMiddleware_ValidToken(t *testing.T) {
	handler := bearerTokenMiddleware("test-secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer test-secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestBearerTokenMiddleware_MissingToken(t *testing.T) {
	handler := bearerTokenMiddleware("test-secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("StatusCode = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if rec.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Errorf("WWW-Authenticate = %q, want 'Bearer'", rec.Header().Get("WWW-Authenticate"))
	}
}

func TestBearerTokenMiddleware_WrongToken(t *testing.T) {
	handler := bearerTokenMiddleware("test-secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("StatusCode = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestBearerTokenMiddleware_BasicAuthRejected(t *testing.T) {
	handler := bearerTokenMiddleware("test-secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic dGVzdDp0ZXN0")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("StatusCode = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// =============================================================================
// HTTP Startup Security Validation Tests
// =============================================================================

func TestValidateHTTPSecurity(t *testing.T) {
	tests := []struct {
		name        string
		addr        string
		bearerToken string
		wantErr     bool
	}{
		{"loopback without token allowed", "127.0.0.1:8080", "", false},
		{"localhost without token allowed", "localhost:8080", "", false},
		{"loopback with token allowed", "127.0.0.1:8080", "secret", false},
		{"non-loopback with token allowed", "0.0.0.0:8080", "secret", false},
		{"non-loopback without token rejected", "0.0.0.0:8080", "", true},
		{"bind-all-ports without token rejected", ":8080", "", true},
		{"external host without token rejected", "192.168.1.10:8080", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHTTPSecurity(tt.addr, tt.bearerToken)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateHTTPSecurity(%q, %q) error = %v, wantErr %v", tt.addr, tt.bearerToken, err, tt.wantErr)
			}
		})
	}
}

func TestIsLoopbackAddr(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{"127.0.0.1:8080", true},
		{"localhost:8080", true},
		{"0.0.0.0:8080", false},
		{":8080", false},
		{"192.168.1.10:8080", false},
	}

	for _, tt := range tests {
		if got := isLoopbackAddr(tt.addr); got != tt.want {
			t.Errorf("isLoopbackAddr(%q) = %v, want %v", tt.addr, got, tt.want)
		}
	}
}

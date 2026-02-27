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

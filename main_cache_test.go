package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// TestCreateMCPServer_BothMiddlewaresFire pins that createMCPServer attaches
// BOTH receiving middlewares and that neither displaces the other.
//
// This server is the only one in the portfolio running two receiving
// middlewares, so it is the only place the composition can break. The two
// assertions fail independently:
//
//   - drop mcpcache from the AddReceivingMiddleware call and the ttlMs
//     assertion fails with "ttlMs = 0, want 1800000"
//   - drop mcpotel and the span assertion fails with "no OTel span recorded"
//
// Both were run. An assertion on the config alone would pass with either
// middleware unwired, which is the whole failure mode.
//
// Driven through buildHTTPMux rather than an in-memory stdio session because
// this server serves both transports from one *mcp.Server; a stdio-only
// assertion would pass while the HTTP path went unstamped.
func TestCreateMCPServer_BothMiddlewaresFire(t *testing.T) {
	// mcpotel resolves otel.GetTracerProvider() when Middleware() is
	// constructed, which happens inside createMCPServer. The global must
	// therefore be swapped BEFORE testHTTPOpts builds the server.
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		otel.SetTracerProvider(prev)
		_ = tp.Shutdown(t.Context())
	})

	mux := buildHTTPMux(testHTTPOpts(t, ""))

	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params": map[string]any{"_meta": map[string]any{
			"io.modelcontextprotocol/protocolVersion":    "2026-07-28",
			"io.modelcontextprotocol/clientInfo":         map[string]any{"name": "test", "version": "1"},
			"io.modelcontextprotocol/clientCapabilities": map[string]any{},
		}},
	})
	if err != nil {
		t.Fatalf("marshalling request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Mcp-Protocol-Version", "2026-07-28")
	req.Header.Set("Mcp-Method", "tools/list")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}

	// mcpcache fired: the result advertises a real TTL rather than the SDK's 0.
	const wantTTL = 1800000 // thirty minutes, matching createMCPServer
	if got := extractTTLMs(t, rec.Body.String()); got != wantTTL {
		t.Errorf("ttlMs = %d, want %d", got, wantTTL)
	}

	// mcpotel fired: a server span was recorded for the same call.
	if !hasSpanForMethod(exporter.GetSpans(), "tools/list") {
		t.Error("no OTel span recorded for tools/list; mcpotel middleware did not fire")
	}
}

// hasSpanForMethod reports whether any exported span carries the mcp.method.name
// attribute for the given method. Matching on the attribute rather than the span
// name keeps this independent of mcpotel's span-naming scheme.
func hasSpanForMethod(spans tracetest.SpanStubs, method string) bool {
	for _, s := range spans {
		for _, attr := range s.Attributes {
			if string(attr.Key) == "mcp.method.name" && attr.Value.AsString() == method {
				return true
			}
		}
	}
	return false
}

// extractTTLMs pulls ttlMs off a tools/list response. The handler may answer as
// plain JSON or as an SSE frame depending on the Accept header, so scan for the
// data payload rather than assuming one shape.
func extractTTLMs(t *testing.T, raw string) int {
	t.Helper()

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimPrefix(strings.TrimSpace(line), "data: ")
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var envelope struct {
			Result struct {
				TTLMs *int `json:"ttlMs"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(line), &envelope); err != nil {
			continue
		}
		if envelope.Result.TTLMs != nil {
			return *envelope.Result.TTLMs
		}
	}

	t.Fatalf("no ttlMs field found in response: %s", raw)
	return 0
}

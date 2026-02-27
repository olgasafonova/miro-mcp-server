package oauth

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"
)

// CallbackResult represents the result of an OAuth callback.
type CallbackResult struct {
	Code  string
	State string
	Error *AuthError
}

// CallbackServer handles the OAuth redirect callback.
type CallbackServer struct {
	server   *http.Server
	listener net.Listener
	logger   *slog.Logger
	resultCh chan CallbackResult
}

// NewCallbackServer creates a new callback server.
func NewCallbackServer(addr string, logger *slog.Logger) (*CallbackServer, error) {
	// Parse the address to get the port
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s := &CallbackServer{
		listener: listener,
		logger:   logger,
		resultCh: make(chan CallbackResult, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", s.handleCallback)
	mux.HandleFunc("/", s.handleRoot)

	s.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return s, nil
}

// Start begins serving HTTP requests.
func (s *CallbackServer) Start() {
	go func() {
		if err := s.server.Serve(s.listener); err != http.ErrServerClosed {
			s.logger.Error("Callback server error", "error", err)
		}
	}()
}

// Stop shuts down the server.
func (s *CallbackServer) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// WaitForCallback waits for the OAuth callback with a timeout.
func (s *CallbackServer) WaitForCallback(ctx context.Context, timeout time.Duration) (*CallbackResult, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case result := <-s.resultCh:
		return &result, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("timed out waiting for OAuth callback")
	}
}

// Addr returns the address the server is listening on.
func (s *CallbackServer) Addr() string {
	return s.listener.Addr().String()
}

func (s *CallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	// Check for error in callback
	if errCode := query.Get("error"); errCode != "" {
		s.resultCh <- CallbackResult{
			Error: &AuthError{
				Code:        errCode,
				Description: query.Get("error_description"),
			},
		}
		s.writeErrorPage(w, errCode, query.Get("error_description"))
		return
	}

	code := query.Get("code")
	state := query.Get("state")

	if code == "" {
		s.resultCh <- CallbackResult{
			Error: &AuthError{
				Code:        "missing_code",
				Description: "Authorization code not found in callback",
			},
		}
		s.writeErrorPage(w, "missing_code", "Authorization code not found")
		return
	}

	s.resultCh <- CallbackResult{
		Code:  code,
		State: state,
	}

	s.writeSuccessPage(w)
}

func (s *CallbackServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head>
    <title>Miro MCP Server - OAuth</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; padding: 40px; text-align: center; }
        h1 { color: #333; }
        p { color: #666; }
    </style>
</head>
<body>
    <h1>Miro MCP Server</h1>
    <p>Waiting for authorization...</p>
</body>
</html>`)
}

func (s *CallbackServer) writeSuccessPage(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head>
    <title>Authorization Successful</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; padding: 40px; text-align: center; background: #f0f9f0; }
        h1 { color: #2e7d32; }
        p { color: #666; }
        .icon { font-size: 64px; margin-bottom: 20px; }
    </style>
</head>
<body>
    <div class="icon">&#10004;</div>
    <h1>Authorization Successful!</h1>
    <p>You can close this window and return to the terminal.</p>
    <p>The MCP server is now connected to your Miro account.</p>
    <script>setTimeout(function() { window.close(); }, 3000);</script>
</body>
</html>`)
}

func (s *CallbackServer) writeErrorPage(w http.ResponseWriter, code, description string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <title>Authorization Failed</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; padding: 40px; text-align: center; background: #fff0f0; }
        h1 { color: #c62828; }
        p { color: #666; }
        .error { color: #c62828; font-family: monospace; background: #ffebee; padding: 10px; border-radius: 4px; }
        .icon { font-size: 64px; margin-bottom: 20px; }
    </style>
</head>
<body>
    <div class="icon">&#10006;</div>
    <h1>Authorization Failed</h1>
    <p class="error">%s: %s</p>
    <p>Please try again or check your Miro app settings.</p>
</body>
</html>`, html.EscapeString(code), html.EscapeString(description))
}

// GetCallbackPort extracts the port from a redirect URI.
func GetCallbackPort(redirectURI string) (string, error) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return "", fmt.Errorf("invalid redirect URI: %w", err)
	}

	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	return "127.0.0.1:" + port, nil
}

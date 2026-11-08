package oauth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// AuthFlow.Login tests
//
// Login binds a real callback server on 127.0.0.1 and normally opens a
// browser. The tests replace AuthFlow.openURL with a fake "browser" that reads
// the authorization URL and calls the callback the way Miro's redirect would,
// so the whole round trip runs locally with no network and no browser.
// =============================================================================

// freeRedirectURI returns a localhost redirect URI on a port that was free a
// moment ago.
func freeRedirectURI(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("finding a free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	if err := l.Close(); err != nil {
		t.Fatalf("releasing port: %v", err)
	}
	return fmt.Sprintf("http://localhost:%d/callback", port)
}

// fakeBrowser returns an openURL replacement that "approves" the consent
// screen by requesting the callback with the given query. The query is built
// from the authorization URL so a test can echo back (or tamper with) state.
func fakeBrowser(t *testing.T, redirectURI string, query func(authState string) url.Values) func(string) error {
	t.Helper()
	return func(authURL string) error {
		u, err := url.Parse(authURL)
		if err != nil {
			return err
		}
		callback := redirectURI + "?" + query(u.Query().Get("state")).Encode()
		go func() {
			resp, err := http.Get(callback) //nolint:gosec // test-local callback URL
			if err != nil {
				t.Errorf("calling callback: %v", err)
				return
			}
			_ = resp.Body.Close()
		}()
		return nil
	}
}

// newLoginFlow builds an AuthFlow on a free callback port whose token
// endpoint is served by tokenHandler.
func newLoginFlow(t *testing.T, tokenHandler func(*http.Request) *http.Response) (*AuthFlow, *MemoryTokenStore) {
	t.Helper()
	flow, store := newTestAuthFlow(t, nil, tokenHandler)
	flow.config.RedirectURI = freeRedirectURI(t)
	flow.provider.config.RedirectURI = flow.config.RedirectURI
	return flow, store
}

func loginCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestAuthFlowLogin_Success(t *testing.T) {
	var exchangeBody string
	flow, store := newLoginFlow(t, func(r *http.Request) *http.Response {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing token request: %v", err)
		}
		exchangeBody = r.PostForm.Encode()
		return jsonResponse(http.StatusOK, `{
			"access_token": "login-access-token",
			"refresh_token": "login-refresh-token",
			"expires_in": 3600,
			"token_type": "bearer",
			"user_id": "user-42",
			"team_id": "team-7"
		}`)
	})
	flow.openURL = fakeBrowser(t, flow.config.RedirectURI, func(state string) url.Values {
		return url.Values{"code": {"auth-code-1"}, "state": {state}}
	})

	tokens, err := flow.Login(loginCtx(t))
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if tokens.AccessToken != "login-access-token" || tokens.UserID != "user-42" || tokens.TeamID != "team-7" {
		t.Errorf("unexpected tokens: %+v", tokens)
	}

	// The code from the callback, plus a PKCE verifier, must reach the
	// token endpoint.
	if !strings.Contains(exchangeBody, "code=auth-code-1") {
		t.Errorf("token request missing code: %s", exchangeBody)
	}
	if !strings.Contains(exchangeBody, "code_verifier=") {
		t.Errorf("token request missing PKCE verifier: %s", exchangeBody)
	}

	saved, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("tokens not saved: %v", err)
	}
	if saved.AccessToken != "login-access-token" {
		t.Errorf("saved access token = %q", saved.AccessToken)
	}
}

func TestAuthFlowLogin_StateMismatchIsRejected(t *testing.T) {
	flow, store := newLoginFlow(t, func(*http.Request) *http.Response {
		t.Error("token endpoint must not be called when state does not match")
		return jsonResponse(http.StatusOK, `{}`)
	})
	flow.openURL = fakeBrowser(t, flow.config.RedirectURI, func(string) url.Values {
		return url.Values{"code": {"auth-code-1"}, "state": {"forged-state"}}
	})

	_, err := flow.Login(loginCtx(t))
	if err == nil || !strings.Contains(err.Error(), "state mismatch") {
		t.Fatalf("expected state mismatch error, got %v", err)
	}
	if store.Exists(context.Background()) {
		t.Error("tokens must not be saved after a state mismatch")
	}
}

func TestAuthFlowLogin_UserDeniesAccess(t *testing.T) {
	flow, _ := newLoginFlow(t, func(*http.Request) *http.Response {
		t.Error("token endpoint must not be called when the user denies access")
		return jsonResponse(http.StatusOK, `{}`)
	})
	flow.openURL = fakeBrowser(t, flow.config.RedirectURI, func(string) url.Values {
		return url.Values{"error": {"access_denied"}, "error_description": {"User said no"}}
	})

	_, err := flow.Login(loginCtx(t))
	var authErr *AuthError
	if !errors.As(err, &authErr) || authErr.Code != "access_denied" {
		t.Fatalf("expected access_denied AuthError, got %v", err)
	}
}

func TestAuthFlowLogin_TokenExchangeFailure(t *testing.T) {
	flow, store := newLoginFlow(t, func(*http.Request) *http.Response {
		return jsonResponse(http.StatusBadRequest, `{"error":"invalid_grant","error_description":"code expired"}`)
	})
	flow.openURL = fakeBrowser(t, flow.config.RedirectURI, func(state string) url.Values {
		return url.Values{"code": {"stale-code"}, "state": {state}}
	})

	_, err := flow.Login(loginCtx(t))
	if err == nil || !strings.Contains(err.Error(), "token exchange failed") {
		t.Fatalf("expected token exchange failure, got %v", err)
	}
	if store.Exists(context.Background()) {
		t.Error("tokens must not be saved after a failed exchange")
	}
}

func TestAuthFlowLogin_BrowserFailureStillCompletes(t *testing.T) {
	// If the browser cannot be opened, Login prints the URL and keeps
	// waiting; a user who opens it by hand still completes the flow.
	flow, _ := newLoginFlow(t, func(*http.Request) *http.Response {
		return jsonResponse(http.StatusOK, `{"access_token":"manual-access","expires_in":3600,"token_type":"bearer"}`)
	})
	approve := fakeBrowser(t, flow.config.RedirectURI, func(state string) url.Values {
		return url.Values{"code": {"auth-code-2"}, "state": {state}}
	})
	flow.openURL = func(authURL string) error {
		_ = approve(authURL) // the "user" opens the printed URL
		return errors.New("no browser available")
	}

	tokens, err := flow.Login(loginCtx(t))
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if tokens.AccessToken != "manual-access" {
		t.Errorf("access token = %q", tokens.AccessToken)
	}
}

func TestAuthFlowLogin_InvalidRedirectURI(t *testing.T) {
	flow, _ := newTestAuthFlow(t, nil, nil)
	flow.config.RedirectURI = "://not a url"

	_, err := flow.Login(loginCtx(t))
	if err == nil || !strings.Contains(err.Error(), "invalid redirect URI") {
		t.Fatalf("expected invalid redirect URI error, got %v", err)
	}
}

func TestAuthFlowLogin_CallbackPortInUse(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("occupying a port: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })

	flow, _ := newTestAuthFlow(t, nil, nil)
	flow.config.RedirectURI = fmt.Sprintf("http://localhost:%d/callback", l.Addr().(*net.TCPAddr).Port)

	_, err = flow.Login(loginCtx(t))
	if err == nil || !strings.Contains(err.Error(), "failed to start callback server") {
		t.Fatalf("expected callback server error, got %v", err)
	}
}

func TestNewAuthFlow_UsesRealBrowserOpener(t *testing.T) {
	flow := NewAuthFlow(&Config{TokenStorePath: t.TempDir() + "/tokens.json"}, nil)
	if flow.openURL == nil {
		t.Fatal("NewAuthFlow must set openURL")
	}
}

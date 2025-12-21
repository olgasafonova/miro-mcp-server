package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// Miro OAuth endpoints
	authorizationURL = "https://miro.com/oauth/authorize"
	tokenURL         = "https://api.miro.com/v1/oauth/token"
	revokeURL        = "https://api.miro.com/v1/oauth/revoke"
)

// Provider handles OAuth 2.1 authorization code flow with PKCE.
type Provider struct {
	config     *Config
	httpClient *http.Client
}

// NewProvider creates a new OAuth provider.
func NewProvider(config *Config) *Provider {
	return &Provider{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GenerateAuthorizationState creates a new state and code verifier for PKCE.
func (p *Provider) GenerateAuthorizationState() (*AuthorizationState, error) {
	// Generate random state (32 bytes = 256 bits)
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}
	state := base64.URLEncoding.EncodeToString(stateBytes)

	// Generate code verifier for PKCE (43-128 characters)
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}
	codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	return &AuthorizationState{
		State:        state,
		CodeVerifier: codeVerifier,
		CreatedAt:    time.Now(),
	}, nil
}

// GetAuthorizationURL returns the URL to redirect users for authorization.
func (p *Provider) GetAuthorizationURL(state *AuthorizationState) string {
	// Generate code challenge from verifier (S256)
	h := sha256.Sum256([]byte(state.CodeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(h[:])

	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {p.config.ClientID},
		"redirect_uri":          {p.config.RedirectURI},
		"scope":                 {strings.Join(p.config.Scopes, " ")},
		"state":                 {state.State},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
	}

	return authorizationURL + "?" + params.Encode()
}

// ExchangeCode trades an authorization code for tokens.
func (p *Provider) ExchangeCode(ctx context.Context, code string, codeVerifier string) (*TokenSet, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {p.config.ClientID},
		"client_secret": {p.config.ClientSecret},
		"code":          {code},
		"redirect_uri":  {p.config.RedirectURI},
		"code_verifier": {codeVerifier},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var authErr AuthError
		if json.Unmarshal(body, &authErr) == nil && authErr.Code != "" {
			return nil, &authErr
		}
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return tokenResp.ToTokenSet(), nil
}

// RefreshToken gets a new access token using the refresh token.
func (p *Provider) RefreshToken(ctx context.Context, refreshToken string) (*TokenSet, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {p.config.ClientID},
		"client_secret": {p.config.ClientSecret},
		"refresh_token": {refreshToken},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var authErr AuthError
		if json.Unmarshal(body, &authErr) == nil && authErr.Code != "" {
			return nil, &authErr
		}
		return nil, fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return tokenResp.ToTokenSet(), nil
}

// RevokeToken invalidates a token.
func (p *Provider) RevokeToken(ctx context.Context, token string) error {
	data := url.Values{
		"client_id":     {p.config.ClientID},
		"client_secret": {p.config.ClientSecret},
		"token":         {token},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, revokeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("token revocation failed: %w", err)
	}
	defer resp.Body.Close()

	// Revocation returns 200 OK even if token was already invalid
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token revocation failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

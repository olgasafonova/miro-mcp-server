package miro

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// whoAmISample mirrors the live GET /v1/oauth-token answer captured 18-08-2026:
// every entity carries a type discriminator, and createdBy duplicates user.
const whoAmISample = `{
  "type": "oAuthToken",
  "application": {"type": "application", "name": "Go Miro", "id": "app-1"},
  "createdBy": {"type": "user", "name": "Olga Safonova", "id": "u-1"},
  "team": {"type": "team", "name": "Dev team", "id": "t-1"},
  "scopes": ["boards:write", "boards:read"],
  "organization": {"type": "organization", "name": "Dev team", "id": "org-1"},
  "user": {"type": "user", "name": "Olga Safonova", "id": "u-1"}
}`

// newWhoAmIServer captures the request path and replies with the given body.
func newWhoAmIServer(t *testing.T, body string, gotPath *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if gotPath != nil {
			*gotPath = r.URL.Path
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
}

func TestWhoAmIParsesTokenContext(t *testing.T) {
	var gotPath string
	server := newWhoAmIServer(t, whoAmISample, &gotPath)
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.WhoAmI(context.Background(), WhoAmIArgs{})
	if err != nil {
		t.Fatalf("WhoAmI() error = %v", err)
	}

	if gotPath != "/oauth-token" {
		t.Errorf("path = %q, want /oauth-token", gotPath)
	}
	if result.User == nil || result.User.ID != "u-1" || result.User.Name != "Olga Safonova" {
		t.Errorf("user = %+v, want u-1/Olga Safonova", result.User)
	}
	if result.Team == nil || result.Team.ID != "t-1" || result.Team.Name != "Dev team" {
		t.Errorf("team = %+v, want t-1/Dev team", result.Team)
	}
	if result.Organization == nil || result.Organization.ID != "org-1" {
		t.Errorf("organization = %+v, want org-1", result.Organization)
	}
	if result.Application == nil || result.Application.ID != "app-1" || result.Application.Name != "Go Miro" {
		t.Errorf("application = %+v, want app-1/Go Miro", result.Application)
	}
	if len(result.Scopes) != 2 || result.Scopes[0] != "boards:write" || result.Scopes[1] != "boards:read" {
		t.Errorf("scopes = %v, want [boards:write boards:read]", result.Scopes)
	}
}

func TestWhoAmIOmitsAbsentEntities(t *testing.T) {
	// A minimal answer: no organization, no application, no scopes array.
	server := newWhoAmIServer(t, `{"type":"oAuthToken","user":{"id":"u-1","name":"Olga"},"team":{"id":"t-1","name":"Dev"}}`, nil)
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.WhoAmI(context.Background(), WhoAmIArgs{})
	if err != nil {
		t.Fatalf("WhoAmI() error = %v", err)
	}

	if result.Organization != nil {
		t.Errorf("organization = %+v, want nil", result.Organization)
	}
	if result.Application != nil {
		t.Errorf("application = %+v, want nil", result.Application)
	}
	if result.Scopes == nil || len(result.Scopes) != 0 {
		t.Errorf("scopes = %v, want empty non-nil slice", result.Scopes)
	}
}

func TestWhoAmIMalformedResponse(t *testing.T) {
	server := newWhoAmIServer(t, `not json`, nil)
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	if _, err := client.WhoAmI(context.Background(), WhoAmIArgs{}); err == nil {
		t.Fatal("WhoAmI() with malformed body: expected error, got nil")
	}
}

func TestWhoAmIAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"invalid token"}`))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	if _, err := client.WhoAmI(context.Background(), WhoAmIArgs{}); err == nil {
		t.Fatal("WhoAmI() on 401: expected error, got nil")
	}
}

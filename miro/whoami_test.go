package miro

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
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

// assertWhoAmIEntity checks one entity against its expected value.
func assertWhoAmIEntity(t *testing.T, label string, got *WhoAmIEntity, want WhoAmIEntity) {
	t.Helper()
	if got == nil {
		t.Errorf("%s = nil, want %+v", label, want)
		return
	}
	if *got != want {
		t.Errorf("%s = %+v, want %+v", label, *got, want)
	}
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
	assertWhoAmIEntity(t, "user", result.User, WhoAmIEntity{ID: "u-1", Name: "Olga Safonova"})
	assertWhoAmIEntity(t, "team", result.Team, WhoAmIEntity{ID: "t-1", Name: "Dev team"})
	assertWhoAmIEntity(t, "organization", result.Organization, WhoAmIEntity{ID: "org-1", Name: "Dev team"})
	assertWhoAmIEntity(t, "application", result.Application, WhoAmIEntity{ID: "app-1", Name: "Go Miro"})
	if !reflect.DeepEqual(result.Scopes, []string{"boards:write", "boards:read"}) {
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

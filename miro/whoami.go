package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// =============================================================================
// Token Introspection (GET /v1/oauth-token)
// =============================================================================

// apiWhoAmIEntity is the wire shape of one entity on the token: the API tags
// each with a type discriminator this projection does not need.
type apiWhoAmIEntity struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// toEntity projects a wire entity, collapsing an absent one to nil so the
// result omits it entirely.
func (e *apiWhoAmIEntity) toEntity() *WhoAmIEntity {
	if e == nil || (e.ID == "" && e.Name == "") {
		return nil
	}
	return &WhoAmIEntity{ID: e.ID, Name: e.Name}
}

// WhoAmI reports the access token's context: whose token it is, which team
// and organization it is scoped to, and which scopes it carries.
//
// This is the one remaining v1 endpoint the server calls; Miro never ported
// token introspection to v2. Deliberately uncached — the whole point of the
// tool is a live answer when debugging a 403.
func (c *Client) WhoAmI(ctx context.Context, _ WhoAmIArgs) (WhoAmIResult, error) {
	respBody, err := c.requestV1(ctx, http.MethodGet, "/oauth-token", nil)
	if err != nil {
		return WhoAmIResult{}, err
	}

	var resp struct {
		User         *apiWhoAmIEntity `json:"user"`
		Team         *apiWhoAmIEntity `json:"team"`
		Organization *apiWhoAmIEntity `json:"organization"`
		Application  *apiWhoAmIEntity `json:"application"`
		Scopes       []string         `json:"scopes"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return WhoAmIResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	scopes := resp.Scopes
	if scopes == nil {
		scopes = []string{}
	}

	return WhoAmIResult{
		User:         resp.User.toEntity(),
		Team:         resp.Team.toEntity(),
		Organization: resp.Organization.toEntity(),
		Application:  resp.Application.toEntity(),
		Scopes:       scopes,
	}, nil
}

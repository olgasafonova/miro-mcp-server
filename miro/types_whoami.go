package miro

// =============================================================================
// Token Introspection Types (GET /v1/oauth-token)
// =============================================================================
//
// Wire shape captured by live probe 18-08-2026: the endpoint answers a plain
// personal access token with the token's user, team, organization, application
// and scopes. It is the REST twin of the official connector's user_who_am_i.

// WhoAmIArgs takes no parameters; the token itself is the subject.
type WhoAmIArgs struct{}

// WhoAmIEntity is a named entity attached to the token (user, team,
// organization, or the app the token was minted for).
type WhoAmIEntity struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// WhoAmIResult describes whose token this is and what it may do.
type WhoAmIResult struct {
	// User is the account the token acts as.
	User *WhoAmIEntity `json:"user,omitempty"`

	// Team is the team the token is scoped to.
	Team *WhoAmIEntity `json:"team,omitempty"`

	// Organization is the org the team belongs to.
	Organization *WhoAmIEntity `json:"organization,omitempty"`

	// Application is the Miro app the token was created under.
	Application *WhoAmIEntity `json:"application,omitempty"`

	// Scopes lists what the token is allowed to do (e.g. boards:read).
	Scopes []string `json:"scopes"`
}

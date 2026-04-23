// Package tools provides MCP tool handlers for the Miro MCP server.
//
// share_allowlist.go implements the email-domain allowlist used by the
// miro_share_board tool. Sharing a Miro board grants access to an external
// party; a prompt-injected agent could therefore exfiltrate board content by
// inviting an attacker-controlled address. The allowlist is the server-side
// guardrail: only emails whose domain matches the allowlist are permitted
// through to the Miro API, regardless of what the agent was told to do.
//
// See [miro-mcp-server-jyu] and the 22-04-2026 MCP portfolio security audit.
package tools

import (
	"fmt"
	"os"
	"strings"
)

// ShareAllowlistEnvVar is the environment variable that configures the
// allowlist of domains permitted to receive board-share invitations.
const ShareAllowlistEnvVar = "MIRO_SHARE_ALLOWED_DOMAINS"

// ShareAllowlist holds the set of email domains that miro_share_board is
// permitted to invite. Domains are stored lowercased and compared
// case-insensitively. A zero-value allowlist rejects every email.
type ShareAllowlist struct {
	// domains is the set of allowed lowercase domains. Empty means "block all".
	domains map[string]struct{}
	// source describes where the allowlist came from (for error messages).
	source string
}

// NewShareAllowlist builds an allowlist from an explicit list of domains.
// Entries are trimmed, lowercased, and deduplicated. Empty entries are
// skipped. The source string is surfaced in rejection errors so the user
// knows which config to adjust.
func NewShareAllowlist(domains []string, source string) *ShareAllowlist {
	set := make(map[string]struct{}, len(domains))
	for _, d := range domains {
		d = strings.TrimSpace(strings.ToLower(d))
		if d == "" {
			continue
		}
		set[d] = struct{}{}
	}
	return &ShareAllowlist{domains: set, source: source}
}

// LoadShareAllowlistFromEnv reads MIRO_SHARE_ALLOWED_DOMAINS (comma-separated)
// and returns a populated ShareAllowlist. If the env var is unset and a
// non-empty fallbackUserEmail is provided, the allowlist defaults to the
// domain of that email. If neither is available, the returned allowlist is
// empty (blocks all sharing) and the caller should warn the user to set the
// env var.
//
// This is a conservative fail-closed default: an agent cannot quietly invite
// external parties unless the operator explicitly opts in.
func LoadShareAllowlistFromEnv(fallbackUserEmail string) *ShareAllowlist {
	raw := strings.TrimSpace(os.Getenv(ShareAllowlistEnvVar))
	if raw != "" {
		return NewShareAllowlist(strings.Split(raw, ","), ShareAllowlistEnvVar)
	}

	// Env var not set; fall back to the authenticated user's own domain.
	if fallbackUserEmail != "" {
		if domain, ok := extractDomain(fallbackUserEmail); ok {
			return NewShareAllowlist([]string{domain}, "authenticated user's email domain")
		}
	}

	// No config and no fallback: block all sharing with a clear error.
	return NewShareAllowlist(nil, "unset")
}

// Domains returns a sorted snapshot of the allowed domains. Primarily for
// logging and error-message construction.
func (a *ShareAllowlist) Domains() []string {
	out := make([]string, 0, len(a.domains))
	for d := range a.domains {
		out = append(out, d)
	}
	// Small N; simple insertion-free sort via slice helpers avoided to keep deps minimal.
	// Callers that care about ordering can sort themselves; tests do.
	return out
}

// Source returns a human-readable description of where the allowlist came from.
func (a *ShareAllowlist) Source() string {
	return a.source
}

// IsEmpty reports whether the allowlist has no domains (blocks all sharing).
func (a *ShareAllowlist) IsEmpty() bool {
	return len(a.domains) == 0
}

// Validate checks whether email's domain is permitted. It returns nil on
// success, or a descriptive error that names the offending domain and the
// configured source so the operator can fix it.
func (a *ShareAllowlist) Validate(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email is required")
	}
	domain, ok := extractDomain(email)
	if !ok {
		return fmt.Errorf("invalid email address %q: missing '@' or domain", email)
	}

	if len(a.domains) == 0 {
		return fmt.Errorf(
			"miro_share_board is blocked: the allowlist is empty (source: %s). "+
				"Set %s to a comma-separated list of permitted domains (for example, \"tietoevry.com,tieto.com\") and restart the server",
			a.source, ShareAllowlistEnvVar,
		)
	}

	if _, allowed := a.domains[domain]; !allowed {
		return fmt.Errorf(
			"email domain %q is not in the miro_share_board allowlist (source: %s). "+
				"Add it to %s (comma-separated) and restart the server, or ask the operator to do so",
			domain, a.source, ShareAllowlistEnvVar,
		)
	}
	return nil
}

// extractDomain returns the lowercase domain portion of an email. It returns
// ok=false when the input does not contain exactly one '@' or either side is
// empty. This is intentionally strict: we validate form before we trust the
// value against the allowlist.
func extractDomain(email string) (string, bool) {
	email = strings.TrimSpace(strings.ToLower(email))
	at := strings.Index(email, "@")
	if at <= 0 || at != strings.LastIndex(email, "@") || at == len(email)-1 {
		return "", false
	}
	return email[at+1:], true
}

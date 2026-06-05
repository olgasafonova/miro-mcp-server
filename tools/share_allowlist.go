// Package tools provides MCP tool handlers for the Miro MCP server.
//
// share_allowlist.go implements the recipient allowlist used by the
// miro_share_board tool. Sharing a Miro board grants access to an external
// party; a prompt-injected agent could therefore exfiltrate board content by
// inviting an attacker-controlled address. The allowlist is the server-side
// guardrail: only emails permitted by the allowlist are passed through to the
// Miro API, regardless of what the agent was told to do.
//
// Two layers, with precedence:
//
//   - Exact-email allowlist (MIRO_SHARE_ALLOWED_EMAILS): when set, the
//     recipient must match one of these addresses exactly. This is the
//     identity-binding layer per the HG-3/HG-4 "destination-binding is not
//     identity-binding" extension: a destination filter answers *where* data
//     goes, never *whose account* receives it, so an allowed domain plus an
//     off-team recipient (attacker@allowed-domain.com) otherwise slips through.
//     When this layer is configured it is authoritative and the domain layer
//     is ignored — a strict tightening that never weakens.
//   - Domain allowlist (MIRO_SHARE_ALLOWED_DOMAINS): the fallback when no
//     exact-email allowlist is configured. The recipient's domain must match.
//
// Ideally the recipient would be bound to Miro team membership (MIRO_TEAM_ID),
// but Miro's team/org member-lookup endpoints are Enterprise-plan + Company-
// Admin only (https://developers.miro.com/reference/enterprise-get-team-members);
// the server authenticates with a personal access token, so a membership gate
// would fail-closed against almost every real caller. The exact-email allowlist
// is the degraded identity binding that works on any plan. See bead
// miro-mcp-server-related ag5g (claude-code-config) and rules/code-review-prompts.md
// HG-3/HG-4 extension.
//
// Scope: the allowlist enforces at the MCP handler boundary
// (HandlerRegistry.ShareBoard). Direct callers of miro.Client.ShareBoard
// (library consumers embedding the package) bypass it intentionally; the
// threat model targets prompt-injected agents reaching the client through
// the MCP transport, not human library consumers with their own trust
// assumptions. See miro-mcp-server-032 for the recorded decision.
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

// ShareEmailAllowlistEnvVar is the environment variable that configures the
// exact-email allowlist. When set, it is authoritative: a recipient must match
// one of these addresses exactly, and the domain allowlist is ignored. This is
// the identity-binding layer that defends against the "approved domain"
// exfiltration gap (an attacker-controlled address inside an allowed domain).
const ShareEmailAllowlistEnvVar = "MIRO_SHARE_ALLOWED_EMAILS"

// ShareAllowlist holds the recipients that miro_share_board is permitted to
// invite, across two layers. Entries are stored lowercased and compared
// case-insensitively. A zero-value allowlist rejects every email.
type ShareAllowlist struct {
	// domains is the set of allowed lowercase domains. Empty means "block all"
	// unless an exact-email allowlist is configured.
	domains map[string]struct{}
	// source describes where the domain allowlist came from (for error messages).
	source string
	// emails is the set of allowed exact lowercase addresses. When non-empty it
	// is authoritative and the domain layer is bypassed.
	emails map[string]struct{}
	// emailSource describes where the exact-email allowlist came from.
	emailSource string
}

// normalizeSet trims, lowercases, deduplicates, and drops empty entries from a
// list of allowlist values (domains or emails), returning a set. Shared by the
// domain and exact-email constructors so both layers normalize identically.
func normalizeSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, v := range values {
		v = strings.TrimSpace(strings.ToLower(v))
		if v == "" {
			continue
		}
		set[v] = struct{}{}
	}
	return set
}

// NewShareAllowlist builds an allowlist from an explicit list of domains.
// Entries are trimmed, lowercased, and deduplicated. Empty entries are
// skipped. The source string is surfaced in rejection errors so the user
// knows which config to adjust.
func NewShareAllowlist(domains []string, source string) *ShareAllowlist {
	return &ShareAllowlist{domains: normalizeSet(domains), source: source}
}

// WithEmails attaches an exact-email allowlist to the receiver and returns it
// for chaining. Entries are trimmed, lowercased, and deduplicated; empty
// entries are skipped. When the resulting set is non-empty it becomes the
// authoritative layer in Validate (the domain allowlist is then ignored). The
// source string is surfaced in rejection errors.
func (a *ShareAllowlist) WithEmails(emails []string, source string) *ShareAllowlist {
	a.emails = normalizeSet(emails)
	a.emailSource = source
	return a
}

// LoadShareAllowlistFromEnv reads MIRO_SHARE_ALLOWED_EMAILS and
// MIRO_SHARE_ALLOWED_DOMAINS (both comma-separated) and returns a populated
// ShareAllowlist.
//
//   - If MIRO_SHARE_ALLOWED_EMAILS is set, those exact addresses are the
//     authoritative layer (the domain layer below is then ignored at Validate
//     time). This is the tighter identity binding.
//   - Otherwise the domain layer applies: MIRO_SHARE_ALLOWED_DOMAINS if set,
//     else the domain of fallbackUserEmail (a fail-safe for single-user
//     deployments), else empty.
//
// If no layer is configured and no fallback is available, the returned
// allowlist is empty (blocks all sharing) and the caller should warn the user.
//
// This is a conservative fail-closed default: an agent cannot quietly invite
// external parties unless the operator explicitly opts in.
func LoadShareAllowlistFromEnv(fallbackUserEmail string) *ShareAllowlist {
	allowlist := loadDomainLayerFromEnv(fallbackUserEmail)

	if rawEmails := strings.TrimSpace(os.Getenv(ShareEmailAllowlistEnvVar)); rawEmails != "" {
		allowlist = allowlist.WithEmails(strings.Split(rawEmails, ","), ShareEmailAllowlistEnvVar)
	}

	return allowlist
}

// loadDomainLayerFromEnv builds the domain-layer allowlist (the historic
// behaviour) without consulting the exact-email layer.
func loadDomainLayerFromEnv(fallbackUserEmail string) *ShareAllowlist {
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

// snapshotSet returns an unordered snapshot of a set's keys. Callers that care
// about ordering sort the result themselves (tests do); no sort here keeps deps
// minimal. Shared by Domains and Emails.
func snapshotSet(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}

// Domains returns an unordered snapshot of the allowed domains. Primarily for
// logging and error-message construction.
func (a *ShareAllowlist) Domains() []string {
	return snapshotSet(a.domains)
}

// Source returns a human-readable description of where the domain allowlist
// came from.
func (a *ShareAllowlist) Source() string {
	return a.source
}

// Emails returns an unordered snapshot of the exact-email allowlist. Primarily
// for logging and error-message construction. Empty when no exact-email layer
// is configured.
func (a *ShareAllowlist) Emails() []string {
	return snapshotSet(a.emails)
}

// EmailSource returns a human-readable description of where the exact-email
// allowlist came from. Empty when no exact-email layer is configured.
func (a *ShareAllowlist) EmailSource() string {
	return a.emailSource
}

// IsEmpty reports whether the allowlist has no configured layer (blocks all
// sharing). An exact-email layer alone is enough to make it non-empty.
func (a *ShareAllowlist) IsEmpty() bool {
	return len(a.domains) == 0 && len(a.emails) == 0
}

// emailLayerConfigured reports whether the exact-email layer was explicitly
// configured (WithEmails was called), regardless of whether it ended up with
// any valid entries. emailSource is set only by WithEmails, so a non-empty
// source is the reliable "operator opted into the exact-email layer" signal.
// A configured-but-empty layer is treated as a fail-closed misconfiguration in
// Validate, not a silent fall-through to the weaker domain layer.
func (a *ShareAllowlist) emailLayerConfigured() bool {
	return a.emailSource != ""
}

// Validate checks whether email is permitted to receive a board-share
// invitation. It returns nil on success, or a descriptive error that names the
// offending value and the configured source so the operator can fix it.
//
// Precedence: when an exact-email allowlist is configured it is authoritative —
// the recipient must match one of those addresses exactly, and the domain layer
// is not consulted. Otherwise the domain allowlist applies.
func (a *ShareAllowlist) Validate(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email is required")
	}
	domain, ok := extractDomain(email)
	if !ok {
		return fmt.Errorf("invalid email address %q: missing '@' or domain", email)
	}

	// Exact-email allowlist is authoritative when configured. A domain match
	// must NOT rescue an address that is not in the exact list — that is the
	// "approved domain" exfiltration gap this layer exists to close.
	if a.emailLayerConfigured() {
		// Configured but empty after normalization (e.g. MIRO_SHARE_ALLOWED_EMAILS=","
		// or whitespace-only entries) is operator misconfiguration. Fail closed
		// rather than silently downgrading to the weaker domain layer.
		if len(a.emails) == 0 {
			return fmt.Errorf(
				"miro_share_board is blocked: %s is set (source: %s) but contains no valid "+
					"addresses after normalization. Provide at least one valid email, or unset "+
					"%s to fall back to the domain allowlist, and restart the server",
				ShareEmailAllowlistEnvVar, a.emailSource, ShareEmailAllowlistEnvVar,
			)
		}
		normalized := strings.ToLower(email)
		if _, allowed := a.emails[normalized]; allowed {
			return nil
		}
		return fmt.Errorf(
			"email %q is not in the miro_share_board exact-email allowlist (source: %s). "+
				"Add it to %s (comma-separated) and restart the server, or ask the operator to do so",
			email, a.emailSource, ShareEmailAllowlistEnvVar,
		)
	}

	// Domain allowlist fallback.
	if len(a.domains) == 0 {
		return fmt.Errorf(
			"miro_share_board is blocked: the allowlist is empty (source: %s). "+
				"Set %s to a comma-separated list of permitted domains (for example, \"tietoevry.com,tieto.com\"), "+
				"or %s to an exact-email allowlist, and restart the server",
			a.source, ShareAllowlistEnvVar, ShareEmailAllowlistEnvVar,
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

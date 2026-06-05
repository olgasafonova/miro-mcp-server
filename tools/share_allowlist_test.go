package tools

import (
	"sort"
	"strings"
	"testing"
)

func TestShareAllowlist_ValidateAllowedDomain(t *testing.T) {
	allow := NewShareAllowlist([]string{"tietoevry.com", "tieto.com"}, "test")

	cases := []string{
		"jane@tietoevry.com",
		"JANE@TietoEvry.com", // mixed case
		"  bob@tieto.com  ",  // whitespace
	}
	for _, email := range cases {
		if err := allow.Validate(email); err != nil {
			t.Errorf("Validate(%q) returned error %v; want nil", email, err)
		}
	}
}

func TestShareAllowlist_ValidateRejectsDisallowedDomain(t *testing.T) {
	allow := NewShareAllowlist([]string{"tietoevry.com"}, "test")

	cases := map[string]string{
		"external attacker":  "attacker@evil.example",
		"close lookalike":    "user@tietoevry.co", // typo-lookalike, different TLD
		"subdomain mismatch": "user@mail.tietoevry.com",
		"empty":              "",
		"no at sign":         "not-an-email",
		"multiple at signs":  "a@b@c.com",
		"no user":            "@tietoevry.com",
		"no domain after at": "user@",
	}
	for name, email := range cases {
		err := allow.Validate(email)
		if err == nil {
			t.Errorf("%s: Validate(%q) returned nil; want error", name, email)
			continue
		}
	}
}

func TestShareAllowlist_EmptyAllowlistRejectsEverything(t *testing.T) {
	allow := NewShareAllowlist(nil, "unset")

	err := allow.Validate("jane@tietoevry.com")
	if err == nil {
		t.Fatal("empty allowlist should reject all emails")
	}
	if !strings.Contains(err.Error(), "MIRO_SHARE_ALLOWED_DOMAINS") {
		t.Errorf("error should mention the env var name so the operator knows how to fix; got %q", err)
	}
}

func TestShareAllowlist_RejectionErrorNamesDomainAndSource(t *testing.T) {
	allow := NewShareAllowlist([]string{"tietoevry.com"}, "unit test")
	err := allow.Validate("attacker@evil.example")
	if err == nil {
		t.Fatal("expected rejection")
	}
	msg := err.Error()
	if !strings.Contains(msg, "evil.example") {
		t.Errorf("error should name the offending domain; got %q", msg)
	}
	if !strings.Contains(msg, "unit test") {
		t.Errorf("error should name the allowlist source; got %q", msg)
	}
}

func TestShareAllowlist_NormalizesEntries(t *testing.T) {
	allow := NewShareAllowlist([]string{" TietoEvry.com ", "", "tieto.com", "tieto.com"}, "test")

	domains := allow.Domains()
	sort.Strings(domains)
	want := []string{"tieto.com", "tietoevry.com"}
	if len(domains) != len(want) {
		t.Fatalf("expected %d unique lowercase domains, got %v", len(want), domains)
	}
	for i := range want {
		if domains[i] != want[i] {
			t.Errorf("domains[%d] = %q, want %q", i, domains[i], want[i])
		}
	}
}

func TestLoadShareAllowlistFromEnv_UsesEnvVarWhenSet(t *testing.T) {
	t.Setenv(ShareEmailAllowlistEnvVar, "") // isolate from ambient exact-email env
	t.Setenv(ShareAllowlistEnvVar, "tietoevry.com, tieto.com ,tieto.no")

	allow := LoadShareAllowlistFromEnv("ignored@example.com")

	if allow.Source() != ShareAllowlistEnvVar {
		t.Errorf("source = %q, want %q", allow.Source(), ShareAllowlistEnvVar)
	}
	for _, d := range []string{"tietoevry.com", "tieto.com", "tieto.no"} {
		if err := allow.Validate("user@" + d); err != nil {
			t.Errorf("domain %q should be allowed from env list; got %v", d, err)
		}
	}
	if err := allow.Validate("user@example.com"); err == nil {
		t.Error("fallback email should be ignored when env var is set")
	}
}

func TestLoadShareAllowlistFromEnv_FallsBackToUserEmailDomain(t *testing.T) {
	t.Setenv(ShareEmailAllowlistEnvVar, "") // isolate from ambient exact-email env
	t.Setenv(ShareAllowlistEnvVar, "")

	allow := LoadShareAllowlistFromEnv("olga@tietoevry.com")

	if allow.IsEmpty() {
		t.Fatal("fallback to user email domain should produce a non-empty allowlist")
	}
	if err := allow.Validate("colleague@tietoevry.com"); err != nil {
		t.Errorf("colleague on same domain should be allowed; got %v", err)
	}
	if err := allow.Validate("outsider@example.com"); err == nil {
		t.Error("external domain should be rejected when only user's own domain is allowed")
	}
}

func TestLoadShareAllowlistFromEnv_EmptyWhenNoEnvNoUser(t *testing.T) {
	t.Setenv(ShareEmailAllowlistEnvVar, "") // isolate from ambient exact-email env
	t.Setenv(ShareAllowlistEnvVar, "")

	allow := LoadShareAllowlistFromEnv("")

	if !allow.IsEmpty() {
		t.Errorf("allowlist should be empty when neither env nor user email is available; domains=%v", allow.Domains())
	}
	if err := allow.Validate("anyone@anywhere.example"); err == nil {
		t.Error("empty allowlist must reject all invitations")
	}
}

func TestLoadShareAllowlistFromEnv_InvalidFallbackEmailYieldsEmpty(t *testing.T) {
	t.Setenv(ShareAllowlistEnvVar, "")
	t.Setenv(ShareEmailAllowlistEnvVar, "")

	allow := LoadShareAllowlistFromEnv("not-an-email")

	if !allow.IsEmpty() {
		t.Error("fallback with malformed email should leave the allowlist empty (fail closed)")
	}
}

func TestShareAllowlist_ExactEmailAllowsExactMatch(t *testing.T) {
	allow := NewShareAllowlist(nil, "unset").
		WithEmails([]string{"jane@tietoevry.com"}, "test")

	cases := []string{
		"jane@tietoevry.com",
		"JANE@TietoEvry.com", // mixed case
		"  jane@tietoevry.com  ",
	}
	for _, email := range cases {
		if err := allow.Validate(email); err != nil {
			t.Errorf("Validate(%q) returned error %v; want nil", email, err)
		}
	}
}

// TestShareAllowlist_ExactEmailIsAuthoritative_NoWeakening is the core guarantee
// of the HG-3/HG-4 identity-binding extension: when an exact-email allowlist is
// configured, a recipient whose DOMAIN would pass the domain layer must still be
// rejected if it is not in the exact list. A domain match must never rescue an
// off-list address — that is the "approved domain" exfiltration gap.
func TestShareAllowlist_ExactEmailIsAuthoritative_NoWeakening(t *testing.T) {
	allow := NewShareAllowlist([]string{"tietoevry.com"}, "domain-test").
		WithEmails([]string{"jane@tietoevry.com"}, "email-test")

	if err := allow.Validate("jane@tietoevry.com"); err != nil {
		t.Errorf("exact-listed recipient should pass; got %v", err)
	}

	// attacker@tietoevry.com shares the allowlisted domain but is NOT in the
	// exact list. The domain match must not rescue it.
	err := allow.Validate("attacker@tietoevry.com")
	if err == nil {
		t.Fatal("recipient in allowlisted domain but absent from exact-email list must be rejected")
	}
	if !strings.Contains(err.Error(), ShareEmailAllowlistEnvVar) {
		t.Errorf("rejection should point at the exact-email env var; got %q", err)
	}
}

func TestShareAllowlist_ExactEmailRejectionNamesEmailAndSource(t *testing.T) {
	allow := NewShareAllowlist(nil, "unset").
		WithEmails([]string{"jane@tietoevry.com"}, "unit test")

	err := allow.Validate("attacker@evil.example")
	if err == nil {
		t.Fatal("expected rejection")
	}
	msg := err.Error()
	if !strings.Contains(msg, "attacker@evil.example") {
		t.Errorf("error should name the offending email; got %q", msg)
	}
	if !strings.Contains(msg, "unit test") {
		t.Errorf("error should name the exact-email source; got %q", msg)
	}
}

func TestShareAllowlist_ExactEmailAloneIsNotEmpty(t *testing.T) {
	allow := NewShareAllowlist(nil, "unset").
		WithEmails([]string{"jane@tietoevry.com"}, "test")

	if allow.IsEmpty() {
		t.Error("an exact-email layer alone should make the allowlist non-empty")
	}
	if err := allow.Validate("jane@tietoevry.com"); err != nil {
		t.Errorf("exact-listed recipient should pass even with no domain layer; got %v", err)
	}
}

func TestLoadShareAllowlistFromEnv_ExactEmailAuthoritativeOverDomain(t *testing.T) {
	t.Setenv(ShareAllowlistEnvVar, "tietoevry.com")
	t.Setenv(ShareEmailAllowlistEnvVar, "jane@tietoevry.com")

	allow := LoadShareAllowlistFromEnv("ignored@example.com")

	if len(allow.Emails()) == 0 {
		t.Fatal("exact-email env var should populate the exact-email layer")
	}
	if err := allow.Validate("jane@tietoevry.com"); err != nil {
		t.Errorf("exact-listed recipient should pass; got %v", err)
	}
	// Same domain as MIRO_SHARE_ALLOWED_DOMAINS, but not in the exact list.
	if err := allow.Validate("bob@tietoevry.com"); err == nil {
		t.Error("domain match must not rescue an off-list recipient when exact-email layer is set")
	}
}

// TestShareAllowlist_ConfiguredButEmptyEmailLayerFailsClosed guards the 2xy7
// hardening: when the exact-email layer is configured (WithEmails called) but
// normalizes to zero entries, Validate must fail closed rather than silently
// downgrading to the weaker domain layer. A recipient the domain layer WOULD
// permit must still be rejected.
func TestShareAllowlist_ConfiguredButEmptyEmailLayerFailsClosed(t *testing.T) {
	allow := NewShareAllowlist([]string{"corp.com"}, "domain-test").
		WithEmails([]string{" ", "", "\t"}, "email-test") // all skipped -> empty

	err := allow.Validate("attacker@corp.com")
	if err == nil {
		t.Fatal("configured-but-empty exact-email layer must fail closed, not fall back to the domain allowlist")
	}
	if !strings.Contains(err.Error(), ShareEmailAllowlistEnvVar) {
		t.Errorf("error should name the misconfigured env var; got %q", err)
	}
}

func TestLoadShareAllowlistFromEnv_MalformedEmailValueFailsClosed(t *testing.T) {
	// "," survives the rawEmails != "" guard (TrimSpace leaves the comma) but
	// normalizes to zero entries. With a domain allowlist also set, the old
	// behavior would silently permit any address in corp.com.
	t.Setenv(ShareAllowlistEnvVar, "corp.com")
	t.Setenv(ShareEmailAllowlistEnvVar, " , ")

	allow := LoadShareAllowlistFromEnv("")

	if err := allow.Validate("attacker@corp.com"); err == nil {
		t.Fatal("malformed MIRO_SHARE_ALLOWED_EMAILS must fail closed, not fall back to the domain allowlist")
	}
}

func TestLoadShareAllowlistFromEnv_ExactEmailOnly(t *testing.T) {
	t.Setenv(ShareAllowlistEnvVar, "")
	t.Setenv(ShareEmailAllowlistEnvVar, "jane@tietoevry.com, bob@tieto.com")

	allow := LoadShareAllowlistFromEnv("")

	if allow.IsEmpty() {
		t.Fatal("exact-email env var alone should produce a non-empty allowlist")
	}
	for _, email := range []string{"jane@tietoevry.com", "bob@tieto.com"} {
		if err := allow.Validate(email); err != nil {
			t.Errorf("exact-listed recipient %q should pass; got %v", email, err)
		}
	}
	if err := allow.Validate("carol@tietoevry.com"); err == nil {
		t.Error("recipient absent from the exact-email list must be rejected even with no domain layer")
	}
}

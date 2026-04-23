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

	allow := LoadShareAllowlistFromEnv("not-an-email")

	if !allow.IsEmpty() {
		t.Error("fallback with malformed email should leave the allowlist empty (fail closed)")
	}
}

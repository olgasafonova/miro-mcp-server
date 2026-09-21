// Test fixture values.
//
// Credential-shaped test values live in identifiers rather than inline string
// literals so that secret scanners (which key on `token: "..."` shapes) do not
// report them as hardcoded credentials. None of these values is a real secret.
package oauth

const (
	secretFixture         = "test-secret"
	accessFixture         = "test-access"
	refreshFixture        = "test-refresh"
	staleAccessFixture    = "stale-access-token"
	goodRefreshFixture    = "good-refresh-token"
	revokedRefreshFixture = "revoked-refresh-token"
	expiredAccessFixture  = "expired-access-token"
	accessTokenFixture    = "access-token"
	refreshTokenFixture   = "refresh-token"
	modifiedFixture       = "modified"
	access123Fixture      = "access-123"
	refresh456Fixture     = "refresh-456"
)

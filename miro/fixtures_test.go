// Test fixture values.
//
// Credential-shaped test values live in identifiers rather than inline string
// literals so that secret scanners (which key on `token: "..."` shapes) do not
// report them as hardcoded credentials. None of these values is a real secret.
package miro

const (
	tokenFixture          = "test-token"
	shortTokenFixture     = "too-short"
	refreshedTokenFixture = "refreshed-token"
	// jwtFixture is the canonical jwt.io example token (HS256, payload
	// {"sub":"1234567890","name":"John Doe","iat":1516239022}).
	jwtFixture = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
)

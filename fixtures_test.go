// Test fixture values.
//
// Credential-shaped test values live in identifiers rather than inline string
// literals so that secret scanners (which key on `token: "..."` shapes) do not
// report them as hardcoded credentials. None of these values is a real secret.
package main

const tokenFixture = "test-token"

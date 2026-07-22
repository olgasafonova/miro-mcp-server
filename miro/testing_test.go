package miro

// SetBaseURLForTesting overrides the API base URL for testing.
// It lives in a _test.go file so it is compiled only into test binaries and
// never ships in a production build. This allows tests to point the client at
// a simulated backend.
func (c *Client) SetBaseURLForTesting(url string) {
	c.baseURL = url
}

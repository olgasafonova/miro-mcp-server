package miro

// SetBaseURLForTesting overrides the API base URL for testing.
// This allows external test packages to point the client at a simulated backend.
func (c *Client) SetBaseURLForTesting(url string) {
	c.baseURL = url
}

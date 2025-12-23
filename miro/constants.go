// Package miro provides constants for API limits and configuration.
package miro

import "time"

// =============================================================================
// API Pagination Limits
// =============================================================================

// Default and maximum limits for paginated API endpoints.
const (
	// DefaultBoardLimit is the default number of boards to return.
	DefaultBoardLimit = 20

	// MaxBoardLimit is the maximum boards allowed per request.
	MaxBoardLimit = 50

	// DefaultItemLimit is the default number of items to return.
	DefaultItemLimit = 50

	// MaxItemLimit is the maximum items allowed per request (varies by endpoint).
	MaxItemLimit = 50

	// MaxItemLimitExtended is the max limit for endpoints supporting 100 items.
	MaxItemLimitExtended = 100

	// DefaultSearchLimit is the default search results to return.
	DefaultSearchLimit = 20

	// MaxSearchLimit is the maximum search results allowed.
	MaxSearchLimit = 50

	// DefaultAuditLimit is the default audit log events to return.
	DefaultAuditLimit = 50

	// MaxAuditLimit is the maximum audit log events allowed.
	MaxAuditLimit = 500

	// DefaultListAllMaxItems is the default for ListAllItems max items.
	DefaultListAllMaxItems = 500

	// MaxListAllItems is the absolute maximum for ListAllItems.
	MaxListAllItems = 10000
)

// =============================================================================
// Bulk Operation Limits
// =============================================================================

const (
	// MaxBulkItems is the maximum items for bulk create/update/delete.
	MaxBulkItems = 20

	// MinGroupItems is the minimum items required to create a group.
	MinGroupItems = 2

	// BulkOperationTimeout is the maximum time for bulk operations.
	BulkOperationTimeout = 30 * time.Second
)

// =============================================================================
// HTTP Server Configuration
// =============================================================================

// HTTP server timeout configuration for the MCP server.
const (
	// HTTPReadTimeout is the maximum duration for reading request.
	HTTPReadTimeout = 30 * time.Second

	// HTTPWriteTimeout is the maximum duration for writing response.
	HTTPWriteTimeout = 60 * time.Second

	// HTTPIdleTimeout is the maximum idle time for keep-alive connections.
	HTTPIdleTimeout = 120 * time.Second
)

// =============================================================================
// Cache Configuration
// =============================================================================

// Cache TTL configuration for different data types.
const (
	// BoardCacheTTL is how long board data is cached.
	BoardCacheTTL = 2 * time.Minute

	// ItemCacheTTL is how long item data is cached.
	ItemCacheTTL = 1 * time.Minute

	// TagCacheTTL is how long tag data is cached.
	TagCacheTTL = 2 * time.Minute

	// CacheMaxEntries is the maximum entries in the cache (0 = unlimited).
	CacheMaxEntries = 1000
)

// =============================================================================
// Rate Limiting Configuration
// =============================================================================

const (
	// RateLimitMaxDelay is the maximum delay for rate limit backoff.
	RateLimitMaxDelay = 2 * time.Second

	// IdleConnTimeout is the idle connection timeout for HTTP transport.
	IdleConnTimeout = 90 * time.Second
)

// =============================================================================
// Circuit Breaker Configuration
// =============================================================================

const (
	// CircuitBreakerTimeout is how long to wait before testing after open.
	CircuitBreakerTimeout = 30 * time.Second

	// CircuitBreakerFailureThreshold is failures before circuit opens.
	CircuitBreakerFailureThreshold = 5

	// CircuitBreakerSuccessThreshold is successes to close from half-open.
	CircuitBreakerSuccessThreshold = 2
)

// =============================================================================
// OAuth Configuration
// =============================================================================

const (
	// OAuthServerReadTimeout is the OAuth callback server read timeout.
	OAuthServerReadTimeout = 10 * time.Second

	// OAuthServerWriteTimeout is the OAuth callback server write timeout.
	OAuthServerWriteTimeout = 10 * time.Second

	// OAuthHTTPTimeout is the HTTP client timeout for OAuth requests.
	OAuthHTTPTimeout = 30 * time.Second

	// TokenRefreshBuffer is how early to refresh tokens before expiry.
	TokenRefreshBuffer = 5 * time.Minute
)

// =============================================================================
// Diagram Configuration
// =============================================================================

const (
	// MaxDiagramNodes is the maximum nodes allowed in a diagram.
	MaxDiagramNodes = 100

	// DefaultNodeWidth is the default width for diagram nodes.
	DefaultNodeWidth = 180

	// DefaultNodeHeight is the default height for diagram nodes.
	DefaultNodeHeight = 80

	// DefaultNodeSpacingX is the horizontal spacing between nodes.
	DefaultNodeSpacingX = 250

	// DefaultNodeSpacingY is the vertical spacing between nodes.
	DefaultNodeSpacingY = 150
)

// =============================================================================
// Audit Log Configuration
// =============================================================================

const (
	// DefaultAuditMaxSizeBytes is the default max size for audit log file (50MB).
	DefaultAuditMaxSizeBytes = 50 * 1024 * 1024

	// DefaultMemoryRingSize is the default size for in-memory audit buffer.
	DefaultMemoryRingSize = 1000
)

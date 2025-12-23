# Error Handling Guide

This document describes all error types, codes, and troubleshooting guidance for the Miro MCP Server.

## Error Types

### APIError

Returned when the Miro API returns an error response.

```go
type APIError struct {
    StatusCode int    // HTTP status code (e.g., 401, 404, 429)
    Code       string // Error code from Miro (e.g., "unauthorized")
    Message    string // Human-readable error message
    Type       string // Error type from Miro
    Context    string // Additional context
    RetryAfter int    // Seconds to wait before retry (rate limits)
}
```

**Example:**
```
Miro API error [401 unauthorized]: Invalid access token
```

### ValidationError

Returned when input validation fails before making an API call.

```go
type ValidationError struct {
    Field   string // Field that failed validation
    Message string // Validation error message
}
```

**Example:**
```
validation error: board_id - is required
```

### DiagramError

Returned when Mermaid diagram parsing or layout fails.

```go
type DiagramError struct {
    Code       string // Error code (e.g., "NO_NODES")
    Message    string // User-friendly error message
    Suggestion string // Actionable fix suggestion
    Line       int    // Line number where error occurred
    Input      string // Relevant input that caused error
}
```

**Example:**
```
no nodes found in diagram. Add node definitions like 'A[Label]' or edges like 'A --> B'
```

### AuthError

Returned for OAuth authentication errors.

```go
type AuthError struct {
    Code        string // OAuth error code
    Description string // Error description
}
```

**Example:**
```
invalid_grant: The authorization code has expired
```

---

## HTTP Status Codes

| Status | Meaning | Typical Cause |
|--------|---------|---------------|
| 400 | Bad Request | Invalid parameters, malformed request |
| 401 | Unauthorized | Invalid or expired access token |
| 403 | Forbidden | Missing required scopes or permissions |
| 404 | Not Found | Board, item, or resource doesn't exist |
| 409 | Conflict | Resource already exists |
| 413 | Payload Too Large | Request body exceeds size limit |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Miro API server error |
| 503 | Service Unavailable | Miro API temporarily unavailable |

---

## Diagram Error Codes

| Code | Meaning | Common Cause |
|------|---------|--------------|
| `NO_NODES` | No nodes found | Diagram has no node definitions |
| `INVALID_SYNTAX` | Syntax error | Malformed Mermaid syntax |
| `MISSING_HEADER` | Missing header | Diagram doesn't start with `flowchart` or `sequenceDiagram` |
| `EMPTY_DIAGRAM` | Empty input | Diagram string is empty |
| `INVALID_SHAPE` | Invalid shape | Unrecognized node shape syntax |
| `CIRCULAR_REFERENCE` | Circular reference | Nodes reference each other in loop |
| `TOO_MANY_NODES` | Too many nodes | Diagram exceeds node limit |
| `INVALID_EDGE` | Invalid edge | Edge references undefined node |
| `UNKNOWN_DIAGRAM_TYPE` | Unknown type | Unsupported diagram type |

---

## Troubleshooting Guide

### Authentication Errors (401)

**Symptom:** `Miro API error [401 unauthorized]: Invalid access token`

**Solutions:**
1. Verify `MIRO_ACCESS_TOKEN` is set:
   ```bash
   echo $MIRO_ACCESS_TOKEN
   ```

2. Check token validity at Miro:
   - Go to https://miro.com/app/settings/user-profile/apps
   - Revoke and regenerate your token

3. If using OAuth, re-authenticate:
   ```bash
   MIRO_CLIENT_ID=xxx MIRO_CLIENT_SECRET=yyy ./miro-mcp-server auth login
   ```

### Permission Errors (403)

**Symptom:** `Miro API error [403 forbidden]: Access denied`

**Solutions:**
1. Check that your token has required scopes:
   - `boards:read` - Required for reading boards
   - `boards:write` - Required for creating/modifying items
   - `team:read` - Required for team operations

2. Verify you have access to the specific board:
   - Open the board in Miro web app
   - Check if you're a member or owner

3. For Enterprise features (export jobs), verify:
   - Your Miro plan includes Enterprise features
   - You have the required organization role

### Not Found Errors (404)

**Symptom:** `Miro API error [404 not_found]: Resource not found`

**Solutions:**
1. Verify the board/item ID is correct:
   ```bash
   # Use find_board to get correct ID
   miro_find_board(name="My Board")
   ```

2. Check if the resource was deleted:
   - Board may have been deleted or archived
   - Item may have been removed from board

3. Check for typos in IDs:
   - Board IDs look like: `uXjVOXQCe5c=`
   - Item IDs look like: `3458764653228771705`

### Rate Limit Errors (429)

**Symptom:** `Miro API error [429]: Rate limit exceeded`

**Solutions:**
1. Wait and retry:
   - Check `Retry-After` header for wait time
   - Server automatically retries with exponential backoff

2. Reduce request frequency:
   - Use bulk operations instead of individual calls
   - Cache board data when possible

3. Check your rate limit status:
   - Standard tier: ~1000 requests per minute
   - Enterprise tier: Higher limits available

### Diagram Parsing Errors

**Symptom:** `no nodes found in diagram`

**Solutions:**
1. Ensure diagram starts with valid header:
   ```mermaid
   flowchart TB
       A[Start] --> B[End]
   ```

2. Check arrow syntax:
   - Flowcharts use `-->` not `->`
   - Sequence diagrams use `->>` for sync messages

3. Define nodes before using them in edges:
   ```mermaid
   flowchart TB
       A[Node A]
       B[Node B]
       A --> B
   ```

**Symptom:** `unrecognized node shape`

**Solutions:**
Valid shapes are:
- `[text]` - Rectangle
- `(text)` - Rounded rectangle
- `{text}` - Diamond
- `((text))` - Circle
- `{{text}}` - Hexagon

### Connection Errors

**Symptom:** `connection refused` or `timeout`

**Solutions:**
1. Check network connectivity:
   ```bash
   curl -I https://api.miro.com
   ```

2. Check for proxy configuration:
   - Set `HTTP_PROXY` / `HTTPS_PROXY` if behind proxy

3. Verify Miro API status:
   - Check https://status.miro.com

---

## Programmatic Error Handling

### Checking Error Types

```go
import "github.com/olgasafonova/miro-mcp-server/miro"

result, err := client.CreateSticky(ctx, args)
if err != nil {
    // Check specific error types
    if miro.IsRateLimitError(err) {
        retryAfter := miro.GetRetryAfter(err)
        time.Sleep(retryAfter)
        // retry...
    }

    if miro.IsAuthError(err) {
        // Refresh token or re-authenticate
    }

    if miro.IsNotFoundError(err) {
        // Resource doesn't exist
    }

    if miro.IsValidationError(err) {
        // Check input parameters
    }
}
```

### Extracting Error Details

```go
var apiErr *miro.APIError
if errors.As(err, &apiErr) {
    fmt.Printf("Status: %d\n", apiErr.StatusCode)
    fmt.Printf("Code: %s\n", apiErr.Code)
    fmt.Printf("Message: %s\n", apiErr.Message)
    fmt.Printf("Suggestion: %s\n", apiErr.Suggestion())
}
```

### Handling Diagram Errors

```go
import "github.com/olgasafonova/miro-mcp-server/miro/diagrams"

result, err := client.GenerateDiagram(ctx, args)
if err != nil {
    var diagramErr *diagrams.DiagramError
    if errors.As(err, &diagramErr) {
        fmt.Printf("Code: %s\n", diagramErr.Code)
        fmt.Printf("Line: %d\n", diagramErr.Line)
        fmt.Printf("Suggestion: %s\n", diagramErr.Suggestion)
    }
}
```

---

## Error Messages Reference

### API Errors by Status Code

| Status | Code | Message | Suggestion |
|--------|------|---------|------------|
| 401 | `unauthorized` | Invalid access token | Check MIRO_ACCESS_TOKEN is set and valid |
| 403 | `forbidden` | Access denied | Check token scopes and board permissions |
| 404 | `not_found` | Resource not found | Verify the ID exists and hasn't been deleted |
| 409 | `conflict` | Resource already exists | The resource may already exist |
| 413 | - | Request payload too large | Reduce content size or split into multiple requests |
| 429 | `rate_limited` | Rate limit exceeded | Wait and retry, or reduce request frequency |
| 500 | `internal_error` | Internal server error | Miro API issue - try again later |
| 503 | - | Service unavailable | Miro API temporarily down - try again later |

### Validation Errors

| Field | Message | Fix |
|-------|---------|-----|
| `board_id` | is required | Provide a valid board ID |
| `content` | is required | Provide content for the item |
| `content` | exceeds maximum length | Reduce content length |
| `shape` | invalid shape type | Use valid shape: rectangle, circle, etc. |
| `color` | invalid color | Use valid color: yellow, blue, red, etc. |
| `item_id` | is required | Provide a valid item ID |
| `parent_id` | invalid frame ID | Verify the frame exists on the board |

---

## Circuit Breaker

The client includes a circuit breaker for resilience against Miro API outages.

**States:**
- **Closed** - Normal operation, requests pass through
- **Open** - Too many failures, requests fail immediately
- **Half-Open** - Testing if service recovered

**Thresholds:**
- Opens after 5 consecutive failures
- Stays open for 30 seconds
- Transitions to half-open to test recovery

**Behavior when open:**
```
circuit breaker is open, request blocked
```

Wait 30 seconds or until service recovers.

---

## Getting Help

1. **Check this guide** for common issues
2. **Review the logs** for detailed error messages
3. **Verify your token** at https://miro.com/app/settings/user-profile/apps
4. **Check Miro status** at https://status.miro.com
5. **Open an issue** at https://github.com/olgasafonova/miro-mcp-server/issues

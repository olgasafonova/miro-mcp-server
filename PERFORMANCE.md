# Performance Guide

Optimization tips and benchmarks for Miro MCP Server.

---

## Benchmarks

### Binary Comparison

| Metric | Miro MCP Server (Go) | TypeScript Servers |
|--------|---------------------|-------------------|
| Binary size | 14MB | 100MB+ (with node_modules) |
| Cold start | ~50ms | 500ms-2s |
| Memory (idle) | ~10MB | ~50MB |
| Memory (active) | ~30MB | ~100MB |

### Response Times (Average)

| Operation | Time | Notes |
|-----------|------|-------|
| `miro_list_boards` | 150-300ms | First call; cached calls <5ms |
| `miro_create_sticky` | 200-400ms | Single item |
| `miro_bulk_create` (10 items) | 300-500ms | Parallel creation |
| `miro_generate_diagram` (10 nodes) | 500-800ms | Parsing + layout + creation |
| `miro_get_board_summary` | 400-600ms | Parallel board + items fetch |

---

## Optimization Features

### 1. Response Caching

Reduces redundant API calls with a 2-minute TTL cache.

**Cached Operations:**
- `miro_list_boards`
- `miro_get_board`
- `miro_list_items`

**Cache Behavior:**
- Memory-based (cleared on restart)
- Automatic invalidation after writes
- ~10x speedup for repeated reads

**Example Impact:**

```
First call:  miro_list_boards → 250ms (API call)
Second call: miro_list_boards → 2ms (cache hit)
```

---

### 2. Rate Limiting

Protects against Miro API limits with intelligent throttling.

**Configuration:**
- Max concurrent requests: 5
- Automatic retry on 429 errors
- Exponential backoff: 1s → 2s → 4s

**Miro API Limits:**
- 100,000 credits per minute per user
- Most operations: 1-10 credits

The server's conservative limits prevent hitting these thresholds.

---

### 3. Connection Pooling

Optimized HTTP client reuses connections.

| Setting | Value |
|---------|-------|
| Max idle connections | 100 |
| Max per host | 10 |
| Idle timeout | 90s |
| TLS handshake timeout | 10s |

---

### 4. Circuit Breaker

Prevents cascading failures when Miro API has issues.

**States:**
1. **Closed** (normal) - Requests pass through
2. **Open** (failing) - Requests fail fast
3. **Half-open** - Testing if recovered

**Thresholds:**
- Opens after: 5 consecutive failures
- Recovery timeout: 30 seconds
- Test requests: 1 at a time

---

### 5. Bulk Operations

Create multiple items efficiently.

**Single Items (Slow):**
```
Create sticky 1 → 300ms
Create sticky 2 → 300ms
Create sticky 3 → 300ms
Total: 900ms
```

**Bulk Create (Fast):**
```
Create 3 stickies → 350ms (parallel)
Total: 350ms
```

**Tip:** Use `miro_bulk_create` or `miro_create_sticky_grid` for multiple items.

---

## Optimization Tips

### 1. Use Composite Tools

Instead of multiple calls, use pre-built composite operations:

| Instead of... | Use... | Speedup |
|---------------|--------|---------|
| `get_board` + `list_items` | `miro_get_board_summary` | 2x |
| Multiple `create_sticky` | `miro_bulk_create` | 3-5x |
| Multiple `create_sticky` in rows | `miro_create_sticky_grid` | 3-5x |

---

### 2. Specify Limits

Always add `limit` to list operations:

```json
// Slow: fetches 50 items by default
{"tool": "miro_list_items", "board_id": "abc123"}

// Fast: only fetches what you need
{"tool": "miro_list_items", "board_id": "abc123", "limit": 10}
```

---

### 3. Filter by Type

Reduce response size with type filtering:

```json
// Returns all items
{"tool": "miro_list_items", "board_id": "abc123"}

// Returns only stickies
{"tool": "miro_list_items", "board_id": "abc123", "type": "sticky_note"}
```

---

### 4. Use Board Name Resolution

Avoid extra calls by using `miro_find_board`:

```json
// Before: Two calls
1. miro_list_boards → Find "Design Sprint" → Get ID
2. miro_create_sticky → Use ID

// After: One call
miro_find_board → Returns ID directly
```

---

### 5. Leverage Frames

Group related items with frames for faster operations:

- `miro_get_frame_items` returns only items in that frame
- `parent_id` parameter places new items directly in frames
- Frame-based organization reduces search scope

---

## Memory Management

### Low Memory Mode

For constrained environments:

1. Reduce concurrent requests (code change needed)
2. Disable caching (code change needed)
3. Use Docker with memory limits:

```bash
docker run --memory=50m ghcr.io/olgasafonova/miro-mcp-server
```

### Memory Usage Patterns

| Scenario | Memory |
|----------|--------|
| Idle | ~10MB |
| Light use (10 calls/min) | ~15MB |
| Heavy use (100 calls/min) | ~30MB |
| Large board (1000+ items) | ~50MB |

---

## Latency Reduction

### Network Optimization

1. **Run close to Miro API** - Miro servers are in AWS us-east-1
2. **Use HTTP/2** - Enabled by default in Go 1.21+
3. **Persistent connections** - Handled automatically

### AI Tool Configuration

Reduce round-trips by:
- Batching related operations
- Using parallel tool calls when possible
- Caching board IDs between sessions

---

## Monitoring

### Audit Log Analysis

Use `miro_get_audit_log` to analyze performance:

```json
{
  "tool": "miro_get_audit_log",
  "since": "2025-12-22T00:00:00Z",
  "limit": 100
}
```

**Fields available:**
- `duration_ms` - Operation time
- `success` - Pass/fail status
- `tool` - Which tool was called
- `board_id` - Which board was accessed

### Slow Operation Detection

Look for patterns in audit logs:
- Operations >1s may indicate API issues
- Repeated failures may trigger circuit breaker
- High call volumes may hit rate limits

---

## Troubleshooting Performance

### Slow First Request

**Cause:** Cold start + token validation
**Solution:** Normal; subsequent requests are faster

### Slow List Operations

**Cause:** Too many items returned
**Solution:** Add `limit` parameter

### Rate Limit Errors

**Cause:** Too many concurrent operations
**Solution:** The server handles this automatically; wait for retry

### Memory Growth

**Cause:** Large board operations
**Solution:** Normal; memory is released after operations complete

---

## Comparison: Go vs TypeScript

### Why Go is Faster

1. **Compiled binary** - No interpretation overhead
2. **Static typing** - No runtime type checks
3. **Goroutines** - Lightweight concurrent operations
4. **Native HTTP/2** - Optimized networking
5. **No GC pauses** - Low-latency garbage collection

### Real-World Impact

| Scenario | Go Server | TypeScript Server |
|----------|-----------|-------------------|
| Start 10 servers | ~500ms | ~15 seconds |
| Handle 100 req/s | Stable ~30MB | Growing ~200MB |
| Cold start | 50ms | 2 seconds |

---

## Best Practices Summary

1. **Use bulk operations** for multiple items
2. **Add limits** to all list calls
3. **Filter by type** when possible
4. **Use frames** for organization
5. **Let caching work** - don't duplicate calls
6. **Use composite tools** like `get_board_summary`
7. **Monitor with audit logs** for performance insights

# Configuration Reference

Complete configuration guide for Miro MCP Server.

---

## Environment Variables

### Required

| Variable | Description | Example |
|----------|-------------|---------|
| `MIRO_ACCESS_TOKEN` | Your Miro OAuth access token | `eyJtaXJvLm9yaWdpbiI6...` |

### Optional

| Variable | Default | Description |
|----------|---------|-------------|
| `MIRO_TEAM_ID` | - | Filter boards to a specific team |
| `MIRO_TIMEOUT` | `30s` | Request timeout (e.g., `10s`, `1m`) |
| `MIRO_LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `MIRO_AUDIT_FILE` | - | Path to audit log file (enables file logging) |
| `MIRO_AUDIT_MAX_SIZE` | `10MB` | Max size before log rotation |

---

## Transport Modes

### stdio Mode (Default)

Standard input/output for MCP communication. Use with Claude Code, Cursor, VS Code.

```bash
MIRO_ACCESS_TOKEN=your-token miro-mcp-server
```

### HTTP Mode

HTTP server for web-based MCP clients or debugging.

```bash
MIRO_ACCESS_TOKEN=your-token miro-mcp-server -http :8080
```

Endpoints:
- `POST /` - MCP JSON-RPC endpoint
- `GET /health` - Health check (returns `{"status":"ok"}`)

---

## Authentication Methods

### 1. Static Token (Simplest)

Get a token from [miro.com/app/settings/user-profile/apps](https://miro.com/app/settings/user-profile/apps):

1. Create new app with `boards:read` and `boards:write` permissions
2. Install to your team
3. Copy the access token

```bash
export MIRO_ACCESS_TOKEN="eyJtaXJvLm9yaWdpbiI6..."
miro-mcp-server
```

### 2. OAuth 2.1 with PKCE (Recommended for Production)

For multi-user scenarios or when tokens need automatic refresh.

**Setup:**

1. Create a Miro app at [developers.miro.com](https://developers.miro.com)
2. Set redirect URI to `http://localhost:9876/callback`
3. Note your Client ID and Client Secret

**Login:**

```bash
export MIRO_CLIENT_ID="your-client-id"
export MIRO_CLIENT_SECRET="your-client-secret"
miro-mcp-server auth login
```

This opens a browser for authentication. Tokens are stored in `~/.miro/tokens.json`.

**Commands:**

```bash
# Check authentication status
miro-mcp-server auth status

# Clear stored tokens
miro-mcp-server auth logout
```

---

## Rate Limiting

The server implements automatic rate limiting:

| Setting | Value | Description |
|---------|-------|-------------|
| Max concurrent | 5 | Simultaneous API requests |
| Backoff base | 1s | Initial retry delay |
| Max retries | 3 | Retry attempts for rate-limited requests |

Rate limits are handled automatically. The server:
1. Respects Miro's rate limit headers
2. Uses exponential backoff on 429 responses
3. Queues requests when concurrency limit reached

---

## Caching

Built-in response caching reduces API calls:

| Data Type | TTL | Description |
|-----------|-----|-------------|
| Board lists | 2 min | `miro_list_boards` results |
| Board details | 2 min | `miro_get_board` results |
| Item lists | 1 min | `miro_list_items` results |

Cache is memory-based and cleared on restart.

---

## AI Tool Configuration

### Claude Code

```bash
# Add the server
claude mcp add miro -e MIRO_ACCESS_TOKEN=your-token -- miro-mcp-server

# Verify
claude mcp list

# Remove if needed
claude mcp remove miro
```

### Claude Desktop

**macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "miro": {
      "command": "/usr/local/bin/miro-mcp-server",
      "env": {
        "MIRO_ACCESS_TOKEN": "your-token"
      }
    }
  }
}
```

**Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "miro": {
      "command": "C:\\Users\\YOU\\AppData\\Local\\miro-mcp-server.exe",
      "env": {
        "MIRO_ACCESS_TOKEN": "your-token"
      }
    }
  }
}
```

### Cursor

Settings → MCP → Edit in settings.json:

```json
{
  "mcpServers": {
    "miro": {
      "command": "/usr/local/bin/miro-mcp-server",
      "env": {
        "MIRO_ACCESS_TOKEN": "your-token"
      }
    }
  }
}
```

### VS Code + GitHub Copilot

Command Palette → "MCP: Edit User Configuration":

```json
{
  "servers": {
    "miro": {
      "type": "stdio",
      "command": "/usr/local/bin/miro-mcp-server",
      "env": {
        "MIRO_ACCESS_TOKEN": "your-token"
      }
    }
  }
}
```

### Windsurf

```json
{
  "mcpServers": {
    "miro": {
      "command": "/usr/local/bin/miro-mcp-server",
      "env": {
        "MIRO_ACCESS_TOKEN": "your-token"
      }
    }
  }
}
```

### Gemini CLI

`~/.gemini/settings.json`:

```json
{
  "mcpServers": {
    "miro": {
      "command": "/usr/local/bin/miro-mcp-server",
      "env": {
        "MIRO_ACCESS_TOKEN": "your-token"
      }
    }
  }
}
```

### Amazon Q

`~/.aws/amazonq/mcp.json`:

```json
{
  "mcpServers": {
    "miro": {
      "command": "/usr/local/bin/miro-mcp-server",
      "env": {
        "MIRO_ACCESS_TOKEN": "your-token"
      }
    }
  }
}
```

### Kiro

`.kiro/mcp.json` (project) or `~/.kiro/mcp.json` (global):

```json
{
  "mcpServers": {
    "miro": {
      "command": "/usr/local/bin/miro-mcp-server",
      "env": {
        "MIRO_ACCESS_TOKEN": "your-token"
      }
    }
  }
}
```

### OpenAI Codex CLI

`~/.codex/config.toml`:

```toml
[mcp.servers.miro]
command = "/usr/local/bin/miro-mcp-server"

[mcp.servers.miro.env]
MIRO_ACCESS_TOKEN = "your-token"
```

---

## Docker Configuration

### Basic Usage

```bash
docker run -e MIRO_ACCESS_TOKEN=your-token ghcr.io/olgasafonova/miro-mcp-server
```

### Docker Compose

```yaml
version: '3.8'
services:
  miro-mcp:
    image: ghcr.io/olgasafonova/miro-mcp-server:latest
    environment:
      - MIRO_ACCESS_TOKEN=${MIRO_ACCESS_TOKEN}
      - MIRO_TEAM_ID=${MIRO_TEAM_ID:-}
    restart: unless-stopped
```

### HTTP Mode with Docker

```bash
docker run -p 8080:8080 -e MIRO_ACCESS_TOKEN=your-token \
  ghcr.io/olgasafonova/miro-mcp-server -http :8080
```

---

## Audit Logging

Track all MCP tool executions for debugging or compliance.

### Enable File Logging

```bash
export MIRO_AUDIT_FILE="/var/log/miro-mcp/audit.jsonl"
miro-mcp-server
```

### Log Format

JSON Lines format, one event per line:

```json
{"timestamp":"2025-12-22T10:30:00Z","tool":"miro_create_sticky","action":"create","board_id":"abc123","duration_ms":245,"success":true}
```

### Query Audit Log

Use the `miro_get_audit_log` tool:

```
"Show me operations from the last hour"
"What did we create today?"
```

Parameters:
- `since` - ISO 8601 timestamp
- `until` - ISO 8601 timestamp
- `tool` - Filter by tool name
- `board_id` - Filter by board
- `action` - create, read, update, delete
- `success` - true/false
- `limit` - Max events (default 50)

---

## Connection Pooling

Built-in HTTP client optimization:

| Setting | Value |
|---------|-------|
| Max idle connections | 100 |
| Max connections per host | 10 |
| Idle connection timeout | 90s |
| TLS handshake timeout | 10s |

---

## Circuit Breaker

Protects against cascading failures:

| Setting | Value |
|---------|-------|
| Failure threshold | 5 consecutive failures |
| Recovery timeout | 30 seconds |
| Half-open requests | 1 |

When a Miro API endpoint fails repeatedly, the circuit breaker:
1. Opens (blocks requests)
2. Waits 30 seconds
3. Allows one test request
4. Closes if successful, stays open if not

---

## Troubleshooting

### Token Validation Failed

```
Invalid MIRO_ACCESS_TOKEN: token validation failed: 401 Unauthorized
```

**Fix:** Regenerate your token at [miro.com/app/settings/user-profile/apps](https://miro.com/app/settings/user-profile/apps)

### Permission Denied on macOS

```
zsh: permission denied: miro-mcp-server
```

**Fix:** `chmod +x /usr/local/bin/miro-mcp-server`

### Rate Limited

```
429 Too Many Requests
```

The server handles this automatically. If persistent:
- Check if other apps use your Miro API quota
- Reduce concurrent operations
- Wait 60 seconds before retrying

### Board Not Found

```
no board found matching 'Design Sprint'
```

**Fix:** Use `miro_list_boards` first, then use the exact board ID or name.

### JSON Config Syntax Error

Check for:
- Trailing commas
- Missing quotes
- Wrong bracket types

Use a JSON validator: [jsonlint.com](https://jsonlint.com)

---

## Security Best Practices

1. **Never commit tokens** - Use environment variables or secrets manager
2. **Rotate tokens regularly** - Regenerate every 90 days
3. **Use minimal permissions** - Only request `boards:read` and `boards:write`
4. **Audit regularly** - Review audit logs for unexpected activity
5. **Restrict team access** - Use `MIRO_TEAM_ID` to limit scope

---

## Performance Tips

1. **Use bulk operations** - `miro_bulk_create` instead of multiple `miro_create_sticky`
2. **Use `miro_get_board_summary`** - One call instead of board + items
3. **Specify limits** - Add `limit` parameter to list operations
4. **Filter by type** - Use `type` parameter in `miro_list_items`
5. **Use frames** - Organize content with `parent_id` for faster queries

---

## Version Compatibility

| Miro MCP Server | Miro API | MCP Protocol |
|-----------------|----------|--------------|
| v1.7.x | v2 | 2024-11-05 |
| v1.6.x | v2 | 2024-11-05 |
| v1.5.x | v2 | 2024-11-05 |

The server uses Miro API v2. Mindmap operations use the v2-experimental endpoint.

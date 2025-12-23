# Setup Guide

Get Miro MCP Server running in **under 5 minutes**.

---

## Step 1: Download

### macOS (Apple Silicon - M1/M2/M3/M4)

```bash
curl -L -o miro-mcp-server https://github.com/olgasafonova/miro-mcp-server/releases/latest/download/miro-mcp-server-darwin-arm64
chmod +x miro-mcp-server
sudo mv miro-mcp-server /usr/local/bin/
```

### macOS (Intel)

```bash
curl -L -o miro-mcp-server https://github.com/olgasafonova/miro-mcp-server/releases/latest/download/miro-mcp-server-darwin-amd64
chmod +x miro-mcp-server
sudo mv miro-mcp-server /usr/local/bin/
```

### Linux

```bash
curl -L -o miro-mcp-server https://github.com/olgasafonova/miro-mcp-server/releases/latest/download/miro-mcp-server-linux-amd64
chmod +x miro-mcp-server
sudo mv miro-mcp-server /usr/local/bin/
```

### Windows (PowerShell as Administrator)

```powershell
Invoke-WebRequest -Uri "https://github.com/olgasafonova/miro-mcp-server/releases/latest/download/miro-mcp-server-windows-amd64.exe" -OutFile "$env:LOCALAPPDATA\miro-mcp-server.exe"
```

### Build from Source (any platform)

Requires Go 1.21+

```bash
go install github.com/olgasafonova/miro-mcp-server@latest
```

---

## Step 2: Get Your Miro Token

1. Go to **[miro.com/app/settings/user-profile/apps](https://miro.com/app/settings/user-profile/apps)**
2. Click **"Create new app"** (or use existing)
3. Give it a name like "MCP Server"
4. Under **Permissions**, enable:
   - `boards:read`
   - `boards:write`
5. Click **"Install app and get OAuth token"**
6. Select your team
7. **Copy the access token** (starts with `eyJ...`)

> **Keep this token secret.** It grants access to your Miro boards.

---

## Step 3: Configure Your AI Tool

Pick your tool below and follow the instructions.

---

## Claude Code

**One command:**

```bash
claude mcp add miro -- miro-mcp-server
```

Then set the token:

```bash
claude mcp add miro -e MIRO_ACCESS_TOKEN=your-token-here -- miro-mcp-server
```

**Verify it works:**

```bash
claude mcp list
```

You should see `miro` in the list.

---

## Claude Desktop

### macOS

Edit `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "miro": {
      "command": "/usr/local/bin/miro-mcp-server",
      "env": {
        "MIRO_ACCESS_TOKEN": "your-token-here"
      }
    }
  }
}
```

### Windows

Edit `%APPDATA%\Claude\claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "miro": {
      "command": "C:\\Users\\YOUR_USERNAME\\AppData\\Local\\miro-mcp-server.exe",
      "env": {
        "MIRO_ACCESS_TOKEN": "your-token-here"
      }
    }
  }
}
```

### Linux

Edit `~/.config/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "miro": {
      "command": "/usr/local/bin/miro-mcp-server",
      "env": {
        "MIRO_ACCESS_TOKEN": "your-token-here"
      }
    }
  }
}
```

**Restart Claude Desktop** after saving.

---

## Cursor

1. Open **Cursor Settings** (Cmd+, on Mac, Ctrl+, on Windows/Linux)
2. Search for **"MCP"**
3. Click **"Edit in settings.json"** under MCP Servers
4. Add:

```json
{
  "mcpServers": {
    "miro": {
      "command": "/usr/local/bin/miro-mcp-server",
      "env": {
        "MIRO_ACCESS_TOKEN": "your-token-here"
      }
    }
  }
}
```

**Windows path:** `C:\\Users\\YOUR_USERNAME\\AppData\\Local\\miro-mcp-server.exe`

**Restart Cursor** after saving.

---

## VS Code + GitHub Copilot

1. Open Command Palette (Cmd+Shift+P / Ctrl+Shift+P)
2. Type **"MCP: Edit User Configuration"**
3. Add:

```json
{
  "servers": {
    "miro": {
      "type": "stdio",
      "command": "/usr/local/bin/miro-mcp-server",
      "env": {
        "MIRO_ACCESS_TOKEN": "your-token-here"
      }
    }
  }
}
```

**Windows path:** `C:\\Users\\YOUR_USERNAME\\AppData\\Local\\miro-mcp-server.exe`

---

## Windsurf

Edit your Windsurf MCP config (similar to Cursor):

```json
{
  "mcpServers": {
    "miro": {
      "command": "/usr/local/bin/miro-mcp-server",
      "env": {
        "MIRO_ACCESS_TOKEN": "your-token-here"
      }
    }
  }
}
```

---

## Replit

1. In your Repl, go to **Tools > MCP Servers**
2. Click **"Add Server"**
3. Configure:
   - **Name:** `miro`
   - **Command:** Download the Linux binary to your Repl first
   - **Environment:** `MIRO_ACCESS_TOKEN=your-token-here`

```bash
# In Replit shell
curl -L -o miro-mcp-server https://github.com/olgasafonova/miro-mcp-server/releases/latest/download/miro-mcp-server-linux-amd64
chmod +x miro-mcp-server
```

---

## Gemini CLI

Edit `~/.gemini/settings.json`:

```json
{
  "mcpServers": {
    "miro": {
      "command": "/usr/local/bin/miro-mcp-server",
      "env": {
        "MIRO_ACCESS_TOKEN": "your-token-here"
      }
    }
  }
}
```

**Windows path:** `C:\\Users\\YOUR_USERNAME\\AppData\\Local\\miro-mcp-server.exe`

---

## Amazon Q IDE Extension

Edit `~/.aws/amazonq/mcp.json`:

```json
{
  "mcpServers": {
    "miro": {
      "command": "/usr/local/bin/miro-mcp-server",
      "env": {
        "MIRO_ACCESS_TOKEN": "your-token-here"
      }
    }
  }
}
```

**Windows:** Edit `%USERPROFILE%\.aws\amazonq\mcp.json`

---

## Kiro IDE / Kiro CLI

Create or edit `.kiro/mcp.json` in your project directory:

```json
{
  "mcpServers": {
    "miro": {
      "command": "/usr/local/bin/miro-mcp-server",
      "env": {
        "MIRO_ACCESS_TOKEN": "your-token-here"
      }
    }
  }
}
```

Or globally at `~/.kiro/mcp.json`.

**Windows path:** `C:\\Users\\YOUR_USERNAME\\AppData\\Local\\miro-mcp-server.exe`

---

## OpenAI Codex CLI

Edit `~/.codex/config.toml`:

```toml
[mcp.servers.miro]
command = "/usr/local/bin/miro-mcp-server"
[mcp.servers.miro.env]
MIRO_ACCESS_TOKEN = "your-token-here"
```

**Windows:**
```toml
[mcp.servers.miro]
command = "C:\\Users\\YOUR_USERNAME\\AppData\\Local\\miro-mcp-server.exe"
[mcp.servers.miro.env]
MIRO_ACCESS_TOKEN = "your-token-here"
```

---

## Tools Without Local MCP Support

The following tools don't currently support local stdio-based MCP servers:

| Tool | Status | Alternative |
|------|--------|-------------|
| **Lovable** | URL-based MCP only | Use the HTTP mode: `miro-mcp-server -http :8080` and expose via tunnel |
| **Devin** | Remote MCP servers only | Devin uses hosted MCP at mcp.devin.ai |
| **Glean** | Enterprise search platform | Not an AI coding assistant; different use case |

---

## n8n

**Note:** n8n has a [built-in Miro integration](https://n8n.io/integrations/miro/) via HTTP Request node. For most n8n workflows, the native Miro node is the better choice.

This MCP server is designed for AI assistants (Claude, Cursor, etc.) that support the Model Context Protocol. n8n does not currently support MCP natively.

### If You Still Want to Use This Server with n8n

You can run the server in HTTP mode and call it via n8n's HTTP Request node:

```bash
MIRO_ACCESS_TOKEN=your-token-here miro-mcp-server -http :8080
```

The server exposes MCP-over-HTTP at `http://localhost:8080`. However, you'll need to format requests according to the MCP protocol, which is more complex than using n8n's native Miro integration.

---

## Other Tools

Any MCP-compatible tool can use this server. The pattern is always:

```json
{
  "command": "/path/to/miro-mcp-server",
  "env": {
    "MIRO_ACCESS_TOKEN": "your-token-here"
  }
}
```

---

## Verify It Works

In your AI tool, try:

> "List my Miro boards"

You should see a list of your boards.

---

## Troubleshooting

### "Token invalid" or "401 Unauthorized"

- Check your token is correct (no extra spaces)
- Verify the token has `boards:read` and `boards:write` permissions
- Try regenerating the token in Miro settings

### "Command not found"

- Verify the binary path is correct
- On macOS/Linux: run `which miro-mcp-server`
- On Windows: verify the .exe exists at the specified path

### "Permission denied"

- macOS/Linux: run `chmod +x /usr/local/bin/miro-mcp-server`
- Windows: run PowerShell as Administrator

### Server not showing in AI tool

- Restart the AI tool after configuration
- Check JSON syntax (no trailing commas, proper quotes)
- Verify the config file is in the correct location

### Rate limiting

The server handles rate limits automatically. If you see rate limit errors:
- Wait a few seconds and retry
- Check if other apps are using your Miro API quota

---

## Debugging

### MCP Inspector (Recommended)

[MCP Inspector](https://modelcontextprotocol.io/docs/tools/inspector) is the official tool for testing and debugging MCP servers.

```bash
# Run with MCP Inspector
npx @modelcontextprotocol/inspector miro-mcp-server

# With your token
MIRO_ACCESS_TOKEN=your-token npx @modelcontextprotocol/inspector miro-mcp-server

# With Docker
MIRO_ACCESS_TOKEN=your-token npx @modelcontextprotocol/inspector \
  docker run -i --rm -e MIRO_ACCESS_TOKEN ghcr.io/olgasafonova/miro-mcp-server
```

Open `http://localhost:6274` to:
- **Browse tools:** See all 76 tools with their input schemas
- **Test interactively:** Call any tool with custom parameters
- **View messages:** See raw JSON-RPC request/response
- **Debug errors:** Get detailed error information

### Manual JSON-RPC Testing

For quick CLI testing without the Inspector:

```bash
# Initialize and list tools
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}
{"jsonrpc":"2.0","method":"notifications/initialized"}
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | \
  MIRO_ACCESS_TOKEN=your-token miro-mcp-server
```

### Enable Debug Logging

Set log level for verbose output:

```bash
# More verbose output
MIRO_ACCESS_TOKEN=your-token miro-mcp-server 2>&1 | tee debug.log
```

### HTTP Mode Testing

Run in HTTP mode for easier debugging with curl:

```bash
# Start in HTTP mode
MIRO_ACCESS_TOKEN=your-token miro-mcp-server -http :8080

# Test with curl (in another terminal)
curl -X POST http://localhost:8080 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MIRO_ACCESS_TOKEN` | Yes | - | Your Miro OAuth token |
| `MIRO_TEAM_ID` | No | - | Filter to specific team |
| `MIRO_TIMEOUT` | No | 30s | Request timeout |

---

## Need Help?

- [GitHub Issues](https://github.com/olgasafonova/miro-mcp-server/issues)
- [Miro API Docs](https://developers.miro.com/reference/api-reference)

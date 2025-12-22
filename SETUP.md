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

## n8n

n8n has a built-in Miro integration via HTTP Request node. However, you can also use this MCP server:

### Option 1: HTTP Mode

Run the server in HTTP mode:

```bash
MIRO_ACCESS_TOKEN=your-token-here miro-mcp-server -http :8080
```

Then connect n8n via HTTP Request node to `http://localhost:8080`.

### Option 2: MCP Node (if available)

If n8n supports MCP nodes, configure:

```json
{
  "command": "/usr/local/bin/miro-mcp-server",
  "env": {
    "MIRO_ACCESS_TOKEN": "your-token-here"
  }
}
```

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

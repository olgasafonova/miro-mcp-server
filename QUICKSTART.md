# Quickstart: Miro MCP Server

Get AI-powered Miro control in 60 seconds.

## Step 1: Install (10 seconds)

**macOS/Linux:**
```bash
brew tap olgasafonova/tap && brew install miro-mcp-server
```

**Alternative (no Homebrew):**
```bash
curl -fsSL https://raw.githubusercontent.com/olgasafonova/miro-mcp-server/main/install.sh | sh
```

## Step 2: Get Miro Token (30 seconds)

1. Go to [miro.com/app/settings/user-profile/apps](https://miro.com/app/settings/user-profile/apps)
2. Click **"Create new app"**
3. Name it "MCP Server" and select your team
4. Enable permissions: `boards:read` and `boards:write`
5. Click **"Install app and get OAuth token"**
6. Copy the token

## Step 3: Configure Claude Code (20 seconds)

```bash
claude mcp add miro -e MIRO_ACCESS_TOKEN=your-token-here -- miro-mcp-server
```

## Done!

Try it:
```
You: "Create a sticky note saying Hello World on my Miro board"
Claude: "Created yellow sticky 'Hello World'"
```

---

## Other AI Tools

**Claude Desktop** - Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "miro": {
      "command": "miro-mcp-server",
      "env": {"MIRO_ACCESS_TOKEN": "your-token"}
    }
  }
}
```

**Cursor/VS Code** - Add to MCP settings:
```json
{
  "miro": {
    "command": "miro-mcp-server",
    "env": {"MIRO_ACCESS_TOKEN": "your-token"}
  }
}
```

---

## What You Can Do

| Say this... | Get this... |
|-------------|-------------|
| "Add 5 sticky notes with ideas" | 5 stickies in a grid |
| "Create a flowchart for login" | Mermaid diagram → Miro shapes |
| "List everything on my board" | Full board contents |
| "Connect box A to box B" | Arrow connector |
| "Tag this as Urgent" | Red tag attached |

See [README.md](README.md) for all 76 tools.

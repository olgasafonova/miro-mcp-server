# Competitive Analysis: Miro MCP Servers

A detailed comparison of Miro MCP servers for AI tool integration.

---

## Quick Summary

| Server | Language | Tools | Best For |
|--------|----------|-------|----------|
| **olgasafonova/miro-mcp-server** | Go | 66 | Production use, performance, single binary |
| k-jarzyna/mcp-miro | TypeScript | 81 | Maximum API coverage |
| LuotoCompany/mcp-server-miro | TypeScript | ~15 | Quick start, OpenAPI-based |
| evalstate/mcp-miro | TypeScript | ~8 | Basic operations |
| Miro Official MCP | Cloud | ~10 | No setup required |

---

## Detailed Comparison

### 1. olgasafonova/miro-mcp-server (This Project)

**Language:** Go
**Tools:** 66
**License:** MIT
**Distribution:** Homebrew, Docker, binaries

**Strengths:**
- Single binary (~14MB) - no runtime dependencies
- Built-in Mermaid diagram generation
- Automatic rate limiting with backoff
- Response caching (2-min TTL)
- Circuit breaker for fault tolerance
- Voice-optimized tool descriptions
- OAuth 2.1 with PKCE
- Local audit logging
- 5 platform binaries + Docker

**Limitations:**
- Fewer tools than k-jarzyna (66 vs 81)
- No Comments API yet
- Export features require Enterprise plan

**Ideal for:** Production deployments, teams needing reliability and performance

---

### 2. k-jarzyna/mcp-miro

**Language:** TypeScript
**Tools:** 81
**License:** Apache 2.0
**Stars:** 59

**Strengths:**
- Most comprehensive API coverage
- Includes Comments, Templates, Organization tools
- Active development

**Limitations:**
- Requires Node.js runtime
- No rate limiting or caching
- ~100MB with node_modules
- No diagram generation

**Ideal for:** Maximum feature coverage, TypeScript projects

---

### 3. LuotoCompany/mcp-server-miro

**Language:** TypeScript (FastMCP)
**Tools:** ~15
**License:** MIT
**Stars:** 14

**Strengths:**
- Auto-generated from Miro OpenAPI spec
- SSE transport support
- Simple architecture

**Limitations:**
- Fewer tools
- Basic functionality only
- No advanced features

**Ideal for:** Quick prototyping, OpenAPI-first approach

---

### 4. evalstate/mcp-miro

**Language:** TypeScript
**Tools:** ~8
**License:** Not specified
**Stars:** 101

**Strengths:**
- Simple, focused implementation
- Good for learning MCP basics
- Bulk item creation

**Limitations:**
- Very limited tool set
- No active development since Nov 2024
- Basic CRUD only

**Ideal for:** Learning MCP, simple use cases

---

### 5. Miro Official MCP

**Type:** Cloud-hosted
**Tools:** ~10

**Strengths:**
- No setup required
- Official support
- Built-in diagram and code generation

**Limitations:**
- Limited control
- Proprietary
- Fewer tools
- Requires internet

**Ideal for:** Quick demos, non-technical users

---

## Feature Matrix

### Board Operations

| Feature | This Project | k-jarzyna | LuotoCompany | evalstate | Official |
|---------|:------------:|:---------:|:------------:|:---------:|:--------:|
| List boards | ✅ | ✅ | ✅ | ✅ | ✅ |
| Get board | ✅ | ✅ | ✅ | ✅ | ✅ |
| Create board | ✅ | ✅ | ❌ | ❌ | ❌ |
| Copy board | ✅ | ✅ | ❌ | ❌ | ❌ |
| Update board | ✅ | ✅ | ❌ | ❌ | ❌ |
| Delete board | ✅ | ✅ | ❌ | ❌ | ❌ |
| Board summary | ✅ | ❌ | ❌ | ❌ | ❌ |
| Find by name | ✅ | ❌ | ❌ | ❌ | ❌ |

### Item Creation

| Feature | This Project | k-jarzyna | LuotoCompany | evalstate | Official |
|---------|:------------:|:---------:|:------------:|:---------:|:--------:|
| Sticky notes | ✅ | ✅ | ✅ | ✅ | ✅ |
| Shapes | ✅ | ✅ | ✅ | ✅ | ✅ |
| Text | ✅ | ✅ | ✅ | ❌ | ✅ |
| Connectors | ✅ | ✅ | ✅ | ❌ | ✅ |
| Frames | ✅ | ✅ | ❌ | ✅ | ✅ |
| Cards | ✅ | ✅ | ✅ | ❌ | ❌ |
| App cards | ✅ | ✅ | ❌ | ❌ | ❌ |
| Images | ✅ | ✅ | ✅ | ❌ | ❌ |
| Documents | ✅ | ✅ | ✅ | ❌ | ❌ |
| Embeds | ✅ | ✅ | ✅ | ❌ | ❌ |
| Bulk create | ✅ | ✅ | ❌ | ✅ | ❌ |
| Sticky grid | ✅ | ❌ | ❌ | ❌ | ❌ |

### Advanced Features

| Feature | This Project | k-jarzyna | LuotoCompany | evalstate | Official |
|---------|:------------:|:---------:|:------------:|:---------:|:--------:|
| Tags CRUD | ✅ | ✅ | ❌ | ❌ | ❌ |
| Groups | ✅ | ✅ | ❌ | ❌ | ❌ |
| Mindmaps | ✅ | ✅ | ❌ | ❌ | ❌ |
| Members/sharing | ✅ | ✅ | ❌ | ❌ | ❌ |
| Export | ✅ | ✅ | ❌ | ❌ | ❌ |
| Diagram generation | ✅ | ❌ | ❌ | ❌ | ✅ |
| Comments | ❌ | ✅ | ❌ | ❌ | ❌ |
| Templates | ❌ | ✅ | ❌ | ❌ | ❌ |

### Infrastructure

| Feature | This Project | k-jarzyna | LuotoCompany | evalstate | Official |
|---------|:------------:|:---------:|:------------:|:---------:|:--------:|
| Rate limiting | ✅ | ❌ | ❌ | ❌ | ✅ |
| Caching | ✅ | ❌ | ❌ | ❌ | ? |
| Circuit breaker | ✅ | ❌ | ❌ | ❌ | ? |
| Retry with backoff | ✅ | ❌ | ❌ | ❌ | ✅ |
| Token validation | ✅ | ❌ | ❌ | ❌ | ✅ |
| OAuth 2.1 | ✅ | ❌ | ❌ | ❌ | ✅ |
| Audit logging | ✅ | ❌ | ❌ | ❌ | ? |
| HTTP mode | ✅ | ❌ | ✅ | ❌ | ✅ |

---

## Performance Comparison

### Binary Size

| Server | Size | Notes |
|--------|------|-------|
| **This project** | ~14MB | Single binary, no dependencies |
| k-jarzyna | ~100MB+ | With node_modules |
| LuotoCompany | ~80MB+ | With node_modules |
| evalstate | ~80MB+ | With node_modules |

### Startup Time

| Server | Cold Start | Hot Start |
|--------|------------|-----------|
| **This project** | ~50ms | <10ms |
| TypeScript servers | ~500ms-2s | ~100ms |

### Memory Usage

| Server | Idle | Active |
|--------|------|--------|
| **This project** | ~10MB | ~30MB |
| TypeScript servers | ~50MB | ~100MB+ |

---

## Unique Features: This Project

### 1. Mermaid Diagram Generation

Convert Mermaid syntax directly to Miro shapes:

```
flowchart TB
    A[Start] --> B{Decision}
    B -->|Yes| C[Success]
    B -->|No| D[Retry]
```

No other Miro MCP server has this capability (except Official, which has limited support).

### 2. Voice-Optimized Descriptions

Tool descriptions designed for voice assistants:
- Short, speakable responses
- "USE WHEN" sections for AI understanding
- "VOICE-FRIENDLY" output templates

### 3. Composite Tools

Efficient multi-step operations:
- `miro_get_board_summary` - board + items + stats in one call
- `miro_create_sticky_grid` - multiple stickies in grid layout

### 4. Board Name Resolution

Find boards by name, not just ID:
```
"Find the Design Sprint board"
```

### 5. Local Audit Logging

Track all tool executions for debugging and compliance.

---

## Distribution Comparison

| Method | This Project | k-jarzyna | LuotoCompany |
|--------|:------------:|:---------:|:------------:|
| Homebrew | ✅ | ❌ | ❌ |
| Docker | ✅ | ✅ | ❌ |
| npm | ❌ | ✅ | ✅ |
| Binary download | ✅ | ❌ | ❌ |
| Install script | ✅ | ❌ | ❌ |
| Go install | ✅ | ❌ | ❌ |

---

## Platform Support

| Platform | This Project | TypeScript Servers |
|----------|:------------:|:------------------:|
| macOS Apple Silicon | ✅ Native | Requires Node.js |
| macOS Intel | ✅ Native | Requires Node.js |
| Linux x64 | ✅ Native | Requires Node.js |
| Linux ARM64 | ✅ Native | Requires Node.js |
| Windows x64 | ✅ Native | Requires Node.js |
| Docker | ✅ | ✅ |

---

## When to Choose Each

### Choose This Project When:

- You need **production-ready** infrastructure
- **Performance** matters (speed, memory)
- You want **single binary** deployment
- You need **Mermaid diagram** generation
- You prefer **zero runtime dependencies**
- You need **audit logging** for compliance
- You want **rate limiting** built-in

### Choose k-jarzyna When:

- You need **maximum API coverage**
- You need **Comments API**
- You're already in a **Node.js** ecosystem
- Tool count is more important than performance

### Choose LuotoCompany When:

- You want **OpenAPI-based** tool generation
- You need **SSE transport**
- You're doing quick **prototyping**

### Choose Official Miro MCP When:

- You want **zero setup**
- You don't need extensive features
- You're **evaluating** MCP technology

---

## Migration Guide

### From k-jarzyna to This Project

Most tools have identical names and parameters:

| k-jarzyna | This Project | Notes |
|-----------|--------------|-------|
| `miro_list_boards` | `miro_list_boards` | Identical |
| `miro_create_sticky_note` | `miro_create_sticky` | Shorter name |
| `miro_create_shape` | `miro_create_shape` | Identical |
| `miro_create_comment` | - | Not yet implemented |

1. Install this project
2. Update MCP config (same format)
3. Test with `miro_list_boards`

---

## Roadmap Comparison

### This Project (Planned)

- [ ] Comments API
- [ ] Templates API
- [ ] Presentation mode
- [ ] Real-time collaboration events

### k-jarzyna

- Active development
- Full API parity focus

---

## Conclusion

**For most production use cases**, this project (olgasafonova/miro-mcp-server) offers the best balance of:
- Performance
- Reliability
- Ease of deployment
- Unique features (diagrams, caching, rate limiting)

**For maximum API coverage**, k-jarzyna/mcp-miro has 15 more tools but lacks infrastructure features.

**For zero-setup demos**, Miro's official cloud MCP works without installation.

---

## Sources

- [k-jarzyna/mcp-miro](https://github.com/k-jarzyna/mcp-miro)
- [LuotoCompany/mcp-server-miro](https://github.com/LuotoCompany/mcp-server-miro)
- [evalstate/mcp-miro](https://github.com/evalstate/mcp-miro)
- [Miro Developer Documentation](https://developers.miro.com)
- [Best MCP Servers 2025](https://www.pomerium.com/blog/best-model-context-protocol-mcp-servers-in-2025)
- [MCP Server Frameworks Comparison](https://medium.com/@FrankGoortani/comparing-model-context-protocol-mcp-server-frameworks-03df586118fd)

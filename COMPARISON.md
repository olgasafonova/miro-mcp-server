# Competitive Analysis: Miro MCP Servers

A detailed comparison of Miro MCP servers for AI tool integration.

---

## Quick Summary

| Server | Language | Tools | Best For |
|--------|----------|-------|----------|
| **olgasafonova/miro-mcp-server** | Go | 76 | Production use, performance, single binary |
| k-jarzyna/mcp-miro | TypeScript | 90+ | Maximum API coverage, Enterprise features |
| LuotoCompany/mcp-server-miro | TypeScript | ~15 | Quick start, OpenAPI-based |
| evalstate/mcp-miro | TypeScript | ~8 | Basic operations |
| Miro Official MCP | Cloud | 5 | No setup, AI-powered diagrams |

---

## Detailed Comparison

### 1. olgasafonova/miro-mcp-server (This Project)

**Language:** Go
**Tools:** 76
**License:** MIT
**Distribution:** Homebrew, Docker, binaries

**Strengths:**
- Single binary (~14MB) - no runtime dependencies
- Built-in Mermaid diagram generation (flowcharts, sequence diagrams)
- Automatic rate limiting with backoff
- Response caching (2-min TTL)
- Circuit breaker for fault tolerance
- Voice-optimized tool descriptions
- OAuth 2.1 with PKCE
- Local audit logging
- Dry-run mode for destructive operations
- 5 platform binaries + Docker

**Limitations:**
- Fewer tools than k-jarzyna (76 vs 90+)
- No Enterprise compliance tools (legal holds, etc.)
- Export jobs require Enterprise plan

**Ideal for:** Production deployments, teams needing reliability and performance

---

### 2. k-jarzyna/mcp-miro

**Language:** TypeScript
**Tools:** 90+
**License:** Apache 2.0
**Stars:** 59

**Strengths:**
- Most comprehensive API coverage
- Enterprise compliance tools (legal holds, audit logs, classifications)
- Organization and project management
- Active development

**Limitations:**
- Requires Node.js runtime
- No rate limiting or caching
- ~100MB with node_modules
- No diagram generation

**Ideal for:** Enterprise users needing compliance features, TypeScript projects

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

**Type:** Cloud-hosted (https://mcp.miro.com/)
**Tools:** 5
**Prompts:** 2 (code_create_from_board, code_explain_on_board)

**Strengths:**
- No setup required - OAuth 2.1 with dynamic client registration
- Official Miro support
- AI-powered diagram generation from PRDs/text
- Code generation from board content
- Enterprise compliance (admin enablement required)

**Limitations:**
- Cloud-only (requires internet)
- Limited to 5 tools (read-focused, diagram generation)
- Uses Miro AI credits for some operations
- Beta status - features may change

**Ideal for:** Quick demos, AI-powered diagram/code generation workflows

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
| Export jobs | ✅ | ✅ | ❌ | ❌ | ❌ |
| Diagram generation | ✅ | ❌ | ❌ | ❌ | ✅ |
| MCP Prompts | ✅ | ❌ | ❌ | ❌ | ✅ |
| MCP Resources | ✅ | ❌ | ❌ | ❌ | ❌ |
| Enterprise compliance | ❌ | ✅ | ❌ | ❌ | ❌ |

> **Note:** Comments API and Templates API do not exist in Miro's public REST API.

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

- You need **Enterprise compliance tools** (legal holds, classifications)
- You need **maximum API coverage** (90+ tools)
- You're already in a **Node.js** ecosystem
- Organization/project management features are critical

### Choose LuotoCompany When:

- You want **OpenAPI-based** tool generation
- You need **SSE transport**
- You're doing quick **prototyping**

### Choose Official Miro MCP When:

- You want **zero setup** (cloud-hosted)
- You need **AI-powered diagram generation** from PRDs/text
- You want **code generation** from board content
- You're doing quick demos or **evaluating** MCP technology

---

## Migration Guide

### From k-jarzyna to This Project

Most tools have identical names and parameters:

| k-jarzyna | This Project | Notes |
|-----------|--------------|-------|
| `miro_list_boards` | `miro_list_boards` | Identical |
| `miro_create_sticky_note` | `miro_create_sticky` | Shorter name |
| `miro_create_shape` | `miro_create_shape` | Identical |

1. Install this project
2. Update MCP config (same format)
3. Test with `miro_list_boards`

---

## Roadmap Comparison

### This Project

- [x] MCP Prompts (5 workflow templates: sprint board, retrospective, brainstorm, story map, kanban)
- [x] MCP Resources (`miro://board/{id}`, `miro://board/{id}/items`, `miro://board/{id}/frames`)
- [ ] Additional diagram types (beyond flowchart and sequence)

### k-jarzyna

- Active development
- Enterprise compliance focus

---

## Conclusion

**For most production use cases**, this project (olgasafonova/miro-mcp-server) offers the best balance of:
- Performance (10-20x faster startup, 1/7th memory)
- Reliability (rate limiting, caching, circuit breaker)
- Ease of deployment (single binary, Homebrew, Docker)
- Unique features (Mermaid diagrams, dry-run mode, audit logging)

**For Enterprise compliance**, k-jarzyna/mcp-miro has legal holds, classifications, and organization tools.

**For AI-powered workflows**, Miro's official cloud MCP offers diagram/code generation with MCP Prompts.

---

## Sources

- [k-jarzyna/mcp-miro](https://github.com/k-jarzyna/mcp-miro)
- [LuotoCompany/mcp-server-miro](https://github.com/LuotoCompany/mcp-server-miro)
- [evalstate/mcp-miro](https://github.com/evalstate/mcp-miro)
- [Miro Official MCP](https://developers.miro.com/docs/miro-mcp)
- [Miro Developer Documentation](https://developers.miro.com)
- [Best MCP Servers 2025](https://www.pomerium.com/blog/best-model-context-protocol-mcp-servers-in-2025)
- [MCP Server Frameworks Comparison](https://medium.com/@FrankGoortani/comparing-model-context-protocol-mcp-server-frameworks-03df586118fd)

*Last updated: December 2025*

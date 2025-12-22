# Miro MCP Server - Development Roadmap

> **Goal**: Build the most comprehensive, performant, secure, and user-friendly Miro MCP server.
> **Language**: Go (unique differentiator - only Go-based Miro MCP server)
> **Status**: 66 tools implemented. Phases 1-7 complete.
> **Last Updated**: 2025-12-22

---

## Table of Contents

1. [Current State](#current-state)
2. [Competitive Analysis](#competitive-analysis)
3. [Gap Analysis](#gap-analysis)
4. [Implementation Roadmap](#implementation-roadmap)
5. [Technical Specifications](#technical-specifications)
6. [Code Patterns](#code-patterns)
7. [Testing Strategy](#testing-strategy)

---

## Current State

### Architecture

```
miro-mcp-server/
├── main.go                 # Entry point, dual transport (stdio/HTTP)
├── miro/
│   ├── client.go          # API client with rate limiting, caching
│   ├── config.go          # Environment-based configuration
│   └── types.go           # All request/response types
└── tools/
    ├── definitions.go     # Tool specifications (voice-optimized)
    └── handlers.go        # Generic handler registration
```

### Implemented Tools (65 total)

| Category | Tool | Method |
|----------|------|--------|
| **Boards (8)** | `miro_list_boards` | ListBoards |
| | `miro_get_board` | GetBoard |
| | `miro_create_board` | CreateBoard |
| | `miro_copy_board` | CopyBoard |
| | `miro_update_board` | UpdateBoard |
| | `miro_delete_board` | DeleteBoard |
| | `miro_find_board` | FindBoardByNameTool |
| | `miro_get_board_summary` | GetBoardSummary |
| **Members (5)** | `miro_list_board_members` | ListBoardMembers |
| | `miro_get_board_member` | GetBoardMember |
| | `miro_share_board` | ShareBoard |
| | `miro_update_board_member` | UpdateBoardMember |
| | `miro_remove_board_member` | RemoveBoardMember |
| **Create (14)** | `miro_create_sticky` | CreateSticky |
| | `miro_create_sticky_grid` | CreateStickyGrid |
| | `miro_create_shape` | CreateShape |
| | `miro_create_text` | CreateText |
| | `miro_create_frame` | CreateFrame |
| | `miro_create_card` | CreateCard |
| | `miro_create_app_card` | CreateAppCard |
| | `miro_create_image` | CreateImage |
| | `miro_create_document` | CreateDocument |
| | `miro_create_embed` | CreateEmbed |
| | `miro_create_connector` | CreateConnector |
| | `miro_create_group` | CreateGroup |
| | `miro_create_mindmap_node` | CreateMindmapNode |
| | `miro_bulk_create` | BulkCreate |
| **Frames (4)** | `miro_get_frame` | GetFrame |
| | `miro_update_frame` | UpdateFrame |
| | `miro_delete_frame` | DeleteFrame |
| | `miro_get_frame_items` | GetFrameItems |
| **Mindmaps (4)** | `miro_create_mindmap_node` | CreateMindmapNode |
| | `miro_get_mindmap_node` | GetMindmapNode |
| | `miro_list_mindmap_nodes` | ListMindmapNodes |
| | `miro_delete_mindmap_node` | DeleteMindmapNode |
| **Read (5)** | `miro_list_items` | ListItems |
| | `miro_list_all_items` | ListAllItems |
| | `miro_get_item` | GetItem |
| | `miro_get_app_card` | GetAppCard |
| | `miro_search_board` | SearchBoard |
| **Update (5)** | `miro_update_item` | UpdateItem |
| | `miro_update_app_card` | UpdateAppCard |
| | `miro_update_connector` | UpdateConnector |
| | `miro_update_tag` | UpdateTag |
| | `miro_update_frame` | UpdateFrame |
| **Delete (6)** | `miro_delete_item` | DeleteItem |
| | `miro_delete_app_card` | DeleteAppCard |
| | `miro_delete_connector` | DeleteConnector |
| | `miro_delete_tag` | DeleteTag |
| | `miro_delete_frame` | DeleteFrame |
| | `miro_delete_group` | DeleteGroup |
| **Tags (5)** | `miro_create_tag` | CreateTag |
| | `miro_list_tags` | ListTags |
| | `miro_attach_tag` | AttachTag |
| | `miro_detach_tag` | DetachTag |
| | `miro_get_item_tags` | GetItemTags |
| **Connectors (4)** | `miro_create_connector` | CreateConnector |
| | `miro_list_connectors` | ListConnectors |
| | `miro_get_connector` | GetConnector |
| | `miro_delete_connector` | DeleteConnector |
| **Groups (5)** | `miro_create_group` | CreateGroup |
| | `miro_ungroup` | Ungroup |
| | `miro_list_groups` | ListGroups |
| | `miro_get_group` | GetGroup |
| | `miro_get_group_items` | GetGroupItems |
| **Export (4)** | `miro_get_board_picture` | GetBoardPicture |
| | `miro_create_export_job` | CreateExportJob |
| | `miro_get_export_job_status` | GetExportJobStatus |
| | `miro_get_export_job_results` | GetExportJobResults |
| **Diagrams (1)** | `miro_generate_diagram` | GenerateDiagram |
| **Audit (1)** | `miro_get_audit_log` | GetAuditLog |

### Existing Strengths

- **Rate Limiting**: Semaphore-based (5 concurrent requests)
- **Caching**: 2-minute TTL for board data
- **Connection Pooling**: 100 max idle, 10 per host
- **Panic Recovery**: Catches and logs panics in handlers
- **Structured Logging**: slog with context
- **Dual Transport**: stdio (default) + HTTP with health endpoint
- **Voice-Optimized**: Tool descriptions designed for voice interaction
- **Token Validation**: Validates MIRO_ACCESS_TOKEN on startup with clear error messages
- **Board Name Resolution**: Find boards by name, not just ID (`miro_find_board`)
- **Input Sanitization**: Validates board IDs and content to prevent injection
- **Retry with Backoff**: Exponential backoff for rate-limited requests
- **Composite Tools**: `miro_get_board_summary`, `miro_create_sticky_grid`

---

## Competitive Analysis

### Competitor Overview

| Server | Language | Stars | Tools | Last Update | License |
|--------|----------|-------|-------|-------------|---------|
| **Official Miro MCP** | Hosted | N/A | ~10 | Active | Proprietary |
| **evalstate/mcp-miro** | TypeScript | 101 | ~8 | Nov 2024 | - |
| **k-jarzyna/mcp-miro** | TypeScript | 59 | 81 | Active | Apache 2.0 |
| **LuotoCompany/mcp-server-miro** | TypeScript | 14 | ~15 | Apr 2025 | MIT |
| **Ours** | **Go** | - | **65** | Active | MIT |

### Feature Comparison Matrix

| Feature | Ours | Official | evalstate | k-jarzyna | LuotoCompany |
|---------|------|----------|-----------|-----------|--------------|
| **Board list/get** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Board create/delete** | ✅ | ? | ❌ | ✅ | ❌ |
| **Board copy** | ✅ | ? | ❌ | ✅ | ❌ |
| **Sticky notes** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Shapes** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Text** | ✅ | ✅ | ? | ✅ | ✅ |
| **Connectors** | ✅ | ✅ | ? | ✅ | ✅ |
| **Frames** | ✅ | ✅ | ✅ | ✅ | ? |
| **Cards** | ✅ | ? | ? | ✅ | ✅ |
| **Images** | ✅ | ? | ? | ✅ | ✅ |
| **Documents** | ✅ | ? | ? | ✅ | ✅ |
| **Embeds** | ✅ | ? | ? | ✅ | ✅ |
| **Tags** | ✅ | ❌ | ❌ | ✅ | ❌ |
| **Groups** | ✅ | ❌ | ❌ | ✅ | ❌ |
| **Members/sharing** | ✅ | ❌ | ❌ | ✅ | ❌ |
| **Mindmaps** | ✅ | ❌ | ❌ | ✅ | ❌ |
| **Export** | ✅ | ❌ | ❌ | ✅ | ❌ |
| **Bulk operations** | ✅ | ? | ✅ | ✅ | ? |
| **Rate limiting** | ✅ | ✅ | ❌ | ❌ | ❌ |
| **Caching** | ✅ | ? | ❌ | ❌ | ❌ |
| **Dual transport** | ✅ | ❌ | ❌ | ❌ | ✅ (SSE) |
| **Voice-optimized** | ✅ | ❌ | ❌ | ❌ | ❌ |
| **Diagram generation** | ✅ | ✅ | ❌ | ❌ | ❌ |
| **Code generation** | ❌ | ✅ | ❌ | ❌ | ❌ |

### Our Unique Advantages

1. **Go Language**: Faster, lower memory, single binary deployment
2. **Rate Limiting**: Built-in protection against API limits
3. **Response Caching**: Reduces redundant API calls
4. **Voice-Optimized Descriptions**: Better for voice assistants
5. **Panic Recovery**: Production-safe error handling
6. **Dual Transport**: Works with any MCP client

---

## Gap Analysis

### Tier 1: High Priority (Must Have)

These features are commonly used and provided by competitors.

| Feature | Miro API Endpoint | Complexity | Impact |
|---------|-------------------|------------|--------|
| Cards | `POST /v2/boards/{id}/cards` | Medium | High |
| Images | `POST /v2/boards/{id}/images` | Low | High |
| Tags (CRUD) | `POST /v2/boards/{id}/tags` | Medium | High |
| Tag attach/detach | `POST /v2/boards/{id}/items/{id}/tags` | Low | High |
| Board create | `POST /v2/boards` | Low | Medium |
| Board copy | `POST /v2/boards/{id}/copy` | Low | Medium |
| Board delete | `DELETE /v2/boards/{id}` | Low | Low |

### Tier 2: Competitive Parity

| Feature | Miro API Endpoint | Complexity | Impact |
|---------|-------------------|------------|--------|
| Documents | `POST /v2/boards/{id}/documents` | Medium | Medium |
| Embeds | `POST /v2/boards/{id}/embeds` | Medium | Medium |
| Groups | `POST /v2/boards/{id}/groups` | Medium | Medium |
| Board members | `GET /v2/boards/{id}/members` | Low | Medium |
| Share board | `POST /v2/boards/{id}/members` | Medium | Medium |
| Mindmap nodes | `POST /v2/boards/{id}/mind_map_nodes` | Medium | Low |

### Tier 3: Differentiation (Beat Everyone)

| Feature | Description | Complexity | Impact |
|---------|-------------|------------|--------|
| Token validation | Verify token on startup | Low | High |
| Board name resolution | Find board by name, not just ID | Low | High |
| Composite tools | Single tool for common workflows | Medium | High |
| Retry with backoff | Handle rate limits gracefully | Medium | Medium |
| Input sanitization | Prevent injection attacks | Low | High |
| Fuzzy search | Typo-tolerant board/item search | Medium | Medium |

### Tier 4: Enterprise Features

| Feature | Description | Complexity | Impact |
|---------|-------------|------------|--------|
| OAuth 2.1 flow | Full OAuth instead of static token | High | High |
| Webhooks | Real-time event notifications | High | Medium |
| Audit logging | Track all operations | Medium | Low |
| Multi-board ops | Operations across multiple boards | High | Medium |

---

## Implementation Roadmap

### Phase 1: Core Completeness

**Goal**: Match k-jarzyna's feature set for common operations.

#### 1.1 Cards

```go
// Types to add in miro/types.go
type CreateCardArgs struct {
    BoardID     string  `json:"board_id" jsonschema:"required"`
    Title       string  `json:"title" jsonschema:"required"`
    Description string  `json:"description,omitempty"`
    DueDate     string  `json:"due_date,omitempty"` // ISO 8601
    X           float64 `json:"x,omitempty"`
    Y           float64 `json:"y,omitempty"`
    Width       float64 `json:"width,omitempty"`
    ParentID    string  `json:"parent_id,omitempty"`
}

// API endpoint: POST /v2/boards/{board_id}/cards
```

#### 1.2 Images

```go
type CreateImageArgs struct {
    BoardID  string  `json:"board_id" jsonschema:"required"`
    URL      string  `json:"url" jsonschema:"required"` // Must be publicly accessible
    Title    string  `json:"title,omitempty"`
    X        float64 `json:"x,omitempty"`
    Y        float64 `json:"y,omitempty"`
    Width    float64 `json:"width,omitempty"`
    ParentID string  `json:"parent_id,omitempty"`
}

// API endpoint: POST /v2/boards/{board_id}/images
// Request body: { "data": { "url": "..." }, "position": {...} }
```

#### 1.3 Tags

```go
// Tag CRUD
type CreateTagArgs struct {
    BoardID string `json:"board_id" jsonschema:"required"`
    Title   string `json:"title" jsonschema:"required"`
    Color   string `json:"color,omitempty"` // red, magenta, violet, blue, cyan, green, yellow, orange, gray
}

type ListTagsArgs struct {
    BoardID string `json:"board_id" jsonschema:"required"`
    Limit   int    `json:"limit,omitempty"`
}

type AttachTagArgs struct {
    BoardID string `json:"board_id" jsonschema:"required"`
    ItemID  string `json:"item_id" jsonschema:"required"`
    TagID   string `json:"tag_id" jsonschema:"required"`
}

// API endpoints:
// POST /v2/boards/{board_id}/tags
// GET /v2/boards/{board_id}/tags
// POST /v2/boards/{board_id}/items/{item_id}/tags/{tag_id}
// DELETE /v2/boards/{board_id}/items/{item_id}/tags/{tag_id}
```

#### 1.4 Board Management

```go
type CreateBoardArgs struct {
    Name        string `json:"name" jsonschema:"required"`
    Description string `json:"description,omitempty"`
    TeamID      string `json:"team_id,omitempty"`
}

type CopyBoardArgs struct {
    BoardID     string `json:"board_id" jsonschema:"required"`
    Name        string `json:"name,omitempty"` // New name, defaults to "Copy of {original}"
    Description string `json:"description,omitempty"`
    TeamID      string `json:"team_id,omitempty"`
}

type DeleteBoardArgs struct {
    BoardID string `json:"board_id" jsonschema:"required"`
}

// API endpoints:
// POST /v2/boards
// POST /v2/boards/{board_id}/copy
// DELETE /v2/boards/{board_id}
```

### Phase 2: Differentiation

#### 2.1 Token Validation

Add to `miro/client.go`:

```go
// ValidateToken verifies the access token is valid by calling /v2/users/me
func (c *Client) ValidateToken(ctx context.Context) (*UserInfo, error) {
    // Check cache first
    if cached, ok := c.getCached("token:valid"); ok {
        return cached.(*UserInfo), nil
    }

    respBody, err := c.request(ctx, http.MethodGet, "/users/me", nil)
    if err != nil {
        return nil, fmt.Errorf("token validation failed: %w", err)
    }

    var user UserInfo
    if err := json.Unmarshal(respBody, &user); err != nil {
        return nil, fmt.Errorf("failed to parse user info: %w", err)
    }

    // Cache for 5 minutes
    c.setCache("token:valid", &user)
    return &user, nil
}

type UserInfo struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}
```

Call in `main.go` at startup:

```go
// After creating client
user, err := client.ValidateToken(context.Background())
if err != nil {
    log.Fatalf("Invalid MIRO_ACCESS_TOKEN: %v", err)
}
logger.Info("Token validated", "user", user.Name, "email", user.Email)
```

#### 2.2 Board Name Resolution

Add helper that wraps ListBoards:

```go
// FindBoardByName finds a board by exact or fuzzy name match
func (c *Client) FindBoardByName(ctx context.Context, name string) (*BoardSummary, error) {
    result, err := c.ListBoards(ctx, ListBoardsArgs{
        Query: name,
        Limit: 10,
    })
    if err != nil {
        return nil, err
    }

    // Exact match first
    nameLower := strings.ToLower(name)
    for _, b := range result.Boards {
        if strings.ToLower(b.Name) == nameLower {
            return &b, nil
        }
    }

    // Partial match
    for _, b := range result.Boards {
        if strings.Contains(strings.ToLower(b.Name), nameLower) {
            return &b, nil
        }
    }

    if len(result.Boards) == 0 {
        return nil, fmt.Errorf("no board found matching '%s'", name)
    }

    return &result.Boards[0], nil
}
```

#### 2.3 Input Sanitization

Add to `miro/client.go`:

```go
import "regexp"

var (
    validIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_=-]+$`)
    maxContentLen  = 10000
)

// ValidateBoardID ensures board ID is safe
func ValidateBoardID(id string) error {
    if id == "" {
        return fmt.Errorf("board_id is required")
    }
    if len(id) > 100 {
        return fmt.Errorf("board_id too long")
    }
    if !validIDPattern.MatchString(id) {
        return fmt.Errorf("board_id contains invalid characters")
    }
    return nil
}

// ValidateContent ensures content is safe and within limits
func ValidateContent(content string) error {
    if len(content) > maxContentLen {
        return fmt.Errorf("content exceeds maximum length of %d", maxContentLen)
    }
    return nil
}
```

#### 2.4 Composite Tools

Add efficient multi-step tools:

```go
// Tool: miro_get_board_summary
// Combines: GetBoard + ListItems + stats
type GetBoardSummaryArgs struct {
    BoardID string `json:"board_id" jsonschema:"required"`
}

type GetBoardSummaryResult struct {
    Board       BoardSummary          `json:"board"`
    ItemCounts  map[string]int        `json:"item_counts"`  // {"sticky_note": 15, "shape": 8, ...}
    RecentItems []ItemSummary         `json:"recent_items"` // Last 5 modified
    Message     string                `json:"message"`
}

func (c *Client) GetBoardSummary(ctx context.Context, args GetBoardSummaryArgs) (GetBoardSummaryResult, error) {
    // Parallel fetch board and items
    var board GetBoardResult
    var items ListItemsResult
    var wg sync.WaitGroup
    var boardErr, itemsErr error

    wg.Add(2)
    go func() {
        defer wg.Done()
        board, boardErr = c.GetBoard(ctx, GetBoardArgs{BoardID: args.BoardID})
    }()
    go func() {
        defer wg.Done()
        items, itemsErr = c.ListItems(ctx, ListItemsArgs{BoardID: args.BoardID, Limit: 100})
    }()
    wg.Wait()

    if boardErr != nil {
        return GetBoardSummaryResult{}, boardErr
    }
    if itemsErr != nil {
        return GetBoardSummaryResult{}, itemsErr
    }

    // Count by type
    counts := make(map[string]int)
    for _, item := range items.Items {
        counts[item.Type]++
    }

    return GetBoardSummaryResult{
        Board: BoardSummary{
            ID:          board.ID,
            Name:        board.Name,
            Description: board.Description,
            ViewLink:    board.ViewLink,
        },
        ItemCounts:  counts,
        RecentItems: items.Items[:min(5, len(items.Items))],
        Message:     fmt.Sprintf("Board '%s' has %d items", board.Name, items.Count),
    }, nil
}
```

```go
// Tool: miro_create_sticky_grid
// Creates multiple stickies in a grid layout
type CreateStickyGridArgs struct {
    BoardID  string   `json:"board_id" jsonschema:"required"`
    Contents []string `json:"contents" jsonschema:"required"` // Text for each sticky
    Columns  int      `json:"columns,omitempty"`              // Default 3
    Color    string   `json:"color,omitempty"`
    StartX   float64  `json:"start_x,omitempty"`
    StartY   float64  `json:"start_y,omitempty"`
    Spacing  float64  `json:"spacing,omitempty"` // Default 220
    ParentID string   `json:"parent_id,omitempty"`
}

type CreateStickyGridResult struct {
    Created int      `json:"created"`
    ItemIDs []string `json:"item_ids"`
    Message string   `json:"message"`
}

func (c *Client) CreateStickyGrid(ctx context.Context, args CreateStickyGridArgs) (CreateStickyGridResult, error) {
    columns := args.Columns
    if columns <= 0 {
        columns = 3
    }
    spacing := args.Spacing
    if spacing == 0 {
        spacing = 220
    }

    items := make([]BulkCreateItem, len(args.Contents))
    for i, content := range args.Contents {
        row := i / columns
        col := i % columns
        items[i] = BulkCreateItem{
            Type:     "sticky_note",
            Content:  content,
            X:        args.StartX + float64(col)*spacing,
            Y:        args.StartY + float64(row)*spacing,
            Color:    args.Color,
            ParentID: args.ParentID,
        }
    }

    result, err := c.BulkCreate(ctx, BulkCreateArgs{
        BoardID: args.BoardID,
        Items:   items,
    })
    if err != nil {
        return CreateStickyGridResult{}, err
    }

    return CreateStickyGridResult{
        Created: result.Created,
        ItemIDs: result.ItemIDs,
        Message: fmt.Sprintf("Created %d stickies in %dx%d grid", result.Created, columns, (len(args.Contents)+columns-1)/columns),
    }, nil
}
```

#### 2.5 Retry with Exponential Backoff

Add to `miro/client.go`:

```go
// requestWithRetry wraps request with retry logic for rate limits
func (c *Client) requestWithRetry(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
    maxRetries := 3
    baseDelay := 1 * time.Second

    var lastErr error
    for attempt := 0; attempt <= maxRetries; attempt++ {
        respBody, err := c.request(ctx, method, path, body)
        if err == nil {
            return respBody, nil
        }

        // Check if rate limited (429)
        if strings.Contains(err.Error(), "429") {
            delay := baseDelay * time.Duration(1<<attempt) // Exponential: 1s, 2s, 4s
            c.logger.Warn("Rate limited, retrying",
                "attempt", attempt+1,
                "delay", delay,
                "path", path,
            )
            select {
            case <-time.After(delay):
                continue
            case <-ctx.Done():
                return nil, ctx.Err()
            }
        }

        lastErr = err
        break // Don't retry non-rate-limit errors
    }

    return nil, lastErr
}
```

### Phase 3: Additional Features

#### 3.1 Documents

```go
type CreateDocumentArgs struct {
    BoardID  string  `json:"board_id" jsonschema:"required"`
    URL      string  `json:"url" jsonschema:"required"` // PDF or document URL
    Title    string  `json:"title,omitempty"`
    X        float64 `json:"x,omitempty"`
    Y        float64 `json:"y,omitempty"`
    Width    float64 `json:"width,omitempty"`
    ParentID string  `json:"parent_id,omitempty"`
}

// API: POST /v2/boards/{board_id}/documents
// Body: { "data": { "url": "..." }, "position": {...} }
```

#### 3.2 Embeds

```go
type CreateEmbedArgs struct {
    BoardID  string  `json:"board_id" jsonschema:"required"`
    URL      string  `json:"url" jsonschema:"required"` // YouTube, Figma, Google Docs, etc.
    Mode     string  `json:"mode,omitempty"`            // "inline" or "modal"
    X        float64 `json:"x,omitempty"`
    Y        float64 `json:"y,omitempty"`
    Width    float64 `json:"width,omitempty"`
    Height   float64 `json:"height,omitempty"`
    ParentID string  `json:"parent_id,omitempty"`
}

// API: POST /v2/boards/{board_id}/embeds
// Body: { "data": { "url": "...", "mode": "inline" }, "position": {...}, "geometry": {...} }
```

#### 3.3 Groups

```go
type CreateGroupArgs struct {
    BoardID string   `json:"board_id" jsonschema:"required"`
    ItemIDs []string `json:"item_ids" jsonschema:"required"` // Items to group
}

type UngroupArgs struct {
    BoardID string `json:"board_id" jsonschema:"required"`
    GroupID string `json:"group_id" jsonschema:"required"`
}

// API: POST /v2/boards/{board_id}/groups
// Body: { "items": ["id1", "id2", ...] }
```

#### 3.4 Board Members

```go
type ListBoardMembersArgs struct {
    BoardID string `json:"board_id" jsonschema:"required"`
    Limit   int    `json:"limit,omitempty"`
}

type ShareBoardArgs struct {
    BoardID string `json:"board_id" jsonschema:"required"`
    Email   string `json:"email" jsonschema:"required"`
    Role    string `json:"role,omitempty"` // "viewer", "commenter", "editor"
}

// API:
// GET /v2/boards/{board_id}/members
// POST /v2/boards/{board_id}/members
```

---

## Technical Specifications

### Miro API Reference

Base URL: `https://api.miro.com/v2`

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/boards` | GET | List boards |
| `/boards` | POST | Create board |
| `/boards/{id}` | GET | Get board |
| `/boards/{id}` | DELETE | Delete board |
| `/boards/{id}/copy` | POST | Copy board |
| `/boards/{id}/items` | GET | List items |
| `/boards/{id}/items/{id}` | GET | Get item |
| `/boards/{id}/items/{id}` | PATCH | Update item |
| `/boards/{id}/items/{id}` | DELETE | Delete item |
| `/boards/{id}/sticky_notes` | POST | Create sticky |
| `/boards/{id}/shapes` | POST | Create shape |
| `/boards/{id}/texts` | POST | Create text |
| `/boards/{id}/connectors` | POST | Create connector |
| `/boards/{id}/frames` | POST | Create frame |
| `/boards/{id}/cards` | POST | Create card |
| `/boards/{id}/images` | POST | Create image |
| `/boards/{id}/documents` | POST | Create document |
| `/boards/{id}/embeds` | POST | Create embed |
| `/boards/{id}/tags` | GET/POST | List/Create tags |
| `/boards/{id}/items/{id}/tags/{id}` | POST/DELETE | Attach/Detach tag |
| `/boards/{id}/groups` | POST | Create group |
| `/boards/{id}/members` | GET/POST | List/Add members |
| `/users/me` | GET | Current user (for token validation) |

### Rate Limits

- 100,000 credits per minute per user
- Each API call costs 1-10 credits depending on operation
- Our implementation: Conservative 5 concurrent requests

### Authentication

Currently: Static access token via `MIRO_ACCESS_TOKEN` environment variable.

Future: OAuth 2.1 authorization code flow for multi-user support.

---

## Code Patterns

### Adding a New Tool

1. **Add types to `miro/types.go`**:

```go
type NewFeatureArgs struct {
    BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
    // ... other fields
}

type NewFeatureResult struct {
    ID      string `json:"id"`
    Message string `json:"message"`
}
```

2. **Add client method to `miro/client.go`**:

```go
func (c *Client) NewFeature(ctx context.Context, args NewFeatureArgs) (NewFeatureResult, error) {
    if args.BoardID == "" {
        return NewFeatureResult{}, fmt.Errorf("board_id is required")
    }

    reqBody := map[string]interface{}{
        // ... build request body
    }

    respBody, err := c.request(ctx, http.MethodPost, "/boards/"+args.BoardID+"/new_feature", reqBody)
    if err != nil {
        return NewFeatureResult{}, err
    }

    var result struct {
        ID string `json:"id"`
    }
    if err := json.Unmarshal(respBody, &result); err != nil {
        return NewFeatureResult{}, fmt.Errorf("failed to parse response: %w", err)
    }

    return NewFeatureResult{
        ID:      result.ID,
        Message: "Created new feature",
    }, nil
}
```

3. **Add tool spec to `tools/definitions.go`**:

```go
{
    Name:     "miro_new_feature",
    Method:   "NewFeature",
    Title:    "New Feature",
    Category: "create",
    Description: `Create a new feature item.

USE WHEN: User says "add new feature", "create X"

PARAMETERS:
- board_id: Required
- ...

VOICE-FRIENDLY: "Created new feature on the board"`,
},
```

4. **Register handler in `tools/handlers.go`**:

Add to `registerByName()` switch:
```go
case "NewFeature":
    h.register(server, tool, spec, h.client.NewFeature)
```

Add to `register()` switch:
```go
case func(context.Context, miro.NewFeatureArgs) (miro.NewFeatureResult, error):
    register(h, server, tool, spec, m)
```

Add to `logExecution()` if custom logging needed.

---

## Testing Strategy

### Unit Tests

```go
// miro/client_test.go
func TestCreateSticky(t *testing.T) {
    // Mock HTTP server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify request
        assert.Equal(t, "POST", r.Method)
        assert.Equal(t, "/v2/boards/test-board/sticky_notes", r.URL.Path)

        // Return mock response
        w.WriteHeader(http.StatusCreated)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "id": "sticky-123",
            "data": map[string]string{"content": "Test"},
        })
    }))
    defer server.Close()

    // Test client
    client := NewTestClient(server.URL)
    result, err := client.CreateSticky(context.Background(), CreateStickyArgs{
        BoardID: "test-board",
        Content: "Test",
    })

    assert.NoError(t, err)
    assert.Equal(t, "sticky-123", result.ID)
}
```

### Integration Tests

```go
// miro/integration_test.go
// +build integration

func TestIntegration_FullWorkflow(t *testing.T) {
    token := os.Getenv("MIRO_TEST_TOKEN")
    if token == "" {
        t.Skip("MIRO_TEST_TOKEN not set")
    }

    client := NewClient(&Config{AccessToken: token}, slog.Default())

    // 1. List boards
    boards, err := client.ListBoards(context.Background(), ListBoardsArgs{Limit: 1})
    require.NoError(t, err)
    require.NotEmpty(t, boards.Boards)

    boardID := boards.Boards[0].ID

    // 2. Create sticky
    sticky, err := client.CreateSticky(context.Background(), CreateStickyArgs{
        BoardID: boardID,
        Content: "Integration Test - " + time.Now().Format(time.RFC3339),
        Color:   "yellow",
    })
    require.NoError(t, err)
    require.NotEmpty(t, sticky.ID)

    // 3. Delete sticky (cleanup)
    _, err = client.DeleteItem(context.Background(), DeleteItemArgs{
        BoardID: boardID,
        ItemID:  sticky.ID,
    })
    require.NoError(t, err)
}
```

### Running Tests

```bash
# Unit tests
go test ./...

# Integration tests (requires real token)
MIRO_TEST_TOKEN=your_token go test -tags=integration ./...

# With coverage
go test -cover ./...
```

---

## Appendix: Full Tool List Target

### Phase 1 Tools (26 implemented)

| Tool | Status |
|------|--------|
| `miro_list_boards` | ✅ Done |
| `miro_get_board` | ✅ Done |
| `miro_create_board` | ✅ Done |
| `miro_copy_board` | ✅ Done |
| `miro_delete_board` | ✅ Done |
| `miro_create_sticky` | ✅ Done |
| `miro_create_shape` | ✅ Done |
| `miro_create_text` | ✅ Done |
| `miro_create_connector` | ✅ Done |
| `miro_create_frame` | ✅ Done |
| `miro_create_card` | ✅ Done |
| `miro_create_image` | ✅ Done |
| `miro_create_document` | ✅ Done |
| `miro_create_embed` | ✅ Done |
| `miro_bulk_create` | ✅ Done |
| `miro_list_items` | ✅ Done |
| `miro_list_all_items` | ✅ Done |
| `miro_get_item` | ✅ Done |
| `miro_search_board` | ✅ Done |
| `miro_update_item` | ✅ Done |
| `miro_delete_item` | ✅ Done |
| `miro_list_tags` | ✅ Done |
| `miro_create_tag` | ✅ Done |
| `miro_attach_tag` | ✅ Done |
| `miro_detach_tag` | ✅ Done |
| `miro_get_item_tags` | ✅ Done |

### Phase 2 Tools (Differentiation)

| Tool | Status |
|------|--------|
| `miro_get_board_summary` | ✅ Done |
| `miro_create_sticky_grid` | ✅ Done |
| `miro_find_board` | ✅ Done |

### Phase 2 Enhancements

| Feature | Status |
|---------|--------|
| Token validation on startup | ✅ Done |
| Board name resolution | ✅ Done |
| Input sanitization | ✅ Done |
| Retry with exponential backoff | ✅ Done |

### Phase 3 Tools (Additional Features)

| Tool | Status |
|------|--------|
| `miro_create_group` | ✅ Done |
| `miro_ungroup` | ✅ Done |
| `miro_list_board_members` | ✅ Done |
| `miro_share_board` | ✅ Done |
| `miro_create_mindmap_node` | ✅ Done |

### Phase 4 Tools (Export)

| Tool | Status | Notes |
|------|--------|-------|
| `miro_get_board_picture` | ✅ Done | All plans - gets board thumbnail URL |
| `miro_create_export_job` | ✅ Done | Enterprise only - PDF/SVG/HTML export |
| `miro_get_export_job_status` | ✅ Done | Enterprise only - check progress |
| `miro_get_export_job_results` | ✅ Done | Enterprise only - get download links |

### Phase 5: Enterprise Features (Complete ✅)

| Feature | Status | Notes |
|---------|--------|-------|
| Audit Logging (Local) | ✅ Done | File/memory logger, middleware integration, query tool |
| OAuth 2.1 Flow | ✅ Done | Full OAuth with PKCE, auto-refresh, CLI commands |
| Webhooks Support | ❌ Removed | Miro sunset experimental webhooks Dec 2025 |

#### Phase 5 Tools

| Tool | Status | Notes |
|------|--------|-------|
| `miro_get_audit_log` | ✅ Done | Query local audit log for MCP tool executions |

#### Phase 5 Enhancements

| Feature | Status |
|---------|--------|
| Audit event logging for all tool calls | ✅ Done |
| File-based audit logger with rotation | ✅ Done |
| Memory-based audit logger (dev/testing) | ✅ Done |
| Sensitive input sanitization | ✅ Done |
| Event builder with fluent API | ✅ Done |
| OAuth 2.1 with PKCE support | ✅ Done |
| OAuth token auto-refresh | ✅ Done |
| OAuth CLI commands (login/status/logout) | ✅ Done |
| Secure token storage (~/.miro/tokens.json) | ✅ Done |

### Phase 6: Extended Features (Complete ✅)

| Feature | Status | Notes |
|---------|--------|-------|
| Diagram Generation | ✅ Done | Mermaid flowchart and sequence diagrams → Miro shapes |
| Connector List/Get | ✅ Done | Full CRUD for connectors |
| Tag Update/Delete | ✅ Done | Complete tag management |

#### Phase 6 Tools

| Tool | Status | Notes |
|------|--------|-------|
| `miro_generate_diagram` | ✅ Done | Convert Mermaid to Miro shapes/connectors |
| `miro_list_connectors` | ✅ Done | List all connectors on a board |
| `miro_get_connector` | ✅ Done | Get full connector details |
| `miro_update_connector` | ✅ Done | Update connector style/endpoints |
| `miro_delete_connector` | ✅ Done | Delete a connector |
| `miro_update_tag` | ✅ Done | Update tag title/color |
| `miro_delete_tag` | ✅ Done | Delete a tag |

#### Phase 6 Enhancements

| Feature | Status |
|---------|--------|
| Mermaid flowchart parser | ✅ Done |
| Mermaid sequence diagram parser | ✅ Done |
| Sugiyama-style layout algorithm | ✅ Done |
| Auto-layout for diagrams | ✅ Done |
| Support for 7 node shapes | ✅ Done |
| Support for 4 edge styles | ✅ Done |

### Phase 7: Frame & Mindmap Tools + Distribution (Complete ✅)

| Feature | Status | Notes |
|---------|--------|-------|
| Frame CRUD | ✅ Done | Get, update, delete, list items in frame |
| Mindmap CRUD | ✅ Done | Get, list, delete mindmap nodes |
| App Card CRUD | ✅ Done | Create, get, update, delete app cards |
| Member Management | ✅ Done | Get, update, remove board members |
| Group Management | ✅ Done | List, get, get items, delete groups |
| Board Update | ✅ Done | Update board name/description |
| Distribution | ✅ Done | Homebrew tap, Docker, install script |

#### Phase 7 Tools (+12 new, 65 total)

| Tool | Status | Notes |
|------|--------|-------|
| `miro_get_frame` | ✅ Done | Get frame details |
| `miro_update_frame` | ✅ Done | Update frame title/color/size |
| `miro_delete_frame` | ✅ Done | Delete a frame |
| `miro_get_frame_items` | ✅ Done | List items inside a frame |
| `miro_get_mindmap_node` | ✅ Done | Get node details (v2-experimental API) |
| `miro_list_mindmap_nodes` | ✅ Done | List all mindmap nodes |
| `miro_delete_mindmap_node` | ✅ Done | Delete a mindmap node |
| `miro_create_app_card` | ✅ Done | Create app card with custom fields |
| `miro_get_app_card` | ✅ Done | Get app card details |
| `miro_update_app_card` | ✅ Done | Update app card fields |
| `miro_delete_app_card` | ✅ Done | Delete an app card |
| `miro_update_board` | ✅ Done | Update board name/description |
| `miro_get_board_member` | ✅ Done | Get member details |
| `miro_update_board_member` | ✅ Done | Update member role |
| `miro_remove_board_member` | ✅ Done | Remove member from board |
| `miro_list_groups` | ✅ Done | List all groups on board |
| `miro_get_group` | ✅ Done | Get group details |
| `miro_get_group_items` | ✅ Done | List items in a group |
| `miro_delete_group` | ✅ Done | Delete a group |

#### Phase 7 Distribution

| Platform | Status | Notes |
|----------|--------|-------|
| Homebrew tap | ✅ Done | `brew tap olgasafonova/tap && brew install miro-mcp-server` |
| Docker image | ✅ Done | `ghcr.io/olgasafonova/miro-mcp-server:latest` |
| Install script | ✅ Done | `curl -fsSL https://...install.sh | sh` |
| Linux ARM64 | ✅ Done | New binary for ARM64 Linux |
| GitHub Release | ✅ Done | v1.7.0 with all binaries |

---

## Notes for Future Claude Code Sessions

1. **Types are already defined** for Cards, Images, Documents, Embeds, and Tags in `miro/types.go` - just need client methods and tool registration.

2. **Follow the existing pattern** - see `CreateSticky` as the template for all create operations.

3. **Tool descriptions are voice-optimized** - keep them short, action-oriented, with "USE WHEN" and "VOICE-FRIENDLY" sections.

4. **Test with real Miro account** - get a token at https://miro.com/app/settings/user-profile/apps

5. **Rate limits** - Miro allows 100k credits/minute, but our semaphore limits to 5 concurrent requests for safety.

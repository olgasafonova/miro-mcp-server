# TODO - Remaining Tasks

Tasks identified during code review session (Dec 2024) that require more time or careful consideration.

## Code Quality

### GetTag Naming Inconsistency ✅ FIXED
- **Status**: Completed (Dec 2024)
- **Fix Applied**: Renamed internal helper to `getTagInternal` (unexported), renamed `GetTagTool` to `GetTag`
- **Files Updated**: `miro/tags.go`, `miro/interfaces.go`, `tools/handlers.go`, `tools/definitions.go`, `tools/mock_client_test.go`

### Tool Description Optimization
- **Status**: Reviewed - current descriptions are reasonable
- **miro_generate_diagram**: Verbose but includes essential Mermaid syntax examples
- **Consideration**: Could extract syntax examples to separate resource/documentation
- **Priority**: Low - descriptions work well for LLMs

## New Features (Miro API)

### Board Templates (v2)
- POST /boards with template_id parameter
- List available templates

### Comments API (v2-experimental)
- Add/list/update comments on items
- Resolve comment threads

### Sticky Note Color Themes
- Support for custom color palettes
- Theme-based sticky creation

### Frame Presets
- Kanban board frame layout
- Sprint retrospective templates

## MCP Protocol Features

### Resources
- Expose board content as MCP resources
- `miro://board/{id}` resource URIs
- Real-time resource updates (when webhooks return)

### Prompts
- Pre-built prompt templates for common workflows
- "Create sprint board" prompt
- "Retrospective template" prompt

## Performance

### Batch Operations
- Parallel item creation already implemented (BulkCreate)
- Consider: Batch update, batch delete operations

### Cache Improvements
- Current: Unified *Cache with TTL
- Consider: LRU eviction, cache size limits

## Testing

### Integration Tests
- Currently require MIRO_TEST_TOKEN
- Consider: Record/replay HTTP for offline testing
- Increase coverage for edge cases

### Load Testing
- Verify rate limiting behavior under load
- Test concurrent operations

## Documentation

### API Examples
- Add more code examples to CLAUDE.md
- Document common workflows

### Troubleshooting Guide
- Common errors and solutions
- Rate limit handling
- Token refresh issues

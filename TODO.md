# TODO - Remaining Tasks

Last updated: Dec 2024

## Recently Completed (Dec 2024)

### Dead Code Removal ✅
- Removed entire `miro/webhooks/` package (Miro sunset Dec 5, 2025)
- Deleted `miro/webhooks.go`, `miro/types_webhooks.go`
- Removed `WebhookService` from `MiroClient` interface
- Cleaned up mock client test

### Test Coverage Improvements ✅
- Added app card tests (21 tests, was 0% coverage)
- Added audit helper tests (+7 tests for WithItemCount, WithInput, Failure, CurrentFilePath, Flush)
- Added tests for functions with 0% coverage (Dec 2024):
  - `GetTag`/`getTagInternal`: 100%/91.7% (was 0%)
  - Frame operations: GetFrame 93.5%, UpdateFrame 67.7%, DeleteFrame 90.0%, GetFrameItems 72.7%
  - Group operations: ListGroups 73.3%, GetGroup 83.3%, DeleteGroup 78.6%
  - Bulk operations: BulkUpdate 83.7%, BulkDelete 96.4%
  - Member operations: GetBoardMember/RemoveBoardMember/UpdateBoardMember 66.7%
  - Board operations: UpdateBoard 88.2%
- Added GetGroupItems tests (3 tests): 82.6% coverage
- Added mindmap tests (8 tests): GetMindmapNode 87.5%, ListMindmapNodes 80.8%, DeleteMindmapNode 90.0%
- Current coverage: miro 79.3%, audit 82.1%, tools 85.0%

### Dependency Updates ✅
- jwt v5.2.2 → v5.3.0
- metadata v0.3.0 → v0.9.0

### GetTag Naming Inconsistency ✅
- Renamed internal helper to `getTagInternal` (unexported)
- Renamed `GetTagTool` to `GetTag`

## Code Quality

### Tool Description Optimization
- **Status**: Reviewed - current descriptions are reasonable
- **miro_generate_diagram**: Verbose but includes essential Mermaid syntax examples
- **Consideration**: Could extract syntax examples to separate resource/documentation
- **Priority**: Low - descriptions work well for LLMs

### Test Coverage Gaps
- **miro package**: 79.3% - remaining gaps in utility functions and error paths
- **diagrams package**: 73.3% - edge cases in Mermaid parsing
- **oauth package**: 72.5% - token refresh edge cases
- **GenerateDiagram**: 0% - diagram generation from Mermaid (miro/diagrams.go)

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

### Prompts
- Pre-built prompt templates for common workflows
- "Create sprint board" prompt
- "Retrospective template" prompt

## Performance

### Batch Operations ✅
- BulkCreate: implemented
- BulkUpdate: implemented
- BulkDelete: implemented

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

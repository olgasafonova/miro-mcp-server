# TODO - Remaining Tasks

Last updated: Dec 2025

## Recently Completed (Dec 2025)

### Type-Specific Update Operations ✅ (NEW)
- Added **3 new update tools** for complete item update coverage:
  - `miro_update_image`: Update image title, URL, position, size
  - `miro_update_document`: Update document title, URL, position, size
  - `miro_update_embed`: Update embed URL, mode, position, dimensions
- **Total tools**: 73 → **76 tools**
- Added implementation in `miro/items.go`
- Added tool specs in `tools/definitions.go`
- Updated mock client for testing
- Added 10 tests: UpdateImage 63.9%, UpdateDocument 63.9%, UpdateEmbed 73.2%

### Diagrams Package Coverage ✅ (NEW)
- **diagrams package**: 73.3% → **92.6%** coverage
- Added `converter_test.go` with tests for:
  - `convertFlowchartToMiro`: 0% → 100%
  - `convertShape`: 0% → 100%
  - `convertArrowType`: 50% → 100%
  - `getShapeColor`: 0% → 100%
  - `ConvertToMiro`: 100%
  - `ConvertSequenceToMiro`: 97.6%
- Added `errors_test.go` with tests for:
  - `DiagramError.Error()`: 0% → 100%
  - `WithLine`, `WithInput`: 0% → 100%
  - `ErrTooManyNodes`, `ErrInvalidNodeShape`, `ErrInvalidEdge`: 0% → 100%
  - `ParseDiagramSyntaxError`: 0% → 100%
  - `DiagramTypeHint`: 0% → 100%
  - `ValidateDiagramInput`: 0% → 100%
- Added `sequence_test.go` with tests for `ParseSequence`: 0% → 100%

### GenerateDiagram Tests ✅ (NEW)
- Added `miro/diagrams_test.go` with 15 tests
- **GenerateDiagram**: 0% → covered via mock HTTP server
- Tests cover validation, flowcharts, sequence diagrams, error paths

### OAuth Package Review ✅ (NEW)
- **oauth package**: 72.5% - reviewed and documented
- 0% coverage functions are integration-level (not feasible to unit test):
  - `Login`: Interactive browser OAuth flow
  - `openBrowser`: Platform-specific browser launching
- Token refresh path tested via `Provider.RefreshToken` tests

### Dead Code Removal ✅
- Removed entire `miro/webhooks/` package (Miro sunset Dec 5, 2025)
- Deleted `miro/webhooks.go`, `miro/types_webhooks.go`
- Removed `WebhookService` from `MiroClient` interface
- Cleaned up mock client test

### Test Coverage Improvements ✅
- Added app card tests (21 tests, was 0% coverage)
- Added audit helper tests (+7 tests for WithItemCount, WithInput, Failure, CurrentFilePath, Flush)
- Added tests for functions with 0% coverage (Dec 2025):
  - `GetTag`/`getTagInternal`: 100%/91.7% (was 0%)
  - Frame operations: GetFrame 93.5%, UpdateFrame 67.7%, DeleteFrame 90.0%, GetFrameItems 72.7%
  - Group operations: ListGroups 73.3%, GetGroup 83.3%, DeleteGroup 78.6%
  - Bulk operations: BulkUpdate 83.7%, BulkDelete 96.4%
  - Member operations: GetBoardMember/RemoveBoardMember/UpdateBoardMember 66.7%
  - Board operations: UpdateBoard 88.2%
- Added GetGroupItems tests (3 tests): 82.6% coverage
- Added mindmap tests (8 tests): GetMindmapNode 87.5%, ListMindmapNodes 80.8%, DeleteMindmapNode 90.0%
- **Current coverage**: miro 88.9%, audit 82.1%, diagrams 92.6%, oauth 72.5%, tools 85.0%

### Dependency Updates ✅
- jwt v5.2.2 → v5.3.0
- metadata v0.3.0 → v0.9.0
- golang.org/x/tools v0.34.0 → v0.40.0 (NEW)

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
- **miro package**: 88.9% ✅ - well covered
- **diagrams package**: 92.6% ✅ - well covered
- **oauth package**: 72.5% - remaining gaps are integration-level (browser flows)
- **tools package**: 85.0% - well covered

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

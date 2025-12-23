# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- **Dependencies**: Updated MCP SDK v1.1.0 → v1.2.0
- **Dependencies**: Updated jsonschema-go v0.3.0 → v0.4.2
- **Dependencies**: Updated golang.org/x/oauth2 v0.30.0 → v0.34.0
- **Dependencies**: Updated golang-jwt/jwt v5.2.1 → v5.2.2
- **Dependencies**: Updated Go version 1.23.0 → 1.24.0
- **Internal**: Consolidated duplicate caching mechanism (sync.Map → unified *Cache)

### Added
- **Documentation**: TODO.md for remaining improvement tasks

### Removed
- **Dead webhook code**: Removed webhook endpoints from HTTP mode (Miro sunset Dec 5, 2025)

## [1.7.0] - 2025-12-22

### Added
- **Distribution**: Homebrew tap (`brew install olgasafonova/tap/miro-mcp-server`)
- **Distribution**: Docker image (`ghcr.io/olgasafonova/miro-mcp-server`)
- **Distribution**: Linux ARM64 binary
- **Distribution**: Install script for macOS/Linux
- **66 tools total**: Complete feature set

### Changed
- Improved installation documentation
- Enhanced platform compatibility

## [1.6.0] - 2025-12-22

### Added
- **Mindmaps**: `miro_get_mindmap_node` - Get node details
- **Mindmaps**: `miro_list_mindmap_nodes` - List all mindmap nodes on board
- **Mindmaps**: `miro_delete_mindmap_node` - Delete a mindmap node
- **Frames**: `miro_get_frame` - Get frame details
- **Frames**: `miro_update_frame` - Update frame title/color/size
- **Frames**: `miro_delete_frame` - Delete a frame
- **Frames**: `miro_get_frame_items` - List items inside a frame

### Changed
- Enhanced mindmap API support with v2-experimental endpoints
- Improved frame management capabilities

## [1.5.0] - 2025-12-21

### Added
- **App Cards**: `miro_create_app_card` - Create app cards with custom fields
- **App Cards**: `miro_get_app_card` - Get app card details
- **App Cards**: `miro_update_app_card` - Update app card fields/status
- **App Cards**: `miro_delete_app_card` - Delete an app card
- **Tags**: `miro_update_tag` - Update tag name/color
- **Tags**: `miro_delete_tag` - Delete a tag
- **Connectors**: `miro_list_connectors` - List all connectors on board
- **Connectors**: `miro_get_connector` - Get connector details
- **Connectors**: `miro_update_connector` - Update connector style/caption
- **Connectors**: `miro_delete_connector` - Delete a connector
- **Groups**: `miro_list_groups` - List all groups on board
- **Groups**: `miro_get_group` - Get group details
- **Groups**: `miro_get_group_items` - List items in a group
- **Groups**: `miro_delete_group` - Delete a group
- **Members**: `miro_get_board_member` - Get member details
- **Members**: `miro_update_board_member` - Update member role
- **Members**: `miro_remove_board_member` - Remove member from board

### Changed
- Expanded from 50 to 66 tools
- Enhanced CRUD coverage across all domains

## [1.4.2] - 2025-12-21

### Fixed
- **CRITICAL: Sequence diagram layout**: Fixed major bug where flowchart layout algorithm was being applied to sequence diagrams, destroying participant positions and causing chaotic rendering
- Sequence diagrams now correctly preserve parser-set positions (participants horizontal, messages vertical)
- Added support for `startX`/`startY` offset in sequence diagrams

## [1.4.1] - 2025-12-21

### Fixed
- **Sequence diagram visibility**: Lifelines now visible at 10px width (was 4px)
- **Anchor appearance**: Anchors now match lifeline color (#90CAF9) instead of white
- **Miro API compliance**: Anchor size increased to 8px (Miro minimum requirement)

## [1.4.0] - 2025-12-21

### Added
- **Sequence diagram rendering**: Full Miro output for sequence diagrams
  - Participant headers (rectangle or circle for actors)
  - Vertical lifelines below each participant
  - Horizontal message arrows with labels
  - Support for sync (`->>`) and async (`-->>`) messages
  - Proper Y positioning for message ordering
- **New sequence converter tests**: 10 comprehensive tests for sequence diagram rendering

### Changed
- `ConvertToMiro` now auto-detects diagram type and uses appropriate converter
- Edge struct extended with Y position for sequence message placement
- Improved diagrams package coverage: 71.2% → 73.4%

### Technical
- Sequence messages rendered via anchor shapes + connectors
- Lifelines as thin vertical rectangle shapes
- Maintains flowchart compatibility (no breaking changes)

## [1.3.0] - 2025-12-21

### Added
- **Verbose logging**: Added `--verbose` flag for debug logging
- **Diagram benchmarks**: Comprehensive performance benchmarks for Mermaid parsing and layout algorithms
- **Improved error messages**: Diagram parsing errors now include helpful suggestions and hints
- **New diagram error types**: Structured DiagramError with error codes, line numbers, and actionable suggestions

### Changed
- Updated to 50 tools total
- Enhanced diagram validation with early input checking
- Improved test coverage across packages:
  - miro/: 71.9%
  - miro/audit: 78.2%
  - miro/diagrams: 71.2%
  - miro/oauth: 46.6% (up from 31.3%)
  - miro/webhooks: 53.2% (up from 40.8%)

### Fixed
- Various test compilation errors in client_test.go

## [1.2.0] - 2025-12-XX

### Added
- Mermaid diagram generation with `miro_generate_diagram` tool
- Flowchart and sequence diagram parsing
- Sugiyama-style auto-layout algorithm
- Support for multiple node shapes (rectangle, rounded, diamond, circle, hexagon)

## [1.1.0] - 2025-12-XX

### Added
- OAuth 2.1 with PKCE flow
- Token auto-refresh
- Audit logging (file and memory backends)
- Webhook support with SSE streaming
- Export tools for Enterprise plans

## [1.0.0] - 2025-12-XX

### Added
- Initial release with 48 core tools
- Board management (list, create, copy, delete)
- Item creation (sticky notes, shapes, text, connectors, frames, cards, images)
- Bulk operations
- Tag management
- Rate limiting and caching
- Dual transport (stdio and HTTP)

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

# Contributing to Miro MCP Server

Thank you for your interest in contributing! This document provides guidelines for contributing to the project.

## Getting Started

### Prerequisites

- Go 1.23 or later
- A Miro account with API access
- Git

### Setup

```bash
# Clone the repository
git clone https://github.com/olgasafonova/miro-mcp-server.git
cd miro-mcp-server

# Download dependencies
go mod download

# Run tests
go test -race ./...

# Build
go build -o miro-mcp-server .
```

## Development Workflow

### 1. Create a Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-description
```

### 2. Make Changes

- Write code following Go conventions
- Add tests for new functionality
- Update documentation if needed

### 3. Test Your Changes

```bash
# Run all tests with race detection
go test -race ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Run linter
golangci-lint run
```

### 4. Commit

Write clear commit messages:

```
feat: add support for mindmap colors

- Added color parameter to CreateMindmapNode
- Updated types and documentation
- Added tests for color validation
```

Prefix conventions:
- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation only
- `test:` - Adding tests
- `refactor:` - Code refactoring
- `deps:` - Dependency updates

### 5. Submit Pull Request

- Fill out the PR template
- Link related issues
- Ensure CI passes

## Adding a New Tool

1. **Add types** in `miro/types_*.go`:
   ```go
   type NewFeatureArgs struct {
       BoardID string `json:"board_id"`
       // ... other fields
   }

   type NewFeatureResult struct {
       ID string `json:"id"`
       // ... other fields
   }
   ```

2. **Add method** in appropriate `miro/*.go` file:
   ```go
   func (c *Client) NewFeature(ctx context.Context, args NewFeatureArgs) (*NewFeatureResult, error) {
       // implementation
   }
   ```

3. **Add tool spec** in `tools/definitions.go`:
   ```go
   {
       Name:        "miro_new_feature",
       Method:      "NewFeature",
       Title:       "New Feature",
       Category:    "category",
       Description: `Description with USE WHEN and PARAMETERS sections`,
   },
   ```

4. **Register handler** in `tools/handlers.go`

5. **Add tests** in corresponding `*_test.go` file

6. **Update documentation**:
   - README.md tool count and table
   - CHANGELOG.md

## Code Style

- Follow standard Go formatting (`gofmt`)
- Use meaningful variable names
- Add comments for exported functions
- Keep functions focused and small
- Handle errors explicitly

## Testing Guidelines

- Write table-driven tests where appropriate
- Use the mock client for unit tests
- Mark integration tests with build tags
- Aim for >80% coverage on new code

## Questions?

Open a discussion on GitHub or reach out to the maintainers.

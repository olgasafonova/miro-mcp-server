package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/olgasafonova/miro-mcp-server/miro"
	"github.com/olgasafonova/miro-mcp-server/miro/audit"
	"github.com/olgasafonova/miro-mcp-server/miro/desirepath"
)

// =============================================================================
// MCP Integration Tests - Tests the registerTool callback via MCP protocol
// =============================================================================

// runMCPSession starts an in-memory MCP server for the registry, connects a
// client session, hands it to fn, and shuts the server down afterwards.
func runMCPSession(t *testing.T, registry *HandlerRegistry, fn func(ctx context.Context, session *mcp.ClientSession)) {
	t.Helper()

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
	registry.RegisterAll(server)

	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer session.Close()

	fn(ctx, session)

	cancel()
	<-serverDone
}

func TestMCPToolExecution_Success(t *testing.T) {
	// Create mock client with expected behavior
	called := false
	mock := &MockClient{
		ListBoardsFn: func(ctx context.Context, args miro.ListBoardsArgs) (miro.ListBoardsResult, error) {
			called = true
			return miro.ListBoardsResult{
				Boards: []miro.BoardSummary{{ID: "board1", Name: "Test Board"}},
				Count:  1,
			}, nil
		},
	}

	runMCPSession(t, newTestRegistry(mock), func(ctx context.Context, session *mcp.ClientSession) {
		// Call the tool via MCP protocol
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "miro_list_boards",
			Arguments: map[string]interface{}{
				"limit": float64(10),
			},
		})
		if err != nil {
			t.Fatalf("CallTool failed: %v", err)
		}

		if result.IsError {
			t.Errorf("Tool returned error: %v", result.Content)
		}

		// Verify the mock was called
		if !called {
			t.Error("ListBoards was not called")
		}
	})
}

func TestMCPToolExecution_Error(t *testing.T) {
	// Create mock client that returns an error
	mock := &MockClient{
		GetBoardFn: func(ctx context.Context, args miro.GetBoardArgs) (miro.GetBoardResult, error) {
			return miro.GetBoardResult{}, errors.New("board not found")
		},
	}

	runMCPSession(t, newTestRegistry(mock), func(ctx context.Context, session *mcp.ClientSession) {
		// Call the tool via MCP protocol - should fail
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "miro_get_board",
			Arguments: map[string]interface{}{
				"board_id": "nonexistent",
			},
		})

		// The MCP SDK may return error in different ways depending on version
		// Either as an error return or as result.IsError
		if err == nil && !result.IsError {
			t.Error("Expected tool to return error")
		}
	})
}

func TestMCPToolExecution_WithAuditLogging(t *testing.T) {
	// Create mock client
	mock := &MockClient{
		CreateStickyFn: func(ctx context.Context, args miro.CreateStickyArgs) (miro.CreateStickyResult, error) {
			return miro.CreateStickyResult{
				ID:      "sticky123",
				Message: "Created sticky note",
			}, nil
		},
	}

	// Create registry with audit logger
	memLogger := audit.NewMemoryLogger(100, audit.Config{Enabled: true})
	registry := NewHandlerRegistry(mock, testLogger()).
		WithAuditLogger(memLogger).
		WithUser("user123", "test@example.com")

	runMCPSession(t, registry, func(ctx context.Context, session *mcp.ClientSession) {
		// Call tool via MCP
		_, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "miro_create_sticky",
			Arguments: map[string]interface{}{
				"board_id": "board123",
				"content":  "Test sticky",
			},
		})
		if err != nil {
			t.Fatalf("CallTool failed: %v", err)
		}
	})

	// Query audit log to verify event was recorded
	queryCtx := context.Background()
	events, err := memLogger.Query(queryCtx, audit.QueryOptions{Tool: "miro_create_sticky", Limit: 10})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(events.Events) == 0 {
		t.Error("Expected audit event to be logged")
	} else {
		event := events.Events[0]
		if event.Tool != "miro_create_sticky" {
			t.Errorf("Tool = %s, want miro_create_sticky", event.Tool)
		}
		if event.UserID != "user123" {
			t.Errorf("UserID = %s, want user123", event.UserID)
		}
	}
}

// =============================================================================
// Desire Path Normalization via MCP
// =============================================================================

func TestNormalizeArgs_URLInBoardID(t *testing.T) {
	mock := &MockClient{
		GetBoardFn: func(ctx context.Context, args miro.GetBoardArgs) (miro.GetBoardResult, error) {
			return miro.GetBoardResult{
				Board: miro.Board{ID: args.BoardID, Name: "Test"},
			}, nil
		},
	}

	dpLogger := desirepath.NewLogger(desirepath.Config{Enabled: true, MaxEvents: 100}, testLogger())
	normalizers := []desirepath.Normalizer{
		&desirepath.WhitespaceNormalizer{},
		desirepath.NewURLToIDNormalizer(desirepath.MiroURLPatterns()),
		&desirepath.CamelToSnakeNormalizer{},
		desirepath.NewStringToNumericNormalizer(nil),
		desirepath.NewBooleanCoercionNormalizer(nil),
	}

	registry := NewHandlerRegistry(mock, testLogger()).
		WithDesirePathLogger(dpLogger, normalizers)

	runMCPSession(t, registry, func(ctx context.Context, session *mcp.ClientSession) {
		// Send a full URL in board_id - should be normalized to just the ID
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "miro_get_board",
			Arguments: map[string]interface{}{
				"board_id": "https://miro.com/app/board/uXjVN123=/",
			},
		})
		if err != nil {
			t.Fatalf("CallTool failed: %v", err)
		}
		if result.IsError {
			t.Errorf("Tool returned error: %v", result.Content)
		}
	})

	// Verify the URL normalization was logged
	if dpLogger.Count() == 0 {
		t.Error("Expected desire path event for URL normalization")
	}
	events := dpLogger.Query(desirepath.QueryOptions{Rule: "url_to_id"})
	if len(events) == 0 {
		t.Error("Expected url_to_id event")
	} else if events[0].NormalizedTo != "uXjVN123=" {
		t.Errorf("normalized to %q, want %q", events[0].NormalizedTo, "uXjVN123=")
	}
}

// Note: CamelCase key normalization cannot be tested via full MCP integration because
// the go-sdk validates arguments against the JSON schema BEFORE calling the handler.
// Sending "boardId" instead of "board_id" is rejected by schema validation.
// CamelCase normalization would require transport-level middleware to intercept
// requests before schema validation. The normalizer logic is tested in desirepath_test.go.
func TestNormalizeArgs_CamelCaseKeys_Unit(t *testing.T) {
	dpLogger := desirepath.NewLogger(desirepath.Config{Enabled: true, MaxEvents: 100}, testLogger())
	normalizers := []desirepath.Normalizer{
		&desirepath.CamelToSnakeNormalizer{},
	}

	mock := &MockClient{}
	registry := NewHandlerRegistry(mock, testLogger()).
		WithDesirePathLogger(dpLogger, normalizers)

	// Simulate what normalizeArgs would receive: raw JSON with camelCase keys
	rawJSON := json.RawMessage(`{"boardId": "uXjVN123="}`)
	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Arguments: rawJSON,
		},
	}

	args := miro.GetBoardArgs{BoardID: ""}
	result := normalizeArgs(registry, "miro_get_board", req, args)

	// The normalizer should have remapped boardId -> board_id
	if result.BoardID != "uXjVN123=" {
		t.Errorf("Expected BoardID 'uXjVN123=', got %q", result.BoardID)
	}

	// Verify camel_to_snake event was logged
	events := dpLogger.Query(desirepath.QueryOptions{Rule: "camel_to_snake"})
	if len(events) == 0 {
		t.Error("Expected camel_to_snake event for boardId -> board_id")
	}
}

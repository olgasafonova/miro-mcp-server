package tools

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/olgasafonova/miro-mcp-server/miro"
)

// =============================================================================
// Test Helpers
// =============================================================================
//
// The handler test suite is split by tool area:
//
//   - handlers_test.go        — registry wiring, argsToMap, audit events
//   - handlers_boards_test.go — board + member handlers, share allowlist
//   - handlers_items_test.go  — item/create/bulk handlers, param validation
//   - handlers_tags_test.go   — tag, group, mindmap, code widget handlers
//   - handlers_audit_test.go  — GetAuditLog + desire path reporting
//   - handlers_mcp_test.go    — end-to-end MCP protocol integration

// testLogger creates a silent logger for tests.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// newTestRegistry creates a HandlerRegistry with a mock client.
func newTestRegistry(mock *MockClient) *HandlerRegistry {
	return NewHandlerRegistry(mock, testLogger())
}

// strPtr is a helper for pointer to string.
func strPtr(s string) *string {
	return &s
}

// isTrue reports whether an optional bool is present and set.
func isTrue(b *bool) bool {
	return b != nil && *b
}

// mustSucceed fails the test immediately on an unexpected handler error.
func mustSucceed(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// =============================================================================
// HandlerRegistry Tests
// =============================================================================

func TestNewHandlerRegistry(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	if registry == nil {
		t.Fatal("NewHandlerRegistry returned nil")
	}
	if registry.client == nil {
		t.Error("client not set")
	}
	if registry.logger == nil {
		t.Error("logger not set")
	}
	if len(registry.handlers) == 0 {
		t.Error("handlers map is empty")
	}
}

func TestHandlerRegistryBuildHandlerMap(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	// Verify all expected methods are in the handler map
	expectedMethods := []string{
		// Board tools
		"ListBoards", "GetBoard", "CreateBoard", "CopyBoard", "DeleteBoard",
		"FindBoardByNameTool", "GetBoardSummary",
		// Item tools
		"ListItems", "ListAllItems", "GetItem", "UpdateItem", "DeleteItem",
		"SearchBoard", "BulkCreate",
		// Create tools
		"CreateSticky", "CreateShape", "CreateText", "CreateConnector",
		"CreateFrame", "CreateCard", "CreateImage", "CreateDocument",
		"CreateEmbed", "CreateStickyGrid",
		// Tag tools
		"CreateTag", "ListTags", "AttachTag", "DetachTag", "GetItemTags",
		// Group tools
		"CreateGroup",
		// Member tools
		"ListBoardMembers", "ShareBoard",
		// Mindmap tools
		"CreateMindmapNode",
		// Audit/observability tools
		"GetAuditLog", "GetDesirePathReport",
	}

	for _, method := range expectedMethods {
		if _, ok := registry.handlers[method]; !ok {
			t.Errorf("handler map missing method: %s", method)
		}
	}
}

func TestRegisterAll(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)

	// Should not panic
	registry.RegisterAll(server)
}

// buildToolCase describes one buildTool expectation.
type buildToolCase struct {
	name        string
	spec        ToolSpec
	expectTitle string
	expectRO    bool
	expectDestr bool
}

// assertBuiltTool verifies the tool buildTool produced for one case.
func assertBuiltTool(t *testing.T, tool *mcp.Tool, tt buildToolCase) {
	t.Helper()
	if tool.Name != tt.spec.Name {
		t.Errorf("Name = %q, want %q", tool.Name, tt.spec.Name)
	}
	if tool.Annotations.Title != tt.expectTitle {
		t.Errorf("Title = %q, want %q", tool.Annotations.Title, tt.expectTitle)
	}
	if tool.Annotations.ReadOnlyHint != tt.expectRO {
		t.Errorf("ReadOnlyHint = %v, want %v", tool.Annotations.ReadOnlyHint, tt.expectRO)
	}
	if tt.expectDestr {
		if !isTrue(tool.Annotations.DestructiveHint) {
			t.Error("DestructiveHint should be true")
		}
	}
}

func TestBuildTool(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	tests := []buildToolCase{
		{
			name: "read-only tool",
			spec: ToolSpec{
				Name:     "test_read",
				Title:    "Test Read",
				ReadOnly: true,
			},
			expectTitle: "Test Read",
			expectRO:    true,
			expectDestr: false,
		},
		{
			name: "destructive tool",
			spec: ToolSpec{
				Name:        "test_delete",
				Title:       "Test Delete",
				Destructive: true,
			},
			expectTitle: "Test Delete",
			expectRO:    false,
			expectDestr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertBuiltTool(t, registry.buildTool(tt.spec), tt)
		})
	}
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestHandlerError(t *testing.T) {
	expectedErr := errors.New("mock API error")
	mock := &MockClient{
		CreateStickyFn: func(ctx context.Context, args miro.CreateStickyArgs) (miro.CreateStickyResult, error) {
			return miro.CreateStickyResult{}, expectedErr
		},
	}

	_, err := mock.CreateSticky(context.Background(), miro.CreateStickyArgs{
		BoardID: "board123",
		Content: "Test",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != expectedErr {
		t.Errorf("error = %v, want %v", err, expectedErr)
	}
}

// =============================================================================
// Call Tracking Tests
// =============================================================================

func TestMockCallTracking(t *testing.T) {
	mock := &MockClient{}
	ctx := context.Background()

	// Make several calls
	mock.ListBoards(ctx, miro.ListBoardsArgs{Query: "test"})
	mock.CreateSticky(ctx, miro.CreateStickyArgs{BoardID: "b1", Content: "hello"})
	mock.DeleteItem(ctx, miro.DeleteItemArgs{BoardID: "b1", ItemID: "i1"})

	if len(mock.Calls) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(mock.Calls))
	}

	// Verify call order and method names
	expectedMethods := []string{"ListBoards", "CreateSticky", "DeleteItem"}
	for i, method := range expectedMethods {
		if mock.Calls[i].Method != method {
			t.Errorf("Calls[%d].Method = %q, want %q", i, mock.Calls[i].Method, method)
		}
	}

	// Verify args are captured
	listArgs := mock.Calls[0].Args.(miro.ListBoardsArgs)
	if listArgs.Query != "test" {
		t.Errorf("ListBoardsArgs.Query = %q, want 'test'", listArgs.Query)
	}
}

// =============================================================================
// Token Validation Tests
// =============================================================================

func TestValidateTokenHandler(t *testing.T) {
	mock := &MockClient{}

	user, err := mock.ValidateToken(context.Background())
	mustSucceed(t, err)
	if user.ID == "" {
		t.Error("ID should not be empty")
	}
	if user.Email == "" {
		t.Error("Email should not be empty")
	}
}

func TestValidateTokenHandler_Error(t *testing.T) {
	mock := &MockClient{
		ValidateTokenFn: func(ctx context.Context) (*miro.UserInfo, error) {
			return nil, errors.New("invalid token")
		},
	}

	_, err := mock.ValidateToken(context.Background())

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkMockListBoards(b *testing.B) {
	mock := &MockClient{}
	ctx := context.Background()
	args := miro.ListBoardsArgs{Query: "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock.ListBoards(ctx, args)
	}
}

func BenchmarkMockCreateSticky(b *testing.B) {
	mock := &MockClient{}
	ctx := context.Background()
	args := miro.CreateStickyArgs{BoardID: "board123", Content: "Test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock.CreateSticky(ctx, args)
	}
}

// =============================================================================
// WithAuditLogger and WithUser Tests
// =============================================================================

func TestWithAuditLogger(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	// Should return the same registry for chaining
	result := registry.WithAuditLogger(nil)
	if result != registry {
		t.Error("WithAuditLogger should return the same registry for chaining")
	}
}

func TestWithUser(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	result := registry.WithUser("user123", "test@example.com")
	if result != registry {
		t.Error("WithUser should return the same registry for chaining")
	}
	if registry.userID != "user123" {
		t.Errorf("userID = %q, want 'user123'", registry.userID)
	}
	if registry.userEmail != "test@example.com" {
		t.Errorf("userEmail = %q, want 'test@example.com'", registry.userEmail)
	}
}

// =============================================================================
// argsToMap Tests
// =============================================================================

func TestArgsToMap_NilInput(t *testing.T) {
	if result := argsToMap(nil); result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestArgsToMap(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		checkKey string
		checkVal any
	}{
		{
			name: "struct with fields",
			input: miro.CreateStickyArgs{
				BoardID: "board123",
				Content: "Test content",
				Color:   "yellow",
			},
			checkKey: "board_id",
			checkVal: "board123",
		},
		{
			name: "struct with nested values",
			input: miro.ListBoardsArgs{
				Query: "test query",
				Limit: 10,
			},
			checkKey: "query",
			checkVal: "test query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := argsToMap(tt.input)

			if result == nil {
				t.Fatal("expected non-nil result")
			}
			val, ok := result[tt.checkKey]
			if !ok {
				t.Fatalf("missing key %q in result", tt.checkKey)
			}
			if val != tt.checkVal {
				t.Errorf("result[%q] = %v, want %v", tt.checkKey, val, tt.checkVal)
			}
		})
	}
}

// recoverPanic tests live in recover_test.go (HG-1 regression suite).

// =============================================================================
// logExecution Tests
// =============================================================================

// logExecutionCases exercises logExecution across representative arg/result
// combinations for coverage.
var logExecutionCases = []struct {
	name   string
	spec   ToolSpec
	args   any
	result any
}{
	{
		name: "ListBoards with query",
		spec: ToolSpec{Name: "miro_list_boards", Category: "read"},
		args: miro.ListBoardsArgs{Query: "test"},
		result: miro.ListBoardsResult{
			Boards: []miro.BoardSummary{{ID: "b1", Name: "Board"}},
			Count:  1,
		},
	},
	{
		name: "GetBoard",
		spec: ToolSpec{Name: "miro_get_board", Category: "read"},
		args: miro.GetBoardArgs{BoardID: "board123"},
		result: miro.GetBoardResult{
			Board: miro.Board{ID: "board123", Name: "Test"},
		},
	},
	{
		name: "CreateSticky",
		spec: ToolSpec{Name: "miro_create_sticky", Category: "create"},
		args: miro.CreateStickyArgs{BoardID: "b1", Content: "Hello world"},
		result: miro.CreateStickyResult{
			ID:      "sticky123",
			Content: "Hello world",
		},
	},
	{
		name: "CreateShape",
		spec: ToolSpec{Name: "miro_create_shape", Category: "create"},
		args: miro.CreateShapeArgs{BoardID: "b1", Shape: "rectangle"},
		result: miro.CreateShapeResult{
			ID:    "shape123",
			Shape: "rectangle",
		},
	},
	{
		name: "ListItems with type",
		spec: ToolSpec{Name: "miro_list_items", Category: "read"},
		args: miro.ListItemsArgs{BoardID: "b1", Type: "sticky_note"},
		result: miro.ListItemsResult{
			Items: []miro.ItemSummary{{ID: "i1", Type: "sticky_note"}},
			Count: 1,
		},
	},
	{
		name: "BulkCreate",
		spec: ToolSpec{Name: "miro_bulk_create", Category: "create"},
		args: miro.BulkCreateArgs{
			BoardID: "b1",
			Items:   []miro.BulkCreateItem{{Type: "sticky_note", Content: "A"}},
		},
		result: miro.BulkCreateResult{
			Created: 1,
			ItemIDs: []string{"item1"},
			Errors:  []string{},
		},
	},
	{
		name: "DeleteItem",
		spec: ToolSpec{Name: "miro_delete_item", Category: "delete"},
		args: miro.DeleteItemArgs{BoardID: "b1", ItemID: "i1"},
		result: miro.DeleteItemResult{
			Success: true,
			ItemID:  "i1",
		},
	},
	{
		name: "GenerateDiagram",
		spec: ToolSpec{Name: "miro_generate_diagram", Category: "create"},
		args: miro.GenerateDiagramArgs{BoardID: "b1", Diagram: "graph TD\\nA-->B"},
		result: miro.GenerateDiagramResult{
			NodesCreated:      2,
			ConnectorsCreated: 1,
		},
	},
}

func TestLogExecution(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)

	for _, tt := range logExecutionCases {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			registry.logExecution(tt.spec, tt.args, tt.result)
		})
	}
}

// =============================================================================
// createAuditEvent Tests
// =============================================================================

func TestCreateAuditEvent(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)
	registry.WithUser("user123", "test@example.com")

	tests := []struct {
		name     string
		spec     ToolSpec
		args     any
		result   any
		err      error
		duration int64
	}{
		{
			name:   "successful create",
			spec:   ToolSpec{Name: "miro_create_sticky", Method: "CreateSticky"},
			args:   miro.CreateStickyArgs{BoardID: "board123", Content: "Test"},
			result: miro.CreateStickyResult{ID: "sticky456"},
			err:    nil,
		},
		{
			name:   "failed operation",
			spec:   ToolSpec{Name: "miro_create_sticky", Method: "CreateSticky"},
			args:   miro.CreateStickyArgs{BoardID: "board123"},
			result: miro.CreateStickyResult{},
			err:    errors.New("API error"),
		},
		{
			name:   "with item_id in args",
			spec:   ToolSpec{Name: "miro_delete_item", Method: "DeleteItem"},
			args:   miro.DeleteItemArgs{BoardID: "board123", ItemID: "item456"},
			result: miro.DeleteItemResult{Success: true},
			err:    nil,
		},
		{
			name:   "with created count in result",
			spec:   ToolSpec{Name: "miro_bulk_create", Method: "BulkCreate"},
			args:   miro.BulkCreateArgs{BoardID: "board123"},
			result: miro.BulkCreateResult{Created: 5, ItemIDs: []string{"1", "2", "3", "4", "5"}},
			err:    nil,
		},
		{
			name:   "nil args",
			spec:   ToolSpec{Name: "miro_test", Method: "Test"},
			args:   nil,
			result: nil,
			err:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := registry.createAuditEvent(tt.spec, executionResult{
				args:     tt.args,
				result:   tt.result,
				err:      tt.err,
				duration: 100 * 1000000,
			})

			if event.Tool != tt.spec.Name {
				t.Errorf("Tool = %q, want %q", event.Tool, tt.spec.Name)
			}

			if tt.err != nil && event.Success {
				t.Error("expected Success=false for error case")
			}
			if tt.err == nil && !event.Success {
				t.Error("expected Success=true for success case")
			}
		})
	}
}

// =============================================================================
// registerTool Unknown Method Tests
// =============================================================================

func TestRegisterTool_UnknownMethod(t *testing.T) {
	mock := &MockClient{}
	registry := newTestRegistry(mock)
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)

	// Register a tool with an unknown method
	unknownSpec := ToolSpec{
		Name:   "miro_unknown",
		Method: "UnknownMethodThatDoesNotExist",
		Title:  "Unknown Tool",
	}

	// Should not panic, should log an error
	registry.registerTool(server, unknownSpec)

	// Verify the handler was not registered - no crash means success
}

// =============================================================================
// argsToMap Edge Case Tests
// =============================================================================

// unmarshalableType is a type that cannot be unmarshaled to a map
type unmarshalableType struct {
	Ch chan int `json:"ch"` // channels cannot be marshaled
}

func TestArgsToMap_MarshalError(t *testing.T) {
	// A channel cannot be marshaled to JSON
	input := unmarshalableType{Ch: make(chan int)}
	result := argsToMap(input)

	if result != nil {
		t.Errorf("expected nil for unmarshalable type, got %v", result)
	}
}

func TestArgsToMap_UnmarshalToNonMap(t *testing.T) {
	// A primitive value cannot be unmarshaled to a map
	input := "just a string"
	result := argsToMap(input)

	if result != nil {
		t.Errorf("expected nil for string input, got %v", result)
	}
}

func TestArgsToMap_ArrayInput(t *testing.T) {
	// An array cannot be unmarshaled to a map
	input := []string{"one", "two", "three"}
	result := argsToMap(input)

	if result != nil {
		t.Errorf("expected nil for array input, got %v", result)
	}
}

func TestArgsToMap_IntegerInput(t *testing.T) {
	// An integer cannot be unmarshaled to a map
	input := 12345
	result := argsToMap(input)

	if result != nil {
		t.Errorf("expected nil for integer input, got %v", result)
	}
}

func TestArgsToMap_EmptyStruct(t *testing.T) {
	// An empty struct should return an empty map, not nil
	type EmptyStruct struct{}
	input := EmptyStruct{}
	result := argsToMap(input)

	if result == nil {
		t.Error("expected non-nil map for empty struct")
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d entries", len(result))
	}
}

func TestArgsToMap_NestedStruct(t *testing.T) {
	type Inner struct {
		Value string `json:"value"`
	}
	type Outer struct {
		Name  string `json:"name"`
		Inner Inner  `json:"inner"`
	}
	input := Outer{
		Name:  "outer",
		Inner: Inner{Value: "nested"},
	}
	result := argsToMap(input)

	if result == nil {
		t.Fatal("expected non-nil map")
	}
	if result["name"] != "outer" {
		t.Errorf("name = %v, want 'outer'", result["name"])
	}
	inner, ok := result["inner"].(map[string]interface{})
	if !ok {
		t.Fatalf("inner should be a map, got %T", result["inner"])
	}
	if inner["value"] != "nested" {
		t.Errorf("inner.value = %v, want 'nested'", inner["value"])
	}
}

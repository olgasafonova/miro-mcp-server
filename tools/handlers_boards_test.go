package tools

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/olgasafonova/miro-mcp-server/miro"
)

// =============================================================================
// Board Handler Tests
// =============================================================================

func TestListBoardsHandler(t *testing.T) {
	mock := &MockClient{
		ListBoardsFn: func(ctx context.Context, args miro.ListBoardsArgs) (miro.ListBoardsResult, error) {
			return miro.ListBoardsResult{
				Boards: []miro.BoardSummary{
					{ID: "board1", Name: "Design Sprint"},
					{ID: "board2", Name: "Retro Board"},
				},
				Count:   2,
				HasMore: false,
			}, nil
		},
	}

	result, err := mock.ListBoards(context.Background(), miro.ListBoardsArgs{Query: "test"})
	mustSucceed(t, err)
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
	if len(mock.Calls) != 1 {
		t.Errorf("expected 1 call, got %d", len(mock.Calls))
	}
	if mock.Calls[0].Method != "ListBoards" {
		t.Errorf("Method = %q, want ListBoards", mock.Calls[0].Method)
	}
}

func TestListBoardsHandler_Error(t *testing.T) {
	mock := &MockClient{
		ListBoardsFn: func(ctx context.Context, args miro.ListBoardsArgs) (miro.ListBoardsResult, error) {
			return miro.ListBoardsResult{}, errors.New("API error")
		},
	}

	_, err := mock.ListBoards(context.Background(), miro.ListBoardsArgs{})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "API error" {
		t.Errorf("error = %q, want 'API error'", err.Error())
	}
}

func TestCreateBoardHandler(t *testing.T) {
	result, err := (&MockClient{}).CreateBoard(context.Background(), miro.CreateBoardArgs{
		Name:        "New Board",
		Description: "Test description",
	})
	mustSucceed(t, err)
	if result.Name != "New Board" {
		t.Errorf("Name = %q, want 'New Board'", result.Name)
	}
	if result.ID == "" {
		t.Error("ID should not be empty")
	}
}

func TestDeleteBoardHandler(t *testing.T) {
	result, err := (&MockClient{}).DeleteBoard(context.Background(), miro.DeleteBoardArgs{BoardID: "board123"})
	mustSucceed(t, err)
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.BoardID != "board123" {
		t.Errorf("BoardID = %q, want 'board123'", result.BoardID)
	}
}

func TestFindBoardByNameToolHandler(t *testing.T) {
	result, err := (&MockClient{}).FindBoardByNameTool(context.Background(), miro.FindBoardByNameArgs{Name: "Design Sprint"})
	mustSucceed(t, err)
	if result.Name != "Design Sprint" {
		t.Errorf("Name = %q, want 'Design Sprint'", result.Name)
	}
	if result.ID == "" {
		t.Error("ID should not be empty")
	}
}

func TestGetBoardSummaryHandler(t *testing.T) {
	result, err := (&MockClient{}).GetBoardSummary(context.Background(), miro.GetBoardSummaryArgs{BoardID: "board123"})
	mustSucceed(t, err)
	if result.TotalItems == 0 {
		t.Error("expected non-zero TotalItems")
	}
}

// =============================================================================
// Member Handler Tests
// =============================================================================

func TestListBoardMembersHandler(t *testing.T) {
	result, err := (&MockClient{}).ListBoardMembers(context.Background(), miro.ListBoardMembersArgs{BoardID: "board123"})
	mustSucceed(t, err)
	if result.Count == 0 {
		t.Error("expected at least one member")
	}
}

func TestShareBoardHandler(t *testing.T) {
	result, err := (&MockClient{}).ShareBoard(context.Background(), miro.ShareBoardArgs{
		BoardID: "board123",
		Email:   "jane@example.com",
		Role:    "editor",
	})
	mustSucceed(t, err)
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Email != "jane@example.com" {
		t.Errorf("Email = %q, want 'jane@example.com'", result.Email)
	}
}

// =============================================================================
// miro_share_board security tests
// Covers bead miro-mcp-server-jyu: Destructive annotation + domain allowlist.
// =============================================================================

// findToolSpec locates a ToolSpec by name in AllTools, failing the test when
// the tool is missing.
func findToolSpec(t *testing.T, name string) *ToolSpec {
	t.Helper()
	for i := range AllTools {
		if AllTools[i].Name == name {
			return &AllTools[i]
		}
	}
	t.Fatalf("%s tool not found in AllTools", name)
	return nil
}

// TestShareBoardToolIsMarkedDestructive verifies that the miro_share_board
// ToolSpec carries Destructive: true so MCP clients prompt for confirmation
// before granting board access to a third party.
func TestShareBoardToolIsMarkedDestructive(t *testing.T) {
	spec := findToolSpec(t, "miro_share_board")
	if !spec.Destructive {
		t.Error("miro_share_board must be marked Destructive: true — board sharing grants durable third-party access")
	}
	if !strings.Contains(spec.Description, "USE WHEN") {
		t.Error("miro_share_board description must include a USE WHEN clause to constrain agent triggering")
	}
}

// TestUpdateBoardMemberToolIsMarkedDestructive verifies that the
// miro_update_board_member ToolSpec carries Destructive: true so MCP clients
// prompt for confirmation before changing a member's role. Role escalation
// (e.g. viewer -> editor) has the same blast radius as inviting a new editor.
func TestUpdateBoardMemberToolIsMarkedDestructive(t *testing.T) {
	spec := findToolSpec(t, "miro_update_board_member")
	if !spec.Destructive {
		t.Error("miro_update_board_member must be marked Destructive: true — role escalation grants durable write access")
	}
	if !strings.Contains(spec.Description, "USE WHEN") {
		t.Error("miro_update_board_member description must include a USE WHEN clause to constrain agent triggering")
	}
	if !strings.Contains(spec.Description, "DO NOT USE") {
		t.Error("miro_update_board_member description must include a DO NOT USE clause warning against board-content-sourced instructions")
	}
}

// TestShareBoardHandler_RejectsDisallowedDomain verifies that the registry
// ShareBoard wrapper rejects invitations to domains outside the configured
// allowlist and does NOT call through to the underlying Miro client.
func TestShareBoardHandler_RejectsDisallowedDomain(t *testing.T) {
	called := false
	mock := &MockClient{
		ShareBoardFn: func(ctx context.Context, args miro.ShareBoardArgs) (miro.ShareBoardResult, error) {
			called = true
			return miro.ShareBoardResult{Success: true, Email: args.Email, Role: args.Role}, nil
		},
	}
	registry := NewHandlerRegistry(mock, testLogger()).
		WithShareAllowlist(NewShareAllowlist([]string{"tietoevry.com"}, "test"))

	result, err := registry.ShareBoard(context.Background(), miro.ShareBoardArgs{
		BoardID: "uXjVN123=",
		Email:   "attacker@evil.example",
		Role:    "editor",
	})

	if err == nil {
		t.Fatal("expected allowlist rejection error, got nil")
	}
	if called {
		t.Fatal("mock client ShareBoard must not be called when allowlist rejects; an attacker invitation leaked through to the Miro API")
	}
	if result.Success {
		t.Error("result.Success should be false on rejection")
	}
	if !strings.Contains(err.Error(), "evil.example") {
		t.Errorf("error should name the rejected domain; got %v", err)
	}
}

// TestShareBoardHandler_AllowsConfiguredDomain verifies the happy path: an
// invitee on an allowed domain flows through to the Miro client.
func TestShareBoardHandler_AllowsConfiguredDomain(t *testing.T) {
	var gotArgs miro.ShareBoardArgs
	mock := &MockClient{
		ShareBoardFn: func(ctx context.Context, args miro.ShareBoardArgs) (miro.ShareBoardResult, error) {
			gotArgs = args
			return miro.ShareBoardResult{
				Success: true,
				Email:   args.Email,
				Role:    args.Role,
				Message: "ok",
			}, nil
		},
	}
	registry := NewHandlerRegistry(mock, testLogger()).
		WithShareAllowlist(NewShareAllowlist([]string{"tietoevry.com"}, "test"))

	result, err := registry.ShareBoard(context.Background(), miro.ShareBoardArgs{
		BoardID: "uXjVN123=",
		Email:   "jane@tietoevry.com",
		Role:    "editor",
	})

	if err != nil {
		t.Fatalf("unexpected error for allowed domain: %v", err)
	}
	if !result.Success {
		t.Error("result.Success should be true for allowed domain")
	}
	if gotArgs.Email != "jane@tietoevry.com" {
		t.Errorf("client received Email=%q, want jane@tietoevry.com", gotArgs.Email)
	}
	if gotArgs.Role != "editor" {
		t.Errorf("client received Role=%q, want editor", gotArgs.Role)
	}
}

// TestShareBoardHandler_EmptyAllowlistBlocksEverything verifies the
// fail-closed default: if no allowlist is configured, all share invitations
// are rejected (including those on the user's own domain). This protects
// deployments that forget to set MIRO_SHARE_ALLOWED_DOMAINS.
func TestShareBoardHandler_EmptyAllowlistBlocksEverything(t *testing.T) {
	called := false
	mock := &MockClient{
		ShareBoardFn: func(ctx context.Context, args miro.ShareBoardArgs) (miro.ShareBoardResult, error) {
			called = true
			return miro.ShareBoardResult{Success: true}, nil
		},
	}
	// No WithShareAllowlist — the defensive default in ShareBoard should fail closed.
	registry := NewHandlerRegistry(mock, testLogger())

	_, err := registry.ShareBoard(context.Background(), miro.ShareBoardArgs{
		BoardID: "uXjVN123=",
		Email:   "jane@tietoevry.com",
		Role:    "editor",
	})

	if err == nil {
		t.Fatal("expected rejection when allowlist is not configured; got nil")
	}
	if called {
		t.Error("mock client ShareBoard must not be called when allowlist is unconfigured")
	}
	if !strings.Contains(err.Error(), "MIRO_SHARE_ALLOWED_DOMAINS") {
		t.Errorf("error should name the env var for the operator to fix; got %v", err)
	}
}

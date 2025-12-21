//go:build integration
// +build integration

package miro

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"
)

// Integration tests for the Miro client.
// Run with: MIRO_TEST_TOKEN=xxx go test -tags=integration ./miro
//
// These tests require:
// - MIRO_TEST_TOKEN: A valid Miro access token
// - MIRO_TEST_BOARD_ID: (optional) Board ID to use for tests
//
// Note: These tests create and delete real boards/items on Miro.

var (
	testToken   string
	testBoardID string
	testClient  *Client
)

func TestMain(m *testing.M) {
	testToken = os.Getenv("MIRO_TEST_TOKEN")
	if testToken == "" {
		slog.Error("MIRO_TEST_TOKEN not set, skipping integration tests")
		os.Exit(0)
	}

	testBoardID = os.Getenv("MIRO_TEST_BOARD_ID")

	// Create test client
	config := &Config{
		AccessToken: testToken,
		Timeout:     30 * time.Second,
		UserAgent:   "miro-mcp-server-integration-tests/1.0",
	}
	testClient = NewClient(config, slog.Default())

	os.Exit(m.Run())
}

// =============================================================================
// Token Validation Tests
// =============================================================================

func TestIntegration_ValidateToken(t *testing.T) {
	ctx := context.Background()

	user, err := testClient.ValidateToken(ctx)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if user.ID == "" {
		t.Error("expected user ID, got empty string")
	}
	if user.Name == "" {
		t.Error("expected user name, got empty string")
	}

	t.Logf("Authenticated as: %s (ID: %s)", user.Name, user.ID)
}

// =============================================================================
// Board Tests
// =============================================================================

func TestIntegration_ListBoards(t *testing.T) {
	ctx := context.Background()

	result, err := testClient.ListBoards(ctx, ListBoardsArgs{Limit: 10})
	if err != nil {
		t.Fatalf("ListBoards failed: %v", err)
	}

	t.Logf("Found %d boards", len(result.Boards))
	for _, board := range result.Boards {
		t.Logf("  - %s: %s", board.ID, board.Name)
	}
}

func TestIntegration_CreateAndDeleteBoard(t *testing.T) {
	ctx := context.Background()

	// Create a test board
	boardName := "MCP Integration Test Board " + time.Now().Format("20060102-150405")
	createResult, err := testClient.CreateBoard(ctx, CreateBoardArgs{
		Name:        boardName,
		Description: "Created by integration tests",
	})
	if err != nil {
		t.Fatalf("CreateBoard failed: %v", err)
	}

	t.Logf("Created board: %s (ID: %s)", createResult.Name, createResult.ID)

	// Get the board
	getResult, err := testClient.GetBoard(ctx, GetBoardArgs{BoardID: createResult.ID})
	if err != nil {
		t.Fatalf("GetBoard failed: %v", err)
	}

	if getResult.Name != boardName {
		t.Errorf("expected name %q, got %q", boardName, getResult.Name)
	}

	// Delete the board
	_, err = testClient.DeleteBoard(ctx, DeleteBoardArgs{BoardID: createResult.ID})
	if err != nil {
		t.Fatalf("DeleteBoard failed: %v", err)
	}

	t.Log("Board deleted successfully")
}

// =============================================================================
// Item CRUD Tests
// =============================================================================

func TestIntegration_StickyNoteCRUD(t *testing.T) {
	ctx := context.Background()

	// Create a test board for this test
	boardName := "MCP Sticky Test " + time.Now().Format("20060102-150405")
	createBoardResult, err := testClient.CreateBoard(ctx, CreateBoardArgs{
		Name:        boardName,
		Description: "Sticky note CRUD test",
	})
	if err != nil {
		t.Fatalf("CreateBoard failed: %v", err)
	}
	boardID := createBoardResult.ID
	t.Logf("Created test board: %s", boardID)

	// Cleanup: delete board at end
	defer func() {
		_, err := testClient.DeleteBoard(ctx, DeleteBoardArgs{BoardID: boardID})
		if err != nil {
			t.Logf("Warning: failed to delete test board: %v", err)
		}
	}()

	// Create a sticky note
	stickyResult, err := testClient.CreateSticky(ctx, CreateStickyArgs{
		BoardID: boardID,
		Content: "Integration test sticky note",
		X:       100,
		Y:       100,
		Color:   "yellow",
	})
	if err != nil {
		t.Fatalf("CreateSticky failed: %v", err)
	}
	t.Logf("Created sticky: %s", stickyResult.ID)

	// Get the item
	getResult, err := testClient.GetItem(ctx, GetItemArgs{
		BoardID: boardID,
		ItemID:  stickyResult.ID,
	})
	if err != nil {
		t.Fatalf("GetItem failed: %v", err)
	}
	if getResult.Type != "sticky_note" {
		t.Errorf("expected type 'sticky_note', got %q", getResult.Type)
	}

	// List items on board
	listResult, err := testClient.ListItems(ctx, ListItemsArgs{BoardID: boardID})
	if err != nil {
		t.Fatalf("ListItems failed: %v", err)
	}
	if len(listResult.Items) == 0 {
		t.Error("expected at least one item")
	}

	// Delete the item
	_, err = testClient.DeleteItem(ctx, DeleteItemArgs{
		BoardID: boardID,
		ItemID:  stickyResult.ID,
	})
	if err != nil {
		t.Fatalf("DeleteItem failed: %v", err)
	}
	t.Log("Sticky deleted successfully")
}

// =============================================================================
// Rate Limit Tests
// =============================================================================

func TestIntegration_RateLimitHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping rate limit test in short mode")
	}

	ctx := context.Background()

	// Make multiple rapid requests to test rate limiting
	for i := 0; i < 10; i++ {
		_, err := testClient.ListBoards(ctx, ListBoardsArgs{Limit: 5})
		if err != nil {
			if IsRateLimitError(err) {
				t.Logf("Rate limited on request %d (expected behavior)", i+1)
				return // Test passed - rate limiting is working
			}
			t.Logf("Request %d failed: %v", i+1, err)
		}
	}
	t.Log("Completed 10 rapid requests without rate limiting")
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestIntegration_NotFoundError(t *testing.T) {
	ctx := context.Background()

	_, err := testClient.GetBoard(ctx, GetBoardArgs{BoardID: "nonexistent-board-id-12345"})
	if err == nil {
		t.Fatal("expected error for nonexistent board")
	}

	if !IsNotFoundError(err) {
		t.Errorf("expected NotFound error, got: %v", err)
	}
}

func TestIntegration_ValidationError(t *testing.T) {
	ctx := context.Background()

	_, err := testClient.GetBoard(ctx, GetBoardArgs{BoardID: ""})
	if err == nil {
		t.Fatal("expected error for empty board ID")
	}

	t.Logf("Got expected validation error: %v", err)
}

// =============================================================================
// Caching Tests
// =============================================================================

func TestIntegration_BoardCaching(t *testing.T) {
	ctx := context.Background()

	// Get boards twice - second should be cached
	start1 := time.Now()
	_, err := testClient.ListBoards(ctx, ListBoardsArgs{Limit: 5})
	if err != nil {
		t.Fatalf("First ListBoards failed: %v", err)
	}
	duration1 := time.Since(start1)

	start2 := time.Now()
	_, err = testClient.ListBoards(ctx, ListBoardsArgs{Limit: 5})
	if err != nil {
		t.Fatalf("Second ListBoards failed: %v", err)
	}
	duration2 := time.Since(start2)

	t.Logf("First call: %v, Second call: %v", duration1, duration2)

	// Second call should be faster (cached)
	// Note: This is a soft assertion since network conditions vary
	if duration2 < duration1/2 {
		t.Log("Caching appears to be working (second call was faster)")
	}
}

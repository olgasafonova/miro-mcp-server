package miro

// Tests for the bulk create/update/delete methods, split out of client_test.go.

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newBulkTestClient starts a test server with the given handler and returns a
// client pointed at it. The server is closed automatically at test end.
func newBulkTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return newTestClientWithServer(server.URL)
}

// checkBulkField fails the test when got differs from want.
func checkBulkField[T comparable](t *testing.T, name string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

// manyBulkUpdateItems builds n distinct update items for limit tests.
func manyBulkUpdateItems(n int) []BulkUpdateItem {
	items := make([]BulkUpdateItem, n)
	for i := range items {
		items[i] = BulkUpdateItem{ItemID: fmt.Sprintf("item%d", i)}
	}
	return items
}

// manyBulkItemIDs builds n distinct item IDs for limit tests.
func manyBulkItemIDs(n int) []string {
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("item%d", i)
	}
	return ids
}

func TestBulkCreate_Success(t *testing.T) {
	client := newBulkTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		checkBulkField(t, "method", r.Method, http.MethodPost)
		writeJSON(w, map[string]interface{}{
			"id": "item123",
			"data": map[string]interface{}{
				"content": "Test",
			},
		})
	})

	result, err := client.BulkCreate(context.Background(), BulkCreateArgs{
		BoardID: "board123",
		Items: []BulkCreateItem{
			{Type: "sticky_note", Content: "Note 1"},
			{Type: "sticky_note", Content: "Note 2"},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkBulkField(t, "Created", result.Created, 2)
}

func TestBulkUpdate_Success(t *testing.T) {
	client := newBulkTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		checkBulkField(t, "method", r.Method, http.MethodPatch)
		writeJSON(w, map[string]interface{}{
			"id": r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:],
		})
	})

	content1 := "Updated content 1"
	content2 := "Updated content 2"
	result, err := client.BulkUpdate(context.Background(), BulkUpdateArgs{
		BoardID: "board123",
		Items: []BulkUpdateItem{
			{ItemID: "item1", Content: &content1},
			{ItemID: "item2", Content: &content2},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkBulkField(t, "Updated", result.Updated, 2)
	checkBulkField(t, "Errors count", len(result.Errors), 0)
}

func TestBulkDelete_Success(t *testing.T) {
	client := newBulkTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		checkBulkField(t, "method", r.Method, http.MethodDelete)
		w.WriteHeader(http.StatusNoContent)
	})

	result, err := client.BulkDelete(context.Background(), BulkDeleteArgs{
		BoardID: "board123",
		ItemIDs: []string{"item1", "item2", "item3"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkBulkField(t, "Deleted", result.Deleted, 3)
	checkBulkField(t, "Errors count", len(result.Errors), 0)
}

// TestBulk_ValidationErrors covers input validation across the bulk methods;
// every case must fail with a message naming the problem.
func TestBulk_ValidationErrors(t *testing.T) {
	client := newTestClientWithServer("http://localhost")
	ctx := context.Background()

	tests := []struct {
		name    string
		call    func() error
		wantErr string
	}{
		{"create with empty board ID", func() error {
			_, err := client.BulkCreate(ctx, BulkCreateArgs{Items: []BulkCreateItem{{Type: "sticky_note", Content: "Test"}}})
			return err
		}, "board_id is required"},
		{"create with empty items", func() error {
			_, err := client.BulkCreate(ctx, BulkCreateArgs{BoardID: "board123", Items: []BulkCreateItem{}})
			return err
		}, "at least one item is required"},
		{"update with empty board ID", func() error {
			_, err := client.BulkUpdate(ctx, BulkUpdateArgs{Items: []BulkUpdateItem{{ItemID: "item1"}}})
			return err
		}, "board_id is required"},
		{"update with empty items", func() error {
			_, err := client.BulkUpdate(ctx, BulkUpdateArgs{BoardID: "board123", Items: []BulkUpdateItem{}})
			return err
		}, "at least one item is required"},
		{"update with too many items", func() error {
			_, err := client.BulkUpdate(ctx, BulkUpdateArgs{BoardID: "board123", Items: manyBulkUpdateItems(21)})
			return err
		}, "maximum 20 items"},
		{"delete with empty board ID", func() error { _, err := client.BulkDelete(ctx, BulkDeleteArgs{ItemIDs: []string{"item1"}}); return err }, "board_id is required"},
		{"delete with empty items", func() error {
			_, err := client.BulkDelete(ctx, BulkDeleteArgs{BoardID: "board123", ItemIDs: []string{}})
			return err
		}, "at least one item_id is required"},
		{"delete with too many items", func() error {
			_, err := client.BulkDelete(ctx, BulkDeleteArgs{BoardID: "board123", ItemIDs: manyBulkItemIDs(21)})
			return err
		}, "maximum 20 items"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

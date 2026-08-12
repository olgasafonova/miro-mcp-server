package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// wireComment builds the API's comment-thread shape as captured live on
// 13-08-2026.
func wireComment(id string, resolved bool, itemID string, messages ...string) map[string]interface{} {
	msgs := make([]map[string]interface{}, len(messages))
	for i, content := range messages {
		msgs[i] = map[string]interface{}{
			"id": "msg-" + content, "type": "message", "content": content,
			"createdAt": "2026-08-12T22:44:21Z",
			"createdBy": map[string]interface{}{"id": "u1", "type": "user", "name": "Olga Safonova"},
		}
	}
	c := map[string]interface{}{
		"id": id, "type": "comment", "resolved": resolved,
		"createdAt": "2026-08-12T22:44:21Z",
		"createdBy": map[string]interface{}{"id": "u1", "type": "user", "name": "Olga Safonova"},
		"messages":  msgs,
		"position":  map[string]interface{}{"type": "canvas", "x": 0.0, "y": 0.0},
	}
	if itemID != "" {
		c["position"] = map[string]interface{}{"type": "attached", "x": 0.0, "y": 0.0, "itemId": itemID}
	}
	return c
}

func TestCreateComment_BoardLevel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/comments") {
			t.Errorf("path = %s, want /comments", r.URL.Path)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["content"] != "First!" {
			t.Errorf("content = %v, want 'First!'", body["content"])
		}
		if _, has := body["itemId"]; has {
			t.Error("itemId sent for a board-level comment")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(wireComment("c1", false, "", "First!"))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateComment(context.Background(), CreateCommentArgs{
		BoardID: "board1", Content: "First!",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "c1" {
		t.Errorf("ID = %q, want 'c1'", result.ID)
	}
	if result.ItemID != "" {
		t.Errorf("ItemID = %q, want empty", result.ItemID)
	}
}

func TestCreateComment_AttachedToItem(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["itemId"] != "item42" {
			t.Errorf("itemId = %v, want 'item42'", body["itemId"])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(wireComment("c2", false, "item42", "On the sticky"))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateComment(context.Background(), CreateCommentArgs{
		BoardID: "board1", Content: "On the sticky", ItemID: "item42",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ItemID != "item42" {
		t.Errorf("ItemID = %q, want 'item42'", result.ItemID)
	}
}

func TestCreateComment_Validation(t *testing.T) {
	client := newTestClientWithServer("http://unused")
	if _, err := client.CreateComment(context.Background(), CreateCommentArgs{BoardID: "b1"}); err == nil {
		t.Error("empty content accepted")
	}
	if _, err := client.CreateComment(context.Background(), CreateCommentArgs{Content: "x"}); err == nil {
		t.Error("empty board_id accepted")
	}
}

func TestListComments_ProjectsThreads(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("limit"); got != "20" {
			t.Errorf("limit = %q, want '20'", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []interface{}{
				wireComment("c1", true, "", "resolved thread", "with a reply"),
				wireComment("c2", false, "item9", "open thread"),
			},
			"total": 2, "offset": 0, "size": 2, "limit": 20, "type": "list",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListComments(context.Background(), ListCommentsArgs{BoardID: "board1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 || result.Total != 2 || result.HasMore {
		t.Errorf("Count/Total/HasMore = %d/%d/%v, want 2/2/false", result.Count, result.Total, result.HasMore)
	}
	first := result.Comments[0]
	if !first.Resolved || len(first.Messages) != 2 {
		t.Errorf("first thread resolved=%v messages=%d, want true/2", first.Resolved, len(first.Messages))
	}
	if first.CreatedBy == nil || first.CreatedBy.Name != "Olga Safonova" {
		t.Errorf("CreatedBy = %+v, want Olga Safonova", first.CreatedBy)
	}
	if got := result.Comments[1].ItemID; got != "item9" {
		t.Errorf("attached ItemID = %q, want 'item9'", got)
	}
}

func TestListComments_EmptyBoardHasMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []interface{}{}, "total": 0, "offset": 0, "size": 0,
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListComments(context.Background(), ListCommentsArgs{BoardID: "board1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message == "" {
		t.Error("zero-result response carries no message; agents cannot tell empty from failed")
	}
}

func TestReplyComment_PostsToMessages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/comments/c1/messages") {
			t.Errorf("got %s %s, want POST .../comments/c1/messages", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(wireComment("c1", false, "", "original", "the reply"))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ReplyComment(context.Background(), ReplyCommentArgs{
		BoardID: "board1", CommentID: "c1", Content: "the reply",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MessageCount != 2 {
		t.Errorf("MessageCount = %d, want 2", result.MessageCount)
	}
}

func TestResolveComment_DefaultsToResolve(t *testing.T) {
	var sentResolved interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %s, want PATCH", r.Method)
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		sentResolved = body["resolved"]
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(wireComment("c1", true, "", "msg"))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ResolveComment(context.Background(), ResolveCommentArgs{
		BoardID: "board1", CommentID: "c1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sentResolved != true {
		t.Errorf("sent resolved = %v, want true", sentResolved)
	}
	if !result.Resolved {
		t.Error("result.Resolved = false, want true")
	}
}

func TestResolveComment_ReopenSendsFalse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["resolved"] != false {
			t.Errorf("sent resolved = %v, want false", body["resolved"])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(wireComment("c1", false, "", "msg"))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	reopen := false
	result, err := client.ResolveComment(context.Background(), ResolveCommentArgs{
		BoardID: "board1", CommentID: "c1", Resolved: &reopen,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Resolved {
		t.Error("result.Resolved = true, want false")
	}
	if !strings.Contains(result.Message, "Reopened") {
		t.Errorf("Message = %q, want it to say Reopened", result.Message)
	}
}

func TestCommentErrors_CarryExperimentalHint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"message": "not found"})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.ListComments(context.Background(), ListCommentsArgs{BoardID: "board1"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "v2-experimental") {
		t.Errorf("404 error lacks the experimental-availability hint: %v", err)
	}
}

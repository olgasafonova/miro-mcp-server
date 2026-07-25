package miro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListBoardMembers_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/boards/board123/members") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "user1", "name": "Alice", "role": "owner"},
				{"id": "user2", "name": "Bob", "role": "editor"},
			},
			"total": 2,
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListBoardMembers(context.Background(), ListBoardMembersArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
	if len(result.Members) != 2 {
		t.Errorf("Members count = %d, want 2", len(result.Members))
	}
}

func TestListBoardMembers_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListBoardMembers(context.Background(), ListBoardMembersArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Message, "No members") {
		t.Errorf("expected 'No members' message, got: %s", result.Message)
	}
}

func TestGetBoardMember_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/members/member456" {
			t.Errorf("expected /boards/board123/members/member456, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "member456",
			"name":  "John Doe",
			"email": "john@example.com",
			"role":  "editor",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetBoardMember(context.Background(), GetBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "member456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "member456" {
		t.Errorf("ID = %q, want 'member456'", result.ID)
	}
	if result.Name != "John Doe" {
		t.Errorf("Name = %q, want 'John Doe'", result.Name)
	}
	if result.Role != "editor" {
		t.Errorf("Role = %q, want 'editor'", result.Role)
	}
}

func TestRemoveBoardMember_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/members/member456" {
			t.Errorf("expected /boards/board123/members/member456, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.RemoveBoardMember(context.Background(), RemoveBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "member456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
}

func TestUpdateBoardMember_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/members/member456" {
			t.Errorf("expected /boards/board123/members/member456, got %s", r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["role"] != "editor" {
			t.Errorf("role = %v, want 'editor'", body["role"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "member456",
			"name": "John Doe",
			"role": "editor",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateBoardMember(context.Background(), UpdateBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "member456",
		Role:     "editor",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "member456" {
		t.Errorf("ID = %q, want 'member456'", result.ID)
	}
	if result.Role != "editor" {
		t.Errorf("Role = %q, want 'editor'", result.Role)
	}
}

func TestGetBoardMember_WithEmptyName(t *testing.T) {
	// Tests the branch where member.Name == "" and email is used as fallback
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "member123",
			"name":  "", // Empty name
			"email": "test@example.com",
			"role":  "editor",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetBoardMember(context.Background(), GetBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "member123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Message should use email as fallback when name is empty
	if !strings.Contains(result.Message, "test@example.com") {
		t.Errorf("message should contain email when name is empty, got: %s", result.Message)
	}
}

func TestRemoveBoardMember_APIError(t *testing.T) {
	// Tests the error branch where API returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  403,
			"message": "Access denied",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.RemoveBoardMember(context.Background(), RemoveBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "member123",
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if result.Success {
		t.Error("expected Success to be false")
	}
	if !strings.Contains(result.Message, "Failed to remove member") {
		t.Errorf("expected failure message, got: %s", result.Message)
	}
}

func TestUpdateBoardMember_WithEmptyName(t *testing.T) {
	// Tests the branch where member.Name == "" and email is used as fallback
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "member123",
			"name":  "", // Empty name
			"email": "user@example.com",
			"role":  "viewer",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateBoardMember(context.Background(), UpdateBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "member123",
		Role:     "viewer",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Message should use email as fallback when name is empty
	if !strings.Contains(result.Message, "user@example.com") {
		t.Errorf("message should contain email when name is empty, got: %s", result.Message)
	}
}

func TestListBoardMembers_EmptyResult(t *testing.T) {
	// Tests the branch for empty member list message
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data":   []interface{}{},
			"total":  0,
			"offset": 0,
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListBoardMembers(context.Background(), ListBoardMembersArgs{
		BoardID: "board123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Message != "No members found on this board" {
		t.Errorf("expected 'No members found on this board', got: %s", result.Message)
	}
}

func TestListBoardMembers_WithOffset(t *testing.T) {
	// Tests the HasMore calculation when offset > 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return exactly limit items to trigger HasMore = true
		members := make([]map[string]interface{}, 50)
		for i := 0; i < 50; i++ {
			members[i] = map[string]interface{}{
				"id":    fmt.Sprintf("member%d", i),
				"name":  fmt.Sprintf("User %d", i),
				"email": fmt.Sprintf("user%d@example.com", i),
				"role":  "viewer",
			}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data":   members,
			"total":  100,
			"offset": 50, // Non-zero offset indicates more pages
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListBoardMembers(context.Background(), ListBoardMembersArgs{
		BoardID: "board123",
		Limit:   50,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasMore {
		t.Error("expected HasMore to be true when offset > 0 and count >= limit")
	}
}

// Tests for RemoveBoardMember error paths
func TestRemoveBoardMember_EmptyMemberID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.RemoveBoardMember(context.Background(), RemoveBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "",
	})
	if err == nil {
		t.Error("expected error for empty member_id")
	}
	if !strings.Contains(err.Error(), "member_id is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRemoveBoardMember_APIError_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Member not found"}`))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.RemoveBoardMember(context.Background(), RemoveBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "member456",
	})
	if err == nil {
		t.Error("expected error for 404 response")
	}
	if result.Success {
		t.Error("expected Success=false on error")
	}
}

// Tests for UpdateBoardMember error paths
func TestUpdateBoardMember_EmptyMemberID(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.UpdateBoardMember(context.Background(), UpdateBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "",
		Role:     "editor",
	})
	if err == nil {
		t.Error("expected error for empty member_id")
	}
}

func TestUpdateBoardMember_EmptyRole(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.UpdateBoardMember(context.Background(), UpdateBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "member456",
		Role:     "",
	})
	if err == nil {
		t.Error("expected error for empty role")
	}
}

func TestUpdateBoardMember_InvalidRole(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	_, err := client.UpdateBoardMember(context.Background(), UpdateBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "member456",
		Role:     "admin",
	})
	if err == nil {
		t.Error("expected error for invalid role")
	}
	if !strings.Contains(err.Error(), "invalid role") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUpdateBoardMember_JSONParseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.UpdateBoardMember(context.Background(), UpdateBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "member456",
		Role:     "editor",
	})
	if err == nil {
		t.Error("expected error for invalid JSON response")
	}
}

func TestUpdateBoardMember_EmptyNameFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "member456",
			"name":  "",
			"email": "user@example.com",
			"role":  "editor",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateBoardMember(context.Background(), UpdateBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "member456",
		Role:     "editor",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Message, "user@example.com") {
		t.Errorf("expected email in message when name is empty, got: %s", result.Message)
	}
}

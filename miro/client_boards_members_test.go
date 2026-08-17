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

// newBoardMembersTestClient starts a test server with the given handler and
// returns a client pointed at it. The server is closed via t.Cleanup.
func newBoardMembersTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return newTestClientWithServer(server.URL)
}

// boardMemberJSONHandler returns a handler that writes the given payload as JSON.
func boardMemberJSONHandler(payload map[string]interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}
}

func TestListBoardMembers_Success(t *testing.T) {
	client := newBoardMembersTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/boards/board123/members") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		boardMemberJSONHandler(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "user1", "name": "Alice", "role": "owner"},
				{"id": "user2", "name": "Bob", "role": "editor"},
			},
			"total": 2,
		})(w, r)
	})

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
	tests := []struct {
		name        string
		payload     map[string]interface{}
		wantContain string
		wantExact   string
	}{
		{
			name:        "MissingTotal",
			payload:     map[string]interface{}{"data": []map[string]interface{}{}},
			wantContain: "No members",
		},
		{
			name:      "ZeroTotalZeroOffset",
			payload:   map[string]interface{}{"data": []interface{}{}, "total": 0, "offset": 0},
			wantExact: "No members found on this board",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newBoardMembersTestClient(t, boardMemberJSONHandler(tt.payload))
			result, err := client.ListBoardMembers(context.Background(), ListBoardMembersArgs{
				BoardID: "board123",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantContain != "" && !strings.Contains(result.Message, tt.wantContain) {
				t.Errorf("expected %q in message, got: %s", tt.wantContain, result.Message)
			}
			if tt.wantExact != "" && result.Message != tt.wantExact {
				t.Errorf("expected %q, got: %s", tt.wantExact, result.Message)
			}
		})
	}
}

func TestGetBoardMember_Success(t *testing.T) {
	client := newBoardMembersTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/members/member456" {
			t.Errorf("expected /boards/board123/members/member456, got %s", r.URL.Path)
		}
		boardMemberJSONHandler(map[string]interface{}{
			"id":    "member456",
			"name":  "John Doe",
			"email": "john@example.com",
			"role":  "editor",
		})(w, r)
	})

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
	client := newBoardMembersTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/boards/board123/members/member456" {
			t.Errorf("expected /boards/board123/members/member456, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

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
	client := newBoardMembersTestClient(t, func(w http.ResponseWriter, r *http.Request) {
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
		boardMemberJSONHandler(map[string]interface{}{
			"id":   "member456",
			"name": "John Doe",
			"role": "editor",
		})(w, r)
	})

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

// TestBoardMember_EmptyNameEmailFallback covers the branches where
// member.Name == "" and the email is used as fallback in the result message.
func TestBoardMember_EmptyNameEmailFallback(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]interface{}
		email   string
		call    func(c *Client) (string, error)
	}{
		{
			name: "GetBoardMember",
			payload: map[string]interface{}{
				"id": "member123", "name": "", "email": "test@example.com", "role": "editor",
			},
			email: "test@example.com",
			call: func(c *Client) (string, error) {
				result, err := c.GetBoardMember(context.Background(), GetBoardMemberArgs{
					BoardID:  "board123",
					MemberID: "member123",
				})
				if err != nil {
					return "", err
				}
				return result.Message, nil
			},
		},
		{
			name: "UpdateBoardMemberViewer",
			payload: map[string]interface{}{
				"id": "member123", "name": "", "email": "user@example.com", "role": "viewer",
			},
			email: "user@example.com",
			call: func(c *Client) (string, error) {
				result, err := c.UpdateBoardMember(context.Background(), UpdateBoardMemberArgs{
					BoardID:  "board123",
					MemberID: "member123",
					Role:     "viewer",
				})
				if err != nil {
					return "", err
				}
				return result.Message, nil
			},
		},
		{
			name: "UpdateBoardMemberEditor",
			payload: map[string]interface{}{
				"id": "member456", "name": "", "email": "user@example.com", "role": "editor",
			},
			email: "user@example.com",
			call: func(c *Client) (string, error) {
				result, err := c.UpdateBoardMember(context.Background(), UpdateBoardMemberArgs{
					BoardID:  "board123",
					MemberID: "member456",
					Role:     "editor",
				})
				if err != nil {
					return "", err
				}
				return result.Message, nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newBoardMembersTestClient(t, boardMemberJSONHandler(tt.payload))
			message, err := tt.call(client)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(message, tt.email) {
				t.Errorf("message should contain email when name is empty, got: %s", message)
			}
		})
	}
}

// TestRemoveBoardMember_APIErrors covers error branches where the API
// returns a failure status.
func TestRemoveBoardMember_APIErrors(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		body        string
		wantMessage string
	}{
		{
			name:        "Forbidden",
			status:      http.StatusForbidden,
			body:        `{"status":403,"message":"Access denied"}`,
			wantMessage: "Failed to remove member",
		},
		{
			name:   "NotFound",
			status: http.StatusNotFound,
			body:   `{"message":"Member not found"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newBoardMembersTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			})
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
			if tt.wantMessage != "" && !strings.Contains(result.Message, tt.wantMessage) {
				t.Errorf("expected failure message, got: %s", result.Message)
			}
		})
	}
}

func TestListBoardMembers_WithOffset(t *testing.T) {
	// Tests the HasMore calculation when offset > 0
	client := newBoardMembersTestClient(t, func(w http.ResponseWriter, r *http.Request) {
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
		boardMemberJSONHandler(map[string]interface{}{
			"data":   members,
			"total":  100,
			"offset": 50, // Non-zero offset indicates more pages
		})(w, r)
	})

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

// TestBoardMemberValidationErrors covers argument validation on
// RemoveBoardMember and UpdateBoardMember; no HTTP call is made.
func TestBoardMemberValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	tests := []struct {
		name    string
		call    func() error
		wantErr string
	}{
		{
			name: "RemoveEmptyMemberID",
			call: func() error {
				_, err := client.RemoveBoardMember(context.Background(), RemoveBoardMemberArgs{
					BoardID: "board123", MemberID: "",
				})
				return err
			},
			wantErr: "member_id is required",
		},
		{
			name: "UpdateEmptyMemberID",
			call: func() error {
				_, err := client.UpdateBoardMember(context.Background(), UpdateBoardMemberArgs{
					BoardID: "board123", MemberID: "", Role: "editor",
				})
				return err
			},
		},
		{
			name: "UpdateEmptyRole",
			call: func() error {
				_, err := client.UpdateBoardMember(context.Background(), UpdateBoardMemberArgs{
					BoardID: "board123", MemberID: "member456", Role: "",
				})
				return err
			},
		},
		{
			name: "UpdateInvalidRole",
			call: func() error {
				_, err := client.UpdateBoardMember(context.Background(), UpdateBoardMemberArgs{
					BoardID: "board123", MemberID: "member456", Role: "admin",
				})
				return err
			},
			wantErr: "invalid role",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if err == nil {
				t.Fatal("expected validation error")
			}
			if tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestUpdateBoardMember_JSONParseError(t *testing.T) {
	client := newBoardMembersTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json}`))
	})

	_, err := client.UpdateBoardMember(context.Background(), UpdateBoardMemberArgs{
		BoardID:  "board123",
		MemberID: "member456",
		Role:     "editor",
	})
	if err == nil {
		t.Error("expected error for invalid JSON response")
	}
}

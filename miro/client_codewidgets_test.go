package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateCodeWidget_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/code_widgets") {
			t.Errorf("expected code_widgets path, got %s", r.URL.Path)
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		data, ok := body["data"].(map[string]interface{})
		if !ok {
			t.Fatal("expected data in request body")
		}
		if data["code"] != `fmt.Println("hi")` {
			t.Errorf("code = %v", data["code"])
		}
		if data["language"] != "go" {
			t.Errorf("language = %v, want 'go'", data["language"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "cw123",
			"data": map[string]interface{}{
				"title":    "My Snippet",
				"language": "go",
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateCodeWidget(context.Background(), CreateCodeWidgetArgs{
		BoardID:  "board123",
		Code:     `fmt.Println("hi")`,
		Language: "go",
		Title:    "My Snippet",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "cw123" {
		t.Errorf("ID = %q, want 'cw123'", result.ID)
	}
	if result.ItemURL == "" {
		t.Error("expected non-empty ItemURL")
	}
	if result.Language != "go" {
		t.Errorf("Language = %q, want 'go'", result.Language)
	}
}

func TestCreateCodeWidget_Validation(t *testing.T) {
	client := newTestClientWithServer("http://unused.invalid")

	if _, err := client.CreateCodeWidget(context.Background(), CreateCodeWidgetArgs{
		BoardID: "board123",
	}); err == nil {
		t.Error("expected error for missing code")
	}

	longCode := strings.Repeat("x", 6001)
	if _, err := client.CreateCodeWidget(context.Background(), CreateCodeWidgetArgs{
		BoardID: "board123",
		Code:    longCode,
	}); err == nil || !strings.Contains(err.Error(), "6000") {
		t.Errorf("expected 6000-char cap error, got: %v", err)
	}

	longTitle := strings.Repeat("t", 101)
	if _, err := client.CreateCodeWidget(context.Background(), CreateCodeWidgetArgs{
		BoardID: "board123",
		Code:    "ok",
		Title:   longTitle,
	}); err == nil || !strings.Contains(err.Error(), "100") {
		t.Errorf("expected 100-char title cap error, got: %v", err)
	}
}

func TestCreateCodeWidget_WithParentAndGeometry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		parent, ok := body["parent"].(map[string]interface{})
		if !ok || parent["id"] != "frame123" {
			t.Errorf("expected parent.id 'frame123', got %v", body["parent"])
		}
		geo, ok := body["geometry"].(map[string]interface{})
		if !ok || geo["width"] != float64(400) {
			t.Errorf("expected geometry.width 400, got %v", body["geometry"])
		}
		pos, ok := body["position"].(map[string]interface{})
		if !ok || pos["x"] != float64(10) || pos["origin"] != "center" {
			t.Errorf("expected position with x=10 origin=center, got %v", body["position"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "cw456", "data": map[string]interface{}{}})
	}))
	defer server.Close()

	width := 400.0
	client := newTestClientWithServer(server.URL)
	_, err := client.CreateCodeWidget(context.Background(), CreateCodeWidgetArgs{
		BoardID:  "board123",
		Code:     "x = 1",
		X:        10,
		Y:        20,
		Width:    &width,
		ParentID: "frame123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetCodeWidget_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/code_widgets/cw123") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "cw123",
			"data": map[string]interface{}{
				"code":               "print('hi')",
				"language":           "python",
				"title":              "Greeting",
				"lineNumbersVisible": true,
			},
			"position":   map[string]interface{}{"x": 15.5, "y": -3.0},
			"geometry":   map[string]interface{}{"width": 300.0, "height": 150.0},
			"parent":     map[string]interface{}{"id": "frame9"},
			"createdAt":  "2026-07-21T09:00:00Z",
			"modifiedAt": "2026-07-21T10:00:00Z",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.GetCodeWidget(context.Background(), GetCodeWidgetArgs{
		BoardID: "board123",
		ItemID:  "cw123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Code != "print('hi')" || result.Language != "python" {
		t.Errorf("unexpected data: %+v", result)
	}
	if !result.LineNumbersVisible {
		t.Error("expected LineNumbersVisible true")
	}
	if result.X != 15.5 || result.Y != -3.0 {
		t.Errorf("position = (%v, %v)", result.X, result.Y)
	}
	if result.Width != 300 || result.Height != 150 {
		t.Errorf("geometry = (%v, %v)", result.Width, result.Height)
	}
	if result.ParentID != "frame9" {
		t.Errorf("ParentID = %q", result.ParentID)
	}
	if result.CreatedAt == "" || result.ModifiedAt == "" {
		t.Error("expected timestamps")
	}
}

func TestGetCodeWidget_NotFoundExperimentalHint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    "not_found",
			"message": "Item not found",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	_, err := client.GetCodeWidget(context.Background(), GetCodeWidgetArgs{
		BoardID: "board123",
		ItemID:  "cw404",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFoundError(err) {
		t.Errorf("expected wrapped not-found to remain detectable, got: %v", err)
	}
	if !strings.Contains(err.Error(), "v2-experimental") {
		t.Errorf("expected experimental-availability hint in error, got: %v", err)
	}
}

func TestListCodeWidgets_DefaultLimitAndPreview(t *testing.T) {
	longCode := strings.Repeat("a", 200)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("limit"); got != "50" {
			t.Errorf("limit = %q, want '50'", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":   "cw-1",
					"data": map[string]interface{}{"code": longCode, "language": "go", "title": "Long"},
				},
				{
					"id":     "cw-2",
					"data":   map[string]interface{}{"code": "short", "language": "python"},
					"parent": map[string]interface{}{"id": "frame1"},
				},
			},
			"cursor": "next-page",
		})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.ListCodeWidgets(context.Background(), ListCodeWidgetsArgs{BoardID: "board123"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Fatalf("Count = %d, want 2", result.Count)
	}
	if len(result.Widgets[0].CodePreview) != 80 || !strings.HasSuffix(result.Widgets[0].CodePreview, "...") {
		t.Errorf("preview not truncated to 80 with ellipsis: %q (len %d)", result.Widgets[0].CodePreview, len(result.Widgets[0].CodePreview))
	}
	if result.Widgets[1].CodePreview != "short" {
		t.Errorf("short code should pass through, got %q", result.Widgets[1].CodePreview)
	}
	if result.Widgets[1].ParentID != "frame1" {
		t.Errorf("ParentID = %q", result.Widgets[1].ParentID)
	}
	if !result.HasMore || result.Cursor != "next-page" {
		t.Errorf("pagination: HasMore=%v Cursor=%q", result.HasMore, result.Cursor)
	}
}

func TestListCodeWidgets_CapsLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("limit"); got != "100" {
			t.Errorf("limit = %q, want capped '100'", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []map[string]interface{}{}})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	if _, err := client.ListCodeWidgets(context.Background(), ListCodeWidgetsArgs{BoardID: "board123", Limit: 500}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateCodeWidget_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/code_widgets/cw123") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		data, ok := body["data"].(map[string]interface{})
		if !ok || data["code"] != "updated" {
			t.Errorf("expected data.code 'updated', got %v", body["data"])
		}
		if _, hasPos := body["position"]; hasPos {
			t.Error("update must not send position; that's MoveCodeWidget's job")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "cw123"})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.UpdateCodeWidget(context.Background(), UpdateCodeWidgetArgs{
		BoardID: "board123",
		ItemID:  "cw123",
		Code:    "updated",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "cw123" {
		t.Errorf("ID = %q", result.ID)
	}
}

func TestUpdateCodeWidget_NoFields(t *testing.T) {
	client := newTestClientWithServer("http://unused.invalid")
	_, err := client.UpdateCodeWidget(context.Background(), UpdateCodeWidgetArgs{
		BoardID: "board123",
		ItemID:  "cw123",
	})
	if err == nil || !strings.Contains(err.Error(), "no fields to update") {
		t.Errorf("expected 'no fields to update' error, got: %v", err)
	}
}

func TestMoveCodeWidget_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/code_widgets/cw123/position") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["x"] != float64(250) || body["y"] != float64(-80) || body["origin"] != "center" {
			t.Errorf("unexpected body: %v", body)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "cw123"})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.MoveCodeWidget(context.Background(), MoveCodeWidgetArgs{
		BoardID: "board123",
		ItemID:  "cw123",
		X:       250,
		Y:       -80,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.X != 250 || result.Y != -80 {
		t.Errorf("result position = (%v, %v)", result.X, result.Y)
	}
}

func TestDeleteCodeWidget_Success(t *testing.T) {
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/code_widgets/cw123") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteCodeWidget(context.Background(), DeleteCodeWidgetArgs{
		BoardID: "board123",
		ItemID:  "cw123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success || !called {
		t.Errorf("Success=%v called=%v", result.Success, called)
	}
}

func TestDeleteCodeWidget_DryRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("dry run must not hit the API")
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.DeleteCodeWidget(context.Background(), DeleteCodeWidgetArgs{
		BoardID: "board123",
		ItemID:  "cw123",
		DryRun:  true,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success || !strings.Contains(result.Message, "DRY RUN") {
		t.Errorf("unexpected result: %+v", result)
	}
}

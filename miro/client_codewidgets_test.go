package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// cwCheckRequest verifies the HTTP method and URL path suffix of a request.
func cwCheckRequest(t *testing.T, r *http.Request, method, pathSuffix string) {
	t.Helper()
	if r.Method != method {
		t.Errorf("expected %s, got %s", method, r.Method)
	}
	if !strings.HasSuffix(r.URL.Path, pathSuffix) {
		t.Errorf("unexpected path %s", r.URL.Path)
	}
}

// cwDecodeBody decodes a JSON request body into a generic map.
func cwDecodeBody(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	json.NewDecoder(r.Body).Decode(&body)
	return body
}

// cwCheckField verifies a top-level field of a decoded JSON body.
func cwCheckField(t *testing.T, body map[string]interface{}, field string, want interface{}) {
	t.Helper()
	if body[field] != want {
		t.Errorf("%s = %v, want %v", field, body[field], want)
	}
}

// cwNested extracts a nested JSON object such as data, parent, or position.
func cwNested(t *testing.T, body map[string]interface{}, key string) map[string]interface{} {
	t.Helper()
	m, ok := body[key].(map[string]interface{})
	if !ok {
		t.Fatalf("expected %s object in request body, got %v", key, body[key])
	}
	return m
}

// cwCheckEq verifies a single result field against its expected value.
func cwCheckEq(t *testing.T, name string, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

// cwCheckNonEmpty verifies that a result field is populated.
func cwCheckNonEmpty(t *testing.T, name, got string) {
	t.Helper()
	if got == "" {
		t.Errorf("expected non-empty %s", name)
	}
}

// cwCheckLimitParam verifies the limit query parameter of a list request.
func cwCheckLimitParam(t *testing.T, r *http.Request, want string) {
	t.Helper()
	if got := r.URL.Query().Get("limit"); got != want {
		t.Errorf("limit = %q, want %q", got, want)
	}
}

// cwServeJSON writes a JSON response.
func cwServeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func TestCreateCodeWidget_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cwCheckRequest(t, r, http.MethodPost, "/code_widgets")
		data := cwNested(t, cwDecodeBody(t, r), "data")
		cwCheckField(t, data, "code", `fmt.Println("hi")`)
		cwCheckField(t, data, "language", "go")

		cwServeJSON(w, map[string]interface{}{
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
	cwCheckEq(t, "ID", result.ID, "cw123")
	cwCheckNonEmpty(t, "ItemURL", result.ItemURL)
	cwCheckEq(t, "Language", result.Language, "go")
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
		body := cwDecodeBody(t, r)
		cwCheckField(t, cwNested(t, body, "parent"), "id", "frame123")
		cwCheckField(t, cwNested(t, body, "geometry"), "width", float64(400))
		pos := cwNested(t, body, "position")
		cwCheckField(t, pos, "x", float64(10))
		cwCheckField(t, pos, "origin", "center")

		cwServeJSON(w, map[string]interface{}{"id": "cw456", "data": map[string]interface{}{}})
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
		cwCheckRequest(t, r, http.MethodGet, "/code_widgets/cw123")

		cwServeJSON(w, map[string]interface{}{
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
	cwCheckEq(t, "Code", result.Code, "print('hi')")
	cwCheckEq(t, "Language", result.Language, "python")
	cwCheckEq(t, "LineNumbersVisible", result.LineNumbersVisible, true)
	cwCheckEq(t, "X", result.X, 15.5)
	cwCheckEq(t, "Y", result.Y, -3.0)
	cwCheckEq(t, "Width", result.Width, float64(300))
	cwCheckEq(t, "Height", result.Height, float64(150))
	cwCheckEq(t, "ParentID", result.ParentID, "frame9")
	cwCheckNonEmpty(t, "CreatedAt", result.CreatedAt)
	cwCheckNonEmpty(t, "ModifiedAt", result.ModifiedAt)
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

// cwCheckTruncatedPreview verifies that a long code preview is cut to 80 chars with an ellipsis.
func cwCheckTruncatedPreview(t *testing.T, preview string) {
	t.Helper()
	if len(preview) != 80 || !strings.HasSuffix(preview, "...") {
		t.Errorf("preview not truncated to 80 with ellipsis: %q (len %d)", preview, len(preview))
	}
}

func TestListCodeWidgets_DefaultLimitAndPreview(t *testing.T) {
	longCode := strings.Repeat("a", 200)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cwCheckLimitParam(t, r, "50")

		cwServeJSON(w, map[string]interface{}{
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
	cwCheckTruncatedPreview(t, result.Widgets[0].CodePreview)
	cwCheckEq(t, "Widgets[1].CodePreview", result.Widgets[1].CodePreview, "short")
	cwCheckEq(t, "Widgets[1].ParentID", result.Widgets[1].ParentID, "frame1")
	cwCheckEq(t, "HasMore", result.HasMore, true)
	cwCheckEq(t, "Cursor", result.Cursor, "next-page")
}

func TestListCodeWidgets_CapsLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cwCheckLimitParam(t, r, "100")
		cwServeJSON(w, map[string]interface{}{"data": []map[string]interface{}{}})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	if _, err := client.ListCodeWidgets(context.Background(), ListCodeWidgetsArgs{BoardID: "board123", Limit: 500}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateCodeWidget_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cwCheckRequest(t, r, http.MethodPatch, "/code_widgets/cw123")
		body := cwDecodeBody(t, r)
		cwCheckField(t, cwNested(t, body, "data"), "code", "updated")
		if _, hasPos := body["position"]; hasPos {
			t.Error("update must not send position; that's MoveCodeWidget's job")
		}

		cwServeJSON(w, map[string]interface{}{"id": "cw123"})
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
	cwCheckEq(t, "ID", result.ID, "cw123")
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
		cwCheckRequest(t, r, http.MethodPatch, "/code_widgets/cw123/position")
		body := cwDecodeBody(t, r)
		cwCheckField(t, body, "x", float64(250))
		cwCheckField(t, body, "y", float64(-80))
		cwCheckField(t, body, "origin", "center")

		cwServeJSON(w, map[string]interface{}{"id": "cw123"})
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
	cwCheckEq(t, "X", result.X, float64(250))
	cwCheckEq(t, "Y", result.Y, float64(-80))
}

func TestDeleteCodeWidget_Success(t *testing.T) {
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		cwCheckRequest(t, r, http.MethodDelete, "/code_widgets/cw123")
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

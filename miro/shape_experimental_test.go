package miro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// =============================================================================
// Experimental / flowchart shape tests
// =============================================================================

// seCheckEq verifies a single value against its expected value.
func seCheckEq(t *testing.T, name string, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func TestBuildExperimentalShapeStyle_MapsToExperimentalKeyNames(t *testing.T) {
	style, err := buildExperimentalShapeStyle("#006400", "#000000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	seCheckEq(t, "fillColor", style["fillColor"], "#006400")
	seCheckEq(t, "borderColor", style["borderColor"], "#000000")
	// The non-experimental endpoint uses "color"; this one must not.
	if _, present := style["color"]; present {
		t.Error("experimental style must not carry a plain color key")
	}
}

func TestBuildExperimentalShapeStyle_ResolvesNamedColors(t *testing.T) {
	style, err := buildExperimentalShapeStyle("red", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if style["fillColor"] == "red" {
		t.Error("named color should resolve to a hex value")
	}
}

func TestBuildExperimentalShapeStyle_OmitsEmptyColors(t *testing.T) {
	style, err := buildExperimentalShapeStyle("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(style) != 0 {
		t.Errorf("style = %v, want empty when no colors are set", style)
	}
}

func TestBuildExperimentalShapeStyle_RejectsUnknownColor(t *testing.T) {
	_, err := buildExperimentalShapeStyle("chartreuse-ish", "")
	if err == nil {
		t.Fatal("expected an error for an unrecognized color")
	}
	if !strings.Contains(err.Error(), "fill_color") {
		t.Errorf("error = %v, want the fill_color tag named", err)
	}
}

func TestCreateShapeExperimentalArgs_ToCoreBody(t *testing.T) {
	args := CreateShapeExperimentalArgs{
		BoardID:     "board123",
		Shape:       "flow_chart_predefined_process",
		Content:     "Process step",
		X:           10,
		Y:           20,
		Width:       300,
		Height:      150,
		FillColor:   "#006400",
		BorderColor: "#000000",
		ParentID:    "frame999",
	}

	core := args.toCoreBody()

	seCheckEq(t, "boardID", core.boardID, "board123")
	seCheckEq(t, "shape", core.shape, "flow_chart_predefined_process")
	seCheckEq(t, "content", core.content, "Process step")
	seCheckEq(t, "x", core.x, float64(10))
	seCheckEq(t, "y", core.y, float64(20))
	seCheckEq(t, "width", core.width, float64(300))
	seCheckEq(t, "height", core.height, float64(150))
	seCheckEq(t, "parentID", core.parentID, "frame999")
}

// seCaptureServer returns a server that checks for POST, captures the request
// body into gotBody, and responds with the given item id.
func seCaptureServer(t *testing.T, id string, gotBody *map[string]interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(gotBody); err != nil {
			t.Fatalf("decoding request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": id})
	}))
}

// seCheckStencilResult verifies the created item id and the stencil-shape wording.
func seCheckStencilResult(t *testing.T, gotID, wantID, message string) {
	t.Helper()
	seCheckEq(t, "ID", gotID, wantID)
	if !strings.Contains(message, "stencil shape") {
		t.Errorf("Message = %q, want it to mention a stencil shape", message)
	}
}

// seBlock extracts a nested block such as data or style from a captured request body.
func seBlock(t *testing.T, body map[string]interface{}, key string) map[string]interface{} {
	t.Helper()
	m, ok := body[key].(map[string]interface{})
	if !ok {
		t.Fatalf("missing %s block in request", key)
	}
	return m
}

func TestCreateShapeExperimental_Success(t *testing.T) {
	var gotBody map[string]interface{}
	server := seCaptureServer(t, "shape123", &gotBody)
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateShapeExperimental(context.Background(), CreateShapeExperimentalArgs{
		BoardID:   "board123",
		Shape:     "flow_chart_predefined_process",
		Content:   "Process step",
		FillColor: "#006400",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	seCheckStencilResult(t, result.ID, "shape123", result.Message)
	seCheckEq(t, "request fillColor", seBlock(t, gotBody, "style")["fillColor"], "#006400")
}

func TestCreateShapeExperimental_ValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		args CreateShapeExperimentalArgs
	}{
		{"empty board", CreateShapeExperimentalArgs{BoardID: "", Shape: "rectangle"}},
		{"empty shape", CreateShapeExperimentalArgs{BoardID: "board123", Shape: ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClientWithServer("http://unused.invalid")
			if _, err := client.CreateShapeExperimental(context.Background(), tt.args); err == nil {
				t.Fatalf("expected an error for %s", tt.name)
			}
		})
	}
}

func TestCreateShapeExperimental_Errors(t *testing.T) {
	tests := []struct {
		name    string
		handler func(*testing.T) http.HandlerFunc
		args    CreateShapeExperimentalArgs
		wantMsg string
	}{
		{
			name: "bad color is rejected before the call",
			handler: func(t *testing.T) http.HandlerFunc {
				return func(_ http.ResponseWriter, _ *http.Request) {
					t.Error("an invalid color must be rejected before any API call")
				}
			},
			args: CreateShapeExperimentalArgs{
				BoardID:   "board123",
				Shape:     "rectangle",
				FillColor: "not-a-real-color",
			},
			wantMsg: "expected an error for an invalid fill color",
		},
		{
			name: "API error",
			handler: func(_ *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
				}
			},
			args: CreateShapeExperimentalArgs{
				BoardID: "board123",
				Shape:   "rectangle",
			},
			wantMsg: "expected an error on HTTP 400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler(t))
			defer server.Close()

			client := newTestClientWithServer(server.URL)
			if _, err := client.CreateShapeExperimental(context.Background(), tt.args); err == nil {
				t.Fatal(tt.wantMsg)
			}
		})
	}
}

func TestCreateFlowchartShape_DelegatesToExperimental(t *testing.T) {
	var gotBody map[string]interface{}
	server := seCaptureServer(t, "flow123", &gotBody)
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.CreateFlowchartShape(context.Background(), CreateFlowchartShapeArgs{
		BoardID:     "board123",
		Shape:       "rhombus",
		Content:     "Decision?",
		Width:       220,
		Height:      120,
		BorderColor: "#000000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	seCheckStencilResult(t, result.ID, "flow123", result.Message)
	data := seBlock(t, gotBody, "data")
	seCheckEq(t, "request shape", data["shape"], "rhombus")
	seCheckEq(t, "request content", data["content"], "Decision?")
}

func TestCreateFlowchartShape_ValidationError(t *testing.T) {
	client := newTestClientWithServer("http://unused.invalid")
	if _, err := client.CreateFlowchartShape(context.Background(), CreateFlowchartShapeArgs{
		BoardID: "",
		Shape:   "rectangle",
	}); err == nil {
		t.Fatal("expected an error for an empty board_id")
	}
}

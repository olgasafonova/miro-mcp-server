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

func TestBuildExperimentalShapeStyle(t *testing.T) {
	t.Run("maps to the experimental key names", func(t *testing.T) {
		style, err := buildExperimentalShapeStyle("#006400", "#000000")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if style["fillColor"] != "#006400" {
			t.Errorf("fillColor = %v, want #006400", style["fillColor"])
		}
		if style["borderColor"] != "#000000" {
			t.Errorf("borderColor = %v, want #000000", style["borderColor"])
		}
		// The non-experimental endpoint uses "color"; this one must not.
		if _, present := style["color"]; present {
			t.Error("experimental style must not carry a plain color key")
		}
	})

	t.Run("resolves named colors", func(t *testing.T) {
		style, err := buildExperimentalShapeStyle("red", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if style["fillColor"] == "red" {
			t.Error("named color should resolve to a hex value")
		}
	})

	t.Run("omits empty colors", func(t *testing.T) {
		style, err := buildExperimentalShapeStyle("", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(style) != 0 {
			t.Errorf("style = %v, want empty when no colors are set", style)
		}
	})

	t.Run("rejects an unknown color", func(t *testing.T) {
		_, err := buildExperimentalShapeStyle("chartreuse-ish", "")
		if err == nil {
			t.Fatal("expected an error for an unrecognized color")
		}
		if !strings.Contains(err.Error(), "fill_color") {
			t.Errorf("error = %v, want the fill_color tag named", err)
		}
	})
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

	if core.boardID != "board123" {
		t.Errorf("boardID = %q, want board123", core.boardID)
	}
	if core.shape != "flow_chart_predefined_process" {
		t.Errorf("shape = %q", core.shape)
	}
	if core.content != "Process step" {
		t.Errorf("content = %q", core.content)
	}
	if core.x != 10 || core.y != 20 {
		t.Errorf("position = (%v, %v), want (10, 20)", core.x, core.y)
	}
	if core.width != 300 || core.height != 150 {
		t.Errorf("geometry = (%v, %v), want (300, 150)", core.width, core.height)
	}
	if core.parentID != "frame999" {
		t.Errorf("parentID = %q, want frame999", core.parentID)
	}
}

func TestCreateShapeExperimental_Success(t *testing.T) {
	var gotBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "shape123"})
	}))
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

	if result.ID != "shape123" {
		t.Errorf("ID = %q, want shape123", result.ID)
	}
	if !strings.Contains(result.Message, "stencil shape") {
		t.Errorf("Message = %q, want it to mention a stencil shape", result.Message)
	}

	style, ok := gotBody["style"].(map[string]interface{})
	if !ok {
		t.Fatal("missing style block in request")
	}
	if style["fillColor"] != "#006400" {
		t.Errorf("request fillColor = %v, want #006400", style["fillColor"])
	}
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

func TestCreateShapeExperimental_BadColorIsRejectedBeforeTheCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("an invalid color must be rejected before any API call")
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	if _, err := client.CreateShapeExperimental(context.Background(), CreateShapeExperimentalArgs{
		BoardID:   "board123",
		Shape:     "rectangle",
		FillColor: "not-a-real-color",
	}); err == nil {
		t.Fatal("expected an error for an invalid fill color")
	}
}

func TestCreateShapeExperimental_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	if _, err := client.CreateShapeExperimental(context.Background(), CreateShapeExperimentalArgs{
		BoardID: "board123",
		Shape:   "rectangle",
	}); err == nil {
		t.Fatal("expected an error on HTTP 400")
	}
}

func TestCreateFlowchartShape_DelegatesToExperimental(t *testing.T) {
	var gotBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "flow123"})
	}))
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

	if result.ID != "flow123" {
		t.Errorf("ID = %q, want flow123", result.ID)
	}
	if !strings.Contains(result.Message, "stencil shape") {
		t.Errorf("Message = %q, want the experimental stencil wording", result.Message)
	}

	data, ok := gotBody["data"].(map[string]interface{})
	if !ok {
		t.Fatal("missing data block in request")
	}
	if data["shape"] != "rhombus" {
		t.Errorf("request shape = %v, want rhombus", data["shape"])
	}
	if data["content"] != "Decision?" {
		t.Errorf("request content = %v, want Decision?", data["content"])
	}
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

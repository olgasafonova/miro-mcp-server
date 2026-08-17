package tools

import (
	"encoding/json"
	"strings"
	"testing"
)

type headerParamsTestArgs struct {
	BoardID string `json:"board_id" jsonschema:"Board ID"`
	Limit   int    `json:"limit,omitempty" jsonschema:"Max items"`
	Filter  struct {
		Kind string `json:"kind,omitempty"`
	} `json:"filter,omitempty"`
}

// TestHeaderAnnotatedSchemaAttachesAnnotation: the returned schema carries the
// x-mcp-header keyword on the named property, in the exact shape the SDK's
// collectParamHeaderAnnotations reads (a plain string on the property object).
func TestHeaderAnnotatedSchemaAttachesAnnotation(t *testing.T) {
	spec := ToolSpec{
		Name:         "test_tool",
		HeaderParams: map[string]string{"board_id": "Board-Id"},
	}

	schema := headerAnnotatedSchema[headerParamsTestArgs](spec)

	raw, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("marshaling schema: %v", err)
	}
	var decoded struct {
		Properties map[string]map[string]any `json:"properties"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshaling schema: %v", err)
	}

	got, ok := decoded.Properties["board_id"]["x-mcp-header"]
	if !ok {
		t.Fatalf("board_id property has no x-mcp-header keyword:\n%s", raw)
	}
	if got != "Board-Id" {
		t.Errorf("x-mcp-header = %v, want Board-Id", got)
	}

	// Un-annotated properties must stay clean.
	if _, ok := decoded.Properties["limit"]["x-mcp-header"]; ok {
		t.Error("limit property unexpectedly carries x-mcp-header")
	}

	// The rest of the inferred schema must survive: descriptions and types.
	if decoded.Properties["board_id"]["description"] != "Board ID" {
		t.Errorf("board_id description = %v, want the jsonschema tag value", decoded.Properties["board_id"]["description"])
	}
}

// TestHeaderAnnotatedSchemaPanicsOnUnknownProperty: a HeaderParams entry that
// names a property the Args struct does not have is a programming error and
// must fail loudly at registration, because the SDK silently ignores
// malformed annotations.
func TestHeaderAnnotatedSchemaPanicsOnUnknownProperty(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for unknown property")
		}
		if !strings.Contains(r.(string), "unknown property") {
			t.Errorf("panic %v does not mention unknown property", r)
		}
	}()
	headerAnnotatedSchema[headerParamsTestArgs](ToolSpec{
		Name:         "test_tool",
		HeaderParams: map[string]string{"nope": "Nope"},
	})
}

// TestHeaderAnnotatedSchemaPanicsOnNonPrimitive: x-mcp-header is only valid
// on string/integer/boolean properties; an object property must be rejected
// at registration rather than dropped from tools/list at serve time.
func TestHeaderAnnotatedSchemaPanicsOnNonPrimitive(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for non-primitive property")
		}
		if !strings.Contains(r.(string), "only string, integer, boolean") {
			t.Errorf("panic %v does not mention the allowed types", r)
		}
	}()
	headerAnnotatedSchema[headerParamsTestArgs](ToolSpec{
		Name:         "test_tool",
		HeaderParams: map[string]string{"filter": "Filter"},
	})
}

// TestAllToolsHeaderParamsAreWellFormed pins which tools declare header
// passthrough and that every declared header suffix is a valid HTTP token, so
// a typo fails here rather than being logged-and-dropped by the SDK's
// tools/list filter.
func TestAllToolsHeaderParamsAreWellFormed(t *testing.T) {
	want := map[string]map[string]string{
		"miro_get_board":  {"board_id": "Board-Id"},
		"miro_list_items": {"board_id": "Board-Id"},
	}

	got := collectHeaderParams()

	if len(got) != len(want) {
		t.Errorf("tools with HeaderParams = %v, want %v", got, want)
	}
	for name, params := range want {
		assertHeaderParamsMatch(t, name, got[name], params)
	}
}

// collectHeaderParams gathers the HeaderParams of every tool that declares any.
func collectHeaderParams() map[string]map[string]string {
	got := map[string]map[string]string{}
	for _, spec := range AllTools {
		if len(spec.HeaderParams) > 0 {
			got[spec.Name] = spec.HeaderParams
		}
	}
	return got
}

// assertHeaderParamsMatch checks one tool's declared header params against the
// expected property-to-header mapping.
func assertHeaderParamsMatch(t *testing.T, name string, got, want map[string]string) {
	t.Helper()
	for prop, header := range want {
		if got[prop] != header {
			t.Errorf("%s HeaderParams[%s] = %q, want %q", name, prop, got[prop], header)
		}
	}
}

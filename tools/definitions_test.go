package tools

import (
	"testing"
)

func TestAllToolsNotEmpty(t *testing.T) {
	if len(AllTools) == 0 {
		t.Error("AllTools should not be empty")
	}
}

func TestAllToolsHaveRequiredFields(t *testing.T) {
	for i, tool := range AllTools {
		t.Run(tool.Name, func(t *testing.T) {
			if tool.Name == "" {
				t.Errorf("tool[%d]: Name is required", i)
			}
			if tool.Method == "" {
				t.Errorf("tool[%d] %s: Method is required", i, tool.Name)
			}
			if tool.Description == "" {
				t.Errorf("tool[%d] %s: Description is required", i, tool.Name)
			}
			if tool.Title == "" {
				t.Errorf("tool[%d] %s: Title is required", i, tool.Name)
			}
			if tool.Category == "" {
				t.Errorf("tool[%d] %s: Category is required", i, tool.Name)
			}
		})
	}
}

func TestToolNamingConvention(t *testing.T) {
	for _, tool := range AllTools {
		t.Run(tool.Name, func(t *testing.T) {
			// All tool names should start with "miro_"
			if len(tool.Name) < 5 || tool.Name[:5] != "miro_" {
				t.Errorf("tool name %q should start with 'miro_'", tool.Name)
			}
		})
	}
}

func TestToolCategories(t *testing.T) {
	validCategories := map[string]bool{
		"boards":     true,
		"create":     true,
		"read":       true,
		"update":     true,
		"delete":     true,
		"tags":       true,
		"export":     true,
		"audit":      true,
		"webhooks":   true,
		"diagrams":   true,
		"connectors": true,
		"groups":     true,
		"members":    true,
		"frames":     true,
	}

	for _, tool := range AllTools {
		t.Run(tool.Name, func(t *testing.T) {
			if !validCategories[tool.Category] {
				t.Errorf("tool %q has unknown category %q", tool.Name, tool.Category)
			}
		})
	}
}

func TestReadOnlyToolsNotDestructive(t *testing.T) {
	for _, tool := range AllTools {
		t.Run(tool.Name, func(t *testing.T) {
			if tool.ReadOnly && tool.Destructive {
				t.Errorf("tool %q cannot be both ReadOnly and Destructive", tool.Name)
			}
		})
	}
}

func TestDestructiveToolsHaveWarning(t *testing.T) {
	for _, tool := range AllTools {
		if tool.Destructive {
			t.Run(tool.Name, func(t *testing.T) {
				if !containsWarning(tool.Description) {
					t.Errorf("destructive tool %q should have WARNING in description", tool.Name)
				}
			})
		}
	}
}

func containsWarning(s string) bool {
	for i := 0; i < len(s)-6; i++ {
		if s[i:i+7] == "WARNING" {
			return true
		}
	}
	return false
}

func TestToolCount(t *testing.T) {
	// Verify the expected number of tools
	// Phase 1-4: 38 tools, Phase 5: +1 audit (webhooks removed - Miro sunset Dec 2025), Phase 6: +1 diagram = 40
	// Quick wins: +2 tag tools (update, delete) + 2 connector tools (update, delete) = 44
	// New: +2 connector tools (list, get) = 46
	// v1.6.0: +1 update_board, +4 group tools (list, get, get_items, delete),
	//         +3 member tools (get, remove, update) = 54
	//         +4 app card tools (create, get, update, delete) = 58
	// v1.7.0: +4 frame tools (get, update, delete, get_items) = 62
	//         +3 mindmap tools (get, list, delete) = 65
	// v1.7.1: +1 tag tool (get_tag) = 66
	// v1.8.0: +2 bulk tools (bulk_update, bulk_delete) = 68
	// v1.9.0: +5 type-specific update tools (update_sticky, update_shape, update_text, update_card, update_group) = 73
	// v1.10.0: +3 type-specific update tools (update_image, update_document, update_embed) = 76
	// v1.11.0: +1 board content tool (get_board_content for AI analysis) = 77
	// v1.12.0: +2 read tools (get_image, get_document) = 79
	// desire paths: +1 desire path report tool = 80
	// v1.13.0: +3 doc format tools (create, get, delete) + 1 get_items_by_tag
	//          +1 upload_image + 1 create_flowchart_shape = 86
	// v1.14.0: +1 upload_document = 87
	//          +2 update_image_from_file, update_document_from_file = 89
	expectedCount := 89
	if len(AllTools) != expectedCount {
		t.Errorf("expected %d tools, got %d", expectedCount, len(AllTools))
	}
}

func TestToolNamesUnique(t *testing.T) {
	seen := make(map[string]bool)
	for _, tool := range AllTools {
		if seen[tool.Name] {
			t.Errorf("duplicate tool name: %q", tool.Name)
		}
		seen[tool.Name] = true
	}
}

func TestToolMethodsUnique(t *testing.T) {
	seen := make(map[string]bool)
	for _, tool := range AllTools {
		if seen[tool.Method] {
			t.Errorf("duplicate method: %q", tool.Method)
		}
		seen[tool.Method] = true
	}
}

func TestPtrHelper(t *testing.T) {
	// Test with int
	intVal := 42
	intPtr := ptr(intVal)
	if *intPtr != 42 {
		t.Errorf("ptr(42) = %d, want 42", *intPtr)
	}

	// Test with string
	strVal := "test"
	strPtr := ptr(strVal)
	if *strPtr != "test" {
		t.Errorf("ptr(\"test\") = %q, want \"test\"", *strPtr)
	}

	// Test with bool
	boolVal := true
	boolPtr := ptr(boolVal)
	if *boolPtr != true {
		t.Errorf("ptr(true) = %v, want true", *boolPtr)
	}
}

// BenchmarkToolLookup measures how long it takes to find a tool by name.
func BenchmarkToolLookup(b *testing.B) {
	targetName := "miro_create_sticky"
	for i := 0; i < b.N; i++ {
		for _, tool := range AllTools {
			if tool.Name == targetName {
				break
			}
		}
	}
}

package tools

import (
	"strings"
	"testing"
)

// TestAnnotationCoherence guards against a read-only tool also declaring
// idempotentHint. idempotentHint carries meaning only for tools that modify
// state: a read-only tool is trivially repeatable, so asserting idempotence
// on it says nothing and misleads a client reasoning about retry safety.
//
// This does not forbid Idempotent on its own. Most of the Idempotent specs in
// this repo sit on write tools, where the hint is exactly the useful signal —
// a repeated update converges, a repeated create does not.
func TestAnnotationCoherence(t *testing.T) {
	var incoherent []string
	for _, spec := range AllTools {
		if spec.ReadOnly && spec.Idempotent {
			incoherent = append(incoherent, spec.Name)
		}
	}

	if len(incoherent) > 0 {
		t.Errorf("%d of %d specs set both ReadOnly and Idempotent; drop Idempotent on read-only specs: %v",
			len(incoherent), len(AllTools), incoherent)
	}
}

// TestWriteToolDestructiveAudit pins the destructive classification: tools
// that delete, remove, or overwrite fields of an existing object are
// destructive; additive tools are not. Field-overwriting patch updates count
// as destructive (the conservative reading), so every update_* is true.
func TestWriteToolDestructiveAudit(t *testing.T) {
	overwriting := []string{"miro_update_", "miro_delete_", "miro_remove_", "miro_detach_", "miro_move_", "miro_resolve_"}
	additive := []string{"miro_create_", "miro_upload_", "miro_attach_", "miro_reply_", "miro_copy_", "miro_generate_", "miro_bulk_create"}
	for _, spec := range AllTools {
		if spec.ReadOnly {
			continue
		}
		if hasAnyPrefix(spec.Name, overwriting) && !spec.Destructive {
			t.Errorf("%s modifies existing state but is not Destructive", spec.Name)
		}
		if hasAnyPrefix(spec.Name, additive) && spec.Destructive {
			t.Errorf("%s is additive but marked Destructive", spec.Name)
		}
		if strings.HasPrefix(spec.Name, "miro_delete_") && !spec.Idempotent {
			t.Errorf("%s deletes by id and should be Idempotent", spec.Name)
		}
	}
}

// TestOpenWorldAudit checks that only the locally answered tools declare
// OpenWorld false; everything else calls the Miro API.
func TestOpenWorldAudit(t *testing.T) {
	local := map[string]bool{
		"miro_tool_search":      true,
		"miro_get_audit_log":    true,
		"miro_get_desire_paths": true,
	}
	for _, spec := range AllTools {
		if got, want := spec.openWorld(), !local[spec.Name]; got != want {
			t.Errorf("%s openWorld = %v, want %v", spec.Name, got, want)
		}
	}
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

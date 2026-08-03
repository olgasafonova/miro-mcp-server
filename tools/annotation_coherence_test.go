package tools

import "testing"

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

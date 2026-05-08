package miro

// =============================================================================
// Path / Endpoint Helpers
// =============================================================================

// knownPathSegments are API path segments that should NOT be treated as IDs.
var knownPathSegments = map[string]bool{
	"boards": true, "items": true, "sticky_notes": true, "shapes": true,
	"text": true, "connectors": true, "frames": true, "cards": true,
	"images": true, "documents": true, "embeds": true, "tags": true,
	"groups": true, "members": true, "mindmaps": true, "nodes": true,
	"export": true, "jobs": true, "picture": true, "copy": true,
	"orgs": true, "users": true, "me": true, "teams": true,
}

// extractEndpoint extracts a normalized endpoint from a path for circuit breaker.
// For example: /boards/abc123/items/xyz -> /boards/{id}/items/{id}
func extractEndpoint(path string) string {
	parts := make([]string, 0)
	for _, part := range splitPath(path) {
		// Skip query strings
		if idx := indexOf(part, "?"); idx != -1 {
			part = part[:idx]
		}
		if part == "" {
			continue
		}
		// Check if this is a known path segment
		if knownPathSegments[part] {
			parts = append(parts, part)
		} else {
			// This is likely an ID - replace with placeholder
			// Avoid consecutive {id} entries
			if len(parts) == 0 || parts[len(parts)-1] != "{id}" {
				parts = append(parts, "{id}")
			}
		}
	}
	return "/" + joinPath(parts)
}

func splitPath(path string) []string {
	if path == "" {
		return nil
	}
	if path[0] == '/' {
		path = path[1:]
	}
	result := make([]string, 0)
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func indexOf(s string, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func joinPath(parts []string) string {
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += "/"
		}
		result += part
	}
	return result
}

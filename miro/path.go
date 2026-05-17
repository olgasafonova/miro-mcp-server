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

// stripQuery returns part with any trailing "?..." removed.
func stripQuery(part string) string {
	if idx := indexOf(part, "?"); idx != -1 {
		return part[:idx]
	}
	return part
}

// classifyPathPart returns the normalized segment for an endpoint, or "" if the
// segment is empty after query-string stripping. Unknown segments collapse to
// "{id}" placeholders.
func classifyPathPart(part string) string {
	part = stripQuery(part)
	if part == "" {
		return ""
	}
	if knownPathSegments[part] {
		return part
	}
	return "{id}"
}

// extractEndpoint extracts a normalized endpoint from a path for circuit breaker.
// For example: /boards/abc123/items/xyz -> /boards/{id}/items/{id}
func extractEndpoint(path string) string {
	parts := make([]string, 0)
	for _, raw := range splitPath(path) {
		seg := classifyPathPart(raw)
		if seg == "" {
			continue
		}
		// Avoid consecutive {id} entries.
		if seg == "{id}" && len(parts) > 0 && parts[len(parts)-1] == "{id}" {
			continue
		}
		parts = append(parts, seg)
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

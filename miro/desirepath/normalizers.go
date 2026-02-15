package desirepath

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// =============================================================================
// URL-to-ID Normalizer
// =============================================================================

// URLPattern maps a parameter name to a regex that extracts an ID from a URL.
type URLPattern struct {
	ParamName string
	Pattern   *regexp.Regexp
	GroupIdx  int // Capture group index (1-based)
}

// URLToIDNormalizer extracts IDs from full URLs that agents paste into ID fields.
// Example: "https://miro.com/app/board/uXjVN123=/" -> "uXjVN123="
type URLToIDNormalizer struct {
	patterns []URLPattern
}

// NewURLToIDNormalizer creates a normalizer with the given URL patterns.
func NewURLToIDNormalizer(patterns []URLPattern) *URLToIDNormalizer {
	return &URLToIDNormalizer{patterns: patterns}
}

// MiroURLPatterns returns the standard URL patterns for Miro parameter extraction.
func MiroURLPatterns() []URLPattern {
	return []URLPattern{
		{
			ParamName: "board_id",
			Pattern:   regexp.MustCompile(`miro\.com/app/board/([^/?]+)`),
			GroupIdx:  1,
		},
		{
			ParamName: "item_id",
			Pattern:   regexp.MustCompile(`/items?/(\d+)`),
			GroupIdx:  1,
		},
	}
}

func (n *URLToIDNormalizer) Name() string { return "url_to_id" }

func (n *URLToIDNormalizer) Normalize(paramName string, rawValue any) (any, NormalizationResult) {
	str, ok := rawValue.(string)
	if !ok {
		return rawValue, NormalizationResult{}
	}

	// Only process if this looks like a URL
	if !strings.Contains(str, "://") && !strings.Contains(str, "miro.com") {
		return rawValue, NormalizationResult{}
	}

	for _, p := range n.patterns {
		if p.ParamName != paramName {
			continue
		}
		matches := p.Pattern.FindStringSubmatch(str)
		if len(matches) > p.GroupIdx {
			extracted := matches[p.GroupIdx]
			// Strip trailing slash if present
			extracted = strings.TrimRight(extracted, "/")
			return extracted, NormalizationResult{
				Changed:  true,
				Rule:     "url_to_id",
				Original: str,
				New:      extracted,
			}
		}
	}

	return rawValue, NormalizationResult{}
}

// =============================================================================
// CamelCase-to-snake_case Normalizer
// =============================================================================

// CamelToSnakeNormalizer converts camelCase JSON keys to snake_case.
// Agents trained on JavaScript conventions often send "boardId" instead of "board_id".
type CamelToSnakeNormalizer struct{}

func (n *CamelToSnakeNormalizer) Name() string { return "camel_to_snake" }

func (n *CamelToSnakeNormalizer) Normalize(paramName string, rawValue any) (any, NormalizationResult) {
	// This normalizer operates on parameter NAMES, not values.
	// It's used at the map key level, so we check if paramName itself is camelCase.
	// If it is, the caller should remap it.
	return rawValue, NormalizationResult{}
}

// ConvertKey checks if a key is camelCase and returns its snake_case equivalent.
// Returns the converted key and whether conversion happened.
func (n *CamelToSnakeNormalizer) ConvertKey(key string) (string, bool) {
	snake := camelToSnake(key)
	if snake != key {
		return snake, true
	}
	return key, false
}

// camelToSnake converts a camelCase string to snake_case.
func camelToSnake(s string) string {
	if s == "" {
		return s
	}

	var result strings.Builder
	result.Grow(len(s) + 4) // Typical overhead for underscores

	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				// Don't add underscore between consecutive uppercase (e.g., "ID" stays as "id")
				prev := rune(s[i-1])
				if !unicode.IsUpper(prev) {
					result.WriteRune('_')
				}
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// =============================================================================
// String-to-Numeric Normalizer
// =============================================================================

// numericParams lists parameter names that should be numeric.
var numericParams = map[string]bool{
	"limit":   true,
	"offset":  true,
	"page":    true,
	"x":       true,
	"y":       true,
	"width":   true,
	"height":  true,
	"columns": true,
}

// StringToNumericNormalizer converts string numbers to actual numbers.
// Agents often send "42" (string) instead of 42 (int) for numeric parameters.
type StringToNumericNormalizer struct {
	params map[string]bool
}

// NewStringToNumericNormalizer creates a normalizer for the given parameter names.
// If params is nil, uses the default set of numeric parameter names.
func NewStringToNumericNormalizer(params map[string]bool) *StringToNumericNormalizer {
	if params == nil {
		params = numericParams
	}
	return &StringToNumericNormalizer{params: params}
}

func (n *StringToNumericNormalizer) Name() string { return "string_to_numeric" }

func (n *StringToNumericNormalizer) Normalize(paramName string, rawValue any) (any, NormalizationResult) {
	if !n.params[paramName] {
		return rawValue, NormalizationResult{}
	}

	str, ok := rawValue.(string)
	if !ok {
		return rawValue, NormalizationResult{}
	}

	str = strings.TrimSpace(str)

	// Try integer first
	if i, err := strconv.ParseInt(str, 10, 64); err == nil {
		return float64(i), NormalizationResult{
			Changed:  true,
			Rule:     "string_to_numeric",
			Original: fmt.Sprintf("%q", str),
			New:      fmt.Sprintf("%d", i),
		}
	}

	// Try float
	if f, err := strconv.ParseFloat(str, 64); err == nil {
		return f, NormalizationResult{
			Changed:  true,
			Rule:     "string_to_numeric",
			Original: fmt.Sprintf("%q", str),
			New:      fmt.Sprintf("%g", f),
		}
	}

	return rawValue, NormalizationResult{}
}

// =============================================================================
// Whitespace Normalizer
// =============================================================================

// WhitespaceNormalizer trims whitespace and surrounding quotes from string values.
// Agents sometimes include leading/trailing spaces or wrap values in extra quotes.
type WhitespaceNormalizer struct{}

func (n *WhitespaceNormalizer) Name() string { return "whitespace" }

func (n *WhitespaceNormalizer) Normalize(paramName string, rawValue any) (any, NormalizationResult) {
	str, ok := rawValue.(string)
	if !ok {
		return rawValue, NormalizationResult{}
	}

	cleaned := strings.TrimSpace(str)

	// Strip surrounding quotes that agents sometimes add
	if len(cleaned) >= 2 {
		if (cleaned[0] == '"' && cleaned[len(cleaned)-1] == '"') ||
			(cleaned[0] == '\'' && cleaned[len(cleaned)-1] == '\'') {
			cleaned = cleaned[1 : len(cleaned)-1]
			cleaned = strings.TrimSpace(cleaned) // Trim again after removing quotes
		}
	}

	if cleaned != str {
		return cleaned, NormalizationResult{
			Changed:  true,
			Rule:     "whitespace",
			Original: fmt.Sprintf("%q", str),
			New:      cleaned,
		}
	}

	return rawValue, NormalizationResult{}
}

// =============================================================================
// Boolean Coercion Normalizer
// =============================================================================

// booleanParams lists parameter names that should be boolean.
var booleanParams = map[string]bool{
	"dry_run":      true,
	"deep":         true,
	"fuzzy":        true,
	"success":      true,
	"delete_items": true,
}

// BooleanCoercionNormalizer converts string booleans to actual booleans.
// Agents send "true"/"false" (strings) instead of true/false (booleans).
type BooleanCoercionNormalizer struct {
	params map[string]bool
}

// NewBooleanCoercionNormalizer creates a normalizer for the given parameter names.
// If params is nil, uses the default set of boolean parameter names.
func NewBooleanCoercionNormalizer(params map[string]bool) *BooleanCoercionNormalizer {
	if params == nil {
		params = booleanParams
	}
	return &BooleanCoercionNormalizer{params: params}
}

func (n *BooleanCoercionNormalizer) Name() string { return "boolean_coercion" }

func (n *BooleanCoercionNormalizer) Normalize(paramName string, rawValue any) (any, NormalizationResult) {
	if !n.params[paramName] {
		return rawValue, NormalizationResult{}
	}

	str, ok := rawValue.(string)
	if !ok {
		return rawValue, NormalizationResult{}
	}

	lower := strings.ToLower(strings.TrimSpace(str))
	switch lower {
	case "true", "1", "yes":
		return true, NormalizationResult{
			Changed:  true,
			Rule:     "boolean_coercion",
			Original: fmt.Sprintf("%q", str),
			New:      "true",
		}
	case "false", "0", "no":
		return false, NormalizationResult{
			Changed:  true,
			Rule:     "boolean_coercion",
			Original: fmt.Sprintf("%q", str),
			New:      "false",
		}
	}

	return rawValue, NormalizationResult{}
}

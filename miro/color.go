package miro

import (
	"fmt"
	"regexp"
	"strings"
)

var hexColorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// commonColorNames maps common color names to 6-character hex strings used by
// Miro's API for hex-only fields (frames, shapes, text, connectors, app card
// fields). Sticky notes and tags use Miro's own named-color enums and are
// normalized by normalizeStickyColor / normalizeTagColor instead.
var commonColorNames = map[string]string{
	"red":     "#FF0000",
	"orange":  "#FFA500",
	"yellow":  "#FFFF00",
	"green":   "#008000",
	"blue":    "#0000FF",
	"purple":  "#800080",
	"violet":  "#800080",
	"pink":    "#FFC0CB",
	"gray":    "#808080",
	"grey":    "#808080",
	"white":   "#FFFFFF",
	"black":   "#000000",
	"cyan":    "#00FFFF",
	"magenta": "#FF00FF",
	"brown":   "#A52A2A",
}

// SupportedColorNamesDescription is the human-readable list of color names
// accepted by normalizeColor. Used in tool descriptions so the LLM sees the
// exact set; keep in sync with commonColorNames.
const SupportedColorNamesDescription = "red, orange, yellow, green, blue, purple, pink, gray, white, black, cyan, magenta, brown"

// colorNameToHex returns the hex code for a common color name. The match is
// case-insensitive and tolerates surrounding whitespace. Returns ("", false)
// when the name is not in commonColorNames.
func colorNameToHex(name string) (string, bool) {
	hex, ok := commonColorNames[strings.ToLower(strings.TrimSpace(name))]
	return hex, ok
}

// normalizeColor converts a color name or hex string to the 6-character hex
// form Miro's API expects for hex-only color fields. Empty input is returned
// as-is so callers can omit the color field. Unrecognized input returns an
// error listing the accepted forms.
func normalizeColor(input string) (string, error) {
	if input == "" {
		return "", nil
	}
	trimmed := strings.TrimSpace(input)
	if hexColorPattern.MatchString(trimmed) {
		return trimmed, nil
	}
	if hex, ok := colorNameToHex(trimmed); ok {
		return hex, nil
	}
	return "", fmt.Errorf("unrecognized color %q: use a 6-char hex like #FF5733 or one of: %s", input, SupportedColorNamesDescription)
}

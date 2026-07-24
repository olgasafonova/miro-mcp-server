// Command token-count estimates the MCP context token cost of registering
// Miro MCP tools and generates a single shields.io-style SVG badge showing
// the context-window range across MIRO_TOOLS_PROFILE options (essentials →
// full). Always prints a per-profile comparison to stdout so the savings
// claim is reproducible from the repo.
//
// Usage:
//
//	go run ./cmd/token-count/
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/olgasafonova/miro-mcp-server/tools"
)

// contextWindow is the Claude context window size used for percentage calculation.
const contextWindow = 200_000

// avgSchemaBytesPerTool is the estimated average JSON Schema size per tool
// for input parameters (name, type, description, required fields).
const avgSchemaBytesPerTool = 200

// charsPerToken is the approximate character-to-token ratio for cl100k_base on JSON.
const charsPerToken = 4

// badgeValueColor is shields.io green: the badge leads with the achievable
// lean footprint, so the value reads as a positive (configurable) signal
// rather than a fixed price tag.
const badgeValueColor = "#4c1"

// mcpTool mirrors the MCP wire format for tools/list responses.
type mcpTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Annotations map[string]bool `json:"annotations,omitempty"`
}

// profileMeasurement captures the token-cost numbers for a registered
// profile so callers (badge generator, stdout printer) can share results.
type profileMeasurement struct {
	Profile      tools.Profile
	ToolCount    int
	DescBytes    int
	DescTokens   int
	SchemaTokens int
	TotalTokens  int
	Percentage   float64
}

func measureProfile(profile tools.Profile) (profileMeasurement, error) {
	specs := tools.ToolsForProfile(profile)
	wire := make([]mcpTool, 0, len(specs))
	for _, spec := range specs {
		t := mcpTool{Name: spec.Name, Description: spec.Description}
		annotations := make(map[string]bool)
		if spec.ReadOnly {
			annotations["readOnlyHint"] = true
		}
		if spec.Destructive {
			annotations["destructiveHint"] = true
		}
		if spec.Idempotent {
			annotations["idempotentHint"] = true
		}
		if len(annotations) > 0 {
			t.Annotations = annotations
		}
		wire = append(wire, t)
	}

	data, err := json.Marshal(wire)
	if err != nil {
		return profileMeasurement{}, fmt.Errorf("marshal %s: %w", profile, err)
	}

	descTokens := len(data) / charsPerToken
	schemaTokens := len(specs) * avgSchemaBytesPerTool / charsPerToken
	totalTokens := descTokens + schemaTokens

	return profileMeasurement{
		Profile:      profile,
		ToolCount:    len(specs),
		DescBytes:    len(data),
		DescTokens:   descTokens,
		SchemaTokens: schemaTokens,
		TotalTokens:  totalTokens,
		Percentage:   float64(totalTokens) / float64(contextWindow) * 100,
	}, nil
}

// formatTokens renders a token count as a compact "~K" string, e.g.
// 2571 -> "2.6K", 17236 -> "17.2K". Rounded to the nearest 100.
func formatTokens(n int) string {
	rounded := (n + 50) / 100 * 100
	return fmt.Sprintf("%.1fK", float64(rounded)/1000)
}

// writeRangeBadge writes a single badge spanning the leanest profile to the
// fullest, e.g. "MCP context | 2.6K–17.2K tokens". Leading with the low end
// frames the footprint as configurable (opt into essentials) rather than a
// fixed price. Token counts are model-agnostic — unlike a "% of context"
// figure, they don't go stale as context-window sizes change. The default
// filename is kept so existing README references hold.
func writeRangeBadge(low, high profileMeasurement) (string, error) {
	label := "MCP context"
	value := fmt.Sprintf("%s–%s tokens", formatTokens(low.TotalTokens), formatTokens(high.TotalTokens))
	svg := generateBadge(label, value, badgeValueColor)

	path := filepath.Join("badges", "mcp-tokens.svg")
	if err := os.WriteFile(path, []byte(svg), 0600); err != nil {
		return "", err
	}
	return path, nil
}

func main() {
	full, err := measureProfile(tools.ProfileFull)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	essentials, err := measureProfile(tools.ProfileEssentials)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	badge, err := writeRangeBadge(essentials, full)
	if err != nil {
		fmt.Fprintln(os.Stderr, "writing badge:", err)
		os.Exit(1)
	}

	saved := full.TotalTokens - essentials.TotalTokens
	pct := float64(saved) / float64(full.TotalTokens) * 100

	fmt.Println("Profile     Tools  DescTokens  SchemaTokens  Total   % of 200K")
	fmt.Println("----------  -----  ----------  ------------  ------  ---------")
	fmt.Printf("%-10s  %5d  %10d  %12d  %6d  %5.1f%%\n",
		full.Profile, full.ToolCount, full.DescTokens, full.SchemaTokens, full.TotalTokens, full.Percentage)
	fmt.Printf("%-10s  %5d  %10d  %12d  %6d  %5.1f%%\n",
		essentials.Profile, essentials.ToolCount, essentials.DescTokens, essentials.SchemaTokens, essentials.TotalTokens, essentials.Percentage)
	fmt.Println()
	fmt.Printf("Savings (essentials vs full): %d tokens (%.1f%% reduction)\n", saved, pct)
	fmt.Println()
	fmt.Println("Badge written:")
	fmt.Println("  ", badge)
}

// generateBadge creates a shields.io-style SVG badge with the given value color.
func generateBadge(label, value, valueColor string) string {
	// Approximate text widths using Verdana 11px metrics (~6.5px per char).
	const charWidth = 6.5
	const padding = 10.0

	// Count runes, not bytes, so a multibyte glyph (the en dash) doesn't
	// over-pad the value box.
	labelWidth := float64(len([]rune(label)))*charWidth + 2*padding
	valueWidth := float64(len([]rune(value)))*charWidth + 2*padding
	totalWidth := labelWidth + valueWidth

	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="20" role="img" aria-label="%s: %s">
  <title>%s: %s</title>
  <linearGradient id="s" x2="0" y2="100%%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <clipPath id="r">
    <rect width="%.0f" height="20" rx="3" fill="#fff"/>
  </clipPath>
  <g clip-path="url(#r)">
    <rect width="%.0f" height="20" fill="#555"/>
    <rect x="%.0f" width="%.0f" height="20" fill="%s"/>
    <rect width="%.0f" height="20" fill="url(#s)"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" text-rendering="geometricPrecision" font-size="11">
    <text aria-hidden="true" x="%.1f" y="15" fill="#010101" fill-opacity=".3">%s</text>
    <text x="%.1f" y="14">%s</text>
    <text aria-hidden="true" x="%.1f" y="15" fill="#010101" fill-opacity=".3">%s</text>
    <text x="%.1f" y="14">%s</text>
  </g>
</svg>`,
		totalWidth, label, value,
		label, value,
		totalWidth,
		labelWidth,
		labelWidth, valueWidth, valueColor,
		totalWidth,
		labelWidth/2, label,
		labelWidth/2, label,
		labelWidth+valueWidth/2, value,
		labelWidth+valueWidth/2, value,
	)
}

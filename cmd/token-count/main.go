// Command token-count estimates the MCP context token cost of registering
// Miro MCP tools and generates shields.io-style SVG badges, one per
// MIRO_TOOLS_PROFILE option (full / essentials). Always prints a comparison
// to stdout so the savings claim is reproducible from the repo.
//
// Usage:
//
//	go run ./cmd/token-count/
package main

import (
	"encoding/json"
	"fmt"
	"math"
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

// badgeFilename returns the relative output path for a profile's SVG badge.
// The default profile keeps the historical filename so existing README
// references don't break.
func badgeFilename(profile tools.Profile) string {
	if profile == tools.ProfileFull {
		return filepath.Join("badges", "mcp-tokens.svg")
	}
	return filepath.Join("badges", "mcp-tokens-"+string(profile)+".svg")
}

func writeBadge(m profileMeasurement) (string, error) {
	displayTokens := int(math.Round(float64(m.TotalTokens)/100) * 100)
	displayK := fmt.Sprintf("~%dK", displayTokens/1000)
	if displayTokens < 1000 {
		displayK = fmt.Sprintf("~%d", displayTokens)
	}

	label := "MCP Context"
	if m.Profile != tools.ProfileFull {
		label = fmt.Sprintf("MCP Context (%s)", m.Profile)
	}
	value := fmt.Sprintf("%s tokens (%.1f%%)", displayK, m.Percentage)
	svg := generateBadge(label, value)

	path := badgeFilename(m.Profile)
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

	fullBadge, err := writeBadge(full)
	if err != nil {
		fmt.Fprintln(os.Stderr, "writing full badge:", err)
		os.Exit(1)
	}
	essBadge, err := writeBadge(essentials)
	if err != nil {
		fmt.Fprintln(os.Stderr, "writing essentials badge:", err)
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
	fmt.Println("Badges written:")
	fmt.Println("  ", fullBadge)
	fmt.Println("  ", essBadge)
}

// generateBadge creates a shields.io-style SVG badge.
func generateBadge(label, value string) string {
	// Approximate text widths using Verdana 11px metrics (~6.5px per char).
	const charWidth = 6.5
	const padding = 10.0

	labelWidth := float64(len(label))*charWidth + 2*padding
	valueWidth := float64(len(value))*charWidth + 2*padding
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
    <rect x="%.0f" width="%.0f" height="20" fill="#007ec6"/>
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
		labelWidth, valueWidth,
		totalWidth,
		labelWidth/2, label,
		labelWidth/2, label,
		labelWidth+valueWidth/2, value,
		labelWidth+valueWidth/2, value,
	)
}

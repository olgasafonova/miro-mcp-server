// Command token-count estimates the MCP context token cost of registering all
// Miro MCP tools and generates a shields.io-style SVG badge.
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

func main() {
	// Build MCP wire-format representation of each tool.
	mcpTools := make([]mcpTool, 0, len(tools.AllTools))
	for _, spec := range tools.AllTools {
		t := mcpTool{
			Name:        spec.Name,
			Description: spec.Description,
		}
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
		mcpTools = append(mcpTools, t)
	}

	// Marshal to JSON to measure description payload size.
	data, err := json.Marshal(mcpTools)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling tools: %v\n", err)
		os.Exit(1)
	}

	descTokens := len(data) / charsPerToken
	schemaTokens := len(tools.AllTools) * avgSchemaBytesPerTool / charsPerToken
	totalTokens := descTokens + schemaTokens
	percentage := float64(totalTokens) / float64(contextWindow) * 100

	// Round to nearest 100 for display.
	displayTokens := int(math.Round(float64(totalTokens)/100) * 100)
	displayK := fmt.Sprintf("~%dK", displayTokens/1000)
	if displayTokens < 1000 {
		displayK = fmt.Sprintf("~%d", displayTokens)
	}

	label := "MCP Context"
	value := fmt.Sprintf("%s tokens (%.1f%%)", displayK, percentage)

	// Generate badge SVG.
	svg := generateBadge(label, value)

	badgePath := filepath.Join("badges", "mcp-tokens.svg")
	if err := os.WriteFile(badgePath, []byte(svg), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "error writing badge: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Tools:             %d\n", len(tools.AllTools))
	fmt.Printf("Description JSON:  %d bytes (%d tokens)\n", len(data), descTokens)
	fmt.Printf("Schema estimate:   %d bytes (%d tokens)\n",
		len(tools.AllTools)*avgSchemaBytesPerTool, schemaTokens)
	fmt.Printf("Total estimate:    %d tokens (%.1f%% of %dK context)\n",
		totalTokens, percentage, contextWindow/1000)
	fmt.Printf("Badge written:     %s\n", badgePath)
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

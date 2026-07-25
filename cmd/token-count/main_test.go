package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/olgasafonova/miro-mcp-server/tools"
)

// =============================================================================
// token-count tests
//
// main() is not covered: it calls os.Exit and writes into the repo's badges/
// directory relative to the working directory.
// =============================================================================

func TestMeasureProfile(t *testing.T) {
	for _, profile := range []tools.Profile{tools.ProfileFull, tools.ProfileEssentials} {
		t.Run(string(profile), func(t *testing.T) {
			m, err := measureProfile(profile)
			if err != nil {
				t.Fatalf("measureProfile(%s) error = %v", profile, err)
			}

			if m.Profile != profile {
				t.Errorf("Profile = %v, want %v", m.Profile, profile)
			}
			if m.ToolCount <= 0 {
				t.Errorf("ToolCount = %d, want a positive count", m.ToolCount)
			}
			if m.DescBytes <= 0 {
				t.Errorf("DescBytes = %d, want positive", m.DescBytes)
			}

			// The arithmetic the badge depends on.
			if want := m.DescBytes / charsPerToken; m.DescTokens != want {
				t.Errorf("DescTokens = %d, want %d", m.DescTokens, want)
			}
			if want := m.ToolCount * avgSchemaBytesPerTool / charsPerToken; m.SchemaTokens != want {
				t.Errorf("SchemaTokens = %d, want %d", m.SchemaTokens, want)
			}
			if want := m.DescTokens + m.SchemaTokens; m.TotalTokens != want {
				t.Errorf("TotalTokens = %d, want %d", m.TotalTokens, want)
			}
			if want := float64(m.TotalTokens) / float64(contextWindow) * 100; m.Percentage != want {
				t.Errorf("Percentage = %v, want %v", m.Percentage, want)
			}
		})
	}
}

func TestMeasureProfile_EssentialsIsLeanerThanFull(t *testing.T) {
	full, err := measureProfile(tools.ProfileFull)
	if err != nil {
		t.Fatalf("full: %v", err)
	}
	essentials, err := measureProfile(tools.ProfileEssentials)
	if err != nil {
		t.Fatalf("essentials: %v", err)
	}

	if essentials.ToolCount >= full.ToolCount {
		t.Errorf("essentials tools (%d) should be fewer than full (%d)",
			essentials.ToolCount, full.ToolCount)
	}
	if essentials.TotalTokens >= full.TotalTokens {
		t.Errorf("essentials tokens (%d) should be fewer than full (%d)",
			essentials.TotalTokens, full.TotalTokens)
	}
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{2571, "2.6K"}, // the doc-comment examples
		{17236, "17.2K"},
		{0, "0.0K"},
		{49, "0.0K"}, // rounds down to 0
		{50, "0.1K"}, // rounds up to 100
		{1000, "1.0K"},
		{1049, "1.0K"},
		{1050, "1.1K"}, // rounding boundary
		{999999, "1000.0K"},
	}

	for _, tt := range tests {
		if got := formatTokens(tt.n); got != tt.want {
			t.Errorf("formatTokens(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestGenerateBadge(t *testing.T) {
	svg := generateBadge("MCP context", "2.6K–17.2K tokens", badgeValueColor)

	if !strings.HasPrefix(svg, "<svg") {
		t.Error("output should be an SVG element")
	}
	if !strings.Contains(svg, "MCP context") {
		t.Error("label should appear in the SVG")
	}
	if !strings.Contains(svg, "2.6K–17.2K tokens") {
		t.Error("value should appear in the SVG")
	}
	if !strings.Contains(svg, badgeValueColor) {
		t.Errorf("value colour %s should appear in the SVG", badgeValueColor)
	}
	if !strings.Contains(svg, `role="img"`) {
		t.Error("badge should carry role=img for accessibility")
	}
	if !strings.Contains(svg, "aria-label=") {
		t.Error("badge should carry an aria-label")
	}
}

func TestGenerateBadge_WidthCountsRunesNotBytes(t *testing.T) {
	// The en dash is 3 bytes but one glyph; width must not over-pad for it.
	withDash := generateBadge("L", "a–b", badgeValueColor)
	withASCII := generateBadge("L", "a-b", badgeValueColor)

	widthOf := func(svg string) string {
		start := strings.Index(svg, `width="`) + len(`width="`)
		return svg[start : start+strings.Index(svg[start:], `"`)]
	}

	if widthOf(withDash) != widthOf(withASCII) {
		t.Errorf("width with en dash = %s, with ASCII hyphen = %s; should match",
			widthOf(withDash), widthOf(withASCII))
	}
}

func TestWriteRangeBadge(t *testing.T) {
	// writeRangeBadge writes to badges/ relative to the working directory.
	tmp := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(orig) })

	if err := os.Mkdir("badges", 0o750); err != nil {
		t.Fatalf("creating badges dir: %v", err)
	}

	low := profileMeasurement{TotalTokens: 2571}
	high := profileMeasurement{TotalTokens: 17236}

	path, err := writeRangeBadge(low, high)
	if err != nil {
		t.Fatalf("writeRangeBadge error = %v", err)
	}
	if path != filepath.Join("badges", "mcp-tokens.svg") {
		t.Errorf("path = %q, want badges/mcp-tokens.svg", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading badge: %v", err)
	}
	svg := string(content)
	if !strings.Contains(svg, "MCP context") {
		t.Error("badge should carry the MCP context label")
	}
	if !strings.Contains(svg, "2.6K–17.2K tokens") {
		t.Errorf("badge should span low to high; got:\n%s", svg)
	}
}

func TestWriteRangeBadge_MissingDirIsAnError(t *testing.T) {
	tmp := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(orig) })

	// No badges/ directory created.
	if _, err := writeRangeBadge(profileMeasurement{}, profileMeasurement{}); err == nil {
		t.Fatal("expected an error when badges/ does not exist")
	}
}

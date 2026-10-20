package evals

import (
	"fmt"
	"strings"
)

// FormatMetrics returns a human-readable summary of evaluation metrics.
func FormatMetrics(metrics *EvalMetrics, suiteName string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("\n=== %s ===\n", suiteName))
	b.WriteString(fmt.Sprintf("Total: %d tests\n", metrics.TotalTests))
	b.WriteString(fmt.Sprintf("Passed: %d (%.1f%%)\n", metrics.PassedTests, metrics.Accuracy*100))
	b.WriteString(fmt.Sprintf("Failed: %d\n", metrics.FailedTests))

	writeBreakdown(&b, "By Category", metrics.ByCategory, 30)
	writeBreakdown(&b, "By Difficulty", metrics.ByDifficulty, 10)
	writeFailureSection(&b, metrics.FailedDetails)

	return b.String()
}

// writeBreakdown renders one "By X" block with right-aligned padding.
func writeBreakdown(b *strings.Builder, title string, breakdown map[string]*CategoryMetrics, padding int) {
	if len(breakdown) == 0 {
		return
	}
	b.WriteString("\n")
	b.WriteString(title)
	b.WriteString(":\n")
	for name, m := range breakdown {
		if m.Total > 0 {
			acc := float64(m.Passed) / float64(m.Total) * 100
			b.WriteString(fmt.Sprintf("  %-*s: %d/%d (%.0f%%)\n", padding, name, m.Passed, m.Total, acc))
		}
	}
}

// writeFailureSection renders the trailing "Failed Tests" block, capping at 10
// entries with a count when truncated.
func writeFailureSection(b *strings.Builder, details []string) {
	if len(details) == 0 {
		return
	}
	if len(details) <= 10 {
		b.WriteString("\nFailed Tests:\n")
	} else {
		b.WriteString(fmt.Sprintf("\nFailed Tests (showing first 10 of %d):\n", len(details)))
		details = details[:10]
	}
	for _, detail := range details {
		b.WriteString(fmt.Sprintf("  - %s\n", detail))
	}
}

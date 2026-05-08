package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/olgasafonova/miro-mcp-server/miro"
	"github.com/olgasafonova/miro-mcp-server/miro/desirepath"
)

// GetDesirePathReport queries the local desire path log and returns the
// aggregated report in the API response shape.
func (h *HandlerRegistry) GetDesirePathReport(_ context.Context, args miro.GetDesirePathReportArgs) (miro.GetDesirePathReportResult, error) {
	if h.desireLogger == nil {
		return miro.GetDesirePathReportResult{
			Message: "Desire path logging is not enabled",
		}, nil
	}

	limit := clampLimit(args.Limit, 20, 100)
	report := h.desireLogger.Report()

	return miro.GetDesirePathReportResult{
		TotalNormalizations: report.TotalEvents,
		ByRule:              report.ByRule,
		ByTool:              report.ByTool,
		ByParam:             report.ByParam,
		TopPatterns:         convertPatterns(report.TopPatterns),
		RecentEvents:        filterRecentEvents(report.RecentOnes, args.Tool, args.Rule, limit),
		Message:             fmt.Sprintf("Recorded %d normalizations across %d patterns", report.TotalEvents, len(report.TopPatterns)),
	}, nil
}

// filterRecentEvents keeps events that match both the tool and rule filters,
// up to limit. Empty filter strings match anything.
func filterRecentEvents(events []desirepath.Event, tool, rule string, limit int) []miro.DesirePathEvent {
	out := make([]miro.DesirePathEvent, 0, len(events))
	for _, e := range events {
		if !matchesFilter(e, tool, rule) {
			continue
		}
		if len(out) >= limit {
			break
		}
		out = append(out, miro.DesirePathEvent{
			Timestamp:    e.Timestamp.Format(time.RFC3339),
			Tool:         e.Tool,
			Parameter:    e.Parameter,
			Rule:         e.Rule,
			RawValue:     e.RawValue,
			NormalizedTo: e.NormalizedTo,
		})
	}
	return out
}

// matchesFilter reports whether an event matches the tool and rule filters.
// Empty filter strings match anything.
func matchesFilter(e desirepath.Event, tool, rule string) bool {
	if tool != "" && e.Tool != tool {
		return false
	}
	if rule != "" && e.Rule != rule {
		return false
	}
	return true
}

// convertPatterns maps internal pattern summaries to the API response type.
func convertPatterns(patterns []desirepath.PatternSummary) []miro.DesirePathPattern {
	out := make([]miro.DesirePathPattern, len(patterns))
	for i, p := range patterns {
		out[i] = miro.DesirePathPattern{
			Rule:      p.Rule,
			Tool:      p.Tool,
			Parameter: p.Parameter,
			Example:   p.Example,
			Count:     p.Count,
		}
	}
	return out
}

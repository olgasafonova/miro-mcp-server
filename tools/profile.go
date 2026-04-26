package tools

// Profile selects which tools the MCP server registers at startup. Operators
// pick this via MIRO_TOOLS_PROFILE: full (default — all 91 tools) or
// essentials (the meta-tool plus ~15 high-frequency tools). The essentials
// profile is graceful degradation for clients without API-level tool_search;
// agents discover the rest via miro_tool_search on demand.
type Profile string

const (
	// ProfileFull registers every tool in AllTools.
	ProfileFull Profile = "full"

	// ProfileEssentials registers only meta-tools plus a curated short list.
	ProfileEssentials Profile = "essentials"
)

// EssentialsToolNames is the list of MCP tool names registered when
// MIRO_TOOLS_PROFILE=essentials. Chosen for the most common workflow
// fragments (find, browse, create core item types, edit/delete). Anything
// not on this list is reachable via miro_tool_search.
var EssentialsToolNames = []string{
	ToolSearchName,
	"miro_list_boards",
	"miro_find_board",
	"miro_get_board_summary",
	"miro_get_board_content",
	"miro_create_sticky",
	"miro_create_text",
	"miro_create_frame",
	"miro_create_connector",
	"miro_list_items",
	"miro_get_item",
	"miro_search_board",
	"miro_bulk_create",
	"miro_delete_item",
	"miro_update_item",
}

// ParseProfile maps an operator-supplied string (typically the value of
// MIRO_TOOLS_PROFILE) to a Profile. Empty / unknown values fall back to
// ProfileFull so a typo never silently strips tools the operator was
// relying on.
func ParseProfile(s string) (Profile, bool) {
	switch s {
	case "":
		return ProfileFull, true
	case string(ProfileFull):
		return ProfileFull, true
	case string(ProfileEssentials):
		return ProfileEssentials, true
	}
	return ProfileFull, false
}

// ToolsForProfile filters AllTools to those that belong in the given profile.
// ProfileFull returns everything (in declaration order). ProfileEssentials
// returns the entries listed in EssentialsToolNames, in that order. Tools
// listed in EssentialsToolNames but missing from AllTools are silently
// skipped — the caller is expected to maintain consistency.
func ToolsForProfile(profile Profile) []ToolSpec {
	if profile != ProfileEssentials {
		return AllTools
	}

	byName := make(map[string]ToolSpec, len(AllTools))
	for _, spec := range AllTools {
		byName[spec.Name] = spec
	}

	out := make([]ToolSpec, 0, len(EssentialsToolNames))
	for _, name := range EssentialsToolNames {
		if spec, ok := byName[name]; ok {
			out = append(out, spec)
		}
	}
	return out
}

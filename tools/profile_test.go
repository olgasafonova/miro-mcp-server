package tools

import "testing"

func TestParseProfile(t *testing.T) {
	tests := []struct {
		input  string
		want   Profile
		wantOK bool
	}{
		{"", ProfileFull, true},
		{"full", ProfileFull, true},
		{"essentials", ProfileEssentials, true},
		{"FULL", ProfileFull, false}, // case-sensitive on purpose; surface the typo
		{"core", ProfileFull, false},
		{"unknown", ProfileFull, false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := ParseProfile(tt.input)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("ParseProfile(%q) = (%v, %v), want (%v, %v)",
					tt.input, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestToolsForProfile_FullReturnsAll(t *testing.T) {
	got := ToolsForProfile(ProfileFull)
	if len(got) != len(AllTools) {
		t.Errorf("ProfileFull returned %d tools, want %d", len(got), len(AllTools))
	}
}

func TestToolsForProfile_EssentialsHasMetaToolFirst(t *testing.T) {
	got := ToolsForProfile(ProfileEssentials)
	if len(got) == 0 {
		t.Fatal("ProfileEssentials returned empty list")
	}
	if got[0].Name != ToolSearchName {
		t.Errorf("essentials profile should start with %s, got %s", ToolSearchName, got[0].Name)
	}
}

func TestToolsForProfile_EssentialsListMatchesNames(t *testing.T) {
	got := ToolsForProfile(ProfileEssentials)
	gotNames := make(map[string]bool, len(got))
	for _, spec := range got {
		gotNames[spec.Name] = true
	}
	for _, want := range EssentialsToolNames {
		// EssentialsToolNames may legitimately include names not in AllTools
		// (the function silently skips those), so don't fail on absence —
		// but every entry that *is* in AllTools should be present.
		if !specInAllTools(want) {
			continue
		}
		if !gotNames[want] {
			t.Errorf("essentials profile missing %s", want)
		}
	}
}

func TestToolsForProfile_EssentialsCoverageOfRealTools(t *testing.T) {
	// Sanity check: every name in EssentialsToolNames should resolve to a real
	// tool in AllTools. If not, the curated list has drifted from reality and
	// should be updated.
	for _, name := range EssentialsToolNames {
		if !specInAllTools(name) {
			t.Errorf("essentials list references unknown tool %q — fix EssentialsToolNames or remove it", name)
		}
	}
}

func specInAllTools(name string) bool {
	for _, spec := range AllTools {
		if spec.Name == name {
			return true
		}
	}
	return false
}

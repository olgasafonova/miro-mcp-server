package miro

import (
	"strings"
	"testing"
)

// =============================================================================
// Small pure-helper tests (URL building, group dry-run messaging)
// =============================================================================

func TestBuildBoardURL(t *testing.T) {
	tests := []struct {
		name    string
		boardID string
		want    string
	}{
		{"empty board id yields empty string", "", ""},
		{"normal board id", "board123", MiroAppBaseURL + "/board123/"},
		{"id with special characters is passed through", "uXjVK-abc_123=", MiroAppBaseURL + "/uXjVK-abc_123=/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildBoardURL(tt.boardID); got != tt.want {
				t.Errorf("BuildBoardURL(%q) = %q, want %q", tt.boardID, got, tt.want)
			}
		})
	}
}

func TestBuildBoardURL_DiffersFromItemURL(t *testing.T) {
	board := BuildBoardURL("board123")
	item := BuildItemURL("board123", "item456")

	if board == item {
		t.Error("board and item URLs should differ")
	}
	if strings.Contains(board, "moveToWidget") {
		t.Errorf("board URL = %q, should carry no widget anchor", board)
	}
}

func TestDeleteGroupDryRunMessage(t *testing.T) {
	tests := []struct {
		name string
		args DeleteGroupArgs
		want string
	}{
		{
			name: "deleting items says so",
			args: DeleteGroupArgs{GroupID: "group123", DeleteItems: true},
			want: "[DRY RUN] Would delete group group123 and its items",
		},
		{
			name: "keeping items says they are ungrouped",
			args: DeleteGroupArgs{GroupID: "group123", DeleteItems: false},
			want: "[DRY RUN] Would delete group group123, items would be ungrouped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deleteGroupDryRunMessage(tt.args)
			if got != tt.want {
				t.Errorf("message = %q, want %q", got, tt.want)
			}
			if !strings.HasPrefix(got, "[DRY RUN]") {
				t.Errorf("message = %q, want a [DRY RUN] prefix", got)
			}
		})
	}
}

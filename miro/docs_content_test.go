package miro

import (
	"strings"
	"testing"
)

// =============================================================================
// Doc Format Content Resolution Tests
// =============================================================================
// Split from docs_test.go: pure-function tests for the content helpers behind
// UpdateDocFormat (find-and-replace, content resolution, update message).

// docContentCase is one scenario for the content-resolution helpers under test.
type docContentCase struct {
	name        string
	input       string
	args        UpdateDocFormatArgs
	wantContent string
	wantCount   int
	wantErr     string
}

// runDocContentCases exercises a content-resolution function against a set of cases.
func runDocContentCases(t *testing.T, tests []docContentCase, fn func(string, UpdateDocFormatArgs) (string, int, error)) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, count, err := fn(tt.input, tt.args)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantContent {
				t.Errorf("content = %q, want %q", got, tt.wantContent)
			}
			if count != tt.wantCount {
				t.Errorf("count = %d, want %d", count, tt.wantCount)
			}
		})
	}
}

func TestFindAndReplaceContent(t *testing.T) {
	runDocContentCases(t, []docContentCase{
		{
			name:        "replaces first occurrence only",
			input:       "foo bar foo",
			args:        UpdateDocFormatArgs{OldContent: "foo", NewContent: "baz"},
			wantContent: "baz bar foo",
			wantCount:   1,
		},
		{
			name:        "replaces all occurrences",
			input:       "foo bar foo foo",
			args:        UpdateDocFormatArgs{OldContent: "foo", NewContent: "baz", ReplaceAll: true},
			wantContent: "baz bar baz baz",
			wantCount:   3,
		},
		{
			name:    "missing old content is an error",
			input:   "foo bar",
			args:    UpdateDocFormatArgs{OldContent: "nope", NewContent: "baz"},
			wantErr: "old_content not found in document",
		},
	}, findAndReplaceContent)
}

func TestResolveDocFormatContent(t *testing.T) {
	runDocContentCases(t, []docContentCase{
		{
			name:        "find-and-replace takes precedence",
			input:       "hello world",
			args:        UpdateDocFormatArgs{Content: "ignored", OldContent: "world", NewContent: "there"},
			wantContent: "hello there",
			wantCount:   1,
		},
		{
			name:        "full replacement when no old_content",
			input:       "hello world",
			args:        UpdateDocFormatArgs{Content: "brand new"},
			wantContent: "brand new",
			wantCount:   0,
		},
		{
			name:    "neither mode set is an error",
			input:   "hello world",
			args:    UpdateDocFormatArgs{},
			wantErr: "either content (full replace) or old_content+new_content",
		},
	}, resolveDocFormatContent)
}

func TestDocFormatUpdateMessage(t *testing.T) {
	if got := docFormatUpdateMessage(0); got != "Updated doc format item" {
		t.Errorf("message(0) = %q", got)
	}
	if got := docFormatUpdateMessage(3); got != "Replaced 3 occurrence(s) in doc format item" {
		t.Errorf("message(3) = %q", got)
	}
}

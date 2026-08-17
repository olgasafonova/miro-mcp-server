package miro

// Tests for the pure string/path utility helpers, split out of client_test.go.

import (
	"testing"
)

// failUtil reports a mismatch for a utility function result.
func failUtil(t *testing.T, desc string, got, want interface{}) {
	t.Helper()
	t.Errorf("%s = %v, want %v", desc, got, want)
}

// checkUtilResult fails the test when got differs from want.
func checkUtilResult[T comparable](t *testing.T, desc string, got, want T) {
	t.Helper()
	if got != want {
		failUtil(t, desc, got, want)
	}
}

// checkUtilErr fails the test when the error presence differs from wantErr.
func checkUtilErr(t *testing.T, desc string, err error, wantErr bool) {
	t.Helper()
	if (err != nil) != wantErr {
		failUtil(t, desc+" error", err, wantErr)
	}
}

// checkStringSlice fails the test when got differs from want element-wise.
func checkStringSlice(t *testing.T, desc string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		failUtil(t, desc, got, want)
		return
	}
	for i := range got {
		checkUtilResult(t, desc+" element", got[i], want[i])
	}
}

func TestValidateBoardID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"valid alphanumeric", "abc123", false},
		{"valid with underscore", "board_123", false},
		{"valid with hyphen", "board-123", false},
		{"valid with equals", "board=123", false},
		{"empty", "", true},
		{"too long", string(make([]byte, 101)), true},
		{"invalid chars space", "board 123", true},
		{"invalid chars slash", "board/123", true},
		{"invalid chars dot", "board.123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkUtilErr(t, "ValidateBoardID("+tt.id+")", ValidateBoardID(tt.id), tt.wantErr)
		})
	}
}

func TestValidateContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{"valid short", "Hello", false},
		{"valid empty", "", false},
		{"valid at limit", string(make([]byte, maxContentLen)), false},
		{"too long", string(make([]byte, maxContentLen+1)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkUtilErr(t, "ValidateContent()", ValidateContent(tt.content), tt.wantErr)
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		max    int
		expect string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 8, "hello..."},
		{"very short max", "hello", 3, "..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkUtilResult(t, "truncate", truncate(tt.input, tt.max), tt.expect)
		})
	}
}

func TestNormalizeTagColor(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"red", "red"},
		{"Red", "red"},
		{"grey", "gray"},
		{"custom", "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			checkUtilResult(t, "normalizeTagColor("+tt.input+")", normalizeTagColor(tt.input), tt.expect)
		})
	}
}

func TestCreateSnippet(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		query      string
		contextLen int
		expect     string
	}{
		{"match at start", "hello world test", "hello", 5, "hello worl..."},
		{"match in middle", "this is a test string", "test", 5, "...is a test stri..."},
		{"no match", "hello world", "xyz", 5, "hello w..."}, // truncate uses contextLen*2
		{"case insensitive", "Hello World", "world", 5, "...ello World"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkUtilResult(t, "createSnippet", createSnippet(tt.content, tt.query, tt.contextLen), tt.expect)
		})
	}
}

func TestSplitPath_AdditionalCases(t *testing.T) {
	tests := []struct {
		path string
		want []string
	}{
		{"", nil},
		{"/", nil},
		{"/boards", []string{"boards"}},
		{"boards/abc123", []string{"boards", "abc123"}}, // without leading slash
		{"/a/b/c", []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			checkStringSlice(t, "splitPath("+tt.path+")", splitPath(tt.path), tt.want)
		})
	}
}

func TestIndexOf_AdditionalCases(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   int
	}{
		{"hello world", "world", 6},
		{"hello world", "hello", 0},
		{"hello world", "x", -1},
		{"", "x", -1},
		{"abc", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.s+"/"+tt.substr, func(t *testing.T) {
			checkUtilResult(t, "indexOf", indexOf(tt.s, tt.substr), tt.want)
		})
	}
}

func TestJoinPath_AdditionalCases(t *testing.T) {
	tests := []struct {
		parts []string
		want  string
	}{
		{nil, ""},
		{[]string{}, ""},
		{[]string{"boards"}, "boards"},
		{[]string{"a", "b", "c", "d"}, "a/b/c/d"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			checkUtilResult(t, "joinPath", joinPath(tt.parts), tt.want)
		})
	}
}

func BenchmarkValidateBoardID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ValidateBoardID("uXjVN1234567890")
	}
}

func BenchmarkTruncate(b *testing.B) {
	content := "This is a test string that needs to be truncated to a shorter length"
	for i := 0; i < b.N; i++ {
		truncate(content, 30)
	}
}

func BenchmarkNormalizeStickyColor(b *testing.B) {
	for i := 0; i < b.N; i++ {
		normalizeStickyColor("yellow")
	}
}

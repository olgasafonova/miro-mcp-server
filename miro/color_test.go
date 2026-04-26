package miro

import (
	"strings"
	"testing"
)

func TestColorNameToHex(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantHex string
		wantOK  bool
	}{
		{name: "lowercase green", input: "green", wantHex: "#008000", wantOK: true},
		{name: "uppercase BLUE", input: "BLUE", wantHex: "#0000FF", wantOK: true},
		{name: "title-case Pink", input: "Pink", wantHex: "#FFC0CB", wantOK: true},
		{name: "whitespace around name", input: "  red  ", wantHex: "#FF0000", wantOK: true},
		{name: "grey alias maps to gray", input: "grey", wantHex: "#808080", wantOK: true},
		{name: "violet alias maps to purple", input: "violet", wantHex: "#800080", wantOK: true},
		{name: "empty string is not a name", input: "", wantHex: "", wantOK: false},
		{name: "unknown name", input: "chartreuse", wantHex: "", wantOK: false},
		{name: "hex value is not a name", input: "#FF0000", wantHex: "", wantOK: false},
		{name: "miro sticky-only name not in map", input: "light_yellow", wantHex: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHex, gotOK := colorNameToHex(tt.input)
			if gotHex != tt.wantHex || gotOK != tt.wantOK {
				t.Errorf("colorNameToHex(%q) = (%q, %v), want (%q, %v)", tt.input, gotHex, gotOK, tt.wantHex, tt.wantOK)
			}
		})
	}
}

func TestNormalizeColor(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "empty passes through as empty", input: "", want: "", wantErr: false},
		{name: "uppercase hex passes through", input: "#FF5733", want: "#FF5733", wantErr: false},
		{name: "lowercase hex passes through", input: "#ff5733", want: "#ff5733", wantErr: false},
		{name: "hex with whitespace is trimmed", input: "  #006400  ", want: "#006400", wantErr: false},
		{name: "color name lowercased", input: "green", want: "#008000", wantErr: false},
		{name: "color name uppercase", input: "GREEN", want: "#008000", wantErr: false},
		{name: "color name with whitespace", input: "  Pink  ", want: "#FFC0CB", wantErr: false},
		{name: "grey alias works", input: "grey", want: "#808080", wantErr: false},
		{name: "3-char hex is rejected", input: "#F00", want: "", wantErr: true},
		{name: "8-char hex with alpha is rejected", input: "#FF0000FF", want: "", wantErr: true},
		{name: "missing hash prefix is rejected", input: "FF0000", want: "", wantErr: true},
		{name: "unknown name is rejected", input: "chartreuse", want: "", wantErr: true},
		{name: "rgb function is rejected", input: "rgb(255, 0, 0)", want: "", wantErr: true},
		{name: "garbage is rejected", input: "not a color", want: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeColor(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeColor(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("normalizeColor(%q) = %q, want %q", tt.input, got, tt.want)
			}
			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), "unrecognized color") {
					t.Errorf("error message %q should mention 'unrecognized color' for caller diagnostics", err.Error())
				}
				if !strings.Contains(err.Error(), tt.input) {
					t.Errorf("error message %q should echo the offending input %q", err.Error(), tt.input)
				}
			}
		})
	}
}

func TestSupportedColorNamesDescription_listsAllMappedNames(t *testing.T) {
	for canonical := range commonColorNames {
		if canonical == "grey" || canonical == "violet" {
			continue // aliases for gray/purple, not listed separately
		}
		if !strings.Contains(SupportedColorNamesDescription, canonical) {
			t.Errorf("SupportedColorNamesDescription is missing %q; LLM tool descriptions reference this constant", canonical)
		}
	}
}

package miro

import (
	"context"
	"strings"
	"testing"
)

// TestValidatePaths_RejectInjectionPayloads is a regression suite for the
// path-injection finding (Carlini scan, miro Finding 2). Each payload class
// the URL parser silently rewrites or that pivots the destination must be
// rejected at the validator layer BEFORE the request reaches Miro.
func TestValidatePaths_RejectInjectionPayloads(t *testing.T) {
	cases := []struct {
		name string
		id   string
	}{
		{"query injection", "valid?team_id=victim"},
		{"path traversal", "../../etc/passwd"},
		{"slash extension", "valid/items/extra"},
		{"hash anchor", "valid#anchor"},
		{"semicolon param", "valid;param"},
		{"ampersand", "valid&query=injected"},
		{"percent-encoded slash", "valid%2F../something"},
		{"newline injection", "valid\nX-Inject: 1"},
		{"null byte", "valid\x00.attacker.com"},
		{"empty", ""},
		{"only whitespace", "   "},
		{"oversize", strings.Repeat("a", 101)},
	}

	for _, c := range cases {
		t.Run("BoardID/"+c.name, func(t *testing.T) {
			if err := ValidateBoardID(c.id); err == nil {
				t.Errorf("ValidateBoardID(%q) returned nil; expected rejection", c.id)
			}
		})
		t.Run("ItemID/"+c.name, func(t *testing.T) {
			if err := ValidateItemID(c.id); err == nil {
				t.Errorf("ValidateItemID(%q) returned nil; expected rejection", c.id)
			}
		})
		t.Run("OrgID/"+c.name, func(t *testing.T) {
			if err := ValidateOrgID(c.id); err == nil {
				t.Errorf("ValidateOrgID(%q) returned nil; expected rejection", c.id)
			}
		})
	}
}

// TestValidatePaths_AcceptRealMiroIDs verifies the validator does not regress
// legitimate Miro IDs. Real IDs observed in production use letters, digits,
// underscore, hyphen, and `=` (Base64 padding).
func TestValidatePaths_AcceptRealMiroIDs(t *testing.T) {
	realIDs := []string{
		"uXjVPzd1234=",
		"uXjVO_abc-DEF",
		"o9J_kxyz123=",
		"3458764500000",
		"abc",
	}

	for _, id := range realIDs {
		if err := ValidateBoardID(id); err != nil {
			t.Errorf("ValidateBoardID(%q) returned %v; expected nil for real Miro ID", id, err)
		}
		if err := ValidateItemID(id); err != nil {
			t.Errorf("ValidateItemID(%q) returned %v; expected nil for real Miro ID", id, err)
		}
		if err := ValidateOrgID(id); err != nil {
			t.Errorf("ValidateOrgID(%q) returned %v; expected nil for real Miro ID", id, err)
		}
	}
}

// TestClient_RejectsInjectedBoardID exercises the integrated path through one
// representative client method and asserts that the request never goes out.
// Uses a real Client with no transport — the validator must reject before the
// HTTP layer is touched.
func TestClient_RejectsInjectedBoardID(t *testing.T) {
	c := &Client{}
	_, err := c.ListItems(context.Background(), ListItemsArgs{BoardID: "valid?team_id=victim"})
	if err == nil {
		t.Fatal("ListItems accepted query-injected board_id; expected validator rejection")
	}
	if !strings.Contains(err.Error(), "board_id") {
		t.Errorf("error did not mention board_id: %v", err)
	}
}

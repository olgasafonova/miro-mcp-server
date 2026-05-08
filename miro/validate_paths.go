package miro

import (
	"fmt"
	"regexp"
)

// =============================================================================
// Input Validation
// =============================================================================

var (
	// validIDPattern matches safe Miro IDs (alphanumeric, underscore, hyphen, equals)
	validIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_=\-]+$`)

	// maxContentLen is the maximum allowed content length
	maxContentLen = 10000

	// maxIDLen is the maximum allowed ID length
	maxIDLen = 100
)

// ValidateBoardID ensures board ID is safe and well-formed.
func ValidateBoardID(id string) error {
	if id == "" {
		return fmt.Errorf("board_id is required")
	}
	if len(id) > maxIDLen {
		return fmt.Errorf("board_id too long (max %d characters)", maxIDLen)
	}
	if !validIDPattern.MatchString(id) {
		return fmt.Errorf("board_id contains invalid characters")
	}
	return nil
}

// ValidateItemID ensures item ID is safe and well-formed.
func ValidateItemID(id string) error {
	if id == "" {
		return fmt.Errorf("item_id is required")
	}
	if len(id) > maxIDLen {
		return fmt.Errorf("item_id too long (max %d characters)", maxIDLen)
	}
	if !validIDPattern.MatchString(id) {
		return fmt.Errorf("item_id contains invalid characters")
	}
	return nil
}

// ValidateOrgID ensures an organization ID is safe and well-formed.
// Used by export tools that hit /orgs/{org_id}/... endpoints. Without this
// validation, a prompt-injected agent could pivot the URL to other Miro
// API paths via path-segment injection.
func ValidateOrgID(id string) error {
	if id == "" {
		return fmt.Errorf("org_id is required")
	}
	if len(id) > maxIDLen {
		return fmt.Errorf("org_id too long (max %d characters)", maxIDLen)
	}
	if !validIDPattern.MatchString(id) {
		return fmt.Errorf("org_id contains invalid characters")
	}
	return nil
}

// ValidateContent ensures content is within allowed limits.
func ValidateContent(content string) error {
	if len(content) > maxContentLen {
		return fmt.Errorf("content exceeds maximum length of %d characters", maxContentLen)
	}
	return nil
}

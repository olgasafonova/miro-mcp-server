// Package miro provides a client for the Miro REST API.
package miro

import "time"

// =============================================================================
// Core Types
// =============================================================================

// Board represents a Miro board.
type Board struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt,omitempty"`
	ModifiedAt  time.Time `json:"modifiedAt,omitempty"`
	ViewLink    string    `json:"viewLink,omitempty"`
	Picture     *Picture  `json:"picture,omitempty"`
	Owner       *User     `json:"owner,omitempty"`
	Team        *Team     `json:"team,omitempty"`
}

// User represents a Miro user.
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Team represents a Miro team.
type Team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// =============================================================================
// Item Base Types
// =============================================================================

// Position defines x,y coordinates on the board.
type Position struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Origin string  `json:"origin,omitempty"` // "center" (default)
}

// Geometry defines width and height of an item.
type Geometry struct {
	Width  float64 `json:"width,omitempty"`
	Height float64 `json:"height,omitempty"`
}

// ItemBase contains common fields for all board items.
type ItemBase struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Position   *Position `json:"position,omitempty"`
	Geometry   *Geometry `json:"geometry,omitempty"`
	CreatedAt  time.Time `json:"createdAt,omitempty"`
	ModifiedAt time.Time `json:"modifiedAt,omitempty"`
	CreatedBy  *User     `json:"createdBy,omitempty"`
	ModifiedBy *User     `json:"modifiedBy,omitempty"`
	ParentID   string    `json:"parentId,omitempty"` // Frame or group ID
}

// =============================================================================
// API Response Types
// =============================================================================

// PaginatedResponse wraps paginated API responses.
type PaginatedResponse struct {
	Data   []interface{} `json:"data"`
	Total  int           `json:"total,omitempty"`
	Size   int           `json:"size,omitempty"`
	Offset string        `json:"offset,omitempty"`
	Limit  int           `json:"limit,omitempty"`
	Cursor string        `json:"cursor,omitempty"`
}

package miro

import (
	"fmt"
	"strconv"
)

// =============================================================================
// Offset Pagination
// =============================================================================
//
// The boards, members, comments and items-by-tag endpoints page by offset, and
// every one of them echoes back the offset of the page it just served rather
// than the offset of the next one. Verified live against api.miro.com on
// 18-09-2026: /boards/{id}/members?offset=1 answers offset=1, and
// /boards/{id}/items?tag_id=...&offset=1 answers offset=1. The echoed value is
// therefore useless as a cursor, and deriving the next one from the offset we
// requested plus the rows we received is the only source immune to that
// semantics.
//
// ListBoards carries its own copies of these helpers (boardsHaveMore,
// nextBoardOffset) because it was fixed first. They are the same two functions;
// fold them into these once both branches have landed.

// parseOffsetArg converts a caller-supplied offset string to an index. An empty
// string means the first page. Anything non-numeric or negative is rejected
// rather than silently treated as zero, which would quietly restart the walk.
func parseOffsetArg(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("invalid offset %q: must be a non-negative integer", raw)
	}
	return parsed, nil
}

// offsetHasMore reports whether another page exists, given the index the next
// page would start at. Total is authoritative when present; Miro's OpenAPI spec
// does not mark it required, so an absent total falls back to the full-page
// heuristic rather than reporting a false end.
func offsetHasMore(nextIndex, total, got, limit int) bool {
	if total > 0 {
		return nextIndex < total
	}
	return got >= limit
}

// nextOffset returns the index the following page starts at, or 0 when the
// collection is exhausted. Zero is unambiguous as "no next page": a next index
// is only ever emitted when at least one row came back, so it is always
// positive when it means anything.
func nextOffset(nextIndex, total, got, limit int) int {
	if !offsetHasMore(nextIndex, total, got, limit) {
		return 0
	}
	return nextIndex
}

// nextOffsetString renders nextOffset for the endpoints whose offset argument
// is a string, with "" for the end of the collection.
func nextOffsetString(nextIndex, total, got, limit int) string {
	if !offsetHasMore(nextIndex, total, got, limit) {
		return ""
	}
	return strconv.Itoa(nextIndex)
}

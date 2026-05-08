package miro

// =============================================================================
// Deep Link Generation
// =============================================================================

// MiroAppBaseURL is the base URL for Miro web application deep links.
const MiroAppBaseURL = "https://miro.com/app/board"

// BuildItemURL generates a deep link URL to focus on a specific item in a Miro board.
// The returned URL opens the board and centers the view on the specified item.
// Format: https://miro.com/app/board/{boardId}/?moveToWidget={itemId}
func BuildItemURL(boardID, itemID string) string {
	if boardID == "" || itemID == "" {
		return ""
	}
	return MiroAppBaseURL + "/" + boardID + "/?moveToWidget=" + itemID
}

// BuildBoardURL generates a deep link URL to a Miro board.
// Format: https://miro.com/app/board/{boardId}/
func BuildBoardURL(boardID string) string {
	if boardID == "" {
		return ""
	}
	return MiroAppBaseURL + "/" + boardID + "/"
}

// BuildItemURLs generates deep link URLs for multiple items on the same board.
func BuildItemURLs(boardID string, itemIDs []string) []string {
	if boardID == "" || len(itemIDs) == 0 {
		return nil
	}
	urls := make([]string, len(itemIDs))
	for i, id := range itemIDs {
		urls[i] = BuildItemURL(boardID, id)
	}
	return urls
}

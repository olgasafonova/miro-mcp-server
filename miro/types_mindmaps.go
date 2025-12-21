package miro

// =============================================================================
// Mindmap Types
// =============================================================================

// MindmapNode represents a node in a mindmap.
type MindmapNode struct {
	ItemBase
	Data MindmapNodeData `json:"data"`
}

// MindmapNodeData contains mindmap node specific data.
type MindmapNodeData struct {
	NodeView string `json:"nodeView,omitempty"` // "text" or "bubble"
}

// =============================================================================
// Create Mindmap Node
// =============================================================================

// CreateMindmapNodeArgs contains parameters for creating a mindmap node.
type CreateMindmapNodeArgs struct {
	BoardID  string  `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	Content  string  `json:"content" jsonschema:"required" jsonschema_description:"Text content of the node"`
	ParentID string  `json:"parent_id,omitempty" jsonschema_description:"ID of the parent node (omit for root node)"`
	NodeView string  `json:"node_view,omitempty" jsonschema_description:"Node style: text (default) or bubble"`
	X        float64 `json:"x,omitempty" jsonschema_description:"X position (only for root nodes)"`
	Y        float64 `json:"y,omitempty" jsonschema_description:"Y position (only for root nodes)"`
}

// CreateMindmapNodeResult contains the created mindmap node.
type CreateMindmapNodeResult struct {
	ID       string `json:"id"`
	Content  string `json:"content"`
	ParentID string `json:"parent_id,omitempty"`
	Message  string `json:"message"`
}

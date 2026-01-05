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
	ItemURL  string `json:"item_url,omitempty"`
	Content  string `json:"content"`
	ParentID string `json:"parent_id,omitempty"`
	Message  string `json:"message"`
}

// =============================================================================
// Get Mindmap Node
// =============================================================================

// GetMindmapNodeArgs contains parameters for getting a mindmap node.
type GetMindmapNodeArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	NodeID  string `json:"node_id" jsonschema:"required" jsonschema_description:"Mindmap node ID to retrieve"`
}

// GetMindmapNodeResult contains the mindmap node details.
type GetMindmapNodeResult struct {
	ID         string   `json:"id"`
	Content    string   `json:"content"`
	NodeView   string   `json:"node_view,omitempty"`
	IsRoot     bool     `json:"is_root"`
	ParentID   string   `json:"parent_id,omitempty"`
	ChildIDs   []string `json:"child_ids,omitempty"`
	X          float64  `json:"x"`
	Y          float64  `json:"y"`
	CreatedAt  string   `json:"created_at,omitempty"`
	ModifiedAt string   `json:"modified_at,omitempty"`
	Message    string   `json:"message"`
}

// =============================================================================
// List Mindmap Nodes
// =============================================================================

// ListMindmapNodesArgs contains parameters for listing mindmap nodes.
type ListMindmapNodesArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	Limit   int    `json:"limit,omitempty" jsonschema_description:"Max nodes to return (default 50, max 100)"`
	Cursor  string `json:"cursor,omitempty" jsonschema_description:"Pagination cursor"`
}

// MindmapNodeSummary is a brief summary of a mindmap node.
type MindmapNodeSummary struct {
	ID       string `json:"id"`
	Content  string `json:"content"`
	IsRoot   bool   `json:"is_root"`
	ParentID string `json:"parent_id,omitempty"`
}

// ListMindmapNodesResult contains the list of mindmap nodes.
type ListMindmapNodesResult struct {
	Nodes   []MindmapNodeSummary `json:"nodes"`
	Count   int                  `json:"count"`
	HasMore bool                 `json:"has_more"`
	Cursor  string               `json:"cursor,omitempty"`
	Message string               `json:"message"`
}

// =============================================================================
// Delete Mindmap Node
// =============================================================================

// DeleteMindmapNodeArgs contains parameters for deleting a mindmap node.
type DeleteMindmapNodeArgs struct {
	BoardID string `json:"board_id" jsonschema:"required" jsonschema_description:"Board ID"`
	NodeID  string `json:"node_id" jsonschema:"required" jsonschema_description:"Mindmap node ID to delete"`
	DryRun  bool   `json:"dry_run,omitempty" jsonschema_description:"If true, returns preview without deleting"`
}

// DeleteMindmapNodeResult confirms the deletion.
type DeleteMindmapNodeResult struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Message string `json:"message"`
}

// Package prompts provides MCP prompt templates for common Miro workflows.
// Prompts help LLMs perform complex multi-step operations with predefined templates.
package prompts

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Registry manages MCP prompt registration.
type Registry struct{}

// NewRegistry creates a new prompt registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// RegisterAll registers all Miro prompts with the MCP server.
func (r *Registry) RegisterAll(server *mcp.Server) {
	// Sprint Board prompt
	server.AddPrompt(&mcp.Prompt{
		Name:        "create-sprint-board",
		Title:       "Create Sprint Board",
		Description: "Create a sprint planning board with standard sections for backlog, in progress, review, and done.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "board_name",
				Description: "Name for the new sprint board",
				Required:    true,
			},
			{
				Name:        "sprint_number",
				Description: "Sprint number (e.g., '42')",
				Required:    false,
			},
		},
	}, r.handleSprintBoard)

	// Retrospective prompt
	server.AddPrompt(&mcp.Prompt{
		Name:        "create-retrospective",
		Title:       "Create Retrospective Board",
		Description: "Create a retrospective board with sections for what went well, what could improve, and action items.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "board_id",
				Description: "Board ID to add retrospective sections to (creates new if not provided)",
				Required:    false,
			},
			{
				Name:        "team_name",
				Description: "Team name for the retrospective",
				Required:    false,
			},
		},
	}, r.handleRetrospective)

	// Brainstorming prompt
	server.AddPrompt(&mcp.Prompt{
		Name:        "create-brainstorm",
		Title:       "Create Brainstorming Session",
		Description: "Set up a brainstorming board with a central topic and space for ideas.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "topic",
				Description: "Central topic or question for brainstorming",
				Required:    true,
			},
			{
				Name:        "board_id",
				Description: "Existing board ID (creates new if not provided)",
				Required:    false,
			},
		},
	}, r.handleBrainstorm)

	// User Story Map prompt
	server.AddPrompt(&mcp.Prompt{
		Name:        "create-story-map",
		Title:       "Create User Story Map",
		Description: "Create a user story mapping board with activities, user tasks, and story cards.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "product_name",
				Description: "Name of the product being mapped",
				Required:    true,
			},
			{
				Name:        "board_id",
				Description: "Existing board ID (creates new if not provided)",
				Required:    false,
			},
		},
	}, r.handleStoryMap)

	// Kanban Board prompt
	server.AddPrompt(&mcp.Prompt{
		Name:        "create-kanban",
		Title:       "Create Kanban Board",
		Description: "Create a kanban board with customizable columns for workflow management.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "board_name",
				Description: "Name for the kanban board",
				Required:    true,
			},
			{
				Name:        "columns",
				Description: "Comma-separated column names (default: To Do,In Progress,Review,Done)",
				Required:    false,
			},
		},
	}, r.handleKanban)
}

// handleSprintBoard generates a sprint board creation prompt
func (r *Registry) handleSprintBoard(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	boardName := req.Params.Arguments["board_name"]
	sprintNum := req.Params.Arguments["sprint_number"]
	if sprintNum == "" {
		sprintNum = "N"
	}

	prompt := fmt.Sprintf(`Create a sprint planning board for "%s" with the following structure:

1. First, create a new board named "%s" using miro_create_board
2. Create 4 frames arranged horizontally (800x600 each, 50px spacing):
   - "Backlog" (x: 0) - gray background
   - "In Progress" (x: 850) - blue background
   - "In Review" (x: 1700) - yellow background
   - "Done" (x: 2550) - green background

3. Add a title text "Sprint %s Planning" at the top (x: 1275, y: -100, font_size: 48)

4. Add starter sticky notes in each frame:
   - Backlog: 3 yellow stickies with placeholder tasks
   - In Progress: 1 blue sticky "Current work"
   - In Review: 1 pink sticky "Awaiting review"
   - Done: 1 green sticky "Completed items"

5. Return the board URL when complete.

Use the Miro MCP tools in sequence: miro_create_board -> miro_create_frame (x4) -> miro_create_text -> miro_create_sticky (multiple)`, boardName, boardName, sprintNum)

	return &mcp.GetPromptResult{
		Description: "Instructions to create a sprint planning board",
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: prompt},
			},
		},
	}, nil
}

// handleRetrospective generates a retrospective board prompt
func (r *Registry) handleRetrospective(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	boardID := req.Params.Arguments["board_id"]
	teamName := req.Params.Arguments["team_name"]
	if teamName == "" {
		teamName = "Team"
	}

	var prompt string
	if boardID != "" {
		prompt = fmt.Sprintf(`Add retrospective sections to the existing board "%s" for %s:

1. Create 3 frames arranged horizontally:
   - "What Went Well" (green background, x: 0)
   - "What Could Improve" (pink background, x: 850)
   - "Action Items" (blue background, x: 1700)

2. Add a title text "%s Retrospective" at the top

3. Add starter stickies in each frame:
   - "What Went Well": 2 green stickies with prompts like "Team collaboration was excellent"
   - "What Could Improve": 2 pink stickies with prompts like "Could improve code review turnaround"
   - "Action Items": 2 yellow stickies with prompts like "Schedule weekly sync meetings"

Use board_id: %s for all operations.`, boardID, teamName, teamName, boardID)
	} else {
		prompt = fmt.Sprintf(`Create a new retrospective board for %s:

1. Create a new board named "%s Retrospective" using miro_create_board
2. Create 3 frames arranged horizontally (800x600 each):
   - "What Went Well" (green background, x: 0)
   - "What Could Improve" (pink background, x: 850)
   - "Action Items" (blue background, x: 1700)

3. Add a title text "%s Retrospective" at the top (x: 850, y: -100, font_size: 48)

4. Add starter stickies in each frame:
   - "What Went Well": 2 green stickies
   - "What Could Improve": 2 pink stickies
   - "Action Items": 2 yellow stickies

5. Return the board URL when complete.`, teamName, teamName, teamName)
	}

	return &mcp.GetPromptResult{
		Description: "Instructions to create a retrospective board",
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: prompt},
			},
		},
	}, nil
}

// handleBrainstorm generates a brainstorming session prompt
func (r *Registry) handleBrainstorm(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	topic := req.Params.Arguments["topic"]
	boardID := req.Params.Arguments["board_id"]

	var prompt string
	if boardID != "" {
		prompt = fmt.Sprintf(`Set up a brainstorming session on board "%s" for the topic: "%s"

1. Create a central shape (circle or diamond) with the topic text at position (400, 300)
2. Create 6 empty sticky notes arranged in a radial pattern around the topic:
   - Use different colors: yellow, green, blue, pink, orange, cyan
   - Position them ~300px away from center in a circle

3. Add a text label "Ideas" near each sticky note area

Use board_id: %s for all operations.`, boardID, topic, boardID)
	} else {
		prompt = fmt.Sprintf(`Create a brainstorming board for the topic: "%s"

1. Create a new board named "Brainstorm: %s" using miro_create_board
2. Create a central diamond shape with the topic text at the center (x: 400, y: 300)
3. Create 6 sticky notes arranged radially around the center:
   - Yellow at (100, 100)
   - Green at (700, 100)
   - Blue at (100, 500)
   - Pink at (700, 500)
   - Orange at (400, 0)
   - Cyan at (400, 600)

4. Add connectors from center to each sticky note
5. Return the board URL when complete.`, topic, topic)
	}

	return &mcp.GetPromptResult{
		Description: "Instructions to set up a brainstorming session",
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: prompt},
			},
		},
	}, nil
}

// handleStoryMap generates a user story map prompt
func (r *Registry) handleStoryMap(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	productName := req.Params.Arguments["product_name"]
	boardID := req.Params.Arguments["board_id"]

	var prompt string
	if boardID != "" {
		prompt = fmt.Sprintf(`Create a user story map for "%s" on board "%s":

1. Create a header row with 3-4 "Activity" frames (blue background):
   - "Discovery" (x: 0, y: 0)
   - "Onboarding" (x: 450, y: 0)
   - "Core Usage" (x: 900, y: 0)
   - "Growth" (x: 1350, y: 0)

2. Below each activity, add "User Task" sticky notes (yellow):
   - 2-3 tasks per activity

3. Below tasks, create "Release" swimlanes:
   - "MVP" line with green stickies
   - "v1.0" line with blue stickies
   - "Future" line with gray stickies

Use board_id: %s for all operations.`, productName, boardID, boardID)
	} else {
		prompt = fmt.Sprintf(`Create a user story map board for "%s":

1. Create a new board named "%s Story Map" using miro_create_board
2. Create header frames for user activities (width: 400, height: 100, blue background):
   - "Discovery" at x: 0
   - "Onboarding" at x: 450
   - "Core Usage" at x: 900
   - "Growth" at x: 1350

3. Add user task stickies (yellow) below each activity at y: 150

4. Create release swimlanes with horizontal lines:
   - "MVP" label at y: 350 with green stickies
   - "v1.0" label at y: 550 with blue stickies
   - "Future" label at y: 750 with gray stickies

5. Return the board URL when complete.`, productName, productName)
	}

	return &mcp.GetPromptResult{
		Description: "Instructions to create a user story map",
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: prompt},
			},
		},
	}, nil
}

// handleKanban generates a kanban board prompt
func (r *Registry) handleKanban(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	boardName := req.Params.Arguments["board_name"]
	columnsArg := req.Params.Arguments["columns"]

	columns := []string{"To Do", "In Progress", "Review", "Done"}
	if columnsArg != "" {
		columns = splitColumns(columnsArg)
	}

	prompt := fmt.Sprintf(`Create a kanban board named "%s" with the following columns:

1. Create a new board named "%s" using miro_create_board

2. Create %d frames arranged horizontally (width: 400, height: 800, spacing: 50):`, boardName, boardName, len(columns))

	for i, col := range columns {
		color := getColumnColor(i)
		prompt += fmt.Sprintf(`
   - "%s" at x: %d (%s background)`, col, i*450, color)
	}

	prompt += `

3. Add column header text above each frame (font_size: 24)

4. Add a title text at the top with the board name (font_size: 36)

5. Add 2-3 sample sticky notes in the first column (To Do or equivalent)

6. Return the board URL when complete.`

	return &mcp.GetPromptResult{
		Description: "Instructions to create a kanban board",
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: prompt},
			},
		},
	}, nil
}

// splitColumns splits a comma-separated column string
func splitColumns(s string) []string {
	var result []string
	for _, col := range splitAndTrim(s, ",") {
		if col != "" {
			result = append(result, col)
		}
	}
	return result
}

// splitAndTrim splits a string and trims each part
func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	for _, p := range splitString(s, sep) {
		trimmed := trimSpace(p)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

// splitString is a simple string split
func splitString(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}

// trimSpace removes leading/trailing whitespace
func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

// getColumnColor returns a color for a kanban column based on index
func getColumnColor(index int) string {
	colors := []string{"gray", "blue", "yellow", "green", "pink", "orange", "cyan"}
	return colors[index%len(colors)]
}

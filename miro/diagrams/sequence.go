package diagrams

import (
	"fmt"
	"regexp"
	"strings"
)

// SequenceParser parses Mermaid sequence diagram syntax.
type SequenceParser struct {
	// Patterns for parsing
	headerPattern      *regexp.Regexp
	participantPattern *regexp.Regexp
	messagePattern     *regexp.Regexp
	notePattern        *regexp.Regexp
	activatePattern    *regexp.Regexp
	loopStartPattern   *regexp.Regexp
	loopEndPattern     *regexp.Regexp
	elsePattern *regexp.Regexp
}

// Participant represents a sequence diagram participant.
type Participant struct {
	ID    string
	Label string
	Type  string // "participant" or "actor"
	Order int    // Position left to right
}

// Message represents a message between participants.
type Message struct {
	From      string
	To        string
	Text      string
	Style     string // sync, async, reply, cross
	Activate  bool
	Deactivate bool
}

// NewSequenceParser creates a new sequence diagram parser.
func NewSequenceParser() *SequenceParser {
	return &SequenceParser{
		// Match: sequenceDiagram
		headerPattern: regexp.MustCompile(`(?i)^\s*sequenceDiagram\s*$`),

		// Match: participant A, actor A, participant A as "Alice"
		participantPattern: regexp.MustCompile(`(?i)^\s*(participant|actor)\s+(\S+)(?:\s+as\s+["']?(.+?)["']?)?\s*$`),

		// Match various message types:
		// A->>B: text (sync)
		// A-->>B: text (async/dotted)
		// A->>+B: text (activate B)
		// A->>-B: text (deactivate B)
		// A-xB: text (cross/lost)
		// A-)B: text (async open arrow)
		messagePattern: regexp.MustCompile(`^\s*(\S+?)\s*(->>|-->>|-\)|--\)|->|-->|-x|--x)(\+|-)?(\S+?)\s*:\s*(.*)$`),

		// Match notes: Note right of A: text, Note over A,B: text
		notePattern: regexp.MustCompile(`(?i)^\s*Note\s+(right of|left of|over)\s+(\S+?)(?:,\s*(\S+?))?\s*:\s*(.+)$`),

		// Match activate/deactivate
		activatePattern: regexp.MustCompile(`(?i)^\s*(activate|deactivate)\s+(\S+)\s*$`),

		// Match loop/rect/opt/alt/par start
		loopStartPattern: regexp.MustCompile(`(?i)^\s*(loop|rect|opt|alt|par|critical)\s*(.*)$`),

		// Match end or else
		loopEndPattern: regexp.MustCompile(`(?i)^\s*end\s*$`),
		elsePattern:    regexp.MustCompile(`(?i)^\s*else\s*(.*)$`),
	}
}

// Parse parses a Mermaid sequence diagram.
func (p *SequenceParser) Parse(input string) (*Diagram, error) {
	diagram := NewDiagram(TypeSequence)
	diagram.Direction = LeftToRight // Sequence diagrams are horizontal

	lines := strings.Split(input, "\n")
	participants := make(map[string]*Participant)
	participantOrder := 0
	var messages []Message
	groupStack := []string{}
	messageIndex := 0

	foundHeader := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}

		// Check for header
		if p.headerPattern.MatchString(line) {
			foundHeader = true
			continue
		}

		// Check for participant declaration
		if matches := p.participantPattern.FindStringSubmatch(line); matches != nil {
			pType := strings.ToLower(matches[1])
			pID := matches[2]
			pLabel := pID
			if len(matches) > 3 && matches[3] != "" {
				pLabel = matches[3]
			}

			if _, exists := participants[pID]; !exists {
				participants[pID] = &Participant{
					ID:    pID,
					Label: pLabel,
					Type:  pType,
					Order: participantOrder,
				}
				participantOrder++
			}
			continue
		}

		// Check for message
		if matches := p.messagePattern.FindStringSubmatch(line); matches != nil {
			from := matches[1]
			arrow := matches[2]
			modifier := matches[3]
			to := matches[4]
			text := strings.TrimSpace(matches[5])

			// Auto-create participants if not declared
			if _, exists := participants[from]; !exists {
				participants[from] = &Participant{
					ID:    from,
					Label: from,
					Type:  "participant",
					Order: participantOrder,
				}
				participantOrder++
			}
			if _, exists := participants[to]; !exists {
				participants[to] = &Participant{
					ID:    to,
					Label: to,
					Type:  "participant",
					Order: participantOrder,
				}
				participantOrder++
			}

			msg := Message{
				From: from,
				To:   to,
				Text: text,
			}

			// Determine message style
			switch arrow {
			case "->>", "->":
				msg.Style = "sync"
			case "-->>", "-->":
				msg.Style = "async"
			case "-)", "--)":
				msg.Style = "async_open"
			case "-x", "--x":
				msg.Style = "cross"
			}

			// Handle activation modifiers
			switch modifier {
			case "+":
				msg.Activate = true
			case "-":
				msg.Deactivate = true
			}

			messages = append(messages, msg)
			messageIndex++
			continue
		}

		// Check for notes
		if matches := p.notePattern.FindStringSubmatch(line); matches != nil {
			// Notes are converted to text shapes
			// For now, we'll skip notes as they require special layout
			continue
		}

		// Check for activate/deactivate
		if matches := p.activatePattern.FindStringSubmatch(line); matches != nil {
			// Activation bars are a visual enhancement - skip for basic implementation
			continue
		}

		// Check for loop/rect/alt start
		if matches := p.loopStartPattern.FindStringSubmatch(line); matches != nil {
			groupType := strings.ToLower(matches[1])
			groupStack = append(groupStack, groupType)
			continue
		}

		// Check for else
		if p.elsePattern.MatchString(line) {
			continue
		}

		// Check for end
		if p.loopEndPattern.MatchString(line) {
			if len(groupStack) > 0 {
				groupStack = groupStack[:len(groupStack)-1]
			}
			continue
		}
	}

	if !foundHeader {
		return nil, fmt.Errorf("not a sequence diagram: missing 'sequenceDiagram' header")
	}

	if len(participants) == 0 {
		return nil, fmt.Errorf("no participants found in sequence diagram")
	}

	// Layout constants
	const (
		participantWidth   = 120.0
		participantHeight  = 50.0
		participantSpacing = 180.0
		messageSpacing     = 60.0
		startY             = 50.0
	)

	// Create nodes for participants (positioned horizontally)
	for _, p := range participants {
		x := float64(p.Order) * participantSpacing

		node := &Node{
			ID:     p.ID,
			Label:  p.Label,
			Shape:  ShapeRectangle,
			X:      x,
			Y:      startY,
			Width:  participantWidth,
			Height: participantHeight,
		}

		if p.Type == "actor" {
			node.Shape = ShapeCircle
			node.Width = 50
			node.Height = 50
		}

		diagram.AddNode(node)
	}

	// Create edges for messages with Y positions
	for i, msg := range messages {
		y := startY + participantHeight + 30 + float64(i)*messageSpacing

		edge := &Edge{
			ID:     fmt.Sprintf("msg_%d", i),
			FromID: msg.From,
			ToID:   msg.To,
			Label:  msg.Text,
			Y:      y, // Store Y position for sequence diagram rendering
		}

		// Set arrow styles based on message type
		switch msg.Style {
		case "sync":
			edge.Style = EdgeSolid
			edge.StartCap = ArrowNone
			edge.EndCap = ArrowNormal
		case "async":
			edge.Style = EdgeDotted
			edge.StartCap = ArrowNone
			edge.EndCap = ArrowNormal
		case "async_open":
			edge.Style = EdgeDotted
			edge.StartCap = ArrowNone
			edge.EndCap = ArrowNormal
		case "cross":
			edge.Style = EdgeSolid
			edge.StartCap = ArrowNone
			edge.EndCap = ArrowCross
		default:
			edge.Style = EdgeSolid
			edge.EndCap = ArrowNormal
		}

		diagram.AddEdge(edge)
	}

	// Calculate diagram dimensions
	if len(participants) > 0 {
		diagram.Width = float64(len(participants)-1)*participantSpacing + participantWidth
	}
	if len(messages) > 0 {
		lastMessageY := startY + participantHeight + 30 + float64(len(messages)-1)*messageSpacing
		diagram.Height = lastMessageY + messageSpacing // Add padding at bottom
	} else {
		diagram.Height = startY + participantHeight + 50
	}

	return diagram, nil
}

// ParseSequence is a convenience function to parse sequence diagram syntax.
func ParseSequence(input string) (*Diagram, error) {
	parser := NewSequenceParser()
	return parser.Parse(input)
}

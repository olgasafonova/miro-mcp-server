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
	elsePattern        *regexp.Regexp
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
	From       string
	To         string
	Text       string
	Style      string // sync, async, reply, cross
	Activate   bool
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

// sequenceLayout collects the spatial constants used to position participants
// and messages on the canvas.
const (
	participantWidth   = 120.0
	participantHeight  = 50.0
	participantSpacing = 180.0
	messageSpacing     = 60.0
	startY             = 50.0
	actorSize          = 50.0
)

// parseState carries the running state of a sequence-diagram parse.
type parseState struct {
	participants     map[string]*Participant
	participantOrder int
	messages         []Message
	groupStack       []string
	foundHeader      bool
}

// newParseState returns an empty parse state.
func newParseState() *parseState {
	return &parseState{participants: make(map[string]*Participant)}
}

// ensureParticipant adds a participant with the given id and label if absent.
// Returns false when the id was already known.
func (s *parseState) ensureParticipant(id, label, pType string) {
	if _, exists := s.participants[id]; exists {
		return
	}
	s.participants[id] = &Participant{
		ID:    id,
		Label: label,
		Type:  pType,
		Order: s.participantOrder,
	}
	s.participantOrder++
}

// Parse parses a Mermaid sequence diagram.
func (p *SequenceParser) Parse(input string) (*Diagram, error) {
	state := p.parseLines(strings.Split(input, "\n"))

	if !state.foundHeader {
		return nil, fmt.Errorf("not a sequence diagram: missing 'sequenceDiagram' header")
	}
	if len(state.participants) == 0 {
		return nil, fmt.Errorf("no participants found in sequence diagram")
	}

	diagram := NewDiagram(TypeSequence)
	diagram.Direction = LeftToRight // Sequence diagrams are horizontal

	addParticipantNodes(diagram, state.participants)
	addMessageEdges(diagram, state.messages)
	setSequenceDimensions(diagram, len(state.participants), len(state.messages))

	return diagram, nil
}

// parseLines walks the input lines and returns the accumulated parse state.
func (p *SequenceParser) parseLines(lines []string) *parseState {
	state := newParseState()
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if isIgnorableLine(line) {
			continue
		}
		p.dispatchLine(line, state)
	}
	return state
}

// isIgnorableLine returns true for empty lines and Mermaid comments.
func isIgnorableLine(line string) bool {
	return line == "" || strings.HasPrefix(line, "%%")
}

// tryParseHeader marks the header as found when line matches the header
// pattern.
func (p *SequenceParser) tryParseHeader(line string, state *parseState) bool {
	if !p.headerPattern.MatchString(line) {
		return false
	}
	state.foundHeader = true
	return true
}

// tryParseLoopStart pushes a new group onto the stack when line opens a loop.
func (p *SequenceParser) tryParseLoopStart(line string, state *parseState) bool {
	matches := p.loopStartPattern.FindStringSubmatch(line)
	if matches == nil {
		return false
	}
	state.groupStack = append(state.groupStack, strings.ToLower(matches[1]))
	return true
}

// tryParseLoopEnd pops a group off the stack when line closes a loop.
func (p *SequenceParser) tryParseLoopEnd(line string, state *parseState) bool {
	if !p.loopEndPattern.MatchString(line) {
		return false
	}
	if len(state.groupStack) > 0 {
		state.groupStack = state.groupStack[:len(state.groupStack)-1]
	}
	return true
}

// dispatchLine matches line against each known pattern in priority order and
// updates state accordingly. Lines that match nothing are silently dropped
// (matches the Mermaid-permissive behavior the original Parse had).
func (p *SequenceParser) dispatchLine(line string, state *parseState) {
	handlers := []func(string, *parseState) bool{
		p.tryParseHeader,
		p.tryParseParticipant,
		p.tryParseMessage,
		// Notes and activation bars are visual enhancements; skipped.
		func(l string, _ *parseState) bool { return p.notePattern.MatchString(l) },
		func(l string, _ *parseState) bool { return p.activatePattern.MatchString(l) },
		p.tryParseLoopStart,
		func(l string, _ *parseState) bool { return p.elsePattern.MatchString(l) },
		p.tryParseLoopEnd,
	}
	for _, h := range handlers {
		if h(line, state) {
			return
		}
	}
}

// tryParseParticipant attempts to match a "participant X" / "actor X as Y"
// declaration. Returns true when matched.
func (p *SequenceParser) tryParseParticipant(line string, state *parseState) bool {
	matches := p.participantPattern.FindStringSubmatch(line)
	if matches == nil {
		return false
	}
	pType := strings.ToLower(matches[1])
	pID := matches[2]
	pLabel := pID
	if len(matches) > 3 && matches[3] != "" {
		pLabel = matches[3]
	}
	state.ensureParticipant(pID, pLabel, pType)
	return true
}

// tryParseMessage attempts to match a "A->>B: text" message line. Auto-creates
// participants that haven't been declared yet. Returns true when matched.
func (p *SequenceParser) tryParseMessage(line string, state *parseState) bool {
	matches := p.messagePattern.FindStringSubmatch(line)
	if matches == nil {
		return false
	}
	from, arrow, modifier, to := matches[1], matches[2], matches[3], matches[4]
	text := strings.TrimSpace(matches[5])

	state.ensureParticipant(from, from, "participant")
	state.ensureParticipant(to, to, "participant")

	msg := Message{
		From:  from,
		To:    to,
		Text:  text,
		Style: messageStyleForArrow(arrow),
	}
	applyMessageModifier(&msg, modifier)
	state.messages = append(state.messages, msg)
	return true
}

// messageStyleForArrow maps a Mermaid arrow token to the style key used by
// the renderer.
func messageStyleForArrow(arrow string) string {
	switch arrow {
	case "->>", "->":
		return "sync"
	case "-->>", "-->":
		return "async"
	case "-)", "--)":
		return "async_open"
	case "-x", "--x":
		return "cross"
	}
	return ""
}

// applyMessageModifier sets Activate/Deactivate based on the +/- modifier.
func applyMessageModifier(msg *Message, modifier string) {
	switch modifier {
	case "+":
		msg.Activate = true
	case "-":
		msg.Deactivate = true
	}
}

// addParticipantNodes adds one node per participant, positioned horizontally
// by participant order.
func addParticipantNodes(diagram *Diagram, participants map[string]*Participant) {
	for _, p := range participants {
		diagram.AddNode(buildParticipantNode(p))
	}
}

// buildParticipantNode produces the node for a single participant, applying
// the actor-specific shape/size where appropriate.
func buildParticipantNode(p *Participant) *Node {
	node := &Node{
		ID:     p.ID,
		Label:  p.Label,
		Shape:  ShapeRectangle,
		X:      float64(p.Order) * participantSpacing,
		Y:      startY,
		Width:  participantWidth,
		Height: participantHeight,
	}
	if p.Type == "actor" {
		node.Shape = ShapeCircle
		node.Width = actorSize
		node.Height = actorSize
	}
	return node
}

// addMessageEdges adds one edge per message, with Y positions stacked from
// startY downward.
func addMessageEdges(diagram *Diagram, messages []Message) {
	for i, msg := range messages {
		diagram.AddEdge(buildMessageEdge(i, msg))
	}
}

// buildMessageEdge produces the edge for a single message, including style
// and arrow caps derived from the message type.
func buildMessageEdge(index int, msg Message) *Edge {
	edge := &Edge{
		ID:     fmt.Sprintf("msg_%d", index),
		FromID: msg.From,
		ToID:   msg.To,
		Label:  msg.Text,
		Y:      messageYPosition(index),
	}
	applyArrowStyleForMessage(edge, msg.Style)
	return edge
}

// messageYPosition is the Y coordinate for the i-th message line.
func messageYPosition(i int) float64 {
	return startY + participantHeight + 30 + float64(i)*messageSpacing
}

// applyArrowStyleForMessage sets edge.Style and edge.{Start,End}Cap based on
// the message's style key.
func applyArrowStyleForMessage(edge *Edge, style string) {
	switch style {
	case "sync":
		edge.Style = EdgeSolid
		edge.StartCap = ArrowNone
		edge.EndCap = ArrowNormal
	case "async", "async_open":
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
}

// setSequenceDimensions populates diagram.Width and diagram.Height from the
// participant and message counts.
func setSequenceDimensions(diagram *Diagram, participantCount, messageCount int) {
	if participantCount > 0 {
		diagram.Width = float64(participantCount-1)*participantSpacing + participantWidth
	}
	if messageCount > 0 {
		diagram.Height = messageYPosition(messageCount-1) + messageSpacing
		return
	}
	diagram.Height = startY + participantHeight + 50
}

// ParseSequence is a convenience function to parse sequence diagram syntax.
func ParseSequence(input string) (*Diagram, error) {
	parser := NewSequenceParser()
	return parser.Parse(input)
}

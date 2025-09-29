package layout

import (
	"edd/diagram"
)

// SequenceLayout implements a layout engine for UML sequence diagrams
type SequenceLayout struct {
	// Configuration
	ParticipantSpacing int // Horizontal spacing between participants
	MessageSpacing     int // Vertical spacing between messages
	ParticipantWidth   int // Default width for participant boxes
	ParticipantHeight  int // Default height for participant boxes
	TopMargin         int // Space above participants
	LeftMargin        int // Space before first participant
}

// SequencePositions holds computed positions for sequence diagram elements
type SequencePositions struct {
	Participants map[int]ParticipantPosition // Node ID -> position
	Messages     []MessagePosition           // Message positions in order
}

// ParticipantPosition holds the computed position of a participant
type ParticipantPosition struct {
	X         int
	Y         int
	Width     int
	Height    int
	LifelineX int // X position of the lifeline (center of participant)
}

// MessagePosition holds the computed position of a message
type MessagePosition struct {
	FromX int
	ToX   int
	Y     int
	Label string
	ConnectionID int // Reference to the original connection for hints
}

// NewSequenceLayout creates a new sequence diagram layout engine
func NewSequenceLayout() *SequenceLayout {
	return &SequenceLayout{
		ParticipantSpacing: 15,
		MessageSpacing:     2,  // Minimum: 1 line for arrow, 1 line for label
		ParticipantWidth:   20,
		ParticipantHeight:  3,
		TopMargin:         2,
		LeftMargin:        5,
	}
}

// ComputePositions calculates positions for all elements without modifying the diagram
func (s *SequenceLayout) ComputePositions(d *diagram.Diagram) *SequencePositions {
	if d == nil || len(d.Nodes) == 0 {
		return &SequencePositions{
			Participants: make(map[int]ParticipantPosition),
			Messages:     []MessagePosition{},
		}
	}
	
	positions := &SequencePositions{
		Participants: make(map[int]ParticipantPosition),
		Messages:     make([]MessagePosition, 0, len(d.Connections)),
	}
	
	// Identify participants
	participantNodes := s.identifyParticipants(d)
	
	// Compute participant positions
	x := s.LeftMargin
	y := s.TopMargin
	
	for _, node := range participantNodes {
		width := s.ParticipantWidth
		if node.Width > 0 {
			width = node.Width
		}
		height := s.ParticipantHeight
		if node.Height > 0 {
			height = node.Height
		}
		
		positions.Participants[node.ID] = ParticipantPosition{
			X:         x,
			Y:         y,
			Width:     width,
			Height:    height,
			LifelineX: x + width/2,
		}
		
		x += width + s.ParticipantSpacing
	}
	
	// Compute message positions
	currentY := s.TopMargin + s.ParticipantHeight + s.MessageSpacing
	
	for _, conn := range d.Connections {
		fromPos, fromOk := positions.Participants[conn.From]
		toPos, toOk := positions.Participants[conn.To]
		
		if fromOk && toOk {
			positions.Messages = append(positions.Messages, MessagePosition{
				FromX: fromPos.LifelineX,
				ToX:   toPos.LifelineX,
				Y:     currentY,
				Label: conn.Label,
				ConnectionID: conn.ID,
			})
			currentY += s.MessageSpacing
		}
	}
	
	return positions
}

// identifyParticipants finds nodes that should be treated as participants
func (s *SequenceLayout) identifyParticipants(d *diagram.Diagram) []diagram.Node {
	var participants []diagram.Node
	
	for _, node := range d.Nodes {
		// Check if node has participant or actor hint
		isParticipant := true
		if node.Hints != nil {
			nodeType := node.Hints["node-type"]
			if nodeType != "" && nodeType != "participant" && nodeType != "actor" {
				isParticipant = false
			}
		}
		
		if isParticipant {
			participants = append(participants, node)
		}
	}
	
	// Don't sort - preserve the order from the diagram
	// This allows user to reorder participants as needed

	return participants
}


// GetDiagramBounds calculates the total bounds needed for the sequence diagram
// WITHOUT modifying the original diagram
func (s *SequenceLayout) GetDiagramBounds(d *diagram.Diagram) (width, height int) {
	if d == nil || len(d.Nodes) == 0 {
		return 0, 0
	}
	
	// Calculate width based on number of participants
	numParticipants := 0
	for _, node := range d.Nodes {
		isParticipant := true
		if node.Hints != nil {
			if nodeType := node.Hints["node-type"]; nodeType != "" && nodeType != "participant" && nodeType != "actor" {
				isParticipant = false
			}
		}
		if isParticipant {
			numParticipants++
		}
	}
	
	// Calculate total width
	width = s.LeftMargin
	width += numParticipants * s.ParticipantWidth
	width += (numParticipants - 1) * s.ParticipantSpacing
	if numParticipants > 0 {
		width += s.LeftMargin  // Right margin
	}
	
	// Calculate height based on number of messages
	height = s.TopMargin + s.ParticipantHeight
	height += len(d.Connections) * s.MessageSpacing
	height += 10 // Bottom margin
	
	return width, height
}
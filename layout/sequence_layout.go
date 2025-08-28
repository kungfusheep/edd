package layout

import (
	"edd/core"
	"sort"
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
}

// NewSequenceLayout creates a new sequence diagram layout engine
func NewSequenceLayout() *SequenceLayout {
	return &SequenceLayout{
		ParticipantSpacing: 15,
		MessageSpacing:     4,
		ParticipantWidth:   20,
		ParticipantHeight:  3,
		TopMargin:         2,
		LeftMargin:        5,
	}
}

// ComputePositions calculates positions for all elements without modifying the diagram
func (s *SequenceLayout) ComputePositions(diagram *core.Diagram) *SequencePositions {
	if diagram == nil || len(diagram.Nodes) == 0 {
		return &SequencePositions{
			Participants: make(map[int]ParticipantPosition),
			Messages:     []MessagePosition{},
		}
	}
	
	positions := &SequencePositions{
		Participants: make(map[int]ParticipantPosition),
		Messages:     make([]MessagePosition, 0, len(diagram.Connections)),
	}
	
	// Identify participants
	participantNodes := s.identifyParticipants(diagram)
	
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
	
	for _, conn := range diagram.Connections {
		fromPos, fromOk := positions.Participants[conn.From]
		toPos, toOk := positions.Participants[conn.To]
		
		if fromOk && toOk {
			positions.Messages = append(positions.Messages, MessagePosition{
				FromX: fromPos.LifelineX,
				ToX:   toPos.LifelineX,
				Y:     currentY,
				Label: conn.Label,
			})
			currentY += s.MessageSpacing
		}
	}
	
	return positions
}

// identifyParticipants finds nodes that should be treated as participants
func (s *SequenceLayout) identifyParticipants(diagram *core.Diagram) []core.Node {
	var participants []core.Node
	
	for _, node := range diagram.Nodes {
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
	
	// Sort by node ID for consistent ordering
	sort.Slice(participants, func(i, j int) bool {
		return participants[i].ID < participants[j].ID
	})
	
	return participants
}


// GetDiagramBounds calculates the total bounds needed for the sequence diagram
// WITHOUT modifying the original diagram
func (s *SequenceLayout) GetDiagramBounds(diagram *core.Diagram) (width, height int) {
	if diagram == nil || len(diagram.Nodes) == 0 {
		return 0, 0
	}
	
	// Calculate width based on number of participants
	numParticipants := 0
	for _, node := range diagram.Nodes {
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
	height += len(diagram.Connections) * s.MessageSpacing
	height += 10 // Bottom margin
	
	return width, height
}
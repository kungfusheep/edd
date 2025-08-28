package layout

import (
	"edd/core"
	"sort"
	"strconv"
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

// Layout arranges nodes and connections for a sequence diagram
func (s *SequenceLayout) Layout(diagram *core.Diagram) {
	if diagram == nil || len(diagram.Nodes) == 0 {
		return
	}
	
	// Identify participants (nodes that should be at the top) - get indices
	participantIndices := s.identifyParticipantIndices(diagram)
	if len(participantIndices) == 0 {
		// No participants found, treat all nodes as participants
		participantIndices = make([]int, len(diagram.Nodes))
		for i := range participantIndices {
			participantIndices[i] = i
		}
	}
	
	// Position participants horizontally across the top
	s.positionParticipants(diagram, participantIndices)
	
	// Calculate lifeline positions for each participant
	lifelines := s.calculateLifelines(diagram, participantIndices)
	
	// Position messages vertically
	s.positionMessages(diagram, lifelines)
}

// identifyParticipantIndices finds indices of nodes that should be treated as participants
func (s *SequenceLayout) identifyParticipantIndices(diagram *core.Diagram) []int {
	var indices []int
	
	for i, node := range diagram.Nodes {
		// Check if node has participant or actor hint
		if node.Hints != nil {
			nodeType := node.Hints["node-type"]
			if nodeType == "participant" || nodeType == "actor" || nodeType == "" {
				indices = append(indices, i)
			}
		} else {
			// No hints, treat as participant by default
			indices = append(indices, i)
		}
	}
	
	// Sort by node ID for consistent ordering
	sort.Slice(indices, func(i, j int) bool {
		return diagram.Nodes[indices[i]].ID < diagram.Nodes[indices[j]].ID
	})
	
	return indices
}

// positionParticipants arranges participant nodes horizontally
func (s *SequenceLayout) positionParticipants(diagram *core.Diagram, indices []int) {
	x := s.LeftMargin
	y := s.TopMargin
	
	for _, idx := range indices {
		// Update node position directly in diagram
		diagram.Nodes[idx].X = x
		diagram.Nodes[idx].Y = y
		
		// Set default size if not specified
		if diagram.Nodes[idx].Width == 0 {
			diagram.Nodes[idx].Width = s.ParticipantWidth
		}
		if diagram.Nodes[idx].Height == 0 {
			diagram.Nodes[idx].Height = s.ParticipantHeight
		}
		
		// Move to next position
		x += diagram.Nodes[idx].Width + s.ParticipantSpacing
	}
}

// calculateLifelines determines the x-position of each participant's lifeline
func (s *SequenceLayout) calculateLifelines(diagram *core.Diagram, indices []int) map[int]int {
	lifelines := make(map[int]int)
	
	for _, idx := range indices {
		node := diagram.Nodes[idx]
		// Lifeline is at the center of the participant box
		lifelineX := node.X + node.Width/2
		lifelines[node.ID] = lifelineX
	}
	
	return lifelines
}

// positionMessages arranges messages vertically and calculates their endpoints
func (s *SequenceLayout) positionMessages(diagram *core.Diagram, lifelines map[int]int) {
	if len(diagram.Connections) == 0 {
		return
	}
	
	// Start below participants
	currentY := s.TopMargin + s.ParticipantHeight + s.MessageSpacing
	
	// For each connection, we need to store the y-position
	// Since we can't modify Connection struct directly, we'll need to use hints
	for i := range diagram.Connections {
		conn := &diagram.Connections[i]
		
		// Initialize hints if needed
		if conn.Hints == nil {
			conn.Hints = make(map[string]string)
		}
		
		// Store the y-position for this message (using strconv for proper conversion)
		conn.Hints["y-position"] = strconv.Itoa(currentY)
		
		// Store lifeline x-positions
		if fromX, ok := lifelines[conn.From]; ok {
			conn.Hints["from-x"] = strconv.Itoa(fromX)
		}
		if toX, ok := lifelines[conn.To]; ok {
			conn.Hints["to-x"] = strconv.Itoa(toX)
		}
		
		// Mark as sequence message
		conn.Hints["message-type"] = "sequence"
		
		// Move to next message position
		currentY += s.MessageSpacing
	}
}

// GetDiagramBounds calculates the total bounds needed for the sequence diagram
func (s *SequenceLayout) GetDiagramBounds(diagram *core.Diagram) (width, height int) {
	if diagram == nil || len(diagram.Nodes) == 0 {
		return 0, 0
	}
	
	// Find rightmost participant
	maxX := 0
	for _, node := range diagram.Nodes {
		rightEdge := node.X + node.Width
		if rightEdge > maxX {
			maxX = rightEdge
		}
	}
	
	// Calculate height based on number of messages
	height = s.TopMargin + s.ParticipantHeight
	height += len(diagram.Connections) * s.MessageSpacing
	height += 10 // Bottom margin
	
	return maxX, height  // maxX already includes left margin from positioning
}
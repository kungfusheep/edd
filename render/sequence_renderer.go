package render

import (
	"edd/diagram"
	"edd/layout"
	"fmt"
)

// SequenceRenderer handles rendering of sequence diagrams
type SequenceRenderer struct {
	nodeRenderer *NodeRenderer
	pathRenderer *PathRenderer
	layout       *layout.SequenceLayout
	capabilities TerminalCapabilities
}

// NewSequenceRenderer creates a new sequence diagram renderer
func NewSequenceRenderer(caps TerminalCapabilities) *SequenceRenderer {
	return &SequenceRenderer{
		nodeRenderer: NewNodeRenderer(caps),
		pathRenderer: NewPathRenderer(caps),
		layout:       layout.NewSequenceLayout(),
		capabilities: caps,
	}
}

// CanRender returns true if this renderer can handle the given diagram type.
func (r *SequenceRenderer) CanRender(diagramType diagram.DiagramType) bool {
	return diagramType == diagram.DiagramTypeSequence
}

// Render renders the sequence diagram and returns the string output.
func (r *SequenceRenderer) Render(d *diagram.Diagram) (string, error) {
	if d == nil {
		return "", fmt.Errorf("diagram is nil")
	}
	
	// Get bounds
	width, height := r.GetBounds(d)
	if width <= 0 || height <= 0 {
		return "", fmt.Errorf("invalid diagram bounds: %dx%d", width, height)
	}
	
	// Create canvas
	needsColor := HasColorHints(d)
	c := CreateCanvas(width, height, needsColor)
	
	// Render to canvas
	if err := r.RenderToCanvas(d, c); err != nil {
		return "", fmt.Errorf("failed to render sequence diagram: %w", err)
	}
	
	// Return colored output if using colored canvas
	if coloredCanvas, ok := c.(*ColoredMatrixCanvas); ok {
		return coloredCanvas.ColoredString(), nil
	}
	return c.String(), nil
}

// RenderToCanvas draws a complete sequence diagram to the provided canvas
func (r *SequenceRenderer) RenderToCanvas(d *diagram.Diagram, c Canvas) error {
	if d == nil {
		return fmt.Errorf("diagram is nil")
	}
	
	// Compute positions without modifying the diagram
	positions := r.layout.ComputePositions(d)
	
	// Draw participants
	for nodeID, pos := range positions.Participants {
		// Find the corresponding node
		var node *diagram.Node
		for i := range d.Nodes {
			if d.Nodes[i].ID == nodeID {
				node = &d.Nodes[i]
				break
			}
		}
		if node != nil {
			// Create a temporary node with computed positions for rendering
			tempNode := *node
			tempNode.X = pos.X
			tempNode.Y = pos.Y
			tempNode.Width = pos.Width
			tempNode.Height = pos.Height
			
			if err := r.nodeRenderer.RenderNode(c, tempNode); err != nil {
				return fmt.Errorf("failed to render node %d: %w", nodeID, err)
			}
		}
	}
	
	// Draw lifelines
	if err := r.drawLifelines(d, positions, c); err != nil {
		return fmt.Errorf("failed to draw lifelines: %w", err)
	}
	
	// Draw messages
	if err := r.drawMessages(d, positions, c); err != nil {
		return fmt.Errorf("failed to draw messages: %w", err)
	}
	
	return nil
}

// drawLifelines draws vertical dashed lines from each participant
func (r *SequenceRenderer) drawLifelines(d *diagram.Diagram, positions *layout.SequencePositions, c Canvas) error {
	// Get diagram bounds to know how far down to draw
	_, totalHeight := r.layout.GetDiagramBounds(d)
	
	for nodeID, pos := range positions.Participants {
		lifelineX := pos.LifelineX
		startY := pos.Y + pos.Height
		
		// Find the node to get its hints
		var node *diagram.Node
		for i := range d.Nodes {
			if d.Nodes[i].ID == nodeID {
				node = &d.Nodes[i]
				break
			}
		}
		
		// Determine lifeline style and color
		lifelineChar := '│' // Default solid
		lifelineColor := ""
		
		if node != nil && node.Hints != nil {
			// Check for lifeline-specific color, fall back to general color
			if lc, ok := node.Hints["lifeline-color"]; ok && lc != "" {
				lifelineColor = lc
			} else if c, ok := node.Hints["color"]; ok && c != "" {
				lifelineColor = c
			}
			
			// Check for lifeline style
			if ls, ok := node.Hints["lifeline-style"]; ok {
				switch ls {
				case "dashed":
					// Use dashed pattern
					for y := startY; y < totalHeight; y++ {
						if (y-startY)%2 == 0 {
							if lifelineColor != "" {
								r.setWithColor(c, diagram.Point{X: lifelineX, Y: y}, '┆', lifelineColor)
							} else {
								c.Set(diagram.Point{X: lifelineX, Y: y}, '┆')
							}
						}
					}
					continue // Skip the default drawing
				case "dotted":
					// Use dotted pattern
					for y := startY; y < totalHeight; y++ {
						if (y-startY)%2 == 0 {
							if lifelineColor != "" {
								r.setWithColor(c, diagram.Point{X: lifelineX, Y: y}, '·', lifelineColor)
							} else {
								c.Set(diagram.Point{X: lifelineX, Y: y}, '·')
							}
						}
					}
					continue // Skip the default drawing
				case "double":
					// Use double line (with space between)
					// For double lines, we use the left line for arrow connections
					// Store this info for later arrow drawing
					for y := startY; y < totalHeight; y++ {
						if lifelineColor != "" {
							r.setWithColor(c, diagram.Point{X: lifelineX - 1, Y: y}, '│', lifelineColor)
							r.setWithColor(c, diagram.Point{X: lifelineX + 1, Y: y}, '│', lifelineColor)
						} else {
							c.Set(diagram.Point{X: lifelineX - 1, Y: y}, '│')
							c.Set(diagram.Point{X: lifelineX + 1, Y: y}, '│')
						}
					}
					continue // Skip the default drawing
				}
			}
		}
		
		// Draw solid lifeline (default)
		for y := startY; y < totalHeight; y++ {
			if lifelineColor != "" {
				r.setWithColor(c, diagram.Point{X: lifelineX, Y: y}, lifelineChar, lifelineColor)
			} else {
				c.Set(diagram.Point{X: lifelineX, Y: y}, lifelineChar)
			}
		}
	}
	
	return nil
}

// ActivationPeriod represents when a participant is active
type ActivationPeriod struct {
	ParticipantID int
	StartY        int
	EndY          int
	Depth         int // For nested activations
}

// drawMessages draws horizontal arrows between lifelines and manages activations
func (r *SequenceRenderer) drawMessages(d *diagram.Diagram, positions *layout.SequencePositions, c Canvas) error {
	// Track active participants and their activation start positions
	activations := make(map[int][]int) // participantID -> stack of activation Y positions
	var allActivations []ActivationPeriod

	for _, msg := range positions.Messages {
		// Find the connection to get its hints
		var conn *diagram.Connection
		var connHints map[string]string
		for j := range d.Connections {
			if d.Connections[j].ID == msg.ConnectionID {
				conn = &d.Connections[j]
				connHints = conn.Hints
				break
			}
		}

		// Handle activation/deactivation based on hints
		if conn != nil && connHints != nil {
			// Check for activation of target
			if connHints["activate"] == "true" {
				// Start activation for the target participant
				targetID := conn.To
				if activations[targetID] == nil {
					activations[targetID] = []int{}
				}
				activations[targetID] = append(activations[targetID], msg.Y)
			}

			// Check for activation of source (orchestrator pattern)
			if connHints["activate_source"] == "true" {
				// Start activation for the source participant (the orchestrator)
				sourceID := conn.From
				if activations[sourceID] == nil {
					activations[sourceID] = []int{}
				}
				activations[sourceID] = append(activations[sourceID], msg.Y)
			}

			// Check for deactivation
			if connHints["deactivate"] == "true" {
				// End activation for the source participant
				sourceID := conn.From
				if stack, exists := activations[sourceID]; exists && len(stack) > 0 {
					startY := stack[len(stack)-1]
					// Pop from stack
					activations[sourceID] = stack[:len(stack)-1]

					// Record the activation period
					allActivations = append(allActivations, ActivationPeriod{
						ParticipantID: sourceID,
						StartY:        startY,
						EndY:          msg.Y + 1, // Extend slightly past the message
						Depth:         len(stack) - 1,
					})
				}
			}
		}

		// Draw the message arrow
		if msg.FromX < msg.ToX {
			// Left to right
			r.drawArrow(c, msg.FromX, msg.ToX, msg.Y, true, msg.Label, connHints)
		} else if msg.FromX > msg.ToX {
			// Right to left
			r.drawArrow(c, msg.FromX, msg.ToX, msg.Y, false, msg.Label, connHints)
		} else {
			// Self-message (loop back)
			r.drawSelfMessage(c, msg.FromX, msg.Y, msg.Label, connHints)
		}
	}

	// Close any remaining open activations at the end
	// But limit them to a reasonable height (not the entire diagram)
	for participantID, stack := range activations {
		for depth, startY := range stack {
			// Find the last message Y position for this participant
			lastY := startY + 10 // Default to 10 lines if no messages found
			for _, msg := range positions.Messages {
				if msg.Y > lastY && msg.Y < startY+100 { // Cap at 100 lines
					lastY = msg.Y
				}
			}

			allActivations = append(allActivations, ActivationPeriod{
				ParticipantID: participantID,
				StartY:        startY,
				EndY:          lastY + 2, // Extend slightly past last message
				Depth:         depth,
			})
		}
	}

	// Draw all activation boxes
	r.drawActivationBoxes(allActivations, positions, c)

	return nil
}

// drawActivationBoxes draws the activation boxes on the lifelines
func (r *SequenceRenderer) drawActivationBoxes(activations []ActivationPeriod, positions *layout.SequencePositions, c Canvas) {
	for _, activation := range activations {
		// Get the participant's lifeline X position
		participantPos, exists := positions.Participants[activation.ParticipantID]
		if !exists {
			continue
		}

		lifelineX := participantPos.LifelineX

		// Sanity check the Y coordinates
		if activation.StartY < 0 || activation.EndY > 1000 || activation.EndY <= activation.StartY {
			continue // Skip invalid activations
		}

		// Draw activation as a box around the lifeline
		for y := activation.StartY; y < activation.EndY && y < activation.StartY+50; y++ { // Cap at 50 lines max
			// Draw a box around the lifeline for visibility
			c.Set(diagram.Point{X: lifelineX - 1, Y: y}, '█') // Left side
			c.Set(diagram.Point{X: lifelineX, Y: y}, '█')     // Center (overwrite lifeline)
			c.Set(diagram.Point{X: lifelineX + 1, Y: y}, '█') // Right side
		}
	}
}


// drawArrow draws a horizontal arrow between two x positions
func (r *SequenceRenderer) drawArrow(c Canvas, fromX, toX, y int, leftToRight bool, label string, hints map[string]string) {
	// Determine arrow characters based on direction and style
	var lineChar rune
	style := "solid"
	if hints != nil && hints["style"] != "" {
		style = hints["style"]
	}
	
	// Set line character based on style
	switch style {
	case "dashed":
		lineChar = '╌'  // or '- ' alternating
	case "dotted":
		lineChar = '·'
	default:
		lineChar = '─'
	}
	
	// Get color from hints if available
	color := ""
	if hints != nil && hints["color"] != "" {
		color = hints["color"]
	}
	
	// Determine iteration bounds (always left to right)
	startX, endX := fromX, toX
	if startX > endX {
		startX, endX = endX, startX
	}
	
	// Draw the line
	for x := startX; x <= endX; x++ {
		var charToDraw rune
		var useArrowColor bool = true
		
		// Determine what character to draw at this position
		if leftToRight {
			if x == fromX {
				// Starting from a lifeline, use branch right
				charToDraw = '├'
				useArrowColor = false // Keep lifeline color at junction
			} else if x == endX {
				// Arrow pointing right at the end
				charToDraw = '▶'
			} else {
				// Middle section
				charToDraw = lineChar
			}
		} else {
			if x == startX {
				// Arrow pointing left at the start
				charToDraw = '◀'
			} else if x == fromX {
				// Starting from a lifeline (on the right), use branch left
				charToDraw = '┤'
				useArrowColor = false // Keep lifeline color at junction
			} else {
				// Middle section
				charToDraw = lineChar
			}
		}
		
		// Handle dashed/dotted styles for middle sections
		if charToDraw == lineChar {
			switch style {
			case "dashed":
				if (x-startX)%3 == 0 || (x-startX)%3 == 1 {
					charToDraw = '─'
				} else {
					continue // Skip this position for gap
				}
			case "dotted":
				if (x-startX)%2 == 0 {
					charToDraw = '·'
				} else {
					continue // Skip this position for gap
				}
			default:
				charToDraw = lineChar
			}
		}
		
		// Set the character
		if !useArrowColor {
			// Junction characters - preserve lifeline color
			c.Set(diagram.Point{X: x, Y: y}, charToDraw)
		} else if color != "" {
			// Arrow has explicit color
			r.setWithColor(c, diagram.Point{X: x, Y: y}, charToDraw, color)
		} else {
			// Arrow has no color - use default/white
			if coloredCanvas, ok := c.(*ColoredMatrixCanvas); ok {
				coloredCanvas.SetWithColor(diagram.Point{X: x, Y: y}, charToDraw, "")
			} else {
				c.Set(diagram.Point{X: x, Y: y}, charToDraw)
			}
		}
	}
	
	// Draw label above the arrow if present (always use default color for text)
	if label != "" {
		labelX := (fromX + toX) / 2 - len(label)/2
		for i, ch := range label {
			// Force default color by using empty string (no color)
			if coloredCanvas, ok := c.(*ColoredMatrixCanvas); ok {
				coloredCanvas.SetWithColor(diagram.Point{X: labelX + i, Y: y - 1}, ch, "")
			} else {
				c.Set(diagram.Point{X: labelX + i, Y: y - 1}, ch)
			}
		}
	}
}

// drawSelfMessage draws a message that loops back to the same lifeline
func (r *SequenceRenderer) drawSelfMessage(c Canvas, x, y int, label string, hints map[string]string) {
	// Draw a small loop to the right
	loopWidth := 6
	
	// Get style and color from hints if available
	style := "solid"
	color := ""
	if hints != nil {
		if hints["style"] != "" {
			style = hints["style"]
		}
		if hints["color"] != "" {
			color = hints["color"]
		}
	}
	
	// Determine line character based on style
	var lineChar rune
	switch style {
	case "dashed":
		lineChar = '╌'
	case "dotted":
		lineChar = '·'
	default:
		lineChar = '─'
	}
	
	// Helper to set with or without color
	setChar := func(p diagram.Point, ch rune) {
		if color != "" {
			r.setWithColor(c, p, ch, color)
		} else {
			c.Set(p, ch)
		}
	}
	
	// Top of loop - start with branch character at lifeline, then continue right
	setChar(diagram.Point{X: x, Y: y}, '├')
	for i := 1; i <= loopWidth; i++ {
		setChar(diagram.Point{X: x + i, Y: y}, lineChar)
	}
	
	// Right side corner and vertical
	setChar(diagram.Point{X: x + loopWidth + 1, Y: y}, '┐')
	setChar(diagram.Point{X: x + loopWidth + 1, Y: y + 1}, '│')
	setChar(diagram.Point{X: x + loopWidth + 1, Y: y + 2}, '┘')
	
	// Bottom of loop (with arrow) - from right corner back to lifeline
	// Draw horizontal line from corner back to position 2
	for i := loopWidth; i >= 2; i-- {
		setChar(diagram.Point{X: x + i, Y: y + 2}, lineChar)
	}
	// Place arrow at position 1 (just to the right of lifeline)
	setChar(diagram.Point{X: x + 1, Y: y + 2}, '◀')
	// The lifeline at position x will be preserved
	
	// Label
	if label != "" {
		for i, ch := range label {
			c.Set(diagram.Point{X: x + 1 + i, Y: y - 1}, ch)
		}
	}
}

// setWithColor sets a character with color if the canvas supports it
func (r *SequenceRenderer) setWithColor(c Canvas, p diagram.Point, char rune, color string) {
	// Try to set with color if the canvas supports it
	if coloredCanvas, ok := c.(*ColoredMatrixCanvas); ok {
		coloredCanvas.SetWithColor(p, char, color)
		return
	}
	// Also check if it's a type that supports SetWithColor method (like offsetCanvas)
	if colorSetter, ok := c.(interface {
		SetWithColor(diagram.Point, rune, string) error
	}); ok {
		colorSetter.SetWithColor(p, char, color)
		return
	}
	// Fall back to regular set
	c.Set(p, char)
}

// GetBounds returns the required canvas size for the diagram
func (r *SequenceRenderer) GetBounds(d *diagram.Diagram) (width, height int) {
	// Just compute bounds without modifying diagram
	return r.layout.GetDiagramBounds(d)
}
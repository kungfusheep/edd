package rendering

import (
	"edd/canvas"
	"edd/core"
	"edd/layout"
	"fmt"
)

// SequenceRenderer handles rendering of sequence diagrams
type SequenceRenderer struct {
	nodeRenderer *canvas.NodeRenderer
	pathRenderer *canvas.PathRenderer
	layout       *layout.SequenceLayout
	capabilities canvas.TerminalCapabilities
}

// NewSequenceRenderer creates a new sequence diagram renderer
func NewSequenceRenderer(caps canvas.TerminalCapabilities) *SequenceRenderer {
	return &SequenceRenderer{
		nodeRenderer: canvas.NewNodeRenderer(caps),
		pathRenderer: canvas.NewPathRenderer(caps),
		layout:       layout.NewSequenceLayout(),
		capabilities: caps,
	}
}

// CanRender returns true if this renderer can handle the given diagram type.
func (r *SequenceRenderer) CanRender(diagramType core.DiagramType) bool {
	return diagramType == core.DiagramTypeSequence
}

// Render renders the sequence diagram and returns the string output.
func (r *SequenceRenderer) Render(diagram *core.Diagram) (string, error) {
	if diagram == nil {
		return "", fmt.Errorf("diagram is nil")
	}
	
	// Get bounds
	width, height := r.GetBounds(diagram)
	if width <= 0 || height <= 0 {
		return "", fmt.Errorf("invalid diagram bounds: %dx%d", width, height)
	}
	
	// Create canvas
	needsColor := HasColorHints(diagram)
	c := CreateCanvas(width, height, needsColor)
	
	// Render to canvas
	if err := r.RenderToCanvas(diagram, c); err != nil {
		return "", fmt.Errorf("failed to render sequence diagram: %w", err)
	}
	
	return c.String(), nil
}

// RenderToCanvas draws a complete sequence diagram to the provided canvas
func (r *SequenceRenderer) RenderToCanvas(diagram *core.Diagram, c canvas.Canvas) error {
	if diagram == nil {
		return fmt.Errorf("diagram is nil")
	}
	
	// Compute positions without modifying the diagram
	positions := r.layout.ComputePositions(diagram)
	
	// Draw participants
	for nodeID, pos := range positions.Participants {
		// Find the corresponding node
		var node *core.Node
		for i := range diagram.Nodes {
			if diagram.Nodes[i].ID == nodeID {
				node = &diagram.Nodes[i]
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
	if err := r.drawLifelines(diagram, positions, c); err != nil {
		return fmt.Errorf("failed to draw lifelines: %w", err)
	}
	
	// Draw messages
	if err := r.drawMessages(positions, c); err != nil {
		return fmt.Errorf("failed to draw messages: %w", err)
	}
	
	return nil
}

// drawLifelines draws vertical dashed lines from each participant
func (r *SequenceRenderer) drawLifelines(diagram *core.Diagram, positions *layout.SequencePositions, c canvas.Canvas) error {
	// Get diagram bounds to know how far down to draw
	_, totalHeight := r.layout.GetDiagramBounds(diagram)
	
	for _, pos := range positions.Participants {
		lifelineX := pos.LifelineX
		startY := pos.Y + pos.Height
		
		// Draw dashed vertical line
		for y := startY; y < totalHeight; y++ {
			// Use dashed pattern: draw every other character
			if (y-startY)%2 == 0 {
				c.Set(core.Point{X: lifelineX, Y: y}, '│')
			}
		}
	}
	
	return nil
}

// drawMessages draws horizontal arrows between lifelines
func (r *SequenceRenderer) drawMessages(positions *layout.SequencePositions, c canvas.Canvas) error {
	for _, msg := range positions.Messages {
		// Draw the message arrow
		if msg.FromX < msg.ToX {
			// Left to right
			r.drawArrow(c, msg.FromX, msg.ToX, msg.Y, true, msg.Label)
		} else if msg.FromX > msg.ToX {
			// Right to left
			r.drawArrow(c, msg.FromX, msg.ToX, msg.Y, false, msg.Label)
		} else {
			// Self-message (loop back)
			r.drawSelfMessage(c, msg.FromX, msg.Y, msg.Label)
		}
	}
	
	return nil
}

// drawArrow draws a horizontal arrow between two x positions
func (r *SequenceRenderer) drawArrow(c canvas.Canvas, fromX, toX, y int, leftToRight bool, label string) {
	// Determine arrow characters based on direction
	var startChar, endChar, lineChar rune
	if leftToRight {
		startChar = '─'
		endChar = '▶'
		lineChar = '─'
	} else {
		startChar = '◀'
		endChar = '─'
		lineChar = '─'
	}
	
	// Ensure we go in the right direction
	if fromX > toX {
		fromX, toX = toX, fromX
	}
	
	// Draw the line
	for x := fromX; x <= toX; x++ {
		if x == fromX && !leftToRight {
			c.Set(core.Point{X: x, Y: y}, startChar)
		} else if x == toX && leftToRight {
			c.Set(core.Point{X: x, Y: y}, endChar)
		} else {
			c.Set(core.Point{X: x, Y: y}, lineChar)
		}
	}
	
	// Draw label above the arrow if present
	if label != "" {
		labelX := (fromX + toX) / 2 - len(label)/2
		for i, ch := range label {
			c.Set(core.Point{X: labelX + i, Y: y - 1}, ch)
		}
	}
}

// drawSelfMessage draws a message that loops back to the same lifeline
func (r *SequenceRenderer) drawSelfMessage(c canvas.Canvas, x, y int, label string) {
	// Draw a small loop to the right
	loopWidth := 6
	
	// Top of loop
	for i := 0; i < loopWidth; i++ {
		c.Set(core.Point{X: x + i, Y: y}, '─')
	}
	
	// Right side
	c.Set(core.Point{X: x + loopWidth, Y: y}, '┐')
	c.Set(core.Point{X: x + loopWidth, Y: y + 1}, '│')
	c.Set(core.Point{X: x + loopWidth, Y: y + 2}, '┘')
	
	// Bottom of loop (with arrow)
	for i := loopWidth; i > 0; i-- {
		if i == 1 {
			c.Set(core.Point{X: x + i, Y: y + 2}, '◀')
		} else {
			c.Set(core.Point{X: x + i, Y: y + 2}, '─')
		}
	}
	
	// Label
	if label != "" {
		for i, ch := range label {
			c.Set(core.Point{X: x + 1 + i, Y: y - 1}, ch)
		}
	}
}

// GetBounds returns the required canvas size for the diagram
func (r *SequenceRenderer) GetBounds(diagram *core.Diagram) (width, height int) {
	// Just compute bounds without modifying diagram
	return r.layout.GetDiagramBounds(diagram)
}
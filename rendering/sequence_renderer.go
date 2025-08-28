package rendering

import (
	"edd/canvas"
	"edd/core"
	"edd/layout"
	"fmt"
	"strconv"
)

// SequenceRenderer handles rendering of sequence diagrams
type SequenceRenderer struct {
	nodeRenderer *canvas.NodeRenderer
	pathRenderer *canvas.PathRenderer
	layout       *layout.SequenceLayout
}

// NewSequenceRenderer creates a new sequence diagram renderer
func NewSequenceRenderer(caps canvas.TerminalCapabilities) *SequenceRenderer {
	return &SequenceRenderer{
		nodeRenderer: canvas.NewNodeRenderer(caps),
		pathRenderer: canvas.NewPathRenderer(caps),
		layout:       layout.NewSequenceLayout(),
	}
}

// Render draws a complete sequence diagram
func (r *SequenceRenderer) Render(diagram *core.Diagram, c canvas.Canvas) error {
	if diagram == nil {
		return fmt.Errorf("diagram is nil")
	}
	
	// Apply sequence layout
	r.layout.Layout(diagram)
	
	// Draw participants (nodes)
	for i, node := range diagram.Nodes {
		if err := r.nodeRenderer.RenderNode(c, node); err != nil {
			return fmt.Errorf("failed to render node %d: %w", i, err)
		}
	}
	
	// Draw lifelines
	if err := r.drawLifelines(diagram, c); err != nil {
		return fmt.Errorf("failed to draw lifelines: %w", err)
	}
	
	// Draw messages (connections)
	if err := r.drawMessages(diagram, c); err != nil {
		return fmt.Errorf("failed to draw messages: %w", err)
	}
	
	return nil
}

// drawLifelines draws vertical dashed lines from each participant
func (r *SequenceRenderer) drawLifelines(diagram *core.Diagram, c canvas.Canvas) error {
	// Get diagram bounds to know how far down to draw
	_, totalHeight := r.layout.GetDiagramBounds(diagram)
	
	for _, node := range diagram.Nodes {
		// Check if this is a participant
		isParticipant := true
		if node.Hints != nil {
			if nodeType, ok := node.Hints["node-type"]; ok {
				if nodeType != "participant" && nodeType != "actor" && nodeType != "" {
					isParticipant = false
				}
			}
		}
		
		if !isParticipant {
			continue
		}
		
		// Calculate lifeline position (center of participant box)
		lifelineX := node.X + node.Width/2
		startY := node.Y + node.Height
		
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
func (r *SequenceRenderer) drawMessages(diagram *core.Diagram, c canvas.Canvas) error {
	for _, conn := range diagram.Connections {
		if conn.Hints == nil {
			continue
		}
		
		// Get message position from hints (stored by layout engine)
		yPosStr, hasY := conn.Hints["y-position"]
		fromXStr, hasFromX := conn.Hints["from-x"]
		toXStr, hasToX := conn.Hints["to-x"]
		
		if !hasY || !hasFromX || !hasToX {
			continue // Skip if position info missing
		}
		
		// Convert string positions back to integers
		y, _ := strconv.Atoi(yPosStr)
		fromX, _ := strconv.Atoi(fromXStr)
		toX, _ := strconv.Atoi(toXStr)
		
		// Draw the message arrow
		if fromX < toX {
			// Left to right
			r.drawArrow(c, fromX, toX, y, true, conn.Label)
		} else if fromX > toX {
			// Right to left
			r.drawArrow(c, fromX, toX, y, false, conn.Label)
		} else {
			// Self-message (loop back)
			r.drawSelfMessage(c, fromX, y, conn.Label)
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
	// Apply layout first to get accurate bounds
	r.layout.Layout(diagram)
	return r.layout.GetDiagramBounds(diagram)
}
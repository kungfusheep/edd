package render

import (
	"edd/diagram"
)

// CalculateNodeDimensions determines the width and height of nodes based on their text content.
func CalculateNodeDimensions(nodes []diagram.Node) []diagram.Node {
	result := make([]diagram.Node, len(nodes))
	copy(result, nodes)
	
	for i := range result {
		maxWidth := 0
		for _, line := range result[i].Text {
			if len(line) > maxWidth {
				maxWidth = len(line)
			}
		}
		
		// Add padding: 2 chars for borders + 2 chars for internal padding
		result[i].Width = maxWidth + 4
		// Height: number of lines + 2 for borders
		result[i].Height = len(result[i].Text) + 2
	}
	
	return result
}

// CalculateBounds determines the canvas size needed to fit all nodes and paths.
func CalculateBounds(nodes []diagram.Node, paths map[int]diagram.Path) diagram.Bounds {
	if len(nodes) == 0 {
		return diagram.Bounds{Min: diagram.Point{X: 0, Y: 0}, Max: diagram.Point{X: 10, Y: 10}}
	}
	
	minX, minY := nodes[0].X, nodes[0].Y
	maxX, maxY := nodes[0].X+nodes[0].Width, nodes[0].Y+nodes[0].Height
	
	// Consider all nodes
	for _, node := range nodes[1:] {
		if node.X < minX {
			minX = node.X
		}
		if node.Y < minY {
			minY = node.Y
		}
		if node.X+node.Width > maxX {
			maxX = node.X + node.Width
		}
		if node.Y+node.Height > maxY {
			maxY = node.Y + node.Height
		}
	}
	
	// Also consider all path points
	for _, path := range paths {
		for _, point := range path.Points {
			if point.X < minX {
				minX = point.X
			}
			if point.Y < minY {
				minY = point.Y
			}
			if point.X > maxX {
				maxX = point.X
			}
			if point.Y > maxY {
				maxY = point.Y
			}
		}
	}
	
	// Add some margin
	minX -= 1
	minY -= 1
	maxX += 1
	maxY += 1
	
	return diagram.Bounds{
		Min: diagram.Point{X: minX, Y: minY},
		Max: diagram.Point{X: maxX, Y: maxY},
	}
}

// HasColorHints checks if any nodes or connections have color or style hints.
func HasColorHints(d *diagram.Diagram) bool {
	// Check nodes
	for _, node := range d.Nodes {
		if node.Hints != nil {
			if _, hasColor := node.Hints["color"]; hasColor {
				return true
			}
			if _, hasLifelineColor := node.Hints["lifeline-color"]; hasLifelineColor {
				return true
			}
			if _, hasLifelineStyle := node.Hints["lifeline-style"]; hasLifelineStyle {
				return true
			}
			if _, hasBold := node.Hints["bold"]; hasBold {
				return true
			}
		}
	}
	
	// Check connections
	for _, conn := range d.Connections {
		if conn.Hints != nil {
			if _, hasColor := conn.Hints["color"]; hasColor {
				return true
			}
			if _, hasBold := conn.Hints["bold"]; hasBold {
				return true
			}
		}
	}
	
	return false
}

// CreateCanvas creates the appropriate canvas type based on whether colors are needed.
func CreateCanvas(width, height int, needsColor bool) Canvas {
	if needsColor {
		return NewColoredMatrixCanvas(width, height)
	}
	return NewMatrixCanvas(width, height)
}

// OffsetCanvas wraps a canvas and translates all coordinates by an offset.
// This is useful when the diagram has negative coordinates.
type OffsetCanvas struct {
	canvas Canvas
	offset diagram.Point
}

// NewOffsetCanvas creates a new offset 
func NewOffsetCanvas(c Canvas, offset diagram.Point) *OffsetCanvas {
	return &OffsetCanvas{
		canvas: c,
		offset: offset,
	}
}

// Set places a character at the given position (after applying offset).
func (oc *OffsetCanvas) Set(p diagram.Point, char rune) error {
	// Translate coordinates
	translated := diagram.Point{
		X: p.X - oc.offset.X,
		Y: p.Y - oc.offset.Y,
	}
	return oc.Set(translated, char)
}

// SetWithColor sets a character with color if the underlying canvas supports it.
func (oc *OffsetCanvas) SetWithColor(p diagram.Point, char rune, color string) error {
	// Translate coordinates
	translated := diagram.Point{
		X: p.X - oc.offset.X,
		Y: p.Y - oc.offset.Y,
	}
	// Try to set with color if the underlying canvas supports it
	if coloredCanvas, ok := oc.canvas.(*ColoredMatrixCanvas); ok {
		return coloredCanvas.SetWithColor(translated, char, color)
	}
	// Fall back to regular set
	return oc.Set(translated, char)
}

// SetWithColorAndStyle sets a character with color and style if the underlying canvas supports it.
func (oc *OffsetCanvas) SetWithColorAndStyle(p diagram.Point, char rune, color string, style string) error {
	// Translate coordinates
	translated := diagram.Point{
		X: p.X - oc.offset.X,
		Y: p.Y - oc.offset.Y,
	}
	// Try to set with color and style if the underlying canvas supports it
	if coloredCanvas, ok := oc.canvas.(*ColoredMatrixCanvas); ok {
		return coloredCanvas.SetWithColorAndStyle(translated, char, color, style)
	}
	// Fall back to regular set
	return oc.Set(translated, char)
}

// Size returns the size of the underlying 
func (oc *OffsetCanvas) Size() (width, height int) {
	return oc.Size()
}

// Get returns the character at the given position (after applying offset).
func (oc *OffsetCanvas) Get(p diagram.Point) rune {
	// Translate coordinates
	translated := diagram.Point{
		X: p.X - oc.offset.X,
		Y: p.Y - oc.offset.Y,
	}
	return oc.Get(translated)
}

// Clear clears the underlying 
func (oc *OffsetCanvas) Clear() {
	oc.Clear()
}

// String returns the canvas as a string.
func (oc *OffsetCanvas) String() string {
	return oc.String()
}
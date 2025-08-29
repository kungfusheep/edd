package rendering

import (
	"edd/canvas"
	"edd/core"
)

// CalculateNodeDimensions determines the width and height of nodes based on their text content.
func CalculateNodeDimensions(nodes []core.Node) []core.Node {
	result := make([]core.Node, len(nodes))
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
func CalculateBounds(nodes []core.Node, paths map[int]core.Path) core.Bounds {
	if len(nodes) == 0 {
		return core.Bounds{Min: core.Point{X: 0, Y: 0}, Max: core.Point{X: 10, Y: 10}}
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
	
	return core.Bounds{
		Min: core.Point{X: minX, Y: minY},
		Max: core.Point{X: maxX, Y: maxY},
	}
}

// HasColorHints checks if any nodes or connections have color or style hints.
func HasColorHints(diagram *core.Diagram) bool {
	// Check nodes
	for _, node := range diagram.Nodes {
		if node.Hints != nil {
			if _, hasColor := node.Hints["color"]; hasColor {
				return true
			}
			if _, hasBold := node.Hints["bold"]; hasBold {
				return true
			}
		}
	}
	
	// Check connections
	for _, conn := range diagram.Connections {
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
func CreateCanvas(width, height int, needsColor bool) canvas.Canvas {
	if needsColor {
		return canvas.NewColoredMatrixCanvas(width, height)
	}
	return canvas.NewMatrixCanvas(width, height)
}

// OffsetCanvas wraps a canvas and translates all coordinates by an offset.
// This is useful when the diagram has negative coordinates.
type OffsetCanvas struct {
	canvas canvas.Canvas
	offset core.Point
}

// NewOffsetCanvas creates a new offset canvas.
func NewOffsetCanvas(c canvas.Canvas, offset core.Point) *OffsetCanvas {
	return &OffsetCanvas{
		canvas: c,
		offset: offset,
	}
}

// Set places a character at the given position (after applying offset).
func (oc *OffsetCanvas) Set(p core.Point, char rune) error {
	// Translate coordinates
	translated := core.Point{
		X: p.X - oc.offset.X,
		Y: p.Y - oc.offset.Y,
	}
	return oc.canvas.Set(translated, char)
}

// SetWithColor sets a character with color if the underlying canvas supports it.
func (oc *OffsetCanvas) SetWithColor(p core.Point, char rune, color string) error {
	// Translate coordinates
	translated := core.Point{
		X: p.X - oc.offset.X,
		Y: p.Y - oc.offset.Y,
	}
	// Try to set with color if the underlying canvas supports it
	if coloredCanvas, ok := oc.canvas.(*canvas.ColoredMatrixCanvas); ok {
		return coloredCanvas.SetWithColor(translated, char, color)
	}
	// Fall back to regular set
	return oc.canvas.Set(translated, char)
}

// SetWithColorAndStyle sets a character with color and style if the underlying canvas supports it.
func (oc *OffsetCanvas) SetWithColorAndStyle(p core.Point, char rune, color string, style string) error {
	// Translate coordinates
	translated := core.Point{
		X: p.X - oc.offset.X,
		Y: p.Y - oc.offset.Y,
	}
	// Try to set with color and style if the underlying canvas supports it
	if coloredCanvas, ok := oc.canvas.(*canvas.ColoredMatrixCanvas); ok {
		return coloredCanvas.SetWithColorAndStyle(translated, char, color, style)
	}
	// Fall back to regular set
	return oc.canvas.Set(translated, char)
}

// Size returns the size of the underlying canvas.
func (oc *OffsetCanvas) Size() (width, height int) {
	return oc.canvas.Size()
}

// Get returns the character at the given position (after applying offset).
func (oc *OffsetCanvas) Get(p core.Point) rune {
	// Translate coordinates
	translated := core.Point{
		X: p.X - oc.offset.X,
		Y: p.Y - oc.offset.Y,
	}
	return oc.canvas.Get(translated)
}

// Clear clears the underlying canvas.
func (oc *OffsetCanvas) Clear() {
	oc.canvas.Clear()
}

// String returns the canvas as a string.
func (oc *OffsetCanvas) String() string {
	return oc.canvas.String()
}
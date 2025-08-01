package layout

import (
	"edd/core"
	"strings"
)

// VisualRenderer creates ASCII visualizations of layouts
type VisualRenderer struct {
	canvas [][]rune
	width  int
	height int
}

// NewVisualRenderer creates a renderer with the given canvas size
func NewVisualRenderer(width, height int) *VisualRenderer {
	canvas := make([][]rune, height)
	for i := range canvas {
		canvas[i] = make([]rune, width)
		for j := range canvas[i] {
			canvas[i][j] = ' '
		}
	}
	return &VisualRenderer{
		canvas: canvas,
		width:  width,
		height: height,
	}
}

// RenderNode draws a node on the canvas
func (r *VisualRenderer) RenderNode(node core.Node) {
	// Draw box
	for y := node.Y; y < node.Y+node.Height && y < r.height; y++ {
		for x := node.X; x < node.X+node.Width && x < r.width; x++ {
			if x == node.X || x == node.X+node.Width-1 {
				r.canvas[y][x] = '│'
			} else if y == node.Y || y == node.Y+node.Height-1 {
				r.canvas[y][x] = '─'
			}
		}
	}
	
	// Draw corners
	if node.Y < r.height && node.X < r.width {
		r.canvas[node.Y][node.X] = '┌'
	}
	if node.Y < r.height && node.X+node.Width-1 < r.width {
		r.canvas[node.Y][node.X+node.Width-1] = '┐'
	}
	if node.Y+node.Height-1 < r.height && node.X < r.width {
		r.canvas[node.Y+node.Height-1][node.X] = '└'
	}
	if node.Y+node.Height-1 < r.height && node.X+node.Width-1 < r.width {
		r.canvas[node.Y+node.Height-1][node.X+node.Width-1] = '┘'
	}
	
	// Draw text
	for i, line := range node.Text {
		textY := node.Y + 1 + i
		if textY >= node.Y+node.Height-1 || textY >= r.height {
			break
		}
		textX := node.X + 2
		for _, ch := range line {
			if textX >= node.X+node.Width-2 || textX >= r.width {
				break
			}
			r.canvas[textY][textX] = ch
			textX++
		}
	}
}

// RenderConnection draws a simple connection arrow
func (r *VisualRenderer) RenderConnection(from, to core.Node) {
	// Simple horizontal line from right edge of 'from' to left edge of 'to'
	fromX := from.X + from.Width
	fromY := from.Y + from.Height/2
	toX := to.X - 1
	_ = to.Y + to.Height/2 // toY - might use for vertical routing later
	
	// Draw horizontal line
	if fromY < r.height && fromY >= 0 {
		for x := fromX; x < toX && x < r.width; x++ {
			if x >= 0 && r.canvas[fromY][x] == ' ' {
				r.canvas[fromY][x] = '─'
			}
		}
		// Arrow head
		if toX < r.width && toX >= 0 {
			r.canvas[fromY][toX] = '>'
		}
	}
}

// String returns the canvas as a string
func (r *VisualRenderer) String() string {
	var lines []string
	for _, row := range r.canvas {
		lines = append(lines, string(row))
	}
	return strings.Join(lines, "\n")
}

// RenderLayout creates a visual representation of a layout
func RenderLayout(nodes []core.Node, connections []core.Connection) string {
	if len(nodes) == 0 {
		return "Empty layout"
	}
	
	// Find bounds
	maxX, maxY := 0, 0
	for _, node := range nodes {
		if node.X+node.Width > maxX {
			maxX = node.X + node.Width
		}
		if node.Y+node.Height > maxY {
			maxY = node.Y + node.Height
		}
	}
	
	// Create renderer with some padding
	renderer := NewVisualRenderer(maxX+5, maxY+2)
	
	// Draw all nodes
	for _, node := range nodes {
		renderer.RenderNode(node)
	}
	
	// Draw connections
	nodeMap := make(map[int]core.Node)
	for _, node := range nodes {
		nodeMap[node.ID] = node
	}
	
	for _, conn := range connections {
		if from, ok := nodeMap[conn.From]; ok {
			if to, ok := nodeMap[conn.To]; ok {
				renderer.RenderConnection(from, to)
			}
		}
	}
	
	return renderer.String()
}


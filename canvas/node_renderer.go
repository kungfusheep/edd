package canvas

import (
	"edd/core"
)

// NodeRenderer handles rendering of nodes with various styles and hints
type NodeRenderer struct {
	caps         TerminalCapabilities
	defaultStyle NodeStyle
}

// NewNodeRenderer creates a new node renderer with the given capabilities
func NewNodeRenderer(caps TerminalCapabilities) *NodeRenderer {
	return &NodeRenderer{
		caps:         caps,
		defaultStyle: DefaultNodeStyle(caps),
	}
}

// RenderNode draws a node on the canvas using default style
func (r *NodeRenderer) RenderNode(canvas Canvas, node core.Node) error {
	return r.RenderNodeWithHints(canvas, node, node.Hints)
}

// RenderNodeWithHints draws a node with visual hints applied
func (r *NodeRenderer) RenderNodeWithHints(canvas Canvas, node core.Node, hints map[string]string) error {
	// Select the box style based on hints
	style := r.defaultStyle
	if hints != nil {
		if styleName, ok := hints["style"]; ok {
			style = GetNodeStyle(styleName, r.caps)
		}
	}
	
	// Get color from hints (if any)
	var nodeColor string
	if hints != nil {
		nodeColor = hints["color"]
	}
	
	// Draw the box border
	if err := r.drawBox(canvas, node, style, nodeColor); err != nil {
		return err
	}
	
	// Draw the text inside the box
	if err := r.drawText(canvas, node, hints); err != nil {
		return err
	}
	
	return nil
}

// RenderShadowOnly draws only the shadow for a node (for render ordering)
func (r *NodeRenderer) RenderShadowOnly(canvas Canvas, node core.Node) {
	if node.Hints != nil {
		if shadow, ok := node.Hints["shadow"]; ok {
			r.drawShadow(canvas, node, shadow, node.Hints["shadow-density"])
		}
	}
}

// drawShadow draws a shadow for the node
func (r *NodeRenderer) drawShadow(canvas Canvas, node core.Node, direction string, density string) {
	// Choose shadow character based on density
	var shadowChar rune
	switch density {
	case "medium":
		shadowChar = '▒'
	default: // "light" or unspecified
		shadowChar = '░'
	}
	
	// Draw shadow based on direction (default to southeast if specified)
	// We only support southeast shadow for clarity
	if direction == "southeast" || direction != "" {
		// Right edge shadow
		for y := node.Y + 1; y <= node.Y + node.Height - 1; y++ {
			r.setChar(canvas, core.Point{X: node.X + node.Width, Y: y}, shadowChar, "")
		}
		// Bottom edge shadow (full width + corner)
		for x := node.X + 1; x <= node.X + node.Width; x++ {
			r.setChar(canvas, core.Point{X: x, Y: node.Y + node.Height}, shadowChar, "")
		}
	}
}

// drawBox draws the box border for a node
func (r *NodeRenderer) drawBox(canvas Canvas, node core.Node, style NodeStyle, color string) error {
	// Top border
	r.setChar(canvas, core.Point{X: node.X, Y: node.Y}, style.TopLeft, color)
	for x := node.X + 1; x < node.X + node.Width - 1; x++ {
		r.setChar(canvas, core.Point{X: x, Y: node.Y}, style.Horizontal, color)
	}
	r.setChar(canvas, core.Point{X: node.X + node.Width - 1, Y: node.Y}, style.TopRight, color)
	
	// Side borders
	for y := node.Y + 1; y < node.Y + node.Height - 1; y++ {
		r.setChar(canvas, core.Point{X: node.X, Y: y}, style.Vertical, color)
		r.setChar(canvas, core.Point{X: node.X + node.Width - 1, Y: y}, style.Vertical, color)
	}
	
	// Bottom border
	r.setChar(canvas, core.Point{X: node.X, Y: node.Y + node.Height - 1}, style.BottomLeft, color)
	for x := node.X + 1; x < node.X + node.Width - 1; x++ {
		r.setChar(canvas, core.Point{X: x, Y: node.Y + node.Height - 1}, style.Horizontal, color)
	}
	r.setChar(canvas, core.Point{X: node.X + node.Width - 1, Y: node.Y + node.Height - 1}, style.BottomRight, color)
	
	return nil
}

// drawText draws the text content inside a node
func (r *NodeRenderer) drawText(canvas Canvas, node core.Node, hints map[string]string) error {
	// Get text color and style from hints (if any)
	var textColor string
	var isBold bool
	var isItalic bool
	if hints != nil {
		textColor = hints["textColor"]
		isBold = hints["bold"] == "true"
		isItalic = hints["italic"] == "true"
	}
	
	// Draw each line of text
	for i, line := range node.Text {
		y := node.Y + 1 + i
		x := node.X + 1 // 1 char padding from left border
		
		// Add space before text
		r.setCharWithStyle(canvas, core.Point{X: x, Y: y}, ' ', textColor, isBold, isItalic)
		x++
		
		for j, ch := range line {
			if x+j < node.X+node.Width-1 { // Keep text within borders
				r.setCharWithStyle(canvas, core.Point{X: x + j, Y: y}, ch, textColor, isBold, isItalic)
			}
		}
		
		// Add space after text if there's room
		textEnd := x + len(line)
		if textEnd < node.X+node.Width-1 {
			r.setCharWithStyle(canvas, core.Point{X: textEnd, Y: y}, ' ', textColor, isBold, isItalic)
		}
	}
	
	return nil
}

// setChar sets a character on the canvas with optional color
func (r *NodeRenderer) setChar(canvas Canvas, p core.Point, char rune, color string) {
	if color != "" {
		// Try to set with color if the canvas supports it
		if coloredCanvas, ok := canvas.(*ColoredMatrixCanvas); ok {
			coloredCanvas.SetWithColor(p, char, color)
			return
		}
		// Also check if it's a type that supports SetWithColor method (like offsetCanvas)
		if colorSetter, ok := canvas.(interface {
			SetWithColor(core.Point, rune, string) error
		}); ok {
			colorSetter.SetWithColor(p, char, color)
			return
		}
	}
	// Fall back to regular set
	canvas.Set(p, char)
}

// setCharWithStyle sets a character on the canvas with optional color and style
func (r *NodeRenderer) setCharWithStyle(canvas Canvas, p core.Point, char rune, color string, bold bool, italic bool) {
	// Build style string
	style := ""
	if bold && italic {
		style = "bold+italic"
	} else if bold {
		style = "bold"
	} else if italic {
		style = "italic"
	}
	
	// Try to set with color and style if the canvas supports it
	if coloredCanvas, ok := canvas.(*ColoredMatrixCanvas); ok {
		coloredCanvas.SetWithColorAndStyle(p, char, color, style)
		return
	}
	// Check if it's an offset canvas that wraps a colored canvas
	if offsetSetter, ok := canvas.(interface {
		SetWithColorAndStyle(core.Point, rune, string, string) error
	}); ok {
		offsetSetter.SetWithColorAndStyle(p, char, color, style)
		return
	}
	// Fall back to color-only setting
	r.setChar(canvas, p, char, color)
}
package canvas

import (
	"edd/core"
	"strings"
)

// ColoredMatrixCanvas extends MatrixCanvas to support colored characters
type ColoredMatrixCanvas struct {
	*MatrixCanvas
	colors [][]string // Color code for each position
	styles [][]string // Style code for each position (e.g., bold)
}

// NewColoredMatrixCanvas creates a new colored matrix canvas
func NewColoredMatrixCanvas(width, height int) *ColoredMatrixCanvas {
	// Initialize color matrix
	colors := make([][]string, height)
	for i := range colors {
		colors[i] = make([]string, width)
	}
	
	// Initialize style matrix
	styles := make([][]string, height)
	for i := range styles {
		styles[i] = make([]string, width)
	}
	
	return &ColoredMatrixCanvas{
		MatrixCanvas: NewMatrixCanvas(width, height),
		colors:       colors,
		styles:       styles,
	}
}

// GetColorAt returns the color code at a given position
func (c *ColoredMatrixCanvas) GetColorAt(p core.Point) string {
	if p.Y >= 0 && p.Y < len(c.colors) && p.X >= 0 && p.X < len(c.colors[0]) {
		return c.colors[p.Y][p.X]
	}
	return ""
}

// SetWithColor sets a character with a specific color
func (c *ColoredMatrixCanvas) SetWithColor(p core.Point, char rune, color string) error {
	// Set the character
	if err := c.MatrixCanvas.Set(p, char); err != nil {
		return err
	}
	
	// Store the color code - always use the new color for arrows/messages
	// This ensures arrow colors take precedence over lifeline colors at junctions
	if p.Y >= 0 && p.Y < len(c.colors) && p.X >= 0 && p.X < len(c.colors[0]) {
		c.colors[p.Y][p.X] = GetColorCode(color)
	}
	
	return nil
}

// SetWithStyle sets a character with a specific style
func (c *ColoredMatrixCanvas) SetWithStyle(p core.Point, char rune, style string) error {
	// Set the character
	if err := c.MatrixCanvas.Set(p, char); err != nil {
		return err
	}
	
	// Store the style code
	if p.Y >= 0 && p.Y < len(c.styles) && p.X >= 0 && p.X < len(c.styles[0]) {
		c.styles[p.Y][p.X] = GetStyleCode(style)
	}
	
	return nil
}

// SetWithColorAndStyle sets a character with both color and style
func (c *ColoredMatrixCanvas) SetWithColorAndStyle(p core.Point, char rune, color string, style string) error {
	// Set the character
	if err := c.MatrixCanvas.Set(p, char); err != nil {
		return err
	}
	
	// Store the color and style codes
	if p.Y >= 0 && p.Y < len(c.colors) && p.X >= 0 && p.X < len(c.colors[0]) {
		c.colors[p.Y][p.X] = GetColorCode(color)
		c.styles[p.Y][p.X] = GetStyleCode(style)
	}
	
	return nil
}

// ColoredString returns the canvas as a string with ANSI color codes
func (c *ColoredMatrixCanvas) ColoredString() string {
	var sb strings.Builder
	
	for y := 0; y < c.height; y++ {
		currentColor := ""
		currentStyle := ""
		for x := 0; x < c.width; x++ {
			char := c.matrix[y][x]
			color := ""
			style := ""
			if y < len(c.colors) && x < len(c.colors[y]) {
				color = c.colors[y][x]
			}
			if y < len(c.styles) && x < len(c.styles[y]) {
				style = c.styles[y][x]
			}
			
			// Check if we need to change color or style
			if color != currentColor || style != currentStyle {
				// Reset if we had any formatting
				if currentColor != "" || currentStyle != "" {
					sb.WriteString(ColorReset)
				}
				
				// Apply new style first (if any)
				if style != "" {
					sb.WriteString(style)
				}
				// Then apply color (if any)
				if color != "" {
					sb.WriteString(color)
				}
				
				currentColor = color
				currentStyle = style
			}
			
			// Write the character
			if char == 0 {
				sb.WriteRune(' ')
			} else {
				sb.WriteRune(char)
			}
		}
		
		// Reset formatting at end of line if needed
		if currentColor != "" || currentStyle != "" {
			sb.WriteString(ColorReset)
		}
		
		// Add newline except for last line
		if y < c.height-1 {
			sb.WriteRune('\n')
		}
	}
	
	return sb.String()
}

// TrackingCanvas wraps a canvas and tracks the current color being used
type TrackingCanvas struct {
	underlying *ColoredMatrixCanvas
	color      string
}

// NewTrackingCanvas creates a canvas that applies a specific color to all operations
func NewTrackingCanvas(canvas *ColoredMatrixCanvas, color string) *TrackingCanvas {
	return &TrackingCanvas{
		underlying: canvas,
		color:      color,
	}
}

// Set sets a character with the tracked color
func (t *TrackingCanvas) Set(p core.Point, char rune) error {
	if t.color != "" {
		return t.underlying.SetWithColor(p, char, t.color)
	}
	return t.underlying.Set(p, char)
}

// Get gets a character from the canvas
func (t *TrackingCanvas) Get(p core.Point) rune {
	return t.underlying.Get(p)
}

// Size returns the canvas size
func (t *TrackingCanvas) Size() (int, int) {
	return t.underlying.Size()
}

// Clear clears the canvas
func (t *TrackingCanvas) Clear() {
	t.underlying.Clear()
}

// String returns the string representation (without colors)
func (t *TrackingCanvas) String() string {
	return t.underlying.String()
}
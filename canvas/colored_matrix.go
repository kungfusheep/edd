package canvas

import (
	"edd/core"
	"strings"
)

// ColoredMatrixCanvas extends MatrixCanvas to support colored characters
type ColoredMatrixCanvas struct {
	*MatrixCanvas
	colors [][]string // Color code for each position
}

// NewColoredMatrixCanvas creates a new colored matrix canvas
func NewColoredMatrixCanvas(width, height int) *ColoredMatrixCanvas {
	// Initialize color matrix
	colors := make([][]string, height)
	for i := range colors {
		colors[i] = make([]string, width)
	}
	
	return &ColoredMatrixCanvas{
		MatrixCanvas: NewMatrixCanvas(width, height),
		colors:       colors,
	}
}

// SetWithColor sets a character with a specific color
func (c *ColoredMatrixCanvas) SetWithColor(p core.Point, char rune, color string) error {
	// Set the character
	if err := c.MatrixCanvas.Set(p, char); err != nil {
		return err
	}
	
	// Store the color code
	if p.Y >= 0 && p.Y < len(c.colors) && p.X >= 0 && p.X < len(c.colors[0]) {
		c.colors[p.Y][p.X] = GetColorCode(color)
	}
	
	return nil
}

// ColoredString returns the canvas as a string with ANSI color codes
func (c *ColoredMatrixCanvas) ColoredString() string {
	var sb strings.Builder
	
	for y := 0; y < c.height; y++ {
		currentColor := ""
		for x := 0; x < c.width; x++ {
			char := c.matrix[y][x]
			color := ""
			if y < len(c.colors) && x < len(c.colors[y]) {
				color = c.colors[y][x]
			}
			
			// Change color if needed
			if color != currentColor {
				if currentColor != "" {
					sb.WriteString(ColorReset)
				}
				if color != "" {
					sb.WriteString(color)
				}
				currentColor = color
			}
			
			// Write the character
			if char == 0 {
				sb.WriteRune(' ')
			} else {
				sb.WriteRune(char)
			}
		}
		
		// Reset color at end of line if needed
		if currentColor != "" {
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
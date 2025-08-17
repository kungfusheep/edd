package canvas

import "edd/core"

// ANSI color codes
const (
	ColorReset   = "\033[0m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorWhite   = "\033[37m"
	
	// Text style codes
	StyleBold    = "\033[1m"
	StyleDim     = "\033[2m"
	StyleItalic  = "\033[3m"
	StyleReset   = "\033[0m"
)

// ColorCanvas wraps a canvas and applies color to characters
type ColorCanvas struct {
	underlying Canvas
	colorCode  string
}

// NewColorCanvas creates a canvas that applies a color to all set characters
func NewColorCanvas(canvas Canvas, color string) *ColorCanvas {
	colorCode := ""
	switch color {
	case "red":
		colorCode = ColorRed
	case "green":
		colorCode = ColorGreen
	case "yellow":
		colorCode = ColorYellow
	case "blue":
		colorCode = ColorBlue
	case "magenta":
		colorCode = ColorMagenta
	case "cyan":
		colorCode = ColorCyan
	case "white":
		colorCode = ColorWhite
	}
	
	return &ColorCanvas{
		underlying: canvas,
		colorCode:  colorCode,
	}
}

// Set sets a character with color
func (c *ColorCanvas) Set(p core.Point, char rune) error {
	if c.colorCode == "" {
		return c.underlying.Set(p, char)
	}
	
	// For now, we can't directly embed color codes in the canvas
	// This would require changing how the canvas stores and renders data
	// Instead, we'll just pass through to the underlying canvas
	return c.underlying.Set(p, char)
}

// Get gets a character from the canvas
func (c *ColorCanvas) Get(p core.Point) rune {
	return c.underlying.Get(p)
}

// Size returns the canvas size
func (c *ColorCanvas) Size() (int, int) {
	return c.underlying.Size()
}

// Clear clears the canvas
func (c *ColorCanvas) Clear() {
	c.underlying.Clear()
}

// String returns the string representation
func (c *ColorCanvas) String() string {
	return c.underlying.String()
}

// ApplyColorToString applies ANSI color codes to specific characters in a string
func ApplyColorToString(output string, shouldColor func(x, y int, char rune) string) string {
	// This would need to parse the output and apply colors
	// For now, return as-is
	return output
}

// GetColorCode returns the ANSI color code for a color name
func GetColorCode(color string) string {
	switch color {
	case "red":
		return ColorRed
	case "green":
		return ColorGreen
	case "yellow":
		return ColorYellow
	case "blue":
		return ColorBlue
	case "magenta":
		return ColorMagenta
	case "cyan":
		return ColorCyan
	case "white":
		return ColorWhite
	default:
		return ""
	}
}

// GetStyleCode returns the ANSI style code for a style name
func GetStyleCode(style string) string {
	switch style {
	case "bold":
		return StyleBold
	case "dim":
		return StyleDim
	case "italic":
		return StyleItalic
	case "bold+italic":
		return StyleBold + StyleItalic
	default:
		return ""
	}
}
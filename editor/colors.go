package editor

import (
	"edd/canvas"
	"edd/core"
	"fmt"
)

// ColoredCanvas wraps a canvas to add color support
type ColoredCanvas struct {
	canvas.Canvas
	currentColor string
}

// NewColoredCanvas creates a canvas wrapper with color support
func NewColoredCanvas(c canvas.Canvas) *ColoredCanvas {
	return &ColoredCanvas{Canvas: c}
}

// RenderColoredPath renders a path with the specified color
func RenderColoredPath(output string, path core.Path, color string, style string) string {
	// This is a simplified implementation that adds color codes to the output
	// In a real implementation, we'd need to track which characters belong to which path
	
	colorCode := getColorCode(color)
	if colorCode == "" {
		return output
	}
	
	// For now, we'll return the output with color codes
	// A full implementation would need to track path positions
	return output
}

// getColorCode returns the ANSI color code for the given color name
func getColorCode(color string) string {
	switch color {
	case "red":
		return "\033[31m"
	case "green":
		return "\033[32m"
	case "yellow":
		return "\033[33m"
	case "blue":
		return "\033[34m"
	case "magenta":
		return "\033[35m"
	case "cyan":
		return "\033[36m"
	case "white":
		return "\033[37m"
	default:
		return ""
	}
}

// getStyleCharacters returns the characters to use for a given style
func getStyleCharacters(style string) (horizontal, vertical rune) {
	switch style {
	case "dashed":
		return '╌', '╎'  // Unicode dashed box drawing
	case "dotted":
		return '·', '·'
	case "double":
		return '═', '║'
	default:
		return '─', '│'
	}
}

// applyColorCode is a placeholder - in reality we need to modify the canvas output
func applyColorCode(c canvas.Canvas, color string) {
	// This would need to be implemented differently
	// We might need to track colored segments in the canvas
}

// resetColorCode resets to default color
func resetColorCode(c canvas.Canvas) {
	// This would need to be implemented differently
}

// WrapWithColor wraps a string segment with ANSI color codes
func WrapWithColor(text string, color string) string {
	colorCode := getColorCode(color)
	if colorCode == "" {
		return text
	}
	return fmt.Sprintf("%s%s\033[0m", colorCode, text)
}
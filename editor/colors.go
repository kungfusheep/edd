package editor

import (
	"edd/render"
	"edd/diagram"
	"fmt"
)

// ColoredCanvas wraps a canvas to add color support
type ColoredCanvas struct {
	render.Canvas
}

// NewColoredCanvas creates a canvas wrapper with color support
func NewColoredCanvas(c render.Canvas) *ColoredCanvas {
	return &ColoredCanvas{Canvas: c}
}

// RenderColoredPath renders a path with the specified color
func RenderColoredPath(output string, path diagram.Path, color string, style string) string {
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


// WrapWithColor wraps a string segment with ANSI color codes
func WrapWithColor(text string, color string) string {
	colorCode := getColorCode(color)
	if colorCode == "" {
		return text
	}
	return fmt.Sprintf("%s%s\033[0m", colorCode, text)
}
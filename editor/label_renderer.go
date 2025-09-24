package editor

import (
	"fmt"
	"strings"
)

// LabelPosition represents where a label should be drawn
type LabelPosition struct {
	NodeID    int
	Label     rune
	ViewportX int
	ViewportY int
	IsFrom    bool // For connection mode, marks the FROM node
}

// CalculateLabelPositions computes where labels should be drawn in the viewport
func (e *TUIEditor) CalculateLabelPositions(hasScrollIndicator bool) []LabelPosition {
	positions := []LabelPosition{}

	// Process node labels
	for nodeID, label := range e.jumpLabels {
		if pos, ok := e.nodePositions[nodeID]; ok {
			viewportY := 0
			viewportX := pos.X

			// Adjust X position based on diagram type
			if e.diagram.Type == "sequence" && pos.Y < 7 {
				// For sequence diagram participants, place label at the left edge of the box
				// The box extends a bit to the left of the text position
				viewportX = pos.X - 1
			} else if e.diagram.Type == "box" {
				// For regular box diagrams, place inside the box corner
				viewportX = pos.X + 2
			}

			// Calculate Y position using consolidated transformation logic
			viewportY = e.TransformToViewport(pos.Y, hasScrollIndicator)

			// Only include if within viewport
			if viewportY >= 1 && viewportY <= e.height-3 {
				positions = append(positions, LabelPosition{
					NodeID:    nodeID,
					Label:     label,
					ViewportX: viewportX,
					ViewportY: viewportY,
					IsFrom:    e.jumpAction == JumpActionConnectTo && nodeID == e.selected,
				})
			}
		}
	}

	return positions
}

// RenderLabelsToString returns ANSI escape sequences to draw labels
func RenderLabelsToString(positions []LabelPosition) string {
	if len(positions) == 0 {
		return ""
	}

	var output strings.Builder

	// Save cursor position
	output.WriteString("\033[s")

	for _, pos := range positions {
		// Move to position
		output.WriteString(fmt.Sprintf("\033[%d;%dH", pos.ViewportY, pos.ViewportX))

		if pos.IsFrom {
			// This is the FROM node in connection mode
			output.WriteString("\033[32;1mFROM\033[0m") // Green "FROM"
		} else {
			// Regular jump label - single character in yellow
			output.WriteString(fmt.Sprintf("\033[33;1m%c\033[0m", pos.Label))
		}
	}

	// Restore cursor position
	output.WriteString("\033[u")

	return output.String()
}

// GetLabelAtViewport returns what label (if any) should appear at the given viewport position
func GetLabelAtViewport(positions []LabelPosition, x, y int) (rune, bool) {
	for _, pos := range positions {
		if pos.ViewportX == x && pos.ViewportY == y {
			return pos.Label, true
		}
	}
	return 0, false
}
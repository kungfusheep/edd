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

	// Special handling for reorder mode insertion points
	if e.jumpAction == JumpActionReorderTo && e.diagram.Type == "sequence" {
		// Calculate positions between participants
		participantY := 3 // Standard Y position for participants (accounting for header)
		if hasScrollIndicator {
			participantY = 4
		}

		// Calculate X positions for insertion points
		startX := 10 // Starting X position
		spacing := 20 // Approximate spacing between participants

		// Get actual participant positions if available
		// IMPORTANT: Collect positions in the order nodes appear in the diagram
		// to ensure insertion labels align correctly with the visual layout
		var participantXPositions []int
		participantWidth := 20 // Standard participant box width
		for _, node := range e.diagram.Nodes {
			if pos, ok := e.nodePositions[node.ID]; ok {
				// Store the center X position of each participant
				// (pos.X is the left edge, so add half the width)
				centerX := pos.X + participantWidth/2
				participantXPositions = append(participantXPositions, centerX)
			}
		}

		// Create insertion point labels
		for nodeID, label := range e.jumpLabels {
			if nodeID < 0 { // Negative IDs are insertion points
				position := (-nodeID) - 1

				var insertX int
				if len(participantXPositions) > 0 {
					if position == 0 {
						// Before first participant (to the left of its center)
						insertX = participantXPositions[0] - participantWidth/2 - 3
						if insertX < 2 {
							insertX = 2
						}
					} else if position >= len(participantXPositions) {
						// After last participant (to the right of its center)
						insertX = participantXPositions[len(participantXPositions)-1] + participantWidth/2 + 3
					} else if position > 0 && position < len(participantXPositions) {
						// Between two participants (midpoint of their centers)
						insertX = (participantXPositions[position-1] + participantXPositions[position]) / 2
					}
				} else {
					// Fallback if no positions available
					insertX = startX + position * spacing
				}

				positions = append(positions, LabelPosition{
					NodeID:    nodeID,
					Label:     label,
					ViewportX: insertX,
					ViewportY: participantY,
					IsFrom:    false,
				})
			}
		}
		return positions
	}

	// Process node labels (existing logic)
	for nodeID, label := range e.jumpLabels {
		if pos, ok := e.nodePositions[nodeID]; ok {
			viewportY := 0
			viewportX := pos.X

			// Adjust X position based on diagram type
			if e.diagram.Type == "sequence" && pos.Y < 7 {
				// For sequence diagram participants, place label inside the box at left edge
				// The box starts at pos.X, so we place the label at pos.X + 1
				viewportX = pos.X + 1
			} else if e.diagram.Type == "box" {
				// For regular box diagrams, place inside the box corner
				viewportX = pos.X + 2
			} else {
				// For sequence diagram elements below participants, use default position
				viewportX = pos.X + 1
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
package editor

import (
	"edd/core"
)

// Available node styles in cycle order
var nodeStyles = []string{"rounded", "sharp", "double", "thick"}

// Available node colors in cycle order  
var nodeColors = []string{"", "red", "green", "yellow", "blue", "magenta", "cyan"}

// cycleNodeStyle cycles through available node styles
func (e *TUIEditor) cycleNodeStyle(nodeID int) {
	// Find the node
	var node *core.Node
	for i := range e.diagram.Nodes {
		if e.diagram.Nodes[i].ID == nodeID {
			node = &e.diagram.Nodes[i]
			break
		}
	}
	
	if node == nil {
		return
	}
	
	// Save state for undo
	e.history.SaveState(e.diagram)
	
	// Initialize hints if needed
	if node.Hints == nil {
		node.Hints = make(map[string]string)
	}
	
	// Get current style
	currentStyle := node.Hints["style"]
	
	// Find current index
	currentIndex := -1
	for i, style := range nodeStyles {
		if style == currentStyle {
			currentIndex = i
			break
		}
	}
	
	// Cycle to next style
	nextIndex := (currentIndex + 1) % len(nodeStyles)
	node.Hints["style"] = nodeStyles[nextIndex]
	
	// If it's the default (first) style and no other hints, remove the hints map
	if nodeStyles[nextIndex] == nodeStyles[0] && len(node.Hints) == 1 {
		delete(node.Hints, "style")
		if len(node.Hints) == 0 {
			node.Hints = nil
		}
	}
}

// cycleNodeColor cycles through available node colors
func (e *TUIEditor) cycleNodeColor(nodeID int) {
	// Find the node
	var node *core.Node
	for i := range e.diagram.Nodes {
		if e.diagram.Nodes[i].ID == nodeID {
			node = &e.diagram.Nodes[i]
			break
		}
	}
	
	if node == nil {
		return
	}
	
	// Save state for undo
	e.history.SaveState(e.diagram)
	
	// Initialize hints if needed
	if node.Hints == nil {
		node.Hints = make(map[string]string)
	}
	
	// Get current color
	currentColor := node.Hints["color"]
	
	// Find current index
	currentIndex := 0 // Default to no color (empty string)
	for i, color := range nodeColors {
		if color == currentColor {
			currentIndex = i
			break
		}
	}
	
	// Cycle to next color
	nextIndex := (currentIndex + 1) % len(nodeColors)
	
	if nodeColors[nextIndex] == "" {
		// Remove color hint
		delete(node.Hints, "color")
		if len(node.Hints) == 0 {
			node.Hints = nil
		}
	} else {
		node.Hints["color"] = nodeColors[nextIndex]
	}
}

// getNodeStyle returns the style hint for a node
func (e *TUIEditor) getNodeStyle(nodeID int) string {
	for _, node := range e.diagram.Nodes {
		if node.ID == nodeID {
			if node.Hints != nil {
				return node.Hints["style"]
			}
			return ""
		}
	}
	return ""
}

// getNodeColor returns the color hint for a node
func (e *TUIEditor) getNodeColor(nodeID int) string {
	for _, node := range e.diagram.Nodes {
		if node.ID == nodeID {
			if node.Hints != nil {
				return node.Hints["color"]
			}
			return ""
		}
	}
	return ""
}
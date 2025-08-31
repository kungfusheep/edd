package editor

import "edd/core"

// HandleHintMenuInput processes input in hint menu mode
func (e *TUIEditor) HandleHintMenuInput(key rune) {
	// Check if we're editing a node or connection
	if e.editingHintNode >= 0 {
		e.handleNodeHintInput(key)
	} else if e.editingHintConn >= 0 {
		e.handleConnectionHintInput(key)
	} else {
		// Nothing selected, exit
		e.SetMode(ModeNormal)
	}
}

// handleNodeHintInput handles hint menu input for nodes
func (e *TUIEditor) handleNodeHintInput(key rune) {
	// Find the node
	var node *core.Node
	for i := range e.diagram.Nodes {
		if e.diagram.Nodes[i].ID == e.editingHintNode {
			node = &e.diagram.Nodes[i]
			break
		}
	}
	
	if node == nil {
		e.editingHintNode = -1
		e.SetMode(ModeNormal)
		return
	}
	
	// Initialize hints map if needed
	if node.Hints == nil {
		node.Hints = make(map[string]string)
	}
	
	isSequence := e.diagram.Type == string(core.DiagramTypeSequence)
	
	switch key {
	// Style options for nodes/boxes
	case 'a': // Rounded (default) for flowcharts, Box style for sequence
		if !isSequence {
			node.Hints["style"] = "rounded"
		} else {
			node.Hints["box-style"] = "rounded"
		}
		e.SaveHistory()
	case 'b': // Sharp for flowcharts, Sharp box for sequence
		if !isSequence {
			node.Hints["style"] = "sharp"
		} else {
			node.Hints["box-style"] = "sharp"
		}
		e.SaveHistory()
	case 'c': // Double for flowcharts, Double box for sequence
		if !isSequence {
			node.Hints["style"] = "double"
		} else {
			node.Hints["box-style"] = "double"
		}
		e.SaveHistory()
	case 'd': // Thick for flowcharts, Thick box for sequence
		if !isSequence {
			node.Hints["style"] = "thick"
		} else {
			node.Hints["box-style"] = "thick"
		}
		e.SaveHistory()
		
	// Color options
	case 'r': // Red
		node.Hints["color"] = "red"
		e.SaveHistory()
	case 'g': // Green
		node.Hints["color"] = "green"
		e.SaveHistory()
	case 'y': // Yellow
		node.Hints["color"] = "yellow"
		e.SaveHistory()
	case 'u': // Blue
		node.Hints["color"] = "blue"
		e.SaveHistory()
	case 'm': // Magenta
		node.Hints["color"] = "magenta"
		e.SaveHistory()
	case 'n': // Cyan
		node.Hints["color"] = "cyan"
		e.SaveHistory()
	case 'w': // Default (no color)
		delete(node.Hints, "color")
		e.SaveHistory()
		
	// Text style options (only for flowcharts)
	case 'o': // Toggle bold
		if !isSequence {
			if node.Hints["bold"] == "true" {
				delete(node.Hints, "bold")
			} else {
				node.Hints["bold"] = "true"
			}
			e.SaveHistory()
		}
	case 'i': // Toggle italic
		if !isSequence {
			if node.Hints["italic"] == "true" {
				delete(node.Hints, "italic")
			} else {
				node.Hints["italic"] = "true"
			}
			e.SaveHistory()
		}
	case 't': // Toggle text alignment (center/left)
		if !isSequence {
			if node.Hints["text-align"] == "center" {
				delete(node.Hints, "text-align") // Back to default (left)
			} else {
				node.Hints["text-align"] = "center"
			}
			e.SaveHistory()
		}
		
	// Shadow options (only for flowcharts)
	case 'z': // Shadow southeast
		if !isSequence {
			node.Hints["shadow"] = "southeast"
			if node.Hints["shadow-density"] == "" {
				node.Hints["shadow-density"] = "light"
			}
			e.SaveHistory()
		}
	case 'x': // No shadow
		if !isSequence {
			delete(node.Hints, "shadow")
			delete(node.Hints, "shadow-density")
			e.SaveHistory()
		}
	case 'l': // Shadow density for flowcharts only
		if !isSequence {
			if node.Hints["shadow-density"] == "light" {
				node.Hints["shadow-density"] = "medium"
			} else {
				node.Hints["shadow-density"] = "light"
			}
			e.SaveHistory()
		}
	
	// Lifeline style options (uppercase for sequence diagrams)
	case 'A': // Solid lifeline (default)
		if isSequence {
			delete(node.Hints, "lifeline-style") // Remove to use default (solid)
			e.SaveHistory()
		}
	case 'B': // Dashed lifeline
		if isSequence {
			node.Hints["lifeline-style"] = "dashed"
			e.SaveHistory()
		}
	case 'C': // Dotted lifeline
		if isSequence {
			node.Hints["lifeline-style"] = "dotted"
			e.SaveHistory()
		}
	case 'D': // Double lifeline
		if isSequence {
			node.Hints["lifeline-style"] = "double"
			e.SaveHistory()
		}
	
	// Lifeline color options (uppercase for sequence diagrams)
	case 'R': // Red lifeline
		if isSequence {
			node.Hints["lifeline-color"] = "red"
			e.SaveHistory()
		}
	case 'G': // Green lifeline
		if isSequence {
			node.Hints["lifeline-color"] = "green"
			e.SaveHistory()
		}
	case 'Y': // Yellow lifeline
		if isSequence {
			node.Hints["lifeline-color"] = "yellow"
			e.SaveHistory()
		}
	case 'U': // Blue lifeline
		if isSequence {
			node.Hints["lifeline-color"] = "blue"
			e.SaveHistory()
		}
	case 'M': // Magenta lifeline
		if isSequence {
			node.Hints["lifeline-color"] = "magenta"
			e.SaveHistory()
		}
	case 'N': // Cyan lifeline
		if isSequence {
			node.Hints["lifeline-color"] = "cyan"
			e.SaveHistory()
		}
	case 'W': // Default lifeline color (no color)
		if isSequence {
			delete(node.Hints, "lifeline-color")
			e.SaveHistory()
		}
		
	// Layout position hints (only for flowcharts)
	case '1': // Top-left
		if !isSequence {
			node.Hints["position"] = "top-left"
			e.SaveHistory()
		}
	case '2': // Top-center
		if !isSequence {
			node.Hints["position"] = "top-center"
			e.SaveHistory()
		}
	case '3': // Top-right
		if !isSequence {
			node.Hints["position"] = "top-right"
			e.SaveHistory()
		}
	case '4': // Middle-left
		if !isSequence {
			node.Hints["position"] = "middle-left"
			e.SaveHistory()
		}
	case '5': // Center
		if !isSequence {
			node.Hints["position"] = "center"
			e.SaveHistory()
		}
	case '6': // Middle-right
		if !isSequence {
			node.Hints["position"] = "middle-right"
			e.SaveHistory()
		}
	case '7': // Bottom-left
		if !isSequence {
			node.Hints["position"] = "bottom-left"
			e.SaveHistory()
		}
	case '8': // Bottom-center
		if !isSequence {
			node.Hints["position"] = "bottom-center"
			e.SaveHistory()
		}
	case '9': // Bottom-right
		if !isSequence {
			node.Hints["position"] = "bottom-right"
			e.SaveHistory()
		}
	case '0': // Clear position hint
		if !isSequence {
			delete(node.Hints, "position")
			e.SaveHistory()
		}
		
	case 27: // ESC - exit to normal mode or back to jump mode
		e.editingHintNode = -1
		if e.previousJumpAction != 0 {
			action := e.previousJumpAction
			e.previousJumpAction = 0  // Clear it
			e.startJump(action)  // Restart jump mode with the same action
		} else {
			e.SetMode(ModeNormal)
		}
	case 13, 10: // Enter - exit to normal mode
		e.editingHintNode = -1
		e.previousJumpAction = 0  // Clear the previous action
		e.SetMode(ModeNormal)
	}
}

// handleConnectionHintInput handles hint menu input for connections
func (e *TUIEditor) handleConnectionHintInput(key rune) {
	if e.editingHintConn < 0 || e.editingHintConn >= len(e.diagram.Connections) {
		// Invalid connection, exit
		e.editingHintConn = -1
		e.SetMode(ModeNormal)
		return
	}
	
	conn := &e.diagram.Connections[e.editingHintConn]
	
	// Initialize hints map if needed
	if conn.Hints == nil {
		conn.Hints = make(map[string]string)
	}
	
	isSequence := e.diagram.Type == string(core.DiagramTypeSequence)
	
	switch key {
	// Style options for connections
	case 'a': // Solid (default)
		delete(conn.Hints, "style") // Remove to use default
		e.SaveHistory()
	case 'b': // Dashed
		conn.Hints["style"] = "dashed"
		e.SaveHistory()
	case 'c': // Dotted
		conn.Hints["style"] = "dotted"
		e.SaveHistory()
	case 'd': // Double (only for flowcharts)
		if !isSequence {
			conn.Hints["style"] = "double"
			e.SaveHistory()
		}
		
	// Color options
	case 'r': // Red
		conn.Hints["color"] = "red"
		e.SaveHistory()
	case 'g': // Green
		conn.Hints["color"] = "green"
		e.SaveHistory()
	case 'y': // Yellow
		conn.Hints["color"] = "yellow"
		e.SaveHistory()
	case 'u': // Blue
		conn.Hints["color"] = "blue"
		e.SaveHistory()
	case 'm': // Magenta
		conn.Hints["color"] = "magenta"
		e.SaveHistory()
	case 'n': // Cyan
		conn.Hints["color"] = "cyan"
		e.SaveHistory()
	case 'w': // White/default
		delete(conn.Hints, "color") // Remove to use default
		e.SaveHistory()
		
	// Text style options
	case 'o': // Toggle bold
		if conn.Hints["bold"] == "true" {
			delete(conn.Hints, "bold")
		} else {
			conn.Hints["bold"] = "true"
		}
		e.SaveHistory()
	case 'i': // Toggle italic
		if conn.Hints["italic"] == "true" {
			delete(conn.Hints, "italic")
		} else {
			conn.Hints["italic"] = "true"
		}
		e.SaveHistory()
		
	// Flow direction hints (only for flowcharts)
	case 'f': // Cycle through flow directions
		if !isSequence {
			currentFlow := conn.Hints["flow"]
			switch currentFlow {
			case "right":
				conn.Hints["flow"] = "down"
			case "down":
				conn.Hints["flow"] = "left"
			case "left":
				conn.Hints["flow"] = "up"
			case "up":
				delete(conn.Hints, "flow") // Remove to go back to auto
			default:
				conn.Hints["flow"] = "right" // Start with right
			}
			e.SaveHistory()
		}
		
	case 27: // ESC - exit to normal mode or back to jump mode
		e.editingHintConn = -1
		if e.previousJumpAction != 0 {
			action := e.previousJumpAction
			e.previousJumpAction = 0  // Clear it
			e.startJump(action)  // Restart jump mode with the same action
		} else {
			e.SetMode(ModeNormal)
		}
	case 13, 10: // Enter - exit to normal mode
		e.editingHintConn = -1
		e.previousJumpAction = 0  // Clear the previous action
		e.SetMode(ModeNormal)
	}
}

// GetHintMenuDisplay returns the display string for hint menu
func (e *TUIEditor) GetHintMenuDisplay() string {
	if e.editingHintNode >= 0 {
		return e.getNodeHintMenuDisplay()
	} else if e.editingHintConn >= 0 {
		return e.getConnectionHintMenuDisplay()
	}
	return ""
}

// getNodeHintMenuDisplay returns the hint menu display for a node
func (e *TUIEditor) getNodeHintMenuDisplay() string {
	// Find the node
	var node *core.Node
	for i := range e.diagram.Nodes {
		if e.diagram.Nodes[i].ID == e.editingHintNode {
			node = &e.diagram.Nodes[i]
			break
		}
	}
	
	if node == nil {
		return ""
	}
	
	// Get current style, color, and shadow
	style := "rounded"
	if s, ok := node.Hints["style"]; ok {
		style = s
	}
	
	color := "default"
	if c, ok := node.Hints["color"]; ok {
		color = c
	}
	
	bold := "off"
	if b, ok := node.Hints["bold"]; ok && b == "true" {
		bold = "on"
	}
	
	italic := "off"
	if i, ok := node.Hints["italic"]; ok && i == "true" {
		italic = "on"
	}
	
	textAlign := "left"
	if a, ok := node.Hints["text-align"]; ok && a == "center" {
		textAlign = "center"
	}
	
	// Get node text
	nodeText := "Node"
	if len(node.Text) > 0 {
		nodeText = node.Text[0]
		if len(nodeText) > 20 {
			nodeText = nodeText[:20] + "..."
		}
	}
	
	// Different menu for sequence diagrams
	if e.diagram.Type == string(core.DiagramTypeSequence) {
		boxStyle := "rounded"
		if s, ok := node.Hints["box-style"]; ok {
			boxStyle = s
		}
		
		lifelineStyle := "solid"
		if s, ok := node.Hints["lifeline-style"]; ok {
			lifelineStyle = s
		}
		
		lifelineColor := "default"
		if c, ok := node.Hints["lifeline-color"]; ok {
			lifelineColor = c
		}
		
		return "\n" +
			"Participant: " + nodeText + " | box=" + boxStyle + "/" + color + " | lifeline=" + lifelineStyle + "/" + lifelineColor + "\n" +
			"Box: [a]Round [b]Sharp [c]Double [d]Thick | [r]Red [g]Green [y]Yellow [u]Blue [m]Magenta [n]Cyan [w]Clear\n" +
			"Line: [A]Solid [B]Dash [C]Dot [D]Double | [R]Red [G]Green [Y]Yellow [U]Blue [M]Magenta [N]Cyan [W]Clear\n" +
			"[ESC]Back [Enter]Done"
	}
	
	// Full menu for flowcharts
	return "\n" +
		"Node: " + nodeText + " | style=" + style + ", color=" + color + "\n" +
		"Style: [a]Rounded [b]Sharp [c]Double [d]Thick | " +
		"Color: [r]Red [g]Green [y]Yellow [u]Blue [m]Magenta [n]Cyan [w]Clear\n" +
		"Text: [o]Bold(" + bold + ") [i]Italic(" + italic + ") [t]Center(" + textAlign + ") | " +
		"Shadow: [z]Add [x]Remove [l]Density\n" +
		"Position: [1-9]Grid [0]Auto | [ESC]Back [Enter]Done"
}

// getConnectionHintMenuDisplay returns the hint menu display for a connection
func (e *TUIEditor) getConnectionHintMenuDisplay() string {
	if e.editingHintConn < 0 || e.editingHintConn >= len(e.diagram.Connections) {
		return ""
	}
	
	conn := &e.diagram.Connections[e.editingHintConn]
	
	// Get current style, color, and bold
	style := "solid"
	if s, ok := conn.Hints["style"]; ok {
		style = s
	}
	
	color := "default"
	if c, ok := conn.Hints["color"]; ok {
		color = c
	}
	
	bold := "off"
	if b, ok := conn.Hints["bold"]; ok && b == "true" {
		bold = "on"
	}
	
	italic := "off"
	if i, ok := conn.Hints["italic"]; ok && i == "true" {
		italic = "on"
	}
	
	flow := "auto"
	if f, ok := conn.Hints["flow"]; ok {
		flow = f
	}
	
	// Find connection info
	var fromText, toText string
	for _, node := range e.diagram.Nodes {
		if node.ID == conn.From && len(node.Text) > 0 {
			fromText = node.Text[0]
			if len(fromText) > 10 {
				fromText = fromText[:10] + "..."
			}
		}
		if node.ID == conn.To && len(node.Text) > 0 {
			toText = node.Text[0]
			if len(toText) > 10 {
				toText = toText[:10] + "..."
			}
		}
	}
	
	// Different menu for sequence diagrams
	if e.diagram.Type == string(core.DiagramTypeSequence) {
		return "\n" +
			"Message: " + fromText + " → " + toText + " | style=" + style + ", color=" + color + "\n" +
			"Style: [a]Solid [b]Dashed [c]Dotted | Color: [r]Red [g]Green [y]Yellow [u]Blue [m]Magenta [n]Cyan [w]Clear\n" +
			"Text: [o]Bold(" + bold + ") [i]Italic(" + italic + ") | [ESC]Back [Enter]Done"
	}
	
	// Full menu for flowcharts
	return "\n" +
		"Connection: " + fromText + " → " + toText + " | style=" + style + ", color=" + color + "\n" +
		"Style: [a]Solid [b]Dashed [c]Dotted [d]Double | " +
		"Color: [r]Red [g]Green [y]Yellow [u]Blue [m]Magenta [n]Cyan [w]Clear\n" +
		"Options: [o]Bold(" + bold + ") [i]Italic(" + italic + ") [f]Flow(" + flow + ") | [ESC]Back [Enter]Done"
}
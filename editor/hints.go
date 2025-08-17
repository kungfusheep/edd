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
	
	switch key {
	// Style options for nodes
	case 'a': // Rounded (default)
		node.Hints["style"] = "rounded"
		e.history.SaveState(e.diagram)
	case 'b': // Sharp
		node.Hints["style"] = "sharp"
		e.history.SaveState(e.diagram)
	case 'c': // Double
		node.Hints["style"] = "double"
		e.history.SaveState(e.diagram)
	case 'd': // Thick
		node.Hints["style"] = "thick"
		e.history.SaveState(e.diagram)
		
	// Color options
	case 'r': // Red
		node.Hints["color"] = "red"
		e.history.SaveState(e.diagram)
	case 'g': // Green
		node.Hints["color"] = "green"
		e.history.SaveState(e.diagram)
	case 'y': // Yellow
		node.Hints["color"] = "yellow"
		e.history.SaveState(e.diagram)
	case 'u': // Blue
		node.Hints["color"] = "blue"
		e.history.SaveState(e.diagram)
	case 'm': // Magenta
		node.Hints["color"] = "magenta"
		e.history.SaveState(e.diagram)
	case 'n': // Cyan
		node.Hints["color"] = "cyan"
		e.history.SaveState(e.diagram)
	case 'w': // Default (no color)
		delete(node.Hints, "color")
		e.history.SaveState(e.diagram)
		
	// Text style options
	case 'o': // Toggle bold
		if node.Hints["bold"] == "true" {
			delete(node.Hints, "bold")
		} else {
			node.Hints["bold"] = "true"
		}
		e.history.SaveState(e.diagram)
	case 'i': // Toggle italic
		if node.Hints["italic"] == "true" {
			delete(node.Hints, "italic")
		} else {
			node.Hints["italic"] = "true"
		}
		e.history.SaveState(e.diagram)
		
	// Shadow options
	case 'z': // Shadow southeast
		node.Hints["shadow"] = "southeast"
		if node.Hints["shadow-density"] == "" {
			node.Hints["shadow-density"] = "light"
		}
		e.history.SaveState(e.diagram)
	case 'x': // No shadow
		delete(node.Hints, "shadow")
		delete(node.Hints, "shadow-density")
		e.history.SaveState(e.diagram)
	case 'l': // Toggle shadow density (light/medium)
		if node.Hints["shadow-density"] == "light" {
			node.Hints["shadow-density"] = "medium"
		} else {
			node.Hints["shadow-density"] = "light"
		}
		e.history.SaveState(e.diagram)
		
	// Layout position hints
	case '1': // Top-left
		node.Hints["position"] = "top-left"
		e.history.SaveState(e.diagram)
	case '2': // Top-center
		node.Hints["position"] = "top-center"
		e.history.SaveState(e.diagram)
	case '3': // Top-right
		node.Hints["position"] = "top-right"
		e.history.SaveState(e.diagram)
	case '4': // Middle-left
		node.Hints["position"] = "middle-left"
		e.history.SaveState(e.diagram)
	case '5': // Center
		node.Hints["position"] = "center"
		e.history.SaveState(e.diagram)
	case '6': // Middle-right
		node.Hints["position"] = "middle-right"
		e.history.SaveState(e.diagram)
	case '7': // Bottom-left
		node.Hints["position"] = "bottom-left"
		e.history.SaveState(e.diagram)
	case '8': // Bottom-center
		node.Hints["position"] = "bottom-center"
		e.history.SaveState(e.diagram)
	case '9': // Bottom-right
		node.Hints["position"] = "bottom-right"
		e.history.SaveState(e.diagram)
	case '0': // Clear position hint
		delete(node.Hints, "position")
		e.history.SaveState(e.diagram)
		
	case 27: // ESC - go back to jump menu
		e.editingHintNode = -1
		// Re-enter jump mode for hints
		e.startJump(JumpActionHint)
	case 13: // Enter - exit to normal mode
		e.editingHintNode = -1
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
	
	switch key {
	// Style options for connections
	case 'a': // Solid (default)
		delete(conn.Hints, "style") // Remove to use default
		e.history.SaveState(e.diagram)
	case 'b': // Dashed
		conn.Hints["style"] = "dashed"
		e.history.SaveState(e.diagram)
	case 'c': // Dotted
		conn.Hints["style"] = "dotted"
		e.history.SaveState(e.diagram)
	case 'd': // Double
		conn.Hints["style"] = "double"
		e.history.SaveState(e.diagram)
		
	// Color options
	case 'r': // Red
		conn.Hints["color"] = "red"
		e.history.SaveState(e.diagram)
	case 'g': // Green
		conn.Hints["color"] = "green"
		e.history.SaveState(e.diagram)
	case 'y': // Yellow
		conn.Hints["color"] = "yellow"
		e.history.SaveState(e.diagram)
	case 'u': // Blue
		conn.Hints["color"] = "blue"
		e.history.SaveState(e.diagram)
	case 'm': // Magenta
		conn.Hints["color"] = "magenta"
		e.history.SaveState(e.diagram)
	case 'n': // Cyan
		conn.Hints["color"] = "cyan"
		e.history.SaveState(e.diagram)
	case 'w': // White/default
		delete(conn.Hints, "color") // Remove to use default
		e.history.SaveState(e.diagram)
		
	// Text style options
	case 'o': // Toggle bold
		if conn.Hints["bold"] == "true" {
			delete(conn.Hints, "bold")
		} else {
			conn.Hints["bold"] = "true"
		}
		e.history.SaveState(e.diagram)
	case 'i': // Toggle italic
		if conn.Hints["italic"] == "true" {
			delete(conn.Hints, "italic")
		} else {
			conn.Hints["italic"] = "true"
		}
		e.history.SaveState(e.diagram)
		
	// Flow direction hints
	case 'f': // Cycle through flow directions
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
		e.history.SaveState(e.diagram)
		
	case 27: // ESC - go back to jump menu
		e.editingHintConn = -1
		// Re-enter jump mode for hints
		e.startJump(JumpActionHint)
	case 13: // Enter - exit to normal mode
		e.editingHintConn = -1
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
	
	shadow := "none"
	if s, ok := node.Hints["shadow"]; ok {
		shadow = s
	}
	
	shadowDensity := "light"
	if d, ok := node.Hints["shadow-density"]; ok {
		shadowDensity = d
	}
	
	bold := "off"
	if b, ok := node.Hints["bold"]; ok && b == "true" {
		bold = "on"
	}
	
	italic := "off"
	if i, ok := node.Hints["italic"]; ok && i == "true" {
		italic = "on"
	}
	
	position := "auto"
	if p, ok := node.Hints["position"]; ok {
		position = p
	}
	
	// Get node text
	nodeText := "Node"
	if len(node.Text) > 0 {
		nodeText = node.Text[0]
		if len(nodeText) > 20 {
			nodeText = nodeText[:20] + "..."
		}
	}
	
	return "\n" +
		"Node: " + nodeText + "\n" +
		"Current: style=" + style + ", color=" + color + ", bold=" + bold + ", italic=" + italic + "\n" +
		"         shadow=" + shadow + " (" + shadowDensity + "), position=" + position + "\n\n" +
		"Style Options:\n" +
		"  [a] Rounded ╭──╮\n" +
		"  [b] Sharp   ┌──┐\n" +
		"  [c] Double  ╔══╗\n" +
		"  [d] Thick   ┏━━┓\n\n" +
		"Color Options:\n" +
		"  [r] Red    [g] Green   [y] Yellow\n" +
		"  [u] Blue   [m] Magenta [n] Cyan\n" +
		"  [w] Default\n\n" +
		"Text Options:\n" +
		"  [o] Toggle bold text\n" +
		"  [i] Toggle italic text\n\n" +
		"Shadow Options:\n" +
		"  [z] Add shadow ░░  [x] Remove shadow\n" +
		"  [l] Toggle density (light/medium)\n\n" +
		"Layout Position Hints:\n" +
		"  [1] Top-left     [2] Top-center    [3] Top-right\n" +
		"  [4] Middle-left  [5] Center        [6] Middle-right\n" +
		"  [7] Bottom-left  [8] Bottom-center [9] Bottom-right\n" +
		"  [0] Auto (clear position hint)\n\n" +
		"[ESC] Back to selection  [Enter] Exit to normal mode"
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
	
	return "\n" +
		"Connection: " + fromText + " → " + toText + "\n" +
		"Current: style=" + style + ", color=" + color + ", bold=" + bold + ", italic=" + italic + "\n" +
		"         flow=" + flow + "\n\n" +
		"Style Options:\n" +
		"  [a] Solid ────\n" +
		"  [b] Dashed ╌╌╌╌\n" +
		"  [c] Dotted ····\n" +
		"  [d] Double ════\n\n" +
		"Color Options:\n" +
		"  [r] Red    [g] Green   [y] Yellow\n" +
		"  [u] Blue   [m] Magenta [n] Cyan\n" +
		"  [w] Default\n\n" +
		"Text Options:\n" +
		"  [o] Toggle bold lines\n" +
		"  [i] Toggle italic lines\n\n" +
		"Flow Direction:\n" +
		"  [f] Cycle flow direction (→ ↓ ← ↑ auto)\n" +
		"      Current: " + flow + "\n\n" +
		"[ESC] Back to selection  [Enter] Exit to normal mode"
}
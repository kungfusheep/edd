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
		
	// Shadow options
	case 'z': // Shadow southeast
		node.Hints["shadow"] = "southeast"
		if node.Hints["shadow-density"] == "" {
			node.Hints["shadow-density"] = "light"
		}
		e.history.SaveState(e.diagram)
	case 'x': // Shadow south
		node.Hints["shadow"] = "south"
		if node.Hints["shadow-density"] == "" {
			node.Hints["shadow-density"] = "light"
		}
		e.history.SaveState(e.diagram)
	case 'v': // No shadow
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
		
	case 27, 13: // ESC or Enter - exit hint menu
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
		
	case 27, 13: // ESC or Enter - exit hint menu
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
		"Current: style=" + style + ", color=" + color + ", shadow=" + shadow + " (" + shadowDensity + ")\n\n" +
		"Style Options:\n" +
		"  [a] Rounded ╭──╮\n" +
		"  [b] Sharp   ┌──┐\n" +
		"  [c] Double  ╔══╗\n" +
		"  [d] Thick   ┏━━┓\n\n" +
		"Color Options:\n" +
		"  [r] Red    [g] Green   [y] Yellow\n" +
		"  [u] Blue   [m] Magenta [n] Cyan\n" +
		"  [w] Default\n\n" +
		"Shadow Options:\n" +
		"  [z] Southeast ░░  [x] South ░░  [v] None\n" +
		"  [l] Toggle density (light/medium)\n\n" +
		"[ESC/Enter] Exit"
}

// getConnectionHintMenuDisplay returns the hint menu display for a connection
func (e *TUIEditor) getConnectionHintMenuDisplay() string {
	if e.editingHintConn < 0 || e.editingHintConn >= len(e.diagram.Connections) {
		return ""
	}
	
	conn := &e.diagram.Connections[e.editingHintConn]
	
	// Get current style and color
	style := "solid"
	if s, ok := conn.Hints["style"]; ok {
		style = s
	}
	
	color := "default"
	if c, ok := conn.Hints["color"]; ok {
		color = c
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
		"Current: style=" + style + ", color=" + color + "\n\n" +
		"Style Options:\n" +
		"  [a] Solid ────\n" +
		"  [b] Dashed ╌╌╌╌\n" +
		"  [c] Dotted ····\n" +
		"  [d] Double ════\n\n" +
		"Color Options:\n" +
		"  [r] Red    [g] Green   [y] Yellow\n" +
		"  [u] Blue   [m] Magenta [n] Cyan\n" +
		"  [w] Default\n\n" +
		"[ESC/Enter] Exit"
}
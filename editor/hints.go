package editor

// HandleHintMenuInput processes input in hint menu mode
func (e *TUIEditor) HandleHintMenuInput(key rune) {
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
	// Style options
	case 'a': // Solid (default)
		delete(conn.Hints, "style") // Remove to use default
		e.SaveHistory()
	case 'b': // Dashed
		conn.Hints["style"] = "dashed"
		e.SaveHistory()
	case 'c': // Dotted
		conn.Hints["style"] = "dotted"
		e.SaveHistory()
	case 'd': // Double
		conn.Hints["style"] = "double"
		e.SaveHistory()
		
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
		
	case 27, 13: // ESC or Enter - exit hint menu
		e.editingHintConn = -1
		e.SetMode(ModeNormal)
	}
}

// GetHintMenuDisplay returns the display string for hint menu
func (e *TUIEditor) GetHintMenuDisplay() string {
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
		"  [b] Dashed - - -\n" +
		"  [c] Dotted ·····\n" +
		"  [d] Double ════\n\n" +
		"Color Options:\n" +
		"  [r] Red    [g] Green   [y] Yellow\n" +
		"  [u] Blue   [m] Magenta [n] Cyan\n" +
		"  [w] White/Default\n\n" +
		"[ESC/Enter] Exit"
}
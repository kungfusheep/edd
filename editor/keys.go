package editor

import (
	"edd/core"
	"strings"
	"unicode"
)

// handleNormalKey processes keys in normal mode
func (e *TUIEditor) handleNormalKey(key rune) bool {
	switch key {
	case 'q', 3: // q or Ctrl+C to quit
		return true
		
	case 'a', 'A': // Add new node (A is same as a since we're already in INSERT mode with continuation)
		e.SetMode(ModeInsert)
		nodeID := e.AddNode([]string{""})
		e.selected = nodeID
		
	case 'c': // Connect nodes (single)
		if len(e.diagram.Nodes) >= 2 {
			e.continuousConnect = false
			e.startJump(JumpActionConnectFrom)
		}
		
	case 'C': // Connect nodes (continuous)
		if len(e.diagram.Nodes) >= 2 {
			e.continuousConnect = true
			e.startJump(JumpActionConnectFrom)
		}
		
	case 'd': // Delete node/connection (single)
		if len(e.diagram.Nodes) > 0 || len(e.diagram.Connections) > 0 {
			e.continuousDelete = false
			e.startJump(JumpActionDelete)
		}
		
	case 'D': // Delete node/connection (continuous)
		if len(e.diagram.Nodes) > 0 || len(e.diagram.Connections) > 0 {
			e.continuousDelete = true
			e.startJump(JumpActionDelete)
		}
		
	case 'e': // Edit node
		if len(e.diagram.Nodes) > 0 {
			e.startJump(JumpActionEdit)
		}
		
	case '?', 'h': // Help
		// TODO: Show help
		
	case ':': // Command mode
		e.SetMode(ModeCommand)
		e.commandBuffer = []rune{}  // Start with empty, : is shown in prompt
	}
	
	return false
}

// handleTextKey processes keys in text input modes (Insert/Edit)
func (e *TUIEditor) handleTextKey(key rune) bool {
	switch key {
	case 27: // ESC - save and return to normal mode
		e.commitText()
		e.SetMode(ModeNormal)
		
	case 127, 8: // Backspace
		if e.cursorPos > 0 {
			e.textBuffer = append(
				e.textBuffer[:e.cursorPos-1],
				e.textBuffer[e.cursorPos:]...,
			)
			e.cursorPos--
		}
		
	case 13, 10: // Enter - commit text
		// Save the mode before committing (in case commit changes it)
		wasInsertMode := e.mode == ModeInsert
		
		e.commitText()
		
		// In INSERT mode, immediately start adding another node
		if wasInsertMode {
			// Create a new node and continue in insert mode
			nodeID := e.AddNode([]string{""})
			e.selected = nodeID
			e.textBuffer = []rune{}
			e.cursorPos = 0
			// Stay in INSERT mode (don't call SetMode as it clears the buffer)
			// e.mode is already ModeInsert
		} else {
			// In EDIT mode, return to normal
			e.SetMode(ModeNormal)
		}
		
	default:
		// Insert printable characters
		if unicode.IsPrint(key) {
			e.textBuffer = append(
				e.textBuffer[:e.cursorPos],
				append([]rune{key}, e.textBuffer[e.cursorPos:]...)...,
			)
			e.cursorPos++
		}
	}
	
	return false
}

// handleCommandKey processes keys in command mode
func (e *TUIEditor) handleCommandKey(key rune) bool {
	switch key {
	case 27: // ESC - cancel command
		e.SetMode(ModeNormal)
		
	case 127, 8: // Backspace
		if len(e.commandBuffer) > 0 {
			e.commandBuffer = e.commandBuffer[:len(e.commandBuffer)-1]
		}
		
	case 13, 10: // Enter - execute command
		e.executeCommand(string(e.commandBuffer))
		e.SetMode(ModeNormal)
		
	default:
		// Add to command buffer
		if unicode.IsPrint(key) {
			e.commandBuffer = append(e.commandBuffer, key)
		}
	}
	
	return false
}

// handleJumpKey processes keys when jump labels are active
func (e *TUIEditor) handleJumpKey(key rune) bool {
	// ESC cancels jump
	if key == 27 {
		e.clearJumpLabels()
		e.continuousConnect = false // Exit continuous connect mode
		e.continuousDelete = false  // Exit continuous delete mode
		e.SetMode(ModeNormal)
		return false
	}
	
	// Look for matching node jump label
	for nodeID, label := range e.jumpLabels {
		if label == key {
			// Found match - execute jump action
			e.executeJumpAction(nodeID)
			return false
		}
	}
	
	// Look for matching connection jump label (in delete or edit mode)
	if e.jumpAction == JumpActionDelete || e.jumpAction == JumpActionEdit {
		for connIndex, label := range e.connectionLabels {
			if label == key {
				if e.jumpAction == JumpActionDelete {
					// Delete the connection
					e.DeleteConnection(connIndex)
					
					// If in continuous delete mode, start another delete
					if e.continuousDelete && (len(e.diagram.Nodes) > 0 || len(e.diagram.Connections) > 0) {
						e.clearJumpLabels()
						e.startJump(JumpActionDelete)
					} else {
						// Normal mode - exit to normal
						e.continuousDelete = false
						e.clearJumpLabels()
						e.SetMode(ModeNormal)
					}
				} else if e.jumpAction == JumpActionEdit {
					// Edit the connection label
					e.StartEditingConnection(connIndex)
				}
				return false
			}
		}
	}
	
	// No match - cancel jump
	e.clearJumpLabels()
	e.SetMode(ModeNormal)
	return false
}

// commitText saves the current text buffer to the selected node or connection
func (e *TUIEditor) commitText() {
	text := string(e.textBuffer)
	
	// Check if we're editing a connection
	if e.selectedConnection >= 0 {
		// Connection labels can be empty (to clear them)
		e.UpdateConnectionLabel(e.selectedConnection, text)
		e.selectedConnection = -1
		return
	}
	
	// Otherwise we're editing a node
	if e.selected < 0 {
		return
	}
	
	text = strings.TrimSpace(text)
	// In INSERT mode, we allow empty text (user might just press Enter to create empty nodes)
	if text == "" && e.mode != ModeInsert {
		return
	}
	
	// Update the node text
	e.UpdateNodeText(e.selected, []string{text})
}

// executeCommand executes a command mode command
func (e *TUIEditor) executeCommand(cmd string) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}
	
	switch parts[0] {
	case "q", "quit":
		// TODO: Implement quit with save check
		
	case "w", "write", "save":
		// TODO: Implement save
		
	case "wq":
		// TODO: Save and quit
		
	case "load", "open":
		// TODO: Load diagram
		
	case "new":
		// Clear diagram
		e.diagram = &core.Diagram{}
		e.selected = -1
	}
}

// executeJumpAction executes the pending action after jump selection
func (e *TUIEditor) executeJumpAction(nodeID int) {
	switch e.jumpAction {
	case JumpActionSelect:
		e.selected = nodeID
		e.clearJumpLabels()
		e.SetMode(ModeNormal)
		
	case JumpActionEdit:
		e.selected = nodeID
		e.clearJumpLabels()
		e.SetMode(ModeEdit)
		return // Don't reset to normal mode
		
	case JumpActionDelete:
		e.DeleteNode(nodeID)
		if e.selected == nodeID {
			e.selected = -1
		}
		
		// If in continuous delete mode, start another delete
		if e.continuousDelete && (len(e.diagram.Nodes) > 0 || len(e.diagram.Connections) > 0) {
			e.clearJumpLabels()
			e.startJump(JumpActionDelete)
		} else {
			// Normal mode - exit to normal
			e.continuousDelete = false
			e.clearJumpLabels()
			e.SetMode(ModeNormal)
		}
		
	case JumpActionConnectFrom:
		e.selected = nodeID
		// Start second jump for target
		e.startJump(JumpActionConnectTo)
		return // Don't clear jump labels yet
		
	case JumpActionConnectTo:
		if e.selected >= 0 && e.selected != nodeID {
			e.AddConnection(e.selected, nodeID, "")
		}
		
		// If in continuous connect mode, start another connection
		if e.continuousConnect {
			e.selected = -1
			e.clearJumpLabels()
			// Start another connection
			e.startJump(JumpActionConnectFrom)
		} else {
			// Normal mode - exit to normal
			e.selected = -1
			e.clearJumpLabels()
			e.SetMode(ModeNormal)
		}
	}
}
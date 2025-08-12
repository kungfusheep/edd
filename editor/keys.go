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
		
	case 'a': // Add new node
		e.SetMode(ModeInsert)
		nodeID := e.AddNode([]string{""})
		e.selected = nodeID
		
	case 'c': // Connect nodes
		if len(e.diagram.Nodes) >= 2 {
			e.startJump(JumpActionConnectFrom)
		}
		
	case 'd': // Delete node
		if len(e.diagram.Nodes) > 0 {
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
		e.textBuffer = []rune{':'}
		e.cursorPos = 1
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
		e.commitText()
		e.SetMode(ModeNormal)
		
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
		if e.cursorPos > 1 { // Keep the ':'
			e.textBuffer = append(
				e.textBuffer[:e.cursorPos-1],
				e.textBuffer[e.cursorPos:]...,
			)
			e.cursorPos--
		}
		
	case 13, 10: // Enter - execute command
		e.executeCommand(string(e.textBuffer[1:])) // Skip the ':'
		e.SetMode(ModeNormal)
		
	default:
		// Add to command buffer
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

// handleJumpKey processes keys when jump labels are active
func (e *TUIEditor) handleJumpKey(key rune) bool {
	// ESC cancels jump
	if key == 27 {
		e.clearJumpLabels()
		e.SetMode(ModeNormal)
		return false
	}
	
	// Look for matching jump label
	for nodeID, label := range e.jumpLabels {
		if label == key {
			// Found match - execute jump action
			e.executeJumpAction(nodeID)
			return false
		}
	}
	
	// No match - cancel jump
	e.clearJumpLabels()
	e.SetMode(ModeNormal)
	return false
}

// commitText saves the current text buffer to the selected node
func (e *TUIEditor) commitText() {
	if e.selected < 0 {
		return
	}
	
	text := strings.TrimSpace(string(e.textBuffer))
	if text == "" {
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
		
	case JumpActionEdit:
		e.selected = nodeID
		e.SetMode(ModeEdit)
		
	case JumpActionDelete:
		e.DeleteNode(nodeID)
		if e.selected == nodeID {
			e.selected = -1
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
		e.selected = -1
	}
	
	// Clear jump state and return to normal mode
	e.clearJumpLabels()
	e.SetMode(ModeNormal)
}
package editor

import (
	"edd/core"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"
)

// handleNormalKey processes keys in normal mode
func (e *TUIEditor) handleNormalKey(key rune) bool {
	switch key {
	case 'q', 3: // q or Ctrl+C to quit
		return true
		
	case 27: // ESC - if we have a previous jump action, restart that jump mode
		if e.previousJumpAction != 0 {
			action := e.previousJumpAction
			e.previousJumpAction = 0  // Clear it
			e.startJump(action)  // Restart jump mode with the same action
		}
		
	case 'a', 'A': // Add new node (A is same as a since we're already in INSERT mode with continuation)
		e.previousJumpAction = 0  // Clear any previous jump action
		e.SetMode(ModeInsert)
		nodeID := e.AddNode([]string{""})
		e.selected = nodeID
		
	case 'c': // Connect nodes (single)
		if len(e.diagram.Nodes) >= 2 {
			e.continuousConnect = false
			e.selected = -1 // Clear any previous selection
			e.startJump(JumpActionConnectFrom)
		}
		
	case 'C': // Connect nodes (continuous)
		if len(e.diagram.Nodes) >= 2 {
			e.continuousConnect = true
			e.selected = -1 // Clear any previous selection
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
		
	case 'E': // Edit in external editor
		// This will be handled by the main loop since it needs file system access
		return false
		
	case 'H': // Edit hints for nodes and connections
		if len(e.diagram.Nodes) > 0 || len(e.diagram.Connections) > 0 {
			e.startJump(JumpActionHint)
		}
		
	case 'j': // Toggle JSON view
		e.SetMode(ModeJSON)
		e.jsonScrollOffset = 0  // Reset scroll when entering JSON mode
		
	case 'u': // Undo
		e.Undo()
		
	case 18: // Ctrl+R for redo
		e.Redo()
		
	case '?', 'h': // Help
		e.SetMode(ModeHelp)
		
	case ':': // Command mode
		e.SetMode(ModeCommand)
		e.commandBuffer = []rune{}  // Start with empty, : is shown in prompt
	}
	
	return false
}

// handleTextKey processes keys in text input modes (Insert/Edit)
func (e *TUIEditor) handleTextKey(key rune) bool {
	switch key {
	case 27: // ESC - save and return to normal mode (or jump mode if we came from there)
		e.commitText()
		if e.previousJumpAction != 0 {
			action := e.previousJumpAction
			e.previousJumpAction = 0  // Clear it
			e.startJump(action)  // Restart jump mode with the same action
		} else {
			e.SetMode(ModeNormal)
		}
		
	case 127, 8: // Backspace
		if e.cursorPos > 0 {
			e.textBuffer = append(
				e.textBuffer[:e.cursorPos-1],
				e.textBuffer[e.cursorPos:]...,
			)
			e.cursorPos--
			e.updateCursorPosition()
		}
		
	case 14: // Ctrl+N - insert newline for multi-line editing
		e.insertNewline()
	
	case 23: // Ctrl+W - delete word backward
		e.deleteWordBackward()
	
	case 21: // Ctrl+U - delete to beginning of line
		e.deleteToBeginningOfLine()
	
	case 11: // Ctrl+K - delete to end of line
		e.deleteToEndOfLine()
	
	case 1: // Ctrl+A - move to beginning of line
		e.moveCursorToBeginningOfLine()
	
	case 5: // Ctrl+E - move to end of line
		e.moveCursorToEndOfLine()
	
	case 6: // Ctrl+F - move forward one character
		e.moveCursorForward()
	
	case 2: // Ctrl+B - move backward one character
		e.moveCursorBackward()
	
	case 16: // Ctrl+P - move up one line (previous)
		e.moveCursorUp()
	
	case 22: // Ctrl+V - move down one line (since Ctrl+N is for newline)
		e.moveCursorDown()
		
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
			e.updateCursorPosition()
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
		e.previousJumpAction = 0    // Clear the previous action since we're canceling
		e.selected = -1            // Clear selected node
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
	
	// Look for matching connection jump label (in delete, edit, or hint mode)
	if e.jumpAction == JumpActionDelete || e.jumpAction == JumpActionEdit || e.jumpAction == JumpActionHint {
		// Log to file
		if f, err := os.OpenFile("/tmp/edd_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			fmt.Fprintf(f, "\n[%s] Looking for key '%c' in connection labels: %v\n", time.Now().Format("15:04:05"), key, e.connectionLabels)
			f.Close()
		}
		for connIndex, label := range e.connectionLabels {
			if label == key {
				// Log to file
				if f, err := os.OpenFile("/tmp/edd_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
					fmt.Fprintf(f, "Found match! Connection index %d has label '%c'\n", connIndex, label)
					f.Close()
				}
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
					e.previousJumpAction = e.jumpAction  // Save the action for ESC handling
					e.StartEditingConnection(connIndex)
				} else if e.jumpAction == JumpActionHint {
					// Enter hint menu for this connection
					e.previousJumpAction = e.jumpAction  // Save the action for ESC handling
					e.editingHintConn = connIndex
					// Log to file
					if f, err := os.OpenFile("/tmp/edd_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
						fmt.Fprintf(f, "Selected connection index %d for hint editing (pressed '%c')\n", connIndex, key)
						f.Close()
					}
					e.clearJumpLabels()
					e.SetMode(ModeHintMenu)
				}
				return false
			}
		}
	}
	
	// No match - handle based on current state
	if e.continuousConnect {
		// In continuous connect mode - just ignore invalid keys
		// User can still press a valid label or ESC to cancel
		return false
	}
	if e.continuousDelete {
		// In continuous delete mode - just ignore invalid keys
		return false
	}
	
	// For single-action modes, cancel jump on invalid key
	e.clearJumpLabels()
	e.SetMode(ModeNormal)
	return false
}

// commitText saves the current text buffer to the selected node or connection
func (e *TUIEditor) commitText() {
	// Check if we're editing a connection
	if e.selectedConnection >= 0 {
		// Connection labels are single line only
		text := strings.TrimSpace(string(e.textBuffer))
		// Connection labels can be empty (to clear them)
		e.UpdateConnectionLabel(e.selectedConnection, text)
		e.selectedConnection = -1
		return
	}
	
	// Otherwise we're editing a node - support multi-line
	if e.selected < 0 {
		return
	}
	
	// Get text as lines for multi-line support
	lines := e.GetTextAsLines()
	
	// Trim empty lines at the end
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	
	// In INSERT mode, we allow empty text (user might just press Enter to create empty nodes)
	if len(lines) == 0 && e.mode != ModeInsert {
		return
	}
	
	// If completely empty, use a single empty line
	if len(lines) == 0 {
		lines = []string{""}
	}
	
	// Update the node text with multiple lines
	e.UpdateNodeText(e.selected, lines)
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
		
	case "type":
		// Change diagram type
		if len(parts) > 1 {
			switch parts[1] {
			case "sequence", "seq":
				e.diagram.Type = string(core.DiagramTypeSequence)
				e.SaveHistory()
			case "flowchart", "flow", "":
				e.diagram.Type = string(core.DiagramTypeFlowchart)  // Empty means flowchart
				e.SaveHistory()
			default:
				// Unknown type, ignore
			}
		}
	}
}

// executeJumpAction executes the pending action after jump selection
func (e *TUIEditor) executeJumpAction(nodeID int) {
	// Save the action type before executing
	e.previousJumpAction = e.jumpAction
	
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
		if e.selected >= 0 {
			e.AddConnection(e.selected, nodeID, "")
		}
		
		// If in continuous connect mode, behavior depends on diagram type
		if e.continuousConnect {
			if e.diagram.Type == string(core.DiagramTypeSequence) {
				// Sequence diagram: chain connections (TO becomes next FROM)
				e.selected = nodeID
				// Jump directly to selecting the next TO node
				e.startJump(JumpActionConnectTo)
			} else {
				// Flowchart: start fresh connection (select new FROM)
				e.selected = -1
				e.startJump(JumpActionConnectFrom)
			}
		} else {
			// Normal mode - exit to normal
			e.selected = -1
			e.clearJumpLabels()
			e.SetMode(ModeNormal)
		}
		
	case JumpActionHint:
		// Enter hint menu for this node
		e.editingHintNode = nodeID
		e.clearJumpLabels()
		e.SetMode(ModeHintMenu)
	}
}

// handleHelpKey processes keys in help mode
func (e *TUIEditor) handleHelpKey(key rune) bool {
	// Any key exits help mode
	e.SetMode(ModeNormal)
	return false
}

// handleJSONKey processes keys in JSON view mode
func (e *TUIEditor) handleJSONKey(key rune) bool {
	switch key {
	case 27, 'q', 'j': // ESC, q, or j to return to diagram view
		e.SetMode(ModeNormal)
		
	case 'E': // Edit in external editor (also works from JSON view)
		// This will be handled by the main loop
		return false
		
	case 'k', 'K': // vim-style up
		e.ScrollJSON(-1)
		
	case 'J': // vim-style down (capital J since j exits)
		e.ScrollJSON(1)
		
	case 'u', 21: // Page up (Ctrl+U in vim)
		e.ScrollJSON(-(e.height / 2))
		
	case 'd', 4: // Page down (Ctrl+D in vim)
		e.ScrollJSON(e.height / 2)
		
	case 'g': // Go to top
		e.jsonScrollOffset = 0
		
	case 'G': // Go to bottom
		e.jsonScrollOffset = 999999 // Will be clamped in renderJSON
	}
	
	return false
}
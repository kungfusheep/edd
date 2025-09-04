package editor

import (
	"fmt"
	"os"
)

// Jump label characters in order of preference (home row first)
const jumpChars = "asdfghjklqwertyuiopzxcvbnm"

// startJump initiates jump mode with labels
func (e *TUIEditor) startJump(action JumpAction) {
	e.jumpAction = action
	e.assignJumpLabels()
	e.SetMode(ModeJump)
}

// assignJumpLabels assigns single-character labels to nodes and connections
func (e *TUIEditor) assignJumpLabels() {
	e.jumpLabels = make(map[int]rune)
	e.connectionLabels = make(map[int]rune)
	
	labelIndex := 0
	
	// Assign labels to nodes
	for _, node := range e.diagram.Nodes {
		if labelIndex < len(jumpChars) {
			e.jumpLabels[node.ID] = rune(jumpChars[labelIndex])
			labelIndex++
		} else {
			break
		}
	}
	
	// If in delete, edit, or hint mode, also assign labels to connections
	if e.jumpAction == JumpActionDelete || e.jumpAction == JumpActionEdit || e.jumpAction == JumpActionHint {
		// Use index-based iteration to ensure consistent ordering
		for i := 0; i < len(e.diagram.Connections); i++ {
			if labelIndex < len(jumpChars) {
				e.connectionLabels[i] = rune(jumpChars[labelIndex])
				// Log to file
				if f, err := os.OpenFile("/tmp/edd_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
					fmt.Fprintf(f, "Jump: Assigned label '%c' to connection index %d\n", jumpChars[labelIndex], i)
					f.Close()
				}
				labelIndex++
			} else {
				break
			}
		}
	}
}

// getJumpLabel returns the jump label for a node ID
func (e *TUIEditor) getJumpLabel(nodeID int) string {
	if label, ok := e.jumpLabels[nodeID]; ok {
		return string(label)
	}
	return ""
}

// clearJumpLabels clears all jump labels
func (e *TUIEditor) clearJumpLabels() {
	e.jumpLabels = make(map[int]rune)
	e.connectionLabels = make(map[int]rune)
	e.jumpAction = JumpActionSelect
}
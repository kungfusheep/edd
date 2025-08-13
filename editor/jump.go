package editor

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
	
	// If in delete mode, also assign labels to connections
	if e.jumpAction == JumpActionDelete {
		// Use index-based iteration to ensure consistent ordering
		for i := 0; i < len(e.diagram.Connections); i++ {
			if labelIndex < len(jumpChars) {
				e.connectionLabels[i] = rune(jumpChars[labelIndex])
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
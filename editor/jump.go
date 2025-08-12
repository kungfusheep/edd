package editor

// Jump label characters in order of preference (home row first)
const jumpChars = "asdfghjklqwertyuiopzxcvbnm"

// startJump initiates jump mode with labels
func (e *TUIEditor) startJump(action JumpAction) {
	e.jumpAction = action
	e.assignJumpLabels()
	e.SetMode(ModeJump)
}

// assignJumpLabels assigns single-character labels to nodes
func (e *TUIEditor) assignJumpLabels() {
	e.jumpLabels = make(map[int]rune)
	
	// Assign labels to nodes
	for i, node := range e.diagram.Nodes {
		if i < len(jumpChars) {
			e.jumpLabels[node.ID] = rune(jumpChars[i])
		} else {
			// If we have more nodes than single chars, use double chars
			// For now, just skip extra nodes
			break
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
	e.jumpAction = JumpActionSelect
}
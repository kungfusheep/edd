package editor

import "edd/diagram"

// Exported methods for testing

func (e *TUIEditor) SetNodePositions(positions map[int]diagram.Point) {
	e.nodePositions = positions
}

func (e *TUIEditor) SetDiagramScrollOffset(offset int) {
	e.diagramScrollOffset = offset
}

func (e *TUIEditor) StartJump(action JumpAction) {
	e.startJump(action)
}

func (e *TUIEditor) AssignJumpLabels() {
	e.assignJumpLabels()
}

func (e *TUIEditor) IsNodeVisible(nodeID int) bool {
	return e.isNodeVisible(nodeID)
}

func (e *TUIEditor) GetTerminalWidth() int {
	return e.width
}
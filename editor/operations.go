package editor

import (
	"edd/core"
	"unicode"
)

// GetMode returns the current mode
func (e *TUIEditor) GetMode() Mode {
	return e.mode
}

// GetEddFrame returns Ed's current animation frame
func (e *TUIEditor) GetEddFrame() string {
	return e.edd.GetFrame(e.mode)
}

// GetJumpLabels returns the current jump labels
func (e *TUIEditor) GetJumpLabels() map[int]rune {
	return e.jumpLabels
}

// GetConnectionLabels returns the current connection jump labels
func (e *TUIEditor) GetConnectionLabels() map[int]rune {
	return e.connectionLabels
}

// GetJumpAction returns the current jump action
func (e *TUIEditor) GetJumpAction() JumpAction {
	return e.jumpAction
}

// GetSelectedNode returns the currently selected node ID
func (e *TUIEditor) GetSelectedNode() int {
	return e.selected
}

// GetNodePositions returns the last rendered node positions
func (e *TUIEditor) GetNodePositions() map[int]core.Point {
	return e.nodePositions
}

// GetConnectionPaths returns the last rendered connection paths
func (e *TUIEditor) GetConnectionPaths() map[int]core.Path {
	return e.connectionPaths
}

// GetTextBuffer returns the current text buffer (for display purposes)
func (e *TUIEditor) GetTextBuffer() []rune {
	return e.textBuffer
}

// StartAddNode begins adding a new node
func (e *TUIEditor) StartAddNode() {
	e.SetMode(ModeInsert)
	nodeID := e.AddNode([]string{""})
	e.selected = nodeID
	e.textBuffer = []rune{}
	e.cursorPos = 0
}

// StartConnect begins connection mode
func (e *TUIEditor) StartConnect() {
	if len(e.diagram.Nodes) >= 2 {
		e.startJump(JumpActionConnectFrom)
	}
}

// StartDelete begins delete mode
func (e *TUIEditor) StartDelete() {
	if len(e.diagram.Nodes) > 0 {
		e.startJump(JumpActionDelete)
	}
}

// StartEdit begins edit mode
func (e *TUIEditor) StartEdit() {
	if len(e.diagram.Nodes) > 0 {
		e.startJump(JumpActionEdit)
	}
}

// StartCommand enters command mode
func (e *TUIEditor) StartCommand() {
	e.SetMode(ModeCommand)
	e.commandBuffer = []rune{}
}

// HandleTextInput processes text input in insert/edit modes
func (e *TUIEditor) HandleTextInput(key rune) {
	switch key {
	case 27: // ESC
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
	case 13, 10: // Enter
		e.commitText()
		e.SetMode(ModeNormal)
	default:
		if unicode.IsPrint(key) {
			e.textBuffer = append(
				e.textBuffer[:e.cursorPos],
				append([]rune{key}, e.textBuffer[e.cursorPos:]...)...,
			)
			e.cursorPos++
		}
	}
}

// HandleJumpInput processes jump label selection for both nodes and connections
func (e *TUIEditor) HandleJumpInput(key rune) {
	// This is the public method that should handle both nodes and connections
	// It delegates to the internal handleJumpKey which has the full logic
	e.handleJumpKey(key)
}

// HandleCommandInput processes command mode input
func (e *TUIEditor) HandleCommandInput(key rune) {
	switch key {
	case 27: // ESC
		e.SetMode(ModeNormal)
	case 127, 8: // Backspace
		if len(e.commandBuffer) > 0 {
			e.commandBuffer = e.commandBuffer[:len(e.commandBuffer)-1]
		}
	default:
		if unicode.IsPrint(key) {
			e.commandBuffer = append(e.commandBuffer, key)
		}
	}
}

// GetCommand returns the current command buffer
func (e *TUIEditor) GetCommand() string {
	return string(e.commandBuffer)
}

// ClearCommand clears the command buffer
func (e *TUIEditor) ClearCommand() {
	e.commandBuffer = []rune{}
}

// AnimateEd advances Ed's animation
func (e *TUIEditor) AnimateEd() {
	e.edd.NextFrame()
}

// GetNodeCount returns the number of nodes
func (e *TUIEditor) GetNodeCount() int {
	return len(e.diagram.Nodes)
}

// GetConnectionCount returns the number of connections
func (e *TUIEditor) GetConnectionCount() int {
	return len(e.diagram.Connections)
}
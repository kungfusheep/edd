package editor

import (
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
	e.textBuffer = []rune{}
	e.cursorPos = 0
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

// HandleJumpInput processes jump label selection
func (e *TUIEditor) HandleJumpInput(key rune) {
	if key == 27 { // ESC cancels
		e.clearJumpLabels()
		e.SetMode(ModeNormal)
		return
	}
	
	// Look for matching label
	for nodeID, label := range e.jumpLabels {
		if label == key {
			e.executeJumpAction(nodeID)
			return
		}
	}
	
	// No match - cancel
	e.clearJumpLabels()
	e.SetMode(ModeNormal)
}

// HandleCommandInput processes command mode input
func (e *TUIEditor) HandleCommandInput(key rune) {
	switch key {
	case 27: // ESC
		e.SetMode(ModeNormal)
	case 127, 8: // Backspace
		if e.cursorPos > 0 {
			e.textBuffer = append(
				e.textBuffer[:e.cursorPos-1],
				e.textBuffer[e.cursorPos:]...,
			)
			e.cursorPos--
		}
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

// GetCommand returns the current command buffer
func (e *TUIEditor) GetCommand() string {
	return string(e.textBuffer)
}

// ClearCommand clears the command buffer
func (e *TUIEditor) ClearCommand() {
	e.textBuffer = []rune{}
	e.cursorPos = 0
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

// GetSelectedNode returns the currently selected node ID
func (e *TUIEditor) GetSelectedNode() int {
	return e.selected
}
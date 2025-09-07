package editor

import (
	"edd/diagram"
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
func (e *TUIEditor) GetNodePositions() map[int]diagram.Point {
	return e.nodePositions
}

// GetConnectionPaths returns the last rendered connection paths
func (e *TUIEditor) GetConnectionPaths() map[int]diagram.Path {
	return e.connectionPaths
}

// GetTextBuffer returns the current text buffer (for display purposes)
func (e *TUIEditor) GetTextBuffer() []rune {
	return e.textBuffer
}

// IsContinuousConnect returns whether we're in continuous connection mode
func (e *TUIEditor) IsContinuousConnect() bool {
	return e.continuousConnect
}

// IsContinuousDelete returns whether we're in continuous delete mode
func (e *TUIEditor) IsContinuousDelete() bool {
	return e.continuousDelete
}

// StartAddNode begins adding a new node
func (e *TUIEditor) StartAddNode() {
	e.SetMode(ModeInsert)
	nodeID := e.AddNode([]string{""})
	e.selected = nodeID
	e.textBuffer = []rune{}
	e.cursorPos = 0
}

// StartConnect begins connection mode (single connection)
func (e *TUIEditor) StartConnect() {
	if len(e.diagram.Nodes) >= 2 {
		e.continuousConnect = false
		e.startJump(JumpActionConnectFrom)
	}
}

// StartContinuousConnect begins continuous connection mode (multiple connections)
func (e *TUIEditor) StartContinuousConnect() {
	if len(e.diagram.Nodes) >= 2 {
		e.continuousConnect = true
		e.startJump(JumpActionConnectFrom)
	}
}

// StartDelete begins delete mode (single deletion)
func (e *TUIEditor) StartDelete() {
	if len(e.diagram.Nodes) > 0 || len(e.diagram.Connections) > 0 {
		e.continuousDelete = false
		e.startJump(JumpActionDelete)
	}
}

// StartContinuousDelete begins continuous delete mode (multiple deletions)
func (e *TUIEditor) StartContinuousDelete() {
	if len(e.diagram.Nodes) > 0 || len(e.diagram.Connections) > 0 {
		e.continuousDelete = true
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

// StartHintEdit starts hint editing mode for nodes and connections
func (e *TUIEditor) StartHintEdit() {
	if len(e.diagram.Nodes) > 0 || len(e.diagram.Connections) > 0 {
		e.startJump(JumpActionHint)
	}
}

// HandleTextInput processes text input in insert/edit modes
func (e *TUIEditor) HandleTextInput(key rune) {
	// Delegate to the actual text handler
	e.handleTextKey(key)
}

// ToggleDiagramType switches between sequence and box diagram types
func (e *TUIEditor) ToggleDiagramType() {
	e.history.SaveState(e.diagram)
	currentType := e.diagram.Type
	if currentType == "" {
		currentType = "box"
	}
	
	if currentType == "sequence" {
		e.diagram.Type = "box"
	} else {
		e.diagram.Type = "sequence"
	}
}

// HandleJumpInput processes jump label selection for both nodes and connections
func (e *TUIEditor) HandleJumpInput(key rune) {
	// This is the public method that should handle both nodes and connections
	// It delegates to the internal handleJumpKey which has the full logic
	e.handleJumpKey(key)
}

// HandleJSONInput processes JSON view mode input
func (e *TUIEditor) HandleJSONInput(key rune) {
	// Delegate to the internal handler
	e.handleJSONKey(key)
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
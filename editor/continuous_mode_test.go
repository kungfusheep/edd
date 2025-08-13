package editor

import (
	"edd/core"
	"testing"
)

func TestContinuousInsertMode(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(&core.Diagram{})

	// Start insert mode
	tui.handleNormalKey('a')
	if tui.GetMode() != ModeInsert {
		t.Errorf("Expected ModeInsert, got %v", tui.GetMode())
	}

	// Type first node text
	for _, ch := range "Node1" {
		tui.HandleTextInput(ch)
	}

	// Press Enter - should commit and stay in INSERT mode
	tui.HandleTextInput(13)

	// Should still be in INSERT mode
	if tui.GetMode() != ModeInsert {
		t.Errorf("Expected to stay in ModeInsert after Enter, got %v", tui.GetMode())
	}

	// Should have two nodes (the initial empty one when entering INSERT, plus the one we just committed)
	if len(tui.diagram.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(tui.diagram.Nodes))
	}

	// Text buffer should be empty for new node
	if len(tui.textBuffer) != 0 {
		t.Errorf("Expected empty text buffer for new node, got '%s'", string(tui.textBuffer))
	}

	// Type second node text
	for _, ch := range "Node2" {
		tui.HandleTextInput(ch)
	}

	// Press Enter again
	tui.HandleTextInput(13)

	// Should still be in INSERT mode
	if tui.GetMode() != ModeInsert {
		t.Errorf("Expected to stay in ModeInsert, got %v", tui.GetMode())
	}

	// Should have three nodes now (initial + 2 created)
	if len(tui.diagram.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(tui.diagram.Nodes))
	}

	// Press ESC to exit INSERT mode
	tui.HandleTextInput(27)

	// Should be in NORMAL mode now
	if tui.GetMode() != ModeNormal {
		t.Errorf("Expected ModeNormal after ESC, got %v", tui.GetMode())
	}

	// Should have three nodes total (initial empty + Node1 + Node2)
	// The third empty node from the last Enter is not saved when we press ESC
	if len(tui.diagram.Nodes) != 3 {
		t.Errorf("Expected 3 nodes total, got %d", len(tui.diagram.Nodes))
	}
}

func TestContinuousConnectionMode(t *testing.T) {
	diagram := &core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"A"}},
			{ID: 2, Text: []string{"B"}},
			{ID: 3, Text: []string{"C"}},
			{ID: 4, Text: []string{"D"}},
		},
	}

	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(diagram)

	// Start continuous connection mode with 'C'
	tui.handleNormalKey('C')
	
	if !tui.IsContinuousConnect() {
		t.Error("Expected continuous connect mode to be enabled")
	}

	// Should be in jump mode
	if tui.GetMode() != ModeJump {
		t.Errorf("Expected ModeJump, got %v", tui.GetMode())
	}

	// Get node labels
	labels := tui.GetJumpLabels()
	
	// Select first node (node 1)
	var label1 rune
	for id, label := range labels {
		if id == 1 {
			label1 = label
			break
		}
	}
	tui.handleJumpKey(label1)

	// Should be in jump mode for selecting target
	if tui.GetMode() != ModeJump {
		t.Errorf("Expected ModeJump for target selection, got %v", tui.GetMode())
	}

	// Get labels again for target selection
	labels = tui.GetJumpLabels()
	
	// Select second node (node 2)
	var label2 rune
	for id, label := range labels {
		if id == 2 {
			label2 = label
			break
		}
	}
	tui.handleJumpKey(label2)

	// Should have one connection
	if len(tui.diagram.Connections) != 1 {
		t.Errorf("Expected 1 connection, got %d", len(tui.diagram.Connections))
	}

	// Should STILL be in jump mode for another connection (continuous mode)
	if tui.GetMode() != ModeJump {
		t.Errorf("Expected to stay in ModeJump for continuous connections, got %v", tui.GetMode())
	}

	// Select another source
	labels = tui.GetJumpLabels()
	var label3 rune
	for id, label := range labels {
		if id == 3 {
			label3 = label
			break
		}
	}
	tui.handleJumpKey(label3)

	// Select another target
	labels = tui.GetJumpLabels()
	var label4 rune
	for id, label := range labels {
		if id == 4 {
			label4 = label
			break
		}
	}
	tui.handleJumpKey(label4)

	// Should have two connections now
	if len(tui.diagram.Connections) != 2 {
		t.Errorf("Expected 2 connections, got %d", len(tui.diagram.Connections))
	}

	// Press ESC to exit continuous mode
	tui.handleJumpKey(27)

	// Should be in normal mode
	if tui.GetMode() != ModeNormal {
		t.Errorf("Expected ModeNormal after ESC, got %v", tui.GetMode())
	}

	// Should no longer be in continuous mode
	if tui.IsContinuousConnect() {
		t.Error("Expected continuous connect mode to be disabled after ESC")
	}
}

func TestSingleConnectionMode(t *testing.T) {
	diagram := &core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"A"}},
			{ID: 2, Text: []string{"B"}},
		},
	}

	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(diagram)

	// Start single connection mode with 'c'
	tui.handleNormalKey('c')
	
	if tui.IsContinuousConnect() {
		t.Error("Expected continuous connect mode to be disabled for 'c'")
	}

	// Select nodes and make connection
	labels := tui.GetJumpLabels()
	var label1, label2 rune
	for id, label := range labels {
		if id == 1 {
			label1 = label
		} else if id == 2 {
			label2 = label
		}
	}
	
	tui.handleJumpKey(label1)
	tui.handleJumpKey(label2)

	// Should have one connection
	if len(tui.diagram.Connections) != 1 {
		t.Errorf("Expected 1 connection, got %d", len(tui.diagram.Connections))
	}

	// Should return to normal mode (not continuous)
	if tui.GetMode() != ModeNormal {
		t.Errorf("Expected ModeNormal after single connection, got %v", tui.GetMode())
	}
}
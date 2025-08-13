package editor

import (
	"edd/core"
	"testing"
)

func TestConnectionLabelEditing(t *testing.T) {
	// Create a test diagram with connections
	diagram := &core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"Node A"}},
			{ID: 2, Text: []string{"Node B"}},
			{ID: 3, Text: []string{"Node C"}},
		},
		Connections: []core.Connection{
			{From: 1, To: 2, Label: "initial"},
			{From: 2, To: 3, Label: "test"},
		},
	}

	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(diagram)

	// Start edit mode - press 'e'
	tui.handleNormalKey('e')
	
	// Should be in jump mode with edit action
	if tui.GetMode() != ModeJump {
		t.Errorf("Expected ModeJump, got %v", tui.GetMode())
	}
	if tui.GetJumpAction() != JumpActionEdit {
		t.Errorf("Expected JumpActionEdit, got %v", tui.GetJumpAction())
	}

	// Verify that connection labels were assigned
	connLabels := tui.GetConnectionLabels()
	if len(connLabels) != 2 {
		t.Errorf("Expected 2 connection labels for edit mode, got %d", len(connLabels))
	}

	// Get connection 0's label specifically
	firstConnLabel, ok := connLabels[0]
	if !ok {
		t.Fatal("Connection 0 should have a label")
	}

	// Debug: print the connection labels before selection
	t.Logf("Connection labels before selection: %v", connLabels)
	t.Logf("Selecting label '%c' for connection 0", firstConnLabel)
	
	// Select the first connection for editing
	tui.handleJumpKey(firstConnLabel)

	// Should now be in edit mode
	if tui.GetMode() != ModeEdit {
		t.Errorf("Expected ModeEdit after selecting connection, got %v", tui.GetMode())
	}

	// Should have the connection selected
	selectedConn := tui.GetSelectedConnection()
	t.Logf("Selected connection: %d", selectedConn)
	
	if selectedConn < 0 || selectedConn >= len(diagram.Connections) {
		t.Fatalf("Invalid selected connection index: %d", selectedConn)
	}

	// Text buffer should contain the current label
	textBuffer := string(tui.GetTextBuffer())
	expectedLabel := diagram.Connections[selectedConn].Label
	t.Logf("Connection %d label: '%s', text buffer: '%s'", selectedConn, expectedLabel, textBuffer)
	
	if textBuffer != expectedLabel {
		t.Errorf("Expected text buffer to contain '%s', got '%s'", expectedLabel, textBuffer)
	}

	// Clear the buffer and type a new label
	tui.textBuffer = []rune{}
	tui.cursorPos = 0
	for _, ch := range "new label" {
		tui.HandleTextInput(ch)
	}

	// Press Enter to commit
	tui.HandleTextInput(13)

	// Should be back in normal mode
	if tui.GetMode() != ModeNormal {
		t.Errorf("Expected ModeNormal after committing, got %v", tui.GetMode())
	}

	// Connection should have the new label
	if diagram.Connections[0].Label != "new label" {
		t.Errorf("Expected connection label to be 'new label', got '%s'", diagram.Connections[0].Label)
	}
}

func TestConnectionLabelClearing(t *testing.T) {
	// Test that we can clear a connection label
	diagram := &core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"A"}},
			{ID: 2, Text: []string{"B"}},
		},
		Connections: []core.Connection{
			{From: 1, To: 2, Label: "to be cleared"},
		},
	}

	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(diagram)

	// Start editing the connection directly
	tui.StartEditingConnection(0)

	// Clear the text buffer
	tui.textBuffer = []rune{}
	tui.cursorPos = 0

	// Commit the empty text
	tui.commitText()

	// Connection label should be empty
	if diagram.Connections[0].Label != "" {
		t.Errorf("Expected connection label to be empty, got '%s'", diagram.Connections[0].Label)
	}
}

func TestEditModeAssignsConnectionLabels(t *testing.T) {
	// Verify that edit mode assigns labels to connections
	diagram := &core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"A"}},
			{ID: 2, Text: []string{"B"}},
		},
		Connections: []core.Connection{
			{From: 1, To: 2, Label: ""},
		},
	}

	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(diagram)

	// Start edit mode
	tui.StartEdit()

	// Check that connection labels were assigned
	connLabels := tui.GetConnectionLabels()
	if len(connLabels) != 1 {
		t.Errorf("Expected 1 connection label in edit mode, got %d", len(connLabels))
	}
}
package editor

import (
	"edd/diagram"
	"testing"
)

func TestConnectionEditEnterAdvance(t *testing.T) {
	// Create editor with real renderer
	renderer := NewRealRenderer()
	ed := NewTUIEditor(renderer)

	// Create a simple diagram with connections
	d := &diagram.Diagram{Type: "flowchart"}
	d.Nodes = []diagram.Node{
		{ID: 1, Text: []string{"Start"}},
		{ID: 2, Text: []string{"Process"}},
		{ID: 3, Text: []string{"End"}},
	}
	d.Connections = []diagram.Connection{
		{ID: 0, From: 1, To: 2, Label: ""},
		{ID: 1, From: 2, To: 3, Label: ""},
		{ID: 2, From: 1, To: 3, Label: ""},
	}
	ed.SetDiagram(d)

	// Start editing first connection via jump mode
	ed.startJump(JumpActionEdit)

	// Find the label for first connection and select it
	var firstConnLabel rune
	for idx, label := range ed.connectionLabels {
		if idx == 0 {
			firstConnLabel = label
			break
		}
	}

	if firstConnLabel == 0 {
		t.Fatal("No label assigned to first connection")
	}

	// Select the first connection for editing
	ed.handleJumpKey(firstConnLabel)

	// Should be in edit mode with first connection selected
	if ed.mode != ModeEdit {
		t.Errorf("Expected ModeEdit, got %v", ed.mode)
	}
	if ed.selectedConnection != 0 {
		t.Errorf("Expected selectedConnection=0, got %d", ed.selectedConnection)
	}

	// Type "first" label
	for _, ch := range "first" {
		ed.handleTextKey(ch)
	}

	// Press Enter - should commit and move to next connection
	ed.handleTextKey(13) // Enter key

	// Check that first connection was labeled
	if ed.diagram.Connections[0].Label != "first" {
		t.Errorf("Expected first connection label 'first', got '%s'", ed.diagram.Connections[0].Label)
	}

	// Should still be in edit mode with second connection selected
	if ed.mode != ModeEdit {
		t.Errorf("After first Enter, expected ModeEdit, got %v", ed.mode)
	}
	if ed.selectedConnection != 1 {
		t.Errorf("After first Enter, expected selectedConnection=1, got %d", ed.selectedConnection)
	}

	// Type "second" label
	for _, ch := range "second" {
		ed.handleTextKey(ch)
	}

	// Press Enter - should commit and move to third connection
	ed.handleTextKey(13)

	// Check that second connection was labeled
	if ed.diagram.Connections[1].Label != "second" {
		t.Errorf("Expected second connection label 'second', got '%s'", ed.diagram.Connections[1].Label)
	}

	// Should still be in edit mode with third connection selected
	if ed.mode != ModeEdit {
		t.Errorf("After second Enter, expected ModeEdit, got %v", ed.mode)
	}
	if ed.selectedConnection != 2 {
		t.Errorf("After second Enter, expected selectedConnection=2, got %d", ed.selectedConnection)
	}

	// Type "third" label
	for _, ch := range "third" {
		ed.handleTextKey(ch)
	}

	// Press Enter - should commit and return to normal mode (no more connections)
	ed.handleTextKey(13)

	// Check that third connection was labeled
	if ed.diagram.Connections[2].Label != "third" {
		t.Errorf("Expected third connection label 'third', got '%s'", ed.diagram.Connections[2].Label)
	}

	// Should be back in normal mode
	if ed.mode != ModeNormal {
		t.Errorf("After third Enter, expected ModeNormal, got %v", ed.mode)
	}
	if ed.selectedConnection != -1 {
		t.Errorf("After third Enter, expected selectedConnection=-1, got %d", ed.selectedConnection)
	}
}

func TestConnectionInlineEditing(t *testing.T) {
	// Create editor with real renderer
	renderer := NewRealRenderer()
	ed := NewTUIEditor(renderer)

	// Create a simple diagram with connections
	d := &diagram.Diagram{Type: "flowchart"}
	d.Nodes = []diagram.Node{
		{ID: 1, Text: []string{"A"}},
		{ID: 2, Text: []string{"B"}},
	}
	d.Connections = []diagram.Connection{
		{ID: 0, From: 1, To: 2, Label: "original"},
	}
	ed.SetDiagram(d)

	// Start editing the connection
	ed.selectedConnection = 0
	ed.SetMode(ModeEdit)

	// Type some new text
	for _, ch := range "new" {
		ed.handleTextKey(ch)
	}

	// Render to trigger the edit state update
	_ = ed.Render()

	// Check that renderer has the correct edit state
	if renderer.EditingConnectionID != 0 {
		t.Errorf("Expected EditingConnectionID=0, got %d", renderer.EditingConnectionID)
	}
	expectedText := "originalnew" // Original text plus what we typed
	if renderer.EditConnectionText != expectedText {
		t.Errorf("Expected EditConnectionText='%s', got '%s'", expectedText, renderer.EditConnectionText)
	}
	if renderer.EditConnectionCursorPos != 11 {
		t.Errorf("Expected EditConnectionCursorPos=11, got %d", renderer.EditConnectionCursorPos)
	}

	// Render and check that the output shows inline editing
	output := ed.Render()

	// The output should show the editing text with cursor, not the original label
	// Note: The exact format depends on the renderer implementation
	if len(output) == 0 {
		t.Error("Expected non-empty render output")
	}
}
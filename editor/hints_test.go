package editor

import (
	"edd/core"
	"encoding/json"
	"testing"
)

func TestConnectionHints(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	
	// Add two nodes
	id1 := tui.AddNode([]string{"Node 1"})
	id2 := tui.AddNode([]string{"Node 2"})
	
	// Add a connection
	tui.AddConnection(id1, id2, "test")
	
	// Get the connection
	diagram := tui.GetDiagram()
	if len(diagram.Connections) != 1 {
		t.Fatal("Expected 1 connection")
	}
	
	// Add hints manually (simulating hint menu)
	conn := &diagram.Connections[0]
	conn.Hints = map[string]string{
		"style": "dashed",
		"color": "blue",
	}
	
	// Test that hints are preserved in JSON
	data, err := json.Marshal(diagram)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	
	// Parse back
	var loaded core.Diagram
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	
	// Check hints were preserved
	if len(loaded.Connections) != 1 {
		t.Fatal("Connection lost after round-trip")
	}
	
	hints := loaded.Connections[0].Hints
	if hints == nil {
		t.Fatal("Hints were not preserved")
	}
	
	if hints["style"] != "dashed" {
		t.Errorf("Style hint lost: got %v", hints["style"])
	}
	
	if hints["color"] != "blue" {
		t.Errorf("Color hint lost: got %v", hints["color"])
	}
}

func TestHintMenuInput(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	
	// Add nodes and connection
	id1 := tui.AddNode([]string{"A"})
	id2 := tui.AddNode([]string{"B"})
	tui.AddConnection(id1, id2, "")
	
	// Simulate entering hint menu for connection 0
	tui.editingHintConn = 0
	tui.SetMode(ModeHintMenu)
	
	// Test setting style to dashed
	tui.HandleHintMenuInput('b')
	conn := tui.GetDiagram().Connections[0]
	if conn.Hints["style"] != "dashed" {
		t.Errorf("Expected style=dashed, got %v", conn.Hints["style"])
	}
	
	// Test setting color to red
	tui.HandleHintMenuInput('r')
	if conn.Hints["color"] != "red" {
		t.Errorf("Expected color=red, got %v", conn.Hints["color"])
	}
	
	// Test ESC exits mode
	tui.HandleHintMenuInput(27)
	if tui.GetMode() != ModeNormal {
		t.Error("ESC should return to normal mode")
	}
}
package editor

import (
	"edd/core"
	"testing"
)

func TestHistoryManager(t *testing.T) {
	h := NewHistoryManager(5) // Small capacity for testing
	
	// Create test diagrams
	diagrams := make([]*core.Diagram, 3)
	for i := 0; i < 3; i++ {
		diagrams[i] = &core.Diagram{
			Nodes: []core.Node{
				{ID: i + 1, Text: []string{"Node"}},
			},
		}
	}
	
	// Save states
	for _, d := range diagrams {
		if err := h.SaveState(d); err != nil {
			t.Fatalf("Failed to save state: %v", err)
		}
	}
	
	// Check position
	current, total := h.Stats()
	if total != 3 {
		t.Errorf("Expected 3 states, got %d", total)
	}
	if current != 3 {
		t.Errorf("Expected current position 3, got %d", current)
	}
	
	// Test undo
	if !h.CanUndo() {
		t.Error("Should be able to undo")
	}
	
	undone, err := h.Undo()
	if err != nil {
		t.Fatalf("Undo failed: %v", err)
	}
	if len(undone.Nodes) != 1 || undone.Nodes[0].ID != 2 {
		t.Error("Undo returned wrong state")
	}
	
	// Test redo
	if !h.CanRedo() {
		t.Error("Should be able to redo after undo")
	}
	
	redone, err := h.Redo()
	if err != nil {
		t.Fatalf("Redo failed: %v", err)
	}
	if len(redone.Nodes) != 1 || redone.Nodes[0].ID != 3 {
		t.Error("Redo returned wrong state")
	}
}

func TestRingBufferOverflow(t *testing.T) {
	h := NewHistoryManager(3) // Very small capacity
	
	// Add more states than capacity
	for i := 0; i < 5; i++ {
		d := &core.Diagram{
			Nodes: []core.Node{
				{ID: i + 1, Text: []string{"Node"}},
			},
		}
		h.SaveState(d)
	}
	
	// Should only have last 3 states
	_, total := h.Stats()
	if total != 3 {
		t.Errorf("Expected 3 states after overflow, got %d", total)
	}
	
	// Oldest should be ID=3 (IDs 1 and 2 were overwritten)
	h.Undo() // From 5 to 4
	state, err := h.Undo() // From 4 to 3
	if err != nil {
		t.Fatalf("Undo failed: %v", err)
	}
	if state == nil || len(state.Nodes) == 0 {
		t.Fatal("Undo returned nil or empty state")
	}
	if state.Nodes[0].ID != 3 {
		t.Errorf("Expected oldest state to have ID=3, got %d", state.Nodes[0].ID)
	}
	
	// Shouldn't be able to undo further
	if h.CanUndo() {
		t.Error("Should not be able to undo past buffer start")
	}
}

func TestUndoRedoIntegration(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	
	// Add some nodes
	id1 := tui.AddNode([]string{"First"})
	id2 := tui.AddNode([]string{"Second"})
	_ = tui.AddNode([]string{"Third"}) // id3 not used in test
	
	// Add a connection
	tui.AddConnection(id1, id2, "link")
	
	// Should have 5 states (initial + 4 modifications)
	_, total := tui.GetHistoryStats()
	if total < 5 {
		t.Errorf("Expected at least 5 history states, got %d", total)
	}
	
	// Undo the connection
	tui.Undo()
	if len(tui.GetDiagram().Connections) != 0 {
		t.Error("Connection should be undone")
	}
	
	// Undo adding third node
	tui.Undo()
	if len(tui.GetDiagram().Nodes) != 2 {
		t.Error("Third node should be undone")
	}
	
	// Redo adding third node
	tui.Redo()
	if len(tui.GetDiagram().Nodes) != 3 {
		t.Error("Third node should be redone")
	}
	
	// Add a new node (should clear redo history)
	tui.AddNode([]string{"Fourth"})
	
	// Shouldn't be able to redo anymore
	beforeNodes := len(tui.GetDiagram().Nodes)
	tui.Redo() // Should do nothing
	afterNodes := len(tui.GetDiagram().Nodes)
	if beforeNodes != afterNodes {
		t.Error("Redo should not work after new modification")
	}
}

func TestUndoDeleteNode(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	
	// Add nodes and connection
	id1 := tui.AddNode([]string{"Node1"})
	id2 := tui.AddNode([]string{"Node2"})
	tui.AddConnection(id1, id2, "conn")
	
	// Delete a node (should also delete connection)
	tui.DeleteNode(id1)
	
	if len(tui.GetDiagram().Nodes) != 1 {
		t.Error("Node should be deleted")
	}
	if len(tui.GetDiagram().Connections) != 0 {
		t.Error("Connection should be deleted with node")
	}
	
	// Undo the deletion
	tui.Undo()
	
	if len(tui.GetDiagram().Nodes) != 2 {
		t.Error("Node should be restored")
	}
	if len(tui.GetDiagram().Connections) != 1 {
		t.Error("Connection should be restored")
	}
}
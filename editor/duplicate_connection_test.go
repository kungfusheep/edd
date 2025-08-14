package editor

import (
	"testing"
)

func TestPreventDuplicateConnections(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	
	// Add two nodes
	id1 := tui.AddNode([]string{"Node 1"})
	id2 := tui.AddNode([]string{"Node 2"})
	
	// Add first connection
	tui.AddConnection(id1, id2, "first")
	
	// Get initial connection count
	initialCount := len(tui.GetDiagram().Connections)
	if initialCount != 1 {
		t.Errorf("Expected 1 connection, got %d", initialCount)
	}
	
	// Try to add duplicate connection (same direction)
	tui.AddConnection(id1, id2, "duplicate")
	
	// Count should remain the same
	afterCount := len(tui.GetDiagram().Connections)
	if afterCount != 1 {
		t.Errorf("Duplicate connection was added! Expected 1, got %d", afterCount)
	}
	
	// Add connection in opposite direction (should be allowed)
	tui.AddConnection(id2, id1, "reverse")
	
	// Count should now be 2
	finalCount := len(tui.GetDiagram().Connections)
	if finalCount != 2 {
		t.Errorf("Reverse connection was not added! Expected 2, got %d", finalCount)
	}
	
	// Verify both connections exist
	conns := tui.GetDiagram().Connections
	if conns[0].Label != "first" {
		t.Errorf("Original connection label changed! Expected 'first', got '%s'", conns[0].Label)
	}
	if conns[1].Label != "reverse" {
		t.Errorf("Reverse connection has wrong label! Expected 'reverse', got '%s'", conns[1].Label)
	}
}

func TestAllowMultipleUniqueConnections(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	
	// Add three nodes
	id1 := tui.AddNode([]string{"Node 1"})
	id2 := tui.AddNode([]string{"Node 2"})
	id3 := tui.AddNode([]string{"Node 3"})
	
	// Add unique connections
	tui.AddConnection(id1, id2, "1-2")
	tui.AddConnection(id2, id3, "2-3")
	tui.AddConnection(id1, id3, "1-3")
	
	// Should have 3 connections
	count := len(tui.GetDiagram().Connections)
	if count != 3 {
		t.Errorf("Expected 3 unique connections, got %d", count)
	}
}
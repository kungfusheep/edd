package editor

import (
	"edd/diagram"
	"testing"
)

func TestConnectionDeletion(t *testing.T) {
	// Create a test diagram
	d := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"Node A"}},
			{ID: 2, Text: []string{"Node B"}},
			{ID: 3, Text: []string{"Node C"}},
		},
		Connections: []diagram.Connection{
			{From: 1, To: 2, Label: "A->B"},
			{From: 2, To: 3, Label: "B->C"},
			{From: 1, To: 3, Label: "A->C"},
		},
	}

	// Create TUI editor
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(d)

	// Verify initial state
	if len(d.Connections) != 3 {
		t.Fatalf("Expected 3 connections, got %d", len(d.Connections))
	}

	// Start delete mode - simulate pressing 'd'
	tui.handleNormalKey('d')
	
	// Verify we're in jump mode with delete action
	if tui.GetMode() != ModeJump {
		t.Errorf("Expected ModeJump, got %v", tui.GetMode())
	}
	if tui.GetJumpAction() != JumpActionDelete {
		t.Errorf("Expected JumpActionDelete, got %v", tui.GetJumpAction())
	}

	// Verify labels were assigned to both nodes and connections
	nodeLabels := tui.GetJumpLabels()
	connLabels := tui.GetConnectionLabels()
	
	if len(nodeLabels) != 3 {
		t.Errorf("Expected 3 node labels, got %d", len(nodeLabels))
	}
	if len(connLabels) != 3 {
		t.Errorf("Expected 3 connection labels, got %d", len(connLabels))
	}

	// Get the first connection's label
	var firstConnLabel rune
	for _, label := range connLabels {
		firstConnLabel = label
		break
	}

	// Simulate pressing the connection's label key
	beforeCount := len(d.Connections)
	tui.handleJumpKey(firstConnLabel)
	afterCount := len(d.Connections)

	// Verify connection was deleted
	if afterCount != beforeCount-1 {
		t.Errorf("Connection not deleted: before=%d, after=%d", beforeCount, afterCount)
	}

	// Verify we're back in normal mode
	if tui.GetMode() != ModeNormal {
		t.Errorf("Expected ModeNormal after deletion, got %v", tui.GetMode())
	}

	// Verify labels were cleared
	if len(tui.GetJumpLabels()) != 0 {
		t.Errorf("Jump labels not cleared after deletion")
	}
	if len(tui.GetConnectionLabels()) != 0 {
		t.Errorf("Connection labels not cleared after deletion")
	}
}

func TestConnectionDeletionFullFlow(t *testing.T) {
	// Test the full key sequence: d -> connection_label
	d := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"A"}},
			{ID: 2, Text: []string{"B"}},
		},
		Connections: []diagram.Connection{
			{From: 1, To: 2, Label: "test"},
		},
	}

	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(d)

	// Press 'd' to enter delete mode
	result := tui.handleKey('d')
	if result {
		t.Error("handleKey returned true (exit) when it shouldn't")
	}

	// Should be in jump mode
	if tui.mode != ModeJump {
		t.Errorf("Expected ModeJump, got %v", tui.mode)
	}

	// Should have assigned labels
	connLabels := tui.connectionLabels
	if len(connLabels) != 1 {
		t.Fatalf("Expected 1 connection label, got %d", len(connLabels))
	}

	// Get the assigned label
	var connLabel rune
	for _, label := range connLabels {
		connLabel = label
		break
	}

	// Press the connection label to delete it
	initialConnCount := len(d.Connections)
	result = tui.handleKey(connLabel)
	if result {
		t.Error("handleKey returned true (exit) when it shouldn't")
	}

	// Check connection was deleted
	if len(d.Connections) != initialConnCount-1 {
		t.Errorf("Connection not deleted: expected %d connections, got %d", 
			initialConnCount-1, len(d.Connections))
	}

	// Should be back in normal mode
	if tui.mode != ModeNormal {
		t.Errorf("Expected ModeNormal after deletion, got %v", tui.mode)
	}
}

func TestNodeDeletionStillWorks(t *testing.T) {
	// Ensure node deletion still works after our changes
	d := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"Node A"}},
			{ID: 2, Text: []string{"Node B"}},
		},
		Connections: []diagram.Connection{
			{From: 1, To: 2, Label: "test"},
		},
	}

	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(d)

	// Press 'd' to enter delete mode
	tui.handleKey('d')

	// Get a node label
	nodeLabels := tui.jumpLabels
	if len(nodeLabels) != 2 {
		t.Fatalf("Expected 2 node labels, got %d", len(nodeLabels))
	}

	// Get the first node's label
	var nodeLabel rune
	var nodeID int
	for id, label := range nodeLabels {
		nodeLabel = label
		nodeID = id
		break
	}

	// Press the node label to delete it
	initialNodeCount := len(d.Nodes)
	tui.handleKey(nodeLabel)

	// Check node was deleted
	if len(d.Nodes) != initialNodeCount-1 {
		t.Errorf("Node not deleted: expected %d nodes, got %d", 
			initialNodeCount-1, len(d.Nodes))
	}

	// Check that connections involving the deleted node were also removed
	for _, conn := range d.Connections {
		if conn.From == nodeID || conn.To == nodeID {
			t.Errorf("Connection involving deleted node %d still exists", nodeID)
		}
	}
}
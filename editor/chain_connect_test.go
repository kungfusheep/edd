package editor

import (
	"testing"
)

func TestContinuousConnectChaining(t *testing.T) {
	tui := NewTUIEditor(NewRealRenderer())
	
	// Set up sequence diagram with 4 nodes
	tui.diagram.Type = "sequence"
	tui.AddNode([]string{"A"})
	tui.AddNode([]string{"B"})
	tui.AddNode([]string{"C"})
	tui.AddNode([]string{"D"})
	
	// Press 'C' to start continuous connect mode
	tui.HandleKey('C')
	
	// Verify we're in jump mode waiting for FROM
	if tui.mode != ModeJump {
		t.Errorf("Expected ModeJump after pressing C, got %v", tui.mode)
	}
	if tui.jumpAction != JumpActionConnectFrom {
		t.Errorf("Expected JumpActionConnectFrom, got %v", tui.jumpAction)
	}
	
	// Select first node as initial FROM (will have label 'a')
	if label, ok := tui.jumpLabels[1]; ok {
		tui.HandleKey(label)
	}
	
	// Should now be waiting for TO
	if tui.jumpAction != JumpActionConnectTo {
		t.Errorf("Expected JumpActionConnectTo after selecting FROM, got %v", tui.jumpAction)
	}
	
	// Select second node as TO - this creates A→B (will have label 's')
	if label, ok := tui.jumpLabels[2]; ok {
		tui.HandleKey(label)
	}
	
	// In continuous mode, should still be in jump mode
	// and 'b' should now be selected as the next FROM
	if tui.mode != ModeJump {
		t.Errorf("Should still be in ModeJump for continuous connect, got %v", tui.mode)
	}
	if tui.jumpAction != JumpActionConnectTo {
		t.Errorf("Expected JumpActionConnectTo for next connection, got %v", tui.jumpAction)
	}
	if tui.selected != 2 { // Node B has ID 2
		t.Errorf("Expected node B (ID 2) to be selected as next FROM, got %d", tui.selected)
	}
	
	// Select third node as next TO - this creates B→C (will have label 'd')
	if label, ok := tui.jumpLabels[3]; ok {
		tui.HandleKey(label)
	}
	
	// Should still be in continuous mode with C as next FROM
	if tui.mode != ModeJump {
		t.Errorf("Should still be in ModeJump for continuous connect, got %v", tui.mode)
	}
	if tui.selected != 3 { // Node C has ID 3
		t.Errorf("Expected node C (ID 3) to be selected as next FROM, got %d", tui.selected)
	}
	
	// Select fourth node as next TO - this creates C→D (will have label 'f')
	if label, ok := tui.jumpLabels[4]; ok {
		tui.HandleKey(label)
	}
	
	// Should still be in continuous mode with D as next FROM
	if tui.selected != 4 { // Node D has ID 4
		t.Errorf("Expected node D (ID 4) to be selected as next FROM, got %d", tui.selected)
	}
	
	// Press ESC to exit continuous mode
	tui.HandleKey(27)
	
	// Should be back in normal mode
	if tui.mode != ModeNormal {
		t.Errorf("Expected ModeNormal after ESC, got %v", tui.mode)
	}
	
	// Verify we created 3 connections: A→B, B→C, C→D
	if len(tui.diagram.Connections) != 3 {
		t.Errorf("Expected 3 connections, got %d", len(tui.diagram.Connections))
	}
	
	// Verify the connections are correct
	expectedConnections := []struct{ from, to int }{
		{1, 2}, // A→B
		{2, 3}, // B→C
		{3, 4}, // C→D
	}
	
	for i, expected := range expectedConnections {
		if i >= len(tui.diagram.Connections) {
			break
		}
		conn := tui.diagram.Connections[i]
		if conn.From != expected.from || conn.To != expected.to {
			t.Errorf("Connection %d: expected %d→%d, got %d→%d",
				i, expected.from, expected.to, conn.From, conn.To)
		}
	}
}

func TestContinuousConnectSelfLoop(t *testing.T) {
	tui := NewTUIEditor(NewRealRenderer())
	
	// Set up sequence diagram with 2 nodes
	tui.diagram.Type = "sequence"
	tui.AddNode([]string{"Server"})
	tui.AddNode([]string{"Client"})
	
	// Press 'C' to start continuous connect mode
	tui.HandleKey('C')
	
	// Select first node as initial FROM
	if label, ok := tui.jumpLabels[1]; ok {
		tui.HandleKey(label)
	}
	
	// Select first node again as TO - creates self-loop
	if label, ok := tui.jumpLabels[1]; ok {
		tui.HandleKey(label)
	}
	
	// Should still be in continuous mode with A as next FROM
	if tui.mode != ModeJump {
		t.Errorf("Should still be in ModeJump for continuous connect, got %v", tui.mode)
	}
	if tui.selected != 1 { // Node A has ID 1
		t.Errorf("Expected node A (ID 1) to be selected as next FROM, got %d", tui.selected)
	}
	
	// Select second node as next TO - creates A→B
	if label, ok := tui.jumpLabels[2]; ok {
		tui.HandleKey(label)
	}
	
	// Press ESC to exit
	tui.HandleKey(27)
	
	// Verify we created 2 connections: A→A (self-loop), A→B
	if len(tui.diagram.Connections) != 2 {
		t.Errorf("Expected 2 connections, got %d", len(tui.diagram.Connections))
	}
	
	// Verify first connection is self-loop
	if tui.diagram.Connections[0].From != 1 || tui.diagram.Connections[0].To != 1 {
		t.Errorf("Expected first connection to be 1→1 (self-loop), got %d→%d",
			tui.diagram.Connections[0].From, tui.diagram.Connections[0].To)
	}
	
	// Verify second connection
	if tui.diagram.Connections[1].From != 1 || tui.diagram.Connections[1].To != 2 {
		t.Errorf("Expected second connection to be 1→2, got %d→%d",
			tui.diagram.Connections[1].From, tui.diagram.Connections[1].To)
	}
}
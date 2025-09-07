package editor

import (
	"edd/diagram"
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
	var loaded diagram.Diagram
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
	
	// Test node hints for text alignment
	t.Run("NodeTextAlignment", func(t *testing.T) {
		// Simulate entering hint menu for node
		tui.editingHintNode = id1
		tui.SetMode(ModeHintMenu)
		
		// Test toggling center alignment
		tui.HandleHintMenuInput('t')
		node := tui.GetDiagram().Nodes[0]
		if node.Hints["text-align"] != "center" {
			t.Errorf("Expected text-align=center, got %v", node.Hints["text-align"])
		}
		
		// Toggle again should remove it (back to default left)
		tui.HandleHintMenuInput('t')
		if _, exists := node.Hints["text-align"]; exists {
			t.Errorf("Expected text-align to be removed, but got %v", node.Hints["text-align"])
		}
		
		// Exit mode
		tui.HandleHintMenuInput(27)
	})
	
	// Test connection hints
	t.Run("ConnectionHints", func(t *testing.T) {
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
		
		// Test ESC exits mode (this test is for the old behavior)
		// Now ESC should return to jump mode if previousJumpAction is set
		tui.HandleHintMenuInput(27)
		// Since we didn't come from jump mode, it should go to normal
		if tui.GetMode() != ModeNormal {
			t.Error("ESC should return to normal mode when no previousJumpAction")
		}
	})
}

func TestHintMenuEnterExitsToNormal(t *testing.T) {
	// Create a simple diagram with nodes
	diagram := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"Node 1"}},
			{ID: 2, Text: []string{"Node 2"}},
		},
		Connections: []diagram.Connection{
			{From: 1, To: 2, Label: "connects"},
		},
	}

	// Create TUI editor
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(diagram)

	// Test node hints menu
	t.Run("NodeHintsEnterKey", func(t *testing.T) {
		// Press 'H' to enter hints jump mode
		tui.HandleKey('H')
		if tui.GetMode() != ModeJump {
			t.Errorf("Expected ModeJump after 'H', got %v", tui.GetMode())
		}

		// Select first node (should have label 'a')
		tui.HandleKey('a')
		if tui.GetMode() != ModeHintMenu {
			t.Errorf("Expected ModeHintMenu after selecting node, got %v", tui.GetMode())
		}

		// Press Enter - should exit to normal mode (test key code 13)
		tui.HandleKey(13) // Enter key (CR)
		if tui.GetMode() != ModeNormal {
			t.Errorf("Expected ModeNormal after Enter(13) in hints menu, got %v", tui.GetMode())
		}
		
		// Verify previousJumpAction was cleared
		if tui.previousJumpAction != 0 {
			t.Errorf("Expected previousJumpAction to be cleared, got %v", tui.previousJumpAction)
		}
		
		// Test again with key code 10 (LF)
		tui.HandleKey('H')
		tui.HandleKey('a')
		tui.HandleKey(10) // Enter key (LF)
		if tui.GetMode() != ModeNormal {
			t.Errorf("Expected ModeNormal after Enter(10) in hints menu, got %v", tui.GetMode())
		}
	})

	// Test connection hints menu
	t.Run("ConnectionHintsEnterKey", func(t *testing.T) {
		// Press 'H' to enter hints jump mode
		tui.HandleKey('H')
		if tui.GetMode() != ModeJump {
			t.Errorf("Expected ModeJump after 'H', got %v", tui.GetMode())
		}

		// Select first connection (should have label 'd' after nodes a,s)
		tui.HandleKey('d')
		if tui.GetMode() != ModeHintMenu {
			t.Errorf("Expected ModeHintMenu after selecting connection, got %v", tui.GetMode())
		}

		// Press Enter - should exit to normal mode
		tui.HandleKey(13) // Enter key
		if tui.GetMode() != ModeNormal {
			t.Errorf("Expected ModeNormal after Enter in hints menu, got %v", tui.GetMode())
		}
		
		// Verify previousJumpAction was cleared
		if tui.previousJumpAction != 0 {
			t.Errorf("Expected previousJumpAction to be cleared, got %v", tui.previousJumpAction)
		}
	})
}

func TestHintMenuESCReturnsToJump(t *testing.T) {
	// Create a simple diagram with nodes
	diagram := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"Node 1"}},
			{ID: 2, Text: []string{"Node 2"}},
		},
		Connections: []diagram.Connection{
			{From: 1, To: 2, Label: "connects"},
		},
	}

	// Create TUI editor
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(diagram)

	// Test node hints menu ESC behavior
	t.Run("NodeHintsESCKey", func(t *testing.T) {
		// Press 'H' to enter hints jump mode
		tui.HandleKey('H')
		if tui.GetMode() != ModeJump {
			t.Errorf("Expected ModeJump after 'H', got %v", tui.GetMode())
		}
		jumpAction := tui.GetJumpAction()

		// Select first node
		tui.HandleKey('a')
		if tui.GetMode() != ModeHintMenu {
			t.Errorf("Expected ModeHintMenu after selecting node, got %v", tui.GetMode())
		}

		// Press ESC - should return to jump mode with same action
		tui.HandleKey(27) // ESC key
		if tui.GetMode() != ModeJump {
			t.Errorf("Expected ModeJump after ESC in hints menu, got %v", tui.GetMode())
		}
		if tui.GetJumpAction() != jumpAction {
			t.Errorf("Expected jump action %v after ESC, got %v", jumpAction, tui.GetJumpAction())
		}
		
		// Clean up - exit jump mode
		tui.HandleKey(27) // ESC to exit jump mode
	})
	
	// Test connection hints menu ESC behavior
	t.Run("ConnectionHintsESCKey", func(t *testing.T) {
		// Press 'H' to enter hints jump mode
		tui.HandleKey('H')
		if tui.GetMode() != ModeJump {
			t.Errorf("Expected ModeJump after 'H', got %v", tui.GetMode())
		}
		jumpAction := tui.GetJumpAction()

		// Select first connection (should be 'd' after nodes a,s)
		tui.HandleKey('d')
		if tui.GetMode() != ModeHintMenu {
			t.Errorf("Expected ModeHintMenu after selecting connection, got %v", tui.GetMode())
		}

		// Press ESC - should return to jump mode with same action
		tui.HandleKey(27) // ESC key
		if tui.GetMode() != ModeJump {
			t.Errorf("Expected ModeJump after ESC in connection hints menu, got %v", tui.GetMode())
		}
		if tui.GetJumpAction() != jumpAction {
			t.Errorf("Expected jump action %v after ESC, got %v", jumpAction, tui.GetJumpAction())
		}
	})
}
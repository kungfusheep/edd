package editor

import (
	"edd/core"
	"testing"
)

func TestConnectionEditESCReturnsToJump(t *testing.T) {
	// Create a simple diagram with nodes and connection
	diagram := &core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"Node 1"}},
			{ID: 2, Text: []string{"Node 2"}},
		},
		Connections: []core.Connection{
			{From: 1, To: 2, Label: "connects"},
		},
	}

	// Create TUI editor
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(diagram)

	// Test connection edit ESC behavior
	t.Run("ConnectionEditESCKey", func(t *testing.T) {
		// Press 'e' to enter edit jump mode
		tui.HandleKey('e')
		if tui.GetMode() != ModeJump {
			t.Errorf("Expected ModeJump after 'e', got %v", tui.GetMode())
		}
		jumpAction := tui.GetJumpAction()
		if jumpAction != JumpActionEdit {
			t.Errorf("Expected JumpActionEdit, got %v", jumpAction)
		}

		// Select first connection (should be 'd' after nodes a,s)
		tui.HandleKey('d')
		if tui.GetMode() != ModeEdit {
			t.Errorf("Expected ModeEdit after selecting connection, got %v", tui.GetMode())
		}

		// Press ESC - should return to jump mode with edit action
		tui.HandleKey(27) // ESC key
		if tui.GetMode() != ModeJump {
			t.Errorf("Expected ModeJump after ESC in edit mode, got %v", tui.GetMode())
		}
		if tui.GetJumpAction() != jumpAction {
			t.Errorf("Expected jump action %v after ESC, got %v", jumpAction, tui.GetJumpAction())
		}

		// Clean up
		tui.HandleKey(27) // ESC to exit jump mode
	})
}
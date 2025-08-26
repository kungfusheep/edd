package editor

import (
	"edd/core"
	"fmt"
	"testing"
)

func TestJumpModeRendering(t *testing.T) {
	// Simulate exactly what happens when jump mode is active
	tui := NewTUIEditor(nil)
	tui.SetDiagram(&core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"hi"}},
			{ID: 2, Text: []string{"there"}},
		},
	})
	
	// Start jump mode (like pressing 'c' for connect)
	tui.startJump(JumpActionConnectFrom)
	
	// Check the state
	fmt.Printf("Mode: %v\n", tui.mode)
	fmt.Printf("Jump labels: %v\n", tui.jumpLabels)
	fmt.Printf("Jump labels active: %v\n", len(tui.jumpLabels) > 0)
	
	// Get the render state
	state := tui.GetState()
	fmt.Printf("State mode: %v\n", state.Mode)
	fmt.Printf("State jump labels: %v\n", state.JumpLabels)
	
	// Mode indicators are now rendered using ANSI escape codes
	// Verify state instead of output text
	_ = RenderTUI(state)
	
	// Check that we're in jump mode with proper labels
	if state.Mode != ModeJump {
		t.Errorf("Expected ModeJump, got %v", state.Mode)
	}
	
	if len(state.JumpLabels) != 2 {
		t.Errorf("Expected 2 jump labels, got %d", len(state.JumpLabels))
	}
}
package editor

import (
	"edd/core"
	"fmt"
	"strings"
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
	
	// Render
	output := RenderTUI(state)
	
	// Check for duplicate mode indicators
	normalCount := strings.Count(output, "NORMAL")
	jumpCount := strings.Count(output, "JUMP")
	
	fmt.Printf("NORMAL appears %d times\n", normalCount)
	fmt.Printf("JUMP appears %d times\n", jumpCount)
	
	if normalCount > 0 && jumpCount > 0 {
		t.Error("Both NORMAL and JUMP modes shown simultaneously")
	}
	
	if jumpCount != 1 {
		t.Errorf("JUMP should appear exactly once, got %d", jumpCount)
	}
	
	fmt.Println("\n=== Output ===")
	fmt.Println(output)
}
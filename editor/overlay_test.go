package editor

import (
	"edd/core"
	"fmt"
	"strings"
	"testing"
)

// TestEdPositionBottomRight verifies Ed appears in bottom-right corner
func TestEdPositionBottomRight(t *testing.T) {
	state := TUIState{
		Diagram: &core.Diagram{
			Nodes: []core.Node{
				{ID: 1, Text: []string{"Node1"}},
				{ID: 2, Text: []string{"Node2"}},
			},
			Connections: []core.Connection{
				{From: 1, To: 2},
			},
		},
		Mode:     ModeNormal,
		EddFrame: "◉‿◉",
		Width:    80,
		Height:   24,
	}
	
	output := RenderTUI(state)
	lines := strings.Split(output, "\n")
	
	// Ed should be in the last few lines, on the right side
	foundEd := false
	for i := len(lines) - 5; i < len(lines) && i >= 0; i++ {
		if strings.Contains(lines[i], "◉‿◉") {
			foundEd = true
			// Check it's on the right side (past column 50)
			edIndex := strings.Index(lines[i], "◉‿◉")
			if edIndex < 40 {
				t.Errorf("Ed should be on the right side, found at column %d", edIndex)
			}
			break
		}
	}
	
	if !foundEd {
		t.Error("Ed mascot not found in bottom area")
	}
}

// TestSingleModeIndicator ensures only one mode shows at a time
func TestSingleModeIndicator(t *testing.T) {
	modes := []Mode{ModeNormal, ModeJump, ModeEdit, ModeCommand}
	
	for _, mode := range modes {
		state := TUIState{
			Diagram:  &core.Diagram{},
			Mode:     mode,
			EddFrame: "◉‿◉",
		}
		
		output := RenderTUI(state)
		
		// Count how many times mode strings appear
		modeCount := 0
		for _, m := range modes {
			if strings.Count(output, m.String()) > 0 {
				modeCount++
			}
		}
		
		if modeCount > 1 {
			t.Errorf("Multiple modes shown for %s mode. Output:\n%s", mode, output)
		}
	}
}

// TestEdFaceRendering ensures Ed's face doesn't get corrupted
func TestEdFaceRendering(t *testing.T) {
	faces := map[string]string{
		"normal":  "◉‿◉",
		"command": ":_",
		"jump":    "◎‿◎",
	}
	
	for name, face := range faces {
		state := TUIState{
			Diagram:  &core.Diagram{},
			Mode:     ModeNormal,
			EddFrame: face,
		}
		
		output := RenderTUI(state)
		
		// Check that the face appears correctly
		if !strings.Contains(output, face) {
			t.Errorf("Ed face %s (%s) not found or corrupted in output", name, face)
		}
		
		// Check for corrupted versions
		corrupted := []string{"◉_◉", "| ◉", "◉ |"}
		for _, bad := range corrupted {
			if strings.Contains(output, bad) {
				t.Errorf("Found corrupted Ed face %q in output for %s", bad, name)
			}
		}
	}
}

// TestOverlayClearance ensures overlays don't overlap diagram content
func TestOverlayClearance(t *testing.T) {
	state := TUIState{
		Diagram: &core.Diagram{
			Nodes: []core.Node{
				{ID: 1, Text: []string{"TopLeft"}},
				{ID: 2, Text: []string{"TopRight"}},
			},
		},
		Mode:     ModeNormal,
		EddFrame: "◉‿◉",
	}
	
	output := RenderTUI(state)
	
	// Both node texts should be visible
	if !strings.Contains(output, "TopLeft") {
		t.Error("TopLeft node text hidden by overlay")
	}
	if !strings.Contains(output, "TopRight") {
		t.Error("TopRight node text hidden by overlay")
	}
	
	fmt.Println("=== Overlay Test Output ===")
	fmt.Println(output)
	fmt.Println("=== End ===")
}

// TestNoOverlayDuplication verifies no duplicate mode indicators
func TestNoOverlayDuplication(t *testing.T) {
	tui := NewTUIEditor(nil)
	tui.SetDiagram(&core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"Node 1"}},
			{ID: 2, Text: []string{"Node 2"}},
			{ID: 3, Text: []string{"Node 3"}},
		},
	})
	
	// Test in normal mode
	tui.SetMode(ModeNormal)
	output := tui.Render()
	normalCount := strings.Count(output, "NORMAL")
	if normalCount != 1 {
		t.Errorf("NORMAL mode should appear exactly once, got %d", normalCount)
		fmt.Println("=== Normal Mode Output ===")
		fmt.Println(output)
	}
	
	// Test transition to jump mode (like pressing 'c')
	tui.StartConnect()
	output = tui.Render()
	jumpCount := strings.Count(output, "JUMP")
	normalCount = strings.Count(output, "NORMAL")
	
	if jumpCount != 1 {
		t.Errorf("JUMP mode should appear exactly once, got %d", jumpCount)
	}
	if normalCount != 0 {
		t.Errorf("NORMAL should not appear in JUMP mode, got %d", normalCount)
	}
	
	fmt.Println("=== Jump Mode Output ===")
	fmt.Println(output)
	
	// Test that Ed mascot appears only once
	edCount := strings.Count(output, "◉‿◉")
	if edCount == 0 {
		edCount = strings.Count(output, "◎‿◎")
	}
	if edCount > 1 {
		t.Errorf("Ed mascot should appear only once, got %d instances", edCount)
	}
}

// TestClearBetweenModes verifies clean transitions between modes
func TestClearBetweenModes(t *testing.T) {
	tui := NewTUIEditor(nil)
	tui.SetDiagram(&core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"Test"}},
		},
	})
	
	// Normal mode
	tui.SetMode(ModeNormal)
	output1 := tui.Render()
	
	// Jump mode
	tui.startJump(JumpActionEdit)
	output2 := tui.Render()
	
	// Back to normal
	tui.HandleJumpInput(27) // ESC
	output3 := tui.Render()
	
	// Check each output independently
	modes := []struct{
		output string
		name string
		expected string
	}{
		{output1, "First Normal", "NORMAL"},
		{output2, "Jump", "JUMP"},
		{output3, "Second Normal", "NORMAL"},
	}
	
	for _, m := range modes {
		count := strings.Count(m.output, m.expected)
		otherModes := []string{"NORMAL", "JUMP", "INSERT", "EDIT", "COMMAND"}
		for _, other := range otherModes {
			if other != m.expected {
				if strings.Contains(m.output, other) {
					t.Errorf("%s mode output contains unexpected %s mode", m.name, other)
				}
			}
		}
		if count != 1 {
			t.Errorf("%s mode should show %s exactly once, got %d", m.name, m.expected, count)
		}
	}
}
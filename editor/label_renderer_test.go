package editor

import (
	"edd/diagram"
	"testing"
)

func TestCalculateLabelPositions(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)

	// Set up a sequence diagram
	d := &diagram.Diagram{
		Type: "sequence",
		Nodes: []diagram.Node{
			{ID: 0, Text: []string{"client"}, X: 5, Y: 2},
			{ID: 1, Text: []string{"server"}, X: 40, Y: 2},
			{ID: 2, Text: []string{"database"}, X: 75, Y: 2},
		},
	}

	tui.SetDiagram(d)
	tui.SetTerminalSize(100, 30)

	// Manually set node positions as renderer would
	tui.nodePositions = map[int]diagram.Point{
		0: {X: 5, Y: 2},
		1: {X: 40, Y: 2},
		2: {X: 75, Y: 2},
	}

	// Test 1: No scroll, no labels assigned yet
	positions := tui.CalculateLabelPositions(false)
	if len(positions) != 0 {
		t.Errorf("Expected 0 positions before labels assigned, got %d", len(positions))
	}

	// Assign labels
	tui.startJump(JumpActionSelect)

	// Test 2: No scroll
	tui.diagramScrollOffset = 0
	positions = tui.CalculateLabelPositions(false)

	if len(positions) != 3 {
		t.Fatalf("Expected 3 label positions, got %d", len(positions))
	}

	// Check first participant label position
	// With no scroll, participant at Y=2 should appear at viewport Y=3 (Y + 1 for 1-based)
	found := false
	for _, pos := range positions {
		if pos.NodeID == 0 {
			found = true
			expectedX := 4 // 5 - 1 for box corner
			expectedY := 3 // 2 + 1 for 1-based terminal
			if pos.ViewportX != expectedX || pos.ViewportY != expectedY {
				t.Errorf("Node 0: Expected position (%d,%d), got (%d,%d)",
					expectedX, expectedY, pos.ViewportX, pos.ViewportY)
			}
			if pos.Label != 'a' {
				t.Errorf("Node 0: Expected label 'a', got '%c'", pos.Label)
			}
		}
	}
	if !found {
		t.Error("Node 0 position not found")
	}

	// Test 3: Scrolled with sticky headers
	tui.diagramScrollOffset = 10
	positions = tui.CalculateLabelPositions(true) // Has scroll indicator

	if len(positions) != 3 {
		t.Errorf("Expected 3 positions with sticky headers, got %d", len(positions))
	}

	// With sticky headers active and scroll indicator:
	// - Line 1: Scroll indicator
	// - Lines 2-3: Padding lines
	// - Lines 4-8: Headers (participants appear with padding)
	// - Line 9: Separator
	// The participant at Y=2 should appear at line 5 (2 + 2 padding + 1 scroll indicator)
	for _, pos := range positions {
		if pos.NodeID == 0 {
			expectedY := 5 // 2 + 2(padding) + 1(scroll indicator)
			if pos.ViewportY != expectedY {
				t.Errorf("Node 0 with sticky headers: Expected Y=%d, got Y=%d",
					expectedY, pos.ViewportY)
			}
		}
	}
}

func TestLabelVisibilityWithScrolling(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)

	// Create a diagram with nodes at different Y positions
	d := &diagram.Diagram{
		Type: "box",
		Nodes: []diagram.Node{
			{ID: 0, Text: []string{"Top"}, X: 10, Y: 5},
			{ID: 1, Text: []string{"Middle"}, X: 10, Y: 20},
			{ID: 2, Text: []string{"Bottom"}, X: 10, Y: 40},
		},
	}

	tui.SetDiagram(d)
	tui.SetTerminalSize(50, 20) // Small viewport

	// Set positions
	tui.nodePositions = map[int]diagram.Point{
		0: {X: 10, Y: 5},
		1: {X: 10, Y: 20},
		2: {X: 10, Y: 40},
	}

	// Assign labels
	tui.jumpLabels = map[int]rune{
		0: 'a',
		1: 'b',
		2: 'c',
	}

	// Test with no scroll - only top node visible
	tui.diagramScrollOffset = 0
	positions := tui.CalculateLabelPositions(false)

	visibleNodes := make(map[int]bool)
	for _, pos := range positions {
		visibleNodes[pos.NodeID] = true
		t.Logf("Visible at scroll=0: Node %d at viewport Y=%d", pos.NodeID, pos.ViewportY)
	}

	if !visibleNodes[0] {
		t.Error("Node 0 should be visible at scroll=0")
	}
	if visibleNodes[2] {
		t.Error("Node 2 should not be visible at scroll=0")
	}

	// Test with scroll=15 - middle node visible
	tui.diagramScrollOffset = 15
	positions = tui.CalculateLabelPositions(true) // Has scroll indicator

	visibleNodes = make(map[int]bool)
	for _, pos := range positions {
		visibleNodes[pos.NodeID] = true
		t.Logf("Visible at scroll=15: Node %d at viewport Y=%d", pos.NodeID, pos.ViewportY)
	}

	if visibleNodes[0] {
		t.Error("Node 0 should not be visible at scroll=15")
	}
	if !visibleNodes[1] {
		t.Error("Node 1 should be visible at scroll=15")
	}
}

func TestRenderLabelsToString(t *testing.T) {
	positions := []LabelPosition{
		{NodeID: 0, Label: 'a', ViewportX: 5, ViewportY: 3, IsFrom: false},
		{NodeID: 1, Label: 'b', ViewportX: 20, ViewportY: 3, IsFrom: false},
	}

	output := RenderLabelsToString(positions)

	// Should contain:
	// - Save cursor: \033[s
	// - Move to (3,5): \033[3;5H
	// - Draw 'a': \033[33;1ma\033[0m
	// - Move to (3,20): \033[3;20H
	// - Draw 'b': \033[33;1mb\033[0m
	// - Restore cursor: \033[u

	expectedParts := []string{
		"\033[s",          // Save cursor
		"\033[3;5H",       // Position for 'a'
		"\033[33;1ma",     // Yellow 'a'
		"\033[3;20H",      // Position for 'b'
		"\033[33;1mb",     // Yellow 'b'
		"\033[u",          // Restore cursor
	}

	for _, part := range expectedParts {
		if !contains(output, part) {
			t.Errorf("Output missing expected part: %q", part)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		   len(s) > len(substr) && contains(s[1:], substr)
}
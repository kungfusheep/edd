package editor

import (
	"edd/core"
	"strings"
	"testing"
)

func TestJSONViewMode(t *testing.T) {
	diagram := &core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"Node A"}},
			{ID: 2, Text: []string{"Node B"}},
		},
		Connections: []core.Connection{
			{From: 1, To: 2, Label: "connects"},
		},
	}

	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(diagram)
	tui.SetTerminalSize(80, 24)

	// Switch to JSON mode
	tui.handleNormalKey('j')
	if tui.GetMode() != ModeJSON {
		t.Errorf("Expected ModeJSON, got %v", tui.GetMode())
	}

	// Render JSON view
	output := tui.Render()
	
	// Should contain JSON structure
	if !strings.Contains(output, "nodes") {
		t.Error("JSON output should contain 'nodes'")
	}
	if !strings.Contains(output, "Node A") {
		t.Error("JSON output should contain node text")
	}
	if !strings.Contains(output, "connections") {
		t.Error("JSON output should contain 'connections'")
	}
	if !strings.Contains(output, "connects") {
		t.Error("JSON output should contain connection label")
	}
	
	// Should have line numbers
	if !strings.Contains(output, "1 │") {
		t.Error("JSON output should have line numbers")
	}
}

func TestJSONViewScrolling(t *testing.T) {
	// Create a large diagram to test scrolling
	diagram := &core.Diagram{
		Nodes: []core.Node{},
	}
	for i := 1; i <= 20; i++ {
		diagram.Nodes = append(diagram.Nodes, core.Node{
			ID:   i,
			Text: []string{strings.Repeat("Node ", 10)}, // Long text to create many JSON lines
		})
	}

	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(diagram)
	tui.SetTerminalSize(80, 10) // Small terminal to force scrolling

	// Switch to JSON mode
	tui.SetMode(ModeJSON)

	// Test scrolling down
	initialOffset := tui.GetJSONScrollOffset()
	tui.handleJSONKey('J') // Scroll down
	if tui.GetJSONScrollOffset() <= initialOffset {
		t.Error("Scrolling down should increase offset")
	}

	// Test scrolling up
	tui.ScrollJSON(5) // Scroll down more
	offset := tui.GetJSONScrollOffset()
	tui.handleJSONKey('k') // Scroll up
	if tui.GetJSONScrollOffset() >= offset {
		t.Error("Scrolling up should decrease offset")
	}

	// Test go to top
	tui.handleJSONKey('g')
	if tui.GetJSONScrollOffset() != 0 {
		t.Error("'g' should go to top (offset 0)")
	}

	// Test go to bottom
	tui.handleJSONKey('G')
	if tui.GetJSONScrollOffset() == 0 {
		t.Error("'G' should go to bottom (offset > 0)")
	}
}

func TestJSONViewExit(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(&core.Diagram{})

	// Enter JSON mode
	tui.SetMode(ModeJSON)
	if tui.GetMode() != ModeJSON {
		t.Error("Should be in JSON mode")
	}

	// Test exit with 'q'
	tui.handleJSONKey('q')
	if tui.GetMode() != ModeNormal {
		t.Errorf("'q' should exit to normal mode, got %v", tui.GetMode())
	}

	// Enter JSON mode again
	tui.SetMode(ModeJSON)

	// Test exit with ESC
	tui.handleJSONKey(27)
	if tui.GetMode() != ModeNormal {
		t.Errorf("ESC should exit to normal mode, got %v", tui.GetMode())
	}

	// Enter JSON mode again
	tui.SetMode(ModeJSON)

	// Test exit with 'j'
	tui.handleJSONKey('j')
	if tui.GetMode() != ModeNormal {
		t.Errorf("'j' should exit to normal mode, got %v", tui.GetMode())
	}
}
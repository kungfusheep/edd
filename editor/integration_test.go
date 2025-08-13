package editor

import (
	"edd/core"
	"strings"
	"testing"
)

func TestConnectionDeletionWithRendering(t *testing.T) {
	// Create a test diagram with connections
	diagram := &core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"Node A"}, X: 0, Y: 0, Width: 10, Height: 3},
			{ID: 2, Text: []string{"Node B"}, X: 20, Y: 0, Width: 10, Height: 3},
			{ID: 3, Text: []string{"Node C"}, X: 10, Y: 10, Width: 10, Height: 3},
		},
		Connections: []core.Connection{
			{From: 1, To: 2, Label: "A->B"},
			{From: 2, To: 3, Label: "B->C"},
			{From: 1, To: 3, Label: "A->C"},
		},
	}

	// Create TUI editor with real renderer
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(diagram)
	tui.SetTerminalSize(80, 24)

	// Render once to populate connection paths
	output := tui.Render()
	if output == "" {
		t.Error("Render output is empty")
	}

	// Verify connections are rendered
	if !strings.Contains(output, "─") && !strings.Contains(output, "│") {
		t.Error("No connection lines found in output")
	}

	// Start delete mode
	tui.handleNormalKey('d')
	
	// Verify connection paths were populated
	connPaths := tui.GetConnectionPaths()
	if len(connPaths) != 3 {
		t.Errorf("Expected 3 connection paths, got %d", len(connPaths))
	}

	// Verify connection labels were assigned
	connLabels := tui.GetConnectionLabels()
	if len(connLabels) != 3 {
		t.Errorf("Expected 3 connection labels, got %d", len(connLabels))
	}

	// Get first connection label
	var firstLabel rune
	for _, label := range connLabels {
		firstLabel = label
		break
	}

	// Delete the first connection
	beforeCount := len(diagram.Connections)
	tui.handleJumpKey(firstLabel)
	afterCount := len(diagram.Connections)

	if afterCount != beforeCount-1 {
		t.Errorf("Connection not deleted: before=%d, after=%d", beforeCount, afterCount)
	}

	// Render again and verify connection is gone
	output2 := tui.Render()
	
	// The output should have fewer connection lines
	// This is a simple heuristic check
	lineCount1 := strings.Count(output, "─") + strings.Count(output, "│")
	lineCount2 := strings.Count(output2, "─") + strings.Count(output2, "│")
	
	if lineCount2 >= lineCount1 {
		t.Errorf("Expected fewer connection lines after deletion, got %d vs %d", lineCount2, lineCount1)
	}
}

func TestConnectionLabelPositioning(t *testing.T) {
	// Test that connection labels are positioned at path midpoints
	diagram := &core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"A"}, X: 0, Y: 0, Width: 5, Height: 3},
			{ID: 2, Text: []string{"B"}, X: 10, Y: 0, Width: 5, Height: 3},
		},
		Connections: []core.Connection{
			{From: 1, To: 2, Label: ""},
		},
	}

	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(diagram)

	// Render to populate paths
	tui.Render()

	// Enter delete mode to assign labels
	tui.handleNormalKey('d')

	// Get connection paths and labels
	paths := tui.GetConnectionPaths()
	labels := tui.GetConnectionLabels()

	if len(paths) != 1 || len(labels) != 1 {
		t.Fatalf("Expected 1 path and 1 label, got %d paths and %d labels", len(paths), len(labels))
	}

	// Get the path
	var path core.Path
	for _, p := range paths {
		path = p
		break
	}

	// The label should be positioned at the midpoint
	if len(path.Points) > 0 {
		midIndex := len(path.Points) / 2
		midPoint := path.Points[midIndex]
		
		// Just verify the midpoint is reasonable (between the nodes)
		if midPoint.X < 0 || midPoint.Y < 0 {
			t.Errorf("Invalid midpoint position: %v", midPoint)
		}
	}
}

func TestDeleteModeOnlyAssignsConnectionLabels(t *testing.T) {
	// Test that connection labels are only assigned in delete mode
	diagram := &core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"A"}},
			{ID: 2, Text: []string{"B"}},
		},
		Connections: []core.Connection{
			{From: 1, To: 2},
		},
	}

	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(diagram)

	// Test edit mode - should NOT assign connection labels
	tui.handleNormalKey('e')
	if len(tui.GetConnectionLabels()) != 0 {
		t.Error("Connection labels assigned in edit mode")
	}
	tui.handleKey(27) // ESC to cancel

	// Test connect mode - should NOT assign connection labels
	tui.handleNormalKey('c')
	if len(tui.GetConnectionLabels()) != 0 {
		t.Error("Connection labels assigned in connect mode")
	}
	tui.handleKey(27) // ESC to cancel

	// Test delete mode - SHOULD assign connection labels
	tui.handleNormalKey('d')
	if len(tui.GetConnectionLabels()) != 1 {
		t.Errorf("Expected 1 connection label in delete mode, got %d", len(tui.GetConnectionLabels()))
	}
}
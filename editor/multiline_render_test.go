package editor

import (
	"edd/diagram"
	"strings"
	"testing"
)

func TestMultilineNodeRendering(t *testing.T) {
	// Create a real renderer
	renderer := NewRealRenderer()
	
	// Create a diagram with multi-line nodes
	diagram := &diagram.Diagram{
		Nodes: []diagram.Node{
			{
				ID:   1,
				Text: []string{"Line 1", "Line 2", "Line 3"},
				X:    5,
				Y:    2,
			},
			{
				ID:   2,
				Text: []string{"Single"},
				X:    20,
				Y:    2,
			},
		},
	}
	
	// Render the diagram
	output, err := renderer.Render(diagram)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}
	
	// Check that the output contains all lines of text
	if !strings.Contains(output, "Line 1") {
		t.Error("Output missing 'Line 1'")
	}
	if !strings.Contains(output, "Line 2") {
		t.Error("Output missing 'Line 2'")
	}
	if !strings.Contains(output, "Line 3") {
		t.Error("Output missing 'Line 3'")
	}
	
	// Parse the output to check text is properly arranged
	lines := strings.Split(output, "\n")
	
	// Find "Line 1" and verify subsequent lines
	line1Index := -1
	for i, line := range lines {
		if strings.Contains(line, "Line 1") {
			line1Index = i
			break
		}
	}
	
	if line1Index == -1 {
		t.Fatal("Could not find 'Line 1' in output")
	}
	
	// Check that Line 2 and Line 3 are on the next lines (within the same box)
	if line1Index+1 >= len(lines) || !strings.Contains(lines[line1Index+1], "Line 2") {
		t.Errorf("'Line 2' should be on the line after 'Line 1'")
		if line1Index+1 < len(lines) {
			t.Logf("Line after 'Line 1': %s", lines[line1Index+1])
		}
	}
	
	if line1Index+2 >= len(lines) || !strings.Contains(lines[line1Index+2], "Line 3") {
		t.Errorf("'Line 3' should be two lines after 'Line 1'")
		if line1Index+2 < len(lines) {
			t.Logf("Two lines after 'Line 1': %s", lines[line1Index+2])
		}
	}
	
	// Visual inspection - log the output
	t.Logf("Multi-line node rendering:\n%s", output)
}

func TestMultilineEditingAndRendering(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	
	// Add a node and edit it to have multiple lines
	nodeID := tui.AddNode([]string{"Initial"})
	tui.selected = nodeID
	tui.SetMode(ModeEdit)
	
	// Clear and type multi-line text
	tui.textBuffer = []rune{}
	tui.cursorPos = 0
	tui.handleTextKey('O')
	tui.handleTextKey('n')
	tui.handleTextKey('e')
	tui.handleTextKey(14) // Ctrl+N for newline
	tui.handleTextKey('T')
	tui.handleTextKey('w')
	tui.handleTextKey('o')
	tui.handleTextKey(14) // Ctrl+N for newline
	tui.handleTextKey('T')
	tui.handleTextKey('h')
	tui.handleTextKey('r')
	tui.handleTextKey('e')
	tui.handleTextKey('e')
	
	// Commit the text
	tui.commitText()
	
	// Check the node has 3 lines
	var node *diagram.Node
	for i := range tui.diagram.Nodes {
		if tui.diagram.Nodes[i].ID == nodeID {
			node = &tui.diagram.Nodes[i]
			break
		}
	}
	
	if node == nil {
		t.Fatal("Node not found")
	}
	
	if len(node.Text) != 3 {
		t.Errorf("Expected 3 lines, got %d: %v", len(node.Text), node.Text)
	}
	
	// Now render and check the output
	output := tui.Render()
	
	// All three lines should be visible
	if !strings.Contains(output, "One") {
		t.Error("'One' not found in rendered output")
	}
	if !strings.Contains(output, "Two") {
		t.Error("'Two' not found in rendered output")
	}
	if !strings.Contains(output, "Three") {
		t.Error("'Three' not found in rendered output")
	}
	
	// Split output into lines for analysis
	lines := strings.Split(output, "\n")
	
	// Find where "One" appears
	oneLineIdx := -1
	for i, line := range lines {
		if strings.Contains(line, "One") {
			oneLineIdx = i
			break
		}
	}
	
	if oneLineIdx == -1 {
		t.Fatal("Could not find 'One' in output")
	}
	
	// Check that "Two" and "Three" are on subsequent lines
	if oneLineIdx+1 >= len(lines) || !strings.Contains(lines[oneLineIdx+1], "Two") {
		t.Error("'Two' should be on the line after 'One'")
		if oneLineIdx+1 < len(lines) {
			t.Logf("Line after 'One': %s", lines[oneLineIdx+1])
		}
	}
	
	if oneLineIdx+2 >= len(lines) || !strings.Contains(lines[oneLineIdx+2], "Three") {
		t.Error("'Three' should be two lines after 'One'")
		if oneLineIdx+2 < len(lines) {
			t.Logf("Two lines after 'One': %s", lines[oneLineIdx+2])
		}
	}
	
	// Log the output for debugging
	t.Logf("Rendered output:\n%s", output)
}
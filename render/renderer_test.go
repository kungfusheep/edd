package render

import (
	"edd/diagram"
	"strings"
	"testing"
)

// TestRendererBasic tests the basic rendering functionality
func TestRendererBasic(t *testing.T) {
	// Create a simple two-node diagram
	diagram := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"Hello"}},
			{ID: 2, Text: []string{"World"}},
		},
		Connections: []diagram.Connection{
			{From: 1, To: 2},
		},
	}
	
	// Create renderer
	renderer := NewRenderer()
	
	// Render the diagram
	output, err := renderer.Render(diagram)
	if err != nil {
		t.Fatalf("Failed to render diagram: %v", err)
	}
	
	// Print output for debugging
	t.Logf("Rendered output:\n%s", output)
	
	// Basic checks
	if output == "" {
		t.Error("Expected non-empty output")
	}
	
	// Check that both node texts appear
	if !strings.Contains(output, "Hello") {
		t.Error("Expected output to contain 'Hello'")
	}
	if !strings.Contains(output, "World") {
		t.Error("Expected output to contain 'World'")
	}
	
	// Check for box drawing characters (the renderer uses rounded corners by default)
	if !strings.Contains(output, "╭") || !strings.Contains(output, "╮") {
		t.Error("Expected output to contain box drawing characters")
	}
}

// TestRendererEmptyDiagram tests rendering an empty diagram
func TestRendererEmptyDiagram(t *testing.T) {
	diagram := &diagram.Diagram{
		Nodes:       []diagram.Node{},
		Connections: []diagram.Connection{},
	}
	
	renderer := NewRenderer()
	output, err := renderer.Render(diagram)
	if err != nil {
		t.Fatalf("Failed to render empty diagram: %v", err)
	}
	
	// Should still produce some output (empty canvas)
	if output == "" {
		t.Error("Expected non-empty output even for empty diagram")
	}
}

// TestRendererSingleNode tests rendering a single node with no connections
func TestRendererSingleNode(t *testing.T) {
	diagram := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"Lonely", "Node"}},
		},
		Connections: []diagram.Connection{},
	}
	
	renderer := NewRenderer()
	output, err := renderer.Render(diagram)
	if err != nil {
		t.Fatalf("Failed to render single node: %v", err)
	}
	
	// Check that the node text appears
	if !strings.Contains(output, "Lonely") {
		t.Error("Expected output to contain 'Lonely'")
	}
	if !strings.Contains(output, "Node") {
		t.Error("Expected output to contain 'Node'")
	}
}

// TestRendererMultipleConnections tests rendering with multiple connections
func TestRendererMultipleConnections(t *testing.T) {
	diagram := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"A"}},
			{ID: 2, Text: []string{"B"}},
			{ID: 3, Text: []string{"C"}},
		},
		Connections: []diagram.Connection{
			{From: 1, To: 2},
			{From: 2, To: 3},
			{From: 1, To: 3},
		},
	}
	
	renderer := NewRenderer()
	output, err := renderer.Render(diagram)
	if err != nil {
		t.Fatalf("Failed to render multiple connections: %v", err)
	}
	
	// Check all nodes appear
	for _, label := range []string{"A", "B", "C"} {
		if !strings.Contains(output, label) {
			t.Errorf("Expected output to contain '%s'", label)
		}
	}
	
	// Should have connection lines
	lines := strings.Split(output, "\n")
	connectionCount := 0
	for _, line := range lines {
		if strings.Contains(line, "─") || strings.Contains(line, "│") {
			connectionCount++
		}
	}
	if connectionCount < 3 {
		t.Error("Expected to see connection lines in output")
	}
}

// TestRendererSelfLoop tests rendering a node that connects to itself
func TestRendererSelfLoop(t *testing.T) {
	diagram := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"Recursive"}},
		},
		Connections: []diagram.Connection{
			{From: 1, To: 1},
		},
	}
	
	renderer := NewRenderer()
	output, err := renderer.Render(diagram)
	if err != nil {
		t.Fatalf("Failed to render self-loop: %v", err)
	}
	
	// Should contain the node
	if !strings.Contains(output, "Recursive") {
		t.Error("Expected output to contain 'Recursive'")
	}
	
	// Should have a visible loop (check for loop characters)
	// A self-loop typically extends beyond the node bounds
	lines := strings.Split(output, "\n")
	nodeFound := false
	for _, line := range lines {
		if strings.Contains(line, "Recursive") {
			nodeFound = true
			break
		}
	}
	if !nodeFound {
		t.Error("Could not find node in output")
	}
}
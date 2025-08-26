package rendering

import (
	"edd/core"
	"fmt"
	"strings"
	"testing"
)

// TestRendererVisualOutput creates visual examples to verify rendering quality
func TestRendererVisualOutput(t *testing.T) {
	tests := []struct {
		name    string
		diagram *core.Diagram
		width   int
		height  int
	}{
		{
			name: "Simple Two Nodes",
			diagram: &core.Diagram{
				Nodes: []core.Node{
					{ID: 1, Text: []string{"Node A"}},
					{ID: 2, Text: []string{"Node B"}},
				},
				Connections: []core.Connection{
					{From: 1, To: 2},
				},
			},
			width:  30,
			height: 10,
		},
		{
			name: "Three Node Chain",
			diagram: &core.Diagram{
				Nodes: []core.Node{
					{ID: 1, Text: []string{"Start"}},
					{ID: 2, Text: []string{"Middle"}},
					{ID: 3, Text: []string{"End"}},
				},
				Connections: []core.Connection{
					{From: 1, To: 2},
					{From: 2, To: 3},
				},
			},
			width:  40,
			height: 10,
		},
		{
			name: "Triangle Layout",
			diagram: &core.Diagram{
				Nodes: []core.Node{
					{ID: 1, Text: []string{"A"}},
					{ID: 2, Text: []string{"B"}},
					{ID: 3, Text: []string{"C"}},
				},
				Connections: []core.Connection{
					{From: 1, To: 2},
					{From: 2, To: 3},
					{From: 3, To: 1},
				},
			},
			width:  30,
			height: 15,
		},
		{
			name: "Self Loop",
			diagram: &core.Diagram{
				Nodes: []core.Node{
					{ID: 1, Text: []string{"Recursive", "Node"}},
				},
				Connections: []core.Connection{
					{From: 1, To: 1},
				},
			},
			width:  20,
			height: 10,
		},
		{
			name: "Multiple Connections",
			diagram: &core.Diagram{
				Nodes: []core.Node{
					{ID: 1, Text: []string{"Server"}},
					{ID: 2, Text: []string{"Client"}},
				},
				Connections: []core.Connection{
					{From: 1, To: 2},
					{From: 1, To: 2},
					{From: 2, To: 1},
				},
			},
			width:  30,
			height: 10,
		},
	}

	renderer := NewRenderer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := renderer.Render(tt.diagram)
			if err != nil {
				t.Errorf("Failed to render: %v", err)
				return
			}

			// Display the output
			fmt.Printf("\n=== %s ===\n", tt.name)
			fmt.Printf("Nodes: %d, Connections: %d\n", len(tt.diagram.Nodes), len(tt.diagram.Connections))
			fmt.Println(strings.Repeat("-", tt.width))
			
			// Print with line numbers for debugging
			lines := strings.Split(output, "\n")
			for i, line := range lines {
				if i < tt.height {
					fmt.Printf("%2d: %s\n", i+1, line)
				}
			}
			fmt.Println(strings.Repeat("-", tt.width))

			// Basic validation
			for _, node := range tt.diagram.Nodes {
				for _, text := range node.Text {
					if !strings.Contains(output, text) {
						t.Errorf("Missing node text: %s", text)
					}
				}
			}

			// Check for box characters (using rounded corners by default)
			if !strings.Contains(output, "╭") || !strings.Contains(output, "╯") {
				t.Error("Missing box drawing characters")
			}
			
			// For connections, check for lines
			if len(tt.diagram.Connections) > 0 {
				if !strings.Contains(output, "─") && !strings.Contains(output, "│") {
					t.Error("Missing connection lines")
				}
			}
		})
	}
}

// TestConnectionPointDebug helps debug connection point calculation
func TestConnectionPointDebug(t *testing.T) {
	// Create a simple diagram
	diagram := &core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"A"}},
			{ID: 2, Text: []string{"B"}},
		},
		Connections: []core.Connection{
			{From: 1, To: 2},
		},
	}
	
	renderer := NewRenderer()
	output, err := renderer.Render(diagram)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}
	
	// Analyze the output character by character
	fmt.Println("\n=== Character Analysis ===")
	lines := strings.Split(output, "\n")
	for y, line := range lines {
		fmt.Printf("Line %d: ", y)
		for x, ch := range line {
			if ch != ' ' {
				fmt.Printf("[%d:%c] ", x, ch)
			}
		}
		fmt.Println()
	}
}
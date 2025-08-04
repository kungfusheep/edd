package main

import (
	"edd/core"
	"fmt"
	"strings"
	"testing"
)

// TestRoutingDebug uses debug mode to visualize obstacles and routing decisions
func TestRoutingDebug(t *testing.T) {
	// Test case 1: App Servers to Cache - should show virtual obstacles
	diagram1 := &core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"App", "Server 1"}, X: 10, Y: 10, Width: 12, Height: 3},
			{ID: 2, Text: []string{"App", "Server 2"}, X: 30, Y: 10, Width: 12, Height: 3},
			{ID: 3, Text: []string{"Cache"}, X: 50, Y: 10, Width: 10, Height: 3},
		},
		Connections: []core.Connection{
			{From: 1, To: 3}, // App Server 1 -> Cache
			{From: 2, To: 3}, // App Server 2 -> Cache
		},
	}

	renderer := NewRenderer()
	renderer.EnableDebug()
	
	output1, err := renderer.Render(diagram1)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	fmt.Println("=== Test 1: Virtual Obstacles Debug ===")
	fmt.Print(output1)
	fmt.Println("=====================================")

	// Test case 2: Complex routing with obstacle between nodes
	diagram2 := &core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"Source"}, X: 5, Y: 10, Width: 10, Height: 3},
			{ID: 2, Text: []string{"Target"}, X: 45, Y: 10, Width: 10, Height: 3},
			{ID: 3, Text: []string{"Obstacle", "In", "Path"}, X: 25, Y: 8, Width: 10, Height: 5},
		},
		Connections: []core.Connection{
			{From: 1, To: 2}, // Should route around obstacle
		},
	}

	output2, err := renderer.Render(diagram2)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	fmt.Println("\n=== Test 2: Obstacle Avoidance Debug ===")
	fmt.Print(output2)
	fmt.Println("======================================")

	// Test case 3: Junction issues reproduction
	diagram3 := &core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"A"}, X: 65, Y: 0, Width: 8, Height: 3},
			{ID: 2, Text: []string{"B"}, X: 75, Y: 0, Width: 8, Height: 3},
			{ID: 3, Text: []string{"C"}, X: 72, Y: 5, Width: 8, Height: 3},
			{ID: 4, Text: []string{"D"}, X: 45, Y: 10, Width: 8, Height: 3},
		},
		Connections: []core.Connection{
			{From: 1, To: 2}, // Horizontal at Y=1
			{From: 3, To: 1}, // Vertical through (72,1)
			{From: 4, To: 3}, // Horizontal that needs to join vertical
		},
	}

	output3, err := renderer.Render(diagram3)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	fmt.Println("\n=== Test 3: Junction Issues Debug ===")
	fmt.Print(output3)
	
	// Analyze specific positions
	lines := strings.Split(output3, "\n")
	if len(lines) > 1 && len(lines[1]) > 72 {
		fmt.Printf("\nCharacter at (72,1): %c\n", lines[1][72])
	}
	fmt.Println("====================================")
}

// TestVirtualObstacleAuthority tests if virtual obstacles are truly authoritative
func TestVirtualObstacleAuthority(t *testing.T) {
	// Create a scenario where virtual obstacles should force specific routing
	diagram := &core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"Left"}, X: 10, Y: 10, Width: 8, Height: 3},
			{ID: 2, Text: []string{"Right"}, X: 40, Y: 10, Width: 8, Height: 3},
			{ID: 3, Text: []string{"Middle", "Block"}, X: 24, Y: 9, Width: 10, Height: 5},
		},
		Connections: []core.Connection{
			{From: 1, To: 2}, // Should be forced to go around, not diagonally near Middle
		},
	}

	renderer := NewRenderer()
	renderer.EnableDebug()
	
	output, err := renderer.Render(diagram)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	fmt.Println("=== Virtual Obstacle Authority Test ===")
	fmt.Print(output)
	
	// Check if the path respects virtual obstacles
	// The path should not come within 3 units of the Middle node's corners
	lines := strings.Split(output, "\n")
	violations := 0
	
	// Check around the Middle node for path violations
	for y := 8; y <= 14; y++ {
		if y < len(lines) {
			for x := 23; x <= 35; x++ {
				if x < len(lines[y]) {
					ch := rune(lines[y][x])
					if ch == '*' || ch == '─' || ch == '│' || ch == '┐' || ch == '└' || ch == '┘' || ch == '┌' {
					// Check if this is too close to a corner
					if (x >= 23 && x <= 25 && y >= 8 && y <= 10) || // Top-left corner area
					   (x >= 32 && x <= 35 && y >= 8 && y <= 10) || // Top-right corner area
					   (x >= 23 && x <= 25 && y >= 12 && y <= 14) || // Bottom-left corner area
					   (x >= 32 && x <= 35 && y >= 12 && y <= 14) { // Bottom-right corner area
						violations++
						fmt.Printf("Potential virtual obstacle violation at (%d,%d): %c\n", x, y, ch)
					}
					}
				}
			}
		}
	}
	
	if violations > 0 {
		fmt.Printf("\nFound %d potential virtual obstacle violations\n", violations)
	}
	fmt.Println("=====================================")
}
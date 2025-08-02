package connections

import (
	"edd/canvas"
	"edd/core"
	"edd/pathfinding"
	"edd/rendering"
	"fmt"
	"strings"
	"testing"
)

// TestVisualConnectionRouting creates visual examples of connection routing
func TestVisualConnectionRouting(t *testing.T) {
	tests := []struct {
		name        string
		nodes       []core.Node
		connections []core.Connection
		arrowConfig func(*ArrowConfig)
		width       int
		height      int
	}{
		{
			name: "Simple Two Nodes",
			nodes: []core.Node{
				{ID: 1, X: 5, Y: 5, Width: 8, Height: 3, Text: []string{"Node A"}},
				{ID: 2, X: 20, Y: 5, Width: 8, Height: 3, Text: []string{"Node B"}},
			},
			connections: []core.Connection{
				{From: 1, To: 2},
			},
			width:  35,
			height: 12,
		},
		{
			name: "Three Nodes Triangle",
			nodes: []core.Node{
				{ID: 1, X: 5, Y: 3, Width: 8, Height: 3, Text: []string{"Top"}},
				{ID: 2, X: 2, Y: 10, Width: 8, Height: 3, Text: []string{"Left"}},
				{ID: 3, X: 15, Y: 10, Width: 8, Height: 3, Text: []string{"Right"}},
			},
			connections: []core.Connection{
				{From: 1, To: 2},
				{From: 2, To: 3},
				{From: 3, To: 1},
			},
			width:  25,
			height: 16,
		},
		{
			name: "Multiple Connections",
			nodes: []core.Node{
				{ID: 1, X: 5, Y: 5, Width: 10, Height: 5, Text: []string{"Server", "Main"}},
				{ID: 2, X: 25, Y: 5, Width: 10, Height: 5, Text: []string{"Client", "App"}},
			},
			connections: []core.Connection{
				{From: 1, To: 2},
				{From: 1, To: 2},
				{From: 2, To: 1},
			},
			arrowConfig: func(ac *ArrowConfig) {
				ac.SetArrowType(2, 1, ArrowStart)
			},
			width:  40,
			height: 15,
		},
		{
			name: "Self Loop",
			nodes: []core.Node{
				{ID: 1, X: 10, Y: 5, Width: 12, Height: 4, Text: []string{"Recursive", "Process"}},
			},
			connections: []core.Connection{
				{From: 1, To: 1},
			},
			width:  30,
			height: 12,
		},
		{
			name: "Complex Layout",
			nodes: []core.Node{
				{ID: 1, X: 5, Y: 5, Width: 8, Height: 3, Text: []string{"A"}},
				{ID: 2, X: 20, Y: 3, Width: 8, Height: 3, Text: []string{"B"}},
				{ID: 3, X: 35, Y: 5, Width: 8, Height: 3, Text: []string{"C"}},
				{ID: 4, X: 20, Y: 12, Width: 8, Height: 3, Text: []string{"D"}},
			},
			connections: []core.Connection{
				{From: 1, To: 2},
				{From: 2, To: 3},
				{From: 1, To: 4},
				{From: 4, To: 3},
				{From: 2, To: 4},
			},
			width:  48,
			height: 18,
		},
		{
			name: "Stress Test - Small Node Many Connections",
			nodes: []core.Node{
				{ID: 1, X: 5, Y: 8, Width: 4, Height: 3, Text: []string{"A"}},
				{ID: 2, X: 15, Y: 8, Width: 4, Height: 3, Text: []string{"B"}},
			},
			connections: []core.Connection{
				{From: 1, To: 2},
				{From: 1, To: 2},
				{From: 1, To: 2},
				{From: 1, To: 2},
				{From: 2, To: 1},
				{From: 2, To: 1},
			},
			width:  25,
			height: 16,
		},
		{
			name: "Obstacle Padding Test",
			nodes: []core.Node{
				{ID: 1, X: 3, Y: 5, Width: 6, Height: 3, Text: []string{"Start"}},
				{ID: 2, X: 15, Y: 2, Width: 6, Height: 3, Text: []string{"Block"}},
				{ID: 3, X: 27, Y: 5, Width: 6, Height: 3, Text: []string{"End"}},
			},
			connections: []core.Connection{
				{From: 1, To: 3}, // Should route around block with padding
			},
			width:  36,
			height: 12,
		},
		{
			name: "Adaptive Self-Loops",
			nodes: []core.Node{
				{ID: 1, X: 5, Y: 8, Width: 16, Height: 4, Text: []string{"Wide Node"}}, // Wide node
				{ID: 2, X: 25, Y: 6, Width: 6, Height: 8, Text: []string{"Tall", "Node"}}, // Tall node
			},
			connections: []core.Connection{
				{From: 1, To: 1}, // Self-loop on wide node
				{From: 2, To: 2}, // Self-loop on tall node
			},
			width:  36,
			height: 20,
		},
		{
			name: "Connection Bundling Test",
			nodes: []core.Node{
				{ID: 1, X: 5, Y: 8, Width: 12, Height: 6, Text: []string{"Source", "System"}},
				{ID: 2, X: 28, Y: 8, Width: 12, Height: 6, Text: []string{"Target", "System"}},
			},
			connections: []core.Connection{
				{From: 1, To: 2},
				{From: 1, To: 2},
				{From: 1, To: 2},
				{From: 1, To: 2},
				{From: 1, To: 2}, // 5 connections should trigger bundling
			},
			width:  45,
			height: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create pathfinder and router
			pf := pathfinding.NewSmartPathFinder(pathfinding.PathCost{
				StraightCost:  10,
				TurnCost:      20,
				ProximityCost: -5, // Prefer paths that hug obstacles
			})
			router := NewRouter(pf)

			// Create canvas
			c := canvas.NewMatrixCanvas(tt.width, tt.height)

			// Create renderer
			caps := rendering.TerminalCapabilities{UnicodeLevel: rendering.UnicodeFull}
			renderer := rendering.NewPathRenderer(caps)
			renderer.SetRenderMode(rendering.RenderModePreserveCorners)

			// Draw nodes first
			for _, node := range tt.nodes {
				// Draw box
				boxPath := core.Path{
					Points: []core.Point{
						{X: node.X, Y: node.Y},
						{X: node.X + node.Width - 1, Y: node.Y},
						{X: node.X + node.Width - 1, Y: node.Y + node.Height - 1},
						{X: node.X, Y: node.Y + node.Height - 1},
						{X: node.X, Y: node.Y},
					},
				}
				renderer.RenderPath(c, boxPath, false)

				// Place text
				for i, line := range node.Text {
					y := node.Y + 1 + i
					x := node.X + 1
					for _, ch := range line {
						if x < node.X+node.Width-1 {
							c.Set(core.Point{X: x, Y: y}, ch)
							x++
						}
					}
				}
			}

			// Route connections
			paths, err := router.RouteConnections(tt.connections, tt.nodes)
			if err != nil {
				t.Errorf("Failed to route connections: %v", err)
				return
			}

			// Apply arrow configuration
			arrowConfig := NewArrowConfig()
			if tt.arrowConfig != nil {
				tt.arrowConfig(arrowConfig)
			}
			connectionsWithArrows := ApplyArrowConfig(tt.connections, paths, arrowConfig)

			// Handle self-loops specially
			for i, conn := range tt.connections {
				if conn.From == conn.To {
					// Find the node
					var node *core.Node
					for j := range tt.nodes {
						if tt.nodes[j].ID == conn.From {
							node = &tt.nodes[j]
							break
						}
					}
					if node != nil {
						paths[i] = HandleSelfLoops(conn, node)
					}
				}
			}

			// Draw connections
			for _, cwa := range connectionsWithArrows {
				hasArrow := cwa.ArrowType == ArrowEnd || cwa.ArrowType == ArrowBoth
				renderer.RenderPath(c, cwa.Path, hasArrow)
			}

			// Display result
			fmt.Printf("\n=== %s ===\n", tt.name)
			fmt.Printf("Nodes: %d, Connections: %d\n", len(tt.nodes), len(tt.connections))
			fmt.Println(strings.Repeat("-", tt.width))
			fmt.Print(c.String())
			fmt.Println(strings.Repeat("-", tt.width))
		})
	}
}

// TestConnectionRouterIntegration tests the full integration of connection routing
func TestConnectionRouterIntegration(t *testing.T) {
	// Create a more complex scenario
	nodes := []core.Node{
		{ID: 1, X: 5, Y: 5, Width: 12, Height: 4},
		{ID: 2, X: 25, Y: 3, Width: 12, Height: 4},
		{ID: 3, X: 45, Y: 5, Width: 12, Height: 4},
		{ID: 4, X: 25, Y: 15, Width: 12, Height: 4},
	}

	connections := []core.Connection{
		{From: 1, To: 2},
		{From: 2, To: 3},
		{From: 3, To: 4},
		{From: 4, To: 1},
		{From: 2, To: 4},
		{From: 1, To: 3},
	}

	// Create pathfinder with specific costs
	pf := pathfinding.NewSmartPathFinder(pathfinding.PathCost{
		StraightCost:  10,
		TurnCost:      20,
		ProximityCost: -5,
	})
	
	// Add caching for performance
	cachedPf := pathfinding.NewCachedPathFinder(pf, 100)
	router := NewRouter(cachedPf)

	// Route all connections
	paths, err := router.RouteConnections(connections, nodes)
	if err != nil {
		t.Fatalf("Failed to route connections: %v", err)
	}

	// Verify we got paths for all connections
	if len(paths) != len(connections) {
		t.Errorf("Expected %d paths, got %d", len(connections), len(paths))
	}

	// Group connections and optimize
	groups := GroupConnections(connections)
	t.Logf("Found %d connection groups", len(groups))

	for _, group := range groups {
		t.Logf("Group %s has %d connections", group.Key, len(group.Connections))
		
		// Optimize grouped paths
		optimizedPaths, err := OptimizeGroupedPaths(group, nodes, router)
		if err != nil {
			t.Errorf("Failed to optimize group %s: %v", group.Key, err)
			continue
		}

		// Verify optimization worked
		if len(optimizedPaths) != len(group.Indices) {
			t.Errorf("Group %s: expected %d optimized paths, got %d", 
				group.Key, len(group.Indices), len(optimizedPaths))
		}
	}
}
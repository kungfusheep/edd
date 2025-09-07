package pathfinding

import (
	"edd/diagram"
	"edd/pathfinding"
	"fmt"
	"strings"
	"testing"
)

// TestLayoutQualityEvaluation evaluates various layout scenarios for visual quality
func TestLayoutQualityEvaluation(t *testing.T) {
	tests := []struct {
		name        string
		nodes       []diagram.Node
		connections []diagram.Connection
		width       int
		height      int
		description string
	}{
		{
			name: "Edge_Spreading_Small_Node",
			nodes: []diagram.Node{
				{ID: 1, X: 5, Y: 5, Width: 4, Height: 3, Text: []string{"A"}},
				{ID: 2, X: 20, Y: 5, Width: 4, Height: 3, Text: []string{"B"}},
			},
			connections: []diagram.Connection{
				{From: 1, To: 2},
				{From: 1, To: 2},
				{From: 1, To: 2},
				{From: 1, To: 2},
			},
			width:       30,
			height:      12,
			description: "Multiple connections on small nodes - testing spread limits",
		},
		{
			name: "Bidirectional_Overlap",
			nodes: []diagram.Node{
				{ID: 1, X: 5, Y: 5, Width: 8, Height: 4, Text: []string{"Server"}},
				{ID: 2, X: 20, Y: 5, Width: 8, Height: 4, Text: []string{"Client"}},
			},
			connections: []diagram.Connection{
				{From: 1, To: 2},
				{From: 2, To: 1},
				{From: 1, To: 2},
				{From: 2, To: 1},
			},
			width:       35,
			height:      14,
			description: "Bidirectional connections - potential overlap issues",
		},
		{
			name: "Dense_Network",
			nodes: []diagram.Node{
				{ID: 1, X: 10, Y: 5, Width: 6, Height: 3, Text: []string{"A"}},
				{ID: 2, X: 25, Y: 5, Width: 6, Height: 3, Text: []string{"B"}},
				{ID: 3, X: 10, Y: 15, Width: 6, Height: 3, Text: []string{"C"}},
				{ID: 4, X: 25, Y: 15, Width: 6, Height: 3, Text: []string{"D"}},
			},
			connections: []diagram.Connection{
				{From: 1, To: 2},
				{From: 1, To: 3},
				{From: 1, To: 4},
				{From: 2, To: 3},
				{From: 2, To: 4},
				{From: 3, To: 4},
			},
			width:       40,
			height:      22,
			description: "Fully connected graph - testing crossing minimization",
		},
		{
			name: "Vertical_vs_Horizontal_Preference",
			nodes: []diagram.Node{
				{ID: 1, X: 10, Y: 10, Width: 8, Height: 4, Text: []string{"Center"}},
				{ID: 2, X: 10, Y: 2, Width: 8, Height: 3, Text: []string{"Top"}},
				{ID: 3, X: 25, Y: 10, Width: 8, Height: 4, Text: []string{"Right"}},
				{ID: 4, X: 10, Y: 20, Width: 8, Height: 3, Text: []string{"Bottom"}},
				{ID: 5, X: 1, Y: 10, Width: 6, Height: 4, Text: []string{"Left"}},
			},
			connections: []diagram.Connection{
				{From: 1, To: 2},
				{From: 1, To: 3},
				{From: 1, To: 4},
				{From: 1, To: 5},
			},
			width:       35,
			height:      25,
			description: "Testing connection point selection preferences",
		},
		{
			name: "Self_Loop_Positioning",
			nodes: []diagram.Node{
				{ID: 1, X: 5, Y: 5, Width: 10, Height: 4, Text: []string{"Process"}},
				{ID: 2, X: 25, Y: 5, Width: 10, Height: 4, Text: []string{"Handler"}},
			},
			connections: []diagram.Connection{
				{From: 1, To: 1},
				{From: 2, To: 2},
				{From: 1, To: 2},
			},
			width:       40,
			height:      15,
			description: "Self-loops with other connections",
		},
		{
			name: "Obstacle_Avoidance_Padding",
			nodes: []diagram.Node{
				{ID: 1, X: 5, Y: 5, Width: 8, Height: 3, Text: []string{"Start"}},
				{ID: 2, X: 25, Y: 5, Width: 8, Height: 3, Text: []string{"End"}},
				{ID: 3, X: 15, Y: 5, Width: 6, Height: 3, Text: []string{"Block"}},
			},
			connections: []diagram.Connection{
				{From: 1, To: 2},
			},
			width:       35,
			height:      12,
			description: "Testing path routing around obstacles without padding",
		},
		{
			name: "Close_Nodes_Connection",
			nodes: []diagram.Node{
				{ID: 1, X: 5, Y: 5, Width: 8, Height: 3, Text: []string{"Node1"}},
				{ID: 2, X: 14, Y: 5, Width: 8, Height: 3, Text: []string{"Node2"}},
			},
			connections: []diagram.Connection{
				{From: 1, To: 2},
			},
			width:       25,
			height:      12,
			description: "Adjacent nodes - minimal space for connections",
		},
		{
			name: "Long_Distance_Routing",
			nodes: []diagram.Node{
				{ID: 1, X: 2, Y: 2, Width: 6, Height: 3, Text: []string{"A"}},
				{ID: 2, X: 40, Y: 20, Width: 6, Height: 3, Text: []string{"B"}},
				{ID: 3, X: 20, Y: 10, Width: 8, Height: 3, Text: []string{"Block"}},
			},
			connections: []diagram.Connection{
				{From: 1, To: 2},
			},
			width:       48,
			height:      25,
			description: "Long distance connection with obstacle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create pathfinder and router
			pf := pathfinding.NewSmartPathFinder(pathfinding.PathCost{
				StraightCost:  10,
				TurnCost:      20,
				ProximityCost: -5,
			})
			router := NewRouter(pf)

			// Create canvas
			c := render.NewMatrixCanvas(tt.width, tt.height)

			// Create renderer
			caps := render.TerminalCapabilities{UnicodeLevel: render.UnicodeFull}
			renderer := render.NewPathRenderer(caps)
			renderer.SetRenderMode(render.RenderModePreserveCorners)

			// Draw nodes
			for _, node := range tt.nodes {
				// Draw box
				boxPath := diagram.Path{
					Points: []diagram.Point{
						{X: node.X, Y: node.Y},
						{X: node.X + node.Width - 1, Y: node.Y},
						{X: node.X + node.Width - 1, Y: node.Y + node.Height - 1},
						{X: node.X, Y: node.Y + node.Height - 1},
						{X: node.X, Y: node.Y},
					},
				}
				renderer.RenderPath(c, boxPath, false)

				// Place text
				if len(node.Text) > 0 && len(node.Text[0]) > 0 {
					y := node.Y + node.Height/2
					x := node.X + 1
					for _, ch := range node.Text[0] {
						if x < node.X+node.Width-1 {
							c.Set(diagram.Point{X: x, Y: y}, ch)
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

			// Handle self-loops
			for i, conn := range tt.connections {
				if conn.From == conn.To {
					var node *diagram.Node
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
			for _, path := range paths {
				renderer.RenderPath(c, path, true)
			}

			// Display result
			fmt.Printf("\n=== %s ===\n", tt.name)
			fmt.Printf("Description: %s\n", tt.description)
			fmt.Printf("Nodes: %d, Connections: %d\n", len(tt.nodes), len(tt.connections))
			fmt.Println(strings.Repeat("-", tt.width))
			fmt.Print(c.String())
			fmt.Println(strings.Repeat("-", tt.width))
			
			// Visual quality analysis
			analyzeVisualQuality(t, tt.name, c, tt.nodes, tt.connections, paths)
		})
	}
}

// analyzeVisualQuality performs automated visual quality checks
func analyzeVisualQuality(t *testing.T, name string, c *render.MatrixCanvas, nodes []diagram.Node, connections []diagram.Connection, paths map[int]diagram.Path) {
	issues := []string{}

	// Check 1: Connection overlap at node boundaries
	connectionPoints := make(map[string]int)
	for _, path := range paths {
		if len(path.Points) > 0 {
			start := fmt.Sprintf("%d,%d", path.Points[0].X, path.Points[0].Y)
			end := fmt.Sprintf("%d,%d", path.Points[len(path.Points)-1].X, path.Points[len(path.Points)-1].Y)
			connectionPoints[start]++
			connectionPoints[end]++
		}
	}
	
	for point, count := range connectionPoints {
		if count > 2 {
			issues = append(issues, fmt.Sprintf("High connection density at point %s: %d connections", point, count))
		}
	}

	// Check 2: Path segments touching node edges (except endpoints)
	for pathIdx, path := range paths {
		for i := 1; i < len(path.Points)-1; i++ {
			pt := path.Points[i]
			for _, node := range nodes {
				// Check if point is on node edge
				onLeft := pt.X == node.X && pt.Y >= node.Y && pt.Y < node.Y+node.Height
				onRight := pt.X == node.X+node.Width-1 && pt.Y >= node.Y && pt.Y < node.Y+node.Height
				onTop := pt.Y == node.Y && pt.X >= node.X && pt.X < node.X+node.Width
				onBottom := pt.Y == node.Y+node.Height-1 && pt.X >= node.X && pt.X < node.X+node.Width
				
				if onLeft || onRight || onTop || onBottom {
					// Skip if this is the source or target node
					conn := connections[pathIdx]
					if node.ID != conn.From && node.ID != conn.To {
						issues = append(issues, fmt.Sprintf("Path %d touches node %d edge unnecessarily", pathIdx, node.ID))
					}
				}
			}
		}
	}

	// Check 3: Self-loop quality
	for i, conn := range connections {
		if conn.From == conn.To {
			path := paths[i]
			if len(path.Points) < 4 {
				issues = append(issues, fmt.Sprintf("Self-loop for node %d has too few points (%d)", conn.From, len(path.Points)))
			}
			// Check if loop extends far enough
			var maxExtension int
			for _, pt := range path.Points {
				for _, node := range nodes {
					if node.ID == conn.From {
						rightExtension := pt.X - (node.X + node.Width)
						if rightExtension > maxExtension {
							maxExtension = rightExtension
						}
					}
				}
			}
			if maxExtension < 2 {
				issues = append(issues, fmt.Sprintf("Self-loop for node %d doesn't extend far enough (only %d units)", conn.From, maxExtension))
			}
		}
	}

	// Report issues
	if len(issues) > 0 {
		t.Logf("Visual quality issues for %s:", name)
		for _, issue := range issues {
			t.Logf("  - %s", issue)
		}
	} else {
		t.Logf("No visual quality issues detected for %s", name)
	}
}
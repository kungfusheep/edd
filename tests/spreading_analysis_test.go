package tests

import (
	"edd/diagram"
	"edd/layout"
	"edd/pathfinding"
	"edd/render"
	"fmt"
	"strings"
	"testing"
)

// TestSpreadingAnalysis examines the connection spreading behavior in detail
func TestSpreadingAnalysis(t *testing.T) {
	tests := []struct {
		name        string
		nodeWidth   int
		nodeHeight  int
		connCount   int
		description string
	}{
		{
			name:        "Small_Node_2_Connections",
			nodeWidth:   6,
			nodeHeight:  3,
			connCount:   2,
			description: "Small node with 2 connections",
		},
		{
			name:        "Small_Node_3_Connections",
			nodeWidth:   6,
			nodeHeight:  3,
			connCount:   3,
			description: "Small node with 3 connections - potential overlap",
		},
		{
			name:        "Medium_Node_4_Connections",
			nodeWidth:   10,
			nodeHeight:  5,
			connCount:   4,
			description: "Medium node with 4 connections",
		},
		{
			name:        "Large_Node_6_Connections",
			nodeWidth:   14,
			nodeHeight:  7,
			connCount:   6,
			description: "Large node with 6 connections",
		},
		{
			name:        "Wide_Node_Multiple",
			nodeWidth:   20,
			nodeHeight:  4,
			connCount:   5,
			description: "Wide but short node",
		},
		{
			name:        "Tall_Node_Multiple",
			nodeWidth:   6,
			nodeHeight:  10,
			connCount:   5,
			description: "Tall but narrow node",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create two nodes
			nodes := []diagram.Node{
				{ID: 1, X: 5, Y: 5, Width: tt.nodeWidth, Height: tt.nodeHeight, Text: []string{"A"}},
				{ID: 2, X: 25, Y: 5, Width: tt.nodeWidth, Height: tt.nodeHeight, Text: []string{"B"}},
			}

			// Create connections
			connections := make([]diagram.Connection, tt.connCount)
			for i := 0; i < tt.connCount; i++ {
				connections[i] = diagram.Connection{From: 1, To: 2}
			}

			// Calculate canvas size
			width := 35 + tt.nodeWidth
			height := 10 + tt.nodeHeight

			// Create pathfinder and router
			pf := pathfinding.NewSmartPathFinder(pathfinding.PathCost{
				StraightCost:  10,
				TurnCost:      20,
				ProximityCost: -5,
			})
			router := pathfinding.NewRouter(pf)

			// Create canvas
			c := render.NewMatrixCanvas(width, height)

			// Create renderer
			caps := render.TerminalCapabilities{UnicodeLevel: render.UnicodeFull}
			renderer := render.NewPathRenderer(caps)
			renderer.SetRenderMode(render.RenderModePreserveCorners)

			// Draw nodes
			for _, node := range nodes {
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

				// Place text in center
				if len(node.Text) > 0 {
					y := node.Y + node.Height/2
					x := node.X + node.Width/2 - len(node.Text[0])/2
					for _, ch := range node.Text[0] {
						if x >= node.X && x < node.X+node.Width {
							c.Set(diagram.Point{X: x, Y: y}, ch)
							x++
						}
					}
				}
			}

			// Route connections
			paths, err := router.RouteConnections(connections, nodes)
			if err != nil {
				t.Errorf("Failed to route connections: %v", err)
				return
			}

			// Draw connections
			for _, path := range paths {
				renderer.RenderPath(c, path, true)
			}

			// Display result
			fmt.Printf("\n=== %s ===\n", tt.name)
			fmt.Printf("Description: %s\n", tt.description)
			fmt.Printf("Node size: %dx%d, Connections: %d\n", tt.nodeWidth, tt.nodeHeight, tt.connCount)
			fmt.Println(strings.Repeat("-", width))
			fmt.Print(c.String())
			fmt.Println(strings.Repeat("-", width))

			// Analyze spreading
			analyzeSpreadingQuality(t, tt.name, nodes, paths, tt.connCount)
		})
	}
}

// analyzeSpreadingQuality checks the quality of connection spreading
func analyzeSpreadingQuality(t *testing.T, name string, nodes []diagram.Node, paths map[int]diagram.Path, expectedCount int) {
	if len(paths) != expectedCount {
		t.Errorf("Expected %d paths, got %d", expectedCount, len(paths))
		return
	}

	// Collect all starting points
	startPoints := make([]diagram.Point, 0, len(paths))
	endPoints := make([]diagram.Point, 0, len(paths))
	
	for _, path := range paths {
		if len(path.Points) > 0 {
			startPoints = append(startPoints, path.Points[0])
			endPoints = append(endPoints, path.Points[len(path.Points)-1])
		}
	}

	// Check spacing between connection points
	t.Logf("Connection point analysis for %s:", name)
	
	// Analyze start points
	if len(startPoints) > 1 {
		minDist := 999
		maxDist := 0
		totalDist := 0
		count := 0
		
		for i := 0; i < len(startPoints)-1; i++ {
			for j := i + 1; j < len(startPoints); j++ {
				dist := layout.Abs(startPoints[i].Y - startPoints[j].Y) + layout.Abs(startPoints[i].X - startPoints[j].X)
				if dist < minDist {
					minDist = dist
				}
				if dist > maxDist {
					maxDist = dist
				}
				totalDist += dist
				count++
			}
		}
		
		avgDist := float64(totalDist) / float64(count)
		t.Logf("  Start points - Min distance: %d, Max distance: %d, Avg: %.1f", minDist, maxDist, avgDist)
		
		if minDist == 0 {
			t.Logf("  WARNING: Some start points are overlapping!")
		}
		if maxDist-minDist > 2 {
			t.Logf("  WARNING: Uneven spacing detected (difference: %d)", maxDist-minDist)
		}
	}

	// Check if spreading is proportional to node size
	node := nodes[0]
	if len(startPoints) > 1 {
		// Check if we're using the full height/width for spreading
		yValues := make(map[int]bool)
		xValues := make(map[int]bool)
		for _, pt := range startPoints {
			yValues[pt.Y] = true
			xValues[pt.X] = true
		}
		
		spreadRange := 0
		if len(yValues) > len(xValues) {
			// Vertical spreading
			spreadRange = len(yValues)
			utilization := float64(spreadRange) / float64(node.Height) * 100
			t.Logf("  Vertical spreading: using %d of %d height units (%.0f%% utilization)", 
				spreadRange, node.Height, utilization)
		} else {
			// Horizontal spreading
			spreadRange = len(xValues)
			utilization := float64(spreadRange) / float64(node.Width) * 100
			t.Logf("  Horizontal spreading: using %d of %d width units (%.0f%% utilization)", 
				spreadRange, node.Width, utilization)
		}
	}
}

// TestBidirectionalSpacing specifically tests spacing between bidirectional connections
func TestBidirectionalSpacing(t *testing.T) {
	// Create test scenario with bidirectional connections
	nodes := []diagram.Node{
		{ID: 1, X: 5, Y: 5, Width: 12, Height: 6, Text: []string{"Server"}},
		{ID: 2, X: 25, Y: 5, Width: 12, Height: 6, Text: []string{"Client"}},
	}

	connections := []diagram.Connection{
		// Forward connections
		{From: 1, To: 2},
		{From: 1, To: 2},
		{From: 1, To: 2},
		// Backward connections
		{From: 2, To: 1},
		{From: 2, To: 1},
	}

	// Create pathfinder and router
	pf := pathfinding.NewSmartPathFinder(pathfinding.PathCost{
		StraightCost:  10,
		TurnCost:      20,
		ProximityCost: -5,
	})
	router := pathfinding.NewRouter(pf)

	// Create canvas
	c := render.NewMatrixCanvas(45, 16)

	// Create renderer
	caps := render.TerminalCapabilities{UnicodeLevel: render.UnicodeFull}
	renderer := render.NewPathRenderer(caps)
	renderer.SetRenderMode(render.RenderModePreserveCorners)

	// Draw nodes
	for _, node := range nodes {
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
		y := node.Y + node.Height/2
		x := node.X + 2
		for _, ch := range node.Text[0] {
			if x < node.X+node.Width-1 {
				c.Set(diagram.Point{X: x, Y: y}, ch)
				x++
			}
		}
	}

	// Route connections
	paths, err := router.RouteConnections(connections, nodes)
	if err != nil {
		t.Errorf("Failed to route connections: %v", err)
		return
	}

	// Draw connections with arrows
	arrowConfig := pathfinding.NewArrowConfig()
	// Set arrows for backward connections
	arrowConfig.SetArrowType(2, 1, pathfinding.ArrowStart)
	
	connectionsWithArrows := pathfinding.ApplyArrowConfig(connections, paths, arrowConfig)
	for _, cwa := range connectionsWithArrows {
		hasArrow := cwa.ArrowType == pathfinding.ArrowEnd || cwa.ArrowType == pathfinding.ArrowBoth
		renderer.RenderPath(c, cwa.Path, hasArrow)
	}

	// Display result
	fmt.Println("\n=== Bidirectional Connection Spacing ===")
	fmt.Println("Testing separation between forward and backward connections")
	fmt.Println(strings.Repeat("-", 45))
	fmt.Print(c.String())
	fmt.Println(strings.Repeat("-", 45))

	// Analyze bidirectional spacing
	forwardPaths := make([]diagram.Path, 0)
	backwardPaths := make([]diagram.Path, 0)
	
	for i, conn := range connections {
		if conn.From == 1 {
			forwardPaths = append(forwardPaths, paths[i])
		} else {
			backwardPaths = append(backwardPaths, paths[i])
		}
	}

	t.Logf("Forward connections: %d, Backward connections: %d", len(forwardPaths), len(backwardPaths))
	
	// Check for crossing or overlap
	crossings := 0
	for _, fPath := range forwardPaths {
		for _, bPath := range backwardPaths {
			if pathsCross(fPath, bPath) {
				crossings++
			}
		}
	}
	
	if crossings > 0 {
		t.Logf("WARNING: Found %d crossing points between forward and backward paths", crossings)
	} else {
		t.Logf("Good: No crossings detected between forward and backward paths")
	}
}

// pathsCross checks if two paths cross each other (simplified check)
func pathsCross(path1, path2 diagram.Path) bool {
	// Simple check: see if paths share any points except endpoints
	for i := 1; i < len(path1.Points)-1; i++ {
		for j := 1; j < len(path2.Points)-1; j++ {
			if path1.Points[i].X == path2.Points[j].X && 
			   path1.Points[i].Y == path2.Points[j].Y {
				return true
			}
		}
	}
	return false
}
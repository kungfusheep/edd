package pathfinding

import (
	"edd/core"
	"testing"
)

func TestConnectionOptimizer_BasicConnections(t *testing.T) {
	pathFinder := NewSmartPathFinder(DefaultPathCost)
	optimizer := NewConnectionOptimizer(pathFinder)
	
	tests := []struct {
		name     string
		fromNode *core.Node
		toNode   *core.Node
		wantFrom core.Point
		wantTo   core.Point
		desc     string
	}{
		{
			name: "Horizontal connection (left to right)",
			fromNode: &core.Node{
				ID: 1, X: 10, Y: 10, Width: 10, Height: 5,
			},
			toNode: &core.Node{
				ID: 2, X: 30, Y: 10, Width: 10, Height: 5,
			},
			wantFrom: core.Point{X: 20, Y: 12}, // Right side of first node
			wantTo:   core.Point{X: 29, Y: 12}, // Left side of second node
			desc:     "Should connect from right side to left side",
		},
		{
			name: "Vertical connection (top to bottom)",
			fromNode: &core.Node{
				ID: 1, X: 10, Y: 10, Width: 10, Height: 5,
			},
			toNode: &core.Node{
				ID: 2, X: 10, Y: 25, Width: 10, Height: 5,
			},
			wantFrom: core.Point{X: 15, Y: 15}, // Bottom of first node
			wantTo:   core.Point{X: 15, Y: 24}, // Top of second node
			desc:     "Should connect from bottom to top",
		},
		{
			name: "Diagonal connection (prefer horizontal)",
			fromNode: &core.Node{
				ID: 1, X: 10, Y: 10, Width: 10, Height: 5,
			},
			toNode: &core.Node{
				ID: 2, X: 30, Y: 20, Width: 10, Height: 5,
			},
			wantFrom: core.Point{X: 20, Y: 12}, // Right side (horizontal dominates)
			wantTo:   core.Point{X: 29, Y: 22}, // Left side
			desc:     "Should prefer horizontal when dx > dy",
		},
		{
			name: "Diagonal connection (prefer vertical)",
			fromNode: &core.Node{
				ID: 1, X: 10, Y: 10, Width: 10, Height: 5,
			},
			toNode: &core.Node{
				ID: 2, X: 15, Y: 30, Width: 10, Height: 5,
			},
			wantFrom: core.Point{X: 15, Y: 15}, // Bottom (vertical dominates)
			wantTo:   core.Point{X: 20, Y: 29}, // Top
			desc:     "Should prefer vertical when dy > dx",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, to := optimizer.OptimizeConnectionPoints(tt.fromNode, tt.toNode, nil)
			
			if from != tt.wantFrom {
				t.Errorf("Wrong from point: got %v, want %v", from, tt.wantFrom)
			}
			if to != tt.wantTo {
				t.Errorf("Wrong to point: got %v, want %v", to, tt.wantTo)
			}
		})
	}
}

func TestConnectionOptimizer_SelfConnection(t *testing.T) {
	pathFinder := NewSmartPathFinder(DefaultPathCost)
	optimizer := NewConnectionOptimizer(pathFinder)
	
	node := &core.Node{
		ID: 1, X: 10, Y: 10, Width: 10, Height: 6,
	}
	
	from, to := optimizer.OptimizeSelfConnection(node)
	
	// Should exit from right side
	if from.X != 20 {
		t.Errorf("Self connection should exit from right side: got X=%d, want X=20", from.X)
	}
	if from.Y != 12 { // Y + Height/3
		t.Errorf("Self connection exit Y incorrect: got Y=%d, want Y=12", from.Y)
	}
	
	// Should enter from bottom
	if to.Y != 16 {
		t.Errorf("Self connection should enter from bottom: got Y=%d, want Y=16", to.Y)
	}
	if to.X != 13 { // X + Width/3
		t.Errorf("Self connection entry X incorrect: got X=%d, want X=13", to.X)
	}
}

func TestGetConnectionSide(t *testing.T) {
	node := &core.Node{
		ID: 1, X: 10, Y: 10, Width: 10, Height: 10,
	}
	
	tests := []struct {
		point    core.Point
		wantSide Side
	}{
		{core.Point{X: 15, Y: 9}, SideTop},
		{core.Point{X: 20, Y: 15}, SideRight},
		{core.Point{X: 15, Y: 20}, SideBottom},
		{core.Point{X: 9, Y: 15}, SideLeft},
	}
	
	for _, tt := range tests {
		side := GetConnectionSide(tt.point, node)
		if side != tt.wantSide {
			t.Errorf("GetConnectionSide(%v) = %v, want %v", tt.point, side, tt.wantSide)
		}
	}
}

func TestGenerateCandidatePoints(t *testing.T) {
	node := &core.Node{
		ID: 1, X: 10, Y: 10, Width: 5, Height: 5,
	}
	
	tests := []struct {
		side      Side
		wantCount int
		checkY    int // For top/bottom
		checkX    int // For left/right
	}{
		{SideTop, 1, 9, -1},      // 1 point along top edge (width=5, gap=2 on each side)
		{SideRight, 1, -1, 15},   // 1 point along right edge
		{SideBottom, 1, 15, -1},  // 1 point along bottom edge
		{SideLeft, 1, -1, 9},     // 1 point along left edge
	}
	
	for _, tt := range tests {
		points := GenerateCandidatePoints(node, tt.side)
		
		if len(points) != tt.wantCount {
			t.Errorf("Side %v: got %d points, want %d", tt.side, len(points), tt.wantCount)
		}
		
		// Check that all points are on the correct side
		for _, p := range points {
			if tt.checkY != -1 && p.Point.Y != tt.checkY {
				t.Errorf("Side %v: point Y=%d, want Y=%d", tt.side, p.Point.Y, tt.checkY)
			}
			if tt.checkX != -1 && p.Point.X != tt.checkX {
				t.Errorf("Side %v: point X=%d, want X=%d", tt.side, p.Point.X, tt.checkX)
			}
		}
	}
}

func TestConnectionOptimizer_WithObstacles(t *testing.T) {
	pathFinder := NewSmartPathFinder(DefaultPathCost)
	optimizer := NewConnectionOptimizer(pathFinder)
	
	// Two nodes with an obstacle between them
	fromNode := &core.Node{
		ID: 1, X: 10, Y: 10, Width: 10, Height: 10,
	}
	toNode := &core.Node{
		ID: 2, X: 40, Y: 10, Width: 10, Height: 10,
	}
	
	// Obstacle in the middle
	obstacle := &core.Node{
		ID: 3, X: 25, Y: 10, Width: 10, Height: 10,
	}
	
	obstacles := CreateNodeObstacleChecker([]core.Node{*obstacle}, 1)
	
	from, to := optimizer.OptimizeConnectionPoints(fromNode, toNode, obstacles)
	
	// Should still choose right-to-left connection
	if from.X != 20 {
		t.Errorf("Should exit from right side: got X=%d, want X=20", from.X)
	}
	if to.X != 39 {
		t.Errorf("Should enter from left side: got X=%d, want X=39", to.X)
	}
	
	// The pathfinder will handle routing around the obstacle
}
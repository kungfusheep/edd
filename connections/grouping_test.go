package connections

import (
	"edd/core"
	"edd/pathfinding"
	"testing"
)

func TestGroupConnections(t *testing.T) {
	connections := []core.Connection{
		{From: 1, To: 2},
		{From: 1, To: 2},
		{From: 2, To: 3},
		{From: 1, To: 2},
		{From: 3, To: 1},
	}
	
	groups := GroupConnections(connections)
	
	// Should have 3 groups: A->B, B->C, C->A
	if len(groups) != 3 {
		t.Fatalf("Expected 3 groups, got %d", len(groups))
	}
	
	// Find the 1->2 group
	var abGroup *ConnectionGroup
	for i := range groups {
		if groups[i].Key == "1->2" {
			abGroup = &groups[i]
			break
		}
	}
	
	if abGroup == nil {
		t.Fatal("1->2 group not found")
	}
	
	// 1->2 group should have 3 connections
	if len(abGroup.Connections) != 3 {
		t.Errorf("1->2 group has %d connections, want 3", len(abGroup.Connections))
	}
	
	// Check indices are correct
	expectedIndices := []int{0, 1, 3}
	for i, idx := range abGroup.Indices {
		if idx != expectedIndices[i] {
			t.Errorf("Index %d: got %d, want %d", i, idx, expectedIndices[i])
		}
	}
}

func TestSpreadPoints(t *testing.T) {
	node := &core.Node{X: 10, Y: 10, Width: 20, Height: 10}
	basePoint := core.Point{X: 30, Y: 15} // Right side center
	
	tests := []struct {
		name         string
		count        int
		index        int
		isHorizontal bool
		wantOffset   bool
	}{
		{
			name:         "single connection no spread",
			count:        1,
			index:        0,
			isHorizontal: true,
			wantOffset:   false,
		},
		{
			name:         "first of three horizontal",
			count:        3,
			index:        0,
			isHorizontal: true,
			wantOffset:   true,
		},
		{
			name:         "middle of three horizontal",
			count:        3,
			index:        1,
			isHorizontal: true,
			wantOffset:   true,
		},
		{
			name:         "first of two vertical",
			count:        2,
			index:        0,
			isHorizontal: false,
			wantOffset:   true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SpreadPoints(basePoint, tt.count, tt.index, node, tt.isHorizontal)
			
			if tt.wantOffset {
				if tt.isHorizontal {
					// Should have vertical offset
					if result.Y == basePoint.Y {
						t.Error("Expected Y offset for horizontal spread")
					}
					if result.X != basePoint.X {
						t.Error("X should not change for horizontal spread")
					}
				} else {
					// Should have horizontal offset
					if result.X == basePoint.X {
						t.Error("Expected X offset for vertical spread")
					}
					if result.Y != basePoint.Y {
						t.Error("Y should not change for vertical spread")
					}
				}
			} else {
				// No offset expected
				if result != basePoint {
					t.Errorf("Expected no offset, got %v != %v", result, basePoint)
				}
			}
		})
	}
}

func TestHandleSelfLoops(t *testing.T) {
	node := &core.Node{
		ID:     1,
		X:      10,
		Y:      10,
		Width:  10,
		Height: 6,
	}
	
	conn := core.Connection{From: 1, To: 1}
	path := HandleSelfLoops(conn, node)
	
	// Should create a loop with multiple points
	if len(path.Points) < 4 {
		t.Errorf("Self-loop path too short: %d points", len(path.Points))
	}
	
	// First point should be on right side of node
	first := path.Points[0]
	if first.X != node.X+node.Width-1 {
		t.Errorf("Self-loop should start from right side")
	}
	
	// Last point should be on top of node
	last := path.Points[len(path.Points)-1]
	if last.Y != node.Y {
		t.Errorf("Self-loop should end at top")
	}
	
	// Should extend beyond node bounds
	maxX := node.X + node.Width - 1
	hasExtension := false
	for _, p := range path.Points {
		if p.X > maxX {
			hasExtension = true
			break
		}
	}
	if !hasExtension {
		t.Error("Self-loop should extend beyond node bounds")
	}
}

func TestOptimizeGroupedPaths(t *testing.T) {
	// Create pathfinder and router
	pf := pathfinding.NewAStarPathFinder(pathfinding.PathCost{
		StraightCost: 10,
		TurnCost: 5,
	})
	router := NewRouter(pf)
	
	nodes := []core.Node{
		{ID: 1, X: 5, Y: 10, Width: 10, Height: 10},
		{ID: 2, X: 30, Y: 10, Width: 10, Height: 10},
	}
	
	// Create a group with multiple connections
	group := ConnectionGroup{
		Key: "1->2",
		Connections: []core.Connection{
			{From: 1, To: 2},
			{From: 1, To: 2},
			{From: 1, To: 2},
		},
		Indices: []int{0, 1, 2},
	}
	
	paths, err := OptimizeGroupedPaths(group, nodes, router)
	if err != nil {
		t.Fatalf("OptimizeGroupedPaths() error = %v", err)
	}
	
	if len(paths) != 3 {
		t.Errorf("Expected 3 paths, got %d", len(paths))
	}
	
	// Verify paths have different starting/ending Y coordinates
	// (they should be spread out)
	startYs := make(map[int]bool)
	endYs := make(map[int]bool)
	
	for _, path := range paths {
		if len(path.Points) > 0 {
			startYs[path.Points[0].Y] = true
			endYs[path.Points[len(path.Points)-1].Y] = true
		}
	}
	
	// With 3 connections, we should have 3 different Y coordinates
	if len(startYs) != 3 {
		t.Errorf("Expected 3 different start Y coordinates, got %d", len(startYs))
	}
	if len(endYs) != 3 {
		t.Errorf("Expected 3 different end Y coordinates, got %d", len(endYs))
	}
}
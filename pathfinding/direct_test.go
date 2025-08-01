package pathfinding

import (
	"edd/core"
	"testing"
)

func TestDirectPathFinder_StraightLines(t *testing.T) {
	strategies := []struct {
		name     string
		strategy RoutingStrategy
	}{
		{"HorizontalFirst", HorizontalFirst},
		{"VerticalFirst", VerticalFirst},
		{"MiddleSplit", MiddleSplit},
	}
	
	tests := []struct {
		name     string
		start    core.Point
		end      core.Point
		expected int // expected number of points
	}{
		{
			name:     "Horizontal line right",
			start:    core.Point{X: 0, Y: 5},
			end:      core.Point{X: 10, Y: 5},
			expected: 11,
		},
		{
			name:     "Horizontal line left",
			start:    core.Point{X: 10, Y: 5},
			end:      core.Point{X: 0, Y: 5},
			expected: 11,
		},
		{
			name:     "Vertical line down",
			start:    core.Point{X: 5, Y: 0},
			end:      core.Point{X: 5, Y: 10},
			expected: 11,
		},
		{
			name:     "Vertical line up",
			start:    core.Point{X: 5, Y: 10},
			end:      core.Point{X: 5, Y: 0},
			expected: 11,
		},
		{
			name:     "Same point",
			start:    core.Point{X: 5, Y: 5},
			end:      core.Point{X: 5, Y: 5},
			expected: 1,
		},
	}
	
	for _, strategy := range strategies {
		t.Run(strategy.name, func(t *testing.T) {
			finder := NewDirectPathFinder(strategy.strategy)
			
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					path, err := finder.FindPath(tt.start, tt.end, nil)
					if err != nil {
						t.Fatalf("FindPath failed: %v", err)
					}
					
					if len(path.Points) != tt.expected {
						t.Errorf("Expected %d points, got %d", tt.expected, len(path.Points))
					}
					
					// Verify start and end points
					if len(path.Points) > 0 {
						if path.Points[0] != tt.start {
							t.Errorf("First point = %v, want %v", path.Points[0], tt.start)
						}
						if path.Points[len(path.Points)-1] != tt.end {
							t.Errorf("Last point = %v, want %v", path.Points[len(path.Points)-1], tt.end)
						}
					}
				})
			}
		})
	}
}

func TestDirectPathFinder_LShapedPaths(t *testing.T) {
	tests := []struct {
		name        string
		start       core.Point
		end         core.Point
		strategy    RoutingStrategy
		expectedMid core.Point // expected middle point for L-shaped path
	}{
		{
			name:        "HorizontalFirst - down-right",
			start:       core.Point{X: 0, Y: 0},
			end:         core.Point{X: 5, Y: 5},
			strategy:    HorizontalFirst,
			expectedMid: core.Point{X: 5, Y: 0}, // horizontal then vertical
		},
		{
			name:        "VerticalFirst - down-right",
			start:       core.Point{X: 0, Y: 0},
			end:         core.Point{X: 5, Y: 5},
			strategy:    VerticalFirst,
			expectedMid: core.Point{X: 0, Y: 5}, // vertical then horizontal
		},
		{
			name:        "HorizontalFirst - up-left",
			start:       core.Point{X: 10, Y: 10},
			end:         core.Point{X: 5, Y: 5},
			strategy:    HorizontalFirst,
			expectedMid: core.Point{X: 5, Y: 10}, // horizontal then vertical
		},
		{
			name:        "VerticalFirst - up-left",
			start:       core.Point{X: 10, Y: 10},
			end:         core.Point{X: 5, Y: 5},
			strategy:    VerticalFirst,
			expectedMid: core.Point{X: 10, Y: 5}, // vertical then horizontal
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			finder := NewDirectPathFinder(tt.strategy)
			path, err := finder.FindPath(tt.start, tt.end, nil)
			if err != nil {
				t.Fatalf("FindPath failed: %v", err)
			}
			
			// L-shaped paths should have exactly 3 points
			if len(path.Points) != 3 {
				t.Errorf("Expected 3 points for L-shaped path, got %d", len(path.Points))
			}
			
			if len(path.Points) == 3 {
				if path.Points[1] != tt.expectedMid {
					t.Errorf("Middle point = %v, want %v", path.Points[1], tt.expectedMid)
				}
			}
			
			// Verify path cost includes turn penalty
			expectedCost := ManhattanDistance(tt.start, tt.end)*DefaultPathCost.StraightCost + DefaultPathCost.TurnCost
			if path.Cost != expectedCost {
				t.Errorf("Path cost = %d, want %d", path.Cost, expectedCost)
			}
		})
	}
}

func TestDirectPathFinder_MiddleSplit(t *testing.T) {
	finder := NewDirectPathFinder(MiddleSplit)
	
	tests := []struct {
		name   string
		start  core.Point
		end    core.Point
		points int // expected number of points
	}{
		{
			name:   "Wide rectangle",
			start:  core.Point{X: 0, Y: 0},
			end:    core.Point{X: 20, Y: 5},
			points: 4, // start -> mid-x,start-y -> mid-x,end-y -> end
		},
		{
			name:   "Tall rectangle",
			start:  core.Point{X: 0, Y: 0},
			end:    core.Point{X: 5, Y: 20},
			points: 4, // start -> start-x,mid-y -> end-x,mid-y -> end
		},
		{
			name:   "Square diagonal",
			start:  core.Point{X: 0, Y: 0},
			end:    core.Point{X: 10, Y: 10},
			points: 4, // split horizontally since it's square
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := finder.FindPath(tt.start, tt.end, nil)
			if err != nil {
				t.Fatalf("FindPath failed: %v", err)
			}
			
			if len(path.Points) != tt.points {
				t.Errorf("Expected %d points, got %d", tt.points, len(path.Points))
				t.Logf("Path: %s", PathToString(path))
			}
		})
	}
}

func TestSimplifyPath(t *testing.T) {
	tests := []struct {
		name     string
		path     core.Path
		expected int // expected number of points after simplification
	}{
		{
			name: "Straight horizontal line",
			path: core.Path{
				Points: []core.Point{
					{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 2, Y: 0}, {X: 3, Y: 0},
				},
			},
			expected: 2, // just start and end
		},
		{
			name: "L-shaped path",
			path: core.Path{
				Points: []core.Point{
					{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 2, Y: 0}, 
					{X: 2, Y: 1}, {X: 2, Y: 2},
				},
			},
			expected: 3, // start, corner, end
		},
		{
			name: "Complex path with multiple turns",
			path: core.Path{
				Points: []core.Point{
					{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 2, Y: 0}, // horizontal
					{X: 2, Y: 1}, {X: 2, Y: 2},              // vertical
					{X: 3, Y: 2}, {X: 4, Y: 2},              // horizontal
				},
			},
			expected: 4, // start, two corners, end
		},
		{
			name: "Already simplified",
			path: core.Path{
				Points: []core.Point{
					{X: 0, Y: 0}, {X: 5, Y: 0}, {X: 5, Y: 5},
				},
			},
			expected: 3, // no change
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			simplified := SimplifyPath(tt.path)
			
			if len(simplified.Points) != tt.expected {
				t.Errorf("Expected %d points after simplification, got %d", 
					tt.expected, len(simplified.Points))
				t.Logf("Original: %s", PathToString(tt.path))
				t.Logf("Simplified: %s", PathToString(simplified))
			}
			
			// Verify start and end points are preserved
			if len(simplified.Points) > 0 && len(tt.path.Points) > 0 {
				if simplified.Points[0] != tt.path.Points[0] {
					t.Error("Start point changed during simplification")
				}
				if simplified.Points[len(simplified.Points)-1] != tt.path.Points[len(tt.path.Points)-1] {
					t.Error("End point changed during simplification")
				}
			}
		})
	}
}

func TestManhattanDistance(t *testing.T) {
	tests := []struct {
		p1       core.Point
		p2       core.Point
		expected int
	}{
		{core.Point{0, 0}, core.Point{0, 0}, 0},
		{core.Point{0, 0}, core.Point{3, 4}, 7},
		{core.Point{3, 4}, core.Point{0, 0}, 7},
		{core.Point{-5, -5}, core.Point{5, 5}, 20},
		{core.Point{10, 0}, core.Point{0, 10}, 20},
	}
	
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			distance := ManhattanDistance(tt.p1, tt.p2)
			if distance != tt.expected {
				t.Errorf("ManhattanDistance(%v, %v) = %d, want %d", 
					tt.p1, tt.p2, distance, tt.expected)
			}
		})
	}
}

func TestGetDirection(t *testing.T) {
	tests := []struct {
		p1       core.Point
		p2       core.Point
		expected Direction
	}{
		{core.Point{5, 5}, core.Point{5, 3}, North},
		{core.Point{5, 5}, core.Point{7, 5}, East},
		{core.Point{5, 5}, core.Point{5, 7}, South},
		{core.Point{5, 5}, core.Point{3, 5}, West},
		{core.Point{5, 5}, core.Point{5, 5}, None},
		{core.Point{5, 5}, core.Point{7, 7}, None}, // diagonal
	}
	
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			dir := GetDirection(tt.p1, tt.p2)
			if dir != tt.expected {
				t.Errorf("GetDirection(%v, %v) = %v, want %v", 
					tt.p1, tt.p2, dir, tt.expected)
			}
		})
	}
}
package pathfinding

import (
	"edd/core"
	"strings"
	"testing"
)

func TestAStarPathFinder_SimplePaths(t *testing.T) {
	finder := NewAStarPathFinder(DefaultPathCost)
	
	tests := []struct {
		name      string
		start     core.Point
		end       core.Point
		obstacles string // ASCII representation of obstacles
		minLength int    // minimum expected path length
	}{
		{
			name:      "Direct horizontal path",
			start:     core.Point{0, 0},
			end:       core.Point{5, 0},
			obstacles: "",
			minLength: 6,
		},
		{
			name:      "Direct vertical path",
			start:     core.Point{0, 0},
			end:       core.Point{0, 5},
			obstacles: "",
			minLength: 6,
		},
		{
			name:      "L-shaped path",
			start:     core.Point{0, 0},
			end:       core.Point{5, 5},
			obstacles: "",
			minLength: 11,
		},
		{
			name:  "Path around obstacle",
			start: core.Point{0, 2},
			end:   core.Point{4, 2},
			obstacles: `
.....
.....
.XX..
.....
.....`,
			minLength: 7, // Must go around
		},
		{
			name:  "Path through maze",
			start: core.Point{0, 0},
			end:   core.Point{4, 4},
			obstacles: `
.XXX.
...X.
.X...
.XXX.
.....`,
			minLength: 9,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obstacles := parseObstacleMap(tt.obstacles)
			path, err := finder.FindPath(tt.start, tt.end, obstacles)
			
			if err != nil {
				t.Fatalf("FindPath failed: %v", err)
			}
			
			// Check path validity
			if len(path.Points) < tt.minLength {
				t.Errorf("Path too short: got %d points, want at least %d", 
					len(path.Points), tt.minLength)
			}
			
			// Verify start and end
			if path.Points[0] != tt.start {
				t.Errorf("Path doesn't start at %v", tt.start)
			}
			if path.Points[len(path.Points)-1] != tt.end {
				t.Errorf("Path doesn't end at %v", tt.end)
			}
			
			// Verify path is continuous
			for i := 1; i < len(path.Points); i++ {
				dist := ManhattanDistance(path.Points[i-1], path.Points[i])
				if dist != 1 {
					t.Errorf("Path not continuous at %d: %v -> %v", 
						i, path.Points[i-1], path.Points[i])
				}
			}
			
			// Verify no obstacles in path
			for _, p := range path.Points {
				if obstacles(p) {
					t.Errorf("Path goes through obstacle at %v", p)
				}
			}
		})
	}
}

func TestAStarPathFinder_NoPath(t *testing.T) {
	finder := NewAStarPathFinder(DefaultPathCost)
	
	tests := []struct {
		name      string
		start     core.Point
		end       core.Point
		obstacles string
	}{
		{
			name:  "End blocked",
			start: core.Point{0, 0},
			end:   core.Point{2, 0},
			obstacles: `
..X`,
		},
		{
			name:  "Start blocked",
			start: core.Point{0, 0},
			end:   core.Point{2, 0},
			obstacles: `
X..`,
		},
		{
			name:  "Completely blocked",
			start: core.Point{0, 0},
			end:   core.Point{2, 2},
			obstacles: `
.....
.XXX.
.X.X.
.XXX.
.....`,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obstacles := parseObstacleMap(tt.obstacles)
			_, err := finder.FindPath(tt.start, tt.end, obstacles)
			
			if err == nil {
				t.Error("Expected error for impossible path, got nil")
			}
		})
	}
}

func TestAStarPathFinder_CostOptimization(t *testing.T) {
	// Test that A* finds optimal paths considering turn costs
	costs := PathCost{
		StraightCost: 10,
		TurnCost:     50, // High turn cost
		CrossingCost: 0,
		ProximityCost: 0,
		DirectionBias: 0,
	}
	finder := NewAStarPathFinder(costs)
	
	// Path from (0,0) to (5,5)
	// With high turn cost, should prefer fewer turns
	path, err := finder.FindPath(
		core.Point{0, 0},
		core.Point{5, 5},
		nil,
	)
	
	if err != nil {
		t.Fatalf("FindPath failed: %v", err)
	}
	
	// Count turns
	turns := 0
	for i := 2; i < len(path.Points); i++ {
		dir1 := GetDirection(path.Points[i-2], path.Points[i-1])
		dir2 := GetDirection(path.Points[i-1], path.Points[i])
		if dir1 != dir2 {
			turns++
		}
	}
	
	// With high turn cost, should minimize turns (expect 1 turn for L-shaped path)
	if turns > 1 {
		t.Errorf("Path has %d turns, expected 1 with high turn cost", turns)
		t.Logf("Path: %s", PathToString(path))
	}
}

func TestAStarPathFinder_Performance(t *testing.T) {
	finder := NewAStarPathFinder(DefaultPathCost)
	finder.SetMaxNodes(10000) // Limit for test
	
	// Create a large grid with scattered obstacles
	obstacleSet := make(map[PointKey]bool)
	// Add some random obstacles
	for i := 0; i < 100; i++ {
		x := (i * 7) % 97  // Pseudo-random distribution
		y := (i * 13) % 97
		// Don't block start or end
		if (x != 0 || y != 0) && (x != 95 || y != 95) {
			obstacleSet[PointKey{x, y}] = true
		}
	}
	
	obstacles := func(p core.Point) bool {
		return obstacleSet[PointKey{p.X, p.Y}]
	}
	
	// Find path across large grid
	path, err := finder.FindPath(
		core.Point{0, 0},
		core.Point{95, 95},
		obstacles,
	)
	
	if err != nil {
		t.Fatalf("FindPath failed on large grid: %v", err)
	}
	
	// Should find a path
	if len(path.Points) == 0 {
		t.Error("No path found on large grid")
	}
	
	// Path should be reasonably efficient (not more than 2x optimal)
	optimalLength := 95 + 95 // Manhattan distance
	if len(path.Points) > optimalLength*2 {
		t.Errorf("Path too long: %d points, optimal would be ~%d", 
			len(path.Points), optimalLength)
	}
}


// parseObstacleMap converts ASCII art to an obstacle function.
// '.' or ' ' = free, 'X' or '#' = obstacle
func parseObstacleMap(mapStr string) func(core.Point) bool {
	lines := strings.Split(strings.TrimSpace(mapStr), "\n")
	obstacleSet := make(map[PointKey]bool)
	
	for y, line := range lines {
		for x, char := range line {
			if char == 'X' || char == '#' {
				obstacleSet[PointKey{x, y}] = true
			}
		}
	}
	
	return func(p core.Point) bool {
		return obstacleSet[PointKey{p.X, p.Y}]
	}
}

func BenchmarkAStar_SmallGrid(b *testing.B) {
	finder := NewAStarPathFinder(DefaultPathCost)
	obstacles := parseObstacleMap(`
..........
.XX....XX.
.XX....XX.
..........
..........
..........
.XX....XX.
.XX....XX.
..........
..........`)
	
	start := core.Point{0, 0}
	end := core.Point{9, 9}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = finder.FindPath(start, end, obstacles)
	}
}

func BenchmarkAStar_LargeGrid(b *testing.B) {
	finder := NewAStarPathFinder(DefaultPathCost)
	
	// Create sparse obstacles
	obstacleSet := make(map[PointKey]bool)
	for i := 0; i < 50; i++ {
		x := (i * 17) % 99
		y := (i * 23) % 99
		obstacleSet[PointKey{x, y}] = true
	}
	
	obstacles := func(p core.Point) bool {
		return obstacleSet[PointKey{p.X, p.Y}]
	}
	
	start := core.Point{0, 0}
	end := core.Point{99, 99}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = finder.FindPath(start, end, obstacles)
	}
}
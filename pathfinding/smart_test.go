package pathfinding

import (
	"edd/core"
	"fmt"
	"testing"
)

func TestSmartPathFinder_Basic(t *testing.T) {
	finder := NewSmartPathFinder(DefaultPathCost)
	
	tests := []struct {
		name      string
		start     core.Point
		end       core.Point
		obstacles string
		checkPath func(path core.Path) error
	}{
		{
			name:      "Direct path when clear",
			start:     core.Point{0, 0},
			end:       core.Point{5, 5},
			obstacles: "",
			checkPath: func(path core.Path) error {
				// Should use direct L-shaped path
				if len(path.Points) != 3 {
					return fmt.Errorf("expected 3 points for L-shaped path, got %d", len(path.Points))
				}
				return nil
			},
		},
		{
			name:  "Falls back to A* when blocked",
			start: core.Point{0, 0},
			end:   core.Point{5, 0},
			obstacles: `
.XX...`,
			checkPath: func(path core.Path) error {
				// Should route around obstacle
				foundDetour := false
				for _, p := range path.Points {
					if p.Y != 0 {
						foundDetour = true
						break
					}
				}
				if !foundDetour {
					return fmt.Errorf("expected path to detour around obstacle")
				}
				return nil
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obstacles := parseObstacleMap(tt.obstacles)
			path, err := finder.FindPath(tt.start, tt.end, obstacles)
			
			if err != nil {
				t.Fatalf("FindPath failed: %v", err)
			}
			
			if err := tt.checkPath(path); err != nil {
				t.Errorf("Path check failed: %v", err)
				t.Logf("Path: %s", PathToString(path))
			}
		})
	}
}

func TestSmartPathFinder_Optimization(t *testing.T) {
	finder := NewSmartPathFinder(DefaultPathCost)
	
	// Test dog-leg reduction
	obstacles := parseObstacleMap(`
..........
.XX....XX.
.XX....XX.
..........
..........`)
	
	// Path that should be optimized
	start := core.Point{0, 0}
	end := core.Point{9, 4}
	
	path, err := finder.FindPath(start, end, obstacles)
	if err != nil {
		t.Fatalf("FindPath failed: %v", err)
	}
	
	// Count turns in final path
	turns := 0
	for i := 2; i < len(path.Points); i++ {
		dir1 := GetDirection(path.Points[i-2], path.Points[i-1])
		dir2 := GetDirection(path.Points[i-1], path.Points[i])
		if dir1 != dir2 {
			turns++
		}
	}
	
	// Optimized path should have minimal turns
	if turns > 3 {
		t.Errorf("Path has %d turns, expected <= 3 after optimization", turns)
		t.Logf("Path: %s", PathToString(path))
	}
}

func TestSmartPathFinder_Padding(t *testing.T) {
	finder := NewSmartPathFinder(DefaultPathCost)
	finder.SetPadding(2) // Keep 2 cells away from obstacles
	
	obstacles := parseObstacleMap(`
.......
...X...
.......`)
	
	// Path from left to right of obstacle - should go around with padding
	path, err := finder.FindPath(
		core.Point{0, 1},
		core.Point{6, 1},
		obstacles,
	)
	
	if err != nil {
		t.Fatalf("FindPath failed: %v", err)
	}
	
	// Path should not go through obstacle
	for _, p := range path.Points {
		if obstacles(p) {
			t.Errorf("Path goes through obstacle at %v", p)
		}
	}
	
	// With padding=2, the direct path finder won't work, so A* should find a route
	// that goes around the obstacle. Just verify the path is valid.
	if len(path.Points) < 7 {
		t.Errorf("Path seems too short for going around obstacle: %d points", len(path.Points))
	}
}

func TestSmartPathFinder_ComplexScenario(t *testing.T) {
	// Test with a complex diagram-like scenario
	finder := NewSmartPathFinder(DefaultPathCost)
	
	obstacleMap := `
....................
.XXXX....XXXX......
.X..X....X..X......
.X..X....X..X......
.XXXX....XXXX......
....................
.......XXXXXX......
.......X....X......
.......X....X......
.......XXXXXX......
....................`
	
	obstacles := parseObstacleMap(obstacleMap)
	
	// Log the obstacle map
	t.Logf("Obstacle map:\n%s", obstacleMap)
	
	// Multiple test paths
	tests := []struct {
		name  string
		start core.Point
		end   core.Point
	}{
		{"Around top boxes", core.Point{0, 0}, core.Point{19, 0}},
		{"Through middle gap", core.Point{0, 5}, core.Point{19, 5}},
		{"Complex route", core.Point{5, 2}, core.Point{14, 8}},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := finder.FindPath(tt.start, tt.end, obstacles)
			
			if err != nil {
				t.Fatalf("FindPath failed: %v", err)
			}
			
			t.Logf("Path for %s: %v", tt.name, path.Points)
			
			// Verify path validity
			for i, p := range path.Points {
				if obstacles(p) {
					t.Errorf("Path goes through obstacle at %v", p)
				}
				
				// Check continuity
				if i > 0 {
					dist := ManhattanDistance(path.Points[i-1], p)
					if dist != 1 {
						t.Errorf("Path not continuous at index %d", i)
					}
				}
			}
			
			t.Logf("Path length: %d", len(path.Points))
		})
	}
}

func BenchmarkSmartPathFinder(b *testing.B) {
	finder := NewSmartPathFinder(DefaultPathCost)
	
	// Create a realistic diagram scenario
	nodes := []core.Node{
		{X: 10, Y: 10, Width: 10, Height: 5},
		{X: 30, Y: 10, Width: 10, Height: 5},
		{X: 50, Y: 10, Width: 10, Height: 5},
		{X: 20, Y: 25, Width: 10, Height: 5},
		{X: 40, Y: 25, Width: 10, Height: 5},
		{X: 30, Y: 40, Width: 10, Height: 5},
	}
	
	obstacles := CreateNodeObstacleChecker(nodes, 1)
	
	start := core.Point{5, 12}
	end := core.Point{55, 42}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = finder.FindPath(start, end, obstacles)
	}
}
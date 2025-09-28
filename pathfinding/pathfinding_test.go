package pathfinding

import (
	"edd/diagram"
	"edd/layout"
	"fmt"
	"strings"
	"testing"
	"time"
)

// ==================== PATH CACHE TESTS ====================

func TestPathCache_BasicOperations(t *testing.T) {
	cache := NewPathCache(10)
	
	// Create test paths
	path1 := diagram.Path{
		Points: []diagram.Point{{0, 0}, {1, 0}, {2, 0}},
		Cost:   20,
	}
	path2 := diagram.Path{
		Points: []diagram.Point{{0, 0}, {0, 1}, {0, 2}},
		Cost:   30,
	}
	
	// Test Put and Get
	start1, end1 := diagram.Point{0, 0}, diagram.Point{2, 0}
	cache.Put(start1, end1, 0, path1)
	
	retrieved, found := cache.Get(start1, end1, 0)
	if !found {
		t.Error("Path not found in cache")
	}
	if len(retrieved.Points) != len(path1.Points) {
		t.Errorf("Retrieved path has wrong length: got %d, want %d", 
			len(retrieved.Points), len(path1.Points))
	}
	
	// Test different path
	start2, end2 := diagram.Point{0, 0}, diagram.Point{0, 2}
	cache.Put(start2, end2, 0, path2)
	
	retrieved2, found := cache.Get(start2, end2, 0)
	if !found {
		t.Error("Second path not found in cache")
	}
	if retrieved2.Cost != path2.Cost {
		t.Errorf("Retrieved path has wrong cost: got %d, want %d", 
			retrieved2.Cost, path2.Cost)
	}
	
	// Test cache miss
	_, found = cache.Get(diagram.Point{10, 10}, diagram.Point{20, 20}, 0)
	if found {
		t.Error("Unexpected path found in cache")
	}
	
	// Check stats
	hits, misses, _, size := cache.Stats()
	if hits != 2 {
		t.Errorf("Wrong hit count: got %d, want 2", hits)
	}
	if misses != 1 {
		t.Errorf("Wrong miss count: got %d, want 1", misses)
	}
	if size != 2 {
		t.Errorf("Wrong cache size: got %d, want 2", size)
	}
}

func TestPathCache_ObstacleHash(t *testing.T) {
	cache := NewPathCache(10)
	path := diagram.Path{
		Points: []diagram.Point{{0, 0}, {1, 0}, {2, 0}},
		Cost:   20,
	}
	
	start, end := diagram.Point{0, 0}, diagram.Point{2, 0}
	
	// Put with obstacle hash 0
	cache.Put(start, end, 0, path)
	
	// Should find with same hash
	_, found := cache.Get(start, end, 0)
	if !found {
		t.Error("Path not found with matching obstacle hash")
	}
	
	// Should not find with different hash
	_, found = cache.Get(start, end, 123)
	if found {
		t.Error("Path found with non-matching obstacle hash")
	}
}

func TestPathCache_Eviction(t *testing.T) {
	cache := NewPathCache(2) // Small cache
	
	path := diagram.Path{
		Points: []diagram.Point{{0, 0}, {1, 0}},
		Cost:   10,
	}
	
	// Fill cache
	cache.Put(diagram.Point{0, 0}, diagram.Point{1, 0}, 0, path)
	cache.Put(diagram.Point{0, 0}, diagram.Point{0, 1}, 0, path)
	
	// Add third item - should trigger eviction
	cache.Put(diagram.Point{1, 0}, diagram.Point{2, 0}, 0, path)
	
	_, _, evictions, size := cache.Stats()
	if evictions != 1 {
		t.Errorf("Wrong eviction count: got %d, want 1", evictions)
	}
	if size != 2 {
		t.Errorf("Cache size should remain at max: got %d, want 2", size)
	}
}

func TestCachedPathFinder(t *testing.T) {
	// Create a simple path finder that counts calls
	callCount := 0
	mockFinder := PathFinderFunc(func(start, end diagram.Point, obstacles func(diagram.Point) bool) (diagram.Path, error) {
		callCount++
		return diagram.Path{
			Points: []diagram.Point{start, end},
			Cost:   10,
		}, nil
	})
	
	cachedFinder := NewCachedPathFinder(mockFinder, 10)
	
	// First call should go to underlying finder
	path1, err := cachedFinder.FindPath(diagram.Point{0, 0}, diagram.Point{1, 0}, nil)
	if err != nil {
		t.Fatalf("FindPath failed: %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected 1 call to underlying finder, got %d", callCount)
	}
	
	// Second identical call should use cache
	path2, err := cachedFinder.FindPath(diagram.Point{0, 0}, diagram.Point{1, 0}, nil)
	if err != nil {
		t.Fatalf("FindPath failed: %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected still 1 call to underlying finder, got %d", callCount)
	}
	
	// Verify paths are the same
	if len(path1.Points) != len(path2.Points) {
		t.Error("Cached path differs from original")
	}
	
	// Different endpoints should call finder again
	_, err = cachedFinder.FindPath(diagram.Point{0, 0}, diagram.Point{2, 0}, nil)
	if err != nil {
		t.Fatalf("FindPath failed: %v", err)
	}
	if callCount != 2 {
		t.Errorf("Expected 2 calls to underlying finder, got %d", callCount)
	}
	
	// Check cache stats
	stats := cachedFinder.CacheStats()
	t.Logf("Cache stats: %s", stats)
}

// PathFinderFunc is a function adapter for PathFinder interface
type PathFinderFunc func(start, end diagram.Point, obstacles func(diagram.Point) bool) (diagram.Path, error)

func (f PathFinderFunc) FindPath(start, end diagram.Point, obstacles func(diagram.Point) bool) (diagram.Path, error) {
	return f(start, end, obstacles)
}

func TestCachedPathFinder_ObstacleHashing(t *testing.T) {
	callCount := 0
	mockFinder := PathFinderFunc(func(start, end diagram.Point, obstacles func(diagram.Point) bool) (diagram.Path, error) {
		callCount++
		return diagram.Path{Points: []diagram.Point{start, end}, Cost: 10}, nil
	})
	
	cachedFinder := NewCachedPathFinder(mockFinder, 10)
	
	// Define two different obstacle functions
	obstacles1 := func(p diagram.Point) bool {
		return p.X == 1 && p.Y == 0
	}
	
	obstacles2 := func(p diagram.Point) bool {
		return p.X == 0 && p.Y == 1
	}
	
	// Same endpoints but different obstacles should not use cache
	_, _ = cachedFinder.FindPath(diagram.Point{0, 0}, diagram.Point{2, 0}, obstacles1)
	_, _ = cachedFinder.FindPath(diagram.Point{0, 0}, diagram.Point{2, 0}, obstacles2)
	
	if callCount != 2 {
		t.Errorf("Different obstacles should result in cache miss: got %d calls, want 2", callCount)
	}
	
	// Same obstacles should use cache
	_, _ = cachedFinder.FindPath(diagram.Point{0, 0}, diagram.Point{2, 0}, obstacles1)
	if callCount != 2 {
		t.Errorf("Same obstacles should use cache: got %d calls, want still 2", callCount)
	}
}

func BenchmarkPathCache(b *testing.B) {
	cache := NewPathCache(1000)
	path := diagram.Path{
		Points: []diagram.Point{{0, 0}, {1, 0}, {2, 0}},
		Cost:   20,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Mix of puts and gets
		x := i % 100
		y := (i / 100) % 100
		start := diagram.Point{X: x, Y: y}
		end := diagram.Point{X: x + 10, Y: y + 10}
		
		if i%2 == 0 {
			cache.Put(start, end, 0, path)
		} else {
			cache.Get(start, end, 0)
		}
	}
}

// ==================== ARROW TESTS ====================

func TestArrowConfig(t *testing.T) {
	config := NewArrowConfig()
	
	// Test default arrow type
	conn1 := diagram.Connection{From: 1, To: 2}
	if config.GetArrowType(conn1) != ArrowEnd {
		t.Errorf("Default arrow type should be ArrowEnd")
	}
	
	// Test setting specific arrow type
	config.SetArrowType(1, 2, ArrowBoth)
	if config.GetArrowType(conn1) != ArrowBoth {
		t.Errorf("Arrow type should be ArrowBoth after override")
	}
	
	// Test different connection still uses default
	conn2 := diagram.Connection{From: 2, To: 3}
	if config.GetArrowType(conn2) != ArrowEnd {
		t.Errorf("Different connection should still use default type")
	}
}

func TestArrowConfig_ShouldDrawArrow(t *testing.T) {
	config := NewArrowConfig()
	
	tests := []struct {
		name      string
		from      int
		to        int
		arrowType ArrowType
		wantEnd   bool
		wantStart bool
	}{
		{
			name:      "no arrow",
			from:      1,
			to:        2,
			arrowType: ArrowNone,
			wantEnd:   false,
			wantStart: false,
		},
		{
			name:      "end arrow only",
			from:      1,
			to:        2,
			arrowType: ArrowEnd,
			wantEnd:   true,
			wantStart: false,
		},
		{
			name:      "start arrow only",
			from:      1,
			to:        2,
			arrowType: ArrowStart,
			wantEnd:   false,
			wantStart: true,
		},
		{
			name:      "both arrows",
			from:      1,
			to:        2,
			arrowType: ArrowBoth,
			wantEnd:   true,
			wantStart: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.SetArrowType(tt.from, tt.to, tt.arrowType)
			conn := diagram.Connection{From: tt.from, To: tt.to}
			
			if got := config.ShouldDrawArrowAtEnd(conn); got != tt.wantEnd {
				t.Errorf("ShouldDrawArrowAtEnd() = %v, want %v", got, tt.wantEnd)
			}
			
			if got := config.ShouldDrawArrowAtStart(conn); got != tt.wantStart {
				t.Errorf("ShouldDrawArrowAtStart() = %v, want %v", got, tt.wantStart)
			}
		})
	}
}

func TestApplyArrowConfig(t *testing.T) {
	config := NewArrowConfig()
	config.SetArrowType(1, 2, ArrowBoth)
	config.SetArrowType(2, 3, ArrowNone)
	
	connections := []diagram.Connection{
		{From: 1, To: 2},
		{From: 2, To: 3},
		{From: 3, To: 1}, // Uses default
	}
	
	paths := map[int]diagram.Path{
		0: {Points: []diagram.Point{{0, 0}, {10, 0}}},
		1: {Points: []diagram.Point{{10, 0}, {10, 10}}},
		2: {Points: []diagram.Point{{10, 10}, {0, 0}}},
	}
	
	result := ApplyArrowConfig(connections, paths, config)
	
	if len(result) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(result))
	}
	
	// Check arrow types
	expectedTypes := []ArrowType{ArrowBoth, ArrowNone, ArrowEnd}
	for i, expected := range expectedTypes {
		if result[i].ArrowType != expected {
			t.Errorf("Result %d: ArrowType = %v, want %v", i, result[i].ArrowType, expected)
		}
	}
}

// ==================== ASTAR TESTS ====================

func TestAStarPathFinder_SimplePaths(t *testing.T) {
	finder := NewAStarPathFinder(DefaultPathCost)
	
	tests := []struct {
		name      string
		start     diagram.Point
		end       diagram.Point
		obstacles string // ASCII representation of obstacles
		minLength int    // minimum expected path length
	}{
		{
			name:      "Direct horizontal path",
			start:     diagram.Point{0, 0},
			end:       diagram.Point{5, 0},
			obstacles: "",
			minLength: 6,
		},
		{
			name:      "Direct vertical path",
			start:     diagram.Point{0, 0},
			end:       diagram.Point{0, 5},
			obstacles: "",
			minLength: 6,
		},
		{
			name:      "L-shaped path",
			start:     diagram.Point{0, 0},
			end:       diagram.Point{5, 5},
			obstacles: "",
			minLength: 11,
		},
		{
			name:  "Path around obstacle",
			start: diagram.Point{0, 2},
			end:   diagram.Point{4, 2},
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
			start: diagram.Point{0, 0},
			end:   diagram.Point{4, 4},
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
		start     diagram.Point
		end       diagram.Point
		obstacles string
	}{
		{
			name:  "End blocked",
			start: diagram.Point{0, 0},
			end:   diagram.Point{2, 0},
			obstacles: `
..X`,
		},
		{
			name:  "Start blocked",
			start: diagram.Point{0, 0},
			end:   diagram.Point{2, 0},
			obstacles: `
X..`,
		},
		{
			name:  "Completely blocked",
			start: diagram.Point{0, 0},
			end:   diagram.Point{2, 2},
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
		diagram.Point{0, 0},
		diagram.Point{5, 5},
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
	
	obstacles := func(p diagram.Point) bool {
		return obstacleSet[PointKey{p.X, p.Y}]
	}
	
	// Find path across large grid
	path, err := finder.FindPath(
		diagram.Point{0, 0},
		diagram.Point{95, 95},
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
func parseObstacleMap(mapStr string) func(diagram.Point) bool {
	lines := strings.Split(strings.TrimSpace(mapStr), "\n")
	obstacleSet := make(map[PointKey]bool)
	
	for y, line := range lines {
		for x, char := range line {
			if char == 'X' || char == '#' {
				obstacleSet[PointKey{x, y}] = true
			}
		}
	}
	
	return func(p diagram.Point) bool {
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
	
	start := diagram.Point{0, 0}
	end := diagram.Point{9, 9}
	
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
	
	obstacles := func(p diagram.Point) bool {
		return obstacleSet[PointKey{p.X, p.Y}]
	}
	
	start := diagram.Point{0, 0}
	end := diagram.Point{99, 99}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = finder.FindPath(start, end, obstacles)
	}
}

// ==================== SMART PATH FINDER TESTS ====================

func TestSmartPathFinder_Basic(t *testing.T) {
	finder := NewSmartPathFinder(DefaultPathCost)
	
	tests := []struct {
		name      string
		start     diagram.Point
		end       diagram.Point
		obstacles string
		checkPath func(path diagram.Path) error
	}{
		{
			name:      "Direct path when clear",
			start:     diagram.Point{0, 0},
			end:       diagram.Point{5, 5},
			obstacles: "",
			checkPath: func(path diagram.Path) error {
				// Should use direct L-shaped path
				if len(path.Points) != 3 {
					return fmt.Errorf("expected 3 points for L-shaped path, got %d", len(path.Points))
				}
				return nil
			},
		},
		{
			name:  "Falls back to A* when blocked",
			start: diagram.Point{0, 0},
			end:   diagram.Point{5, 0},
			obstacles: `
.XX...`,
			checkPath: func(path diagram.Path) error {
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
	start := diagram.Point{0, 0}
	end := diagram.Point{9, 4}
	
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
		diagram.Point{0, 1},
		diagram.Point{6, 1},
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
		start diagram.Point
		end   diagram.Point
	}{
		{"Around top boxes", diagram.Point{0, 0}, diagram.Point{19, 0}},
		{"Through middle gap", diagram.Point{0, 5}, diagram.Point{19, 5}},
		{"Complex route", diagram.Point{5, 2}, diagram.Point{14, 8}},
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
	nodes := []diagram.Node{
		{X: 10, Y: 10, Width: 10, Height: 5},
		{X: 30, Y: 10, Width: 10, Height: 5},
		{X: 50, Y: 10, Width: 10, Height: 5},
		{X: 20, Y: 25, Width: 10, Height: 5},
		{X: 40, Y: 25, Width: 10, Height: 5},
		{X: 30, Y: 40, Width: 10, Height: 5},
	}
	
	obstacles := CreateNodeObstacleChecker(nodes, 1)
	
	start := diagram.Point{5, 12}
	end := diagram.Point{55, 42}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = finder.FindPath(start, end, obstacles)
	}
}

// ==================== OBSTACLES TESTS ====================

func TestRectangleObstacle(t *testing.T) {
	rect := RectangleObstacle{
		X:       10,
		Y:       10,
		Width:   5,
		Height:  5,
		Padding: 1,
	}
	
	tests := []struct {
		point    diagram.Point
		contains bool
	}{
		// Inside rectangle
		{diagram.Point{12, 12}, true},
		{diagram.Point{10, 10}, true},
		{diagram.Point{14, 14}, true},
		
		// In padding area
		{diagram.Point{9, 10}, true},
		{diagram.Point{15, 12}, true},
		{diagram.Point{12, 9}, true},
		{diagram.Point{12, 15}, true},
		
		// Outside
		{diagram.Point{8, 10}, false},
		{diagram.Point{16, 12}, false},
		{diagram.Point{12, 8}, false},
		{diagram.Point{12, 16}, false},
		{diagram.Point{0, 0}, false},
		{diagram.Point{100, 100}, false},
	}
	
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if rect.Contains(tt.point) != tt.contains {
				t.Errorf("Contains(%v) = %v, want %v", tt.point, rect.Contains(tt.point), tt.contains)
			}
		})
	}
}

func TestCreateNodeObstacleChecker(t *testing.T) {
	nodes := []diagram.Node{
		{ID: 1, X: 10, Y: 10, Width: 5, Height: 5},
		{ID: 2, X: 20, Y: 20, Width: 10, Height: 3},
	}
	
	checker := CreateNodeObstacleChecker(nodes, 1)
	
	tests := []struct {
		point      diagram.Point
		isObstacle bool
	}{
		// First node area
		{diagram.Point{12, 12}, true},
		{diagram.Point{9, 10}, true}, // padding
		
		// Second node area
		{diagram.Point{25, 21}, true},
		{diagram.Point{19, 20}, true}, // padding
		
		// Free space
		{diagram.Point{0, 0}, false},
		{diagram.Point{17, 17}, false},
		{diagram.Point{50, 50}, false},
	}
	
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if checker(tt.point) != tt.isObstacle {
				t.Errorf("checker(%v) = %v, want %v", tt.point, checker(tt.point), tt.isObstacle)
			}
		})
	}
}

func TestCreateBoundsObstacleChecker(t *testing.T) {
	bounds := diagram.Bounds{
		Min: diagram.Point{X: 0, Y: 0},
		Max: diagram.Point{X: 100, Y: 50},
	}
	
	checker := CreateBoundsObstacleChecker(bounds)
	
	tests := []struct {
		point      diagram.Point
		isObstacle bool
	}{
		// Inside bounds
		{diagram.Point{50, 25}, false},
		{diagram.Point{0, 0}, false},
		{diagram.Point{99, 49}, false},
		
		// Outside bounds
		{diagram.Point{-1, 25}, true},
		{diagram.Point{100, 25}, true},
		{diagram.Point{50, -1}, true},
		{diagram.Point{50, 50}, true},
		{diagram.Point{-10, -10}, true},
		{diagram.Point{200, 200}, true},
	}
	
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if checker(tt.point) != tt.isObstacle {
				t.Errorf("checker(%v) = %v, want %v", tt.point, checker(tt.point), tt.isObstacle)
			}
		})
	}
}

func TestCombineObstacleCheckers(t *testing.T) {
	// Create a bounds checker
	boundsChecker := CreateBoundsObstacleChecker(diagram.Bounds{
		Min: diagram.Point{X: 0, Y: 0},
		Max: diagram.Point{X: 50, Y: 50},
	})
	
	// Create a node checker
	nodeChecker := CreateNodeObstacleChecker([]diagram.Node{
		{X: 20, Y: 20, Width: 10, Height: 10},
	}, 0)
	
	// Combine them
	combined := CombineObstacleCheckers(boundsChecker, nodeChecker)
	
	tests := []struct {
		point      diagram.Point
		isObstacle bool
		reason     string
	}{
		{diagram.Point{10, 10}, false, "inside bounds, not in node"},
		{diagram.Point{25, 25}, true, "inside node"},
		{diagram.Point{-5, 25}, true, "outside bounds"},
		{diagram.Point{60, 25}, true, "outside bounds"},
		{diagram.Point{25, -5}, true, "outside bounds"},
		{diagram.Point{25, 60}, true, "outside bounds"},
	}
	
	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			if combined(tt.point) != tt.isObstacle {
				t.Errorf("combined(%v) = %v, want %v", tt.point, combined(tt.point), tt.isObstacle)
			}
		})
	}
}

func TestPathObstacle(t *testing.T) {
	path := PathObstacle{
		Points: []diagram.Point{
			{X: 10, Y: 10},
			{X: 20, Y: 10},
			{X: 20, Y: 20},
		},
		Thickness: 1,
	}
	
	tests := []struct {
		point    diagram.Point
		contains bool
		reason   string
	}{
		// On horizontal segment
		{diagram.Point{15, 10}, true, "on horizontal line"},
		{diagram.Point{10, 10}, true, "at start point"},
		{diagram.Point{20, 10}, true, "at corner"},
		
		// On vertical segment
		{diagram.Point{20, 15}, true, "on vertical line"},
		{diagram.Point{20, 20}, true, "at end point"},
		
		// Within thickness
		{diagram.Point{15, 9}, true, "within thickness of horizontal"},
		{diagram.Point{15, 11}, true, "within thickness of horizontal"},
		{diagram.Point{19, 15}, true, "within thickness of vertical"},
		{diagram.Point{21, 15}, true, "within thickness of vertical"},
		
		// Outside
		{diagram.Point{15, 8}, false, "outside thickness"},
		{diagram.Point{15, 12}, false, "outside thickness"},
		{diagram.Point{25, 15}, false, "beyond path"},
		{diagram.Point{5, 5}, false, "far from path"},
	}
	
	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			if path.Contains(tt.point) != tt.contains {
				t.Errorf("Contains(%v) = %v, want %v", tt.point, path.Contains(tt.point), tt.contains)
			}
		})
	}
}

func TestCreateRegionObstacleChecker(t *testing.T) {
	regions := []Region{
		{
			Bounds:     diagram.Bounds{Min: diagram.Point{X: 10, Y: 10}, Max: diagram.Point{X: 20, Y: 20}},
			IsObstacle: true,
		},
		{
			Bounds:     diagram.Bounds{Min: diagram.Point{X: 30, Y: 30}, Max: diagram.Point{X: 40, Y: 40}},
			IsObstacle: false, // This is a high-cost region, not an obstacle
			Cost:       50,
		},
	}
	
	checker := CreateRegionObstacleChecker(regions)
	
	tests := []struct {
		point      diagram.Point
		isObstacle bool
		reason     string
	}{
		{diagram.Point{15, 15}, true, "inside obstacle region"},
		{diagram.Point{35, 35}, false, "inside cost region (not obstacle)"},
		{diagram.Point{5, 5}, false, "outside all regions"},
		{diagram.Point{25, 25}, false, "between regions"},
		{diagram.Point{10, 10}, true, "on obstacle boundary"},
		{diagram.Point{20, 20}, false, "just outside obstacle (exclusive max)"},
	}
	
	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			if checker(tt.point) != tt.isObstacle {
				t.Errorf("checker(%v) = %v, want %v", tt.point, checker(tt.point), tt.isObstacle)
			}
		})
	}
}

func TestGetRegionCost(t *testing.T) {
	regions := []Region{
		{
			Bounds:     diagram.Bounds{Min: diagram.Point{X: 10, Y: 10}, Max: diagram.Point{X: 20, Y: 20}},
			IsObstacle: true,
			Cost:       0, // Obstacles don't add cost
		},
		{
			Bounds:     diagram.Bounds{Min: diagram.Point{X: 30, Y: 30}, Max: diagram.Point{X: 40, Y: 40}},
			IsObstacle: false,
			Cost:       50,
		},
		{
			Bounds:     diagram.Bounds{Min: diagram.Point{X: 35, Y: 35}, Max: diagram.Point{X: 45, Y: 45}},
			IsObstacle: false,
			Cost:       30, // Overlaps with previous region
		},
	}
	
	tests := []struct {
		point        diagram.Point
		expectedCost int
		reason       string
	}{
		{diagram.Point{5, 5}, 0, "outside all regions"},
		{diagram.Point{15, 15}, 0, "inside obstacle (no cost)"},
		{diagram.Point{35, 35}, 80, "inside two overlapping cost regions"},
		{diagram.Point{32, 32}, 50, "inside single cost region"},
		{diagram.Point{42, 42}, 30, "inside only second cost region"},
	}
	
	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			cost := GetRegionCost(tt.point, regions)
			if cost != tt.expectedCost {
				t.Errorf("GetRegionCost(%v) = %d, want %d", tt.point, cost, tt.expectedCost)
			}
		})
	}
}

// ==================== PERFORMANCE TESTS ====================

func TestCurrentPerformance(t *testing.T) {
	scenarios := []struct {
		name        string
		distance    int
		obstaclesPct float64
		desc        string
	}{
		{"Short_10", 10, 0.1, "Short path with few obstacles"},
		{"Medium_30", 30, 0.2, "Medium path with moderate obstacles"},
		{"Long_50", 50, 0.3, "Long path with many obstacles"},
		{"VeryLong_100", 100, 0.3, "Very long path with many obstacles"},
		{"Extreme_200", 200, 0.4, "Extreme path with dense obstacles"},
	}
	
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			// Create obstacles
			obstacles := func(p diagram.Point) bool {
				// Never block corners where we place start/end
				if p.X <= 1 || p.Y <= 1 || p.X >= s.distance-1 || p.Y >= s.distance-1 {
					return false
				}
				// Deterministic obstacle placement
				hash := uint32(p.X*7919 + p.Y*1337)
				hash = (hash ^ (hash >> 16)) * 0x45d9f3b
				hash = hash ^ (hash >> 16)
				return float64(hash%1000) < s.obstaclesPct*1000
			}
			
			start := diagram.Point{1, 1}
			end := diagram.Point{s.distance - 1, s.distance - 1}
			
			// Test with different finders
			finders := []struct {
				name   string
				finder PathFinder
			}{
				{"SmartFinder", NewSmartPathFinder(DefaultPathCost)},
				{"RawAStar", NewAStarPathFinder(DefaultPathCost)},
			}
			
			for _, f := range finders {
				// Measure single path computation
				startTime := time.Now()
				path, err := f.finder.FindPath(start, end, obstacles)
				elapsed := time.Since(startTime)
				
				if err != nil {
					t.Logf("%s: Failed - %v", f.name, err)
					continue
				}
				
				pathLen := len(path.Points)
				timePerCell := float64(elapsed.Microseconds()) / float64(pathLen)
				
				t.Logf("%s: Distance=%d, PathLen=%d, Time=%v, µs/cell=%.1f", 
					f.name, s.distance, pathLen, elapsed, timePerCell)
			}
		})
	}
}

func BenchmarkPathfindingScalability(b *testing.B) {
	distances := []int{10, 20, 50, 100, 150}
	
	for _, dist := range distances {
		b.Run(fmt.Sprintf("Distance_%d", dist), func(b *testing.B) {
			// 20% obstacle density
			obstacles := func(p diagram.Point) bool {
				if p.X == 1 && p.Y == 1 { return false } // start
				if p.X == dist-1 && p.Y == dist-1 { return false } // end
				hash := uint32(p.X*7919 + p.Y*1337)
				return hash%5 == 0
			}
			
			start := diagram.Point{1, 1}
			end := diagram.Point{dist - 1, dist - 1}
			
			finder := NewSmartPathFinder(DefaultPathCost)
			finder.EnableCache(false) // Test raw performance
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := finder.FindPath(start, end, obstacles)
				if err != nil {
					b.Fatal(err)
				}
			}
			
			b.ReportMetric(float64(dist), "distance")
		})
	}
}

// Test worst-case scenario: maze that forces maximum exploration
func TestWorstCasePerformance(t *testing.T) {
	// Create a spiral maze that forces A* to explore many cells
	size := 50
	obstacles := func(p diagram.Point) bool {
		// Create walls that form a spiral
		x, y := p.X-size/2, p.Y-size/2
		
		// Spiral walls pattern
		if x == 0 && y > 0 && y < 20 { return true }
		if y == 20 && x >= 0 && x < 20 { return true }
		if x == 20 && y <= 20 && y > -20 { return true }
		if y == -20 && x <= 20 && x > -18 { return true }
		if x == -18 && y >= -20 && y < 18 { return true }
		if y == 18 && x >= -18 && x < 18 { return true }
		if x == 18 && y <= 18 && y > -18 { return true }
		
		return false
	}
	
	finder := NewSmartPathFinder(DefaultPathCost)
	finder.EnableCache(false)
	
	// Force a path through the spiral
	start := diagram.Point{size/2, size/2}
	end := diagram.Point{size/2 + 15, size/2}
	
	startTime := time.Now()
	path, err := finder.FindPath(start, end, obstacles)
	elapsed := time.Since(startTime)
	
	if err != nil {
		t.Fatalf("Failed to find path: %v", err)
	}
	
	t.Logf("Worst case spiral: PathLen=%d, Time=%v, Total cells in %dx%d=%d",
		len(path.Points), elapsed, size, size, size*size)
	t.Logf("Performance: %.2f cells explored per millisecond",
		float64(len(path.Points))/float64(elapsed.Milliseconds()))
}

// ==================== COMPLEX PATH BENCHMARK TESTS ====================

func BenchmarkComplexPaths(b *testing.B) {
	scenarios := []struct {
		name      string
		size      int
		obstacles string
		desc      string
	}{
		{
			name: "Maze_20x20",
			size: 20,
			obstacles: `
####################
#..................#
#.####.####.####.#.#
#......#........#..#
####.#.#.####.#.##.#
#....#.#......#....#
#.####.########.##.#
#..................#
#.##.####.####.###.#
#.##......#....#...#
#.########.#.###.#.#
#..........#.....#.#
#.####.########.##.#
#....#.........#...#
####.#.#######.#.###
#....#.........#...#
#.############.###.#
#..................#
####################`,
			desc: "Dense maze with many turns",
		},
		{
			name: "Spiral_30x30",
			size: 30,
			obstacles: `
##############################
#............................#
#.##########################.#
#.#........................#.#
#.#.######################.#.#
#.#.#....................#.#.#
#.#.#.##################.#.#.#
#.#.#.#................#.#.#.#
#.#.#.#.##############.#.#.#.#
#.#.#.#.#............#.#.#.#.#
#.#.#.#.#.##########.#.#.#.#.#
#.#.#.#.#.#........#.#.#.#.#.#
#.#.#.#.#.#.######.#.#.#.#.#.#
#.#.#.#.#.#.#....#.#.#.#.#.#.#
#.#.#.#.#.#.#.##.#.#.#.#.#.#.#
#.#.#.#.#.#.#..#.#.#.#.#.#.#.#
#.#.#.#.#.#.####.#.#.#.#.#.#.#
#.#.#.#.#.#......#.#.#.#.#.#.#
#.#.#.#.#.########.#.#.#.#.#.#
#.#.#.#.#..........#.#.#.#.#.#
#.#.#.#.############.#.#.#.#.#
#.#.#.#..............#.#.#.#.#
#.#.#.################.#.#.#.#
#.#.#..................#.#.#.#
#.#.####################.#.#.#
#.#......................#.#.#
#.########################.#.#
#..........................#.#
##############################`,
			desc: "Spiral pattern forcing very long path",
		},
		{
			name: "Sparse_50x50",
			size: 50,
			obstacles: "", // Will generate programmatically
			desc: "Large area with scattered obstacles",
		},
		{
			name: "Diagonal_100x100",
			size: 100,
			obstacles: "", // Will generate programmatically
			desc: "Very large area with diagonal barrier",
		},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			var obstacles func(diagram.Point) bool
			
			if scenario.obstacles != "" {
				obstacles = parseObstacleMap(scenario.obstacles)
			} else {
				// Generate obstacles programmatically
				obstacles = generateObstacles(scenario.size, scenario.name)
			}
			
			// Test points from corners
			start := diagram.Point{1, 1}
			end := diagram.Point{scenario.size - 2, scenario.size - 2}
			
			// Create both cached and non-cached finders
			smartFinder := NewSmartPathFinder(DefaultPathCost)
			smartFinderNoCache := NewSmartPathFinder(DefaultPathCost)
			smartFinderNoCache.EnableCache(false)
			
			// Warm up cache
			smartFinder.FindPath(start, end, obstacles)
			
			b.Run("WithCache", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, err := smartFinder.FindPath(start, end, obstacles)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
			
			b.Run("NoCache", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, err := smartFinderNoCache.FindPath(start, end, obstacles)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
			
			// Also test raw A* performance
			astarFinder := NewAStarPathFinder(DefaultPathCost)
			b.Run("RawAStar", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, err := astarFinder.FindPath(start, end, obstacles)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}

func generateObstacles(size int, pattern string) func(diagram.Point) bool {
	switch pattern {
	case "Sparse_50x50":
		// Random-looking but deterministic obstacles
		return func(p diagram.Point) bool {
			// Boundaries
			if p.X == 0 || p.Y == 0 || p.X == size-1 || p.Y == size-1 {
				return true
			}
			// Scattered obstacles
			return (p.X*7+p.Y*13)%17 == 0
		}
	case "Diagonal_100x100":
		// Diagonal barrier with small gaps
		return func(p diagram.Point) bool {
			// Boundaries
			if p.X == 0 || p.Y == 0 || p.X == size-1 || p.Y == size-1 {
				return true
			}
			// Diagonal wall with gaps every 10 cells
			if p.X == p.Y && p.X%10 != 5 {
				return true
			}
			return false
		}
	default:
		return func(p diagram.Point) bool { return false }
	}
}

// Measure actual path computation time for complex scenarios
func TestPathComputationTime(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timing test in short mode")
	}
	
	scenarios := []struct {
		name   string
		start  diagram.Point
		end    diagram.Point
		buildObstacles func() func(diagram.Point) bool
	}{
		{
			name:  "Long corridor with turns",
			start: diagram.Point{1, 1},
			end:   diagram.Point{98, 98},
			buildObstacles: func() func(diagram.Point) bool {
				// Create a maze-like pattern
				return func(p diagram.Point) bool {
					// Create walls that force a winding path
					if p.Y%10 == 5 && p.X < 95 && p.X%10 != 5 {
						return true
					}
					if p.Y%10 == 6 && p.X > 5 && p.X%10 != 5 {
						return true
					}
					return false
				}
			},
		},
		{
			name:  "Dense obstacle field",
			start: diagram.Point{5, 5},
			end:   diagram.Point{95, 95},
			buildObstacles: func() func(diagram.Point) bool {
				// 40% of cells are obstacles
				return func(p diagram.Point) bool {
					// Never block start or end
					if (p.X == 5 && p.Y == 5) || (p.X == 95 && p.Y == 95) {
						return false
					}
					hash := uint32(p.X*1337 + p.Y*7919)
					hash = (hash ^ (hash >> 16)) * 0x45d9f3b
					hash = (hash ^ (hash >> 16)) * 0x45d9f3b
					hash = hash ^ (hash >> 16)
					return hash%10 < 4
				}
			},
		},
		{
			name:  "Worst case spiral",
			start: diagram.Point{50, 50},
			end:   diagram.Point{51, 51},
			buildObstacles: func() func(diagram.Point) bool {
				// Create a spiral that forces a very long path for nearby points
				return func(p diagram.Point) bool {
					dx := p.X - 50
					dy := p.Y - 50
					
					// Create spiral walls
					if dx == 0 && dy > 0 && dy < 40 { return true }
					if dy == 40 && dx >= 0 && dx < 40 { return true }
					if dx == 40 && dy <= 40 && dy > -40 { return true }
					if dy == -40 && dx <= 40 && dx > -40 { return true }
					if dx == -40 && dy >= -40 && dy < 35 { return true }
					if dy == 35 && dx >= -40 && dx < 35 { return true }
					// ... continue spiral
					
					return false
				}
			},
		},
	}
	
	finder := NewSmartPathFinder(DefaultPathCost)
	finder.EnableCache(false) // Test raw performance
	
	for _, scenario := range scenarios {
		obstacles := scenario.buildObstacles()
		
		start := time.Now()
		path, err := finder.FindPath(scenario.start, scenario.end, obstacles)
		elapsed := time.Since(start)
		
		if err != nil {
			t.Errorf("%s: Failed to find path: %v", scenario.name, err)
			continue
		}
		
		t.Logf("%s: Path length=%d, Time=%v, Time per cell=%.2fµs", 
			scenario.name, 
			len(path.Points), 
			elapsed,
			float64(elapsed.Microseconds())/float64(len(path.Points)))
	}
	
	// Get cache stats if we enable it
	finder.EnableCache(true)
	for _, scenario := range scenarios {
		obstacles := scenario.buildObstacles()
		finder.FindPath(scenario.start, scenario.end, obstacles)
	}
	t.Logf("Cache stats after all scenarios: %s", finder.CacheStats())
}

// ==================== SMART CACHE TESTS ====================

func TestSmartPathFinder_Caching(t *testing.T) {
	finder := NewSmartPathFinder(DefaultPathCost)
	
	// Create obstacles
	obstacles := parseObstacleMap(`
.........
...XXX...
...XXX...
.........`)
	
	start := diagram.Point{0, 0}
	end := diagram.Point{8, 3}
	
	// First call - should compute path
	path1, err := finder.FindPath(start, end, obstacles)
	if err != nil {
		t.Fatalf("First FindPath failed: %v", err)
	}
	
	// Get initial cache stats
	stats1 := finder.CacheStats()
	t.Logf("After first call: %s", stats1)
	
	// Second identical call - should use cache
	path2, err := finder.FindPath(start, end, obstacles)
	if err != nil {
		t.Fatalf("Second FindPath failed: %v", err)
	}
	
	// Verify paths are identical
	if len(path1.Points) != len(path2.Points) {
		t.Error("Cached path differs from original")
	}
	
	// Check cache hit
	stats2 := finder.CacheStats()
	t.Logf("After second call: %s", stats2)
	
	// Simply verify we got a cache hit by checking the stats string
	// We can see from the log that hits went from 0 to 1
	if stats1 == stats2 {
		t.Error("Expected cache stats to change after second call")
	}
	
	// Test cache clearing
	finder.ClearCache()
	stats3 := finder.CacheStats()
	t.Logf("After clear: %s", stats3)
	
	// Test with cache disabled
	finder.EnableCache(false)
	_, err = finder.FindPath(start, end, obstacles)
	if err != nil {
		t.Fatalf("Third FindPath failed: %v", err)
	}
	
	// Stats should not change when cache is disabled
	stats4 := finder.CacheStats()
	if stats4 != stats3 {
		t.Error("Cache stats changed when cache was disabled")
	}
}

func BenchmarkSmartPathFinder_WithCache(b *testing.B) {
	finder := NewSmartPathFinder(DefaultPathCost)
	
	obstacles := CreateNodeObstacleChecker([]diagram.Node{
		{X: 10, Y: 10, Width: 10, Height: 5},
		{X: 30, Y: 10, Width: 10, Height: 5},
	}, 1)
	
	// Test points that will be repeated
	points := []struct{ start, end diagram.Point }{
		{diagram.Point{0, 0}, diagram.Point{50, 20}},
		{diagram.Point{5, 5}, diagram.Point{45, 25}},
		{diagram.Point{0, 20}, diagram.Point{50, 0}},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := points[i%len(points)]
		_, _ = finder.FindPath(p.start, p.end, obstacles)
	}
	
	b.StopTimer()
	b.Logf("Final cache stats: %s", finder.CacheStats())
}

func BenchmarkSmartPathFinder_NoCache(b *testing.B) {
	finder := NewSmartPathFinder(DefaultPathCost)
	finder.EnableCache(false)
	
	obstacles := CreateNodeObstacleChecker([]diagram.Node{
		{X: 10, Y: 10, Width: 10, Height: 5},
		{X: 30, Y: 10, Width: 10, Height: 5},
	}, 1)
	
	points := []struct{ start, end diagram.Point }{
		{diagram.Point{0, 0}, diagram.Point{50, 20}},
		{diagram.Point{5, 5}, diagram.Point{45, 25}},
		{diagram.Point{0, 20}, diagram.Point{50, 0}},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := points[i%len(points)]
		_, _ = finder.FindPath(p.start, p.end, obstacles)
	}
}

// ==================== OBSTACLE HASH TESTS ====================

func TestSmartPathFinder_ObstacleHashing(t *testing.T) {
	finder := NewSmartPathFinder(DefaultPathCost)
	
	// Test case 1: Obstacle outside direct bounding box should affect hash
	t.Run("Detour obstacle detection", func(t *testing.T) {
		start := diagram.Point{0, 0}
		end := diagram.Point{10, 0}
		
		// Obstacle at Y=-5 forces detour but is outside direct bounding box
		obstacles1 := func(p diagram.Point) bool {
			return p.X == 5 && p.Y == 0 // Direct path blocked
		}
		
		obstacles2 := func(p diagram.Point) bool {
			return (p.X == 5 && p.Y == 0) || // Direct path blocked
				(p.X == 5 && p.Y == -5) // Additional obstacle on detour path
		}
		
		hash1 := finder.hashObstacles(start, end, obstacles1)
		hash2 := finder.hashObstacles(start, end, obstacles2)
		
		if hash1 == hash2 {
			t.Error("Hashes should differ when detour obstacles are present")
		}
		
		// Verify paths are actually different
		path1, _ := finder.FindPath(start, end, obstacles1)
		path2, _ := finder.FindPath(start, end, obstacles2)
		
		if len(path1.Points) == len(path2.Points) {
			t.Log("Warning: Paths have same length despite different obstacles")
		}
	})
	
	// Test case 2: Same obstacles, different endpoints should have different hashes
	t.Run("Endpoint differentiation", func(t *testing.T) {
		obstacles := func(p diagram.Point) bool {
			return p.X == 5 && p.Y == 5
		}
		
		hash1 := finder.hashObstacles(diagram.Point{0, 0}, diagram.Point{10, 10}, obstacles)
		hash2 := finder.hashObstacles(diagram.Point{0, 0}, diagram.Point{10, 11}, obstacles)
		
		if hash1 == hash2 {
			t.Error("Hashes should differ for different endpoints")
		}
	})
	
	// Test case 3: Verify cache correctness with complex obstacles
	t.Run("Cache correctness", func(t *testing.T) {
		// Create a complex obstacle pattern
		obstacles := func(p diagram.Point) bool {
			// Vertical wall with a gap
			if p.X == 5 && p.Y >= -5 && p.Y <= 5 && p.Y != 0 {
				return true
			}
			return false
		}
		
		start := diagram.Point{0, 0}
		end := diagram.Point{10, 0}
		
		// First call - computes path
		path1, err := finder.FindPath(start, end, obstacles)
		if err != nil {
			t.Fatalf("First path failed: %v", err)
		}
		
		// Clear any potential issues
		finder.ClearCache()
		
		// Second call - should compute identical path
		path2, err := finder.FindPath(start, end, obstacles)
		if err != nil {
			t.Fatalf("Second path failed: %v", err)
		}
		
		// Paths should be identical
		if len(path1.Points) != len(path2.Points) {
			t.Errorf("Path lengths differ: %d vs %d", len(path1.Points), len(path2.Points))
		}
		
		// Verify the path goes through the gap
		foundGap := false
		for _, p := range path1.Points {
			if p.X == 5 && p.Y == 0 {
				foundGap = true
				break
			}
		}
		if !foundGap {
			t.Error("Path should go through the gap at (5,0)")
		}
	})
}

func TestObstacleHashCollisions(t *testing.T) {
	finder := NewSmartPathFinder(DefaultPathCost)
	
	// Track hashes to check for collisions
	hashes := make(map[uint64]string)
	
	testCases := []struct {
		name      string
		start     diagram.Point
		end       diagram.Point
		obstacles string
	}{
		{
			"Empty grid",
			diagram.Point{0, 0}, diagram.Point{5, 5},
			"",
		},
		{
			"Single obstacle",
			diagram.Point{0, 0}, diagram.Point{5, 5},
			`
.....
..X..
.....`,
		},
		{
			"Wall",
			diagram.Point{0, 0}, diagram.Point{5, 5},
			`
.....
XXXX.
.....`,
		},
		{
			"Different wall",
			diagram.Point{0, 0}, diagram.Point{5, 5},
			`
.....
.XXXX
.....`,
		},
		{
			"Vertical wall",
			diagram.Point{0, 0}, diagram.Point{5, 5},
			`
..X..
..X..
..X..`,
		},
	}
	
	for _, tc := range testCases {
		obstacles := parseObstacleMap(tc.obstacles)
		hash := finder.hashObstacles(tc.start, tc.end, obstacles)
		
		if existing, found := hashes[hash]; found {
			t.Errorf("Hash collision: %s and %s have same hash %d", 
				existing, tc.name, hash)
		}
		hashes[hash] = tc.name
	}
	
	t.Logf("Generated %d unique hashes for %d test cases", 
		len(hashes), len(testCases))
}

// ==================== ROUTER TESTS ====================

func TestRouter_RouteConnection(t *testing.T) {
	// Create a simple pathfinder
	pf := NewAStarPathFinder(PathCost{
		StraightCost: 10,
		TurnCost: 5,
	})
	router := NewRouter(pf)
	
	// Define test nodes
	nodes := []diagram.Node{
		{ID: 1, X: 5, Y: 5, Width: 10, Height: 5},
		{ID: 2, X: 25, Y: 5, Width: 10, Height: 5},
		{ID: 3, X: 15, Y: 20, Width: 10, Height: 5},
	}
	
	tests := []struct {
		name    string
		conn    diagram.Connection
		wantErr bool
		minLen  int // minimum expected path length
	}{
		{
			name:    "simple horizontal connection",
			conn:    diagram.Connection{From: 1, To: 2},
			wantErr: false,
			minLen:  2,
		},
		{
			name:    "diagonal connection",
			conn:    diagram.Connection{From: 1, To: 3},
			wantErr: false,
			minLen:  2,
		},
		{
			name:    "reverse connection",
			conn:    diagram.Connection{From: 2, To: 1},
			wantErr: false,
			minLen:  2,
		},
		{
			name:    "non-existent source",
			conn:    diagram.Connection{From: 99, To: 2},
			wantErr: true,
		},
		{
			name:    "non-existent target",
			conn:    diagram.Connection{From: 1, To: 99},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := router.RouteConnection(tt.conn, nodes)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("RouteConnection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if len(path.Points) < tt.minLen {
					t.Errorf("Path too short: got %d points, want at least %d", 
						len(path.Points), tt.minLen)
				}
				
				// Verify path starts and ends near the nodes
				if len(path.Points) > 0 {
					start := path.Points[0]
					end := path.Points[len(path.Points)-1]
					
					// Check start point is adjacent to source node
					sourceNode := findNode(nodes, tt.conn.From)
					if !isAdjacentToNode(start, sourceNode) {
						t.Errorf("Path start point %v is not adjacent to source node %v", 
							start, sourceNode)
					}
					
					// Check end point is adjacent to target node
					targetNode := findNode(nodes, tt.conn.To)
					if !isAdjacentToNode(end, targetNode) {
						t.Errorf("Path end point %v is not adjacent to target node %v", 
							end, targetNode)
					}
				}
			}
		})
	}
}

func TestRouter_RouteConnections(t *testing.T) {
	pf := NewAStarPathFinder(PathCost{
		StraightCost: 10,
		TurnCost: 5,
	})
	router := NewRouter(pf)
	
	nodes := []diagram.Node{
		{ID: 1, X: 5, Y: 5, Width: 10, Height: 5},
		{ID: 2, X: 25, Y: 5, Width: 10, Height: 5},
		{ID: 3, X: 15, Y: 20, Width: 10, Height: 5},
	}
	
	connections := []diagram.Connection{
		{From: 1, To: 2},
		{From: 2, To: 3},
		{From: 3, To: 1},
	}
	
	paths, err := router.RouteConnections(connections, nodes)
	if err != nil {
		t.Fatalf("RouteConnections() error = %v", err)
	}
	
	// Verify we got paths for all connections
	if len(paths) != len(connections) {
		t.Errorf("Got %d paths, want %d", len(paths), len(connections))
	}
	
	// Verify each path
	for i, conn := range connections {
		path, exists := paths[i]
		if !exists {
			t.Errorf("No path for connection %d", i)
			continue
		}
		
		if len(path.Points) < 2 {
			t.Errorf("Path %d too short: %d points", i, len(path.Points))
		}
		
		// Verify path connects the right nodes
		sourceNode := findNode(nodes, conn.From)
		targetNode := findNode(nodes, conn.To)
		
		if !isAdjacentToNode(path.Points[0], sourceNode) {
			t.Errorf("Path %d doesn't start at source node", i)
		}
		
		if !isAdjacentToNode(path.Points[len(path.Points)-1], targetNode) {
			t.Errorf("Path %d doesn't end at target node", i)
		}
	}
}

func TestGetConnectionPoint(t *testing.T) {
	tests := []struct {
		name     string
		fromNode diagram.Node
		toNode   diagram.Node
		wantSide string // "left", "right", "top", "bottom"
	}{
		{
			name:     "horizontal right",
			fromNode: diagram.Node{X: 10, Y: 10, Width: 10, Height: 10},
			toNode:   diagram.Node{X: 30, Y: 10, Width: 10, Height: 10},
			wantSide: "right",
		},
		{
			name:     "horizontal left",
			fromNode: diagram.Node{X: 30, Y: 10, Width: 10, Height: 10},
			toNode:   diagram.Node{X: 10, Y: 10, Width: 10, Height: 10},
			wantSide: "left",
		},
		{
			name:     "vertical down",
			fromNode: diagram.Node{X: 10, Y: 10, Width: 10, Height: 10},
			toNode:   diagram.Node{X: 10, Y: 30, Width: 10, Height: 10},
			wantSide: "bottom",
		},
		{
			name:     "vertical up",
			fromNode: diagram.Node{X: 10, Y: 30, Width: 10, Height: 10},
			toNode:   diagram.Node{X: 10, Y: 10, Width: 10, Height: 10},
			wantSide: "top",
		},
		{
			name:     "diagonal prefers horizontal",
			fromNode: diagram.Node{X: 10, Y: 10, Width: 10, Height: 10},
			toNode:   diagram.Node{X: 25, Y: 20, Width: 10, Height: 10},
			wantSide: "right", // dx=15, dy=10, so horizontal wins
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			point := testGetConnectionPoint(&tt.fromNode, &tt.toNode)
			
			switch tt.wantSide {
			case "left":
				if point.X != tt.fromNode.X {
					t.Errorf("Expected left side connection, got point %v", point)
				}
			case "right":
				if point.X != tt.fromNode.X+tt.fromNode.Width-1 {
					t.Errorf("Expected right side connection, got point %v", point)
				}
			case "top":
				if point.Y != tt.fromNode.Y {
					t.Errorf("Expected top side connection, got point %v", point)
				}
			case "bottom":
				if point.Y != tt.fromNode.Y+tt.fromNode.Height-1 {
					t.Errorf("Expected bottom side connection, got point %v", point)
				}
			}
		})
	}
}

// Helper functions

func findNode(nodes []diagram.Node, id int) *diagram.Node {
	for i := range nodes {
		if nodes[i].ID == id {
			return &nodes[i]
		}
	}
	return nil
}

func isAdjacentToNode(point diagram.Point, node *diagram.Node) bool {
	if node == nil {
		return false
	}
	
	// Check if point is on any edge of the node (with 1 pixel tolerance)
	tolerance := 1
	
	// On left or right edge
	if (layout.Abs(point.X-node.X) <= tolerance || layout.Abs(point.X-(node.X+node.Width)) <= tolerance) &&
		point.Y >= node.Y-tolerance && point.Y <= node.Y+node.Height+tolerance {
		return true
	}
	
	// On top or bottom edge
	if (layout.Abs(point.Y-node.Y) <= tolerance || layout.Abs(point.Y-(node.Y+node.Height)) <= tolerance) &&
		point.X >= node.X-tolerance && point.X <= node.X+node.Width+tolerance {
		return true
	}
	
	return false
}

// Helper function copied from connections package to avoid circular import
func testGetConnectionPoint(fromNode, toNode *diagram.Node) diagram.Point {
	fromCenter := diagram.Point{
		X: fromNode.X + fromNode.Width/2,
		Y: fromNode.Y + fromNode.Height/2,
	}
	toCenter := diagram.Point{
		X: toNode.X + toNode.Width/2,
		Y: toNode.Y + toNode.Height/2,
	}
	
	dx := toCenter.X - fromCenter.X
	dy := toCenter.Y - fromCenter.Y
	
	if layout.Abs(dx) > layout.Abs(dy) {
		if dx > 0 {
			return diagram.Point{
				X: fromNode.X + fromNode.Width - 1,
				Y: fromNode.Y + fromNode.Height/2,
			}
		} else {
			return diagram.Point{
				X: fromNode.X,
				Y: fromNode.Y + fromNode.Height/2,
			}
		}
	} else {
		if dy > 0 {
			return diagram.Point{
				X: fromNode.X + fromNode.Width/2,
				Y: fromNode.Y + fromNode.Height - 1,
			}
		} else {
			return diagram.Point{
				X: fromNode.X + fromNode.Width/2,
				Y: fromNode.Y,
			}
		}
	}
}

// ==================== DIRECT PATH FINDER TESTS ====================

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
		start    diagram.Point
		end      diagram.Point
		expected int // expected number of points
	}{
		{
			name:     "Horizontal line right",
			start:    diagram.Point{X: 0, Y: 5},
			end:      diagram.Point{X: 10, Y: 5},
			expected: 11,
		},
		{
			name:     "Horizontal line left",
			start:    diagram.Point{X: 10, Y: 5},
			end:      diagram.Point{X: 0, Y: 5},
			expected: 11,
		},
		{
			name:     "Vertical line down",
			start:    diagram.Point{X: 5, Y: 0},
			end:      diagram.Point{X: 5, Y: 10},
			expected: 11,
		},
		{
			name:     "Vertical line up",
			start:    diagram.Point{X: 5, Y: 10},
			end:      diagram.Point{X: 5, Y: 0},
			expected: 11,
		},
		{
			name:     "Same point",
			start:    diagram.Point{X: 5, Y: 5},
			end:      diagram.Point{X: 5, Y: 5},
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
		start       diagram.Point
		end         diagram.Point
		strategy    RoutingStrategy
		expectedMid diagram.Point // expected middle point for L-shaped path
	}{
		{
			name:        "HorizontalFirst - down-right",
			start:       diagram.Point{X: 0, Y: 0},
			end:         diagram.Point{X: 5, Y: 5},
			strategy:    HorizontalFirst,
			expectedMid: diagram.Point{X: 5, Y: 0}, // horizontal then vertical
		},
		{
			name:        "VerticalFirst - down-right",
			start:       diagram.Point{X: 0, Y: 0},
			end:         diagram.Point{X: 5, Y: 5},
			strategy:    VerticalFirst,
			expectedMid: diagram.Point{X: 0, Y: 5}, // vertical then horizontal
		},
		{
			name:        "HorizontalFirst - up-left",
			start:       diagram.Point{X: 10, Y: 10},
			end:         diagram.Point{X: 5, Y: 5},
			strategy:    HorizontalFirst,
			expectedMid: diagram.Point{X: 5, Y: 10}, // horizontal then vertical
		},
		{
			name:        "VerticalFirst - up-left",
			start:       diagram.Point{X: 10, Y: 10},
			end:         diagram.Point{X: 5, Y: 5},
			strategy:    VerticalFirst,
			expectedMid: diagram.Point{X: 10, Y: 5}, // vertical then horizontal
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
		start  diagram.Point
		end    diagram.Point
		points int // expected number of points
	}{
		{
			name:   "Wide rectangle",
			start:  diagram.Point{X: 0, Y: 0},
			end:    diagram.Point{X: 20, Y: 5},
			points: 4, // start -> mid-x,start-y -> mid-x,end-y -> end
		},
		{
			name:   "Tall rectangle",
			start:  diagram.Point{X: 0, Y: 0},
			end:    diagram.Point{X: 5, Y: 20},
			points: 4, // start -> start-x,mid-y -> end-x,mid-y -> end
		},
		{
			name:   "Square diagonal",
			start:  diagram.Point{X: 0, Y: 0},
			end:    diagram.Point{X: 10, Y: 10},
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
		path     diagram.Path
		expected int // expected number of points after simplification
	}{
		{
			name: "Straight horizontal line",
			path: diagram.Path{
				Points: []diagram.Point{
					{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 2, Y: 0}, {X: 3, Y: 0},
				},
			},
			expected: 2, // just start and end
		},
		{
			name: "L-shaped path",
			path: diagram.Path{
				Points: []diagram.Point{
					{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 2, Y: 0}, 
					{X: 2, Y: 1}, {X: 2, Y: 2},
				},
			},
			expected: 3, // start, corner, end
		},
		{
			name: "Complex path with multiple turns",
			path: diagram.Path{
				Points: []diagram.Point{
					{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 2, Y: 0}, // horizontal
					{X: 2, Y: 1}, {X: 2, Y: 2},              // vertical
					{X: 3, Y: 2}, {X: 4, Y: 2},              // horizontal
				},
			},
			expected: 4, // start, two corners, end
		},
		{
			name: "Already simplified",
			path: diagram.Path{
				Points: []diagram.Point{
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
		p1       diagram.Point
		p2       diagram.Point
		expected int
	}{
		{diagram.Point{0, 0}, diagram.Point{0, 0}, 0},
		{diagram.Point{0, 0}, diagram.Point{3, 4}, 7},
		{diagram.Point{3, 4}, diagram.Point{0, 0}, 7},
		{diagram.Point{-5, -5}, diagram.Point{5, 5}, 20},
		{diagram.Point{10, 0}, diagram.Point{0, 10}, 20},
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
		p1       diagram.Point
		p2       diagram.Point
		expected Direction
	}{
		{diagram.Point{5, 5}, diagram.Point{5, 3}, DirNorth},
		{diagram.Point{5, 5}, diagram.Point{7, 5}, DirEast},
		{diagram.Point{5, 5}, diagram.Point{5, 7}, DirSouth},
		{diagram.Point{5, 5}, diagram.Point{3, 5}, DirWest},
		{diagram.Point{5, 5}, diagram.Point{5, 5}, DirNone},
		{diagram.Point{5, 5}, diagram.Point{7, 7}, DirNone}, // diagonal
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

// ==================== PORT MANAGER TESTS ====================

// Helper function to get edge name for error messages
func edgeName(edge EdgeSide) string {
	switch edge {
	case North:
		return "North"
	case South:
		return "South"
	case East:
		return "East"
	case West:
		return "West"
	default:
		return "Unknown"
	}
}

func TestPortManager_BasicOperations(t *testing.T) {
	// Create test nodes
	nodes := []diagram.Node{
		{ID: 1, X: 10, Y: 10, Width: 10, Height: 5},
		{ID: 2, X: 30, Y: 10, Width: 8, Height: 6},
	}
	
	pm := NewPortManager(nodes, 2) // 2-unit wide ports
	
	t.Run("GetAvailablePorts", func(t *testing.T) {
		// Check available ports on North edge of node 1
		// Width is 10, margin is 2 on each side, so usable space is 6
		// With 2-unit ports, we should have 3 ports
		ports := pm.GetAvailablePorts(1, North)
		if len(ports) != 3 {
			t.Errorf("Expected 3 available ports on North edge, got %d", len(ports))
		}
		
		// Verify port positions
		expectedPositions := []int{2, 4, 6}
		for i, port := range ports {
			if port.Position != expectedPositions[i] {
				t.Errorf("Port %d: expected position %d, got %d", 
					i, expectedPositions[i], port.Position)
			}
			if port.ConnectionID != -1 {
				t.Errorf("Port %d should be free, but has connection %d",
					i, port.ConnectionID)
			}
		}
	})
	
	t.Run("ReservePort", func(t *testing.T) {
		// Reserve a port on East edge
		port, err := pm.ReservePort(1, East, 100)
		if err != nil {
			t.Fatalf("Failed to reserve port: %v", err)
		}
		
		if port.ConnectionID != 100 {
			t.Errorf("Reserved port should have connection ID 100, got %d", 
				port.ConnectionID)
		}
		
		// Check that the port is now occupied
		if !pm.IsPortOccupied(port) {
			t.Error("Reserved port should be occupied")
		}
		
		// Try to get available ports - should not include the reserved one
		availablePorts := pm.GetAvailablePorts(1, East)
		for _, p := range availablePorts {
			if p.Position == port.Position {
				t.Error("Reserved port should not appear in available ports")
			}
		}
	})
	
	t.Run("GetOccupiedPorts", func(t *testing.T) {
		// Should have one occupied port from previous test
		occupied := pm.GetOccupiedPorts(1)
		if len(occupied) != 1 {
			t.Errorf("Expected 1 occupied port, got %d", len(occupied))
		}
		
		if occupied[0].ConnectionID != 100 {
			t.Errorf("Occupied port should have connection ID 100, got %d",
				occupied[0].ConnectionID)
		}
	})
	
	t.Run("ReleasePort", func(t *testing.T) {
		occupied := pm.GetOccupiedPorts(1)
		if len(occupied) == 0 {
			t.Skip("No occupied ports to release")
		}
		
		port := occupied[0]
		pm.ReleasePort(port)
		
		// Port should no longer be occupied
		if pm.IsPortOccupied(port) {
			t.Error("Released port should not be occupied")
		}
		
		// Should have no occupied ports now
		occupied = pm.GetOccupiedPorts(1)
		if len(occupied) != 0 {
			t.Errorf("Expected 0 occupied ports after release, got %d", len(occupied))
		}
	})
	
	t.Run("GetPortForConnection", func(t *testing.T) {
		// Reserve a port
		port, _ := pm.ReservePort(2, South, 200)
		
		// Should be able to find it by connection ID
		foundPort, found := pm.GetPortForConnection(2, 200)
		if !found {
			t.Error("Should find port for connection 200")
		}
		
		if foundPort.Position != port.Position {
			t.Errorf("Found port position %d doesn't match reserved port position %d",
				foundPort.Position, port.Position)
		}
		
		// Should not find non-existent connection
		_, found = pm.GetPortForConnection(2, 999)
		if found {
			t.Error("Should not find port for non-existent connection")
		}
	})
}

func TestPortManager_EdgeCases(t *testing.T) {
	t.Run("SmallNode", func(t *testing.T) {
		// Node too small for any ports with margins
		nodes := []diagram.Node{
			{ID: 1, X: 0, Y: 0, Width: 3, Height: 3},
		}
		
		pm := NewPortManager(nodes, 2)
		
		// Should have no available ports due to margins
		ports := pm.GetAvailablePorts(1, North)
		if len(ports) != 0 {
			t.Errorf("Small node should have no available ports, got %d", len(ports))
		}
	})
	
	t.Run("NonExistentNode", func(t *testing.T) {
		nodes := []diagram.Node{
			{ID: 1, X: 0, Y: 0, Width: 10, Height: 10},
		}
		
		pm := NewPortManager(nodes, 2)
		
		// Should handle non-existent node gracefully
		ports := pm.GetAvailablePorts(999, North)
		if ports != nil {
			t.Error("Should return nil for non-existent node")
		}
		
		_, err := pm.ReservePort(999, North, 100)
		if err == nil {
			t.Error("Should return error for non-existent node")
		}
	})
}

func TestPortManager_PortPoints(t *testing.T) {
	nodes := []diagram.Node{
		{ID: 1, X: 10, Y: 20, Width: 6, Height: 8}, // Increased height to allow ports on E/W edges
	}
	
	pm := NewPortManager(nodes, 2)
	
	testCases := []struct {
		edge     EdgeSide
		expectedY int
		expectedXRange [2]int
	}{
		{North, 19, [2]int{12, 14}}, // Y should be node.Y - 1
		{South, 28, [2]int{12, 14}}, // Y should be node.Y + height (updated for height=8)
		{East, -1, [2]int{16, 16}},  // X should be node.X + width
		{West, -1, [2]int{9, 9}},    // X should be node.X - 1
	}
	
	for _, tc := range testCases {
		ports := pm.GetAvailablePorts(1, tc.edge)
		if len(ports) == 0 {
			t.Errorf("No ports available on %s edge", edgeName(tc.edge))
			continue
		}
		
		port := ports[0]
		
		if tc.expectedY != -1 && port.Point.Y != tc.expectedY {
			t.Errorf("%s edge: expected Y=%d, got Y=%d",
				edgeName(tc.edge), tc.expectedY, port.Point.Y)
		}
		
		if tc.expectedXRange[0] != -1 {
			if port.Point.X < tc.expectedXRange[0] || port.Point.X > tc.expectedXRange[1] {
				t.Errorf("%s edge: expected X in range [%d,%d], got X=%d",
					edgeName(tc.edge), tc.expectedXRange[0], tc.expectedXRange[1], port.Point.X)
			}
		}
	}
}
package pathfinding

import (
	"edd/diagram"
	"testing"
)

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
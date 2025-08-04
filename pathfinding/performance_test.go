package pathfinding

import (
	"edd/core"
	"fmt"
	"testing"
	"time"
)

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
			obstacles := func(p core.Point) bool {
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
			
			start := core.Point{1, 1}
			end := core.Point{s.distance - 1, s.distance - 1}
			
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
			obstacles := func(p core.Point) bool {
				if p.X == 1 && p.Y == 1 { return false } // start
				if p.X == dist-1 && p.Y == dist-1 { return false } // end
				hash := uint32(p.X*7919 + p.Y*1337)
				return hash%5 == 0
			}
			
			start := core.Point{1, 1}
			end := core.Point{dist - 1, dist - 1}
			
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
	obstacles := func(p core.Point) bool {
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
	start := core.Point{size/2, size/2}
	end := core.Point{size/2 + 15, size/2}
	
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
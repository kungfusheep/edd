package pathfinding

import (
	"edd/core"
	"testing"
	"time"
)

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
			var obstacles func(core.Point) bool
			
			if scenario.obstacles != "" {
				obstacles = parseObstacleMap(scenario.obstacles)
			} else {
				// Generate obstacles programmatically
				obstacles = generateObstacles(scenario.size, scenario.name)
			}
			
			// Test points from corners
			start := core.Point{1, 1}
			end := core.Point{scenario.size - 2, scenario.size - 2}
			
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

func generateObstacles(size int, pattern string) func(core.Point) bool {
	switch pattern {
	case "Sparse_50x50":
		// Random-looking but deterministic obstacles
		return func(p core.Point) bool {
			// Boundaries
			if p.X == 0 || p.Y == 0 || p.X == size-1 || p.Y == size-1 {
				return true
			}
			// Scattered obstacles
			return (p.X*7+p.Y*13)%17 == 0
		}
	case "Diagonal_100x100":
		// Diagonal barrier with small gaps
		return func(p core.Point) bool {
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
		return func(p core.Point) bool { return false }
	}
}

// Measure actual path computation time for complex scenarios
func TestPathComputationTime(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timing test in short mode")
	}
	
	scenarios := []struct {
		name   string
		start  core.Point
		end    core.Point
		buildObstacles func() func(core.Point) bool
	}{
		{
			name:  "Long corridor with turns",
			start: core.Point{1, 1},
			end:   core.Point{98, 98},
			buildObstacles: func() func(core.Point) bool {
				// Create a maze-like pattern
				return func(p core.Point) bool {
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
			start: core.Point{5, 5},
			end:   core.Point{95, 95},
			buildObstacles: func() func(core.Point) bool {
				// 40% of cells are obstacles
				return func(p core.Point) bool {
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
			start: core.Point{50, 50},
			end:   core.Point{51, 51},
			buildObstacles: func() func(core.Point) bool {
				// Create a spiral that forces a very long path for nearby points
				return func(p core.Point) bool {
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
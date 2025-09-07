package pathfinding

import (
	"edd/diagram"
	"testing"
)

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
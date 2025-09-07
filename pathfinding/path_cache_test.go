package pathfinding

import (
	"edd/diagram"
	"testing"
)

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
package pathfinding

import (
	"edd/core"
	"fmt"
	"sync"
	"sync/atomic"
)

// PathCacheKey represents a unique key for caching paths
type PathCacheKey struct {
	FromX, FromY int
	ToX, ToY     int
	ObstacleHash uint64 // Hash of obstacle configuration
}

// PathCache stores previously computed paths for reuse
type PathCache struct {
	mu        sync.RWMutex
	cache     map[PathCacheKey]core.Path
	maxSize   int
	hits      int64 // Use atomic operations
	misses    int64 // Use atomic operations
	evictions int64 // Use atomic operations
}

// NewPathCache creates a new path cache with the specified maximum size
func NewPathCache(maxSize int) *PathCache {
	return &PathCache{
		cache:   make(map[PathCacheKey]core.Path),
		maxSize: maxSize,
	}
}

// Get retrieves a path from the cache if it exists
func (pc *PathCache) Get(start, end core.Point, obstacleHash uint64) (core.Path, bool) {
	key := PathCacheKey{
		FromX: start.X, FromY: start.Y,
		ToX: end.X, ToY: end.Y,
		ObstacleHash: obstacleHash,
	}
	
	pc.mu.RLock()
	path, found := pc.cache[key]
	pc.mu.RUnlock()
	
	if found {
		atomic.AddInt64(&pc.hits, 1)
	} else {
		atomic.AddInt64(&pc.misses, 1)
	}
	
	return path, found
}

// Put stores a path in the cache
func (pc *PathCache) Put(start, end core.Point, obstacleHash uint64, path core.Path) {
	key := PathCacheKey{
		FromX: start.X, FromY: start.Y,
		ToX: end.X, ToY: end.Y,
		ObstacleHash: obstacleHash,
	}
	
	pc.mu.Lock()
	defer pc.mu.Unlock()
	
	// Check if we need to evict entries
	if len(pc.cache) >= pc.maxSize && pc.maxSize > 0 {
		// Simple eviction: remove the first entry found
		// In production, consider LRU or other strategies
		for k := range pc.cache {
			delete(pc.cache, k)
			atomic.AddInt64(&pc.evictions, 1)
			break
		}
	}
	
	pc.cache[key] = path
}

// Clear removes all entries from the cache
func (pc *PathCache) Clear() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	
	pc.cache = make(map[PathCacheKey]core.Path)
	atomic.StoreInt64(&pc.hits, 0)
	atomic.StoreInt64(&pc.misses, 0)
	atomic.StoreInt64(&pc.evictions, 0)
}

// Stats returns cache statistics
func (pc *PathCache) Stats() (hits, misses, evictions, size int) {
	pc.mu.RLock()
	size = len(pc.cache)
	pc.mu.RUnlock()
	
	hits = int(atomic.LoadInt64(&pc.hits))
	misses = int(atomic.LoadInt64(&pc.misses))
	evictions = int(atomic.LoadInt64(&pc.evictions))
	
	return hits, misses, evictions, size
}

// String returns a string representation of cache statistics
func (pc *PathCache) String() string {
	hits, misses, evictions, size := pc.Stats()
	hitRate := 0.0
	if total := hits + misses; total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}
	
	return fmt.Sprintf("PathCache[size=%d/%d, hits=%d, misses=%d, hitRate=%.1f%%, evictions=%d]",
		size, pc.maxSize, hits, misses, hitRate, evictions)
}

// CachedPathFinder wraps a PathFinder with caching functionality
type CachedPathFinder struct {
	finder PathFinder
	cache  *PathCache
}

// NewCachedPathFinder creates a new cached path finder
func NewCachedPathFinder(finder PathFinder, cacheSize int) *CachedPathFinder {
	return &CachedPathFinder{
		finder: finder,
		cache:  NewPathCache(cacheSize),
	}
}

// FindPath finds a path, using the cache when possible
func (cpf *CachedPathFinder) FindPath(start, end core.Point, obstacles func(core.Point) bool) (core.Path, error) {
	// Compute a simple hash of the obstacle function
	// In production, this would need a more sophisticated approach
	obstacleHash := cpf.hashObstacles(start, end, obstacles)
	
	// Check cache first
	if path, found := cpf.cache.Get(start, end, obstacleHash); found {
		return path, nil
	}
	
	// Compute path
	path, err := cpf.finder.FindPath(start, end, obstacles)
	if err != nil {
		return path, err
	}
	
	// Store in cache
	cpf.cache.Put(start, end, obstacleHash, path)
	
	return path, nil
}

// hashObstacles creates a simple hash of obstacles along the potential path
// This is a simplified version - in production, you'd want a more robust approach
func (cpf *CachedPathFinder) hashObstacles(start, end core.Point, obstacles func(core.Point) bool) uint64 {
	if obstacles == nil {
		return 0
	}
	
	// Sample a few points along the potential path area
	// This is a very simple hash and may have collisions
	var hash uint64
	minX, maxX := min(start.X, end.X), max(start.X, end.X)
	minY, maxY := min(start.Y, end.Y), max(start.Y, end.Y)
	
	// Sample grid points in the bounding box
	step := max(1, (maxX-minX+maxY-minY)/20) // Sample ~20 points
	for x := minX; x <= maxX; x += step {
		for y := minY; y <= maxY; y += step {
			if obstacles(core.Point{X: x, Y: y}) {
				// Simple hash combining
				hash = hash*31 + uint64(x)*7 + uint64(y)*13
			}
		}
	}
	
	return hash
}

// ClearCache clears the path cache
func (cpf *CachedPathFinder) ClearCache() {
	cpf.cache.Clear()
}

// CacheStats returns the cache statistics
func (cpf *CachedPathFinder) CacheStats() string {
	return cpf.cache.String()
}
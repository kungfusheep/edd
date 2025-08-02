package pathfinding

import (
	"edd/core"
	"fmt"
)

// SmartPathFinder combines multiple path finding strategies for optimal results.
// It tries simple approaches first, then falls back to more complex algorithms.
type SmartPathFinder struct {
	directFinder *DirectPathFinder
	astarFinder  *AStarPathFinder
	costs        PathCost
	padding      int // Minimum distance from obstacles
	cache        *PathCache
	cacheEnabled bool
}

// NewSmartPathFinder creates a new smart path finder.
func NewSmartPathFinder(costs PathCost) *SmartPathFinder {
	astar := NewAStarPathFinder(costs)
	astar.SetMaxNodes(100000) // Increase limit for complex paths
	
	return &SmartPathFinder{
		directFinder: NewDirectPathFinder(HorizontalFirst),
		astarFinder:  astar,
		costs:        costs,
		padding:      1,
		cache:        NewPathCache(100), // Default cache size
		cacheEnabled: true,
	}
}

// FindPath finds an optimal path using the most appropriate algorithm.
func (s *SmartPathFinder) FindPath(start, end core.Point, obstacles func(core.Point) bool) (core.Path, error) {
	if start == end {
		return core.Path{Points: []core.Point{start}, Cost: 0}, nil
	}
	
	// Check cache if enabled
	var obstacleHash uint64
	if s.cacheEnabled && s.cache != nil {
		obstacleHash = s.hashObstacles(start, end, obstacles)
		if cachedPath, found := s.cache.Get(start, end, obstacleHash); found {
			return cachedPath, nil
		}
	}
	
	// Try direct path first if no obstacles
	if obstacles == nil {
		path, err := s.directFinder.FindPath(start, end, nil)
		if err == nil && s.cacheEnabled && s.cache != nil {
			s.cache.Put(start, end, obstacleHash, path)
		}
		return path, err
	}
	
	// Check if start or end is blocked
	if obstacles(start) {
		return core.Path{}, fmt.Errorf("start point is blocked")
	}
	if obstacles(end) {
		return core.Path{}, fmt.Errorf("end point is blocked")
	}
	
	// Try different routing strategies
	strategies := []RoutingStrategy{HorizontalFirst, VerticalFirst}
	
	for _, strategy := range strategies {
		s.directFinder.strategy = strategy
		directPath, err := s.directFinder.FindPath(start, end, obstacles) // FIXED: Pass obstacles to direct finder too
		if err == nil && s.isPathClear(directPath, obstacles) {
			// Direct path works!
			optimized := s.optimizePath(directPath, obstacles)
			if s.cacheEnabled && s.cache != nil {
				s.cache.Put(start, end, obstacleHash, optimized)
			}
			return optimized, nil
		}
	}
	
	// Direct paths don't work, use A*
	astarPath, err := s.astarFinder.FindPath(start, end, obstacles)
	if err != nil {
		return core.Path{}, err
	}
	
	// Optimize the A* path
	optimized := s.optimizePath(astarPath, obstacles)
	
	// Safety check - make sure optimization didn't break the path
	if len(optimized.Points) < 2 || !s.isPathClear(optimized, obstacles) {
		// Return original A* path if optimization failed
		if s.cacheEnabled && s.cache != nil {
			s.cache.Put(start, end, obstacleHash, astarPath)
		}
		return astarPath, nil
	}
	
	if s.cacheEnabled && s.cache != nil {
		s.cache.Put(start, end, obstacleHash, optimized)
	}
	return optimized, nil
}

// isPathClear checks if a path is free of obstacles.
func (s *SmartPathFinder) isPathClear(path core.Path, obstacles func(core.Point) bool) bool {
	if len(path.Points) < 2 {
		return true
	}
	
	// Check each segment of the path
	for i := 0; i < len(path.Points)-1; i++ {
		if !s.isSegmentClear(path.Points[i], path.Points[i+1], obstacles) {
			return false
		}
	}
	
	return true
}

// optimizePath improves the visual quality of a path.
func (s *SmartPathFinder) optimizePath(path core.Path, obstacles func(core.Point) bool) core.Path {
	if len(path.Points) <= 2 {
		return path
	}
	
	// First, simplify to remove redundant points
	simplified := SimplifyPath(path)
	// Make sure we didn't lose essential points
	if len(simplified.Points) >= 2 {
		path = simplified
	}
	
	// Only try to optimize if we have enough points
	if len(path.Points) > 3 {
		// Try to reduce unnecessary detours
		path = s.reduceDogLegs(path, obstacles)
		
		// Try to align segments for cleaner appearance
		path = s.alignSegments(path, obstacles)
	}
	
	return path
}

// reduceDogLegs tries to straighten zig-zag patterns.
func (s *SmartPathFinder) reduceDogLegs(path core.Path, obstacles func(core.Point) bool) core.Path {
	if len(path.Points) < 4 {
		return path
	}
	
	points := path.Points
	improved := []core.Point{points[0]}
	
	i := 0
	for i < len(points)-1 {
		current := points[i]
		
		// Look for L-shaped patterns that can be straightened
		if i+2 < len(points) {
			next := points[i+1]
			afterNext := points[i+2]
			
			// Check if we have an L-shape
			if (current.X == next.X && next.Y == afterNext.Y) ||
				(current.Y == next.Y && next.X == afterNext.X) {
				
				// Try direct connection
				if s.canConnectDirect(current, afterNext, obstacles) {
					improved = append(improved, afterNext)
					i += 2
					continue
				}
			}
		}
		
		// Standard progression
		if i+1 < len(points) {
			improved = append(improved, points[i+1])
		}
		i++
	}
	
	return core.Path{Points: improved, Cost: path.Cost}
}

// canConnectDirect checks if two points can be connected via an L-shaped route.
func (s *SmartPathFinder) canConnectDirect(start, end core.Point, obstacles func(core.Point) bool) bool {
	// Check both L-shaped routes
	corner1 := core.Point{X: end.X, Y: start.Y}
	corner2 := core.Point{X: start.X, Y: end.Y}
	
	// Try horizontal-first route
	if s.isSegmentClear(start, corner1, obstacles) && s.isSegmentClear(corner1, end, obstacles) {
		return true
	}
	
	// Try vertical-first route
	if s.isSegmentClear(start, corner2, obstacles) && s.isSegmentClear(corner2, end, obstacles) {
		return true
	}
	
	return false
}

// tryDirectRoute attempts to find a clear L-shaped path between two points.
func (s *SmartPathFinder) tryDirectRoute(start, end core.Point, obstacles func(core.Point) bool) []core.Point {
	// Try horizontal-first
	hPath := []core.Point{start, {X: end.X, Y: start.Y}, end}
	if s.isSegmentClear(start, hPath[1], obstacles) && s.isSegmentClear(hPath[1], end, obstacles) {
		return hPath
	}
	
	// Try vertical-first
	vPath := []core.Point{start, {X: start.X, Y: end.Y}, end}
	if s.isSegmentClear(start, vPath[1], obstacles) && s.isSegmentClear(vPath[1], end, obstacles) {
		return vPath
	}
	
	return nil
}

// isSegmentClear checks if a straight line segment is clear of obstacles.
func (s *SmartPathFinder) isSegmentClear(start, end core.Point, obstacles func(core.Point) bool) bool {
	// Only works for horizontal or vertical segments
	if start.X != end.X && start.Y != end.Y {
		return false // Diagonal segments not supported
	}
	
	// Generate all points on the segment
	dx := 0
	if end.X > start.X {
		dx = 1
	} else if end.X < start.X {
		dx = -1
	}
	
	dy := 0
	if end.Y > start.Y {
		dy = 1
	} else if end.Y < start.Y {
		dy = -1
	}
	
	current := start
	// Check the starting point too
	if obstacles(current) {
		return false
	}
	
	for current != end {
		if current.X != end.X {
			current.X += dx
		}
		if current.Y != end.Y {
			current.Y += dy
		}
		
		if obstacles(current) {
			return false
		}
		
		// Check padding
		if s.padding > 0 {
			for _, neighbor := range GetNeighbors(current) {
				if obstacles(neighbor) {
					return false
				}
			}
		}
	}
	
	return true
}

// alignSegments tries to align path segments for cleaner appearance.
func (s *SmartPathFinder) alignSegments(path core.Path, obstacles func(core.Point) bool) core.Path {
	points := make([]core.Point, len(path.Points))
	copy(points, path.Points)
	
	// Try to align middle points with their neighbors
	for i := 1; i < len(points)-2; i++ {
		p0, p1, p2, p3 := points[i-1], points[i], points[i+1], points[i+2]
		
		// Check if we can align horizontally
		if p0.Y == p3.Y && abs(p1.Y-p0.Y) <= 3 && abs(p2.Y-p0.Y) <= 3 {
			// Try to flatten this section
			aligned1 := core.Point{X: p1.X, Y: p0.Y}
			aligned2 := core.Point{X: p2.X, Y: p0.Y}
			
			if !obstacles(aligned1) && !obstacles(aligned2) &&
				s.isSegmentClear(p0, aligned1, obstacles) &&
				s.isSegmentClear(aligned1, aligned2, obstacles) &&
				s.isSegmentClear(aligned2, p3, obstacles) {
				points[i] = aligned1
				points[i+1] = aligned2
			}
		}
		
		// Check if we can align vertically
		if p0.X == p3.X && abs(p1.X-p0.X) <= 3 && abs(p2.X-p0.X) <= 3 {
			// Try to straighten this section
			aligned1 := core.Point{X: p0.X, Y: p1.Y}
			aligned2 := core.Point{X: p0.X, Y: p2.Y}
			
			if !obstacles(aligned1) && !obstacles(aligned2) &&
				s.isSegmentClear(p0, aligned1, obstacles) &&
				s.isSegmentClear(aligned1, aligned2, obstacles) &&
				s.isSegmentClear(aligned2, p3, obstacles) {
				points[i] = aligned1
				points[i+1] = aligned2
			}
		}
	}
	
	// Remove any redundant points created by alignment
	finalPath := SimplifyPath(core.Path{Points: points})
	finalPath.Cost = path.Cost
	return finalPath
}

// SetPadding sets the minimum distance to maintain from obstacles.
func (s *SmartPathFinder) SetPadding(padding int) {
	s.padding = padding
}

// SetMaxNodes sets the maximum nodes for A* exploration.
func (s *SmartPathFinder) SetMaxNodes(max int) {
	s.astarFinder.SetMaxNodes(max)
}

// EnableCache enables or disables path caching.
func (s *SmartPathFinder) EnableCache(enabled bool) {
	s.cacheEnabled = enabled
}

// ClearCache clears the path cache.
func (s *SmartPathFinder) ClearCache() {
	if s.cache != nil {
		s.cache.Clear()
	}
}

// CacheStats returns cache statistics.
func (s *SmartPathFinder) CacheStats() string {
	if s.cache != nil {
		return s.cache.String()
	}
	return "Cache not initialized"
}

// hashObstacles creates a hash of obstacles that could affect the path.
func (s *SmartPathFinder) hashObstacles(start, end core.Point, obstacles func(core.Point) bool) uint64 {
	if obstacles == nil {
		return 0
	}
	
	// Expand bounding box to account for paths that might go around obstacles
	minX, maxX := min(start.X, end.X), max(start.X, end.X)
	minY, maxY := min(start.Y, end.Y), max(start.Y, end.Y)
	
	// Add margin based on the Manhattan distance (paths might detour this far)
	margin := (maxX - minX + maxY - minY) / 2
	if margin < 5 {
		margin = 5
	}
	if margin > 20 {
		margin = 20 // Cap the margin to avoid hashing too large an area
	}
	
	minX -= margin
	maxX += margin
	minY -= margin
	maxY += margin
	
	// Use finer sampling for better accuracy
	var hash uint64
	step := 1
	if (maxX-minX)*(maxY-minY) > 400 {
		// For large areas, sample less densely
		step = 2
	}
	
	// Hash obstacle positions
	for x := minX; x <= maxX; x += step {
		for y := minY; y <= maxY; y += step {
			if obstacles(core.Point{X: x, Y: y}) {
				// Mix the coordinates well to avoid collisions
				hash = hash*31 + uint64(x+1000)*7919 + uint64(y+1000)*6971
			}
		}
	}
	
	// Include the exact start and end points in the hash
	// This helps differentiate paths with same endpoints but different obstacle configs
	hash = hash*31 + uint64(start.X)*13 + uint64(start.Y)*17
	hash = hash*31 + uint64(end.X)*19 + uint64(end.Y)*23
	
	return hash
}
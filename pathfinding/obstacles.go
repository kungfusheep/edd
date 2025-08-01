package pathfinding

import (
	"edd/core"
)

// ObstacleChecker is a function that returns true if a point is blocked.
type ObstacleChecker func(core.Point) bool

// RectangleObstacle represents a rectangular obstacle (like a node).
type RectangleObstacle struct {
	X, Y          int
	Width, Height int
	Padding       int // Extra space around the rectangle
}

// Contains checks if a point is inside the rectangle (including padding).
func (r RectangleObstacle) Contains(p core.Point) bool {
	return p.X >= r.X-r.Padding &&
		p.X < r.X+r.Width+r.Padding &&
		p.Y >= r.Y-r.Padding &&
		p.Y < r.Y+r.Height+r.Padding
}

// CreateNodeObstacleChecker creates an obstacle checker for a set of nodes.
func CreateNodeObstacleChecker(nodes []core.Node, padding int) ObstacleChecker {
	rectangles := make([]RectangleObstacle, len(nodes))
	for i, node := range nodes {
		rectangles[i] = RectangleObstacle{
			X:       node.X,
			Y:       node.Y,
			Width:   node.Width,
			Height:  node.Height,
			Padding: padding,
		}
	}
	
	return func(p core.Point) bool {
		for _, rect := range rectangles {
			if rect.Contains(p) {
				return true
			}
		}
		return false
	}
}

// CreateBoundsObstacleChecker creates an obstacle checker that blocks points outside bounds.
func CreateBoundsObstacleChecker(bounds core.Bounds) ObstacleChecker {
	return func(p core.Point) bool {
		return p.X < bounds.Min.X || p.X >= bounds.Max.X ||
			p.Y < bounds.Min.Y || p.Y >= bounds.Max.Y
	}
}

// CombineObstacleCheckers combines multiple obstacle checkers with OR logic.
func CombineObstacleCheckers(checkers ...ObstacleChecker) ObstacleChecker {
	return func(p core.Point) bool {
		for _, checker := range checkers {
			if checker(p) {
				return true
			}
		}
		return false
	}
}

// PathObstacle represents an existing path that new paths should avoid crossing.
type PathObstacle struct {
	Points    []core.Point
	Thickness int // How many pixels wide the path is
}

// Contains checks if a point intersects with the path.
func (p PathObstacle) Contains(point core.Point) bool {
	// Check if point is on any segment of the path
	for i := 0; i < len(p.Points)-1; i++ {
		if p.isPointOnSegment(point, p.Points[i], p.Points[i+1]) {
			return true
		}
	}
	return false
}

// isPointOnSegment checks if a point is on a line segment (with thickness).
func (p PathObstacle) isPointOnSegment(point, start, end core.Point) bool {
	// For simplicity, check if point is within thickness distance of the line
	// This is a simplified check - for production, use proper line distance calculation
	
	// Check bounding box first
	minX := min(start.X, end.X) - p.Thickness
	maxX := max(start.X, end.X) + p.Thickness
	minY := min(start.Y, end.Y) - p.Thickness
	maxY := max(start.Y, end.Y) + p.Thickness
	
	if point.X < minX || point.X > maxX || point.Y < minY || point.Y > maxY {
		return false
	}
	
	// For horizontal or vertical lines, use simple distance
	if start.X == end.X {
		// Vertical line
		return abs(point.X-start.X) <= p.Thickness &&
			point.Y >= min(start.Y, end.Y) &&
			point.Y <= max(start.Y, end.Y)
	}
	if start.Y == end.Y {
		// Horizontal line
		return abs(point.Y-start.Y) <= p.Thickness &&
			point.X >= min(start.X, end.X) &&
			point.X <= max(start.X, end.X)
	}
	
	// For diagonal lines, this is approximate
	// A proper implementation would calculate perpendicular distance
	return false
}

// CreatePathObstacleChecker creates an obstacle checker for existing paths.
func CreatePathObstacleChecker(paths []core.Path, thickness int) ObstacleChecker {
	pathObstacles := make([]PathObstacle, len(paths))
	for i, path := range paths {
		pathObstacles[i] = PathObstacle{
			Points:    path.Points,
			Thickness: thickness,
		}
	}
	
	return func(p core.Point) bool {
		for _, obstacle := range pathObstacles {
			if obstacle.Contains(p) {
				return true
			}
		}
		return false
	}
}

// Region represents a rectangular area that can be marked as obstacle or preferred.
type Region struct {
	Bounds     core.Bounds
	IsObstacle bool     // If true, region blocks paths
	Cost       int      // Additional cost for passing through (if not obstacle)
}

// CreateRegionObstacleChecker creates an obstacle checker for regions.
func CreateRegionObstacleChecker(regions []Region) ObstacleChecker {
	obstacleRegions := []Region{}
	for _, r := range regions {
		if r.IsObstacle {
			obstacleRegions = append(obstacleRegions, r)
		}
	}
	
	return func(p core.Point) bool {
		for _, region := range obstacleRegions {
			if p.X >= region.Bounds.Min.X && p.X < region.Bounds.Max.X &&
				p.Y >= region.Bounds.Min.Y && p.Y < region.Bounds.Max.Y {
				return true
			}
		}
		return false
	}
}

// GetRegionCost returns the additional cost for a point based on regions.
func GetRegionCost(p core.Point, regions []Region) int {
	cost := 0
	for _, region := range regions {
		if !region.IsObstacle &&
			p.X >= region.Bounds.Min.X && p.X < region.Bounds.Max.X &&
			p.Y >= region.Bounds.Min.Y && p.Y < region.Bounds.Max.Y {
			cost += region.Cost
		}
	}
	return cost
}
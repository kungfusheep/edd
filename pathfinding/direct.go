package pathfinding

import (
	"edd/core"
	"edd/utils"
	"fmt"
)

// RoutingStrategy defines how direct paths are routed.
type RoutingStrategy int

const (
	// HorizontalFirst routes horizontally then vertically.
	HorizontalFirst RoutingStrategy = iota
	// VerticalFirst routes vertically then horizontally.
	VerticalFirst
	// MiddleSplit routes to the midpoint then to the target.
	MiddleSplit
)

// DirectPathFinder creates simple L-shaped paths without obstacle avoidance.
type DirectPathFinder struct {
	strategy RoutingStrategy
}

// NewDirectPathFinder creates a new direct path finder with the given strategy.
func NewDirectPathFinder(strategy RoutingStrategy) *DirectPathFinder {
	return &DirectPathFinder{strategy: strategy}
}

// FindPath returns a direct path from start to end.
// The obstacles function is ignored as this finder doesn't avoid obstacles.
func (d *DirectPathFinder) FindPath(start, end core.Point, obstacles func(core.Point) bool) (core.Path, error) {
	if start == end {
		return core.Path{Points: []core.Point{start}, Cost: 0}, nil
	}
	
	var points []core.Point
	
	switch d.strategy {
	case HorizontalFirst:
		points = d.horizontalFirstPath(start, end)
	case VerticalFirst:
		points = d.verticalFirstPath(start, end)
	case MiddleSplit:
		points = d.middleSplitPath(start, end)
	default:
		return core.Path{}, fmt.Errorf("unknown routing strategy: %v", d.strategy)
	}
	
	// Calculate cost based on Manhattan distance
	cost := ManhattanDistance(start, end) * DefaultPathCost.StraightCost
	
	// Add turn cost if there's a turn
	if len(points) == 3 && points[0].X != points[2].X && points[0].Y != points[2].Y {
		cost += DefaultPathCost.TurnCost
	}
	
	return core.Path{Points: points, Cost: cost}, nil
}

// horizontalFirstPath creates a path going horizontal then vertical.
func (d *DirectPathFinder) horizontalFirstPath(start, end core.Point) []core.Point {
	if start.Y == end.Y {
		// Same row - direct horizontal
		return d.straightLine(start, end)
	}
	if start.X == end.X {
		// Same column - direct vertical
		return d.straightLine(start, end)
	}
	
	// L-shaped path: horizontal then vertical
	corner := core.Point{X: end.X, Y: start.Y}
	return []core.Point{start, corner, end}
}

// verticalFirstPath creates a path going vertical then horizontal.
func (d *DirectPathFinder) verticalFirstPath(start, end core.Point) []core.Point {
	if start.Y == end.Y {
		// Same row - direct horizontal
		return d.straightLine(start, end)
	}
	if start.X == end.X {
		// Same column - direct vertical
		return d.straightLine(start, end)
	}
	
	// L-shaped path: vertical then horizontal
	corner := core.Point{X: start.X, Y: end.Y}
	return []core.Point{start, corner, end}
}

// middleSplitPath creates a path that goes to the midpoint between start and end.
func (d *DirectPathFinder) middleSplitPath(start, end core.Point) []core.Point {
	if start.Y == end.Y || start.X == end.X {
		// Already aligned - use straight line
		return d.straightLine(start, end)
	}
	
	// Calculate midpoint
	midX := (start.X + end.X) / 2
	midY := (start.Y + end.Y) / 2
	
	// Choose which dimension to split on based on aspect ratio
	dx := utils.Abs(end.X - start.X)
	dy := utils.Abs(end.Y - start.Y)
	
	if dx > dy {
		// Wider than tall - split vertically
		corner1 := core.Point{X: midX, Y: start.Y}
		corner2 := core.Point{X: midX, Y: end.Y}
		return []core.Point{start, corner1, corner2, end}
	} else {
		// Taller than wide - split horizontally
		corner1 := core.Point{X: start.X, Y: midY}
		corner2 := core.Point{X: end.X, Y: midY}
		return []core.Point{start, corner1, corner2, end}
	}
}

// straightLine creates a straight line path between two points.
func (d *DirectPathFinder) straightLine(start, end core.Point) []core.Point {
	points := []core.Point{start}
	
	// Determine direction
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
	
	// Generate intermediate points
	current := start
	for current != end {
		if current.X != end.X {
			current.X += dx
		}
		if current.Y != end.Y {
			current.Y += dy
		}
		points = append(points, current)
	}
	
	return points
}

// String returns a string representation of the path finder.
func (d *DirectPathFinder) String() string {
	strategies := map[RoutingStrategy]string{
		HorizontalFirst: "HorizontalFirst",
		VerticalFirst:   "VerticalFirst",
		MiddleSplit:     "MiddleSplit",
	}
	return fmt.Sprintf("DirectPathFinder{strategy=%s}", strategies[d.strategy])
}
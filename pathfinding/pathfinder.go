// Package pathfinding provides algorithms for finding paths between points in a diagram.
package pathfinding

import (
	"edd/core"
	"fmt"
	"math"
)

// PathFinder finds paths between points.
// Re-exported from core package for convenience.
type PathFinder = core.PathFinder

// PathCost defines the cost model for pathfinding algorithms.
type PathCost struct {
	StraightCost  int // Base cost for straight movement
	TurnCost      int // Penalty for changing direction
	CrossingCost  int // Penalty for crossing existing paths
	ProximityCost int // Cost modifier near obstacles (positive=avoid walls, negative=hug walls)
	DirectionBias int // Prefer certain directions (0=none, positive=horizontal, negative=vertical)
}

// DefaultPathCost provides reasonable defaults for path finding.
var DefaultPathCost = PathCost{
	StraightCost:  10,
	TurnCost:      20,
	CrossingCost:  50,
	ProximityCost: 30,
	DirectionBias: 0,
}

// EdgeHuggingPathCost provides a cost model that prefers paths along edges.
var EdgeHuggingPathCost = PathCost{
	StraightCost:  10,
	TurnCost:      20,
	CrossingCost:  50,
	ProximityCost: -5,  // Negative value encourages hugging walls
	DirectionBias: 0,
}

// ManhattanDistance calculates the Manhattan distance between two points.
func ManhattanDistance(p1, p2 core.Point) int {
	return abs(p1.X-p2.X) + abs(p1.Y-p2.Y)
}

// EuclideanDistance calculates the Euclidean distance between two points.
func EuclideanDistance(p1, p2 core.Point) float64 {
	dx := float64(p1.X - p2.X)
	dy := float64(p1.Y - p2.Y)
	return math.Sqrt(dx*dx + dy*dy)
}

// Direction represents a movement direction.
type Direction int

const (
	North Direction = iota
	East
	South
	West
	None
)

// GetDirection returns the direction from p1 to p2.
func GetDirection(p1, p2 core.Point) Direction {
	if p1.X == p2.X {
		if p1.Y < p2.Y {
			return South
		} else if p1.Y > p2.Y {
			return North
		}
	} else if p1.Y == p2.Y {
		if p1.X < p2.X {
			return East
		} else if p1.X > p2.X {
			return West
		}
	}
	return None
}

// GetNeighbors returns the 4-connected neighbors of a point.
func GetNeighbors(p core.Point) []core.Point {
	return []core.Point{
		{X: p.X, Y: p.Y - 1}, // North
		{X: p.X + 1, Y: p.Y}, // East
		{X: p.X, Y: p.Y + 1}, // South
		{X: p.X - 1, Y: p.Y}, // West
	}
}

// IsAligned checks if three points are aligned horizontally or vertically.
func IsAligned(p1, p2, p3 core.Point) bool {
	// Check horizontal alignment
	if p1.Y == p2.Y && p2.Y == p3.Y {
		return true
	}
	// Check vertical alignment
	if p1.X == p2.X && p2.X == p3.X {
		return true
	}
	return false
}

// SimplifyPath removes unnecessary waypoints from a path.
func SimplifyPath(path core.Path) core.Path {
	if len(path.Points) <= 2 {
		return path
	}
	
	simplified := []core.Point{path.Points[0]}
	
	for i := 1; i < len(path.Points)-1; i++ {
		if !IsAligned(path.Points[i-1], path.Points[i], path.Points[i+1]) {
			simplified = append(simplified, path.Points[i])
		}
	}
	
	// Always include the last point
	simplified = append(simplified, path.Points[len(path.Points)-1])
	
	return core.Path{Points: simplified, Cost: path.Cost}
}

// PathToString converts a path to a string representation for debugging.
func PathToString(path core.Path) string {
	if path.IsEmpty() {
		return "empty path"
	}
	
	result := fmt.Sprintf("Path (cost=%d): ", path.Cost)
	for i, p := range path.Points {
		if i > 0 {
			result += " → "
		}
		result += fmt.Sprintf("(%d,%d)", p.X, p.Y)
	}
	return result
}

// abs returns the absolute value of an integer.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
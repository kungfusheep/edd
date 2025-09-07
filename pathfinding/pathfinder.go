// Package pathfinding provides algorithms for finding paths between points in a diagram.
package pathfinding

import (
	"edd/diagram"
	"edd/geometry"
	"fmt"
	"math"
)

// PathFinder finds paths between points.
// Re-exported from core package for convenience.
type PathFinder = diagram.PathFinder

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
func ManhattanDistance(p1, p2 diagram.Point) int {
	return geometry.Abs(p1.X-p2.X) + geometry.Abs(p1.Y-p2.Y)
}

// EuclideanDistance calculates the Euclidean distance between two points.
func EuclideanDistance(p1, p2 diagram.Point) float64 {
	dx := float64(p1.X - p2.X)
	dy := float64(p1.Y - p2.Y)
	return math.Sqrt(dx*dx + dy*dy)
}

// Direction represents a movement direction.
type Direction int

const (
	DirNorth Direction = iota
	DirEast
	DirSouth
	DirWest
	DirNone
)

// GetDirection returns the direction from p1 to p2.
func GetDirection(p1, p2 diagram.Point) Direction {
	if p1.X == p2.X {
		if p1.Y < p2.Y {
			return DirSouth
		} else if p1.Y > p2.Y {
			return DirNorth
		}
	} else if p1.Y == p2.Y {
		if p1.X < p2.X {
			return DirEast
		} else if p1.X > p2.X {
			return DirWest
		}
	}
	return DirNone
}

// GetNeighbors returns the 4-connected neighbors of a point.
func GetNeighbors(p diagram.Point) []diagram.Point {
	return []diagram.Point{
		{X: p.X, Y: p.Y - 1}, // North
		{X: p.X + 1, Y: p.Y}, // East
		{X: p.X, Y: p.Y + 1}, // South
		{X: p.X - 1, Y: p.Y}, // West
	}
}

// GetNeighborsSymmetric returns neighbors ordered to promote symmetric paths.
// When the goal is diagonal from current position, it returns neighbors in an order
// that explores both axes equally, preventing bias toward one direction.
func GetNeighborsSymmetric(p, goal diagram.Point) []diagram.Point {
	dx := goal.X - p.X
	dy := goal.Y - p.Y
	
	neighbors := []diagram.Point{
		{X: p.X, Y: p.Y - 1}, // North
		{X: p.X + 1, Y: p.Y}, // East
		{X: p.X, Y: p.Y + 1}, // South
		{X: p.X - 1, Y: p.Y}, // West
	}
	
	// If moving diagonally, alternate between horizontal and vertical moves
	// to ensure symmetric exploration
	if dx != 0 && dy != 0 {
		// Determine primary directions
		var primary, secondary []diagram.Point
		
		// Prioritize based on which axis has more distance to cover
		if geometry.Abs(dx) > geometry.Abs(dy) {
			// Horizontal is primary
			if dx > 0 {
				primary = append(primary, diagram.Point{X: p.X + 1, Y: p.Y}) // East
			} else {
				primary = append(primary, diagram.Point{X: p.X - 1, Y: p.Y}) // West
			}
			if dy > 0 {
				secondary = append(secondary, diagram.Point{X: p.X, Y: p.Y + 1}) // South
			} else {
				secondary = append(secondary, diagram.Point{X: p.X, Y: p.Y - 1}) // North
			}
		} else if geometry.Abs(dy) > geometry.Abs(dx) {
			// Vertical is primary
			if dy > 0 {
				primary = append(primary, diagram.Point{X: p.X, Y: p.Y + 1}) // South
			} else {
				primary = append(primary, diagram.Point{X: p.X, Y: p.Y - 1}) // North
			}
			if dx > 0 {
				secondary = append(secondary, diagram.Point{X: p.X + 1, Y: p.Y}) // East
			} else {
				secondary = append(secondary, diagram.Point{X: p.X - 1, Y: p.Y}) // West
			}
		} else {
			// Equal distance on both axes - this is where we need symmetry
			// For symmetric behavior, we should explore both directions equally
			// Order based on a consistent rule to ensure determinism
			var horizontal, vertical diagram.Point
			
			if dx > 0 {
				horizontal = diagram.Point{X: p.X + 1, Y: p.Y} // East
			} else {
				horizontal = diagram.Point{X: p.X - 1, Y: p.Y} // West
			}
			
			if dy > 0 {
				vertical = diagram.Point{X: p.X, Y: p.Y + 1} // South
			} else {
				vertical = diagram.Point{X: p.X, Y: p.Y - 1} // North
			}
			
			// Return both options first, then the opposite directions
			// This ensures both paths are explored with equal priority
			return []diagram.Point{horizontal, vertical,
				{X: p.X - dx/geometry.Abs(dx), Y: p.Y}, // Opposite horizontal
				{X: p.X, Y: p.Y - dy/geometry.Abs(dy)},  // Opposite vertical
			}
		}
		
		// Add opposite directions
		remaining := []diagram.Point{}
		for _, n := range neighbors {
			if !containsPoint(primary, n) && !containsPoint(secondary, n) {
				remaining = append(remaining, n)
			}
		}
		
		// Return ordered: primary, secondary, remaining
		result := append(primary, secondary...)
		return append(result, remaining...)
	}
	
	// If moving straight, return normal order
	return neighbors
}

// containsPoint checks if a slice contains a specific point
func containsPoint(points []diagram.Point, p diagram.Point) bool {
	for _, point := range points {
		if point == p {
			return true
		}
	}
	return false
}

// IsAligned checks if three points are aligned horizontally or vertically.
func IsAligned(p1, p2, p3 diagram.Point) bool {
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
func SimplifyPath(path diagram.Path) diagram.Path {
	if len(path.Points) <= 2 {
		return path
	}
	
	simplified := []diagram.Point{path.Points[0]}
	
	for i := 1; i < len(path.Points)-1; i++ {
		if !IsAligned(path.Points[i-1], path.Points[i], path.Points[i+1]) {
			simplified = append(simplified, path.Points[i])
		}
	}
	
	// Always include the last point
	simplified = append(simplified, path.Points[len(path.Points)-1])
	
	return diagram.Path{Points: simplified, Cost: path.Cost, Metadata: path.Metadata}
}

// OptimizePath performs aggressive path optimization to minimize turns
func OptimizePath(path diagram.Path, obstacles func(diagram.Point) bool) diagram.Path {
	if len(path.Points) <= 2 {
		return path
	}
	
	// First do basic simplification
	path = SimplifyPath(path)
	
	// Try to connect non-adjacent points directly
	optimized := []diagram.Point{path.Points[0]}
	i := 0
	
	for i < len(path.Points)-1 {
		// Try to connect to the furthest point possible
		furthest := i + 1
		for j := len(path.Points) - 1; j > i+1; j-- {
			if canConnectDirectly(path.Points[i], path.Points[j], obstacles) {
				furthest = j
				break
			}
		}
		
		// Add the furthest reachable point
		optimized = append(optimized, path.Points[furthest])
		i = furthest
	}
	
	return diagram.Path{Points: optimized, Cost: path.Cost, Metadata: path.Metadata}
}

// canConnectDirectly checks if two points can be connected with a straight line
func canConnectDirectly(p1, p2 diagram.Point, obstacles func(diagram.Point) bool) bool {
	// Use Bresenham's line algorithm to check all points on the line
	dx := geometry.Abs(p2.X - p1.X)
	dy := geometry.Abs(p2.Y - p1.Y)
	
	x, y := p1.X, p1.Y
	
	xInc := 1
	if p1.X > p2.X {
		xInc = -1
	}
	
	yInc := 1
	if p1.Y > p2.Y {
		yInc = -1
	}
	
	// Special case for straight lines
	if dx == 0 {
		// Vertical line
		for y != p2.Y {
			y += yInc
			if obstacles != nil && obstacles(diagram.Point{X: x, Y: y}) {
				return false
			}
		}
		return true
	} else if dy == 0 {
		// Horizontal line
		for x != p2.X {
			x += xInc
			if obstacles != nil && obstacles(diagram.Point{X: x, Y: y}) {
				return false
			}
		}
		return true
	}
	
	// General case - we only allow horizontal/vertical connections for cleaner paths
	return false
}

// PathToString converts a path to a string representation for debugging.
func PathToString(path diagram.Path) string {
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
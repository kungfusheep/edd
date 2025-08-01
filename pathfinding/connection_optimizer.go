package pathfinding

import (
	"edd/core"
	"math"
)

// Side represents which side of a node a connection point is on
type Side int

const (
	SideTop Side = iota
	SideRight
	SideBottom
	SideLeft
)

// ConnectionPoint represents a potential connection point on a node's perimeter
type ConnectionPoint struct {
	Point core.Point
	Side  Side
	Node  *core.Node
}

// ConnectionOptimizer finds optimal connection points on node perimeters
type ConnectionOptimizer struct {
	pathFinder PathFinder
}

// NewConnectionOptimizer creates a new connection optimizer
func NewConnectionOptimizer(pathFinder PathFinder) *ConnectionOptimizer {
	return &ConnectionOptimizer{
		pathFinder: pathFinder,
	}
}

// OptimizeConnectionPoints finds the best entry/exit points for a connection between two nodes
func (c *ConnectionOptimizer) OptimizeConnectionPoints(
	fromNode, toNode *core.Node,
	obstacles func(core.Point) bool,
) (from, to core.Point) {
	// Simple implementation: choose the sides that face each other
	// and pick the center point of that side
	
	fromCenter := core.Point{
		X: fromNode.X + fromNode.Width/2,
		Y: fromNode.Y + fromNode.Height/2,
	}
	toCenter := core.Point{
		X: toNode.X + toNode.Width/2,
		Y: toNode.Y + toNode.Height/2,
	}
	
	// Determine relative positions
	dx := toCenter.X - fromCenter.X
	dy := toCenter.Y - fromCenter.Y
	
	// Choose exit point on fromNode
	if abs(dx) > abs(dy) {
		// Horizontal connection
		if dx > 0 {
			// To node is to the right
			from = core.Point{
				X: fromNode.X + fromNode.Width,
				Y: fromNode.Y + fromNode.Height/2,
			}
		} else {
			// To node is to the left
			from = core.Point{
				X: fromNode.X - 1,
				Y: fromNode.Y + fromNode.Height/2,
			}
		}
	} else {
		// Vertical connection
		if dy > 0 {
			// To node is below
			from = core.Point{
				X: fromNode.X + fromNode.Width/2,
				Y: fromNode.Y + fromNode.Height,
			}
		} else {
			// To node is above
			from = core.Point{
				X: fromNode.X + fromNode.Width/2,
				Y: fromNode.Y - 1,
			}
		}
	}
	
	// Choose entry point on toNode (opposite logic)
	dx = fromCenter.X - toCenter.X
	dy = fromCenter.Y - toCenter.Y
	
	if abs(dx) > abs(dy) {
		// Horizontal connection
		if dx > 0 {
			// From node is to the right
			to = core.Point{
				X: toNode.X + toNode.Width,
				Y: toNode.Y + toNode.Height/2,
			}
		} else {
			// From node is to the left
			to = core.Point{
				X: toNode.X - 1,
				Y: toNode.Y + toNode.Height/2,
			}
		}
	} else {
		// Vertical connection
		if dy > 0 {
			// From node is below
			to = core.Point{
				X: toNode.X + toNode.Width/2,
				Y: toNode.Y + toNode.Height,
			}
		} else {
			// From node is above
			to = core.Point{
				X: toNode.X + toNode.Width/2,
				Y: toNode.Y - 1,
			}
		}
	}
	
	// Try to find a better connection by testing a few alternatives
	from, to = c.refinConnectionPoints(fromNode, toNode, from, to, obstacles)
	
	return from, to
}

// refinConnectionPoints tries to find better connection points by testing alternatives
func (c *ConnectionOptimizer) refinConnectionPoints(
	fromNode, toNode *core.Node,
	initialFrom, initialTo core.Point,
	obstacles func(core.Point) bool,
) (from, to core.Point) {
	// If no pathfinder or obstacles, just use initial points
	if c.pathFinder == nil || obstacles == nil {
		return initialFrom, initialTo
	}
	
	// Determine which sides we're connecting from/to
	fromSide := GetConnectionSide(initialFrom, fromNode)
	toSide := GetConnectionSide(initialTo, toNode)
	
	// Generate candidate points on the appropriate sides
	fromCandidates := GenerateCandidatePoints(fromNode, fromSide)
	toCandidates := GenerateCandidatePoints(toNode, toSide)
	
	// If we don't have enough candidates, use initial points
	if len(fromCandidates) == 0 || len(toCandidates) == 0 {
		return initialFrom, initialTo
	}
	
	// Always include the initial points as candidates
	fromCandidates = append([]ConnectionPoint{{Point: initialFrom, Side: fromSide, Node: fromNode}}, fromCandidates...)
	toCandidates = append([]ConnectionPoint{{Point: initialTo, Side: toSide, Node: toNode}}, toCandidates...)
	
	// Find the best combination by testing a subset of possibilities
	bestFrom, bestTo := initialFrom, initialTo
	bestCost := math.MaxInt32
	
	// Test the center point and a few alternatives
	maxTests := min(3, len(fromCandidates))
	for i := 0; i < maxTests; i++ {
		fromPoint := fromCandidates[i].Point
		
		for j := 0; j < min(3, len(toCandidates)); j++ {
			toPoint := toCandidates[j].Point
			
			// Try to find a path between these points
			path, err := c.pathFinder.FindPath(fromPoint, toPoint, obstacles)
			if err != nil {
				continue
			}
			
			// Evaluate the path quality
			cost := len(path.Points) + countTurns(path)
			
			if cost < bestCost {
				bestCost = cost
				bestFrom = fromPoint
				bestTo = toPoint
			}
		}
	}
	
	return bestFrom, bestTo
}

// countTurns counts the number of direction changes in a path
func countTurns(path core.Path) int {
	if len(path.Points) < 3 {
		return 0
	}
	
	turns := 0
	for i := 2; i < len(path.Points); i++ {
		dir1 := GetDirection(path.Points[i-2], path.Points[i-1])
		dir2 := GetDirection(path.Points[i-1], path.Points[i])
		if dir1 != dir2 && dir1 != None && dir2 != None {
			turns++
		}
	}
	return turns
}

// OptimizeSelfConnection finds good points for a self-connecting edge
func (c *ConnectionOptimizer) OptimizeSelfConnection(node *core.Node) (from, to core.Point) {
	// For self connections, exit from right, go around, enter from bottom
	from = core.Point{
		X: node.X + node.Width,
		Y: node.Y + node.Height/3,
	}
	to = core.Point{
		X: node.X + node.Width/3,
		Y: node.Y + node.Height,
	}
	return from, to
}

// GetConnectionSide determines which side of a node a point is closest to
func GetConnectionSide(point core.Point, node *core.Node) Side {
	// Calculate distances to each side
	distTop := float64(point.Y - (node.Y - 1))
	distRight := float64((node.X + node.Width) - point.X)
	distBottom := float64((node.Y + node.Height) - point.Y)
	distLeft := float64(point.X - (node.X - 1))
	
	// Find minimum distance
	minDist := math.Min(math.Min(distTop, distRight), math.Min(distBottom, distLeft))
	
	switch minDist {
	case distTop:
		return SideTop
	case distRight:
		return SideRight
	case distBottom:
		return SideBottom
	default:
		return SideLeft
	}
}

// GenerateCandidatePoints generates all possible connection points for a node
func GenerateCandidatePoints(node *core.Node, side Side) []ConnectionPoint {
	points := []ConnectionPoint{}
	
	// Minimum gap from corners to avoid arrow rendering issues
	cornerGap := 2
	
	switch side {
	case SideTop:
		y := node.Y - 1
		startX := node.X + cornerGap
		endX := node.X + node.Width - cornerGap
		if startX < endX {
			for x := startX; x < endX; x++ {
				points = append(points, ConnectionPoint{
					Point: core.Point{X: x, Y: y},
					Side:  SideTop,
					Node:  node,
				})
			}
		}
	case SideRight:
		x := node.X + node.Width
		startY := node.Y + cornerGap
		endY := node.Y + node.Height - cornerGap
		if startY < endY {
			for y := startY; y < endY; y++ {
				points = append(points, ConnectionPoint{
					Point: core.Point{X: x, Y: y},
					Side:  SideRight,
					Node:  node,
				})
			}
		}
	case SideBottom:
		y := node.Y + node.Height
		startX := node.X + cornerGap
		endX := node.X + node.Width - cornerGap
		if startX < endX {
			for x := startX; x < endX; x++ {
				points = append(points, ConnectionPoint{
					Point: core.Point{X: x, Y: y},
					Side:  SideBottom,
					Node:  node,
				})
			}
		}
	case SideLeft:
		x := node.X - 1
		startY := node.Y + cornerGap
		endY := node.Y + node.Height - cornerGap
		if startY < endY {
			for y := startY; y < endY; y++ {
				points = append(points, ConnectionPoint{
					Point: core.Point{X: x, Y: y},
					Side:  SideLeft,
					Node:  node,
				})
			}
		}
	}
	
	return points
}
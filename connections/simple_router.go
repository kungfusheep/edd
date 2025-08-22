package connections

import (
	"edd/core"
	"edd/obstacles"
	"edd/utils"
	"fmt"
)

// SimpleRouter implements center-to-center routing with path truncation
type SimpleRouter struct {
	pathFinder      core.PathFinder
	obstacleManager obstacles.ObstacleManager
}

// NewSimpleRouter creates a new simple router
func NewSimpleRouter(pathFinder core.PathFinder, obstacleManager obstacles.ObstacleManager) *SimpleRouter {
	return &SimpleRouter{
		pathFinder:      pathFinder,
		obstacleManager: obstacleManager,
	}
}

// RouteConnection finds a path between node centers and truncates at edges
func (sr *SimpleRouter) RouteConnection(conn core.Connection, nodes []core.Node) (core.Path, error) {
	// Find source and target nodes
	var sourceNode, targetNode *core.Node
	for i := range nodes {
		if nodes[i].ID == conn.From {
			sourceNode = &nodes[i]
		}
		if nodes[i].ID == conn.To {
			targetNode = &nodes[i]
		}
	}
	
	if sourceNode == nil || targetNode == nil {
		return core.Path{}, fmt.Errorf("source or target node not found")
	}
	
	// Handle self-loops
	if conn.From == conn.To {
		return HandleSelfLoops(conn, sourceNode), nil
	}
	
	// Get node centers
	sourceCenter := core.Point{
		X: sourceNode.X + sourceNode.Width/2,
		Y: sourceNode.Y + sourceNode.Height/2,
	}
	targetCenter := core.Point{
		X: targetNode.X + targetNode.Width/2,
		Y: targetNode.Y + targetNode.Height/2,
	}
	
	// Create obstacle function that allows entry into source and target
	obstacleFunc := sr.createCenterToCenterObstacles(nodes, conn.From, conn.To)
	
	// Find path between centers
	centerPath, err := sr.pathFinder.FindPath(sourceCenter, targetCenter, obstacleFunc)
	
	if err != nil {
		return core.Path{}, fmt.Errorf("failed to find path: %w", err)
	}
	
	// Debug: print path for Load Balancer connections
	// if conn.From == 2 && (conn.To == 3 || conn.To == 4) {
	// 	fmt.Printf("\nConnection %d->%d:\n", conn.From, conn.To)
	// 	fmt.Printf("  Source node: pos=(%d,%d) size=(%dx%d) center=(%d,%d)\n", 
	// 		sourceNode.X, sourceNode.Y, sourceNode.Width, sourceNode.Height, sourceCenter.X, sourceCenter.Y)
	// 	fmt.Printf("  Target node: pos=(%d,%d) size=(%dx%d) center=(%d,%d)\n",
	// 		targetNode.X, targetNode.Y, targetNode.Width, targetNode.Height, targetCenter.X, targetCenter.Y)
	// 	fmt.Printf("  Distance: dx=%d, dy=%d, |dy|=%d\n", 
	// 		targetCenter.X - sourceCenter.X, targetCenter.Y - sourceCenter.Y, utils.Abs(targetCenter.Y - sourceCenter.Y))
	// 	
	// 	// Check the Y ranges
	// 	fmt.Printf("  Source Y range: %d to %d\n", sourceNode.Y, sourceNode.Y + sourceNode.Height - 1)
	// 	fmt.Printf("  Target Y range: %d to %d\n", targetNode.Y, targetNode.Y + targetNode.Height - 1)
	// }
	
	// Truncate path at node boundaries
	truncatedPath := sr.truncatePath(centerPath, sourceNode, targetNode)
	
	// Debug: show truncation results for Load Balancer connections
	// if conn.From == 2 && (conn.To == 3 || conn.To == 4) {
	// 	fmt.Printf("\nConnection %d->%d:\n", conn.From, conn.To)
	// 	fmt.Printf("  Center path points:")
	// 	for i, p := range centerPath.Points {
	// 		fmt.Printf(" (%d,%d)", p.X, p.Y)
	// 		if i < len(centerPath.Points)-1 {
	// 			fmt.Printf(" ->")
	// 		}
	// 	}
	// 	fmt.Printf("\n")
	// 	fmt.Printf("  After truncation: start=(%d,%d) end=(%d,%d)\n", 
	// 		truncatedPath.Points[0].X, truncatedPath.Points[0].Y,
	// 		truncatedPath.Points[len(truncatedPath.Points)-1].X, truncatedPath.Points[len(truncatedPath.Points)-1].Y)
	// }
	
	return truncatedPath, nil
}

// createCenterToCenterObstacles creates obstacles that allow entry into source/target
func (sr *SimpleRouter) createCenterToCenterObstacles(nodes []core.Node, sourceID, targetID int) func(core.Point) bool {
	return func(p core.Point) bool {
		for _, node := range nodes {
			// Skip source and target nodes - paths can enter them completely
			if node.ID == sourceID || node.ID == targetID {
				continue
			}
			
			// Check if point is inside this node (with padding)
			// Use consistent padding of 1 unit around all nodes
			// Be precise with boundaries to ensure consistency
			left := node.X - 1
			right := node.X + node.Width
			top := node.Y - 1
			bottom := node.Y + node.Height
			
			if p.X >= left && p.X <= right && p.Y >= top && p.Y <= bottom {
				return true
			}
		}
		return false
	}
}

// truncatePath removes the portions of the path inside source and target nodes
func (sr *SimpleRouter) truncatePath(path core.Path, sourceNode, targetNode *core.Node) core.Path {
	if len(path.Points) < 2 {
		return path
	}
	
	points := path.Points
	truncatedPoints := make([]core.Point, 0)
	
	// Find where path exits source node
	sourceExitPoint := core.Point{}
	sourceExitFound := false
	for i := 0; i < len(points)-1; i++ {
		curr := points[i]
		next := points[i+1]
		
		if isInsideNode(curr, sourceNode) && !isInsideNode(next, sourceNode) {
			// Found exit point - calculate intersection with node boundary
			sourceExitPoint = findNodeEdgeIntersection(curr, next, sourceNode, true)
			sourceExitFound = true
			truncatedPoints = append(truncatedPoints, sourceExitPoint)
			
			// Add remaining points outside both nodes
			for j := i + 1; j < len(points); j++ {
				if !isInsideNode(points[j], sourceNode) && !isInsideNode(points[j], targetNode) {
					truncatedPoints = append(truncatedPoints, points[j])
				} else if isInsideNode(points[j], targetNode) {
					// Found entry to target
					break
				}
			}
			break
		}
	}
	
	// Find where path enters target node
	targetEntryPoint := core.Point{}
	targetEntryFound := false
	for i := len(points) - 1; i > 0; i-- {
		curr := points[i]
		prev := points[i-1]
		
		if isInsideNode(curr, targetNode) && !isInsideNode(prev, targetNode) {
			// Found entry point - calculate intersection with node boundary
			targetEntryPoint = findNodeEdgeIntersection(prev, curr, targetNode, false)
			targetEntryFound = true
			break
		}
	}
	
	// Add target entry point if found
	if targetEntryFound {
		// Check if it's different from the last added point
		if len(truncatedPoints) == 0 || truncatedPoints[len(truncatedPoints)-1] != targetEntryPoint {
			truncatedPoints = append(truncatedPoints, targetEntryPoint)
		}
	}
	
	// Fallback if we couldn't find proper exit/entry points
	if !sourceExitFound || !targetEntryFound || len(truncatedPoints) < 2 {
		// Use simple edge-to-edge connection
		truncatedPoints = []core.Point{
			getNodeEdgePoint(sourceNode, targetNode),
			getNodeEdgePoint(targetNode, sourceNode),
		}
	}
	
	return core.Path{
		Points: truncatedPoints,
		Cost:   path.Cost,
		Metadata: map[string]interface{}{
			"sourceNode": sourceNode.ID,
			"targetNode": targetNode.ID,
		},
	}
}

// findNodeEdgeIntersection finds where a line segment intersects a node boundary
// Returns the exact point on the node edge
func findNodeEdgeIntersection(from, to core.Point, node *core.Node, isExit bool) core.Point {
	// Check which edge we're crossing
	dx := to.X - from.X
	dy := to.Y - from.Y
	
	// Node boundaries
	left := node.X
	right := node.X + node.Width - 1
	top := node.Y
	bottom := node.Y + node.Height - 1
	
	// Check intersection with each edge
	// Return the exact point on the edge
	
	// Moving horizontally
	if dx != 0 {
		// Check left edge
		if (from.X < left && to.X >= left) || (from.X >= left && to.X < left) {
			// Calculate Y at intersection
			t := float64(left - from.X) / float64(dx)
			y := from.Y + int(t*float64(dy))
			if y >= top && y <= bottom {
				return core.Point{X: left, Y: y}
			}
		}
		
		// Check right edge
		if (from.X <= right && to.X > right) || (from.X > right && to.X <= right) {
			// Calculate Y at intersection
			t := float64(right - from.X) / float64(dx)
			y := from.Y + int(t*float64(dy))
			if y >= top && y <= bottom {
				return core.Point{X: right, Y: y}
			}
		}
	}
	
	// Moving vertically
	if dy != 0 {
		// Check top edge
		if (from.Y < top && to.Y >= top) || (from.Y >= top && to.Y < top) {
			// Calculate X at intersection
			t := float64(top - from.Y) / float64(dy)
			x := from.X + int(t*float64(dx))
			if x >= left && x <= right {
				return core.Point{X: x, Y: top}
			}
		}
		
		// Check bottom edge
		if (from.Y <= bottom && to.Y > bottom) || (from.Y > bottom && to.Y <= bottom) {
			// Calculate X at intersection
			t := float64(bottom - from.Y) / float64(dy)
			x := from.X + int(t*float64(dx))
			if x >= left && x <= right {
				return core.Point{X: x, Y: bottom}
			}
		}
	}
	
	// Fallback - return the point closest to node center
	centerX := node.X + node.Width/2
	centerY := node.Y + node.Height/2
	
	// Determine which edge based on direction
	if utils.Abs(to.X - centerX) > utils.Abs(to.Y - centerY) {
		// Horizontal edge
		if to.X > centerX {
			return core.Point{X: right, Y: centerY}
		} else {
			return core.Point{X: left, Y: centerY}
		}
	} else {
		// Vertical edge
		if to.Y > centerY {
			return core.Point{X: centerX, Y: bottom}
		} else {
			return core.Point{X: centerX, Y: top}
		}
	}
}

// isInsideNode checks if a point is inside a node
func isInsideNode(p core.Point, node *core.Node) bool {
	return p.X >= node.X && p.X < node.X+node.Width &&
	       p.Y >= node.Y && p.Y < node.Y+node.Height
}

// getNodeEdgePoint finds the best edge point for connecting two nodes
func getNodeEdgePoint(fromNode, toNode *core.Node) core.Point {
	// Calculate centers
	fromCenter := core.Point{
		X: fromNode.X + fromNode.Width/2,
		Y: fromNode.Y + fromNode.Height/2,
	}
	toCenter := core.Point{
		X: toNode.X + toNode.Width/2,
		Y: toNode.Y + toNode.Height/2,
	}
	
	// Determine direction
	dx := toCenter.X - fromCenter.X
	dy := toCenter.Y - fromCenter.Y
	
	// Choose edge point based on direction
	if utils.Abs(dx) > utils.Abs(dy) {
		// Horizontal connection
		if dx > 0 {
			// Exit from right edge
			return core.Point{
				X: fromNode.X + fromNode.Width - 1,
				Y: fromNode.Y + fromNode.Height/2,
			}
		} else {
			// Exit from left edge
			return core.Point{
				X: fromNode.X,
				Y: fromNode.Y + fromNode.Height/2,
			}
		}
	} else {
		// Vertical connection
		if dy > 0 {
			// Exit from bottom edge
			return core.Point{
				X: fromNode.X + fromNode.Width/2,
				Y: fromNode.Y + fromNode.Height - 1,
			}
		} else {
			// Exit from top edge
			return core.Point{
				X: fromNode.X + fromNode.Width/2,
				Y: fromNode.Y,
			}
		}
	}
}
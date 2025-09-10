package pathfinding

import (
	"edd/diagram"
	"edd/layout"
	"fmt"
)

// AreaRouter implements area-based routing where paths go from edge to edge.
// This naturally creates symmetric paths and eliminates the need for path truncation.
type AreaRouter struct {
	pathFinder      diagram.AreaPathFinder
	obstacleManager ObstacleManager
}

// NewAreaRouter creates a new area-based router
func NewAreaRouter(pathFinder diagram.AreaPathFinder, obstacleManager ObstacleManager) *AreaRouter {
	return &AreaRouter{
		pathFinder:      pathFinder,
		obstacleManager: obstacleManager,
	}
}

// RouteConnection finds a path from source edge to target edge
func (ar *AreaRouter) RouteConnection(conn diagram.Connection, nodes []diagram.Node) (diagram.Path, error) {
	// Find source and target nodes
	var sourceNode, targetNode *diagram.Node
	for i := range nodes {
		if nodes[i].ID == conn.From {
			sourceNode = &nodes[i]
		}
		if nodes[i].ID == conn.To {
			targetNode = &nodes[i]
		}
	}
	
	if sourceNode == nil || targetNode == nil {
		return diagram.Path{}, fmt.Errorf("source or target node not found")
	}
	
	// Handle self-loops
	if conn.From == conn.To {
		return HandleSelfLoops(conn, sourceNode), nil
	}
	
	// Get obstacle function that includes virtual obstacles for ports
	obstacleFunc := ar.obstacleManager.GetObstacleFuncForConnection(nodes, conn)
	
	// For area-based routing, we find paths from source edge to target edge
	// The virtual obstacles will guide us to valid ports
	
	// Calculate centers to determine general direction
	sourceCenter := diagram.Point{
		X: sourceNode.X + sourceNode.Width/2,
		Y: sourceNode.Y + sourceNode.Height/2,
	}
	targetCenter := diagram.Point{
		X: targetNode.X + targetNode.Width/2,
		Y: targetNode.Y + targetNode.Height/2,
	}
	
	// Determine which edge of source to start from based on target direction
	dx := targetCenter.X - sourceCenter.X
	dy := targetCenter.Y - sourceCenter.Y
	
	var startPoint diagram.Point
	
	// Debug: print the calculated centers
	// fmt.Printf("Connection %d->%d:\n", conn.From, conn.To)
	// fmt.Printf("  Source node %d: pos(%d,%d) size(%dx%d) center(%d,%d)\n", 
	//     sourceNode.ID, sourceNode.X, sourceNode.Y, sourceNode.Width, sourceNode.Height, sourceCenter.X, sourceCenter.Y)
	// fmt.Printf("  Target node %d: pos(%d,%d) size(%dx%d) center(%d,%d)\n", 
	//     targetNode.ID, targetNode.X, targetNode.Y, targetNode.Width, targetNode.Height, targetCenter.X, targetCenter.Y)
	
	// Choose exit point on source edge based on direction to target
	if layout.Abs(dx) > layout.Abs(dy) {
		// Primarily horizontal movement
		if dx > 0 {
			// Exit from right edge (one unit past the actual edge for pathfinding)
			startPoint = diagram.Point{
				X: sourceNode.X + sourceNode.Width,
				Y: sourceNode.Y + sourceNode.Height/2,
			}
			// fmt.Printf("  Exiting from right edge at (%d,%d)\n", startPoint.X, startPoint.Y)
		} else {
			// Exit from left edge (one unit before the actual edge for pathfinding)
			startPoint = diagram.Point{
				X: sourceNode.X - 1,
				Y: sourceNode.Y + sourceNode.Height/2,
			}
		}
	} else {
		// Primarily vertical movement
		if dy > 0 {
			// Exit from bottom edge
			startPoint = diagram.Point{
				X: sourceNode.X + sourceNode.Width/2,
				Y: sourceNode.Y + sourceNode.Height,
			}
			// fmt.Printf("  Exiting from bottom edge at (%d,%d)\n", startPoint.X, startPoint.Y)
		} else {
			// Exit from top edge
			startPoint = diagram.Point{
				X: sourceNode.X + sourceNode.Width/2,
				Y: sourceNode.Y - 1,
			}
		}
	}
	
	// Find the path from source edge to target edge
	finalPath, err := ar.pathFinder.FindPathToArea(startPoint, *targetNode, obstacleFunc)
	if err != nil {
		// Fallback: try other edges if the chosen edge is blocked
		alternativePoints := ar.getAlternativeStartPoints(sourceNode, startPoint)
		for _, altPoint := range alternativePoints {
			altPath, altErr := ar.pathFinder.FindPathToArea(altPoint, *targetNode, obstacleFunc)
			if altErr == nil {
				finalPath = altPath
				err = nil
				break
			}
		}
		
		if err != nil {
			return diagram.Path{}, fmt.Errorf("failed to find path from any edge: %w", err)
		}
	}
	
	// Adjust the path to ensure proper visual connection to box edges
	finalPath = ar.adjustPathEndpoints(finalPath, sourceNode, targetNode)

	// Add metadata about the connection
	finalPath.Metadata = map[string]interface{}{
		"sourceNode": sourceNode.ID,
		"targetNode": targetNode.ID,
		"routerType": "area",
	}
	
	return finalPath, nil
}

// getAlternativeStartPoints returns other edge points to try if the primary fails
func (ar *AreaRouter) getAlternativeStartPoints(node *diagram.Node, exclude diagram.Point) []diagram.Point {
	points := []diagram.Point{}
	
	// Try all four edge centers, excluding the one we already tried
	candidates := []diagram.Point{
		{X: node.X + node.Width/2, Y: node.Y - 1},              // Top
		{X: node.X + node.Width, Y: node.Y + node.Height/2},   // Right
		{X: node.X + node.Width/2, Y: node.Y + node.Height},   // Bottom
		{X: node.X - 1, Y: node.Y + node.Height/2},            // Left
	}
	
	for _, p := range candidates {
		if p != exclude {
			points = append(points, p)
		}
	}
	
	return points
}

// adjustPathEndpoints ensures the path visually connects to the box edges
// by adjusting the first and last points if they are one unit away from the boxes
func (ar *AreaRouter) adjustPathEndpoints(path diagram.Path, sourceNode, targetNode *diagram.Node) diagram.Path {
	if len(path.Points) < 2 {
		return path
	}

	adjustedPoints := make([]diagram.Point, len(path.Points))
	copy(adjustedPoints, path.Points)
	
	// fmt.Printf("  Path before adjustment: start(%d,%d) end(%d,%d)\n", 
	//     path.Points[0].X, path.Points[0].Y,
	//     path.Points[len(path.Points)-1].X, path.Points[len(path.Points)-1].Y)

	// Adjust start point to be on the source box edge
	// We'll use a branch character at this position to merge with the box
	start := adjustedPoints[0]
	if len(adjustedPoints) > 1 {
		next := adjustedPoints[1]
		
		// If moving horizontally from the start
		if start.Y == next.Y {
			if start.X == sourceNode.X - 1 {
				// Starting from left side, move to box edge
				adjustedPoints[0].X = sourceNode.X
			} else if start.X == sourceNode.X + sourceNode.Width {
				// Starting from right side, move to box edge
				adjustedPoints[0].X = sourceNode.X + sourceNode.Width - 1
			}
		}
		// If moving vertically from the start
		if start.X == next.X {
			if start.Y == sourceNode.Y - 1 {
				// Starting from top side, move to box edge
				adjustedPoints[0].Y = sourceNode.Y
			} else if start.Y == sourceNode.Y + sourceNode.Height {
				// Starting from bottom side, move to box edge
				adjustedPoints[0].Y = sourceNode.Y + sourceNode.Height - 1
			}
		}
	}

	// Adjust end point to be on the target box edge
	// The pathfinding ends one unit outside the box, so we move it back by one
	end := adjustedPoints[len(adjustedPoints)-1]
	if len(adjustedPoints) > 1 {
		prev := adjustedPoints[len(adjustedPoints)-2]
		
		// If moving horizontally to the end
		if end.Y == prev.Y {
			if end.X == targetNode.X - 1 {
				// Ending at left side, move to box edge
				adjustedPoints[len(adjustedPoints)-1].X = targetNode.X
			} else if end.X == targetNode.X + targetNode.Width {
				// Ending at right side, move to box edge
				adjustedPoints[len(adjustedPoints)-1].X = targetNode.X + targetNode.Width - 1
			}
		}
		// If moving vertically to the end
		if end.X == prev.X {
			if end.Y == targetNode.Y - 1 {
				// Ending at top side, move to box edge
				adjustedPoints[len(adjustedPoints)-1].Y = targetNode.Y
			} else if end.Y == targetNode.Y + targetNode.Height {
				// Ending at bottom side, move to box edge
				adjustedPoints[len(adjustedPoints)-1].Y = targetNode.Y + targetNode.Height - 1
			}
		}
	}

	// fmt.Printf("  Path after adjustment: start(%d,%d) end(%d,%d)\n", 
	//     adjustedPoints[0].X, adjustedPoints[0].Y,
	//     adjustedPoints[len(adjustedPoints)-1].X, adjustedPoints[len(adjustedPoints)-1].Y)
	
	return diagram.Path{
		Points:   adjustedPoints,
		Cost:     path.Cost,
		Metadata: path.Metadata,
	}
}


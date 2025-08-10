package connections

import (
	"edd/core"
	"edd/obstacles"
	"fmt"
)

// AreaRouter implements area-based routing where paths go from edge to edge.
// This naturally creates symmetric paths and eliminates the need for path truncation.
type AreaRouter struct {
	pathFinder      core.AreaPathFinder
	obstacleManager obstacles.ObstacleManager
}

// NewAreaRouter creates a new area-based router
func NewAreaRouter(pathFinder core.AreaPathFinder, obstacleManager obstacles.ObstacleManager) *AreaRouter {
	return &AreaRouter{
		pathFinder:      pathFinder,
		obstacleManager: obstacleManager,
	}
}

// RouteConnection finds a path from source edge to target edge
func (ar *AreaRouter) RouteConnection(conn core.Connection, nodes []core.Node) (core.Path, error) {
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
	
	// Get obstacle function that includes virtual obstacles for ports
	obstacleFunc := ar.obstacleManager.GetObstacleFuncForConnection(nodes, conn)
	
	// For area-based routing, we find paths from source edge to target edge
	// The virtual obstacles will guide us to valid ports
	
	// Calculate centers to determine general direction
	sourceCenter := core.Point{
		X: sourceNode.X + sourceNode.Width/2,
		Y: sourceNode.Y + sourceNode.Height/2,
	}
	targetCenter := core.Point{
		X: targetNode.X + targetNode.Width/2,
		Y: targetNode.Y + targetNode.Height/2,
	}
	
	// Determine which edge of source to start from based on target direction
	dx := targetCenter.X - sourceCenter.X
	dy := targetCenter.Y - sourceCenter.Y
	
	var startPoint core.Point
	
	// Debug: print the calculated centers
	// fmt.Printf("Connection %d->%d:\n", conn.From, conn.To)
	// fmt.Printf("  Source node %d: pos(%d,%d) size(%dx%d) center(%d,%d)\n", 
	//     sourceNode.ID, sourceNode.X, sourceNode.Y, sourceNode.Width, sourceNode.Height, sourceCenter.X, sourceCenter.Y)
	// fmt.Printf("  Target node %d: pos(%d,%d) size(%dx%d) center(%d,%d)\n", 
	//     targetNode.ID, targetNode.X, targetNode.Y, targetNode.Width, targetNode.Height, targetCenter.X, targetCenter.Y)
	
	// Choose exit point on source edge based on direction to target
	if abs(dx) > abs(dy) {
		// Primarily horizontal movement
		if dx > 0 {
			// Exit from right edge (one unit past the actual edge for pathfinding)
			startPoint = core.Point{
				X: sourceNode.X + sourceNode.Width,
				Y: sourceNode.Y + sourceNode.Height/2,
			}
			// fmt.Printf("  Exiting from right edge at (%d,%d)\n", startPoint.X, startPoint.Y)
		} else {
			// Exit from left edge (one unit before the actual edge for pathfinding)
			startPoint = core.Point{
				X: sourceNode.X - 1,
				Y: sourceNode.Y + sourceNode.Height/2,
			}
		}
	} else {
		// Primarily vertical movement
		if dy > 0 {
			// Exit from bottom edge
			startPoint = core.Point{
				X: sourceNode.X + sourceNode.Width/2,
				Y: sourceNode.Y + sourceNode.Height,
			}
			// fmt.Printf("  Exiting from bottom edge at (%d,%d)\n", startPoint.X, startPoint.Y)
		} else {
			// Exit from top edge
			startPoint = core.Point{
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
			return core.Path{}, fmt.Errorf("failed to find path from any edge: %w", err)
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
func (ar *AreaRouter) getAlternativeStartPoints(node *core.Node, exclude core.Point) []core.Point {
	points := []core.Point{}
	
	// Try all four edge centers, excluding the one we already tried
	candidates := []core.Point{
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
func (ar *AreaRouter) adjustPathEndpoints(path core.Path, sourceNode, targetNode *core.Node) core.Path {
	if len(path.Points) < 2 {
		return path
	}

	adjustedPoints := make([]core.Point, len(path.Points))
	copy(adjustedPoints, path.Points)
	
	// fmt.Printf("  Path before adjustment: start(%d,%d) end(%d,%d)\n", 
	//     path.Points[0].X, path.Points[0].Y,
	//     path.Points[len(path.Points)-1].X, path.Points[len(path.Points)-1].Y)

	// Keep start point one unit away from the source box edge
	// This prevents overlap with arrow characters at the box edge
	// The pathfinding already starts one unit outside, so we don't adjust it
	// (Previously we were moving it back to the edge, causing the overlap)

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
	
	return core.Path{
		Points:   adjustedPoints,
		Cost:     path.Cost,
		Metadata: path.Metadata,
	}
}


package pathfinding

import (
	"edd/diagram"
	"fmt"
)

// FlowDirection defines the primary flow direction for diagrams
type FlowDirection int

const (
	FlowVertical   FlowDirection = iota // Top-to-bottom (default for flowcharts)
	FlowHorizontal                      // Left-to-right (for pipelines, timelines)
)

// AreaRouter implements area-based routing where paths go from edge to edge.
// This naturally creates symmetric paths and eliminates the need for path truncation.
type AreaRouter struct {
	pathFinder      diagram.AreaPathFinder
	obstacleManager ObstacleManager
	flowDirection   FlowDirection // Primary flow direction for exit point selection
}

// NewAreaRouter creates a new area-based router with vertical flow (top-to-bottom)
func NewAreaRouter(pathFinder diagram.AreaPathFinder, obstacleManager ObstacleManager) *AreaRouter {
	return &AreaRouter{
		pathFinder:      pathFinder,
		obstacleManager: obstacleManager,
		flowDirection:   FlowVertical, // Default to vertical flow
	}
}

// SetFlowDirection sets the primary flow direction for this router
func (ar *AreaRouter) SetFlowDirection(direction FlowDirection) {
	ar.flowDirection = direction
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
	

	// Choose exit point based on flow direction and target position
	if ar.flowDirection == FlowVertical {
		// Vertical flow (top-to-bottom): prefer vertical exits
		if dy != 0 {
			// ANY vertical component means we use vertical exit
			if dy > 0 {
				// Exit from bottom edge
				startPoint = diagram.Point{
					X: sourceNode.X + sourceNode.Width/2,
					Y: sourceNode.Y + sourceNode.Height,
				}
			} else {
				// Exit from top edge
				startPoint = diagram.Point{
					X: sourceNode.X + sourceNode.Width/2,
					Y: sourceNode.Y - 1,
				}
			}
		} else {
			// Purely horizontal - use horizontal exit
			if dx > 0 {
				startPoint = diagram.Point{
					X: sourceNode.X + sourceNode.Width,
					Y: sourceNode.Y + sourceNode.Height/2,
				}
			} else {
				startPoint = diagram.Point{
					X: sourceNode.X - 1,
					Y: sourceNode.Y + sourceNode.Height/2,
				}
			}
		}
	} else {
		// Horizontal flow (left-to-right): prefer horizontal exits
		if dx != 0 {
			// ANY horizontal component means we use horizontal exit
			// For horizontal exits, bias Y position toward target to spread fan-out connections
			exitY := sourceNode.Y + sourceNode.Height/2 // Default to center

			// If target is significantly above or below, adjust exit Y to spread connections
			if dy > sourceNode.Height/4 {
				// Target is below - exit from lower part of edge
				exitY = sourceNode.Y + (sourceNode.Height * 2 / 3)
			} else if dy < -sourceNode.Height/4 {
				// Target is above - exit from upper part of edge
				exitY = sourceNode.Y + (sourceNode.Height / 3)
			}

			if dx > 0 {
				// Exit from right edge
				startPoint = diagram.Point{
					X: sourceNode.X + sourceNode.Width,
					Y: exitY,
				}
			} else {
				// Exit from left edge
				startPoint = diagram.Point{
					X: sourceNode.X - 1,
					Y: exitY,
				}
			}
		} else {
			// Purely vertical - use vertical exit
			if dy > 0 {
				startPoint = diagram.Point{
					X: sourceNode.X + sourceNode.Width/2,
					Y: sourceNode.Y + sourceNode.Height,
				}
			} else {
				startPoint = diagram.Point{
					X: sourceNode.X + sourceNode.Width/2,
					Y: sourceNode.Y - 1,
				}
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


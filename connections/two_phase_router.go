package connections

import (
	"edd/core"
	"edd/obstacles"
	"fmt"
)

// TwoPhaseRouter implements routing that first finds a rough path,
// then selects ports based on the actual approach direction.
type TwoPhaseRouter struct {
	pathFinder      core.PathFinder
	obstacleManager obstacles.ObstacleManager
	portManager     obstacles.PortManager
}

// NewTwoPhaseRouter creates a router that selects ports after routing
func NewTwoPhaseRouter(pathFinder core.PathFinder, obstacleManager obstacles.ObstacleManager) *TwoPhaseRouter {
	return &TwoPhaseRouter{
		pathFinder:      pathFinder,
		obstacleManager: obstacleManager,
	}
}

// SetPortManager sets the port manager for port-based routing
func (r *TwoPhaseRouter) SetPortManager(pm obstacles.PortManager) {
	r.portManager = pm
	r.obstacleManager.SetPortManager(pm)
}

// RouteConnectionWithPorts routes a connection using two-phase approach:
// 1. Route from center to center to determine approach direction
// 2. Select and reserve ports based on actual path
// 3. Route final path to exact port positions
func (r *TwoPhaseRouter) RouteConnectionWithPorts(conn core.Connection, nodes []core.Node) (core.Path, error) {
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
	
	// Phase 1: Route rough path to determine approach directions
	_, sourceEdge, targetEdge, err := r.findRoughPath(conn, sourceNode, targetNode, nodes)
	if err != nil {
		return core.Path{}, fmt.Errorf("failed to find rough path: %w", err)
	}
	
	// Phase 2: Select and reserve ports based on approach
	sourcePort, err := r.portManager.ReservePort(sourceNode.ID, sourceEdge, conn.ID)
	if err != nil {
		return core.Path{}, fmt.Errorf("failed to reserve source port: %w", err)
	}
	
	targetPort, err := r.portManager.ReservePort(targetNode.ID, targetEdge, conn.ID)
	if err != nil {
		// Release source port before returning
		r.portManager.ReleasePort(sourcePort)
		return core.Path{}, fmt.Errorf("failed to reserve target port: %w", err)
	}
	
	// Phase 3: Route final path to exact port positions
	obstacles := r.obstacleManager.GetObstacleFuncForConnection(nodes, conn)
	finalPath, err := r.pathFinder.FindPath(sourcePort.Point, targetPort.Point, obstacles)
	if err != nil {
		// Release ports on failure
		r.portManager.ReleasePort(sourcePort)
		r.portManager.ReleasePort(targetPort)
		return core.Path{}, fmt.Errorf("failed to find final path: %w", err)
	}
	
	// Add port metadata to path
	if finalPath.Metadata == nil {
		finalPath.Metadata = make(map[string]interface{})
	}
	finalPath.Metadata["sourcePort"] = sourcePort
	finalPath.Metadata["targetPort"] = targetPort
	
	return finalPath, nil
}

// findRoughPath finds a rough path and determines approach edges
func (r *TwoPhaseRouter) findRoughPath(conn core.Connection, sourceNode, targetNode *core.Node, nodes []core.Node) (core.Path, obstacles.EdgeSide, obstacles.EdgeSide, error) {
	// For rough path, we need special obstacle handling that allows centers
	roughObstacles := r.createRoughPathObstacles(nodes, conn)
	
	// Start from node centers
	sourceCenter := core.Point{
		X: sourceNode.X + sourceNode.Width/2,
		Y: sourceNode.Y + sourceNode.Height/2,
	}
	targetCenter := core.Point{
		X: targetNode.X + targetNode.Width/2,
		Y: targetNode.Y + targetNode.Height/2,
	}
	
	// Debug: print node info
	// fmt.Printf("Source node: ID=%d, X=%d, Y=%d, W=%d, H=%d, Center=(%d,%d)\n", 
	//     sourceNode.ID, sourceNode.X, sourceNode.Y, sourceNode.Width, sourceNode.Height,
	//     sourceCenter.X, sourceCenter.Y)
	// fmt.Printf("Target node: ID=%d, X=%d, Y=%d, W=%d, H=%d, Center=(%d,%d)\n",
	//     targetNode.ID, targetNode.X, targetNode.Y, targetNode.Width, targetNode.Height,
	//     targetCenter.X, targetCenter.Y)
	
	// Find rough path
	roughPath, err := r.pathFinder.FindPath(sourceCenter, targetCenter, roughObstacles)
	if err != nil {
		return core.Path{}, 0, 0, fmt.Errorf("failed to find path from (%d,%d) to (%d,%d): %w", 
			sourceCenter.X, sourceCenter.Y, targetCenter.X, targetCenter.Y, err)
	}
	
	// Determine exit edge from source based on first few path points
	sourceEdge := r.determineExitEdge(sourceNode, roughPath.Points)
	
	// Determine entry edge to target based on last few path points
	targetEdge := r.determineEntryEdge(targetNode, roughPath.Points)
	
	return roughPath, sourceEdge, targetEdge, nil
}

// determineExitEdge analyzes the path to determine which edge it exits from
func (r *TwoPhaseRouter) determineExitEdge(node *core.Node, pathPoints []core.Point) obstacles.EdgeSide {
	if len(pathPoints) < 2 {
		return obstacles.East // Default
	}
	
	// Look at the first point outside the node
	nodeCenter := core.Point{
		X: node.X + node.Width/2,
		Y: node.Y + node.Height/2,
	}
	
	// Find first point that's clearly outside the node
	var exitPoint core.Point
	for _, p := range pathPoints[1:] {
		if p.X < node.X-1 || p.X > node.X+node.Width ||
		   p.Y < node.Y-1 || p.Y > node.Y+node.Height {
			exitPoint = p
			break
		}
	}
	
	// Determine direction based on angle
	dx := exitPoint.X - nodeCenter.X
	dy := exitPoint.Y - nodeCenter.Y
	
	// Use aspect ratio aware comparison
	if abs(dx) > abs(dy)*2 {
		// Horizontal exit
		if dx > 0 {
			return obstacles.East
		}
		return obstacles.West
	} else {
		// Vertical exit
		if dy > 0 {
			return obstacles.South
		}
		return obstacles.North
	}
}

// determineEntryEdge analyzes the path to determine which edge it enters from
func (r *TwoPhaseRouter) determineEntryEdge(node *core.Node, pathPoints []core.Point) obstacles.EdgeSide {
	if len(pathPoints) < 2 {
		return obstacles.West // Default
	}
	
	// Look at the last point outside the node
	nodeCenter := core.Point{
		X: node.X + node.Width/2,
		Y: node.Y + node.Height/2,
	}
	
	// Find last point that's clearly outside the node (traverse backwards)
	var entryPoint core.Point
	for i := len(pathPoints) - 2; i >= 0; i-- {
		p := pathPoints[i]
		if p.X < node.X-1 || p.X > node.X+node.Width ||
		   p.Y < node.Y-1 || p.Y > node.Y+node.Height {
			entryPoint = p
			break
		}
	}
	
	// Determine direction based on angle
	dx := entryPoint.X - nodeCenter.X
	dy := entryPoint.Y - nodeCenter.Y
	
	// Use aspect ratio aware comparison
	if abs(dx) > abs(dy)*2 {
		// Horizontal entry
		if dx > 0 {
			return obstacles.East
		}
		return obstacles.West
	} else {
		// Vertical entry
		if dy > 0 {
			return obstacles.South
		}
		return obstacles.North
	}
}

// Helper function to get edge center for port
func getEdgeCenterForPort(node *core.Node, edge obstacles.EdgeSide) core.Point {
	switch edge {
	case obstacles.North:
		return core.Point{X: node.X + node.Width/2, Y: node.Y - 1}
	case obstacles.South:
		return core.Point{X: node.X + node.Width/2, Y: node.Y + node.Height}
	case obstacles.East:
		return core.Point{X: node.X + node.Width, Y: node.Y + node.Height/2}
	case obstacles.West:
		return core.Point{X: node.X - 1, Y: node.Y + node.Height/2}
	}
	return core.Point{} // Should not reach here
}

// createRoughPathObstacles creates obstacles for rough path that allows centers
func (r *TwoPhaseRouter) createRoughPathObstacles(nodes []core.Node, conn core.Connection) func(core.Point) bool {
	// For rough path, only use physical obstacles without virtual zones
	// This allows finding a basic path to determine approach direction
	
	return func(p core.Point) bool {
		// Check each node
		for _, node := range nodes {
			// For source and target, don't block at all during rough path
			if node.ID == conn.From || node.ID == conn.To {
				continue
			}
			
			// For other nodes, block with padding
			padding := 1
			if p.X >= node.X-padding && p.X < node.X+node.Width+padding &&
			   p.Y >= node.Y-padding && p.Y < node.Y+node.Height+padding {
				return true
			}
		}
		return false
	}
}

// HandleSelfLoops handles connections where a node connects to itself.
// These require special routing to create a visible loop.
func HandleSelfLoops(conn core.Connection, node *core.Node) core.Path {
	// Make loop size proportional to node size
	minDimension := node.Width
	if node.Height < minDimension {
		minDimension = node.Height
	}
	
	// Loop size should be at least 3, but scale with node size
	loopSize := minDimension / 3
	if loopSize < 3 {
		loopSize = 3
	} else if loopSize > 8 {
		loopSize = 8 // Cap at reasonable maximum
	}
	
	// Determine best position based on node aspect ratio
	// For wide nodes, prefer top loop; for tall nodes, prefer right loop
	aspectRatio := float64(node.Width) / float64(node.Height)
	
	if aspectRatio > 1.5 {
		// Wide node - use top loop
		start := core.Point{
			X: node.X + node.Width/2,
			Y: node.Y,
		}
		
		points := []core.Point{
			start,
			{X: start.X, Y: start.Y - loopSize},
			{X: node.X + node.Width - 1, Y: start.Y - loopSize},
			{X: node.X + node.Width - 1, Y: node.Y + node.Height/2},
			{X: node.X + node.Width - 1, Y: node.Y + node.Height/2},
		}
		
		return core.Path{Points: points}
	} else {
		// Default: right-side loop (original behavior but adaptive size)
		start := core.Point{
			X: node.X + node.Width - 1,
			Y: node.Y + node.Height/2,
		}
		
		points := []core.Point{
			start,
			{X: start.X + loopSize, Y: start.Y},
			{X: start.X + loopSize, Y: node.Y - loopSize},
			{X: node.X + node.Width/2, Y: node.Y - loopSize},
			{X: node.X + node.Width/2, Y: node.Y},
		}
		
		return core.Path{Points: points}
	}
}
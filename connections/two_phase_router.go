package connections

import (
	"edd/core"
	"edd/obstacles"
	"edd/pathfinding"
	"fmt"
	"math"
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
	
	// Calculate centers for angle-based port selection
	sourceCenter := core.Point{
		X: sourceNode.X + sourceNode.Width/2,
		Y: sourceNode.Y + sourceNode.Height/2,
	}
	targetCenter := core.Point{
		X: targetNode.X + targetNode.Width/2,
		Y: targetNode.Y + targetNode.Height/2,
	}
	
	// Calculate preferred port positions
	// For source ports on vertical edges (East/West), prefer Y position aligned with target
	var sourcePreferredPoint core.Point
	if sourceEdge == obstacles.East || sourceEdge == obstacles.West {
		// For vertical edges, align port Y with target center Y
		if sourceEdge == obstacles.East {
			sourcePreferredPoint = core.Point{X: sourceNode.X + sourceNode.Width, Y: targetCenter.Y}
		} else {
			sourcePreferredPoint = core.Point{X: sourceNode.X - 1, Y: targetCenter.Y}
		}
	} else {
		// For horizontal edges, use angle-based positioning
		angle := calculateAngle(sourceCenter, targetCenter)
		sourcePreferredPoint = r.calculatePreferredPortPosition(sourceNode, sourceEdge, angle)
	}
	
	// For target ports on vertical edges (East/West), prefer Y position aligned with source
	var targetPreferredPoint core.Point
	if targetEdge == obstacles.East || targetEdge == obstacles.West {
		// For vertical edges, align port Y with source center Y
		if targetEdge == obstacles.East {
			targetPreferredPoint = core.Point{X: targetNode.X + targetNode.Width, Y: sourceCenter.Y}
		} else {
			targetPreferredPoint = core.Point{X: targetNode.X - 1, Y: sourceCenter.Y}
		}
	} else {
		// For horizontal edges, use center of edge
		targetPreferredPoint = getEdgeCenterForPort(targetNode, targetEdge)
	}
	
	// Phase 2: Select and reserve ports based on approach with angle hints
	sourcePort, err := r.portManager.ReservePortWithHint(sourceNode.ID, sourceEdge, conn.ID, sourcePreferredPoint)
	if err != nil {
		return core.Path{}, fmt.Errorf("failed to reserve source port: %w", err)
	}
	
	targetPort, err := r.portManager.ReservePortWithHint(targetNode.ID, targetEdge, conn.ID, targetPreferredPoint)
	if err != nil {
		// Release source port before returning
		r.portManager.ReleasePort(sourcePort)
		return core.Path{}, fmt.Errorf("failed to reserve target port: %w", err)
	}
	
	// Phase 3: Route final path to exact port positions with perpendicular exit
	obstacleFunc := r.obstacleManager.GetObstacleFuncForConnection(nodes, conn)
	
	// Always create minimal waypoints to ensure perpendicular exits
	// Use distance 2 as default, which should be enough for perpendicular exit
	sourceWaypoint := r.createPerpendicularWaypoint(sourcePort, sourceNode, 2, obstacleFunc)
	targetWaypoint := r.createPerpendicularWaypoint(targetPort, targetNode, 2, obstacleFunc)
	
	// Route through waypoints
	var finalPath core.Path
	
	// If waypoints are same as ports (no clear perpendicular path), route directly
	if sourceWaypoint == sourcePort.Point && targetWaypoint == targetPort.Point {
		finalPath, err = r.pathFinder.FindPath(sourcePort.Point, targetPort.Point, obstacleFunc)
		if err != nil {
			r.portManager.ReleasePort(sourcePort)
			r.portManager.ReleasePort(targetPort)
			return core.Path{}, fmt.Errorf("failed to find direct path: %w", err)
		}
	} else {
		// Route through waypoints: source -> sourceWaypoint -> targetWaypoint -> target
		// Build path segments
		segments := []struct {
			from, to core.Point
			name     string
		}{
			{sourcePort.Point, sourceWaypoint, "source to waypoint"},
			{sourceWaypoint, targetWaypoint, "between waypoints"},
			{targetWaypoint, targetPort.Point, "waypoint to target"},
		}
		
		// Remove unnecessary segments
		if sourceWaypoint == sourcePort.Point {
			segments = segments[1:] // Skip first segment
		}
		if targetWaypoint == targetPort.Point {
			segments = segments[:len(segments)-1] // Skip last segment
		}
		if sourceWaypoint == targetWaypoint {
			// Direct connection between ports through single waypoint
			segments = []struct {
				from, to core.Point
				name     string
			}{
				{sourcePort.Point, sourceWaypoint, "source to waypoint"},
				{targetWaypoint, targetPort.Point, "waypoint to target"},
			}
		}
		
		// Route each segment
		for i, seg := range segments {
			path, err := r.pathFinder.FindPath(seg.from, seg.to, obstacleFunc)
			if err != nil {
				r.portManager.ReleasePort(sourcePort)
				r.portManager.ReleasePort(targetPort)
				return core.Path{}, fmt.Errorf("failed to find path %s: %w", seg.name, err)
			}
			
			// Append path, avoiding duplicates at joins
			if i == 0 {
				finalPath.Points = append(finalPath.Points, path.Points...)
			} else if len(path.Points) > 1 {
				finalPath.Points = append(finalPath.Points, path.Points[1:]...)
			}
		}
	}
	
	// Optimize the path to minimize turns
	finalPath = pathfinding.OptimizePath(finalPath, obstacleFunc)
	
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
	
	// Calculate angle from source to target
	angle := calculateAngle(sourceCenter, targetCenter)
	
	// Determine edges based on angle
	// For source edge, we use the direct angle (where we're exiting TO)
	sourceEdge := selectEdgeByAngle(angle)
	
	// For target edge, we use the reverse angle (where we're entering FROM)
	targetAngle := angle + 180
	if targetAngle >= 360 {
		targetAngle -= 360
	}
	targetEdge := selectEdgeByAngle(targetAngle)
	
	return roughPath, sourceEdge, targetEdge, nil
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



// createPerpendicularWaypoint creates a waypoint that forces perpendicular exit/entry
func (r *TwoPhaseRouter) createPerpendicularWaypoint(port obstacles.Port, node *core.Node, maxDistance int, obstacleFunc func(core.Point) bool) core.Point {
	// Try different distances to find a clear waypoint
	for distance := maxDistance; distance > 0; distance-- {
		var waypoint core.Point
		switch port.Edge {
		case obstacles.North:
			waypoint = core.Point{X: port.Point.X, Y: port.Point.Y - distance}
		case obstacles.South:
			waypoint = core.Point{X: port.Point.X, Y: port.Point.Y + distance}
		case obstacles.East:
			waypoint = core.Point{X: port.Point.X + distance, Y: port.Point.Y}
		case obstacles.West:
			waypoint = core.Point{X: port.Point.X - distance, Y: port.Point.Y}
		default:
			return port.Point
		}
		
		// Check if waypoint is clear
		if !obstacleFunc(waypoint) {
			return waypoint
		}
	}
	
	// If no clear waypoint found, just return the port point
	// This will fall back to direct routing
	return port.Point
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

// calculateAngle calculates the angle from one point to another in degrees (0-360)
func calculateAngle(from, to core.Point) float64 {
	dx := float64(to.X - from.X)
	dy := float64(to.Y - from.Y)
	
	// Calculate angle in radians, then convert to degrees
	angle := math.Atan2(dy, dx) * 180 / math.Pi
	
	// Normalize to 0-360 range
	if angle < 0 {
		angle += 360
	}
	
	return angle
}

// selectEdgeByAngle determines which edge to use based on angle
func selectEdgeByAngle(angle float64) obstacles.EdgeSide {
	// Normalize angle to 0-360 range
	for angle < 0 {
		angle += 360
	}
	for angle >= 360 {
		angle -= 360
	}
	
	// Determine edge based on quadrant
	// East: -45° to 45° (315° to 45°)
	// North: 45° to 135°  
	// West: 135° to 225°
	// South: 225° to 315°
	switch {
	case angle >= 315 || angle < 45:
		return obstacles.East
	case angle >= 45 && angle < 135:
		return obstacles.North
	case angle >= 135 && angle < 225:
		return obstacles.West
	default: // 225 to 315
		return obstacles.South
	}
}

// calculatePreferredPortPosition calculates the ideal port position based on angle
func (r *TwoPhaseRouter) calculatePreferredPortPosition(node *core.Node, edge obstacles.EdgeSide, angle float64) core.Point {
	// Normalize angle
	for angle < 0 {
		angle += 360
	}
	for angle >= 360 {
		angle -= 360
	}
	
	// Calculate the ideal position on the edge based on angle
	// This helps align ports when multiple connections go in similar directions
	switch edge {
	case obstacles.North, obstacles.South:
		// For horizontal edges, position based on the horizontal component of the angle
		// Convert angle to radians for calculation
		rad := angle * math.Pi / 180
		// Calculate horizontal offset from center (-1 to 1)
		offset := math.Cos(rad)
		// Convert to position on edge
		centerX := node.X + node.Width/2
		idealX := centerX + int(offset * float64(node.Width/2-1))
		
		// Clamp to valid range
		if idealX < node.X+1 {
			idealX = node.X + 1
		} else if idealX >= node.X+node.Width-1 {
			idealX = node.X + node.Width - 2
		}
		
		if edge == obstacles.North {
			return core.Point{X: idealX, Y: node.Y - 1}
		} else {
			return core.Point{X: idealX, Y: node.Y + node.Height}
		}
		
	case obstacles.East, obstacles.West:
		// For vertical edges, position based on the vertical component of the angle
		rad := angle * math.Pi / 180
		// Calculate vertical offset from center (-1 to 1)
		offset := math.Sin(rad)
		// Convert to position on edge
		centerY := node.Y + node.Height/2
		idealY := centerY + int(offset * float64(node.Height/2-1))
		
		// Clamp to valid range
		if idealY < node.Y+1 {
			idealY = node.Y + 1
		} else if idealY >= node.Y+node.Height-1 {
			idealY = node.Y + node.Height - 2
		}
		
		if edge == obstacles.East {
			return core.Point{X: node.X + node.Width, Y: idealY}
		} else {
			return core.Point{X: node.X - 1, Y: idealY}
		}
	}
	
	// Fallback to edge center
	return getEdgeCenterForPort(node, edge)
}
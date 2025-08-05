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
	
	// Note: Path simplification removed - we should generate clean paths from the start
	
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
	exitIndex := -1
	
	// First, find where the path actually exits the node boundary
	for i, p := range pathPoints {
		// Skip points inside the node
		if p.X >= node.X && p.X < node.X+node.Width && 
		   p.Y >= node.Y && p.Y < node.Y+node.Height {
			continue
		}
		
		// Check if point is on or just outside the node boundary
		onNorthEdge := p.Y == node.Y-1 && p.X >= node.X && p.X < node.X+node.Width
		onSouthEdge := p.Y == node.Y+node.Height && p.X >= node.X && p.X < node.X+node.Width
		onEastEdge := p.X == node.X+node.Width && p.Y >= node.Y && p.Y < node.Y+node.Height
		onWestEdge := p.X == node.X-1 && p.Y >= node.Y && p.Y < node.Y+node.Height
		
		if onNorthEdge || onSouthEdge || onEastEdge || onWestEdge {
			exitPoint = p
			exitIndex = i
			
			if onNorthEdge {
				return obstacles.North
			} else if onSouthEdge {
				return obstacles.South
			} else if onEastEdge {
				// Check if path turns down soon after exiting east
				turnsDown := false
				for j := i+1; j < len(pathPoints) && j <= i+10; j++ {
					if pathPoints[j].Y > p.Y+1 {
						turnsDown = true
						break
					}
					// Stop if path goes too far horizontally (more than 10 units)
					if pathPoints[j].X > p.X+10 {
						break
					}
				}
				if turnsDown {
					return obstacles.South
				}
				return obstacles.East
			} else if onWestEdge {
				// Check if path turns down soon after exiting west
				turnsDown := false
				for j := i+1; j < len(pathPoints) && j <= i+10; j++ {
					if pathPoints[j].Y > p.Y+1 {
						turnsDown = true
						break
					}
					// Stop if path goes too far horizontally
					if abs(pathPoints[j].X - p.X) > 10 {
						break
					}
				}
				if turnsDown {
					return obstacles.South
				}
				return obstacles.West
			}
			break
		}
	}
	
	// Fallback: find first point clearly outside
	if exitIndex == -1 {
		for i, p := range pathPoints[1:] {
			if p.X < node.X-1 || p.X > node.X+node.Width ||
			   p.Y < node.Y-1 || p.Y > node.Y+node.Height {
				exitPoint = p
				exitIndex = i + 1
				break
			}
		}
	}
	
	// Determine direction based on angle
	dx := exitPoint.X - nodeCenter.X
	dy := exitPoint.Y - nodeCenter.Y
	
	// Calculate aspect ratio for more balanced edge selection
	// Add 1 to avoid division by zero
	aspectRatio := float64(abs(dx)) / float64(abs(dy)+1)
	
	// Use more balanced thresholds
	if aspectRatio > 1.5 {
		// Clearly horizontal exit
		if dx > 0 {
			return obstacles.East
		}
		return obstacles.West
	} else if aspectRatio < 0.67 { // Reciprocal of 1.5
		// Clearly vertical exit
		if dy > 0 {
			return obstacles.South
		}
		return obstacles.North
	} else {
		// Diagonal path - use congestion as tie-breaker
		if r.portManager != nil {
			// Count occupied ports on each edge
			occupiedPorts := r.portManager.GetOccupiedPorts(node.ID)
			eastCount, southCount, westCount, northCount := 0, 0, 0, 0
			
			for _, port := range occupiedPorts {
				switch port.Edge {
				case obstacles.East:
					eastCount++
				case obstacles.South:
					southCount++
				case obstacles.West:
					westCount++
				case obstacles.North:
					northCount++
				}
			}
			
			// For diagonal paths, prefer the less congested edge
			if dx > 0 && dy > 0 {
				// Southeast direction - choose between East and South
				if southCount < eastCount {
					return obstacles.South
				}
				return obstacles.East
			} else if dx < 0 && dy > 0 {
				// Southwest direction - choose between West and South
				if southCount < westCount {
					return obstacles.South
				}
				return obstacles.West
			} else if dx < 0 && dy < 0 {
				// Northwest direction - choose between West and North
				if northCount < westCount {
					return obstacles.North
				}
				return obstacles.West
			} else {
				// Northeast direction - choose between East and North
				if northCount < eastCount {
					return obstacles.North
				}
				return obstacles.East
			}
		}
		
		// Fallback to simple horizontal/vertical decision
		if abs(dx) > abs(dy) {
			if dx > 0 {
				return obstacles.East
			}
			return obstacles.West
		} else {
			if dy > 0 {
				return obstacles.South
			}
			return obstacles.North
		}
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
	
	// Calculate aspect ratio for more balanced edge selection
	// Add 1 to avoid division by zero
	aspectRatio := float64(abs(dx)) / float64(abs(dy)+1)
	
	// Use more balanced thresholds
	if aspectRatio > 1.5 {
		// Clearly horizontal entry
		if dx > 0 {
			return obstacles.East
		}
		return obstacles.West
	} else if aspectRatio < 0.67 { // Reciprocal of 1.5
		// Clearly vertical entry
		if dy > 0 {
			return obstacles.South
		}
		return obstacles.North
	} else {
		// Diagonal path - use congestion as tie-breaker
		if r.portManager != nil {
			// Count occupied ports on each edge
			occupiedPorts := r.portManager.GetOccupiedPorts(node.ID)
			eastCount, southCount, westCount, northCount := 0, 0, 0, 0
			
			for _, port := range occupiedPorts {
				switch port.Edge {
				case obstacles.East:
					eastCount++
				case obstacles.South:
					southCount++
				case obstacles.West:
					westCount++
				case obstacles.North:
					northCount++
				}
			}
			
			// For diagonal paths, prefer the less congested edge
			if dx > 0 && dy > 0 {
				// Southeast direction - choose between East and South
				if southCount < eastCount {
					return obstacles.South
				}
				return obstacles.East
			} else if dx < 0 && dy > 0 {
				// Southwest direction - choose between West and South
				if southCount < westCount {
					return obstacles.South
				}
				return obstacles.West
			} else if dx < 0 && dy < 0 {
				// Northwest direction - choose between West and North
				if northCount < westCount {
					return obstacles.North
				}
				return obstacles.West
			} else {
				// Northeast direction - choose between East and North
				if northCount < eastCount {
					return obstacles.North
				}
				return obstacles.East
			}
		}
		
		// Fallback to simple horizontal/vertical decision
		if abs(dx) > abs(dy) {
			if dx > 0 {
				return obstacles.East
			}
			return obstacles.West
		} else {
			if dy > 0 {
				return obstacles.South
			}
			return obstacles.North
		}
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
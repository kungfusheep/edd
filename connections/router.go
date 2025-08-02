package connections

import (
	"edd/core"
	"fmt"
)

// Router handles the routing of connections between nodes in a diagram.
type Router struct {
	pathFinder core.PathFinder
}

// NewRouter creates a new connection router.
func NewRouter(pathFinder core.PathFinder) *Router {
	return &Router{
		pathFinder: pathFinder,
	}
}

// RouteConnection finds the best path for a connection between two nodes.
// It returns a Path that avoids obstacles and creates clean routes.
func (r *Router) RouteConnection(conn core.Connection, nodes []core.Node) (core.Path, error) {
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
	
	if sourceNode == nil {
		return core.Path{}, fmt.Errorf("source node %d not found", conn.From)
	}
	if targetNode == nil {
		return core.Path{}, fmt.Errorf("target node %d not found", conn.To)
	}
	
	// Handle self-loops specially
	if conn.From == conn.To {
		return HandleSelfLoops(conn, sourceNode), nil
	}
	
	// Get connection points for the nodes (1 unit away from edges)
	sourcePoint := getConnectionPoint(sourceNode, targetNode)
	targetPoint := getConnectionPoint(targetNode, sourceNode)
	
	// Find path between the connection points
	// Create obstacles function that avoids the node interiors
	obstacles := createObstaclesFunction(nodes, sourceNode.ID, targetNode.ID)
	
	path, err := r.pathFinder.FindPath(sourcePoint, targetPoint, obstacles)
	if err != nil {
		return core.Path{}, fmt.Errorf("failed to find path: %w", err)
	}
	
	return path, nil
}

// RouteConnections routes multiple connections, handling grouping and optimization.
func (r *Router) RouteConnections(connections []core.Connection, nodes []core.Node) (map[int]core.Path, error) {
	paths := make(map[int]core.Path)
	
	// Group connections by endpoints
	groups := GroupConnections(connections)
	
	// Route each group
	for _, group := range groups {
		if shouldBundle(group) {
			// Many connections - use bundling
			bundledPaths, err := BundleConnections(group, nodes, r)
			if err != nil {
				return nil, fmt.Errorf("failed to bundle group %s: %w", group.Key, err)
			}
			// Merge bundled paths into main map
			for idx, path := range bundledPaths {
				paths[idx] = path
			}
		} else if len(group.Connections) > 1 {
			// Multiple connections between same nodes - optimize with spreading
			optimizedPaths, err := OptimizeGroupedPaths(group, nodes, r)
			if err != nil {
				return nil, fmt.Errorf("failed to optimize group %s: %w", group.Key, err)
			}
			// Merge optimized paths into main map
			for idx, path := range optimizedPaths {
				paths[idx] = path
			}
		} else {
			// Single connection - route normally
			idx := group.Indices[0]
			path, err := r.RouteConnection(group.Connections[0], nodes)
			if err != nil {
				return nil, fmt.Errorf("failed to route connection %d (%d->%d): %w", 
					idx, group.Connections[0].From, group.Connections[0].To, err)
			}
			paths[idx] = path
		}
	}
	
	return paths, nil
}

// getConnectionPoint determines the best connection point on a node for connecting to another node.
// This creates cleaner diagrams by choosing appropriate sides of boxes.
// Connection points are placed ON the box edges for proper connection termination.
func getConnectionPoint(fromNode, toNode *core.Node) core.Point {
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
	
	// Choose connection point based on direction
	// Connection points are placed ON the box edges for proper connection termination
	// Prefer horizontal connections over vertical when possible
	if abs(dx) > abs(dy) {
		// Horizontal connection
		if dx > 0 {
			// Connect from right side (on the edge)
			return core.Point{
				X: fromNode.X + fromNode.Width - 1,
				Y: fromNode.Y + fromNode.Height/2,
			}
		} else {
			// Connect from left side (on the edge)
			return core.Point{
				X: fromNode.X,
				Y: fromNode.Y + fromNode.Height/2,
			}
		}
	} else {
		// Vertical connection
		if dy > 0 {
			// Connect from bottom (on the edge)
			return core.Point{
				X: fromNode.X + fromNode.Width/2,
				Y: fromNode.Y + fromNode.Height - 1,
			}
		} else {
			// Connect from top (on the edge)
			return core.Point{
				X: fromNode.X + fromNode.Width/2,
				Y: fromNode.Y,
			}
		}
	}
}

// getEdgePoint returns the exact edge point of a box for junction creation.
// This is the point ON the box edge where the connection meets it.
func getEdgePoint(fromNode, toNode *core.Node) core.Point {
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
	// Edge points are ON the box edges for proper junction creation
	if abs(dx) > abs(dy) {
		// Horizontal connection
		if dx > 0 {
			// Connect from right side (at the edge)
			return core.Point{
				X: fromNode.X + fromNode.Width - 1,
				Y: fromNode.Y + fromNode.Height/2,
			}
		} else {
			// Connect from left side (at the edge)
			return core.Point{
				X: fromNode.X,
				Y: fromNode.Y + fromNode.Height/2,
			}
		}
	} else {
		// Vertical connection
		if dy > 0 {
			// Connect from bottom (at the edge)
			return core.Point{
				X: fromNode.X + fromNode.Width/2,
				Y: fromNode.Y + fromNode.Height - 1,
			}
		} else {
			// Connect from top (at the edge)
			return core.Point{
				X: fromNode.X + fromNode.Width/2,
				Y: fromNode.Y,
			}
		}
	}
}

// abs returns the absolute value of an integer.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// createObstaclesFunction creates an obstacle checking function that considers
// node interiors as obstacles, except for the source and target nodes.
func createObstaclesFunction(nodes []core.Node, sourceID, targetID int) func(core.Point) bool {
	return createObstaclesFunctionWithVirtualObstacles(nodes, sourceID, targetID, 2, true)
}

// createObstaclesFunctionWithVirtualObstacles creates an enhanced obstacle checking function 
// that includes virtual obstacles around boxes to control connection aesthetics.
func createObstaclesFunctionWithVirtualObstacles(nodes []core.Node, sourceID, targetID int, padding int, enableVirtualObstacles bool) func(core.Point) bool {
	return func(p core.Point) bool {
		for _, node := range nodes {
			// For source and target nodes, only block the interior (not edges)
			if node.ID == sourceID || node.ID == targetID {
				// Allow points on the edge but not inside
				if p.X > node.X && p.X < node.X+node.Width-1 &&
				   p.Y > node.Y && p.Y < node.Y+node.Height-1 {
					return true
				}
				// Add virtual obstacles around source/target for aesthetic control
				if enableVirtualObstacles {
					if isInVirtualObstacleZone(p, node, sourceID, targetID) {
						return true
					}
				}
				continue
			}
			
			// For other nodes, block with padding using proper boundaries
			if p.X >= node.X-padding && p.X < node.X+node.Width+padding &&
			   p.Y >= node.Y-padding && p.Y < node.Y+node.Height+padding {
				return true
			}
			
			// Add virtual obstacles around other nodes for aesthetic control
			if enableVirtualObstacles {
				if isInVirtualObstacleZone(p, node, sourceID, targetID) {
					return true
				}
			}
		}
		return false
	}
}

// createObstaclesFunctionWithPadding creates an obstacle checking function with configurable padding.
func createObstaclesFunctionWithPadding(nodes []core.Node, sourceID, targetID int, padding int) func(core.Point) bool {
	return func(p core.Point) bool {
		for _, node := range nodes {
			// For source and target nodes, only block the interior (not edges)
			if node.ID == sourceID || node.ID == targetID {
				// Allow points on the edge but not inside
				if p.X > node.X && p.X < node.X+node.Width-1 &&
				   p.Y > node.Y && p.Y < node.Y+node.Height-1 {
					return true
				}
				continue
			}
			
			// For other nodes, block with padding using proper boundaries
			// Fixed: Use < instead of <= for upper bounds (half-open interval)
			if p.X >= node.X-padding && p.X < node.X+node.Width+padding &&
			   p.Y >= node.Y-padding && p.Y < node.Y+node.Height+padding {
				return true
			}
		}
		return false
	}
}

// isInVirtualObstacleZone checks if a point is in a virtual obstacle zone around a node.
// Virtual obstacles create "approach zones" that force connections to approach boxes orthogonally,
// preventing diagonal lines near boxes and improving visual aesthetics.
func isInVirtualObstacleZone(p core.Point, node core.Node, sourceID, targetID int) bool {
	// Define approach zone parameters
	const (
		approachZoneSize = 1 // Size of the approach zone around each box
		cornerRadius     = 2 // Radius for corner exclusion zones
	)
	
	// DEBUG: Log virtual obstacle checks for debugging
	debugVirtualObstacles := false // Set to true for debugging
	if debugVirtualObstacles {
		fmt.Printf("Virtual obstacle check: point(%d,%d) node[%d](%d,%d,%dx%d) source=%d target=%d\n", 
			p.X, p.Y, node.ID, node.X, node.Y, node.Width, node.Height, sourceID, targetID)
	}
	
	// FIXED: For the current connection's source/target nodes, allow more flexible access
	// For other nodes, apply virtual obstacles to force orthogonal approaches
	if node.ID == sourceID || node.ID == targetID {
		// For source/target nodes, only apply minimal virtual obstacles
		// Allow the connection to work but still prevent extremely close diagonal approaches
		
		// Allow points that are the exact connection points (on the box edges)
		connectionPoints := []core.Point{
			{X: node.X, Y: node.Y + node.Height/2},          // Left connection point
			{X: node.X + node.Width - 1, Y: node.Y + node.Height/2}, // Right connection point
			{X: node.X + node.Width/2, Y: node.Y},           // Top connection point
			{X: node.X + node.Width/2, Y: node.Y + node.Height - 1}, // Bottom connection point
		}
		
		for _, cp := range connectionPoints {
			if p.X == cp.X && p.Y == cp.Y {
				return false // Allow exact connection points
			}
		}
		
		// For source/target nodes, only block very close diagonal approaches
		if abs(p.X - (node.X + node.Width/2)) <= 1 && abs(p.Y - (node.Y + node.Height/2)) <= 1 {
			// Very close to node center - check if it's a diagonal approach that we should block
			if p.X != node.X + node.Width/2 && p.Y != node.Y + node.Height/2 {
				return true // Block diagonal approaches very close to the node
			}
		}
		
		return false // Otherwise allow for source/target nodes
	}
	
	// For OTHER nodes (not involved in this connection), create approach corridors
	// These virtual obstacles force orthogonal approaches while preserving access corridors
	
	// Calculate approach zone boundaries  
	leftBoundary := node.X - approachZoneSize
	rightBoundary := node.X + node.Width + approachZoneSize - 1
	topBoundary := node.Y - approachZoneSize  
	bottomBoundary := node.Y + node.Height + approachZoneSize - 1
	
	// Check if point is within the approach zone
	if p.X < leftBoundary || p.X > rightBoundary || 
	   p.Y < topBoundary || p.Y > bottomBoundary {
		return false // Outside approach zone - no obstacles
	}
	
	// Create orthogonal approach corridors to each side of the box
	// Allow corridors that lead directly to connection points
	
	// Horizontal corridors (left and right approaches)
	midY := node.Y + node.Height/2
	if p.Y == midY {
		// Allow direct horizontal approach corridors at box center height
		if p.X == node.X || p.X == node.X+node.Width-1 {
			return false // Allow connection points
		}
		if (p.X < node.X && p.X >= leftBoundary) || 
		   (p.X > node.X+node.Width-1 && p.X <= rightBoundary) {
			return false // Allow horizontal approach corridors
		}
	}
	
	// Vertical corridors (top and bottom approaches)  
	midX := node.X + node.Width/2
	if p.X == midX {
		// Allow direct vertical approach corridors at box center width
		if p.Y == node.Y || p.Y == node.Y+node.Height-1 {
			return false // Allow connection points
		}
		if (p.Y < node.Y && p.Y >= topBoundary) ||
		   (p.Y > node.Y+node.Height-1 && p.Y <= bottomBoundary) {
			return false // Allow vertical approach corridors
		}
	}
	
	// If we reach here, the point is in the approach zone but not in any allowed corridor
	// Block it to force use of the orthogonal approach corridors
	return true
	
	// Note: The corner exclusion logic below is now unreachable, but keeping for reference
	// Block diagonal approaches near corners
	// Top-left corner exclusion
	if p.X <= node.X && p.Y <= node.Y {
		dx := node.X - p.X
		dy := node.Y - p.Y
		if dx + dy <= cornerRadius {
			return true // Block diagonal approach to corner
		}
	}
	
	// Top-right corner exclusion
	if p.X >= node.X+node.Width-1 && p.Y <= node.Y {
		dx := p.X - (node.X + node.Width - 1)
		dy := node.Y - p.Y
		if dx + dy <= cornerRadius {
			return true // Block diagonal approach to corner
		}
	}
	
	// Bottom-left corner exclusion  
	if p.X <= node.X && p.Y >= node.Y+node.Height-1 {
		dx := node.X - p.X
		dy := p.Y - (node.Y + node.Height - 1)
		if dx + dy <= cornerRadius {
			return true // Block diagonal approach to corner
		}
	}
	
	// Bottom-right corner exclusion
	if p.X >= node.X+node.Width-1 && p.Y >= node.Y+node.Height-1 {
		dx := p.X - (node.X + node.Width - 1)
		dy := p.Y - (node.Y + node.Height - 1)
		if dx + dy <= cornerRadius {
			return true // Block diagonal approach to corner
		}
	}
	
	return false // Allow other points in approach zone
}
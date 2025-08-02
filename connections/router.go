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
	
	// Get connection points for the nodes
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
	// Prefer horizontal connections over vertical when possible
	if abs(dx) > abs(dy) {
		// Horizontal connection
		if dx > 0 {
			// Connect from right side (place point outside the node)
			return core.Point{
				X: fromNode.X + fromNode.Width,
				Y: fromNode.Y + fromNode.Height/2,
			}
		} else {
			// Connect from left side (place point outside the node)
			return core.Point{
				X: fromNode.X - 1,
				Y: fromNode.Y + fromNode.Height/2,
			}
		}
	} else {
		// Vertical connection
		if dy > 0 {
			// Connect from bottom (place point outside the node)
			return core.Point{
				X: fromNode.X + fromNode.Width/2,
				Y: fromNode.Y + fromNode.Height,
			}
		} else {
			// Connect from top (place point outside the node)
			return core.Point{
				X: fromNode.X + fromNode.Width/2,
				Y: fromNode.Y - 1,
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
	return createObstaclesFunctionWithPadding(nodes, sourceID, targetID, 2) // Default padding of 2
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
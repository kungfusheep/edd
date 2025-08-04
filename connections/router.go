package connections

import (
	"edd/core"
	"edd/obstacles"
	"fmt"
)

// Router handles the routing of connections between nodes in a diagram.
type Router struct {
	pathFinder       core.PathFinder
	obstacleManager  obstacles.ObstacleManager
	portManager      obstacles.PortManager
	twoPhaseRouter   *TwoPhaseRouter
}

// NewRouter creates a new connection router.
func NewRouter(pathFinder core.PathFinder) *Router {
	// Create default obstacle manager
	obstacleConfig := obstacles.DefaultVirtualObstacleConfig()
	obstacleManager := obstacles.NewObstacleManager(obstacleConfig)
	
	router := &Router{
		pathFinder:      pathFinder,
		obstacleManager: obstacleManager,
	}
	
	// Create two-phase router
	router.twoPhaseRouter = NewTwoPhaseRouter(pathFinder, obstacleManager)
	
	return router
}

// SetObstacleManager allows setting a custom obstacle manager
func (r *Router) SetObstacleManager(manager obstacles.ObstacleManager) {
	r.obstacleManager = manager
	r.twoPhaseRouter.obstacleManager = manager
}

// GetObstacleManager returns the current obstacle manager
func (r *Router) GetObstacleManager() obstacles.ObstacleManager {
	return r.obstacleManager
}

// SetPortManager sets the port manager for port-based routing
func (r *Router) SetPortManager(pm obstacles.PortManager) {
	r.portManager = pm
	r.obstacleManager.SetPortManager(pm)
	r.twoPhaseRouter.SetPortManager(pm)
}

// RouteConnection finds the best path for a connection between two nodes.
// It returns a Path that avoids obstacles and creates clean routes.
func (r *Router) RouteConnection(conn core.Connection, nodes []core.Node) (core.Path, error) {
	// Always use two-phase routing with port manager
	return r.twoPhaseRouter.RouteConnectionWithPorts(conn, nodes)
}

// RouteConnections routes multiple connections, handling grouping and optimization.
func (r *Router) RouteConnections(connections []core.Connection, nodes []core.Node) (map[int]core.Path, error) {
	// Always use dynamic routing with port manager to prevent overlaps
	return r.routeConnectionsWithDynamicObstacles(connections, nodes)
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


// routeConnectionsWithDynamicObstacles routes connections with dynamic obstacle updates.
// Each connection respects previously routed connections' port usage.
func (r *Router) routeConnectionsWithDynamicObstacles(connections []core.Connection, nodes []core.Node) (map[int]core.Path, error) {
	paths := make(map[int]core.Path)
	
	// Create a deterministic ordering for connections
	orderedConns := make([]struct {
		conn  core.Connection
		index int
	}, len(connections))
	
	for i, conn := range connections {
		orderedConns[i] = struct {
			conn  core.Connection
			index int
		}{conn, i}
	}
	
	// Sort connections by: 1) source ID, 2) target ID, 3) connection ID
	// This ensures consistent routing order across runs
	for i := 0; i < len(orderedConns)-1; i++ {
		for j := i + 1; j < len(orderedConns); j++ {
			if shouldSwap(orderedConns[i].conn, orderedConns[j].conn) {
				orderedConns[i], orderedConns[j] = orderedConns[j], orderedConns[i]
			}
		}
	}
	
	// Route each connection in order
	for _, item := range orderedConns {
		// Route the connection
		path, err := r.RouteConnection(item.conn, nodes)
		if err != nil {
			// On failure, release any ports we've reserved so far
			for _, p := range paths {
				if p.Metadata != nil {
					if sourcePort, ok := p.Metadata["sourcePort"].(obstacles.Port); ok {
						r.portManager.ReleasePort(sourcePort)
					}
					if targetPort, ok := p.Metadata["targetPort"].(obstacles.Port); ok {
						r.portManager.ReleasePort(targetPort)
					}
				}
			}
			return nil, fmt.Errorf("failed to route connection %d (%d->%d): %w",
				item.conn.ID, item.conn.From, item.conn.To, err)
		}
		
		// Store the path
		paths[item.index] = path
		
		// The port manager automatically tracks occupied ports,
		// so subsequent connections will avoid them
	}
	
	return paths, nil
}

// shouldSwap determines if two connections should be swapped in the ordering
func shouldSwap(a, b core.Connection) bool {
	// Sort by source ID first
	if a.From != b.From {
		return a.From > b.From
	}
	// Then by target ID
	if a.To != b.To {
		return a.To > b.To
	}
	// Finally by connection ID
	return a.ID > b.ID
}

// createObstaclesFunction is a temporary stub for backward compatibility.
// This should be removed once all callers are updated to use ObstacleManager.
func createObstaclesFunction(nodes []core.Node, sourceID, targetID int) func(core.Point) bool {
	// Create a temporary obstacle manager for compatibility
	config := obstacles.DefaultVirtualObstacleConfig()
	manager := obstacles.NewObstacleManager(config)
	conn := core.Connection{From: sourceID, To: targetID}
	return manager.GetObstacleFuncForConnection(nodes, conn)
}
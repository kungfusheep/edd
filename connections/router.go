package connections

import (
	"edd/core"
	"edd/obstacles"
	"fmt"
	"math"
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
	
	// Calculate distances for each connection
	distances := make(map[int]float64)
	for _, item := range orderedConns {
		var sourceNode, targetNode *core.Node
		for i := range nodes {
			if nodes[i].ID == item.conn.From {
				sourceNode = &nodes[i]
			}
			if nodes[i].ID == item.conn.To {
				targetNode = &nodes[i]
			}
		}
		if sourceNode != nil && targetNode != nil {
			// Calculate center-to-center distance
			dx := float64((sourceNode.X + sourceNode.Width/2) - (targetNode.X + targetNode.Width/2))
			dy := float64((sourceNode.Y + sourceNode.Height/2) - (targetNode.Y + targetNode.Height/2))
			distance := dx*dx + dy*dy // squared distance is fine for sorting
			distances[item.index] = distance
			// Debug: print distances and positions
			// if item.conn.From == 2 && (item.conn.To == 3 || item.conn.To == 4) {
			// 	fmt.Printf("Connection %d (%d->%d): distance = %.2f\n", item.conn.ID, item.conn.From, item.conn.To, math.Sqrt(distance))
			// 	fmt.Printf("  Source (node %d): pos=(%d,%d) size=(%dx%d) center=(%d,%d)\n", 
			// 		sourceNode.ID, sourceNode.X, sourceNode.Y, sourceNode.Width, sourceNode.Height,
			// 		sourceNode.X + sourceNode.Width/2, sourceNode.Y + sourceNode.Height/2)
			// 	fmt.Printf("  Target (node %d): pos=(%d,%d) size=(%dx%d) center=(%d,%d)\n",
			// 		targetNode.ID, targetNode.X, targetNode.Y, targetNode.Width, targetNode.Height,
			// 		targetNode.X + targetNode.Width/2, targetNode.Y + targetNode.Height/2)
			// }
		}
	}
	
	// Sort connections by distance (nearest first) for better port allocation
	// Group connections with similar distances (within 1 unit) together
	// Within each group, sort by source ID, then target ID for determinism
	const distanceTolerance = 1.0 // Consider distances within 1 unit as equal
	
	for i := 0; i < len(orderedConns)-1; i++ {
		for j := i + 1; j < len(orderedConns); j++ {
			dist1 := math.Sqrt(distances[orderedConns[i].index])
			dist2 := math.Sqrt(distances[orderedConns[j].index])
			
			// Check if distances are significantly different
			if math.Abs(dist1-dist2) > distanceTolerance {
				// Sort by distance
				if dist2 < dist1 {
					orderedConns[i], orderedConns[j] = orderedConns[j], orderedConns[i]
				}
			} else {
				// Distances are similar, use deterministic ordering
				// First by source node ID, then by target node ID
				conn1 := orderedConns[i].conn
				conn2 := orderedConns[j].conn
				
				if conn1.From != conn2.From {
					if conn2.From < conn1.From {
						orderedConns[i], orderedConns[j] = orderedConns[j], orderedConns[i]
					}
				} else if conn1.To != conn2.To {
					if conn2.To < conn1.To {
						orderedConns[i], orderedConns[j] = orderedConns[j], orderedConns[i]
					}
				} else if conn2.ID < conn1.ID {
					// Same source and target, sort by connection ID
					orderedConns[i], orderedConns[j] = orderedConns[j], orderedConns[i]
				}
			}
		}
	}
	
	// Route each connection in order
	// fmt.Println("\nRouting connections in order:")
	for _, item := range orderedConns {
		// fmt.Printf("%d. Connection %d (%d->%d) - distance: %.2f\n", i+1, item.conn.ID, item.conn.From, item.conn.To, math.Sqrt(distances[item.index]))
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


// createObstaclesFunction is a temporary stub for backward compatibility.
// This should be removed once all callers are updated to use ObstacleManager.
func createObstaclesFunction(nodes []core.Node, sourceID, targetID int) func(core.Point) bool {
	// Create a temporary obstacle manager for compatibility
	config := obstacles.DefaultVirtualObstacleConfig()
	manager := obstacles.NewObstacleManager(config)
	conn := core.Connection{From: sourceID, To: targetID}
	return manager.GetObstacleFuncForConnection(nodes, conn)
}
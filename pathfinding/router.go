package pathfinding

import (
	"edd/diagram"
	"fmt"
	"math"
)

// RouterType defines the type of routing algorithm to use
type RouterType string

const (
	RouterTypeSimple   RouterType = "simple"    // Center-to-center with truncation
	RouterTypeTwoPhase RouterType = "two-phase" // Two-phase with port selection
	RouterTypeArea     RouterType = "area"      // Area-based edge-to-edge
)

// Router handles the routing of connections between nodes in a diagram.
type Router struct {
	pathFinder       diagram.PathFinder
	obstacleManager  ObstacleManager
	portManager      PortManager
	twoPhaseRouter   *TwoPhaseRouter
	simpleRouter     *SimpleRouter
	areaRouter       *AreaRouter
	routerType       RouterType
}

// NewRouter creates a new connection router.
func NewRouter(pathFinder diagram.PathFinder) *Router {
	// Create default obstacle manager
	obstacleConfig := DefaultVirtualObstacleConfig()
	obstacleManager := NewObstacleManager(obstacleConfig)
	
	router := &Router{
		pathFinder:      pathFinder,
		obstacleManager: obstacleManager,
		routerType:      RouterTypeArea, // Default to area-based routing
	}
	
	// Create two-phase router
	router.twoPhaseRouter = NewTwoPhaseRouter(pathFinder, obstacleManager)
	
	// Create simple router
	router.simpleRouter = NewSimpleRouter(pathFinder, obstacleManager)
	
	// Create area router if pathfinder supports area-based routing
	if areaPathFinder, ok := pathFinder.(diagram.AreaPathFinder); ok {
		router.areaRouter = NewAreaRouter(areaPathFinder, obstacleManager)
	}
	
	return router
}

// SetObstacleManager allows setting a custom obstacle manager
func (r *Router) SetObstacleManager(manager ObstacleManager) {
	r.obstacleManager = manager
	r.twoPhaseRouter.obstacleManager = manager
}

// GetObstacleManager returns the current obstacle manager
func (r *Router) GetObstacleManager() ObstacleManager {
	return r.obstacleManager
}

// SetPortManager sets the port manager for port-based routing
func (r *Router) SetPortManager(pm PortManager) {
	r.portManager = pm
	r.obstacleManager.SetPortManager(pm)
	r.twoPhaseRouter.SetPortManager(pm)
}

// SetRouterType sets the type of router to use
func (r *Router) SetRouterType(routerType RouterType) {
	r.routerType = routerType
}

// RouteConnection finds the best path for a connection between two nodes.
// It returns a Path that avoids obstacles and creates clean routes.
func (r *Router) RouteConnection(conn diagram.Connection, nodes []diagram.Node) (diagram.Path, error) {
	switch r.routerType {
	case RouterTypeArea:
		if r.areaRouter != nil {
			return r.areaRouter.RouteConnection(conn, nodes)
		}
		// Fallback to simple if area router not available
		return r.simpleRouter.RouteConnection(conn, nodes)
	case RouterTypeTwoPhase:
		return r.twoPhaseRouter.RouteConnectionWithPorts(conn, nodes)
	case RouterTypeSimple:
		fallthrough
	default:
		return r.simpleRouter.RouteConnection(conn, nodes)
	}
}

// RouteConnections routes multiple connections, handling grouping and optimization.
func (r *Router) RouteConnections(connections []diagram.Connection, nodes []diagram.Node) (map[int]diagram.Path, error) {
	// Always use dynamic routing with port manager to prevent overlaps
	return r.routeConnectionsWithDynamicObstacles(connections, nodes)
}





// routeConnectionsWithDynamicObstacles routes connections with dynamic obstacle updates.
// Each connection respects previously routed connections' port usage.
func (r *Router) routeConnectionsWithDynamicObstacles(connections []diagram.Connection, nodes []diagram.Node) (map[int]diagram.Path, error) {
	paths := make(map[int]diagram.Path)
	
	// Create a deterministic ordering for connections
	orderedConns := make([]struct {
		conn  diagram.Connection
		index int
	}, len(connections))
	
	for i, conn := range connections {
		orderedConns[i] = struct {
			conn  diagram.Connection
			index int
		}{conn, i}
	}
	
	// Calculate distances for each connection
	distances := make(map[int]float64)
	for _, item := range orderedConns {
		var sourceNode, targetNode *diagram.Node
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
	
	// Group connections by target for coordinated port assignment
	targetGroups := make(map[int][]int) // target ID -> indices in orderedConns
	for i, item := range orderedConns {
		targetGroups[item.conn.To] = append(targetGroups[item.conn.To], i)
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
				conn1 := orderedConns[i].conn
				conn2 := orderedConns[j].conn
				
				// Special handling: if connections go to the same target, sort by source Y position
				if conn1.To == conn2.To {
					// Find source nodes
					var source1Y, source2Y int
					for _, node := range nodes {
						if node.ID == conn1.From {
							source1Y = node.Y + node.Height/2
						}
						if node.ID == conn2.From {
							source2Y = node.Y + node.Height/2
						}
					}
					// Sort by Y position (higher Y = lower on screen)
					if source2Y < source1Y {
						orderedConns[i], orderedConns[j] = orderedConns[j], orderedConns[i]
					} else if source1Y == source2Y && conn2.From < conn1.From {
						// Same Y, sort by source ID
						orderedConns[i], orderedConns[j] = orderedConns[j], orderedConns[i]
					}
				} else {
					// Different targets, use original logic
					if conn1.From != conn2.From {
						if conn2.From < conn1.From {
							orderedConns[i], orderedConns[j] = orderedConns[j], orderedConns[i]
						}
					} else if conn2.To < conn1.To {
						orderedConns[i], orderedConns[j] = orderedConns[j], orderedConns[i]
					}
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
					if sourcePort, ok := p.Metadata["sourcePort"].(Port); ok {
						r.portManager.ReleasePort(sourcePort)
					}
					if targetPort, ok := p.Metadata["targetPort"].(Port); ok {
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

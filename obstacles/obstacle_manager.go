package obstacles

import (
	"edd/core"
	"fmt"
)

// ObstacleManager provides a unified interface for all obstacle types.
// Consumers don't need to know about virtual vs physical obstacles.
type ObstacleManager interface {
	// GetObstacleFunc returns a function that checks if a point is an obstacle.
	// This includes physical obstacles (node bodies), virtual obstacles (approach zones),
	// and port corridors (for reserved ports).
	GetObstacleFunc(nodes []core.Node, activeConnID int) func(core.Point) bool
	
	// GetObstacleFuncForConnection returns obstacles for a specific connection
	GetObstacleFuncForConnection(nodes []core.Node, conn core.Connection) func(core.Point) bool
	
	// SetPortManager sets the port manager for port-based obstacles
	SetPortManager(pm PortManager)
	
	// ReservePort reserves a port and updates obstacles accordingly
	ReservePort(nodeID int, edge EdgeSide, connID int) (Port, error)
	
	// ReleasePort releases a previously reserved port
	ReleasePort(port Port)
	
	// SetConfig updates the obstacle configuration
	SetConfig(config VirtualObstacleConfig)
}

// unifiedObstacleManager implements ObstacleManager
type unifiedObstacleManager struct {
	config       VirtualObstacleConfig
	portManager  PortManager
	baseChecker  VirtualObstacleChecker
}

// NewObstacleManager creates a new unified obstacle manager
func NewObstacleManager(config VirtualObstacleConfig) ObstacleManager {
	return &unifiedObstacleManager{
		config:      config,
		baseChecker: NewVirtualObstacleChecker(config),
	}
}

// GetObstacleFunc returns a unified obstacle checking function
func (m *unifiedObstacleManager) GetObstacleFunc(nodes []core.Node, activeConnID int) func(core.Point) bool {
	// For generic obstacle checking, we don't have source/target info
	// so we pass -1 to include all obstacles
	baseObstacles := m.baseChecker.CreateObstacleFunc(nodes, -1, -1)
	
	// If no port manager, just return base obstacles
	if m.portManager == nil {
		return baseObstacles
	}
	
	// Create combined obstacle function
	return func(p core.Point) bool {
		// Check base obstacles first
		if baseObstacles(p) {
			return true
		}
		
		// Check port-based obstacles
		return m.isPortObstacle(p, nodes, activeConnID)
	}
}

// isPortObstacle checks if a point is blocked by a port corridor
func (m *unifiedObstacleManager) isPortObstacle(p core.Point, nodes []core.Node, activeConnID int) bool {
	if m.portManager == nil {
		return false
	}
	
	// Check each node's occupied ports
	for _, node := range nodes {
		occupiedPorts := m.portManager.GetOccupiedPorts(node.ID)
		for _, port := range occupiedPorts {
			// Don't block the active connection's own ports
			if port.ConnectionID == activeConnID {
				continue
			}
			
			// Check if point is in port corridor
			if m.isInPortCorridor(p, port, node) {
				// Debug
				// fmt.Printf("Point (%d,%d) blocked by port corridor for conn %d at (%d,%d)\n",
				//     p.X, p.Y, port.ConnectionID, port.Point.X, port.Point.Y)
				return true
			}
		}
	}
	
	return false
}

// isInPortCorridor checks if a point is in a port's approach corridor
func (m *unifiedObstacleManager) isInPortCorridor(p core.Point, port Port, node core.Node) bool {
	// Port corridors should only extend from the port outward,
	// not through the node to the other side
	
	// For stacked ports, we need to be more careful about corridors
	// to avoid blocking other stacked ports
	corridorLength := m.config.ApproachZoneSize
	if port.StackLevel > 0 {
		// Reduce corridor length for stacked ports to avoid conflicts
		corridorLength = 1
	}
	
	switch port.Edge {
	case North:
		// Corridor extends upward from the north edge
		if p.X == port.Point.X &&
		   p.Y >= port.Point.Y - corridorLength && p.Y < port.Point.Y {
			return true
		}
		
	case South:
		// Corridor extends downward from the south edge
		if p.X == port.Point.X &&
		   p.Y > port.Point.Y && p.Y <= port.Point.Y + corridorLength {
			return true
		}
		
	case East:
		// Corridor extends rightward from the east edge
		if p.Y == port.Point.Y &&
		   p.X > port.Point.X && p.X <= port.Point.X + corridorLength {
			return true
		}
		
	case West:
		// Corridor extends leftward from the west edge
		if p.Y == port.Point.Y &&
		   p.X >= port.Point.X - corridorLength && p.X < port.Point.X {
			return true
		}
	}
	
	return false
}

// SetPortManager sets the port manager
func (m *unifiedObstacleManager) SetPortManager(pm PortManager) {
	m.portManager = pm
}

// ReservePort reserves a port and updates obstacles
func (m *unifiedObstacleManager) ReservePort(nodeID int, edge EdgeSide, connID int) (Port, error) {
	if m.portManager == nil {
		return Port{}, fmt.Errorf("no port manager configured")
	}
	return m.portManager.ReservePort(nodeID, edge, connID)
}

// ReleasePort releases a port
func (m *unifiedObstacleManager) ReleasePort(port Port) {
	if m.portManager != nil {
		m.portManager.ReleasePort(port)
	}
}

// SetConfig updates the configuration
func (m *unifiedObstacleManager) SetConfig(config VirtualObstacleConfig) {
	m.config = config
	m.baseChecker = NewVirtualObstacleChecker(config)
}

// GetObstacleFuncForConnection returns obstacles for a specific connection
func (m *unifiedObstacleManager) GetObstacleFuncForConnection(nodes []core.Node, conn core.Connection) func(core.Point) bool {
	// Get base obstacles with source/target awareness
	baseObstacles := m.baseChecker.CreateObstacleFunc(nodes, conn.From, conn.To)
	
	// If no port manager, just return base obstacles
	if m.portManager == nil {
		return baseObstacles
	}
	
	// Create combined obstacle function
	return func(p core.Point) bool {
		// Check base obstacles first
		if baseObstacles(p) {
			return true
		}
		
		// Check port-based obstacles
		return m.isPortObstacle(p, nodes, conn.ID)
	}
}
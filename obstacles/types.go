// Package obstacles provides centralized virtual obstacle management with port awareness
package obstacles

import (
	"edd/core"
)

// EdgeSide represents which side of a node an edge is on
type EdgeSide int

const (
	North EdgeSide = iota
	South
	East
	West
)

// Port represents a connection point on a node edge
type Port struct {
	NodeID   int      // Which node this port belongs to
	Edge     EdgeSide // Which edge of the node
	Position int      // Position along the edge (0 to edge length)
	Width    int      // How many units this port occupies
	Point    core.Point // The actual connection point (1 unit away from edge)
	ConnectionID int  // ID of the connection using this port (-1 if free)
	StackLevel int   // Stacking level when multiple connections share a port (0 = first)
}

// PortManager manages available ports on nodes
type PortManager interface {
	// GetAvailablePorts returns all free ports on the specified edge of a node
	GetAvailablePorts(nodeID int, edge EdgeSide) []Port
	
	// GetOccupiedPorts returns all occupied ports on a node
	GetOccupiedPorts(nodeID int) []Port
	
	// IsPortOccupied checks if a specific port is occupied
	IsPortOccupied(port Port) bool
	
	// ReservePort reserves a port for a connection, returns the reserved port
	ReservePort(nodeID int, edge EdgeSide, connectionID int) (Port, error)
	
	// ReleasePort releases a previously reserved port
	ReleasePort(port Port)
	
	// GetPortForConnection returns the port reserved for a specific connection
	GetPortForConnection(nodeID int, connectionID int) (Port, bool)
}

// VirtualObstacleConfig defines parameters for virtual obstacle generation
type VirtualObstacleConfig struct {
	ApproachZoneSize  int     // Size of the approach zone around nodes (default: 2)
	CorridorWidth     int     // Width of allowed approach corridors (default: 1)
	CornerRadius      int     // Radius for corner clearance (default: 2)
	SourceTargetScale float64 // Scale factor for source/target obstacles (default: 0.75)
	EnableForSource   bool    // Apply obstacles to source nodes (default: true with scale)
	EnableForTarget   bool    // Apply obstacles to target nodes (default: true with scale)
}

// DefaultVirtualObstacleConfig returns sensible defaults
func DefaultVirtualObstacleConfig() VirtualObstacleConfig {
	return VirtualObstacleConfig{
		ApproachZoneSize:  2,
		CorridorWidth:     1,
		CornerRadius:      2,
		SourceTargetScale: 0.75,
		EnableForSource:   true,
		EnableForTarget:   true,
	}
}

// ObstacleZone represents a rectangular obstacle area
type ObstacleZone struct {
	MinX, MinY, MaxX, MaxY int
	Type                   string // "physical", "virtual", "port"
	NodeID                 int
}

// VirtualObstacleChecker provides consistent virtual obstacle checking
type VirtualObstacleChecker interface {
	// IsObstacle checks if a point is in any obstacle zone
	IsObstacle(p core.Point, sourceID, targetID int) bool
	
	// GetObstacleZones returns all obstacle zones for visualization
	GetObstacleZones(nodes []core.Node, sourceID, targetID int) []ObstacleZone
	
	// CreateObstacleFunc returns a function that checks both physical and virtual obstacles
	CreateObstacleFunc(nodes []core.Node, sourceID, targetID int) func(core.Point) bool
}

// DynamicVirtualObstacleChecker provides port-aware dynamic obstacles
type DynamicVirtualObstacleChecker interface {
	VirtualObstacleChecker
	
	// SetPortManager sets the port manager for dynamic obstacle generation
	SetPortManager(pm PortManager)
	
	// UpdateActiveConnection updates which connection is currently being routed
	UpdateActiveConnection(connectionID int)
	
	// GetDynamicObstacles returns obstacles based on current port usage
	GetDynamicObstacles(nodes []core.Node, connectionID int) []ObstacleZone
}
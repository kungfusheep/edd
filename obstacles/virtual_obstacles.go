package obstacles

import (
	"edd/core"
)

// virtualObstacleChecker implements VirtualObstacleChecker
type virtualObstacleChecker struct {
	config VirtualObstacleConfig
}

// NewVirtualObstacleChecker creates a new virtual obstacle checker
func NewVirtualObstacleChecker(config VirtualObstacleConfig) VirtualObstacleChecker {
	return &virtualObstacleChecker{
		config: config,
	}
}

// IsObstacle checks if a point is in any obstacle zone
func (v *virtualObstacleChecker) IsObstacle(p core.Point, sourceID, targetID int) bool {
	// This is a simplified check - the full implementation would check zones
	return false
}

// GetObstacleZones returns all obstacle zones for visualization
func (v *virtualObstacleChecker) GetObstacleZones(nodes []core.Node, sourceID, targetID int) []ObstacleZone {
	zones := []ObstacleZone{}
	
	for _, node := range nodes {
		// Add physical obstacle (node body with padding)
		// For source and target nodes, only block the interior
		if node.ID == sourceID || node.ID == targetID {
			zones = append(zones, ObstacleZone{
				MinX:   node.X,
				MinY:   node.Y,
				MaxX:   node.X + node.Width - 1,
				MaxY:   node.Y + node.Height - 1,
				Type:   "physical",
				NodeID: node.ID,
			})
		} else {
			// For other nodes, add 1-unit padding
			zones = append(zones, ObstacleZone{
				MinX:   node.X - 1,
				MinY:   node.Y - 1,
				MaxX:   node.X + node.Width,
				MaxY:   node.Y + node.Height,
				Type:   "physical",
				NodeID: node.ID,
			})
		}
		
		// Add virtual obstacle zones around the node
		if v.shouldAddVirtualObstacles(node.ID, sourceID, targetID) {
			zones = append(zones, v.createVirtualZones(node, sourceID, targetID)...)
		}
	}
	
	return zones
}

// CreateObstacleFunc returns a function that checks both physical and virtual obstacles
func (v *virtualObstacleChecker) CreateObstacleFunc(nodes []core.Node, sourceID, targetID int) func(core.Point) bool {
	// Pre-calculate all obstacle zones
	zones := v.GetObstacleZones(nodes, sourceID, targetID)
	
	return func(p core.Point) bool {
		// Check if point is in any obstacle zone
		for _, zone := range zones {
			if p.X >= zone.MinX && p.X <= zone.MaxX &&
			   p.Y >= zone.MinY && p.Y <= zone.MaxY {
				return true
			}
		}
		return false
	}
}

// shouldAddVirtualObstacles determines if virtual obstacles should be added for a node
func (v *virtualObstacleChecker) shouldAddVirtualObstacles(nodeID, sourceID, targetID int) bool {
	// Apply to all nodes by default, with scaling for source/target
	if nodeID == sourceID && !v.config.EnableForSource {
		return false
	}
	if nodeID == targetID && !v.config.EnableForTarget {
		return false
	}
	return true
}

// createVirtualZones creates virtual obstacle zones around a node
func (v *virtualObstacleChecker) createVirtualZones(node core.Node, sourceID, targetID int) []ObstacleZone {
	// For now, don't create any additional virtual zones
	// The 1-unit padding in GetObstacleZones is sufficient
	return []ObstacleZone{}
}

// dynamicVirtualObstacleChecker implements DynamicVirtualObstacleChecker
type dynamicVirtualObstacleChecker struct {
	*virtualObstacleChecker
	portManager      PortManager
	activeConnection int
}

// NewDynamicVirtualObstacleChecker creates a new dynamic virtual obstacle checker
func NewDynamicVirtualObstacleChecker(config VirtualObstacleConfig) DynamicVirtualObstacleChecker {
	return &dynamicVirtualObstacleChecker{
		virtualObstacleChecker: &virtualObstacleChecker{
			config: config,
		},
		activeConnection: -1,
	}
}

// SetPortManager sets the port manager for dynamic obstacle generation
func (d *dynamicVirtualObstacleChecker) SetPortManager(pm PortManager) {
	d.portManager = pm
}

// UpdateActiveConnection updates which connection is currently being routed
func (d *dynamicVirtualObstacleChecker) UpdateActiveConnection(connectionID int) {
	d.activeConnection = connectionID
}

// GetDynamicObstacles returns obstacles based on current port usage
func (d *dynamicVirtualObstacleChecker) GetDynamicObstacles(nodes []core.Node, connectionID int) []ObstacleZone {
	// Start with base obstacles
	zones := d.GetObstacleZones(nodes, -1, -1)
	
	if d.portManager == nil {
		return zones
	}
	
	// Add port-based obstacles
	for _, node := range nodes {
		occupiedPorts := d.portManager.GetOccupiedPorts(node.ID)
		for _, port := range occupiedPorts {
			// Don't create obstacles for the current connection's ports
			if port.ConnectionID == connectionID {
				continue
			}
			
			// Create obstacle zone around occupied port
			zones = append(zones, d.createPortObstacle(node, port))
		}
	}
	
	return zones
}

// createPortObstacle creates an obstacle zone around an occupied port
func (d *dynamicVirtualObstacleChecker) createPortObstacle(node core.Node, port Port) ObstacleZone {
	// Create a corridor-shaped obstacle that blocks crossing paths
	corridorLength := d.config.ApproachZoneSize * 2
	
	switch port.Edge {
	case North, South:
		// Vertical corridor
		y := port.Point.Y
		if port.Edge == North {
			y -= corridorLength
		}
		return ObstacleZone{
			MinX:   port.Point.X - 1,
			MinY:   y,
			MaxX:   port.Point.X + 1,
			MaxY:   y + corridorLength,
			Type:   "port",
			NodeID: node.ID,
		}
	case East, West:
		// Horizontal corridor
		x := port.Point.X
		if port.Edge == West {
			x -= corridorLength
		}
		return ObstacleZone{
			MinX:   x,
			MinY:   port.Point.Y - 1,
			MaxX:   x + corridorLength,
			MaxY:   port.Point.Y + 1,
			Type:   "port",
			NodeID: node.ID,
		}
	}
	
	return ObstacleZone{}
}
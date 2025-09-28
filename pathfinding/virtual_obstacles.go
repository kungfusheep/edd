package pathfinding

import (
	"edd/diagram"
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
func (v *virtualObstacleChecker) IsObstacle(p diagram.Point, sourceID, targetID int) bool {
	// This is a simplified check - the full implementation would check zones
	return false
}

// GetObstacleZones returns all obstacle zones for visualization
func (v *virtualObstacleChecker) GetObstacleZones(nodes []diagram.Node, sourceID, targetID int) []ObstacleZone {
	zones := []ObstacleZone{}
	
	for _, node := range nodes {
		// Add physical obstacle (node body only, no padding)
		// ALL nodes are treated the same way for global consistency
		zones = append(zones, ObstacleZone{
			MinX:   node.X,
			MinY:   node.Y,
			MaxX:   node.X + node.Width - 1,
			MaxY:   node.Y + node.Height - 1,
			Type:   "physical",
			NodeID: node.ID,
		})
		
		// Add virtual obstacle zones around ALL nodes to create port corridors
		// This ensures consistent routing for all connections
		// Pass -1, -1 to indicate these are global obstacles
		zones = append(zones, v.createVirtualZones(node, -1, -1)...)
	}
	
	return zones
}

// CreateObstacleFunc returns a function that checks both physical and virtual obstacles
func (v *virtualObstacleChecker) CreateObstacleFunc(nodes []diagram.Node, sourceID, targetID int) func(diagram.Point) bool {
	// Pre-calculate all obstacle zones
	zones := v.GetObstacleZones(nodes, sourceID, targetID)
	
	return func(p diagram.Point) bool {
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


// createVirtualZones creates virtual obstacle zones around a node
func (v *virtualObstacleChecker) createVirtualZones(node diagram.Node, sourceID, targetID int) []ObstacleZone {
	zones := []ObstacleZone{}
	
	// Create virtual obstacles that block the perimeter except at port locations
	// This applies to ALL nodes to create consistent routing corridors
	
	// Calculate port positions (center of each edge)
	centerX := node.X + node.Width/2
	centerY := node.Y + node.Height/2
	
	// Debug output
	// fmt.Printf("Creating virtual zones for node %d: pos(%d,%d) size(%dx%d) center(%d,%d)\n",
	//     node.ID, node.X, node.Y, node.Width, node.Height, centerX, centerY)
	
	// Top edge - block everything except center port
	// Left part of top edge (from left corner to just before center)
	if centerX > node.X {
		zones = append(zones, ObstacleZone{
			MinX:   node.X - 1,
			MinY:   node.Y - 1,
			MaxX:   centerX - 1,
			MaxY:   node.Y - 1,
			Type:   "virtual-port",
			NodeID: node.ID,
		})
	}
	// Right part of top edge (from just after center to right corner)
	if centerX < node.X + node.Width - 1 {
		zones = append(zones, ObstacleZone{
			MinX:   centerX + 1,
			MinY:   node.Y - 1,
			MaxX:   node.X + node.Width,
			MaxY:   node.Y - 1,
			Type:   "virtual-port",
			NodeID: node.ID,
		})
	}
	
	// Bottom edge - block everything except center port
	// Left part of bottom edge
	if centerX > node.X {
		zones = append(zones, ObstacleZone{
			MinX:   node.X - 1,
			MinY:   node.Y + node.Height,
			MaxX:   centerX - 1,
			MaxY:   node.Y + node.Height,
			Type:   "virtual-port",
			NodeID: node.ID,
		})
	}
	// Right part of bottom edge
	if centerX < node.X + node.Width - 1 {
		zones = append(zones, ObstacleZone{
			MinX:   centerX + 1,
			MinY:   node.Y + node.Height,
			MaxX:   node.X + node.Width,
			MaxY:   node.Y + node.Height,
			Type:   "virtual-port",
			NodeID: node.ID,
		})
	}
	
	// Left edge - block everything except center port
	// Top part of left edge
	if centerY > node.Y {
		zones = append(zones, ObstacleZone{
			MinX:   node.X - 1,
			MinY:   node.Y - 1,
			MaxX:   node.X - 1,
			MaxY:   centerY - 1,
			Type:   "virtual-port",
			NodeID: node.ID,
		})
	}
	// Bottom part of left edge
	if centerY < node.Y + node.Height - 1 {
		zones = append(zones, ObstacleZone{
			MinX:   node.X - 1,
			MinY:   centerY + 1,
			MaxX:   node.X - 1,
			MaxY:   node.Y + node.Height,
			Type:   "virtual-port",
			NodeID: node.ID,
		})
	}
	
	// Right edge - block everything except center port
	// Top part of right edge
	if centerY > node.Y {
		zones = append(zones, ObstacleZone{
			MinX:   node.X + node.Width,
			MinY:   node.Y - 1,
			MaxX:   node.X + node.Width,
			MaxY:   centerY - 1,
			Type:   "virtual-port",
			NodeID: node.ID,
		})
	}
	// Bottom part of right edge
	if centerY < node.Y + node.Height - 1 {
		zones = append(zones, ObstacleZone{
			MinX:   node.X + node.Width,
			MinY:   centerY + 1,
			MaxX:   node.X + node.Width,
			MaxY:   node.Y + node.Height,
			Type:   "virtual-port",
			NodeID: node.ID,
		})
	}
	
	return zones
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
func (d *dynamicVirtualObstacleChecker) GetDynamicObstacles(nodes []diagram.Node, connectionID int) []ObstacleZone {
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
func (d *dynamicVirtualObstacleChecker) createPortObstacle(node diagram.Node, port Port) ObstacleZone {
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
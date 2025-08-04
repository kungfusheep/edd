package obstacles

import (
	"edd/core"
	"fmt"
	"sync"
)

// portManagerImpl implements the PortManager interface
type portManagerImpl struct {
	nodes      map[int]*core.Node
	ports      map[string]*Port // Key: "nodeID-edge-position"
	mu         sync.RWMutex
	portWidth  int // Default width for each port
}

// NewPortManager creates a new port manager
func NewPortManager(nodes []core.Node, portWidth int) PortManager {
	pm := &portManagerImpl{
		nodes:     make(map[int]*core.Node),
		ports:     make(map[string]*Port),
		portWidth: portWidth,
	}
	
	// Store nodes for reference
	for i := range nodes {
		pm.nodes[nodes[i].ID] = &nodes[i]
	}
	
	return pm
}

// GetAvailablePorts returns all free ports on the specified edge
func (pm *portManagerImpl) GetAvailablePorts(nodeID int, edge EdgeSide) []Port {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	node, exists := pm.nodes[nodeID]
	if !exists {
		return nil
	}
	
	edgeLength := pm.getEdgeLength(node, edge)
	availablePorts := []Port{}
	
	// Calculate available positions along the edge
	// Leave space at corners for clean junctions
	margin := 1 // Leave 1 unit at each corner (reduced from 2)
	step := pm.portWidth
	
	for pos := margin; pos < edgeLength-margin; pos += step {
		port := Port{
			NodeID:       nodeID,
			Edge:         edge,
			Position:     pos,
			Width:        pm.portWidth,
			Point:        pm.calculatePortPoint(node, edge, pos),
			ConnectionID: -1,
		}
		
		key := pm.portKey(nodeID, edge, pos)
		if existingPort, occupied := pm.ports[key]; !occupied || existingPort.ConnectionID == -1 {
			availablePorts = append(availablePorts, port)
		}
	}
	
	return availablePorts
}

// GetOccupiedPorts returns all occupied ports on a node
func (pm *portManagerImpl) GetOccupiedPorts(nodeID int) []Port {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	occupiedPorts := []Port{}
	for _, port := range pm.ports {
		if port.NodeID == nodeID && port.ConnectionID != -1 {
			occupiedPorts = append(occupiedPorts, *port)
		}
	}
	
	return occupiedPorts
}

// IsPortOccupied checks if a specific port is occupied
func (pm *portManagerImpl) IsPortOccupied(port Port) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	key := pm.portKey(port.NodeID, port.Edge, port.Position)
	if p, exists := pm.ports[key]; exists {
		return p.ConnectionID != -1
	}
	return false
}

// ReservePort reserves a port for a connection
func (pm *portManagerImpl) ReservePort(nodeID int, edge EdgeSide, connectionID int) (Port, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	node, exists := pm.nodes[nodeID]
	if !exists {
		return Port{}, fmt.Errorf("node %d not found", nodeID)
	}
	
	// Find available port closest to center
	availablePorts := pm.getAvailablePortsUnsafe(nodeID, edge)
	
	// If no ports are available, try stacking on existing ports
	if len(availablePorts) == 0 {
		// Find the port position with the least stacking
		bestPos, stackLevel := pm.findBestStackingPosition(nodeID, edge)
		if bestPos == -1 {
			return Port{}, fmt.Errorf("no ports available for stacking on %s edge of node %d", edgeName(edge), nodeID)
		}
		
		// Create a stacked port
		port := Port{
			NodeID:       nodeID,
			Edge:         edge,
			Position:     bestPos,
			Width:        pm.portWidth,
			Point:        pm.calculateStackedPortPoint(node, edge, bestPos, stackLevel),
			ConnectionID: connectionID,
			StackLevel:   stackLevel,
		}
		
		// Debug
		// fmt.Printf("Creating stacked port: node %d, edge %s, pos %d, stack level %d, point (%d,%d)\n",
		//     nodeID, edgeName(edge), bestPos, stackLevel, port.Point.X, port.Point.Y)
		
		// Store with a key that includes stack level
		key := pm.stackedPortKey(nodeID, edge, bestPos, stackLevel)
		pm.ports[key] = &port
		
		return port, nil
	}
	
	// Select port closest to center
	edgeLength := pm.getEdgeLength(node, edge)
	centerPos := edgeLength / 2
	
	var bestPort *Port
	minDistance := edgeLength
	
	for i := range availablePorts {
		port := &availablePorts[i]
		distance := abs(port.Position - centerPos)
		if distance < minDistance {
			minDistance = distance
			bestPort = port
		}
	}
	
	// Reserve the port
	bestPort.ConnectionID = connectionID
	bestPort.StackLevel = 0 // First level
	key := pm.portKey(nodeID, edge, bestPort.Position)
	pm.ports[key] = bestPort
	
	// Debug
	// fmt.Printf("Creating normal port: node %d, edge %s, pos %d, point (%d,%d)\n",
	//     nodeID, edgeName(edge), bestPort.Position, bestPort.Point.X, bestPort.Point.Y)
	
	return *bestPort, nil
}

// ReleasePort releases a previously reserved port
func (pm *portManagerImpl) ReleasePort(port Port) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	// Use the appropriate key based on stack level
	var key string
	if port.StackLevel > 0 {
		key = pm.stackedPortKey(port.NodeID, port.Edge, port.Position, port.StackLevel)
	} else {
		key = pm.portKey(port.NodeID, port.Edge, port.Position)
	}
	
	if p, exists := pm.ports[key]; exists {
		p.ConnectionID = -1
	}
}

// GetPortForConnection returns the port reserved for a specific connection
func (pm *portManagerImpl) GetPortForConnection(nodeID int, connectionID int) (Port, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	for _, port := range pm.ports {
		if port.NodeID == nodeID && port.ConnectionID == connectionID {
			return *port, true
		}
	}
	
	return Port{}, false
}

// Helper methods

func (pm *portManagerImpl) getEdgeLength(node *core.Node, edge EdgeSide) int {
	switch edge {
	case North, South:
		return node.Width
	case East, West:
		return node.Height
	}
	return 0
}

func (pm *portManagerImpl) calculatePortPoint(node *core.Node, edge EdgeSide, position int) core.Point {
	switch edge {
	case North:
		return core.Point{X: node.X + position, Y: node.Y - 1}
	case South:
		return core.Point{X: node.X + position, Y: node.Y + node.Height}
	case East:
		return core.Point{X: node.X + node.Width, Y: node.Y + position}
	case West:
		return core.Point{X: node.X - 1, Y: node.Y + position}
	}
	return core.Point{}
}

func (pm *portManagerImpl) portKey(nodeID int, edge EdgeSide, position int) string {
	return fmt.Sprintf("%d-%d-%d", nodeID, edge, position)
}

func (pm *portManagerImpl) getAvailablePortsUnsafe(nodeID int, edge EdgeSide) []Port {
	node, exists := pm.nodes[nodeID]
	if !exists {
		return nil
	}
	
	edgeLength := pm.getEdgeLength(node, edge)
	availablePorts := []Port{}
	
	// Debug
	// fmt.Printf("Getting available ports for node %d edge %s: edgeLength=%d, portWidth=%d\n", 
	//     nodeID, edgeName(edge), edgeLength, pm.portWidth)
	
	margin := 1 // Reduced margin to allow ports on smaller edges
	step := pm.portWidth
	
	for pos := margin; pos < edgeLength-margin; pos += step {
		port := Port{
			NodeID:       nodeID,
			Edge:         edge,
			Position:     pos,
			Width:        pm.portWidth,
			Point:        pm.calculatePortPoint(node, edge, pos),
			ConnectionID: -1,
		}
		
		key := pm.portKey(nodeID, edge, pos)
		if existingPort, occupied := pm.ports[key]; !occupied || existingPort.ConnectionID == -1 {
			availablePorts = append(availablePorts, port)
		}
	}
	
	return availablePorts
}

func edgeName(edge EdgeSide) string {
	switch edge {
	case North:
		return "North"
	case South:
		return "South"
	case East:
		return "East"
	case West:
		return "West"
	}
	return "Unknown"
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// findBestStackingPosition finds the position with the least stacking
func (pm *portManagerImpl) findBestStackingPosition(nodeID int, edge EdgeSide) (int, int) {
	node, exists := pm.nodes[nodeID]
	if !exists {
		return -1, 0
	}
	
	edgeLength := pm.getEdgeLength(node, edge)
	margin := 1
	
	// Count stack levels at each position
	stackCounts := make(map[int]int)
	for pos := margin; pos < edgeLength-margin; pos += pm.portWidth {
		stackCounts[pos] = 0
	}
	
	// Count existing ports at each position
	for _, port := range pm.ports {
		if port.NodeID == nodeID && port.Edge == edge && port.ConnectionID != -1 {
			if count, exists := stackCounts[port.Position]; exists {
				if port.StackLevel+1 > count {
					stackCounts[port.Position] = port.StackLevel + 1
				}
			}
		}
	}
	
	// Find position with minimum stacking, preferring center
	centerPos := edgeLength / 2
	bestPos := -1
	minStackLevel := 999999
	minDistanceFromCenter := edgeLength
	
	for pos, stackLevel := range stackCounts {
		distFromCenter := abs(pos - centerPos)
		if stackLevel < minStackLevel || (stackLevel == minStackLevel && distFromCenter < minDistanceFromCenter) {
			bestPos = pos
			minStackLevel = stackLevel
			minDistanceFromCenter = distFromCenter
		}
	}
	
	if bestPos == -1 {
		return -1, 0
	}
	
	return bestPos, minStackLevel
}

// calculateStackedPortPoint calculates the point for a stacked port
func (pm *portManagerImpl) calculateStackedPortPoint(node *core.Node, edge EdgeSide, position int, stackLevel int) core.Point {
	basePoint := pm.calculatePortPoint(node, edge, position)
	
	// For stacked ports, we need to be careful not to place them in blocked areas
	// Use a zigzag pattern along the edge
	
	// Calculate offset direction based on stack level
	direction := 1
	if stackLevel % 2 == 1 {
		direction = -1
	}
	
	// Calculate actual offset amount (increase with each pair)
	offsetAmount := ((stackLevel + 1) / 2)
	
	switch edge {
	case North, South:
		// Offset horizontally for vertical edges
		basePoint.X += direction * offsetAmount
		// Ensure we stay within reasonable bounds
		if basePoint.X < node.X - 2 {
			basePoint.X = node.X - 2
		} else if basePoint.X > node.X + node.Width + 1 {
			basePoint.X = node.X + node.Width + 1
		}
		
	case East, West:
		// Offset vertically for horizontal edges
		basePoint.Y += direction * offsetAmount
		// Ensure we stay within reasonable bounds
		if basePoint.Y < node.Y - 2 {
			basePoint.Y = node.Y - 2
		} else if basePoint.Y > node.Y + node.Height + 1 {
			basePoint.Y = node.Y + node.Height + 1
		}
	}
	
	return basePoint
}

// stackedPortKey creates a unique key for stacked ports
func (pm *portManagerImpl) stackedPortKey(nodeID int, edge EdgeSide, position int, stackLevel int) string {
	return fmt.Sprintf("%d-%d-%d-%d", nodeID, edge, position, stackLevel)
}
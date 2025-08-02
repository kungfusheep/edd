package connections

import (
	"edd/core"
	"fmt"
	"sort"
)

// ConnectionGroup represents a group of connections that share common endpoints.
type ConnectionGroup struct {
	// Key identifies the group (e.g., "nodeA->nodeB")
	Key string
	// Connections in this group
	Connections []core.Connection
	// Indices of these connections in the original list
	Indices []int
}

// GroupConnections groups connections by their endpoints.
// This is useful for handling multiple connections between the same nodes.
func GroupConnections(connections []core.Connection) []ConnectionGroup {
	groups := make(map[string]*ConnectionGroup)
	
	for i, conn := range connections {
		// Create a key for the connection
		key := fmt.Sprintf("%d->%d", conn.From, conn.To)
		
		// Add to existing group or create new one
		if group, exists := groups[key]; exists {
			group.Connections = append(group.Connections, conn)
			group.Indices = append(group.Indices, i)
		} else {
			groups[key] = &ConnectionGroup{
				Key:         key,
				Connections: []core.Connection{conn},
				Indices:     []int{i},
			}
		}
	}
	
	// Convert map to slice and sort for consistent ordering
	result := make([]ConnectionGroup, 0, len(groups))
	for _, group := range groups {
		result = append(result, *group)
	}
	
	// Sort by key for consistent output
	sort.Slice(result, func(i, j int) bool {
		return result[i].Key < result[j].Key
	})
	
	return result
}

// SpreadPoints spreads connection points when multiple connections share the same endpoints.
// This prevents connections from overlapping completely.
func SpreadPoints(basePoint core.Point, count int, index int, node *core.Node, isHorizontal bool) core.Point {
	if count <= 1 {
		return basePoint
	}
	
	// Calculate spacing between connection points
	var spread int
	if isHorizontal {
		// Spread vertically along the node edge
		spread = node.Height / (count + 1)
		offset := spread * (index + 1) - node.Height/2
		return core.Point{
			X: basePoint.X,
			Y: basePoint.Y + offset,
		}
	} else {
		// Spread horizontally along the node edge
		spread = node.Width / (count + 1)
		offset := spread * (index + 1) - node.Width/2
		return core.Point{
			X: basePoint.X + offset,
			Y: basePoint.Y,
		}
	}
}

// OptimizeGroupedPaths optimizes paths within a connection group to reduce overlap.
// It adjusts starting and ending points to spread them out when multiple connections
// exist between the same nodes.
func OptimizeGroupedPaths(group ConnectionGroup, nodes []core.Node, router *Router) (map[int]core.Path, error) {
	paths := make(map[int]core.Path)
	
	// Find source and target nodes
	var sourceNode, targetNode *core.Node
	for i := range nodes {
		if nodes[i].ID == group.Connections[0].From {
			sourceNode = &nodes[i]
		}
		if nodes[i].ID == group.Connections[0].To {
			targetNode = &nodes[i]
		}
	}
	
	if sourceNode == nil || targetNode == nil {
		// Fall back to regular routing
		for i, connIdx := range group.Indices {
			path, err := router.RouteConnection(group.Connections[i], nodes)
			if err != nil {
				return nil, err
			}
			paths[connIdx] = path
		}
		return paths, nil
	}
	
	// Determine if connection is primarily horizontal or vertical
	dx := abs(targetNode.X - sourceNode.X)
	dy := abs(targetNode.Y - sourceNode.Y)
	isHorizontal := dx > dy
	
	// Route each connection with spread points
	for i, connIdx := range group.Indices {
		// Get base connection points
		sourcePoint := getConnectionPoint(sourceNode, targetNode)
		targetPoint := getConnectionPoint(targetNode, sourceNode)
		
		// Spread the connection points
		sourcePoint = SpreadPoints(sourcePoint, len(group.Connections), i, sourceNode, isHorizontal)
		targetPoint = SpreadPoints(targetPoint, len(group.Connections), i, targetNode, isHorizontal)
		
		// Find path between the spread points
		obstacles := createObstaclesFunction(nodes, sourceNode.ID, targetNode.ID)
		path, err := router.pathFinder.FindPath(sourcePoint, targetPoint, obstacles)
		if err != nil {
			return nil, err
		}
		
		paths[connIdx] = path
	}
	
	return paths, nil
}

// HandleSelfLoops handles connections where a node connects to itself.
// These require special routing to create a visible loop.
func HandleSelfLoops(conn core.Connection, node *core.Node) core.Path {
	// Create a loop that goes out from the right side and comes back to the top
	loopSize := 3 // Size of the loop extension
	
	// Start from right side (adjusted for proper connection)
	start := core.Point{
		X: node.X + node.Width - 1,
		Y: node.Y + node.Height/2,
	}
	
	// Create loop points
	points := []core.Point{
		start,
		{X: start.X + loopSize, Y: start.Y},
		{X: start.X + loopSize, Y: node.Y - loopSize},
		{X: node.X + node.Width/2, Y: node.Y - loopSize},
		{X: node.X + node.Width/2, Y: node.Y},
	}
	
	return core.Path{Points: points}
}
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
	
	const minSpacing = 2 // Minimum spacing between connections in characters
	
	// Calculate available edge length and required space
	var edgeLength, requiredSpace int
	if isHorizontal {
		edgeLength = node.Height
		requiredSpace = (count - 1) * minSpacing
	} else {
		edgeLength = node.Width
		requiredSpace = (count - 1) * minSpacing
	}
	
	// Determine spreading strategy based on available space
	var offset int
	if requiredSpace >= edgeLength {
		// Not enough space - distribute evenly across entire edge
		if count > 1 {
			position := float64(index) / float64(count - 1)
			offset = int(position * float64(edgeLength - 1)) - edgeLength/2
		}
	} else {
		// Use 10-90% of edge with proper spacing
		margin := edgeLength / 10
		if margin < 1 {
			margin = 1
		}
		usableSpace := edgeLength - (2 * margin)
		
		// Calculate spacing
		var spacing float64
		if count > 1 {
			spacing = float64(usableSpace) / float64(count - 1)
		}
		
		// Calculate offset from center
		centerOffset := float64(index) * spacing - float64(usableSpace)/2
		offset = int(centerOffset)
	}
	
	// Apply offset based on direction
	if isHorizontal {
		return core.Point{
			X: basePoint.X,
			Y: basePoint.Y + offset,
		}
	} else {
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
		obstacles := createObstaclesFunctionWithPadding(nodes, sourceNode.ID, targetNode.ID, 2)
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
	// Make loop size proportional to node size
	minDimension := node.Width
	if node.Height < minDimension {
		minDimension = node.Height
	}
	
	// Loop size should be at least 3, but scale with node size
	loopSize := minDimension / 3
	if loopSize < 3 {
		loopSize = 3
	} else if loopSize > 8 {
		loopSize = 8 // Cap at reasonable maximum
	}
	
	// Determine best position based on node aspect ratio
	// For wide nodes, prefer top loop; for tall nodes, prefer right loop
	aspectRatio := float64(node.Width) / float64(node.Height)
	
	if aspectRatio > 1.5 {
		// Wide node - use top loop
		start := core.Point{
			X: node.X + node.Width/2,
			Y: node.Y,
		}
		
		points := []core.Point{
			start,
			{X: start.X, Y: start.Y - loopSize},
			{X: node.X + node.Width - 1, Y: start.Y - loopSize},
			{X: node.X + node.Width - 1, Y: node.Y + node.Height/2},
			{X: node.X + node.Width - 1, Y: node.Y + node.Height/2},
		}
		
		return core.Path{Points: points}
	} else {
		// Default: right-side loop (original behavior but adaptive size)
		start := core.Point{
			X: node.X + node.Width - 1,
			Y: node.Y + node.Height/2,
		}
		
		points := []core.Point{
			start,
			{X: start.X + loopSize, Y: start.Y},
			{X: start.X + loopSize, Y: node.Y - loopSize},
			{X: node.X + node.Width/2, Y: node.Y - loopSize},
			{X: node.X + node.Width/2, Y: node.Y},
		}
		
		return core.Path{Points: points}
	}
}
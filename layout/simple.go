package layout

import (
	"edd/core"
	"fmt"
	"sort"
)

// SimpleLayout implements a basic left-to-right layout algorithm.
// Nodes are arranged in columns based on their distance from root nodes.
type SimpleLayout struct {
	horizontalSpacing int
	verticalSpacing   int
	minNodeWidth      int
	minNodeHeight     int
	maxNodeWidth      int
	maxNodesPerColumn int // Maximum nodes in a single column before splitting
}

// NewSimpleLayout creates a SimpleLayout with default settings.
func NewSimpleLayout() *SimpleLayout {
	return &SimpleLayout{
		horizontalSpacing: 4,
		verticalSpacing:   2,
		minNodeWidth:      3,
		minNodeHeight:     3,
		maxNodeWidth:      50,
		maxNodesPerColumn: 10, // Split wide layers into columns of 10
	}
}

// Layout positions nodes in a simple left-to-right arrangement.
func (s *SimpleLayout) Layout(nodes []core.Node, connections []core.Connection) ([]core.Node, error) {
	if len(nodes) == 0 {
		return []core.Node{}, nil
	}
	
	// Create a copy of nodes to avoid modifying input
	result := make([]core.Node, len(nodes))
	copy(result, nodes)
	
	// Calculate dimensions for all nodes
	for i := range result {
		s.calculateNodeDimensions(&result[i])
	}
	
	// Build adjacency information
	nodeMap := make(map[int]int) // ID to index
	for i, node := range result {
		nodeMap[node.ID] = i
	}
	
	// Validate connections
	for _, conn := range connections {
		if _, ok := nodeMap[conn.From]; !ok {
			return nil, fmt.Errorf("invalid connection: node %d not found", conn.From)
		}
		if _, ok := nodeMap[conn.To]; !ok {
			return nil, fmt.Errorf("invalid connection: node %d not found", conn.To)
		}
	}
	
	// Build adjacency lists
	outgoing := make(map[int][]int)
	incoming := make(map[int][]int)
	
	for _, conn := range connections {
		// Skip self-loops for layout purposes
		if conn.From == conn.To {
			continue
		}
		
		outgoing[conn.From] = append(outgoing[conn.From], conn.To)
		incoming[conn.To] = append(incoming[conn.To], conn.From)
	}
	
	// Detect connected components
	components := s.detectComponents(result, outgoing, incoming)
	
	// Layout each component separately
	xOffset := 0
	for _, component := range components {
		// Get nodes for this component
		componentNodes := make([]core.Node, len(component))
		for i, nodeID := range component {
			componentNodes[i] = result[nodeMap[nodeID]]
		}
		
		// Assign layers for this component
		layers := s.assignLayers(componentNodes, outgoing, incoming)
		
		// Position nodes within component
		s.positionNodesWithOffset(result, layers, xOffset)
		
		// Calculate component width for next offset
		maxX := xOffset
		for _, layer := range layers {
			for _, nodeID := range layer {
				node := &result[nodeMap[nodeID]]
				if node.X + node.Width > maxX {
					maxX = node.X + node.Width
				}
			}
		}
		
		// Update offset for next component
		xOffset = maxX + s.horizontalSpacing
	}
	
	return result, nil
}

// Name returns the name of this layout algorithm.
func (s *SimpleLayout) Name() string {
	return "SimpleLayout"
}

// calculateNodeDimensions sets the width and height based on text content.
func (s *SimpleLayout) calculateNodeDimensions(node *core.Node) {
	// Height is number of lines plus borders
	node.Height = len(node.Text) + 2
	if node.Height < s.minNodeHeight {
		node.Height = s.minNodeHeight
	}
	
	// Width is longest line plus borders
	maxWidth := 0
	for _, line := range node.Text {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}
	
	node.Width = maxWidth + 4 // 2 chars padding on each side
	if node.Width < s.minNodeWidth {
		node.Width = s.minNodeWidth
	}
	if node.Width > s.maxNodeWidth {
		node.Width = s.maxNodeWidth
	}
}

// assignLayers determines which vertical layer each node belongs to.
// Uses a modified topological sort for O(n + e) complexity.
func (s *SimpleLayout) assignLayers(nodes []core.Node, outgoing, incoming map[int][]int) [][]int {
	// Calculate in-degrees
	inDegree := make(map[int]int)
	nodeSet := make(map[int]bool)
	for _, node := range nodes {
		nodeSet[node.ID] = true
		inDegree[node.ID] = 0
	}
	
	// Count actual in-degrees (excluding self-loops)
	for nodeID := range nodeSet {
		for _, predID := range incoming[nodeID] {
			if predID != nodeID { // Ignore self-loops
				inDegree[nodeID]++
			}
		}
	}
	
	// Find all nodes with in-degree 0 (roots)
	queue := make([]int, 0)
	for nodeID := range nodeSet {
		if inDegree[nodeID] == 0 {
			queue = append(queue, nodeID)
		}
	}
	sort.Ints(queue) // Deterministic ordering
	
	layers := make([][]int, 0)
	assigned := make(map[int]int)
	processed := 0
	
	// Process nodes layer by layer
	for len(queue) > 0 {
		// Current layer consists of all nodes in queue
		currentLayer := make([]int, len(queue))
		copy(currentLayer, queue)
		layers = append(layers, currentLayer)
		
		// Mark these nodes as assigned
		for _, nodeID := range currentLayer {
			assigned[nodeID] = len(layers) - 1
			processed++
		}
		
		// Prepare next layer
		nextQueue := make([]int, 0)
		for _, nodeID := range currentLayer {
			// Reduce in-degree of all successors
			for _, succID := range outgoing[nodeID] {
				if succID == nodeID { // Skip self-loops
					continue
				}
				inDegree[succID]--
				if inDegree[succID] == 0 {
					if _, alreadyAssigned := assigned[succID]; !alreadyAssigned {
						nextQueue = append(nextQueue, succID)
					}
				}
			}
		}
		
		sort.Ints(nextQueue) // Deterministic ordering
		queue = nextQueue
	}
	
	// Handle any remaining nodes (cycles or disconnected components)
	if processed < len(nodes) {
		remaining := make([]int, 0)
		for _, node := range nodes {
			if _, ok := assigned[node.ID]; !ok {
				remaining = append(remaining, node.ID)
			}
		}
		sort.Ints(remaining)
		
		// For cycles, break them by placing one node arbitrarily
		// For disconnected nodes, they go in their own layer
		if len(remaining) > 0 {
			layers = append(layers, remaining)
		}
	}
	
	return layers
}


// detectComponents finds connected components using DFS.
func (s *SimpleLayout) detectComponents(nodes []core.Node, outgoing, incoming map[int][]int) [][]int {
	visited := make(map[int]bool)
	components := make([][]int, 0)
	
	// Create bidirectional adjacency for component detection
	adjacent := make(map[int][]int)
	for nodeID, successors := range outgoing {
		for _, succ := range successors {
			adjacent[nodeID] = append(adjacent[nodeID], succ)
			adjacent[succ] = append(adjacent[succ], nodeID)
		}
	}
	
	// DFS to find components
	var dfs func(nodeID int, component *[]int)
	dfs = func(nodeID int, component *[]int) {
		if visited[nodeID] {
			return
		}
		visited[nodeID] = true
		*component = append(*component, nodeID)
		
		for _, neighbor := range adjacent[nodeID] {
			dfs(neighbor, component)
		}
	}
	
	// Find all components
	nodeIDs := make([]int, 0, len(nodes))
	for _, node := range nodes {
		nodeIDs = append(nodeIDs, node.ID)
	}
	sort.Ints(nodeIDs) // Process in order for determinism
	
	for _, nodeID := range nodeIDs {
		if !visited[nodeID] {
			component := make([]int, 0)
			dfs(nodeID, &component)
			sort.Ints(component) // Sort for determinism
			components = append(components, component)
		}
	}
	
	return components
}

// positionNodes assigns X,Y coordinates to nodes based on their layers.
func (s *SimpleLayout) positionNodes(nodes []core.Node, layers [][]int) {
	s.positionNodesWithOffset(nodes, layers, 0)
}

// positionNodesWithOffset assigns X,Y coordinates with a starting X offset.
func (s *SimpleLayout) positionNodesWithOffset(nodes []core.Node, layers [][]int, xOffset int) {
	nodeMap := make(map[int]*core.Node)
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}
	
	x := xOffset
	for _, layer := range layers {
		if len(layer) == 0 {
			continue
		}
		
		// Calculate maximum width in this layer
		maxWidth := 0
		for _, nodeID := range layer {
			if node := nodeMap[nodeID]; node != nil && node.Width > maxWidth {
				maxWidth = node.Width
			}
		}
		
		// Position nodes within the layer, distributing into columns if needed
		if len(layer) <= s.maxNodesPerColumn {
			// Single column - original behavior
			y := 0
			for _, nodeID := range layer {
				if node := nodeMap[nodeID]; node != nil {
					node.X = x
					node.Y = y
					y += node.Height + s.verticalSpacing
				}
			}
		} else {
			// Multiple columns needed
			columns := (len(layer) + s.maxNodesPerColumn - 1) / s.maxNodesPerColumn
			nodesPerColumn := (len(layer) + columns - 1) / columns
			
			colX := x
			for col := 0; col < columns; col++ {
				// Calculate column width
				colWidth := 0
				startIdx := col * nodesPerColumn
				endIdx := startIdx + nodesPerColumn
				if endIdx > len(layer) {
					endIdx = len(layer)
				}
				
				// Position nodes in this column
				y := 0
				for i := startIdx; i < endIdx; i++ {
					if node := nodeMap[layer[i]]; node != nil {
						node.X = colX
						node.Y = y
						y += node.Height + s.verticalSpacing
						if node.Width > colWidth {
							colWidth = node.Width
						}
					}
				}
				
				// Move to next column
				colX += colWidth + s.horizontalSpacing
			}
			
			// Update maxWidth to span all columns
			maxWidth = colX - x - s.horizontalSpacing
		}
		
		// Move to next layer
		x += maxWidth + s.horizontalSpacing
	}
}
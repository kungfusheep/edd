package layout

import (
	"edd/core"
	"fmt"
	"math"
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
		horizontalSpacing: 12, // Increased from 8 to allow space for inline labels
		verticalSpacing:   4,
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
		
		// Check if this is a hub-spoke pattern
		isHubSpoke, hubID := s.isHubSpokePattern(component, outgoing, incoming)
		
		// Use different layout strategies based on graph properties
		var layers [][]int
		if isHubSpoke {
			// For hub-spoke patterns, use radial layout
			layers = s.assignRadialLayout(componentNodes, hubID, outgoing, incoming)
		} else {
			// For all other graphs (including cycles), use layered layout
			// The layered layout handles cycles by finding strongly connected components
			layers = s.assignLayers(componentNodes, outgoing, incoming)
		}
		
		// Position nodes within component
		if isHubSpoke {
			s.positionRadialNodesWithOffset(result, layers, xOffset, hubID)
		} else {
			s.positionNodesWithOffset(result, layers, xOffset)
		}
		
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
// For graphs with cycles, it identifies back-edges and ignores them during layout.
func (s *SimpleLayout) assignLayers(nodes []core.Node, outgoing, incoming map[int][]int) [][]int {
	// First, identify back-edges that create cycles
	backEdges := s.findBackEdges(nodes, outgoing)
	
	// Create modified adjacency lists without back-edges
	outgoingNoCycles := make(map[int][]int)
	incomingNoCycles := make(map[int][]int)
	
	for nodeID, neighbors := range outgoing {
		outgoingNoCycles[nodeID] = make([]int, 0)
		for _, neighbor := range neighbors {
			// Skip if this is a back-edge
			isBackEdge := false
			for _, edge := range backEdges {
				if edge.from == nodeID && edge.to == neighbor {
					isBackEdge = true
					break
				}
			}
			if !isBackEdge {
				outgoingNoCycles[nodeID] = append(outgoingNoCycles[nodeID], neighbor)
			}
		}
	}
	
	// Rebuild incoming without back-edges
	for nodeID := range nodes {
		incomingNoCycles[nodes[nodeID].ID] = make([]int, 0)
	}
	for nodeID, neighbors := range outgoingNoCycles {
		for _, neighbor := range neighbors {
			incomingNoCycles[neighbor] = append(incomingNoCycles[neighbor], nodeID)
		}
	}
	
	// Now run normal topological sort on the DAG (without cycles)
	return s.assignLayersDAG(nodes, outgoingNoCycles, incomingNoCycles)
}

// backEdge represents an edge that creates a cycle
type backEdge struct {
	from, to int
}

// findBackEdges identifies edges that create cycles using DFS
func (s *SimpleLayout) findBackEdges(nodes []core.Node, outgoing map[int][]int) []backEdge {
	backEdges := make([]backEdge, 0)
	visited := make(map[int]int) // 0=unvisited, 1=visiting, 2=visited
	
	var dfs func(nodeID int)
	dfs = func(nodeID int) {
		visited[nodeID] = 1 // Mark as visiting
		
		for _, neighbor := range outgoing[nodeID] {
			if visited[neighbor] == 1 {
				// Found a back-edge (cycle)
				backEdges = append(backEdges, backEdge{from: nodeID, to: neighbor})
			} else if visited[neighbor] == 0 {
				dfs(neighbor)
			}
		}
		
		visited[nodeID] = 2 // Mark as visited
	}
	
	// Run DFS from each unvisited node
	for _, node := range nodes {
		if visited[node.ID] == 0 {
			dfs(node.ID)
		}
	}
	
	return backEdges
}

// assignLayersDAG is the original assignLayers logic for acyclic graphs
func (s *SimpleLayout) assignLayersDAG(nodes []core.Node, outgoing, incoming map[int][]int) [][]int {
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
		
		// Try to assign remaining nodes to existing layers based on their connections
		for len(remaining) > 0 {
			placed := false
			for i, nodeID := range remaining {
				// Find the best layer for this node based on its connections
				bestLayer := s.findBestLayerForNode(nodeID, layers, assigned, outgoing, incoming)
				if bestLayer >= 0 && bestLayer < len(layers) {
					layers[bestLayer] = append(layers[bestLayer], nodeID)
					assigned[nodeID] = bestLayer
					remaining = append(remaining[:i], remaining[i+1:]...)
					placed = true
					break
				}
			}
			
			// If we couldn't place any nodes, create a new layer
			if !placed && len(remaining) > 0 {
				// Take the first remaining node and start a new layer
				newLayer := []int{remaining[0]}
				assigned[remaining[0]] = len(layers)
				remaining = remaining[1:]
				layers = append(layers, newLayer)
			}
		}
	}
	
	return layers
}

// findBestLayerForNode determines the best layer for a node based on its connections
func (s *SimpleLayout) findBestLayerForNode(nodeID int, layers [][]int, assigned map[int]int, outgoing, incoming map[int][]int) int {
	// Count connections to nodes in each layer
	layerScores := make(map[int]int)
	
	// Check incoming connections
	for _, predID := range incoming[nodeID] {
		if layer, ok := assigned[predID]; ok {
			// Prefer to be after predecessors
			if layer+1 < len(layers) {
				layerScores[layer+1]++
			}
		}
	}
	
	// Check outgoing connections
	for _, succID := range outgoing[nodeID] {
		if layer, ok := assigned[succID]; ok {
			// Prefer to be before successors
			if layer > 0 {
				layerScores[layer-1]++
			}
		}
	}
	
	// Find layer with highest score
	bestLayer := -1
	bestScore := 0
	for layer, score := range layerScores {
		if score > bestScore {
			bestScore = score
			bestLayer = layer
		}
	}
	
	// If no connections to existing layers, try to find a reasonable position
	if bestLayer == -1 && len(layers) > 0 {
		// Default to middle layer to avoid extreme positions
		bestLayer = len(layers) / 2
	}
	
	return bestLayer
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

// hasCycle detects if the given component contains cycles using DFS
func (s *SimpleLayout) hasCycle(component []int, outgoing map[int][]int) bool {
	// Track visit states: 0 = unvisited, 1 = visiting, 2 = visited
	state := make(map[int]int)
	
	// DFS to detect cycles
	var dfs func(nodeID int) bool
	dfs = func(nodeID int) bool {
		state[nodeID] = 1 // Mark as visiting
		
		for _, neighbor := range outgoing[nodeID] {
			// Only check neighbors within this component
			inComponent := false
			for _, compNode := range component {
				if compNode == neighbor {
					inComponent = true
					break
				}
			}
			if !inComponent {
				continue
			}
			
			if state[neighbor] == 1 {
				// Found a back edge - cycle detected
				return true
			}
			if state[neighbor] == 0 {
				if dfs(neighbor) {
					return true
				}
			}
		}
		
		state[nodeID] = 2 // Mark as visited
		return false
	}
	
	// Check each node in the component
	for _, nodeID := range component {
		if state[nodeID] == 0 {
			if dfs(nodeID) {
				return true
			}
		}
	}
	
	return false
}

// assignGridLayout arranges nodes in a grid pattern for small cyclic graphs
func (s *SimpleLayout) assignGridLayout(nodes []core.Node) [][]int {
	if len(nodes) == 0 {
		return [][]int{}
	}
	
	// Calculate grid dimensions
	// Aim for a roughly square grid, slightly wider than tall
	cols := int(math.Sqrt(float64(len(nodes)))) + 1
	// rows := (len(nodes) + cols - 1) / cols // not needed for column-based layout
	
	// Create layers (each layer is a column in the grid)
	layers := make([][]int, cols)
	
	// Sort nodes by ID for consistent ordering
	nodeIDs := make([]int, len(nodes))
	for i, node := range nodes {
		nodeIDs[i] = node.ID
	}
	sort.Ints(nodeIDs)
	
	// Distribute nodes across columns
	for i, nodeID := range nodeIDs {
		col := i % cols
		layers[col] = append(layers[col], nodeID)
	}
	
	return layers
}

// isHubSpokePattern detects if the component forms a hub-spoke pattern
// Returns true and the hub node ID if pattern is detected
func (s *SimpleLayout) isHubSpokePattern(component []int, outgoing, incoming map[int][]int) (bool, int) {
	// A hub-spoke pattern has one node with many connections and other nodes mostly connected to it
	// Criteria:
	// 1. One node has significantly more connections than others
	// 2. Most other nodes are primarily connected to the hub
	
	if len(component) < 4 {
		return false, -1
	}
	
	// Count connections for each node
	connectionCounts := make(map[int]int)
	for _, nodeID := range component {
		count := len(outgoing[nodeID]) + len(incoming[nodeID])
		connectionCounts[nodeID] = count
	}
	
	// Find node with most connections
	maxConnections := 0
	hubCandidate := -1
	for nodeID, count := range connectionCounts {
		if count > maxConnections {
			maxConnections = count
			hubCandidate = nodeID
		}
	}
	
	// Check if this node has significantly more connections than average
	avgConnections := 0
	for _, count := range connectionCounts {
		avgConnections += count
	}
	avgConnections /= len(component)
	
	// Hub should have at least 3x average connections and connect to >60% of nodes
	minHubConnections := len(component) * 6 / 10 // 60% of nodes
	if maxConnections < avgConnections*3 || maxConnections < minHubConnections {
		return false, -1
	}
	
	// Verify most nodes connect to the hub
	connectedToHub := 0
	for _, nodeID := range component {
		if nodeID == hubCandidate {
			continue
		}
		// Check if this node connects to hub
		isConnected := false
		for _, neighbor := range outgoing[nodeID] {
			if neighbor == hubCandidate {
				isConnected = true
				break
			}
		}
		if !isConnected {
			for _, neighbor := range incoming[nodeID] {
				if neighbor == hubCandidate {
					isConnected = true
					break
				}
			}
		}
		if isConnected {
			connectedToHub++
		}
	}
	
	// At least 70% of non-hub nodes should connect to hub
	if float64(connectedToHub) >= float64(len(component)-1)*0.7 {
		return true, hubCandidate
	}
	
	return false, -1
}

// assignRadialLayout arranges nodes in a radial pattern around a hub
func (s *SimpleLayout) assignRadialLayout(nodes []core.Node, hubID int, outgoing, incoming map[int][]int) [][]int {
	// Create layers: hub in center, spokes around it
	layers := make([][]int, 3)
	
	// Find hub index
	hubIndex := -1
	for i, node := range nodes {
		if node.ID == hubID {
			hubIndex = i
			break
		}
	}
	
	if hubIndex == -1 {
		// Fallback to grid layout
		return s.assignGridLayout(nodes)
	}
	
	// Layer 0: empty (for spacing)
	layers[0] = []int{}
	
	// Layer 1: hub in the middle
	layers[1] = []int{hubID}
	
	// Layer 2: all spoke nodes
	layers[2] = []int{}
	for _, node := range nodes {
		if node.ID != hubID {
			layers[2] = append(layers[2], node.ID)
		}
	}
	
	// Sort spoke nodes for consistent ordering
	sort.Ints(layers[2])
	
	return layers
}

// positionNodes assigns X,Y coordinates to nodes based on their layers.
func (s *SimpleLayout) positionNodes(nodes []core.Node, layers [][]int) {
	s.positionNodesWithOffset(nodes, layers, 0)
}

// positionRadialNodesWithOffset positions nodes in a radial/hub-spoke pattern
func (s *SimpleLayout) positionRadialNodesWithOffset(nodes []core.Node, layers [][]int, xOffset int, hubID int) {
	nodeMap := make(map[int]*core.Node)
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}
	
	// First, calculate the total height needed for all spokes
	totalSpokeHeight := 0
	if len(layers) >= 3 && len(layers[2]) > 0 {
		for _, nodeID := range layers[2] {
			if node := nodeMap[nodeID]; node != nil {
				totalSpokeHeight += node.Height + s.verticalSpacing
			}
		}
		totalSpokeHeight -= s.verticalSpacing // Remove last spacing
	}
	
	// Position hub (layer 1) centered vertically relative to spokes
	if len(layers) >= 2 && len(layers[1]) > 0 {
		hubNode := nodeMap[layers[1][0]]
		if hubNode != nil {
			// Calculate spoke node max width to determine hub position
			maxSpokeWidth := 0
			if len(layers) >= 3 {
				for _, nodeID := range layers[2] {
					if node := nodeMap[nodeID]; node != nil && node.Width > maxSpokeWidth {
						maxSpokeWidth = node.Width
					}
				}
			}
			
			// Position hub to left of spokes with reasonable spacing
			// This creates space for bidirectional connections
			hubNode.X = xOffset + maxSpokeWidth/2
			
			// Center hub vertically relative to spokes
			hubNode.Y = totalSpokeHeight / 2 - hubNode.Height/2
			if hubNode.Y < 0 {
				hubNode.Y = 0
			}
		}
	}
	
	// Position spokes (layer 2) in a column to the right
	if len(layers) >= 3 && len(layers[2]) > 0 {
		spokes := layers[2]
		hubNode := nodeMap[hubID]
		
		// Position spokes with consistent spacing from hub
		// Use double the normal spacing for cleaner routing
		spokeX := hubNode.X + hubNode.Width + s.horizontalSpacing*2
		
		// Distribute spokes vertically
		y := 0
		for _, nodeID := range spokes {
			if node := nodeMap[nodeID]; node != nil {
				node.X = spokeX
				node.Y = y
				y += node.Height + s.verticalSpacing
			}
		}
	}
}

// positionNodesWithOffset assigns X,Y coordinates with a starting X offset.
func (s *SimpleLayout) positionNodesWithOffset(nodes []core.Node, layers [][]int, xOffset int) {
	nodeMap := make(map[int]*core.Node)
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}
	
	// First pass: position all nodes normally
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
			// Single column - position nodes normally first
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
	
	// Second pass: center nodes vertically within each layer based on total height
	layerHeights := make([]int, len(layers))
	maxHeight := 0
	
	// Calculate total height for each layer
	for i, layer := range layers {
		totalHeight := 0
		for j, nodeID := range layer {
			if node := nodeMap[nodeID]; node != nil {
				totalHeight += node.Height
				if j < len(layer)-1 {
					totalHeight += s.verticalSpacing
				}
			}
		}
		layerHeights[i] = totalHeight
		if totalHeight > maxHeight {
			maxHeight = totalHeight
		}
	}
	
	// Now center each layer based on the maximum height
	for layerIdx, layer := range layers {
		if len(layer) > 0 && layerHeights[layerIdx] < maxHeight {
			// Calculate offset to center this layer
			offset := (maxHeight - layerHeights[layerIdx]) / 2
			
			// Apply offset to all nodes in this layer
			for _, nodeID := range layer {
				if node := nodeMap[nodeID]; node != nil {
					node.Y += offset
				}
			}
		}
	}
}
package layout

import (
	"edd/diagram"
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
	maxNodesPerColumn int     // Maximum nodes in a single column before splitting
	hintStrength      float64 // Strength of position hints (0.0 = ignored, 1.0 = absolute)
}

// NewSimpleLayout creates a SimpleLayout with default settings.
func NewSimpleLayout() *SimpleLayout {
	return &SimpleLayout{
		horizontalSpacing: 12, // Increased from 8 to allow space for inline labels
		verticalSpacing:   4,
		minNodeWidth:      3,
		minNodeHeight:     3,
		maxNodeWidth:      50,
		maxNodesPerColumn: 10,  // Split wide layers into columns of 10
		hintStrength:      0.5, // Moderate strength for flow hints
	}
}

// Layout positions nodes in a simple left-to-right arrangement.
func (s *SimpleLayout) Layout(nodes []diagram.Node, connections []diagram.Connection) ([]diagram.Node, error) {
	if len(nodes) == 0 {
		return []diagram.Node{}, nil
	}

	// Create a copy of nodes to avoid modifying input
	result := make([]diagram.Node, len(nodes))
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
		componentNodes := make([]diagram.Node, len(component))
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
				if node.X+node.Width > maxX {
					maxX = node.X + node.Width
				}
			}
		}

		// Update offset for next component
		xOffset = maxX + s.horizontalSpacing
	}

	// Apply strong hints as post-processing if enabled
	if s.hintStrength > 0 {
		result = s.applyStrongHints(result, connections)
	}

	return result, nil
}

// Name returns the name of this layout algorithm.
func (s *SimpleLayout) Name() string {
	return "SimpleLayout"
}

// SetHintStrength sets how strongly position hints affect the layout
// 0.0 = hints are ignored, 1.0 = hints completely override layout
func (s *SimpleLayout) SetHintStrength(strength float64) {
	if strength < 0 {
		strength = 0
	}
	if strength > 1 {
		strength = 1
	}
	s.hintStrength = strength
}

// calculateNodeDimensions sets the width and height based on text content.
func (s *SimpleLayout) calculateNodeDimensions(node *diagram.Node) {
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

// applyStrongHints applies position and flow hints as strong post-processing
func (s *SimpleLayout) applyStrongHints(nodes []diagram.Node, connections []diagram.Connection) []diagram.Node {
	// Build adjacency information for relationship-aware positioning
	nodeMap := make(map[int]*diagram.Node)
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}
	
	// Build connection graph
	neighbors := make(map[int][]int)
	for _, conn := range connections {
		neighbors[conn.From] = append(neighbors[conn.From], conn.To)
		neighbors[conn.To] = append(neighbors[conn.To], conn.From)
	}
	
	// Calculate the bounding box of the current layout
	bounds := s.calculateBounds(nodes)
	
	// Collect nodes with position hints
	hintedNodes := make(map[int]string)
	for i := range nodes {
		if hint, ok := nodes[i].Hints["position"]; ok {
			hintedNodes[nodes[i].ID] = hint
		}
	}
	
	// Apply position hints with relationship awareness
	if len(hintedNodes) > 0 {
		s.applyPositionHintsWithRelationships(nodes, hintedNodes, neighbors, bounds)
	}
	
	// Apply flow hints - these enforce directional constraints
	for _, conn := range connections {
		if flow, ok := conn.Hints["flow"]; ok {
			s.enforceFlowHint(&nodes, conn.From, conn.To, flow)
		}
	}
	
	// Resolve any overlaps created by hints
	s.resolveOverlaps(nodes)
	
	return nodes
}

// calculateBounds finds the bounding box of all nodes
func (s *SimpleLayout) calculateBounds(nodes []diagram.Node) diagram.Bounds {
	if len(nodes) == 0 {
		return diagram.Bounds{}
	}
	
	minX, minY := nodes[0].X, nodes[0].Y
	maxX, maxY := nodes[0].X+nodes[0].Width, nodes[0].Y+nodes[0].Height
	
	for _, node := range nodes[1:] {
		if node.X < minX {
			minX = node.X
		}
		if node.Y < minY {
			minY = node.Y
		}
		if node.X+node.Width > maxX {
			maxX = node.X + node.Width
		}
		if node.Y+node.Height > maxY {
			maxY = node.Y + node.Height
		}
	}
	
	return diagram.Bounds{
		Min: diagram.Point{X: minX, Y: minY},
		Max: diagram.Point{X: maxX, Y: maxY},
	}
}

// applyPositionHintsWithRelationships applies position hints while pulling connected nodes
func (s *SimpleLayout) applyPositionHintsWithRelationships(nodes []diagram.Node, hintedNodes map[int]string, neighbors map[int][]int, bounds diagram.Bounds) {
	// Create a map for quick node lookup
	nodeMap := make(map[int]*diagram.Node)
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}
	
	// First pass: Apply hints to hinted nodes only
	for nodeID, hint := range hintedNodes {
		node := nodeMap[nodeID]
		if node == nil {
			continue
		}
		s.applyPositionHint(node, hint, bounds)
	}
	
	// Second pass: Gently pull direct neighbors closer
	for nodeID := range hintedNodes {
		node := nodeMap[nodeID]
		if node == nil {
			continue
		}
		
		// Only pull immediate neighbors, not recursively
		for _, neighborID := range neighbors[nodeID] {
			// Skip if neighbor is also hinted (they have their own position)
			if _, isHinted := hintedNodes[neighborID]; isHinted {
				continue
			}
			
			neighbor := nodeMap[neighborID]
			if neighbor == nil {
				continue
			}
			
			// Calculate desired position relative to hinted node
			dx := node.X - neighbor.X
			dy := node.Y - neighbor.Y
			
			// Apply gentle pull (30% strength)
			if Abs(dx) > s.horizontalSpacing*3 {
				neighbor.X += dx / 3
			}
			if Abs(dy) > s.verticalSpacing*3 {
				neighbor.Y += dy / 3
			}
		}
	}
}

// applyPositionHint moves a node to its hinted position in the 3x3 grid
func (s *SimpleLayout) applyPositionHint(node *diagram.Node, hint string, bounds diagram.Bounds) {
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y
	
	// Add some padding to avoid placing nodes at the exact edges
	padding := s.horizontalSpacing
	
	// Define zones as thirds of the space
	leftX := bounds.Min.X + padding
	centerX := bounds.Min.X + (width-node.Width)/2
	rightX := bounds.Max.X - node.Width - padding
	
	topY := bounds.Min.Y + padding
	middleY := bounds.Min.Y + (height-node.Height)/2
	bottomY := bounds.Max.Y - node.Height - padding
	
	// Calculate target position based on hint
	switch hint {
	case "top-left":
		node.X = leftX
		node.Y = topY
	case "top-center":
		node.X = centerX
		node.Y = topY
	case "top-right":
		node.X = rightX
		node.Y = topY
	case "middle-left":
		node.X = leftX
		node.Y = middleY
	case "center":
		node.X = centerX
		node.Y = middleY
	case "middle-right":
		node.X = rightX
		node.Y = middleY
	case "bottom-left":
		node.X = leftX
		node.Y = bottomY
	case "bottom-center":
		node.X = centerX
		node.Y = bottomY
	case "bottom-right":
		node.X = rightX
		node.Y = bottomY
	}
}

// enforceFlowHint ensures the target node is positioned according to flow direction
func (s *SimpleLayout) enforceFlowHint(nodes *[]diagram.Node, fromID, toID int, direction string) {
	// Find the nodes
	var fromNode, toNode *diagram.Node
	for i := range *nodes {
		if (*nodes)[i].ID == fromID {
			fromNode = &(*nodes)[i]
		}
		if (*nodes)[i].ID == toID {
			toNode = &(*nodes)[i]
		}
	}
	
	if fromNode == nil || toNode == nil {
		return
	}
	
	// Enforce the flow direction with minimum spacing
	minSpacing := s.horizontalSpacing * 2
	
	switch direction {
	case "right":
		// Target should be to the right of source
		if toNode.X <= fromNode.X+fromNode.Width {
			toNode.X = fromNode.X + fromNode.Width + minSpacing
		}
	case "left":
		// Target should be to the left of source
		if toNode.X+toNode.Width >= fromNode.X {
			toNode.X = fromNode.X - toNode.Width - minSpacing
		}
	case "down":
		// Target should be below source
		if toNode.Y <= fromNode.Y+fromNode.Height {
			toNode.Y = fromNode.Y + fromNode.Height + minSpacing
		}
	case "up":
		// Target should be above source
		if toNode.Y+toNode.Height >= fromNode.Y {
			toNode.Y = fromNode.Y - toNode.Height - minSpacing
		}
	}
}

// resolveOverlaps separates overlapping nodes
func (s *SimpleLayout) resolveOverlaps(nodes []diagram.Node) {
	// Simple overlap resolution - push overlapping nodes apart
	maxIterations := 10
	minSpacing := 2
	
	for iteration := 0; iteration < maxIterations; iteration++ {
		overlapsFound := false
		
		for i := 0; i < len(nodes); i++ {
			for j := i + 1; j < len(nodes); j++ {
				if s.nodesOverlap(&nodes[i], &nodes[j], minSpacing) {
					overlapsFound = true
					s.separateNodes(&nodes[i], &nodes[j], minSpacing)
				}
			}
		}
		
		if !overlapsFound {
			break
		}
	}
}

// nodesOverlap checks if two nodes overlap with given spacing
func (s *SimpleLayout) nodesOverlap(a, b *diagram.Node, spacing int) bool {
	return a.X < b.X+b.Width+spacing && 
	       b.X < a.X+a.Width+spacing &&
	       a.Y < b.Y+b.Height+spacing && 
	       b.Y < a.Y+a.Height+spacing
}

// separateNodes pushes two overlapping nodes apart
func (s *SimpleLayout) separateNodes(a, b *diagram.Node, spacing int) {
	// Calculate centers
	aCenterX := a.X + a.Width/2
	aCenterY := a.Y + a.Height/2
	bCenterX := b.X + b.Width/2
	bCenterY := b.Y + b.Height/2
	
	// Calculate separation vector
	dx := bCenterX - aCenterX
	dy := bCenterY - aCenterY
	
	// If nodes are at same position, push horizontally
	if dx == 0 && dy == 0 {
		dx = 1
	}
	
	// Determine primary separation direction (horizontal or vertical)
	if Abs(dx) > Abs(dy) {
		// Separate horizontally
		if dx > 0 {
			// B is to the right of A
			overlap := (a.X + a.Width + spacing) - b.X
			if overlap > 0 {
				b.X += overlap
			}
		} else {
			// B is to the left of A
			overlap := (b.X + b.Width + spacing) - a.X
			if overlap > 0 {
				a.X += overlap
			}
		}
	} else {
		// Separate vertically
		if dy > 0 {
			// B is below A
			overlap := (a.Y + a.Height + spacing) - b.Y
			if overlap > 0 {
				b.Y += overlap
			}
		} else {
			// B is above A
			overlap := (b.Y + b.Height + spacing) - a.Y
			if overlap > 0 {
				a.Y += overlap
			}
		}
	}
}


// assignLayers determines which vertical layer each node belongs to.
// Uses a modified topological sort for O(n + e) complexity.
// For graphs with cycles, it identifies back-edges and ignores them during layout.
func (s *SimpleLayout) assignLayers(nodes []diagram.Node, outgoing, incoming map[int][]int) [][]int {
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
func (s *SimpleLayout) findBackEdges(nodes []diagram.Node, outgoing map[int][]int) []backEdge {
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
func (s *SimpleLayout) assignLayersDAG(nodes []diagram.Node, outgoing, incoming map[int][]int) [][]int {
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
func (s *SimpleLayout) detectComponents(nodes []diagram.Node, outgoing, incoming map[int][]int) [][]int {
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


// assignGridLayout arranges nodes in a grid pattern for small cyclic graphs
func (s *SimpleLayout) assignGridLayout(nodes []diagram.Node) [][]int {
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
func (s *SimpleLayout) assignRadialLayout(nodes []diagram.Node, hubID int, outgoing, incoming map[int][]int) [][]int {
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


// positionRadialNodesWithOffset positions nodes in a radial/hub-spoke pattern
func (s *SimpleLayout) positionRadialNodesWithOffset(nodes []diagram.Node, layers [][]int, xOffset int, hubID int) {
	nodeMap := make(map[int]*diagram.Node)
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}

	// Position hub (layer 1) 
	if len(layers) >= 2 && len(layers[1]) > 0 {
		hubNode := nodeMap[layers[1][0]]
		if hubNode != nil {
			// Position hub at the left
			hubNode.X = xOffset
			hubNode.Y = 0 // Will be adjusted after spokes are positioned
		}
	}

	// Position spokes (layer 2) in columns if needed
	if len(layers) >= 3 && len(layers[2]) > 0 {
		spokes := layers[2]
		hubNode := nodeMap[hubID]
		
		// Determine how many columns we need
		numSpokes := len(spokes)
		columns := (numSpokes + s.maxNodesPerColumn - 1) / s.maxNodesPerColumn
		if columns < 1 {
			columns = 1
		}
		
		// Calculate spokes per column
		spokesPerColumn := (numSpokes + columns - 1) / columns
		
		// Position spokes in columns
		baseX := hubNode.X + hubNode.Width + s.horizontalSpacing*2
		maxY := 0
		minY := 0
		
		for i, nodeID := range spokes {
			if node := nodeMap[nodeID]; node != nil {
				col := i / spokesPerColumn
				rowInCol := i % spokesPerColumn
				
				// Calculate X position based on column
				node.X = baseX + col*(node.Width+s.horizontalSpacing)
				
				// Calculate Y position within column
				node.Y = rowInCol * (node.Height + s.verticalSpacing)
				
				// Track min/max Y for centering the hub
				if i == 0 || node.Y < minY {
					minY = node.Y
				}
				if node.Y+node.Height > maxY {
					maxY = node.Y + node.Height
				}
			}
		}
		
		// Center hub vertically relative to spokes
		if hubNode != nil {
			hubNode.Y = minY + (maxY-minY-hubNode.Height)/2
			if hubNode.Y < 0 {
				hubNode.Y = 0
			}
		}
	}
}

// positionNodesWithOffset assigns X,Y coordinates with a starting X offset.
func (s *SimpleLayout) positionNodesWithOffset(nodes []diagram.Node, layers [][]int, xOffset int) {
	nodeMap := make(map[int]*diagram.Node)
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


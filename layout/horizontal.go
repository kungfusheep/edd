package layout

import (
	"edd/diagram"
	"sort"
)

// HorizontalLayout implements a left-to-right layout algorithm for flowcharts.
// This is designed for pipelines, timelines, and process flows where flow goes rightward.
type HorizontalLayout struct {
	horizontalSpacing int
	verticalSpacing   int
	minNodeWidth      int
	minNodeHeight     int
	maxNodeWidth      int
}

// NewHorizontalLayout creates a HorizontalLayout with default settings.
func NewHorizontalLayout() *HorizontalLayout {
	return &HorizontalLayout{
		horizontalSpacing: 8,
		verticalSpacing:   4,
		minNodeWidth:      3,
		minNodeHeight:     3,
		maxNodeWidth:      50,
	}
}

// Layout positions nodes in a left-to-right arrangement.
func (h *HorizontalLayout) Layout(nodes []diagram.Node, connections []diagram.Connection) ([]diagram.Node, error) {
	if len(nodes) == 0 {
		return []diagram.Node{}, nil
	}

	// Create a copy of nodes to avoid modifying input
	result := make([]diagram.Node, len(nodes))
	copy(result, nodes)

	// Calculate dimensions for all nodes
	for i := range result {
		h.calculateNodeDimensions(&result[i])
	}

	// Build adjacency information
	nodeMap := make(map[int]int) // ID to index
	for i, node := range result {
		nodeMap[node.ID] = i
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

	// Assign nodes to columns (left to right)
	columns := h.assignColumns(result, outgoing, incoming)

	// Position nodes within each column
	h.positionNodes(result, columns, nodeMap)

	return result, nil
}

// Name returns the name of this layout algorithm.
func (h *HorizontalLayout) Name() string {
	return "HorizontalLayout"
}

// calculateNodeDimensions sets the width and height based on text content.
func (h *HorizontalLayout) calculateNodeDimensions(node *diagram.Node) {
	// Height is number of lines plus borders
	node.Height = len(node.Text) + 2
	if node.Height < h.minNodeHeight {
		node.Height = h.minNodeHeight
	}

	// Width is longest line plus borders
	maxWidth := 0
	for _, line := range node.Text {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}

	node.Width = maxWidth + 4 // 2 chars padding on each side
	if node.Width < h.minNodeWidth {
		node.Width = h.minNodeWidth
	}
	if node.Width > h.maxNodeWidth {
		node.Width = h.maxNodeWidth
	}
}

// assignColumns determines which horizontal column each node belongs to.
func (h *HorizontalLayout) assignColumns(nodes []diagram.Node, outgoing, incoming map[int][]int) [][]int {
	// Calculate in-degrees
	inDegree := make(map[int]int)
	nodeSet := make(map[int]bool)
	for _, node := range nodes {
		nodeSet[node.ID] = true
		inDegree[node.ID] = 0
	}

	// Count actual in-degrees
	for nodeID := range nodeSet {
		for _, predID := range incoming[nodeID] {
			if predID != nodeID { // Ignore self-loops
				inDegree[nodeID]++
			}
		}
	}

	// Find all nodes with in-degree 0 (leftmost/starting nodes)
	queue := make([]int, 0)
	for nodeID := range nodeSet {
		if inDegree[nodeID] == 0 {
			queue = append(queue, nodeID)
		}
	}
	sort.Ints(queue) // Deterministic ordering

	columns := make([][]int, 0)
	assigned := make(map[int]int)
	processed := 0

	// Process nodes column by column
	for len(queue) > 0 {
		// Current column consists of all nodes in queue
		currentColumn := make([]int, len(queue))
		copy(currentColumn, queue)
		columns = append(columns, currentColumn)

		// Mark these nodes as assigned
		for _, nodeID := range currentColumn {
			assigned[nodeID] = len(columns) - 1
			processed++
		}

		// Prepare next column
		nextQueue := make([]int, 0)
		for _, nodeID := range currentColumn {
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

	// Handle any remaining nodes (disconnected components)
	if processed < len(nodes) {
		remaining := make([]int, 0)
		for _, node := range nodes {
			if _, ok := assigned[node.ID]; !ok {
				remaining = append(remaining, node.ID)
			}
		}
		if len(remaining) > 0 {
			columns = append(columns, remaining)
		}
	}

	return columns
}

// positionNodes positions nodes within their assigned columns.
func (h *HorizontalLayout) positionNodes(nodes []diagram.Node, columns [][]int, nodeMap map[int]int) {
	x := 0

	for _, column := range columns {
		if len(column) == 0 {
			continue
		}

		// Calculate total height needed for this column
		totalHeight := 0
		maxWidth := 0
		for i, nodeID := range column {
			node := &nodes[nodeMap[nodeID]]
			totalHeight += node.Height
			if i > 0 {
				totalHeight += h.verticalSpacing
			}
			if node.Width > maxWidth {
				maxWidth = node.Width
			}
		}

		// Start y position (centered around 0, will adjust later)
		y := -totalHeight / 2

		// Position each node in this column
		for _, nodeID := range column {
			node := &nodes[nodeMap[nodeID]]
			node.X = x
			node.Y = y
			y += node.Height + h.verticalSpacing
		}

		// Move to next column
		x += maxWidth + h.horizontalSpacing
	}

	// Find the minimum Y coordinate
	minY := 0
	for i := range nodes {
		if nodes[i].Y < minY {
			minY = nodes[i].Y
		}
	}

	// Shift all nodes to ensure Y coordinates are non-negative
	if minY < 0 {
		offset := -minY + 2 // Add small margin
		for i := range nodes {
			nodes[i].Y += offset
		}
	}
}

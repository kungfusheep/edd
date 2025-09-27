package layout

import (
	"edd/diagram"
	"sort"
)

// VerticalLayout implements a top-to-bottom layout algorithm for flowcharts.
// This is designed for decision trees and flowcharts where flow goes downward.
type VerticalLayout struct {
	horizontalSpacing int
	verticalSpacing   int
	minNodeWidth      int
	minNodeHeight     int
	maxNodeWidth      int
}

// NewVerticalLayout creates a VerticalLayout with default settings.
func NewVerticalLayout() *VerticalLayout {
	return &VerticalLayout{
		horizontalSpacing: 8,
		verticalSpacing:   4,
		minNodeWidth:      3,
		minNodeHeight:     3,
		maxNodeWidth:      50,
	}
}

// Layout positions nodes in a top-to-bottom arrangement.
func (v *VerticalLayout) Layout(nodes []diagram.Node, connections []diagram.Connection) ([]diagram.Node, error) {
	if len(nodes) == 0 {
		return []diagram.Node{}, nil
	}

	// Create a copy of nodes to avoid modifying input
	result := make([]diagram.Node, len(nodes))
	copy(result, nodes)

	// Calculate dimensions for all nodes
	for i := range result {
		v.calculateNodeDimensions(&result[i])
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

	// Assign nodes to levels (top to bottom)
	levels := v.assignLevels(result, outgoing, incoming)

	// Position nodes within each level
	v.positionNodes(result, levels, nodeMap)

	return result, nil
}

// Name returns the name of this layout algorithm.
func (v *VerticalLayout) Name() string {
	return "VerticalLayout"
}

// calculateNodeDimensions sets the width and height based on text content.
func (v *VerticalLayout) calculateNodeDimensions(node *diagram.Node) {
	// Height is number of lines plus borders
	node.Height = len(node.Text) + 2
	if node.Height < v.minNodeHeight {
		node.Height = v.minNodeHeight
	}

	// Width is longest line plus borders
	maxWidth := 0
	for _, line := range node.Text {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}

	node.Width = maxWidth + 4 // 2 chars padding on each side
	if node.Width < v.minNodeWidth {
		node.Width = v.minNodeWidth
	}
	if node.Width > v.maxNodeWidth {
		node.Width = v.maxNodeWidth
	}
}

// assignLevels determines which vertical level (row) each node belongs to.
func (v *VerticalLayout) assignLevels(nodes []diagram.Node, outgoing, incoming map[int][]int) [][]int {
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

	// Find all nodes with in-degree 0 (roots)
	queue := make([]int, 0)
	for nodeID := range nodeSet {
		if inDegree[nodeID] == 0 {
			queue = append(queue, nodeID)
		}
	}
	sort.Ints(queue) // Deterministic ordering

	levels := make([][]int, 0)
	assigned := make(map[int]int)
	processed := 0

	// Process nodes level by level
	for len(queue) > 0 {
		// Current level consists of all nodes in queue
		currentLevel := make([]int, len(queue))
		copy(currentLevel, queue)
		levels = append(levels, currentLevel)

		// Mark these nodes as assigned
		for _, nodeID := range currentLevel {
			assigned[nodeID] = len(levels) - 1
			processed++
		}

		// Prepare next level
		nextQueue := make([]int, 0)
		for _, nodeID := range currentLevel {
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
			levels = append(levels, remaining)
		}
	}

	return levels
}

// positionNodes positions nodes within their assigned levels.
func (v *VerticalLayout) positionNodes(nodes []diagram.Node, levels [][]int, nodeMap map[int]int) {
	y := 0

	for _, level := range levels {
		if len(level) == 0 {
			continue
		}

		// Calculate total width needed for this level
		totalWidth := 0
		maxHeight := 0
		for i, nodeID := range level {
			node := &nodes[nodeMap[nodeID]]
			totalWidth += node.Width
			if i > 0 {
				totalWidth += v.horizontalSpacing
			}
			if node.Height > maxHeight {
				maxHeight = node.Height
			}
		}

		// Start x position (centered around 0, will adjust later)
		x := -totalWidth / 2

		// Position each node in this level
		for _, nodeID := range level {
			node := &nodes[nodeMap[nodeID]]
			node.X = x
			node.Y = y
			x += node.Width + v.horizontalSpacing
		}

		// Move to next level
		y += maxHeight + v.verticalSpacing
	}

	// Find the minimum X coordinate
	minX := 0
	for i := range nodes {
		if nodes[i].X < minX {
			minX = nodes[i].X
		}
	}

	// Shift all nodes to ensure X coordinates are non-negative
	if minX < 0 {
		offset := -minX + 2 // Add small margin
		for i := range nodes {
			nodes[i].X += offset
		}
	}
}
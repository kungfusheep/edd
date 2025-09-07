package layout

import (
	"edd/diagram"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// TestValidator provides comprehensive validation for layout tests.
type TestValidator struct {
	t *testing.T
}

// NewTestValidator creates a validator for the given test.
func NewTestValidator(t *testing.T) *TestValidator {
	return &TestValidator{t: t}
}

// ValidateNoOverlaps ensures no two nodes occupy the same space.
func (v *TestValidator) ValidateNoOverlaps(nodes []diagram.Node) {
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			if v.nodesOverlap(nodes[i], nodes[j]) {
				v.t.Errorf("Nodes %d and %d overlap: %v and %v",
					nodes[i].ID, nodes[j].ID,
					v.nodeBounds(nodes[i]), v.nodeBounds(nodes[j]))
			}
		}
	}
}

// ValidateSpacing ensures minimum spacing between nodes.
func (v *TestValidator) ValidateSpacing(nodes []diagram.Node, minSpacing int) {
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			dist := v.nodeDistance(nodes[i], nodes[j])
			if dist < minSpacing && dist >= 0 {
				v.t.Errorf("Nodes %d and %d too close: distance %d < minimum %d",
					nodes[i].ID, nodes[j].ID, dist, minSpacing)
			}
		}
	}
}

// ValidateBounds ensures all nodes fit within reasonable canvas size.
func (v *TestValidator) ValidateBounds(nodes []diagram.Node, maxWidth, maxHeight int) {
	for _, node := range nodes {
		if node.X < 0 || node.Y < 0 {
			v.t.Errorf("Node %d has negative position: (%d, %d)",
				node.ID, node.X, node.Y)
		}
		if node.X+node.Width > maxWidth {
			v.t.Errorf("Node %d exceeds max width: %d > %d",
				node.ID, node.X+node.Width, maxWidth)
		}
		if node.Y+node.Height > maxHeight {
			v.t.Errorf("Node %d exceeds max height: %d > %d",
				node.ID, node.Y+node.Height, maxHeight)
		}
	}
}

// ValidateDeterminism ensures layout is consistent across runs.
func (v *TestValidator) ValidateDeterminism(
	layout LayoutEngine,
	nodes []diagram.Node,
	connections []diagram.Connection,
	runs int,
) {
	var firstResult []diagram.Node
	
	for i := 0; i < runs; i++ {
		result, err := layout.Layout(nodes, connections)
		if err != nil {
			v.t.Fatalf("Layout failed on run %d: %v", i, err)
		}
		
		if i == 0 {
			firstResult = result
		} else {
			if !v.layoutsEqual(firstResult, result) {
				v.t.Errorf("Layout not deterministic: run %d differs from run 0", i)
			}
		}
	}
}

// ValidatePerformance ensures layout completes within time limit.
func (v *TestValidator) ValidatePerformance(
	layout LayoutEngine,
	nodes []diagram.Node,
	connections []diagram.Connection,
	maxDuration time.Duration,
) {
	start := time.Now()
	_, err := layout.Layout(nodes, connections)
	duration := time.Since(start)
	
	if err != nil {
		v.t.Fatalf("Layout failed: %v", err)
	}
	
	if duration > maxDuration {
		v.t.Errorf("Layout too slow: %v > %v", duration, maxDuration)
	}
}

// ValidateNodeSizes ensures all nodes have positive dimensions.
func (v *TestValidator) ValidateNodeSizes(nodes []diagram.Node) {
	for _, node := range nodes {
		if node.Width <= 0 {
			v.t.Errorf("Node %d has invalid width: %d", node.ID, node.Width)
		}
		if node.Height <= 0 {
			v.t.Errorf("Node %d has invalid height: %d", node.ID, node.Height)
		}
	}
}

// Helper methods

func (v *TestValidator) nodesOverlap(a, b diagram.Node) bool {
	return !(a.X+a.Width <= b.X || b.X+b.Width <= a.X ||
		a.Y+a.Height <= b.Y || b.Y+b.Height <= a.Y)
}

func (v *TestValidator) nodeDistance(a, b diagram.Node) int {
	// Calculate minimum distance between node edges
	if v.nodesOverlap(a, b) {
		return -1 // Overlapping
	}
	
	// Horizontal distance
	hDist := 0
	if a.X+a.Width < b.X {
		hDist = b.X - (a.X + a.Width)
	} else if b.X+b.Width < a.X {
		hDist = a.X - (b.X + b.Width)
	}
	
	// Vertical distance
	vDist := 0
	if a.Y+a.Height < b.Y {
		vDist = b.Y - (a.Y + a.Height)
	} else if b.Y+b.Height < a.Y {
		vDist = a.Y - (b.Y + b.Height)
	}
	
	// Return Manhattan distance
	return hDist + vDist
}

func (v *TestValidator) nodeBounds(n diagram.Node) string {
	return fmt.Sprintf("[%d,%d - %d,%d]", n.X, n.Y, n.X+n.Width, n.Y+n.Height)
}

func (v *TestValidator) layoutsEqual(a, b []diagram.Node) bool {
	if len(a) != len(b) {
		return false
	}
	
	// Create maps for quick lookup
	aMap := make(map[int]diagram.Node)
	for _, node := range a {
		aMap[node.ID] = node
	}
	
	for _, bNode := range b {
		aNode, exists := aMap[bNode.ID]
		if !exists {
			return false
		}
		if aNode.X != bNode.X || aNode.Y != bNode.Y ||
			aNode.Width != bNode.Width || aNode.Height != bNode.Height {
			return false
		}
	}
	
	return true
}

// Graph generators for stress testing

// GenerateLinearChain creates a simple A->B->C... chain.
func GenerateLinearChain(length int) ([]diagram.Node, []diagram.Connection) {
	nodes := make([]diagram.Node, length)
	connections := make([]diagram.Connection, 0)
	
	for i := 0; i < length; i++ {
		nodes[i] = diagram.Node{
			ID:   i,
			Text: []string{fmt.Sprintf("Node %d", i)},
		}
		if i > 0 {
			connections = append(connections, diagram.Connection{
				From: i - 1,
				To:   i,
			})
		}
	}
	
	return nodes, connections
}

// GenerateTree creates a tree with specified branching factor.
func GenerateTree(depth, branchingFactor int) ([]diagram.Node, []diagram.Connection) {
	nodes := make([]diagram.Node, 0)
	connections := make([]diagram.Connection, 0)
	nodeID := 0
	
	var generateLevel func(parentID, level int)
	generateLevel = func(parentID, level int) {
		if level >= depth {
			return
		}
		
		for i := 0; i < branchingFactor; i++ {
			nodeID++
			nodes = append(nodes, diagram.Node{
				ID:   nodeID,
				Text: []string{fmt.Sprintf("N%d", nodeID)},
			})
			
			if parentID >= 0 {
				connections = append(connections, diagram.Connection{
					From: parentID,
					To:   nodeID,
				})
			}
			
			generateLevel(nodeID, level+1)
		}
	}
	
	// Create root
	nodes = append(nodes, diagram.Node{
		ID:   0,
		Text: []string{"Root"},
	})
	generateLevel(0, 1)
	
	return nodes, connections
}

// GenerateStarGraph creates a hub connected to many spokes.
func GenerateStarGraph(spokeCount int) ([]diagram.Node, []diagram.Connection) {
	nodes := make([]diagram.Node, spokeCount+1)
	connections := make([]diagram.Connection, spokeCount)
	
	// Hub
	nodes[0] = diagram.Node{
		ID:   0,
		Text: []string{"Hub"},
	}
	
	// Spokes
	for i := 1; i <= spokeCount; i++ {
		nodes[i] = diagram.Node{
			ID:   i,
			Text: []string{fmt.Sprintf("Spoke %d", i)},
		}
		connections[i-1] = diagram.Connection{
			From: 0,
			To:   i,
		}
	}
	
	return nodes, connections
}

// GenerateCompleteGraph creates a graph where every node connects to every other.
func GenerateCompleteGraph(nodeCount int) ([]diagram.Node, []diagram.Connection) {
	nodes := make([]diagram.Node, nodeCount)
	connections := make([]diagram.Connection, 0)
	
	for i := 0; i < nodeCount; i++ {
		nodes[i] = diagram.Node{
			ID:   i,
			Text: []string{fmt.Sprintf("N%d", i)},
		}
		
		for j := 0; j < i; j++ {
			connections = append(connections, diagram.Connection{
				From: j,
				To:   i,
			})
		}
	}
	
	return nodes, connections
}

// GenerateCycle creates a simple cycle A->B->...->A.
func GenerateCycle(length int) ([]diagram.Node, []diagram.Connection) {
	nodes, connections := GenerateLinearChain(length)
	
	// Close the cycle
	connections = append(connections, diagram.Connection{
		From: length - 1,
		To:   0,
	})
	
	return nodes, connections
}

// GenerateRandomDAG creates a random directed acyclic graph.
func GenerateRandomDAG(nodeCount int, edgeProbability float64) ([]diagram.Node, []diagram.Connection) {
	rand.Seed(time.Now().UnixNano())
	
	nodes := make([]diagram.Node, nodeCount)
	connections := make([]diagram.Connection, 0)
	
	for i := 0; i < nodeCount; i++ {
		nodes[i] = diagram.Node{
			ID:   i,
			Text: []string{fmt.Sprintf("N%d", i)},
		}
	}
	
	// Create edges only from lower to higher numbered nodes (ensures DAG)
	for i := 0; i < nodeCount; i++ {
		for j := i + 1; j < nodeCount; j++ {
			if rand.Float64() < edgeProbability {
				connections = append(connections, diagram.Connection{
					From: i,
					To:   j,
				})
			}
		}
	}
	
	return nodes, connections
}

// GenerateDisconnectedComponents creates multiple separate graphs.
func GenerateDisconnectedComponents(componentCount, nodesPerComponent int) ([]diagram.Node, []diagram.Connection) {
	nodes := make([]diagram.Node, 0)
	connections := make([]diagram.Connection, 0)
	nodeID := 0
	
	for c := 0; c < componentCount; c++ {
		// Create a small chain for each component
		for i := 0; i < nodesPerComponent; i++ {
			nodes = append(nodes, diagram.Node{
				ID:   nodeID,
				Text: []string{fmt.Sprintf("C%d-N%d", c, i)},
			})
			
			if i > 0 {
				connections = append(connections, diagram.Connection{
					From: nodeID - 1,
					To:   nodeID,
				})
			}
			
			nodeID++
		}
	}
	
	return nodes, connections
}

// GenerateTextSizeVariations creates nodes with extreme text variations.
func GenerateTextSizeVariations() []diagram.Node {
	return []diagram.Node{
		{ID: 0, Text: []string{""}}, // Empty
		{ID: 1, Text: []string{"A"}}, // Single char
		{ID: 2, Text: []string{"This is a very long node label that should test width handling in the layout algorithm"}},
		{ID: 3, Text: []string{"Line 1", "Line 2", "Line 3", "Line 4", "Line 5"}}, // Multi-line
		{ID: 4, Text: []string{"Normal Node"}}, // Normal
		{ID: 5, Text: make([]string, 20)}, // 20 empty lines
	}
}
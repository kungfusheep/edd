package layout

import (
	"edd/core"
	"fmt"
	"testing"
	"time"
)

// TestSimpleLayout_BasicCorrectness tests fundamental layout behavior.
func TestSimpleLayout_BasicCorrectness(t *testing.T) {
	layout := NewSimpleLayout()
	validator := NewTestValidator(t)
	
	t.Run("Empty graph", func(t *testing.T) {
		nodes := []core.Node{}
		connections := []core.Connection{}
		
		result, err := layout.Layout(nodes, connections)
		if err != nil {
			t.Fatalf("Failed on empty graph: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("Expected 0 nodes, got %d", len(result))
		}
	})
	
	t.Run("Single node", func(t *testing.T) {
		nodes := []core.Node{
			{ID: 0, Text: []string{"Single Node"}},
		}
		connections := []core.Connection{}
		
		result, err := layout.Layout(nodes, connections)
		if err != nil {
			t.Fatalf("Failed on single node: %v", err)
		}
		
		if len(result) != 1 {
			t.Fatalf("Expected 1 node, got %d", len(result))
		}
		
		// Single node should be at origin
		if result[0].X != 0 || result[0].Y != 0 {
			t.Errorf("Single node not at origin: (%d, %d)", result[0].X, result[0].Y)
		}
		
		// Should have calculated dimensions
		if result[0].Width <= 0 || result[0].Height <= 0 {
			t.Errorf("Invalid dimensions: %dx%d", result[0].Width, result[0].Height)
		}
		
		validator.ValidateNodeSizes(result)
	})
	
	t.Run("Linear chain", func(t *testing.T) {
		for _, length := range []int{2, 5, 10, 50} {
			nodes, connections := GenerateLinearChain(length)
			
			result, err := layout.Layout(nodes, connections)
			if err != nil {
				t.Fatalf("Failed on chain of length %d: %v", length, err)
			}
			
			validator.ValidateNoOverlaps(result)
			validator.ValidateNodeSizes(result)
			validator.ValidateSpacing(result, 1) // Minimum 1 unit spacing
			
			// Verify left-to-right ordering
			for i := 1; i < len(result); i++ {
				if result[i].X <= result[i-1].X {
					t.Errorf("Node %d not to the right of node %d", i, i-1)
				}
			}
		}
	})
	
	t.Run("Simple tree", func(t *testing.T) {
		nodes, connections := GenerateTree(3, 2) // depth 3, branching factor 2
		
		result, err := layout.Layout(nodes, connections)
		if err != nil {
			t.Fatalf("Failed on tree: %v", err)
		}
		
		validator.ValidateNoOverlaps(result)
		validator.ValidateNodeSizes(result)
		validator.ValidateSpacing(result, 1)
	})
	
	t.Run("Disconnected components", func(t *testing.T) {
		nodes, connections := GenerateDisconnectedComponents(3, 4)
		
		result, err := layout.Layout(nodes, connections)
		if err != nil {
			t.Fatalf("Failed on disconnected graph: %v", err)
		}
		
		validator.ValidateNoOverlaps(result)
		validator.ValidateNodeSizes(result)
		validator.ValidateSpacing(result, 1)
	})
}

// TestSimpleLayout_TextSizeHandling tests various text size edge cases.
func TestSimpleLayout_TextSizeHandling(t *testing.T) {
	layout := NewSimpleLayout()
	validator := NewTestValidator(t)
	
	t.Run("Text size variations", func(t *testing.T) {
		nodes := GenerateTextSizeVariations()
		connections := []core.Connection{} // No connections
		
		result, err := layout.Layout(nodes, connections)
		if err != nil {
			t.Fatalf("Failed with text variations: %v", err)
		}
		
		validator.ValidateNoOverlaps(result)
		validator.ValidateNodeSizes(result)
		
		// Verify empty text still gets minimum size
		for _, node := range result {
			if node.Width < 3 || node.Height < 3 {
				t.Errorf("Node %d too small: %dx%d", node.ID, node.Width, node.Height)
			}
		}
	})
	
	t.Run("Very long text", func(t *testing.T) {
		longText := ""
		for i := 0; i < 200; i++ {
			longText += "X"
		}
		
		nodes := []core.Node{
			{ID: 0, Text: []string{longText}},
		}
		connections := []core.Connection{}
		
		result, err := layout.Layout(nodes, connections)
		if err != nil {
			t.Fatalf("Failed with long text: %v", err)
		}
		
		// Should handle gracefully, perhaps with max width
		if result[0].Width > 100 {
			t.Logf("Warning: Very wide node created: %d", result[0].Width)
		}
	})
	
	t.Run("Many lines", func(t *testing.T) {
		lines := make([]string, 50)
		for i := range lines {
			lines[i] = "Line"
		}
		
		nodes := []core.Node{
			{ID: 0, Text: lines},
		}
		connections := []core.Connection{}
		
		result, err := layout.Layout(nodes, connections)
		if err != nil {
			t.Fatalf("Failed with many lines: %v", err)
		}
		
		if result[0].Height < len(lines) {
			t.Errorf("Height too small for %d lines: %d", len(lines), result[0].Height)
		}
	})
}

// TestSimpleLayout_TopologyStress tests extreme graph structures.
func TestSimpleLayout_TopologyStress(t *testing.T) {
	layout := NewSimpleLayout()
	validator := NewTestValidator(t)
	
	t.Run("Star graph", func(t *testing.T) {
		for _, spokeCount := range []int{5, 20, 100} {
			nodes, connections := GenerateStarGraph(spokeCount)
			
			result, err := layout.Layout(nodes, connections)
			if err != nil {
				t.Fatalf("Failed on star with %d spokes: %v", spokeCount, err)
			}
			
			validator.ValidateNoOverlaps(result)
			validator.ValidateNodeSizes(result)
			validator.ValidateBounds(result, 1000, 1000) // Reasonable bounds
			
			// Hub should be on the left
			hubNode := result[0]
			for i := 1; i < len(result); i++ {
				if result[i].X <= hubNode.X {
					t.Errorf("Spoke %d not to the right of hub", i)
				}
			}
		}
	})
	
	t.Run("Complete graph", func(t *testing.T) {
		// Complete graphs are challenging - every node connects to every other
		for _, nodeCount := range []int{3, 5, 10} {
			nodes, connections := GenerateCompleteGraph(nodeCount)
			
			result, err := layout.Layout(nodes, connections)
			if err != nil {
				t.Fatalf("Failed on complete graph with %d nodes: %v", nodeCount, err)
			}
			
			validator.ValidateNoOverlaps(result)
			validator.ValidateNodeSizes(result)
		}
	})
	
	t.Run("Wide tree", func(t *testing.T) {
		// Tree with many children per node
		nodes, connections := GenerateTree(2, 20) // depth 2, 20 children per node
		
		result, err := layout.Layout(nodes, connections)
		if err != nil {
			t.Fatalf("Failed on wide tree: %v", err)
		}
		
		validator.ValidateNoOverlaps(result)
		validator.ValidateBounds(result, 2000, 2000)
	})
	
	t.Run("Deep tree", func(t *testing.T) {
		// Very deep but narrow tree
		nodes, connections := GenerateTree(20, 1) // depth 20, 1 child per node
		
		result, err := layout.Layout(nodes, connections)
		if err != nil {
			t.Fatalf("Failed on deep tree: %v", err)
		}
		
		validator.ValidateNoOverlaps(result)
		// Should create a long horizontal chain
	})
}

// TestSimpleLayout_EdgeCases tests unusual input conditions.
func TestSimpleLayout_EdgeCases(t *testing.T) {
	layout := NewSimpleLayout()
	validator := NewTestValidator(t)
	
	t.Run("Self loops", func(t *testing.T) {
		nodes := []core.Node{
			{ID: 0, Text: []string{"A"}},
			{ID: 1, Text: []string{"B"}},
		}
		connections := []core.Connection{
			{From: 0, To: 0}, // Self loop
			{From: 0, To: 1},
			{From: 1, To: 1}, // Another self loop
		}
		
		result, err := layout.Layout(nodes, connections)
		if err != nil {
			t.Fatalf("Failed with self loops: %v", err)
		}
		
		validator.ValidateNoOverlaps(result)
	})
	
	t.Run("Duplicate connections", func(t *testing.T) {
		nodes := []core.Node{
			{ID: 0, Text: []string{"A"}},
			{ID: 1, Text: []string{"B"}},
		}
		connections := []core.Connection{
			{From: 0, To: 1},
			{From: 0, To: 1}, // Duplicate
			{From: 0, To: 1}, // Another duplicate
		}
		
		result, err := layout.Layout(nodes, connections)
		if err != nil {
			t.Fatalf("Failed with duplicate connections: %v", err)
		}
		
		if len(result) != 2 {
			t.Errorf("Node count changed: expected 2, got %d", len(result))
		}
	})
	
	t.Run("Bidirectional connections", func(t *testing.T) {
		nodes := []core.Node{
			{ID: 0, Text: []string{"A"}},
			{ID: 1, Text: []string{"B"}},
			{ID: 2, Text: []string{"C"}},
		}
		connections := []core.Connection{
			{From: 0, To: 1},
			{From: 1, To: 0}, // Reverse connection
			{From: 1, To: 2},
			{From: 2, To: 1}, // Another reverse
		}
		
		result, err := layout.Layout(nodes, connections)
		if err != nil {
			t.Fatalf("Failed with bidirectional connections: %v", err)
		}
		
		validator.ValidateNoOverlaps(result)
	})
	
	t.Run("Invalid node references", func(t *testing.T) {
		nodes := []core.Node{
			{ID: 0, Text: []string{"A"}},
			{ID: 1, Text: []string{"B"}},
		}
		connections := []core.Connection{
			{From: 0, To: 1},
			{From: 0, To: 99}, // Non-existent node
			{From: 99, To: 1}, // Non-existent source
		}
		
		// Should handle gracefully
		_, err := layout.Layout(nodes, connections)
		if err == nil {
			t.Log("Layout handled invalid references gracefully")
		}
	})
	
	t.Run("Orphaned nodes", func(t *testing.T) {
		nodes := []core.Node{
			{ID: 0, Text: []string{"Connected A"}},
			{ID: 1, Text: []string{"Connected B"}},
			{ID: 2, Text: []string{"Orphan 1"}},
			{ID: 3, Text: []string{"Orphan 2"}},
		}
		connections := []core.Connection{
			{From: 0, To: 1},
		}
		
		result, err := layout.Layout(nodes, connections)
		if err != nil {
			t.Fatalf("Failed with orphaned nodes: %v", err)
		}
		
		if len(result) != 4 {
			t.Errorf("Lost nodes: expected 4, got %d", len(result))
		}
		
		validator.ValidateNoOverlaps(result)
	})
}

// TestSimpleLayout_Determinism ensures consistent results.
func TestSimpleLayout_Determinism(t *testing.T) {
	layout := NewSimpleLayout()
	validator := NewTestValidator(t)
	
	testCases := []struct {
		name string
		nodes []core.Node
		connections []core.Connection
	}{
		{
			name: "Simple chain",
			nodes: func() []core.Node {
				n, _ := GenerateLinearChain(10)
				return n
			}(),
			connections: func() []core.Connection {
				_, c := GenerateLinearChain(10)
				return c
			}(),
		},
		{
			name: "Complex tree",
			nodes: func() []core.Node {
				n, _ := GenerateTree(4, 3)
				return n
			}(),
			connections: func() []core.Connection {
				_, c := GenerateTree(4, 3)
				return c
			}(),
		},
		{
			name: "Random DAG",
			nodes: func() []core.Node {
				n, _ := GenerateRandomDAG(20, 0.3)
				return n
			}(),
			connections: func() []core.Connection {
				_, c := GenerateRandomDAG(20, 0.3)
				return c
			}(),
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator.ValidateDeterminism(layout, tc.nodes, tc.connections, 10)
		})
	}
}

// TestSimpleLayout_Performance ensures reasonable performance.
func TestSimpleLayout_Performance(t *testing.T) {
	layout := NewSimpleLayout()
	validator := NewTestValidator(t)
	
	testCases := []struct {
		name string
		nodes []core.Node
		connections []core.Connection
		maxDuration time.Duration
	}{
		{
			name: "Small graph (10 nodes)",
			nodes: func() []core.Node {
				n, _ := GenerateRandomDAG(10, 0.3)
				return n
			}(),
			connections: func() []core.Connection {
				_, c := GenerateRandomDAG(10, 0.3)
				return c
			}(),
			maxDuration: 10 * time.Millisecond,
		},
		{
			name: "Medium graph (100 nodes)",
			nodes: func() []core.Node {
				n, _ := GenerateRandomDAG(100, 0.1)
				return n
			}(),
			connections: func() []core.Connection {
				_, c := GenerateRandomDAG(100, 0.1)
				return c
			}(),
			maxDuration: 50 * time.Millisecond,
		},
		{
			name: "Large graph (1000 nodes)",
			nodes: func() []core.Node {
				n, _ := GenerateLinearChain(1000)
				return n
			}(),
			connections: func() []core.Connection {
				_, c := GenerateLinearChain(1000)
				return c
			}(),
			maxDuration: 100 * time.Millisecond,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator.ValidatePerformance(layout, tc.nodes, tc.connections, tc.maxDuration)
		})
	}
}

// TestSimpleLayout_LargeGraphPerformance ensures linear time complexity.
func TestSimpleLayout_LargeGraphPerformance(t *testing.T) {
	layout := NewSimpleLayout()
	
	// Test that doubling nodes doesn't quadruple time
	sizes := []int{1000, 2000, 4000}
	times := make([]time.Duration, len(sizes))
	
	for i, size := range sizes {
		nodes, connections := GenerateLinearChain(size)
		
		start := time.Now()
		_, err := layout.Layout(nodes, connections)
		times[i] = time.Since(start)
		
		if err != nil {
			t.Fatalf("Failed on size %d: %v", size, err)
		}
		
		t.Logf("Size %d: %v", size, times[i])
	}
	
	// Check that time growth is roughly linear, not quadratic
	// If quadratic: time[2] / time[1] ≈ 4
	// If linear: time[2] / time[1] ≈ 2
	ratio1 := float64(times[1]) / float64(times[0])
	ratio2 := float64(times[2]) / float64(times[1])
	
	t.Logf("Time ratios: %.2f, %.2f", ratio1, ratio2)
	
	if ratio1 > 3.0 || ratio2 > 3.0 {
		t.Errorf("Performance appears to be worse than O(n log n): ratios %.2f, %.2f", ratio1, ratio2)
	}
}

// TestSimpleLayout_PropertyBased uses random inputs to find edge cases.
func TestSimpleLayout_PropertyBased(t *testing.T) {
	layout := NewSimpleLayout()
	validator := NewTestValidator(t)
	
	// Run many random test cases
	for i := 0; i < 50; i++ {
		// Generate random graph
		nodeCount := 5 + (i % 20) // 5-24 nodes
		edgeProbability := 0.1 + float64(i%10)*0.05 // 0.1-0.55
		
		nodes, connections := GenerateRandomDAG(nodeCount, edgeProbability)
		
		result, err := layout.Layout(nodes, connections)
		if err != nil {
			t.Errorf("Failed on random graph %d: %v", i, err)
			continue
		}
		
		// Properties that must always hold
		validator.ValidateNoOverlaps(result)
		validator.ValidateNodeSizes(result)
		validator.ValidateSpacing(result, 1)
		validator.ValidateBounds(result, 5000, 5000)
		
		// Same number of nodes
		if len(result) != len(nodes) {
			t.Errorf("Node count changed: %d -> %d", len(nodes), len(result))
		}
		
		// All node IDs preserved
		resultIDs := make(map[int]bool)
		for _, node := range result {
			resultIDs[node.ID] = true
		}
		for _, node := range nodes {
			if !resultIDs[node.ID] {
				t.Errorf("Node %d missing from result", node.ID)
			}
		}
	}
}

// TestSimpleLayout_DisconnectedComponentSpacing verifies proper spacing between components.
func TestSimpleLayout_DisconnectedComponentSpacing(t *testing.T) {
	layout := NewSimpleLayout()
	validator := NewTestValidator(t)
	
	// Create 3 disconnected chains
	nodes := []core.Node{
		// Component 1
		{ID: 0, Text: []string{"A1"}},
		{ID: 1, Text: []string{"A2"}},
		// Component 2
		{ID: 2, Text: []string{"B1"}},
		{ID: 3, Text: []string{"B2"}},
		// Component 3
		{ID: 4, Text: []string{"C1"}},
		{ID: 5, Text: []string{"C2"}},
	}
	connections := []core.Connection{
		{From: 0, To: 1},
		{From: 2, To: 3},
		{From: 4, To: 5},
	}
	
	result, err := layout.Layout(nodes, connections)
	if err != nil {
		t.Fatalf("Failed to layout disconnected components: %v", err)
	}
	
	validator.ValidateNoOverlaps(result)
	
	// Check that components are properly separated
	// Component roots should have different X coordinates
	roots := []int{0, 2, 4}
	xCoords := make(map[int]bool)
	
	for _, rootID := range roots {
		for _, node := range result {
			if node.ID == rootID {
				if xCoords[node.X] {
					t.Errorf("Components overlap: multiple roots at X=%d", node.X)
				}
				xCoords[node.X] = true
				break
			}
		}
	}
	
	// Log positions for debugging
	for _, node := range result {
		t.Logf("Node %d at (%d, %d)", node.ID, node.X, node.Y)
	}
}

// TestSimpleLayout_VerticalDistribution tests handling of wide layers.
func TestSimpleLayout_VerticalDistribution(t *testing.T) {
	layout := NewSimpleLayout()
	
	// Create a star with many spokes
	nodes, connections := GenerateStarGraph(20)
	
	result, err := layout.Layout(nodes, connections)
	if err != nil {
		t.Fatalf("Failed to layout star graph: %v", err)
	}
	
	// Find the second layer (spokes)
	spokeNodes := make([]core.Node, 0)
	for _, node := range result {
		if node.ID != 0 {
			spokeNodes = append(spokeNodes, node)
		}
	}
	
	// Check that spokes are distributed better than just vertically
	totalHeight := 0
	for _, node := range spokeNodes {
		totalHeight += node.Height
	}
	totalHeight += (len(spokeNodes) - 1) * 2 // spacing
	
	// Get actual height span
	minY, maxY := spokeNodes[0].Y, spokeNodes[0].Y
	for _, node := range spokeNodes {
		if node.Y < minY {
			minY = node.Y
		}
		if node.Y + node.Height > maxY {
			maxY = node.Y + node.Height
		}
	}
	actualHeight := maxY - minY
	
	t.Logf("Total stacked height would be: %d", totalHeight)
	t.Logf("Actual height span: %d", actualHeight)
	
	// Check if nodes are distributed in multiple columns
	xPositions := make(map[int]int)
	for _, node := range spokeNodes {
		xPositions[node.X]++
	}
	
	t.Logf("Number of X positions used: %d", len(xPositions))
	
	// With 20 spokes and max 10 per column, we should use 2 columns
	if len(spokeNodes) > 10 && len(xPositions) < 2 {
		t.Errorf("Expected multiple columns for %d nodes, got %d columns", 
			len(spokeNodes), len(xPositions))
	}
	
	// Height should be significantly less than stacked height
	if actualHeight >= totalHeight * 3 / 4 {
		t.Errorf("Height not reduced enough: %d vs stacked %d", 
			actualHeight, totalHeight)
	}
}

// TestSimpleLayout_CompleteBipartiteGraph tests a pathological case.
func TestSimpleLayout_CompleteBipartiteGraph(t *testing.T) {
	layout := NewSimpleLayout()
	validator := NewTestValidator(t)
	
	// Create K(5,5) - every node in set A connects to every node in set B
	nodes := make([]core.Node, 10)
	connections := make([]core.Connection, 0)
	
	// Set A: nodes 0-4
	for i := 0; i < 5; i++ {
		nodes[i] = core.Node{ID: i, Text: []string{fmt.Sprintf("A%d", i)}}
	}
	
	// Set B: nodes 5-9
	for i := 5; i < 10; i++ {
		nodes[i] = core.Node{ID: i, Text: []string{fmt.Sprintf("B%d", i-5)}}
	}
	
	// Connect every A to every B
	for i := 0; i < 5; i++ {
		for j := 5; j < 10; j++ {
			connections = append(connections, core.Connection{From: i, To: j})
		}
	}
	
	result, err := layout.Layout(nodes, connections)
	if err != nil {
		t.Fatalf("Failed on complete bipartite graph: %v", err)
	}
	
	validator.ValidateNoOverlaps(result)
	validator.ValidateNodeSizes(result)
	
	// Should have exactly 2 layers
	xPositions := make(map[int][]int)
	for _, node := range result {
		xPositions[node.X] = append(xPositions[node.X], node.ID)
	}
	
	if len(xPositions) != 2 {
		t.Errorf("Expected 2 layers for bipartite graph, got %d", len(xPositions))
	}
}

// TestSimpleLayout_DeepDiamondPattern tests multiple connected diamonds.
func TestSimpleLayout_DeepDiamondPattern(t *testing.T) {
	layout := NewSimpleLayout()
	validator := NewTestValidator(t)
	
	// Create 3 connected diamonds
	nodes := []core.Node{
		// Diamond 1
		{ID: 0, Text: []string{"Start"}},
		{ID: 1, Text: []string{"D1-Left"}},
		{ID: 2, Text: []string{"D1-Right"}},
		{ID: 3, Text: []string{"D1-End"}},
		// Diamond 2
		{ID: 4, Text: []string{"D2-Left"}},
		{ID: 5, Text: []string{"D2-Right"}},
		{ID: 6, Text: []string{"D2-End"}},
		// Diamond 3
		{ID: 7, Text: []string{"D3-Left"}},
		{ID: 8, Text: []string{"D3-Right"}},
		{ID: 9, Text: []string{"End"}},
	}
	
	connections := []core.Connection{
		// Diamond 1
		{From: 0, To: 1}, {From: 0, To: 2},
		{From: 1, To: 3}, {From: 2, To: 3},
		// Connect to Diamond 2
		{From: 3, To: 4}, {From: 3, To: 5},
		{From: 4, To: 6}, {From: 5, To: 6},
		// Connect to Diamond 3
		{From: 6, To: 7}, {From: 6, To: 8},
		{From: 7, To: 9}, {From: 8, To: 9},
	}
	
	result, err := layout.Layout(nodes, connections)
	if err != nil {
		t.Fatalf("Failed on deep diamond pattern: %v", err)
	}
	
	validator.ValidateNoOverlaps(result)
	validator.ValidateNodeSizes(result)
	
	// Should progress left to right
	for i := 1; i < len(result); i++ {
		if result[i].ID > result[i-1].ID {
			// Generally, later nodes should not be to the left
			if result[i].X < result[i-1].X - 20 {
				t.Logf("Node %d at X=%d, Node %d at X=%d", 
					result[i-1].ID, result[i-1].X,
					result[i].ID, result[i].X)
			}
		}
	}
}

// Benchmarks
func BenchmarkSimpleLayout_SmallGraph(b *testing.B) {
	nodes, connections := GenerateRandomDAG(10, 0.3)
	layout := NewSimpleLayout()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = layout.Layout(nodes, connections)
	}
}

func BenchmarkSimpleLayout_MediumGraph(b *testing.B) {
	nodes, connections := GenerateRandomDAG(100, 0.1)
	layout := NewSimpleLayout()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = layout.Layout(nodes, connections)
	}
}

func BenchmarkSimpleLayout_LargeGraph(b *testing.B) {
	nodes, connections := GenerateLinearChain(1000)
	layout := NewSimpleLayout()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = layout.Layout(nodes, connections)
	}
}
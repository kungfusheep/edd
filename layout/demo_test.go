package layout

import (
	"edd/core"
	"fmt"
	"testing"
)

func TestVisualDemo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping visual demo in short mode")
	}
	
	l := NewSimpleLayout()
	
	// Test 1: Simple chain
	fmt.Println("\n=== Test 1: Simple Chain ===")
	nodes, connections := GenerateLinearChain(5)
	result, _ := l.Layout(nodes, connections)
	fmt.Println(RenderLayout(result, connections))
	
	// Test 2: Disconnected components
	fmt.Println("\n=== Test 2: Disconnected Components ===")
	nodes = []core.Node{
		{ID: 0, Text: []string{"Comp1-A"}},
		{ID: 1, Text: []string{"Comp1-B"}},
		{ID: 2, Text: []string{"Comp2-A"}},
		{ID: 3, Text: []string{"Comp2-B"}},
		{ID: 4, Text: []string{"Isolated"}},
	}
	connections = []core.Connection{
		{From: 0, To: 1},
		{From: 2, To: 3},
	}
	result, _ = l.Layout(nodes, connections)
	fmt.Println(RenderLayout(result, connections))
	
	// Test 3: Star with many spokes (testing vertical distribution)
	fmt.Println("\n=== Test 3: Star with 15 Spokes ===")
	nodes, connections = GenerateStarGraph(15)
	result, _ = l.Layout(nodes, connections)
	fmt.Println(RenderLayout(result, connections))
	
	// Test 4: Complete bipartite graph
	fmt.Println("\n=== Test 4: Complete Bipartite K(3,3) ===")
	nodes = []core.Node{
		{ID: 0, Text: []string{"A1"}},
		{ID: 1, Text: []string{"A2"}},
		{ID: 2, Text: []string{"A3"}},
		{ID: 3, Text: []string{"B1"}},
		{ID: 4, Text: []string{"B2"}},
		{ID: 5, Text: []string{"B3"}},
	}
	connections = []core.Connection{}
	for i := 0; i < 3; i++ {
		for j := 3; j < 6; j++ {
			connections = append(connections, core.Connection{From: i, To: j})
		}
	}
	result, _ = l.Layout(nodes, connections)
	fmt.Println(RenderLayout(result, connections))
	
	// Test 5: Diamond pattern
	fmt.Println("\n=== Test 5: Diamond Pattern ===")
	nodes = []core.Node{
		{ID: 0, Text: []string{"Start"}},
		{ID: 1, Text: []string{"Path A"}},
		{ID: 2, Text: []string{"Path B"}},
		{ID: 3, Text: []string{"End"}},
	}
	connections = []core.Connection{
		{From: 0, To: 1},
		{From: 0, To: 2},
		{From: 1, To: 3},
		{From: 2, To: 3},
	}
	result, _ = l.Layout(nodes, connections)
	fmt.Println(RenderLayout(result, connections))
}
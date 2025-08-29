package editor

import (
	"fmt"
	"testing"
)

func TestAdjacentNodeConnection(t *testing.T) {
	tui := NewTUIEditor(NewRealRenderer())
	
	// Create nodes A, B, C, D, E, F
	tui.AddNode([]string{"A"})
	tui.AddNode([]string{"B"})
	tui.AddNode([]string{"C"})
	tui.AddNode([]string{"D"})
	tui.AddNode([]string{"E"})
	tui.AddNode([]string{"F"})
	
	// Start continuous connect mode
	tui.HandleKey('C')
	
	fmt.Printf("Jump labels assigned:\n")
	for nodeID, label := range tui.jumpLabels {
		fmt.Printf("  Node %d -> '%c'\n", nodeID, label)
	}
	
	// The labels should be: a, s, d, f, g, h
	// Let's verify what we actually get
	expectedLabels := map[int]rune{
		1: 'a', // A
		2: 's', // B
		3: 'd', // C
		4: 'f', // D
		5: 'g', // E
		6: 'h', // F
	}
	
	for nodeID, expected := range expectedLabels {
		if actual, ok := tui.jumpLabels[nodeID]; ok {
			if actual != expected {
				t.Errorf("Node %d: expected label '%c', got '%c'", nodeID, expected, actual)
			}
		} else {
			t.Errorf("Node %d: no label assigned", nodeID)
		}
	}
	
	// Now test connecting D to F
	fmt.Println("\nTrying to connect D -> F")
	
	// Select D (label 'f') as FROM
	fmt.Printf("Pressing 'f' to select node D (id=4)\n")
	tui.HandleKey('f')
	
	fmt.Printf("After selecting D: selected=%d, jumpAction=%v\n", tui.selected, tui.jumpAction)
	
	// Labels might be reassigned after selecting FROM, let's check
	fmt.Printf("\nJump labels after selecting FROM:\n")
	for nodeID, label := range tui.jumpLabels {
		fmt.Printf("  Node %d -> '%c'\n", nodeID, label)
	}
	
	// Try to select F
	// PROBLEM: F might have label 'h' but when we press 'f' it selects D again!
	// This is the issue - 'f' is D's label
	
	// Let me try selecting h for F
	fmt.Printf("\nPressing 'h' to select node F (id=6)\n")
	tui.HandleKey('h')
	
	fmt.Printf("After attempting F: selected=%d, jumpAction=%v\n", tui.selected, tui.jumpAction)
	
	// Check if connection was made
	fmt.Printf("\nConnections created: %d\n", len(tui.diagram.Connections))
	for i, conn := range tui.diagram.Connections {
		fmt.Printf("  Connection %d: %d -> %d\n", i, conn.From, conn.To)
	}
}

func TestLabelConflict(t *testing.T) {
	tui := NewTUIEditor(NewRealRenderer())
	
	// Create many nodes to see label assignment pattern
	for i := 1; i <= 10; i++ {
		tui.AddNode([]string{fmt.Sprintf("Node%d", i)})
	}
	
	// Start continuous connect mode
	tui.HandleKey('C')
	
	fmt.Printf("Jump labels for 10 nodes:\n")
	for i := 1; i <= 10; i++ {
		if label, ok := tui.jumpLabels[i]; ok {
			fmt.Printf("  Node %d -> '%c'\n", i, label)
		}
	}
	
	// Check if we have conflicts
	labelToNode := make(map[rune]int)
	for nodeID, label := range tui.jumpLabels {
		if existing, exists := labelToNode[label]; exists {
			t.Errorf("Label conflict: '%c' assigned to both node %d and node %d", label, existing, nodeID)
		}
		labelToNode[label] = nodeID
	}
}
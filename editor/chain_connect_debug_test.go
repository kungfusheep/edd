package editor

import (
	"fmt"
	"testing"
)

func TestContinuousConnectDebug(t *testing.T) {
	tui := NewTUIEditor(NewRealRenderer())
	
	// Set up sequence diagram with 3 nodes
	tui.diagram.Type = "sequence"
	tui.AddNode([]string{"A"})
	tui.AddNode([]string{"B"})
	tui.AddNode([]string{"C"})
	
	fmt.Printf("Initial state: mode=%v, jumpAction=%v\n", tui.mode, tui.jumpAction)
	
	// Press 'C' to start continuous connect mode
	tui.HandleKey('C')
	fmt.Printf("After 'C': mode=%v, jumpAction=%v, jumpLabels=%v\n", tui.mode, tui.jumpAction, tui.jumpLabels)
	
	// Check jump labels
	for nodeID, label := range tui.jumpLabels {
		fmt.Printf("  Node %d has label '%c'\n", nodeID, label)
	}
	
	// Select first node
	if label, ok := tui.jumpLabels[1]; ok {
		fmt.Printf("Pressing '%c' to select node 1\n", label)
		tui.HandleKey(label)
		fmt.Printf("After selecting FROM: mode=%v, jumpAction=%v, selected=%d\n", tui.mode, tui.jumpAction, tui.selected)
	}
	
	// Select second node
	if label, ok := tui.jumpLabels[2]; ok {
		fmt.Printf("Pressing '%c' to select node 2\n", label)
		tui.HandleKey(label)
		fmt.Printf("After selecting TO: mode=%v, jumpAction=%v, selected=%d, continuousConnect=%v\n", 
			tui.mode, tui.jumpAction, tui.selected, tui.continuousConnect)
	}
	
	// Check connections
	fmt.Printf("Connections created: %d\n", len(tui.diagram.Connections))
	for i, conn := range tui.diagram.Connections {
		fmt.Printf("  Connection %d: %d -> %d\n", i, conn.From, conn.To)
	}
	
	// Try to select third node
	if label, ok := tui.jumpLabels[3]; ok {
		fmt.Printf("Pressing '%c' to select node 3\n", label)
		tui.HandleKey(label)
		fmt.Printf("After second TO: mode=%v, jumpAction=%v, selected=%d\n", tui.mode, tui.jumpAction, tui.selected)
	}
	
	// Check final connections
	fmt.Printf("Final connections: %d\n", len(tui.diagram.Connections))
	for i, conn := range tui.diagram.Connections {
		fmt.Printf("  Connection %d: %d -> %d\n", i, conn.From, conn.To)
	}
}
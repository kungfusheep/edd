package editor

import (
	"fmt"
	"testing"
)

func TestSelectedStateAcrossSessions(t *testing.T) {
	tui := NewTUIEditor(NewRealRenderer())
	
	// Create nodes A, B, C, D
	tui.AddNode([]string{"A"})
	tui.AddNode([]string{"B"})
	tui.AddNode([]string{"C"})
	tui.AddNode([]string{"D"})
	
	fmt.Println("=== Initial state ===")
	fmt.Printf("Selected: %d\n", tui.selected)
	
	// Start continuous connect mode
	tui.HandleKey('C')
	fmt.Printf("After C: selected=%d\n", tui.selected)
	
	// Make A -> B
	tui.HandleKey('a')
	fmt.Printf("After selecting A: selected=%d\n", tui.selected)
	tui.HandleKey('s')
	fmt.Printf("After selecting B (creates A->B): selected=%d\n", tui.selected)
	
	// Now in continuous mode, B is selected as next FROM
	// Make B -> C
	tui.HandleKey('d')
	fmt.Printf("After selecting C (creates B->C): selected=%d\n", tui.selected)
	
	// Exit
	tui.HandleKey(27)
	fmt.Printf("After ESC: selected=%d\n", tui.selected)
	
	// Start connect mode again - is selected still set?
	fmt.Println("\n=== Restart connect mode ===")
	tui.HandleKey('C')
	fmt.Printf("After second C: selected=%d, jumpAction=%v\n", tui.selected, tui.jumpAction)
	
	// If selected is still set, we might be in ConnectTo mode instead of ConnectFrom!
	if tui.selected >= 0 && tui.jumpAction == JumpActionConnectFrom {
		t.Errorf("Starting connect mode with node %d already selected but in ConnectFrom mode!", tui.selected)
	}
	
	// Try to make a connection - what happens?
	fmt.Println("\nTrying to press 'd' (should select D):")
	tui.HandleKey('d')
	fmt.Printf("After 'd': selected=%d, jumpAction=%v\n", tui.selected, tui.jumpAction)
	
	// Print connections
	fmt.Printf("\nConnections: %d\n", len(tui.diagram.Connections))
	for i, conn := range tui.diagram.Connections {
		fmt.Printf("  %d: %d -> %d\n", i, conn.From, conn.To)
	}
}

func TestClearSelectedOnExit(t *testing.T) {
	tui := NewTUIEditor(NewRealRenderer())
	
	// Create nodes
	tui.AddNode([]string{"A"})
	tui.AddNode([]string{"B"})
	
	// Manually set selected to simulate state
	tui.selected = 1
	
	// Start connect mode with a node already selected
	tui.HandleKey('C')
	
	fmt.Printf("Starting C with selected=%d\n", tui.selected)
	fmt.Printf("JumpAction: %v (should be %v for ConnectFrom)\n", tui.jumpAction, JumpActionConnectFrom)
	
	// The issue might be here - if selected is already set when starting connect mode,
	// we should probably clear it or handle it differently
}
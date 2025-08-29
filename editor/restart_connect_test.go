package editor

import (
	"fmt"
	"testing"
)

func TestRestartConnectMode(t *testing.T) {
	tui := NewTUIEditor(NewRealRenderer())
	
	// Create nodes A, B, C, D
	tui.AddNode([]string{"A"})
	tui.AddNode([]string{"B"})
	tui.AddNode([]string{"C"})
	tui.AddNode([]string{"D"})
	
	fmt.Println("=== FIRST CONNECT SESSION ===")
	
	// Start continuous connect mode
	tui.HandleKey('C')
	
	fmt.Println("Jump labels in first session:")
	for nodeID, label := range tui.jumpLabels {
		fmt.Printf("  Node %d -> '%c'\n", nodeID, label)
	}
	
	// Make connection A -> B
	tui.HandleKey('a') // Select A as FROM
	tui.HandleKey('s') // Select B as TO
	
	fmt.Printf("Created connection: %d -> %d\n", 
		tui.diagram.Connections[0].From, tui.diagram.Connections[0].To)
	
	// Exit continuous mode
	tui.HandleKey(27) // ESC
	
	fmt.Printf("Mode after ESC: %v\n", tui.mode)
	fmt.Printf("Jump labels after ESC: %v\n", tui.jumpLabels)
	fmt.Printf("Selected after ESC: %d\n", tui.selected)
	fmt.Printf("ContinuousConnect after ESC: %v\n", tui.continuousConnect)
	
	fmt.Println("\n=== SECOND CONNECT SESSION ===")
	
	// Start continuous connect mode AGAIN
	tui.HandleKey('C')
	
	fmt.Printf("Mode after second C: %v\n", tui.mode)
	fmt.Printf("JumpAction after second C: %v\n", tui.jumpAction)
	fmt.Printf("Selected after second C: %d\n", tui.selected)
	
	fmt.Println("Jump labels in second session:")
	for nodeID, label := range tui.jumpLabels {
		fmt.Printf("  Node %d -> '%c'\n", nodeID, label)
	}
	
	// Try to make connection B -> C (adjacent nodes)
	fmt.Println("\nTrying to connect B -> C:")
	fmt.Println("Pressing 's' to select B as FROM")
	tui.HandleKey('s') // Select B as FROM
	
	fmt.Printf("After selecting B: selected=%d, jumpAction=%v\n", tui.selected, tui.jumpAction)
	
	fmt.Println("Jump labels after selecting B:")
	for nodeID, label := range tui.jumpLabels {
		fmt.Printf("  Node %d -> '%c'\n", nodeID, label)
	}
	
	fmt.Println("Pressing 'd' to select C as TO")
	tui.HandleKey('d') // Select C as TO
	
	fmt.Printf("After selecting C: selected=%d, jumpAction=%v\n", tui.selected, tui.jumpAction)
	
	// Check if connection was made
	fmt.Printf("\nTotal connections: %d\n", len(tui.diagram.Connections))
	for i, conn := range tui.diagram.Connections {
		fmt.Printf("  Connection %d: %d -> %d\n", i, conn.From, conn.To)
	}
	
	// Verify B->C connection exists
	hasBtoC := false
	for _, conn := range tui.diagram.Connections {
		if conn.From == 2 && conn.To == 3 {
			hasBtoC = true
			break
		}
	}
	
	if !hasBtoC {
		t.Error("Failed to create B -> C connection in second session")
	}
}

func TestRestartAfterPartialConnection(t *testing.T) {
	tui := NewTUIEditor(NewRealRenderer())
	
	// Create nodes
	tui.AddNode([]string{"A"})
	tui.AddNode([]string{"B"})
	tui.AddNode([]string{"C"})
	
	fmt.Println("=== TEST: Exit after selecting FROM ===")
	
	// Start connect mode
	tui.HandleKey('C')
	
	// Select FROM but don't select TO
	tui.HandleKey('a') // Select A as FROM
	
	fmt.Printf("Selected after FROM: %d\n", tui.selected)
	fmt.Printf("JumpAction after FROM: %v\n", tui.jumpAction)
	
	// Exit without completing connection
	tui.HandleKey(27) // ESC
	
	fmt.Printf("Selected after ESC: %d\n", tui.selected)
	fmt.Printf("Mode after ESC: %v\n", tui.mode)
	
	// Start connect mode again
	tui.HandleKey('C')
	
	fmt.Printf("Selected after restart: %d\n", tui.selected)
	fmt.Printf("JumpAction after restart: %v\n", tui.jumpAction)
	
	// Try to make a connection
	tui.HandleKey('a') // Select A as FROM
	tui.HandleKey('s') // Select B as TO
	
	if len(tui.diagram.Connections) != 1 {
		t.Errorf("Expected 1 connection after restart, got %d", len(tui.diagram.Connections))
	}
}
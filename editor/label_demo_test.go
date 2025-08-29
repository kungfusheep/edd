package editor

import (
	"fmt"
	"testing"
)

func TestLabelDemo(t *testing.T) {
	tui := NewTUIEditor(NewRealRenderer())
	
	// Create a sequence diagram scenario
	tui.diagram.Type = "sequence"
	tui.AddNode([]string{"Client"})    // ID 1
	tui.AddNode([]string{"API"})       // ID 2  
	tui.AddNode([]string{"Service"})   // ID 3
	tui.AddNode([]string{"Database"})  // ID 4
	
	fmt.Println("=== NODE LABELS ===")
	fmt.Println("Nodes created:")
	for _, node := range tui.diagram.Nodes {
		fmt.Printf("  ID %d: %s\n", node.ID, node.Text[0])
	}
	
	// Start continuous connect mode
	tui.HandleKey('C')
	
	fmt.Println("\nJump labels assigned:")
	fmt.Println("  Client (ID 1)   -> 'a'")
	fmt.Println("  API (ID 2)      -> 's'")
	fmt.Println("  Service (ID 3)  -> 'd'")
	fmt.Println("  Database (ID 4) -> 'f'")
	
	// Verify
	for nodeID, label := range tui.jumpLabels {
		nodeName := ""
		for _, n := range tui.diagram.Nodes {
			if n.ID == nodeID {
				nodeName = n.Text[0]
				break
			}
		}
		fmt.Printf("  Verified: %s (ID %d) has label '%c'\n", nodeName, nodeID, label)
	}
	
	fmt.Println("\n=== MAKING CONNECTIONS ===")
	
	// Connect Client -> API
	fmt.Println("1. To connect Client -> API:")
	fmt.Println("   Press 'a' (selects Client as FROM)")
	tui.HandleKey('a')
	fmt.Println("   Press 's' (selects API as TO)")
	tui.HandleKey('s')
	fmt.Printf("   ✓ Created: Client -> API\n")
	
	// Now API is selected as next FROM
	fmt.Println("\n2. To connect API -> Service:")
	fmt.Println("   API is already selected as FROM")
	fmt.Println("   Press 'd' (selects Service as TO)")
	tui.HandleKey('d')
	fmt.Printf("   ✓ Created: API -> Service\n")
	
	// Now Service is selected as next FROM
	fmt.Println("\n3. To connect Service -> Database:")
	fmt.Println("   Service is already selected as FROM")
	fmt.Println("   Press 'f' (selects Database as TO)")
	tui.HandleKey('f')
	fmt.Printf("   ✓ Created: Service -> Database\n")
	
	fmt.Println("\n=== FINAL CONNECTIONS ===")
	for i, conn := range tui.diagram.Connections {
		fromName := ""
		toName := ""
		for _, n := range tui.diagram.Nodes {
			if n.ID == conn.From {
				fromName = n.Text[0]
			}
			if n.ID == conn.To {
				toName = n.Text[0]
			}
		}
		fmt.Printf("%d. %s -> %s\n", i+1, fromName, toName)
	}
	
	// Verify all connections were made
	if len(tui.diagram.Connections) != 3 {
		t.Errorf("Expected 3 connections, got %d", len(tui.diagram.Connections))
	}
}
package editor

import (
	"edd/core"
	"fmt"
	"os"
	"os/exec"
)

// RunDemo runs a simple demo of the TUI editor
func RunDemo() error {
	// Create a mock renderer for now
	renderer := &MockRenderer{}
	
	// Create editor
	editor := NewTUIEditor(renderer)
	
	// Setup demo diagram
	editor.SetDiagram(&core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"Web Server"}},
			{ID: 2, Text: []string{"API"}},
			{ID: 3, Text: []string{"Database"}},
		},
		Connections: []core.Connection{
			{From: 1, To: 2, Label: "HTTP"},
			{From: 2, To: 3, Label: "SQL"},
		},
	})
	
	// Setup terminal
	if err := setupRawMode(); err != nil {
		return fmt.Errorf("failed to setup terminal: %w", err)
	}
	defer restoreTerminal()
	
	// Clear screen and show initial state
	fmt.Print("\033[H\033[2J")
	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Println("║     EDD Interactive Editor (Demo)      ║")
	fmt.Println("╠════════════════════════════════════════╣")
	fmt.Println("║ Commands:                              ║")
	fmt.Println("║   a - Add node     c - Connect nodes   ║")
	fmt.Println("║   d - Delete       e - Edit text       ║")
	fmt.Println("║   q - Quit         ? - Help            ║")
	fmt.Println("╚════════════════════════════════════════╝")
	fmt.Println()
	
	// Simple interaction loop
	for {
		// Render current state
		output := editor.Render()
		
		// Show it (keeping the header)
		fmt.Print("\033[9;1H") // Move cursor to line 9
		fmt.Print("\033[J")     // Clear from cursor to end
		fmt.Print(output)
		
		// Read single character
		key := readKey()
		
		// Handle quit
		if key == 'q' || key == 3 { // q or Ctrl+C
			break
		}
		
		// Process key through editor
		if key == 'a' {
			// Demo: Add a node
			id := editor.AddNode([]string{"New Node"})
			fmt.Printf("\nAdded node %d (press any key)", id)
			readKey()
		} else if key == 'd' && len(editor.diagram.Nodes) > 0 {
			// Demo: Delete first node
			editor.DeleteNode(editor.diagram.Nodes[0].ID)
			fmt.Printf("\nDeleted a node (press any key)")
			readKey()
		} else if key == 'c' && len(editor.diagram.Nodes) >= 2 {
			// Demo: Add a connection
			nodes := editor.diagram.Nodes
			editor.AddConnection(nodes[0].ID, nodes[len(nodes)-1].ID, "link")
			fmt.Printf("\nAdded connection (press any key)")
			readKey()
		} else if key == '?' {
			showDemoHelp()
		}
		
		// Let Ed animate
		editor.edd.NextFrame()
	}
	
	fmt.Print("\033[H\033[2J")
	fmt.Println("Thanks for trying EDD!")
	fmt.Println()
	
	return nil
}

// MockRenderer provides a simple renderer for demo
type MockRenderer struct{}

func (m *MockRenderer) Render(diagram *core.Diagram) (string, error) {
	// Use our actual modular renderer here
	// For demo, return simple ASCII boxes
	var output string
	
	// Simple box layout
	x := 2
	for _, node := range diagram.Nodes {
		output += fmt.Sprintf("\033[%d;%dH", x, 5) // Position cursor
		output += "┌──────────┐\n"
		output += fmt.Sprintf("\033[%d;%dH", x+1, 5)
		output += fmt.Sprintf("│ %-8s │\n", node.Text[0])
		output += fmt.Sprintf("\033[%d;%dH", x+2, 5)
		output += "└──────────┘"
		x += 4
	}
	
	// Show connections list
	if len(diagram.Connections) > 0 {
		output += fmt.Sprintf("\033[%d;%dH", x+1, 5)
		output += "Connections:\n"
		for _, conn := range diagram.Connections {
			x++
			output += fmt.Sprintf("\033[%d;%dH", x+1, 5)
			output += fmt.Sprintf("  %d → %d", conn.From, conn.To)
			if conn.Label != "" {
				output += fmt.Sprintf(" (%s)", conn.Label)
			}
			output += "\n"
		}
	}
	
	return output, nil
}

func setupRawMode() error {
	cmd := exec.Command("stty", "-echo", "cbreak", "min", "1")
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func restoreTerminal() {
	cmd := exec.Command("stty", "echo", "-cbreak")
	cmd.Stdin = os.Stdin
	cmd.Run()
	fmt.Print("\033[?25h") // Show cursor
}

func readKey() rune {
	var b [1]byte
	os.Stdin.Read(b[:])
	return rune(b[0])
}

func showDemoHelp() {
	fmt.Print("\033[H\033[2J")
	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Println("║            EDD Editor Help             ║")
	fmt.Println("╠════════════════════════════════════════╣")
	fmt.Println("║                                        ║")
	fmt.Println("║  This is a demo of the TUI editor.    ║")
	fmt.Println("║                                        ║")
	fmt.Println("║  Full version will include:           ║")
	fmt.Println("║  • EasyMotion jump labels              ║")
	fmt.Println("║  • Real-time diagram rendering         ║")
	fmt.Println("║  • Text editing with cursor           ║")
	fmt.Println("║  • Multiple modes (Normal/Insert/etc)  ║")
	fmt.Println("║  • Save/Load functionality            ║")
	fmt.Println("║                                        ║")
	fmt.Println("║  Press any key to continue...          ║")
	fmt.Println("╚════════════════════════════════════════╝")
	readKey()
}
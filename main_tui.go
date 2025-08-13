package main

import (
	"bytes"
	"edd/core"
	"edd/editor"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// RunInteractive launches the TUI editor
func RunInteractive(filename string) error {
	// Create the real renderer
	renderer := editor.NewRealRenderer()

	// Create TUI editor
	tui := editor.NewTUIEditor(renderer)

	// Load diagram if filename provided
	if filename != "" {
		diagram, err := loadDiagramFile(filename)
		if err != nil {
			return fmt.Errorf("failed to load diagram: %w", err)
		}
		tui.SetDiagram(diagram)
	}

	// Setup terminal
	if err := setupTerminal(); err != nil {
		return fmt.Errorf("failed to setup terminal: %w", err)
	}
	defer restoreTerminal()

	// Run the interactive loop
	return runInteractiveLoop(tui, filename)
}

func loadDiagramFile(filename string) (*core.Diagram, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var diagram core.Diagram
	if err := json.Unmarshal(data, &diagram); err != nil {
		return nil, err
	}

	return &diagram, nil
}

func saveDiagramFile(filename string, diagram *core.Diagram) error {
	data, err := json.MarshalIndent(diagram, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, data, 0644)
}

func setupTerminal() error {
	// Put terminal in raw mode
	cmd := exec.Command("stty", "-echo", "cbreak", "min", "1")
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func restoreTerminal() {
	// Restore terminal settings
	cmd := exec.Command("stty", "echo", "-cbreak")
	cmd.Stdin = os.Stdin
	cmd.Run()
	fmt.Print("\033[?25h") // Show cursor
}

func runInteractiveLoop(tui *editor.TUIEditor, filename string) error {
	// Switch to alternate screen buffer and hide cursor
	fmt.Print("\033[?1049h") // Enter alternate screen
	fmt.Print("\033[2J\033[H\033[?25l") // Clear and hide cursor
	
	// Ensure we restore on exit
	defer fmt.Print("\033[?1049l") // Exit alternate screen

	// Get terminal size
	width, height := getTerminalSize()
	tui.SetTerminalSize(width, height)

	// Create a channel for keyboard input
	keyChan := make(chan rune)
	go func() {
		for {
			keyChan <- readSingleKey()
		}
	}()

	// Animation ticker - update Ed every 200ms
	animTicker := time.NewTicker(200 * time.Millisecond)
	defer animTicker.Stop()

	// Main render loop
	var buf bytes.Buffer
	
	for {
		// Buffer all output to reduce flicker
		buf.Reset()
		
		// Clear and move to home - this prevents artifacts
		buf.WriteString("\033[H\033[2J")
		
		// Render current state
		output := tui.Render()
		buf.WriteString(output)
		
		// Write main content first
		fmt.Print(buf.String())
		
		// Now draw overlays directly (these use absolute positioning)
		// Draw jump labels if in jump mode
		if tui.GetMode() == editor.ModeJump {
			drawJumpLabels(tui, output)
			// Also draw connection labels if in delete or edit mode
			if tui.GetJumpAction() == editor.JumpActionDelete || tui.GetJumpAction() == editor.JumpActionEdit {
				drawConnectionLabels(tui)
			}
		}

		// Show status line
		showStatusLine(tui, filename)

		// Draw Ed
		drawEd(tui)

		// Position cursor if in edit mode
		if tui.GetMode() == editor.ModeEdit || tui.GetMode() == editor.ModeInsert {
			positionCursor(tui)
		} else {
			// Hide cursor when not editing
			fmt.Print("\033[?25l")
		}

		// Handle input or animation
		select {
		case key := <-keyChan:
			// Handle key
			switch tui.GetMode() {
			case editor.ModeNormal:
				if handleNormalMode(tui, key, &filename) {
					return nil // Exit requested
				}
			case editor.ModeInsert, editor.ModeEdit:
				handleTextMode(tui, key)
			case editor.ModeJump:
				handleJumpMode(tui, key)
			case editor.ModeCommand:
				if handleCommandMode(tui, key, &filename) {
					return nil // Exit requested
				}
			}

		case <-animTicker.C:
			// Animate Ed
			tui.AnimateEd()
		}
	}
}

func readSingleKey() rune {
	var b [1]byte
	os.Stdin.Read(b[:])
	return rune(b[0])
}

func positionCursor(tui *editor.TUIEditor) {
	// Get the node being edited
	selectedNode := tui.GetSelectedNode()
	if selectedNode < 0 {
		return
	}

	// Get node positions
	positions := tui.GetNodePositions()
	pos, ok := positions[selectedNode]
	if !ok {
		return
	}

	// Get cursor position in text
	state := tui.GetState()
	cursorPos := state.CursorPos

	// Calculate actual cursor position on screen
	// Node text starts at X+2, Y+1 (inside the box)
	cursorX := pos.X + 2 + cursorPos + 1 // +1 for terminal indexing
	cursorY := pos.Y + 1 + 1             // +1 for terminal indexing

	// Move cursor to position and show it
	fmt.Printf("\033[%d;%dH", cursorY, cursorX)
	fmt.Print("\033[?25h") // Show cursor
}

func drawJumpLabels(tui *editor.TUIEditor, output string) {
	// Get jump labels and state from TUI
	labels := tui.GetJumpLabels()
	if len(labels) == 0 {
		return
	}

	nodePositions := tui.GetNodePositions()
	selectedNode := tui.GetSelectedNode()
	jumpAction := tui.GetJumpAction()

	// Save cursor once
	fmt.Print("\033[s")

	// Draw labels using known positions
	for nodeID, label := range labels {
		if pos, ok := nodePositions[nodeID]; ok {
			// Position at top-left corner of node (offset by 1 for terminal indexing)
			// Add 1 to position to draw label inside the box corner
			fmt.Printf("\033[%d;%dH", pos.Y+1, pos.X+2)

			// Determine what to display
			if jumpAction == editor.JumpActionConnectTo && nodeID == selectedNode {
				// This is the "FROM" node in connection mode
				fmt.Printf("\033[32;1mFROM\033[0m") // Green "FROM"
			} else {
				// Regular jump label - single character in yellow
				fmt.Printf("\033[33;1m%c\033[0m", label) // Yellow label
			}
		}
	}

	// Restore cursor
	fmt.Print("\033[u")
}

func drawConnectionLabels(tui *editor.TUIEditor) {
	// Get connection labels from TUI
	labels := tui.GetConnectionLabels()
	if len(labels) == 0 {
		return
	}

	connectionPaths := tui.GetConnectionPaths()
	diagram := tui.GetDiagram()
	
	// Save cursor once
	fmt.Print("\033[s")
	
	// First, draw simple labels on the connections themselves
	// Track occupied positions to avoid overlaps
	occupiedPositions := make(map[string]bool)
	
	for connIndex := 0; connIndex < len(diagram.Connections); connIndex++ {
		if label, hasLabel := labels[connIndex]; hasLabel {
			if path, ok := connectionPaths[connIndex]; ok && len(path.Points) > 1 {
				// Place label at different percentages for each connection
				percentages := []float64{0.25, 0.40, 0.55, 0.70, 0.85}
				percentage := percentages[connIndex % len(percentages)]
				
				labelIndex := int(float64(len(path.Points)) * percentage)
				if labelIndex < 1 {
					labelIndex = 1
				}
				if labelIndex >= len(path.Points) {
					labelIndex = len(path.Points) - 1
				}
				
				labelPoint := path.Points[labelIndex]
				
				// Try to find a clear spot near this point
				offsets := []struct{ dx, dy int }{
					{0, 0},   // On the line
					{1, 0},   // Right
					{-1, 0},  // Left
					{0, -1},  // Above
					{0, 1},   // Below
				}
				
				var labelX, labelY int
				for _, offset := range offsets {
					testX := labelPoint.X + offset.dx
					testY := labelPoint.Y + offset.dy
					posKey := fmt.Sprintf("%d,%d", testX, testY)
					
					if !occupiedPositions[posKey] {
						labelX = testX
						labelY = testY
						occupiedPositions[posKey] = true
						occupiedPositions[fmt.Sprintf("%d,%d", testX+1, testY)] = true
						occupiedPositions[fmt.Sprintf("%d,%d", testX+2, testY)] = true
						break
					}
				}
				
				// If no position found, use the original point
				if labelX == 0 && labelY == 0 {
					labelX = labelPoint.X
					labelY = labelPoint.Y
				}
				
				// Draw simple label on the connection
				fmt.Printf("\033[%d;%dH", labelY+1, labelX+1)
				
				// Choose color based on action
				jumpAction := tui.GetJumpAction()
				if jumpAction == editor.JumpActionEdit {
					// Yellow background for edit mode
					fmt.Printf("\033[43;30;1m %c \033[0m", label) // Yellow bg, black text
				} else {
					// Red background for delete mode
					fmt.Printf("\033[41;97;1m %c \033[0m", label) // Red bg, white text
				}
			}
		}
	}
	
	// Now draw a legend at the bottom showing what each label means
	// Find the bottom of the screen (we'll put it above the status line)
	fmt.Print("\033[999;1H") // Go to bottom
	fmt.Print("\033[5A")      // Move up 5 lines from bottom
	fmt.Print("\033[K")       // Clear line
	
	// Draw connection legend header based on action
	jumpAction := tui.GetJumpAction()
	if jumpAction == editor.JumpActionDelete {
		fmt.Print("\033[91mDelete Connection:\033[0m ")
	} else if jumpAction == editor.JumpActionEdit {
		fmt.Print("\033[93mEdit Connection Label:\033[0m ")
	} else {
		fmt.Print("\033[91mConnection Labels:\033[0m ")
	}
	
	// Draw each connection label with its endpoints
	labelCount := 0
	for connIndex := 0; connIndex < len(diagram.Connections); connIndex++ {
		if label, hasLabel := labels[connIndex]; hasLabel {
			conn := diagram.Connections[connIndex]
			
			// Find node names
			var fromText, toText string
			for _, node := range diagram.Nodes {
				if node.ID == conn.From && len(node.Text) > 0 {
					fromText = node.Text[0]
					if len(fromText) > 6 {
						fromText = fromText[:6] // Shorter truncation for legend
					}
				}
				if node.ID == conn.To && len(node.Text) > 0 {
					toText = node.Text[0]
					if len(toText) > 6 {
						toText = toText[:6]
					}
				}
			}
			
			// Choose color based on action
			if jumpAction == editor.JumpActionEdit {
				// Yellow background for edit mode
				fmt.Printf("\033[43;30m%c\033[0m=%s→%s  ", label, fromText, toText)
			} else {
				// Red background for delete mode
				fmt.Printf("\033[41;97m%c\033[0m=%s→%s  ", label, fromText, toText)
			}
			
			labelCount++
			// Start a new line after every 3 entries to avoid running off screen
			if labelCount % 3 == 0 && connIndex < len(diagram.Connections)-1 {
				fmt.Print("\n                     ") // Indent continuation lines
			}
		}
	}
	
	// Restore cursor
	fmt.Print("\033[u")
}

func drawEd(tui *editor.TUIEditor) {
	// Draw Ed in bottom-right corner using ANSI positioning
	mode := tui.GetMode()
	frame := tui.GetEddFrame()

	// Determine color based on mode
	var color string
	switch mode {
	case editor.ModeNormal:
		color = "\033[36m" // Cyan
	case editor.ModeInsert:
		color = "\033[32m" // Green
	case editor.ModeEdit:
		color = "\033[33m" // Yellow
	case editor.ModeJump:
		color = "\033[35m" // Magenta
	case editor.ModeCommand:
		color = "\033[34m" // Blue
	default:
		color = "\033[37m" // White
	}
	reset := "\033[0m"

	// Save cursor position
	fmt.Print("\033[s")

	// Draw Ed's box - position above status line
	// We need to position Ed carefully to avoid the status line

	// Top of box (4 lines from bottom)
	fmt.Print("\033[999;999H") // Go to bottom-right
	fmt.Print("\033[4A")       // Move up 4 lines from bottom
	fmt.Print("\033[20D")      // Move left 20 chars from right edge
	fmt.Printf("%s╭────╮%s", color, reset)

	// Ed's face and mode (3 lines from bottom)
	fmt.Print("\033[999;999H") // Go to bottom-right again
	fmt.Print("\033[3A")       // Move up 3 lines from bottom
	fmt.Print("\033[20D")      // Move left 20 chars from right edge
	fmt.Printf("%s│%s│%s %s", color, frame, reset, mode)

	// Bottom of box (2 lines from bottom)
	fmt.Print("\033[999;999H") // Go to bottom-right again
	fmt.Print("\033[2A")       // Move up 2 lines from bottom
	fmt.Print("\033[20D")      // Move left 20 chars from right edge
	fmt.Printf("%s╰────╯%s", color, reset)

	// Restore cursor position
	fmt.Print("\033[u")
}

func showStatusLine(tui *editor.TUIEditor, filename string) {
	// Move to bottom of screen
	fmt.Print("\033[999;1H") // Move to bottom
	fmt.Print("\033[K")      // Clear line

	// Special handling for command mode - show the command being typed
	if tui.GetMode() == editor.ModeCommand {
		cmd := tui.GetCommand()
		fmt.Printf(":%s│", cmd) // Show command with cursor
		return
	}

	// Show filename and mode
	if filename != "" {
		fmt.Printf("[ %s ] ", filename)
	} else {
		fmt.Print("[ untitled ] ")
	}

	// Show node/connection count
	diagram := tui.GetDiagram()
	
	// Check if we're editing a connection
	if tui.GetMode() == editor.ModeEdit && tui.GetSelectedConnection() >= 0 {
		connIdx := tui.GetSelectedConnection()
		if connIdx < len(diagram.Connections) {
			conn := diagram.Connections[connIdx]
			// Find node names for clarity
			var fromName, toName string
			for _, node := range diagram.Nodes {
				if node.ID == conn.From && len(node.Text) > 0 {
					fromName = node.Text[0]
				}
				if node.ID == conn.To && len(node.Text) > 0 {
					toName = node.Text[0]
				}
			}
			fmt.Printf("Editing connection: %s → %s | Label: ", fromName, toName)
			// Show the current text being edited
			fmt.Print(string(tui.GetTextBuffer()))
			fmt.Print("│") // Show cursor
		}
	} else {
		fmt.Printf("Nodes: %d | Connections: %d | Mode: %s",
			len(diagram.Nodes),
			len(diagram.Connections),
			tui.GetMode())
	}
}

func handleNormalMode(tui *editor.TUIEditor, key rune, filename *string) bool {
	switch key {
	case 'q', 3: // q or Ctrl+C
		return true // Exit
	case 'a': // Add node
		tui.StartAddNode()
	case 'c': // Connect
		tui.StartConnect()
	case 'd': // Delete
		tui.StartDelete()
	case 'e': // Edit
		tui.StartEdit()
	case ':': // Command mode
		tui.StartCommand()
	case '?', 'h': // Help
		showHelp()
	}
	return false
}

func handleTextMode(tui *editor.TUIEditor, key rune) {
	tui.HandleTextInput(key)
}

func handleJumpMode(tui *editor.TUIEditor, key rune) {
	tui.HandleJumpInput(key)
}

func handleCommandMode(tui *editor.TUIEditor, key rune, filename *string) bool {
	if key == 13 || key == 10 { // Enter
		cmd := tui.GetCommand()

		// Parse command
		parts := strings.Fields(cmd)
		if len(parts) > 0 {
			switch parts[0] {
			case "w", "write", "save":
				if *filename == "" && len(parts) > 1 {
					*filename = parts[1]
				}
				if *filename != "" {
					if err := saveDiagramFile(*filename, tui.GetDiagram()); err != nil {
						fmt.Printf("\nError saving: %v", err)
					} else {
						fmt.Printf("\nSaved to %s", *filename)
					}
				}
			case "q", "quit":
				return true
			case "wq":
				if *filename != "" {
					saveDiagramFile(*filename, tui.GetDiagram())
				}
				return true
			}
		}

		tui.ClearCommand()
		tui.SetMode(editor.ModeNormal)
	} else {
		tui.HandleCommandInput(key)
	}

	return false
}

// Terminal size constants for Darwin/macOS
const TIOCGWINSZ = 0x40087468

// winsize struct for ioctl
type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

func getTerminalSize() (int, int) {
	// Use ioctl to get actual terminal size
	ws := &winsize{}
	retCode, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdout),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)))

	if int(retCode) == -1 {
		// Try stderr or stdin
		retCode, _, errno = syscall.Syscall(syscall.SYS_IOCTL,
			uintptr(syscall.Stderr),
			uintptr(syscall.TIOCGWINSZ),
			uintptr(unsafe.Pointer(ws)))

		if int(retCode) == -1 {
			retCode, _, errno = syscall.Syscall(syscall.SYS_IOCTL,
				uintptr(syscall.Stdin),
				uintptr(syscall.TIOCGWINSZ),
				uintptr(unsafe.Pointer(ws)))

			if int(retCode) == -1 {
				// Last resort: try environment variables
				if cols := os.Getenv("COLUMNS"); cols != "" {
					if lines := os.Getenv("LINES"); lines != "" {
						var c, l int
						fmt.Sscanf(cols, "%d", &c)
						fmt.Sscanf(lines, "%d", &l)
						if c > 0 && l > 0 {
							return c, l
						}
					}
				}
				// Fallback
				_ = errno // avoid unused variable warning
				return 80, 24
			}
		}
	}

	return int(ws.Col), int(ws.Row)
}

func showHelp() {
	fmt.Print("\033[2J\033[H") // Clear screen and move to home
	fmt.Println("EDD Interactive Editor")
	fmt.Println("═══════════════════════")
	fmt.Println()
	fmt.Println("Normal Mode Commands:")
	fmt.Println("  a     - Add new node")
	fmt.Println("  c     - Connect nodes (with jump labels)")
	fmt.Println("  d     - Delete node (with jump labels)")
	fmt.Println("  e     - Edit node text (with jump labels)")
	fmt.Println("  q     - Quit")
	fmt.Println("  :     - Command mode")
	fmt.Println()
	fmt.Println("Command Mode:")
	fmt.Println("  :w [file]  - Save diagram")
	fmt.Println("  :q         - Quit")
	fmt.Println("  :wq        - Save and quit")
	fmt.Println()
	fmt.Println("Text Editing:")
	fmt.Println("  ESC   - Exit to normal mode")
	fmt.Println("  Enter - Confirm text")
	fmt.Println()
	fmt.Println("Press any key to continue...")
	readSingleKey()

	// Clear screen completely after help is dismissed
	fmt.Print("\033[2J\033[H")
}


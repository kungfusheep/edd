package main

import (
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
	// Initial clear and hide cursor
	fmt.Print("\033[2J\033[H\033[?25l")
	
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
	for {
		// Move cursor to home and clear everything
		fmt.Print("\033[H")       // Move cursor to home position (0,0)
		fmt.Print("\033[2J")      // Clear entire screen
		
		// Render current state
		output := tui.Render()
		fmt.Print(output)
		
		// Draw jump labels if in jump mode
		if tui.GetMode() == editor.ModeJump {
			drawJumpLabels(tui, output)
		}
		
		// Show status line first
		showStatusLine(tui, filename)
		
		// Then draw Ed on top (so he doesn't get overwritten)
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
	cursorY := pos.Y + 1 + 1              // +1 for terminal indexing
	
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
	fmt.Print("\033[999;999H")      // Go to bottom-right
	fmt.Print("\033[4A")            // Move up 4 lines from bottom
	fmt.Print("\033[20D")           // Move left 20 chars from right edge
	fmt.Printf("%s╭─────╮%s", color, reset)
	
	// Ed's face and mode (3 lines from bottom)
	fmt.Print("\033[999;999H")      // Go to bottom-right again
	fmt.Print("\033[3A")            // Move up 3 lines from bottom
	fmt.Print("\033[20D")           // Move left 20 chars from right edge
	fmt.Printf("%s│%s│%s %s", color, frame, reset, mode)
	
	// Bottom of box (2 lines from bottom)
	fmt.Print("\033[999;999H")      // Go to bottom-right again
	fmt.Print("\033[2A")            // Move up 2 lines from bottom
	fmt.Print("\033[20D")           // Move left 20 chars from right edge
	fmt.Printf("%s╰─────╯%s", color, reset)
	
	// Restore cursor position
	fmt.Print("\033[u")
	
	// Force flush to ensure it renders
	os.Stdout.Sync()
}

func showStatusLine(tui *editor.TUIEditor, filename string) {
	// Move to bottom of screen
	fmt.Print("\033[999;1H") // Move to bottom
	fmt.Print("\033[K")       // Clear line
	
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
	fmt.Printf("Nodes: %d | Connections: %d | Mode: %s",
		len(diagram.Nodes),
		len(diagram.Connections),
		tui.GetMode())
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
package terminal

import (
	"bufio"
	"bytes"
	"edd/diagram"
	"edd/demo"
	"edd/editor"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// DemoSettings configures demo playback
type DemoSettings struct {
	MinDelay  int // Min delay between keystrokes in ms
	MaxDelay  int // Max delay between keystrokes in ms
	LineDelay int // Extra delay between lines in ms
}

// RunTUILoop runs the terminal UI loop with an already-configured TUI editor
// This is called from main.go after setting up the editor
func RunTUILoop(tui *editor.TUIEditor, filename string, demoSettings *DemoSettings) error {
	// Setup terminal
	if err := setupTerminal(); err != nil {
		return fmt.Errorf("failed to setup terminal: %w", err)
	}
	
	// Ensure terminal is restored even on panic
	defer func() {
		restoreTerminal()
		// Extra safety - ensure cursor is visible
		fmt.Print("\033[?25h")
	}()

	// Run the interactive loop
	return runInteractiveLoop(tui, filename, demoSettings)
}

func loadDiagramFile(filename string) (*diagram.Diagram, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var d diagram.Diagram
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}

	// Ensure all connections have unique IDs
	diagram.EnsureUniqueConnectionIDs(&d)

	return &d, nil
}

func saveDiagramFile(filename string, d *diagram.Diagram) error {
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, data, 0644)
}

// validateDiagram checks if a diagram has valid structure
func validateDiagram(d *diagram.Diagram) error {
	// Check for duplicate node IDs
	nodeIDs := make(map[int]bool)
	for _, node := range d.Nodes {
		if nodeIDs[node.ID] {
			return fmt.Errorf("duplicate node ID: %d", node.ID)
		}
		nodeIDs[node.ID] = true
	}

	// Check that connections reference valid nodes
	for i, conn := range d.Connections {
		if !nodeIDs[conn.From] {
			return fmt.Errorf("connection %d references non-existent 'from' node: %d", i, conn.From)
		}
		if !nodeIDs[conn.To] {
			return fmt.Errorf("connection %d references non-existent 'to' node: %d", i, conn.To)
		}
	}

	return nil
}

// launchExternalEditor opens the diagram in the user's $EDITOR
func launchExternalEditor(tui *editor.TUIEditor) error {
	// Get the editor from environment
	editorCmd := os.Getenv("EDITOR")
	if editorCmd == "" {
		editorCmd = os.Getenv("VISUAL")
	}
	if editorCmd == "" {
		// Try common defaults
		if _, err := exec.LookPath("vim"); err == nil {
			editorCmd = "vim"
		} else if _, err := exec.LookPath("nano"); err == nil {
			editorCmd = "nano"
		} else if _, err := exec.LookPath("vi"); err == nil {
			editorCmd = "vi"
		} else {
			return fmt.Errorf("no editor found. Please set $EDITOR environment variable")
		}
	}

	// Create a temporary file for editing
	tmpFile, err := ioutil.TempFile("", "edd-edit-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpFileName := tmpFile.Name()
	defer os.Remove(tmpFileName) // Clean up after we're done

	// Write current diagram to temp file
	d := tui.GetDiagram()
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to marshal diagram: %w", err)
	}

	// Write data and ensure it's flushed to disk
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	
	// Add newline at end of file (some editors expect this)
	tmpFile.Write([]byte("\n"))
	
	// Ensure all data is written to disk
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}
	tmpFile.Close()

	// Store original file stats to detect changes
	originalStat, err := os.Stat(tmpFileName)
	if err != nil {
		return fmt.Errorf("failed to stat temp file: %w", err)
	}

	// First, completely clean the terminal state
	// Exit alternate screen
	fmt.Print("\033[?1049l")
	
	// Clear screen and reset cursor
	fmt.Print("\033[2J")       // Clear entire screen
	fmt.Print("\033[H")        // Move cursor to home
	fmt.Print("\033[0m")       // Reset all attributes
	fmt.Print("\033[?25h")     // Show cursor
	
	// Restore terminal settings
	restoreTerminal()
	
	// Ensure everything is flushed
	os.Stdout.Sync()
	
	// Clear any pending input from stdin
	// This prevents vim from receiving stray characters
	clearStdinBuffer()
	
	// Small delay for terminal to process
	time.Sleep(50 * time.Millisecond)

	// Launch the editor
	cmd := exec.Command(editorCmd, tmpFileName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Re-setup terminal even if editor failed
		setupTerminal()
		// Re-enter alternate screen
		fmt.Print("\033[?1049h\033[2J\033[H\033[?25l")
		return fmt.Errorf("editor failed: %w", err)
	}

	// Check if file was actually modified
	newStat, err := os.Stat(tmpFileName)
	if err != nil {
		setupTerminal()
		fmt.Print("\033[?1049h\033[2J\033[H\033[?25l")
		return fmt.Errorf("failed to stat edited file: %w", err)
	}

	// Re-setup terminal after editor exits
	if err := setupTerminal(); err != nil {
		fmt.Print("\033[?1049h\033[2J\033[H\033[?25l")
		return fmt.Errorf("failed to re-setup terminal: %w", err)
	}

	// Re-enter alternate screen and clear
	fmt.Print("\033[?1049h\033[2J\033[H\033[?25l")

	// If file wasn't modified, just return
	if originalStat.ModTime().Equal(newStat.ModTime()) {
		// File wasn't changed, no need to parse
		return nil
	}

	// Read the edited file
	editedData, err := ioutil.ReadFile(tmpFileName)
	if err != nil {
		return fmt.Errorf("failed to read edited file: %w", err)
	}

	// Trim any trailing whitespace that might cause issues
	editedData = bytes.TrimSpace(editedData)
	
	// If the file is empty after trimming, user cleared it - ignore
	if len(editedData) == 0 {
		return nil
	}

	// Parse the edited JSON
	var editedDiagram diagram.Diagram
	if err := json.Unmarshal(editedData, &editedDiagram); err != nil {
		// Save the invalid JSON for debugging
		debugFile := filepath.Join(os.TempDir(), "edd-invalid.json")
		ioutil.WriteFile(debugFile, editedData, 0644)
		
		// Try to provide more helpful error message
		var syntaxErr *json.SyntaxError
		if errors.As(err, &syntaxErr) {
			// Calculate line and column
			lines := bytes.Split(editedData[:syntaxErr.Offset], []byte("\n"))
			line := len(lines)
			col := len(lines[line-1]) + 1
			return fmt.Errorf("JSON syntax error at line %d, column %d: %v (saved to %s)", 
				line, col, err, debugFile)
		}
		return fmt.Errorf("invalid JSON after editing: %w (saved to %s)", err, debugFile)
	}

	// Validate the diagram has valid structure
	if err := validateDiagram(&editedDiagram); err != nil {
		return fmt.Errorf("invalid diagram structure: %w", err)
	}

	// Update the diagram
	tui.SetDiagram(&editedDiagram)

	return nil
}

func setupTerminal() error {
	// Put terminal in raw mode
	// Use < /dev/tty for input redirection (more portable)
	cmd := exec.Command("sh", "-c", "stty -echo cbreak min 1 < /dev/tty")
	return cmd.Run()
}

func restoreTerminal() {
	// Restore terminal to cooked mode with echo
	cmd := exec.Command("sh", "-c", "stty echo cooked < /dev/tty")
	cmd.Run()
	
	// Also ensure cursor is visible and colors are reset
	fmt.Print("\033[?25h") // Show cursor
	fmt.Print("\033[0m")   // Reset all attributes
}

// clearStdinBuffer reads and discards any pending input
func clearStdinBuffer() {
	// Use tcflush to clear the input buffer more reliably
	// This is more effective than trying to read pending bytes
	cmd := exec.Command("stty", "-F", "/dev/tty", "sane")
	cmd.Run()
	
	// Alternative approach: read with timeout
	// Set non-blocking read with very short timeout
	cmd = exec.Command("stty", "-icanon", "min", "0", "time", "1")
	cmd.Stdin = os.Stdin
	cmd.Run()
	
	// Read and discard any pending bytes
	var discard [256]byte
	for i := 0; i < 3; i++ { // Try a few times
		n, _ := os.Stdin.Read(discard[:])
		if n == 0 {
			break
		}
	}
}

func runInteractiveLoop(tui *editor.TUIEditor, filename string, demoSettings *DemoSettings) error {
	// Switch to alternate screen buffer and hide cursor
	fmt.Print("\033[?1049h") // Enter alternate screen
	fmt.Print("\033[2J\033[H\033[?25l") // Clear and hide cursor
	
	// Ensure we restore on exit
	defer func() {
		fmt.Print("\033[?25h")   // Show cursor
		fmt.Print("\033[?1049l") // Exit alternate screen
		fmt.Print("\033[0m")      // Reset colors
	}()

	// Get terminal size
	width, height := getTerminalSize()
	tui.SetTerminalSize(width, height)

	// Create a channel for keyboard input with support for special keys
	keyChan := make(chan editor.KeyEvent)
	
	// Create demo channel for demo playback
	demoChan := make(chan editor.KeyEvent, 10)
	
	if demoSettings != nil {
		// Demo mode - read from stdin with randomized timing
		go func() {
			rand.Seed(time.Now().UnixNano())
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				line := scanner.Text()
				// Process each character with random delay
				for i, ch := range line {
					demoChan <- editor.KeyEvent{Rune: ch}
					if i < len(line)-1 {
						delay := demoSettings.MinDelay + rand.Intn(demoSettings.MaxDelay-demoSettings.MinDelay+1)
						time.Sleep(time.Duration(delay) * time.Millisecond)
					}
				}
				// Send newline and wait before next line
				demoChan <- editor.KeyEvent{Rune: '\n'}
				time.Sleep(time.Duration(demoSettings.LineDelay) * time.Millisecond)
			}
		}()
		
		// Also read from keyboard (/dev/tty) to allow ESC to stop demo
		tty, err := os.Open("/dev/tty")
		if err == nil {
			go func() {
				defer tty.Close()
				for {
					var b [1]byte
					n, _ := tty.Read(b[:])
					if n > 0 {
						keyChan <- editor.KeyEvent{Rune: rune(b[0])}
					}
				}
			}()
		}
	} else {
		// Normal mode - read from keyboard
		go func() {
			for {
				// Pass the TUI so we can check the mode
				keyChan <- readKeyWithMode(tui)
			}
		}()
	}
	
	// Create demo player (for file-based demos)
	demoPlayer := demo.NewPlayer(
		func(r rune) { // onKey callback
			demoChan <- editor.KeyEvent{Rune: r}
		},
		func(s string) { // onText callback
			for _, r := range s {
				demoChan <- editor.KeyEvent{Rune: r}
			}
		},
	)

	// Animation ticker - update Ed every 200ms
	animTicker := time.NewTicker(200 * time.Millisecond)
	defer animTicker.Stop()

	// Main render loop
	var buf bytes.Buffer
	
	for {
		// Update terminal size on each iteration (handles resize)
		width, height := getTerminalSize()
		tui.SetTerminalSize(width, height)
		
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
		// Draw jump labels if in jump mode (but not in JSON mode)
		if tui.GetMode() == editor.ModeJump {
			drawJumpLabels(tui, output)
			// Also draw connection labels if in delete, edit, or hint mode
			if tui.GetJumpAction() == editor.JumpActionDelete || tui.GetJumpAction() == editor.JumpActionEdit || tui.GetJumpAction() == editor.JumpActionHint {
				drawConnectionLabels(tui)
			}
		}
		
		// Draw hint menu if in hint mode
		if tui.GetMode() == editor.ModeHintMenu {
			hintDisplay := tui.GetHintMenuDisplay()
			fmt.Print(hintDisplay)
		}

		// Show status line
		showStatusLine(tui, filename, demoPlayer)

		// Draw Ed (but not in JSON mode)
		if tui.GetMode() != editor.ModeJSON {
			drawEd(tui)
		}

		// Position cursor if in edit mode
		if tui.GetMode() == editor.ModeEdit || tui.GetMode() == editor.ModeInsert {
			positionCursor(tui)
		} else {
			// Hide cursor when not editing
			fmt.Print("\033[?25l")
		}

		// Handle input or animation
		select {
		case keyEvent := <-keyChan:
			// Check for demo control keys in normal mode
			if !demoPlayer.IsPlaying() && tui.GetMode() == editor.ModeNormal {
				if keyEvent.Rune == 'P' { // Play demo
					if err := playDemoFile(demoPlayer); err == nil {
						continue // Skip normal key handling
					}
				} else if keyEvent.Rune == 'R' { // Record demo (create example)
					createExampleDemo()
					continue
				}
			}
			
			// Stop demo if playing
			if demoPlayer.IsPlaying() && keyEvent.Rune == 27 { // ESC
				demoPlayer.Stop()
				continue
			}
			
			// Normal key handling
			handleKeyEvent(tui, keyEvent, &filename, demoPlayer)
			if keyEvent.Rune == 'q' && tui.GetMode() == editor.ModeNormal {
				return nil // Exit requested
			}
			
		case demoKeyEvent := <-demoChan:
			// Handle demo input just like real input
			handleKeyEvent(tui, demoKeyEvent, &filename, demoPlayer)

		case <-animTicker.C:
			// Animate Ed
			tui.AnimateEd()
		}
	}
}

// handleKeyEvent processes a key event from either real input or demo playback
func handleKeyEvent(tui *editor.TUIEditor, keyEvent editor.KeyEvent, filename *string, demoPlayer *demo.Player) {
	if keyEvent.IsSpecial() {
		// Handle special keys (arrows, etc.)
		switch tui.GetMode() {
		case editor.ModeInsert, editor.ModeEdit:
			handleSpecialKeyInTextMode(tui, keyEvent.SpecialKey)
		// Could add special key handling for other modes here
		}
	} else {
		// Handle regular keys
		key := keyEvent.Rune
		switch tui.GetMode() {
		case editor.ModeNormal:
			handleNormalMode(tui, key, filename)
		case editor.ModeInsert, editor.ModeEdit:
			handleTextMode(tui, key)
		case editor.ModeJump:
			handleJumpMode(tui, key)
		case editor.ModeCommand:
			handleCommandMode(tui, key, filename)
		case editor.ModeJSON:
			handleJSONMode(tui, key)
		case editor.ModeHintMenu:
			tui.HandleHintMenuInput(key)
		}
	}
}

// playDemoFile loads and plays a demo script
func playDemoFile(player *demo.Player) error {
	// Look for demo.json in current directory
	demoPath := "demo.json"
	if _, err := os.Stat(demoPath); os.IsNotExist(err) {
		// Try in .edd directory
		homeDir, _ := os.UserHomeDir()
		demoPath = filepath.Join(homeDir, ".edd", "demo.json")
		if _, err := os.Stat(demoPath); os.IsNotExist(err) {
			return fmt.Errorf("no demo.json found")
		}
	}
	
	if err := player.LoadScript(demoPath); err != nil {
		return err
	}
	
	return player.Play()
}

// createExampleDemo creates an example demo script
func createExampleDemo() {
	example := demo.GenerateExample()
	ioutil.WriteFile("demo.json", []byte(example), 0644)
	fmt.Print("\033[2K\033[1G") // Clear line
	fmt.Print("Created demo.json - press P to play it")
	time.Sleep(2 * time.Second)
}

func readSingleKey() rune {
	var b [1]byte
	os.Stdin.Read(b[:])
	return rune(b[0])
}

// readKeyWithMode reads a key and handles escape sequences only in edit modes
func readKeyWithMode(tui *editor.TUIEditor) editor.KeyEvent {
	var b [1]byte
	n, _ := os.Stdin.Read(b[:])
	
	if n == 0 {
		return editor.KeyEvent{Rune: 0}
	}
	
	// Only check for escape sequences in edit modes
	mode := tui.GetMode()
	if (mode == editor.ModeEdit || mode == editor.ModeInsert) && b[0] == 27 {
		// In edit mode and got ESC - check for arrow keys
		var seq [10]byte
		seq[0] = b[0]
		
		// Set a very short timeout for reading the rest of the sequence
		oldFlags, _ := fcntl(int(os.Stdin.Fd()), syscall.F_GETFL, 0)
		syscall.SetNonblock(int(os.Stdin.Fd()), true)
		n, _ := os.Stdin.Read(seq[1:])
		fcntl(int(os.Stdin.Fd()), syscall.F_SETFL, oldFlags)
		
		if n > 0 {
			// Parse escape sequence
			return parseEscapeSequence(seq[:n+1])
		}
		// Just ESC
		return editor.KeyEvent{Rune: 27}
	}
	
	// Normal key
	return editor.KeyEvent{Rune: rune(b[0])}
}

// fcntl is a wrapper around the fcntl system call
func fcntl(fd int, cmd int, arg int) (int, error) {
	val, _, err := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), uintptr(cmd), uintptr(arg))
	if err != 0 {
		return 0, err
	}
	return int(val), nil
}

// readKeyWithEscape reads a key and handles escape sequences for special keys
func readKeyWithEscape() editor.KeyEvent {
	var b [1]byte
	n, _ := os.Stdin.Read(b[:])
	
	if n == 0 {
		return editor.KeyEvent{Rune: 0}
	}
	
	// Just return the raw key - we'll handle escape sequences separately when needed
	return editor.KeyEvent{Rune: rune(b[0])}
}

// readKeyWithArrowSupport reads a key and handles arrow keys in edit modes
func readKeyWithArrowSupport() editor.KeyEvent {
	var b [1]byte
	n, _ := os.Stdin.Read(b[:])
	
	if n == 0 {
		return editor.KeyEvent{Rune: 0}
	}
	
	// Check for escape sequences
	if b[0] == 27 { // ESC character
		// Try to read more bytes for escape sequence
		var seq [10]byte
		seq[0] = b[0]
		
		// Use a select with a very short timeout to check if more data is immediately available
		done := make(chan int)
		go func() {
			n, _ := os.Stdin.Read(seq[1:])
			done <- n
		}()
		
		select {
		case n := <-done:
			if n > 0 {
				// Parse escape sequence
				return parseEscapeSequence(seq[:n+1])
			}
		case <-time.After(10 * time.Millisecond):
			// No more bytes quickly available, it's just ESC
		}
		
		return editor.KeyEvent{Rune: 27} // Just ESC
	}
	
	return editor.KeyEvent{Rune: rune(b[0])}
}

// parseEscapeSequence parses ANSI escape sequences into special keys
func parseEscapeSequence(seq []byte) editor.KeyEvent {
	// Handle common escape sequences
	seqStr := string(seq)
	
	// Arrow keys
	if seqStr == "\033[A" || seqStr == "\033OA" {
		return editor.KeyEvent{SpecialKey: editor.KeyArrowUp}
	}
	if seqStr == "\033[B" || seqStr == "\033OB" {
		return editor.KeyEvent{SpecialKey: editor.KeyArrowDown}
	}
	if seqStr == "\033[C" || seqStr == "\033OC" {
		return editor.KeyEvent{SpecialKey: editor.KeyArrowRight}
	}
	if seqStr == "\033[D" || seqStr == "\033OD" {
		return editor.KeyEvent{SpecialKey: editor.KeyArrowLeft}
	}
	
	// Home/End keys
	if seqStr == "\033[H" || seqStr == "\033[1~" || seqStr == "\033OH" {
		return editor.KeyEvent{SpecialKey: editor.KeyHome}
	}
	if seqStr == "\033[F" || seqStr == "\033[4~" || seqStr == "\033OF" {
		return editor.KeyEvent{SpecialKey: editor.KeyEnd}
	}
	
	// Alt+Backspace (word delete) - various terminals send different sequences
	if len(seq) == 2 && seq[0] == 27 && seq[1] == 127 {
		// Alt+Backspace - treat as Ctrl+W (delete word backward)
		return editor.KeyEvent{Rune: 23} // Ctrl+W
	}
	
	// If we don't recognize it, return ESC
	return editor.KeyEvent{Rune: 27}
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
	
	// Calculate actual cursor position on screen using line and column
	// Node text starts at X+2, Y+1 (inside the box)
	cursorX := pos.X + 2 + state.CursorCol + 1 // +1 for terminal indexing
	cursorY := pos.Y + 1 + state.CursorLine + 1 // +1 for terminal indexing

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
	d := tui.GetDiagram()
	
	// Save cursor once
	fmt.Print("\033[s")
	
	// First, draw simple labels on the connections themselves
	// Track occupied positions to avoid overlaps
	occupiedPositions := make(map[string]bool)
	
	for connIndex := 0; connIndex < len(d.Connections); connIndex++ {
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
				} else if jumpAction == editor.JumpActionHint {
					// Magenta background for hint mode
					fmt.Printf("\033[45;97;1m %c \033[0m", label) // Magenta bg, white text
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
	} else if jumpAction == editor.JumpActionHint {
		fmt.Print("\033[95mEdit Hints - Connections:\033[0m ")
	} else {
		fmt.Print("\033[91mConnection Labels:\033[0m ")
	}
	
	// Draw each connection label with its endpoints
	labelCount := 0
	for connIndex := 0; connIndex < len(d.Connections); connIndex++ {
		if label, hasLabel := labels[connIndex]; hasLabel {
			conn := d.Connections[connIndex]
			
			// Find node names
			var fromText, toText string
			for _, node := range d.Nodes {
				if node.ID == conn.From && len(node.Text) > 0 {
					fromText = node.Text[0]
					// Truncate by runes, not bytes
					runes := []rune(fromText)
					if len(runes) > 8 {
						fromText = string(runes[:8])
					}
				}
				if node.ID == conn.To && len(node.Text) > 0 {
					toText = node.Text[0]
					// Truncate by runes, not bytes
					runes := []rune(toText)
					if len(runes) > 8 {
						toText = string(runes[:8])
					}
				}
			}
			
			// Choose color based on action
			if jumpAction == editor.JumpActionEdit {
				// Yellow background for edit mode
				fmt.Printf("\033[43;30m%c\033[0m=%s→%s  ", label, fromText, toText)
			} else if jumpAction == editor.JumpActionHint {
				// Magenta background for hint mode
				fmt.Printf("\033[45;97m%c\033[0m=%s→%s  ", label, fromText, toText)
			} else {
				// Red background for delete mode
				fmt.Printf("\033[41;97m%c\033[0m=%s→%s  ", label, fromText, toText)
			}
			
			labelCount++
			// Start a new line after every 3 entries to avoid running off screen
			if labelCount % 3 == 0 && connIndex < len(d.Connections)-1 {
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

func showStatusLine(tui *editor.TUIEditor, filename string, demoPlayer *demo.Player) {
	// Move to bottom of screen
	fmt.Print("\033[999;1H") // Move to bottom
	fmt.Print("\033[K")      // Clear line

	// Special handling for command mode - show the command being typed
	if tui.GetMode() == editor.ModeCommand {
		cmd := tui.GetCommand()
		fmt.Printf(":%s│", cmd) // Show command with cursor
		return
	}

	// Show demo status if playing
	if demoPlayer.IsPlaying() {
		fmt.Print("\033[33m[DEMO PLAYING - Press ESC to stop] \033[0m")
	}

	// Show filename and mode
	if filename != "" {
		fmt.Printf("[ %s ] ", filename)
	} else {
		fmt.Print("[ untitled ] ")
	}

	// Show node/connection count
	d := tui.GetDiagram()
	
	// Check if we're editing a connection
	if tui.GetMode() == editor.ModeEdit && tui.GetSelectedConnection() >= 0 {
		connIdx := tui.GetSelectedConnection()
		if connIdx < len(d.Connections) {
			conn := d.Connections[connIdx]
			// Find node names for clarity
			var fromName, toName string
			for _, node := range d.Nodes {
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
		mode := tui.GetMode()
		modeStr := mode.String()
		
		// Add indicator for continuous modes
		if mode == editor.ModeJump && tui.GetJumpAction() == editor.JumpActionConnectFrom {
			if tui.IsContinuousConnect() {
				modeStr = "CONNECT (continuous)"
			}
		} else if mode == editor.ModeJump && tui.GetJumpAction() == editor.JumpActionDelete {
			if tui.IsContinuousDelete() {
				modeStr = "DELETE (continuous)"
			}
		}
		
		// Get history status
		histCurrent, histTotal := tui.GetHistoryStats()
		historyStr := ""
		if histTotal > 1 {
			historyStr = fmt.Sprintf(" | History: %d/%d", histCurrent, histTotal)
		}
		
		fmt.Printf("Nodes: %d | Connections: %d | Mode: %s%s",
			len(d.Nodes),
			len(d.Connections),
			modeStr,
			historyStr)
	}
}

func handleNormalMode(tui *editor.TUIEditor, key rune, filename *string) bool {
	switch key {
	case 'q', 3: // q or Ctrl+C
		return true // Exit
	case 'a': // Add node
		tui.StartAddNode()
	case 'c': // Connect (single)
		tui.StartConnect()
	case 'C': // Connect (continuous)
		tui.StartContinuousConnect()
	case 'd': // Delete (single)
		tui.StartDelete()
	case 'D': // Delete (continuous)
		tui.StartContinuousDelete()
	case 'e': // Edit
		tui.StartEdit()
	case 'E': // Edit in external editor
		// Show loading message
		fmt.Print("\033[999;1H\033[K") // Go to bottom and clear line
		fmt.Print("\033[93mLaunching external editor...\033[0m")
		
		if err := launchExternalEditor(tui); err != nil {
			// Show error briefly
			fmt.Print("\033[999;1H\033[K") // Go to bottom and clear line
			fmt.Printf("\033[91mError: %v\033[0m", err)
			time.Sleep(2 * time.Second)
		}
	case 'H': // Edit connection hints
		tui.StartHintEdit()
	case 'j': // JSON view
		tui.SetMode(editor.ModeJSON)
	case 't': // Toggle diagram type
		tui.ToggleDiagramType()
	case 'u': // Undo
		tui.Undo()
	case 18: // Ctrl+R for redo
		tui.Redo()
	case 'J': // Scroll down
		tui.ScrollDiagram(tui.GetTerminalHeight() / 2)
	case 'K': // Scroll up
		tui.ScrollDiagram(-tui.GetTerminalHeight() / 2)
	case 21: // Ctrl+U - scroll up half page
		tui.ScrollDiagram(-tui.GetTerminalHeight() / 2)
	case 4: // Ctrl+D - scroll down half page
		tui.ScrollDiagram(tui.GetTerminalHeight() / 2)
	case 'g': // Go to top
		tui.ScrollToTop()
	case 'G': // Go to bottom
		tui.ScrollToBottom()
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

func handleSpecialKeyInTextMode(tui *editor.TUIEditor, key editor.SpecialKey) {
	switch key {
	case editor.KeyArrowUp:
		tui.HandleArrowKey('U') // Use a special marker for up
	case editor.KeyArrowDown:
		tui.HandleArrowKey('D') // Use a special marker for down
	case editor.KeyArrowLeft:
		tui.HandleArrowKey('L') // Use a special marker for left
	case editor.KeyArrowRight:
		tui.HandleArrowKey('R') // Use a special marker for right
	case editor.KeyHome:
		tui.HandleArrowKey('H') // Home key
	case editor.KeyEnd:
		tui.HandleArrowKey('E') // End key
	}
}

func handleJumpMode(tui *editor.TUIEditor, key rune) {
	tui.HandleJumpInput(key)
}

func handleJSONMode(tui *editor.TUIEditor, key rune) {
	// Handle 'E' for external editor from JSON mode
	if key == 'E' {
		// Exit JSON mode first
		tui.SetMode(editor.ModeNormal)
		
		// Show loading message
		fmt.Print("\033[999;1H\033[K") // Go to bottom and clear line
		fmt.Print("\033[93mLaunching external editor...\033[0m")
		
		if err := launchExternalEditor(tui); err != nil {
			// Show error briefly
			fmt.Print("\033[999;1H\033[K") // Go to bottom and clear line
			fmt.Printf("\033[91mError: %v\033[0m", err)
			time.Sleep(2 * time.Second)
		}
		return
	}
	
	// The TUI editor handles other JSON mode keys internally
	tui.HandleJSONInput(key)
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
	fmt.Println("  a     - Add new node (Enter creates more)")
	fmt.Println("  c     - Connect nodes (single)")
	fmt.Println("  C     - Connect nodes (continuous)")
	fmt.Println("  d     - Delete node/connection (single)")
	fmt.Println("  D     - Delete node/connection (continuous)")
	fmt.Println("  e     - Edit node/connection text")
	fmt.Println("  E     - Edit JSON in $EDITOR")
	fmt.Println("  H     - Edit connection hints (style/color)")
	fmt.Println("  j     - Toggle JSON view")
	fmt.Println("  u     - Undo")
	fmt.Println("  Ctrl+R - Redo")
	fmt.Println()
	fmt.Println("  Scrolling (for large diagrams):")
	fmt.Println("  J     - Scroll down half page")
	fmt.Println("  K     - Scroll up half page")
	fmt.Println("  g     - Go to top")
	fmt.Println("  G     - Go to bottom")
	fmt.Println("  Ctrl+U - Scroll up half page")
	fmt.Println("  Ctrl+D - Scroll down half page")
	fmt.Println()
	fmt.Println("  P     - Play demo (from demo.json)")
	fmt.Println("  R     - Create example demo.json")
	fmt.Println("  q     - Quit")
	fmt.Println("  :     - Command mode")
	fmt.Println()
	fmt.Println("Command Mode:")
	fmt.Println("  :w [file]  - Save diagram")
	fmt.Println("  :q         - Quit")
	fmt.Println("  :wq        - Save and quit")
	fmt.Println()
	fmt.Println("Text Editing:")
	fmt.Println("  ESC    - Exit to normal mode")
	fmt.Println("  Enter  - Confirm text")
	fmt.Println()
	fmt.Println("  Movement:")
	fmt.Println("    Arrow Keys - Move cursor (↑ ↓ ← →)")
	fmt.Println("    Ctrl+a / Home - Move to beginning of line")
	fmt.Println("    Ctrl+e / End  - Move to end of line")
	fmt.Println("    Ctrl+f - Move forward one character")
	fmt.Println("    Ctrl+b - Move backward one character")
	fmt.Println("    Ctrl+p - Move up one line")
	fmt.Println("    Ctrl+v - Move down one line")
	fmt.Println()
	fmt.Println("  Editing:")
	fmt.Println("    Ctrl+n - Insert newline (multi-line)")
	fmt.Println("    Ctrl+w - Delete word backward")
	fmt.Println("    Ctrl+u - Delete to beginning of line")
	fmt.Println("    Ctrl+k - Delete to end of line")
	fmt.Println()
	fmt.Println("JSON View Mode:")
	fmt.Println("  j/q/ESC - Return to diagram")
	fmt.Println("  k       - Scroll up")
	fmt.Println("  J       - Scroll down")
	fmt.Println("  g       - Go to top")
	fmt.Println("  G       - Go to bottom")
	fmt.Println("  E       - Edit in $EDITOR")
	fmt.Println()
	fmt.Println("Press any key to continue...")
	readSingleKey()

	// Clear screen completely after help is dismissed
	fmt.Print("\033[2J\033[H")
}


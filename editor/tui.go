package editor

import (
	"edd/core"
	"fmt"
)

// TUIEditor represents the interactive terminal UI editor
type TUIEditor struct {
	diagram  *core.Diagram
	renderer DiagramRenderer

	// UI State (minimal!)
	mode       Mode
	selected   int            // Currently selected node ID (-1 for none)
	jumpLabels map[int]rune   // Node ID -> jump label mapping
	jumpAction JumpAction     // What to do after jump selection

	// Text input state
	textBuffer []rune // Unicode-aware text buffer
	cursorPos  int    // Position in text buffer

	// Ed mascot
	edd *EddCharacter

	// Terminal state
	width  int
	height int
	
	// Node positions from last layout (for jump label positioning)
	nodePositions map[int]core.Point // Node ID -> position from last render
}

// NewTUIEditor creates a new TUI editor instance
func NewTUIEditor(renderer DiagramRenderer) *TUIEditor {
	return &TUIEditor{
		diagram:       &core.Diagram{},
		renderer:      renderer,
		mode:          ModeNormal,
		selected:      -1,
		jumpLabels:    make(map[int]rune),
		textBuffer:    []rune{},
		cursorPos:     0,
		edd:           NewEddCharacter(),
		width:         80,
		height:        24,
		nodePositions: make(map[int]core.Point),
	}
}

// SetDiagram sets the diagram to edit
func (e *TUIEditor) SetDiagram(d *core.Diagram) {
	e.diagram = d
}

// GetDiagram returns the current diagram
func (e *TUIEditor) GetDiagram() *core.Diagram {
	return e.diagram
}

// SetTerminalSize updates the terminal dimensions
func (e *TUIEditor) SetTerminalSize(width, height int) {
	e.width = width
	e.height = height
}

// Run starts the interactive editor loop
func (e *TUIEditor) Run() error {
	// Setup terminal
	if err := e.setupTerminal(); err != nil {
		return err
	}
	defer e.restoreTerminal()

	// Main loop
	for {
		// Render
		output := e.Render()
		e.clearScreen()
		fmt.Print(output)

		// Read input
		key, err := e.readKey()
		if err != nil {
			return err
		}

		// Handle input
		if e.handleKey(key) {
			break // Exit requested
		}
	}

	return nil
}

// Render produces the current display output
func (e *TUIEditor) Render() string {
	// If we have a real renderer that can provide positions, use it
	if realRenderer, ok := e.renderer.(*RealRenderer); ok {
		positions, output, err := realRenderer.RenderWithPositions(e.diagram)
		if err == nil && positions != nil {
			// Store node positions for jump label rendering
			e.nodePositions = positions.Positions
			return output
		}
	}
	
	// Fall back to simple rendering
	state := e.GetState()
	return RenderTUIWithRenderer(state, e.renderer)
}

// GetState extracts the current state for stateless rendering
func (e *TUIEditor) GetState() TUIState {
	return TUIState{
		Diagram:    e.diagram,
		Mode:       e.mode,
		Selected:   e.selected,
		JumpLabels: e.jumpLabels,
		TextBuffer: e.textBuffer,
		CursorPos:  e.cursorPos,
		EddFrame:   e.edd.GetFrame(e.mode),
		Width:      e.width,
		Height:     e.height,
	}
}

// handleKey processes keyboard input
func (e *TUIEditor) handleKey(key rune) bool {
	// Handle jump mode first
	if len(e.jumpLabels) > 0 {
		return e.handleJumpKey(key)
	}

	// Handle based on mode
	switch e.mode {
	case ModeNormal:
		return e.handleNormalKey(key)
	case ModeInsert, ModeEdit:
		return e.handleTextKey(key)
	case ModeCommand:
		return e.handleCommandKey(key)
	}

	return false
}

// clearScreen clears the terminal
func (e *TUIEditor) clearScreen() {
	fmt.Print("\033[H\033[2J")
}

// setupTerminal configures the terminal for raw input
func (e *TUIEditor) setupTerminal() error {
	// TODO: Implement terminal setup (raw mode, etc.)
	// For now, return nil to allow testing
	return nil
}

// restoreTerminal restores normal terminal settings
func (e *TUIEditor) restoreTerminal() {
	// TODO: Implement terminal restoration
	fmt.Print("\033[?25h") // Show cursor
}

// readKey reads a single key from input
func (e *TUIEditor) readKey() (rune, error) {
	// TODO: Implement actual key reading
	// For now, read from stdin (will need proper implementation)
	var key rune
	_, err := fmt.Scanf("%c", &key)
	return key, err
}

// AddNode adds a new node to the diagram
func (e *TUIEditor) AddNode(text []string) int {
	// Find next available ID
	maxID := 0
	for _, node := range e.diagram.Nodes {
		if node.ID > maxID {
			maxID = node.ID
		}
	}

	newNode := core.Node{
		ID:   maxID + 1,
		Text: text,
	}

	e.diagram.Nodes = append(e.diagram.Nodes, newNode)
	return newNode.ID
}

// DeleteNode removes a node from the diagram
func (e *TUIEditor) DeleteNode(nodeID int) {
	// Remove node
	for i, node := range e.diagram.Nodes {
		if node.ID == nodeID {
			e.diagram.Nodes = append(e.diagram.Nodes[:i], e.diagram.Nodes[i+1:]...)
			break
		}
	}

	// Remove connections involving this node
	newConnections := []core.Connection{}
	for _, conn := range e.diagram.Connections {
		if conn.From != nodeID && conn.To != nodeID {
			newConnections = append(newConnections, conn)
		}
	}
	e.diagram.Connections = newConnections
}

// AddConnection adds a connection between two nodes
func (e *TUIEditor) AddConnection(from, to int, label string) {
	conn := core.Connection{
		From:  from,
		To:    to,
		Label: label,
	}
	e.diagram.Connections = append(e.diagram.Connections, conn)
}

// UpdateNodeText updates the text of a node
func (e *TUIEditor) UpdateNodeText(nodeID int, text []string) {
	for i, node := range e.diagram.Nodes {
		if node.ID == nodeID {
			e.diagram.Nodes[i].Text = text
			break
		}
	}
}
package editor

import (
	"edd/core"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// TUIEditor represents the interactive terminal UI editor
type TUIEditor struct {
	diagram  *core.Diagram
	renderer DiagramRenderer

	// UI State (minimal!)
	mode               Mode
	selected           int            // Currently selected node ID (-1 for none)
	selectedConnection int            // Currently selected connection index (-1 for none)
	jumpLabels         map[int]rune   // Node ID -> jump label mapping
	connectionLabels   map[int]rune   // Connection index -> jump label mapping
	jumpAction         JumpAction     // What to do after jump selection
	continuousConnect  bool           // Whether to continue connecting after each connection
	continuousDelete   bool           // Whether to continue deleting after each deletion

	// Text input state
	textBuffer    []rune // Unicode-aware text buffer for editing nodes
	cursorPos     int    // Position in text buffer
	cursorLine    int    // Current line in multi-line edit (0-based)
	cursorCol     int    // Current column in current line (0-based)
	commandBuffer []rune // Separate buffer for command mode

	// Ed mascot
	edd *EddCharacter

	// Terminal state
	width  int
	height int
	
	// Positions from last layout (for jump label positioning)
	nodePositions       map[int]core.Point // Node ID -> position from last render
	connectionPaths     map[int]core.Path  // Connection index -> path from last render
	
	// JSON view state
	jsonScrollOffset    int  // Current scroll position in JSON view
	
	// History management
	history            *StructHistory  // Undo/redo history (optimized struct-based)
}

// NewTUIEditor creates a new TUI editor instance
func NewTUIEditor(renderer DiagramRenderer) *TUIEditor {
	editor := &TUIEditor{
		diagram:            &core.Diagram{},
		renderer:           renderer,
		mode:               ModeNormal,
		selected:           -1,
		selectedConnection: -1,
		jumpLabels:         make(map[int]rune),
		connectionLabels:   make(map[int]rune),
		textBuffer:         []rune{},
		commandBuffer:      []rune{},
		cursorPos:          0,
		edd:                NewEddCharacter(),
		width:              80,
		height:             24,
		nodePositions:      make(map[int]core.Point),
		connectionPaths:    make(map[int]core.Path),
		continuousConnect:  false,
		continuousDelete:   false,
		jsonScrollOffset:   0,
		history:            NewStructHistory(50), // 50 states max (optimized)
	}
	
	// Save initial empty state
	editor.history.SaveState(editor.diagram)
	
	return editor
}

// SetDiagram sets the diagram to edit
func (e *TUIEditor) SetDiagram(d *core.Diagram) {
	e.diagram = d
	// Save this as a new state in history
	e.history.SaveState(d)
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
	// If in JSON mode, render JSON instead
	if e.mode == ModeJSON {
		return e.renderJSON()
	}
	
	// If we have a real renderer that can provide positions, use it
	if realRenderer, ok := e.renderer.(*RealRenderer); ok {
		// Set edit state if we're editing or inserting
		if e.mode == ModeEdit || e.mode == ModeInsert {
			realRenderer.SetEditState(e.selected, string(e.textBuffer), e.cursorPos)
		} else {
			realRenderer.SetEditState(-1, "", 0)
		}
		
		positions, output, err := realRenderer.RenderWithPositions(e.diagram)
		if err == nil && positions != nil {
			// Store node positions and connection paths for jump label rendering
			e.nodePositions = positions.Positions
			e.connectionPaths = positions.ConnectionPaths
			return output
		}
		// If there was an error, fall through to simple rendering
		if err != nil {
			return fmt.Sprintf("Render error: %v\n", err)
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
		CursorLine: e.cursorLine,
		CursorCol:  e.cursorCol,
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
	case ModeJSON:
		return e.handleJSONKey(key)
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
	
	// Save to history after modification
	e.SaveHistory()
	
	return newNode.ID
}

// DeleteNode removes a node from the diagram
func (e *TUIEditor) DeleteNode(nodeID int) {
	// Remove node
	for i, node := range e.diagram.Nodes {
		if node.ID == nodeID {
			e.diagram.Nodes = slices.Delete(e.diagram.Nodes, i, i+1)
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
	
	// Save to history after modification
	e.SaveHistory()
}

// AddConnection adds a connection between two nodes
func (e *TUIEditor) AddConnection(from, to int, label string) {
	conn := core.Connection{
		From:  from,
		To:    to,
		Label: label,
	}
	e.diagram.Connections = append(e.diagram.Connections, conn)
	
	// Save to history after modification
	e.SaveHistory()
}

// DeleteConnection removes a connection by index
func (e *TUIEditor) DeleteConnection(index int) {
	if index >= 0 && index < len(e.diagram.Connections) {
		e.diagram.Connections = append(
			e.diagram.Connections[:index],
			e.diagram.Connections[index+1:]...,
		)
		
		// Save to history after modification
		e.SaveHistory()
	}
}

// UpdateNodeText updates the text of a node
func (e *TUIEditor) UpdateNodeText(nodeID int, text []string) {
	for i, node := range e.diagram.Nodes {
		if node.ID == nodeID {
			e.diagram.Nodes[i].Text = text
			break
		}
	}
	
	// Save to history after modification
	e.SaveHistory()
}

// StartEditingConnection begins editing a connection's label
func (e *TUIEditor) StartEditingConnection(connIndex int) {
	if connIndex >= 0 && connIndex < len(e.diagram.Connections) {
		e.selectedConnection = connIndex
		e.selected = -1 // Clear node selection
		
		// Load current connection label into text buffer
		currentLabel := e.diagram.Connections[connIndex].Label
		e.textBuffer = []rune(currentLabel)
		e.cursorPos = len(e.textBuffer)
		
		// Clear jump labels and enter edit mode
		e.clearJumpLabels()
		e.SetMode(ModeEdit)
	}
}

// UpdateConnectionLabel updates the label of a connection
func (e *TUIEditor) UpdateConnectionLabel(connIndex int, label string) {
	if connIndex >= 0 && connIndex < len(e.diagram.Connections) {
		e.diagram.Connections[connIndex].Label = label
		
		// Save to history after modification
		e.SaveHistory()
	}
}

// GetSelectedConnection returns the currently selected connection index
func (e *TUIEditor) GetSelectedConnection() int {
	return e.selectedConnection
}

// renderJSON renders the diagram as formatted JSON
func (e *TUIEditor) renderJSON() string {
	// Marshal with indentation
	jsonBytes, err := json.MarshalIndent(e.diagram, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error rendering JSON: %v", err)
	}
	
	// Split into lines for scrolling
	lines := strings.Split(string(jsonBytes), "\n")
	
	// Calculate visible lines (leave room for status)
	visibleLines := e.height - 2
	if visibleLines < 1 {
		visibleLines = 1
	}
	
	// Adjust scroll offset if needed
	maxOffset := len(lines) - visibleLines
	if maxOffset < 0 {
		maxOffset = 0
	}
	if e.jsonScrollOffset > maxOffset {
		e.jsonScrollOffset = maxOffset
	}
	if e.jsonScrollOffset < 0 {
		e.jsonScrollOffset = 0
	}
	
	// Build output
	var output strings.Builder
	
	// Show line numbers and content
	endLine := e.jsonScrollOffset + visibleLines
	if endLine > len(lines) {
		endLine = len(lines)
	}
	
	for i := e.jsonScrollOffset; i < endLine; i++ {
		// Add line number in gray
		output.WriteString(fmt.Sprintf("\033[90m%4d │\033[0m %s\n", i+1, lines[i]))
	}
	
	// Add scroll indicator if there's more content
	if len(lines) > visibleLines {
		scrollPercent := 0
		if maxOffset > 0 {
			scrollPercent = (e.jsonScrollOffset * 100) / maxOffset
		}
		output.WriteString(fmt.Sprintf("\n\033[90m[Line %d-%d of %d | %d%%]\033[0m",
			e.jsonScrollOffset+1, endLine, len(lines), scrollPercent))
	}
	
	return output.String()
}

// GetJSONScrollOffset returns the current JSON scroll offset
func (e *TUIEditor) GetJSONScrollOffset() int {
	return e.jsonScrollOffset
}

// ScrollJSON scrolls the JSON view
func (e *TUIEditor) ScrollJSON(delta int) {
	e.jsonScrollOffset += delta
	// Bounds checking will be done in renderJSON
}

// Undo undoes the last action
func (e *TUIEditor) Undo() {
	if diagram, err := e.history.Undo(); err == nil && diagram != nil {
		e.diagram = diagram
		// Clear any selection
		e.selected = -1
		e.selectedConnection = -1
		e.clearJumpLabels()
	}
}

// Redo redoes the next action
func (e *TUIEditor) Redo() {
	if diagram, err := e.history.Redo(); err == nil && diagram != nil {
		e.diagram = diagram
		// Clear any selection
		e.selected = -1
		e.selectedConnection = -1
		e.clearJumpLabels()
	}
}

// SaveHistory saves the current state to history (call after modifications)
func (e *TUIEditor) SaveHistory() {
	e.history.SaveState(e.diagram)
}

// GetHistoryStats returns undo/redo statistics
func (e *TUIEditor) GetHistoryStats() (current, total int) {
	return e.history.Stats()
}

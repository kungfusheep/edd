package editor

import (
	"edd/diagram"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"
	"unicode"
)

// TUIEditor represents the interactive terminal UI editor
type TUIEditor struct {
	diagram  *diagram.Diagram
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
	editingHintConn    int            // Connection being edited for hints (-1 for none)
	editingHintNode    int            // Node being edited for hints (-1 for none)
	previousJumpAction JumpAction     // Remember the jump action for ESC handling

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
	nodePositions       map[int]diagram.Point // Node ID -> position from last render
	connectionPaths     map[int]diagram.Path  // Connection index -> path from last render
	
	// JSON view state
	jsonScrollOffset    int  // Current scroll position in JSON view
	
	// History management
	history            *StructHistory  // Undo/redo history (optimized struct-based)
	
	// Test-only field (not used in production)
	connectFrom        int            // Used by test helpers for connection tracking
}

// NewTUIEditor creates a new TUI editor instance
func NewTUIEditor(renderer DiagramRenderer) *TUIEditor {
	editor := &TUIEditor{
		diagram:            &diagram.Diagram{Type: "box"},  // Default to box diagram
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
		nodePositions:      make(map[int]diagram.Point),
		connectionPaths:    make(map[int]diagram.Path),
		continuousConnect:  false,
		continuousDelete:   false,
		editingHintConn:    -1,  // Initialize to -1 (no connection being edited)
		editingHintNode:    -1,  // Initialize to -1 (no node being edited)
		jsonScrollOffset:   0,
		history:            NewStructHistory(50), // 50 states max (optimized)
		connectFrom:        -1,  // Initialize test-only field
	}
	
	// Save initial empty state
	editor.history.SaveState(editor.diagram)
	
	return editor
}

// SetDiagram sets the diagram to edit
func (e *TUIEditor) SetDiagram(d *diagram.Diagram) {
	e.diagram = d
	// Save this as a new state in history
	e.history.SaveState(d)
}

// GetDiagram returns the current diagram
func (e *TUIEditor) GetDiagram() *diagram.Diagram {
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
	
	// If in Help mode, render help text
	if e.mode == ModeHelp {
		return GetHelpText()
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
	case ModeHelp:
		return e.handleHelpKey(key)
	case ModeHintMenu:
		e.HandleHintMenuInput(key)
		return false
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

	newNode := diagram.Node{
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
	newConnections := []diagram.Connection{}
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
	// In sequence diagrams, allow multiple messages between same participants
	// In flowcharts, check for duplicate connections
	if e.diagram.Type != string(diagram.DiagramTypeSequence) {
		// Check for duplicate connections in the same direction only
		for _, existing := range e.diagram.Connections {
			if existing.From == from && existing.To == to {
				// Connection already exists in this direction
				// TODO: Consider providing feedback to user
				return
			}
		}
	}
	
	conn := diagram.Connection{
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

// HandleKey processes a key (exported for testing)
// HandleKey is the public entry point for key handling - used by tests
func (e *TUIEditor) HandleKey(key rune) bool {
	// In production, the TUI package handles keys directly
	// This is only used by tests, delegate to appropriate handler
	
	// Check mode to determine which handler to use
	if len(e.jumpLabels) > 0 {
		// In jump mode
		return e.handleJumpKey(key)
	}
	
	switch e.mode {
	case ModeNormal:
		return e.handleNormalKey(key)
	case ModeInsert, ModeEdit:
		return e.handleTextKey(key)
	case ModeCommand:
		return e.handleCommandKey(key)
	case ModeJSON:
		return e.handleJSONKey(key)
	case ModeHelp:
		return e.handleHelpKey(key)
	case ModeHintMenu:
		e.HandleHintMenuInput(key)
		return false
	}
	
	return false
}

// ============================================
// Methods from arrow_keys.go
// ============================================

// HandleArrowKey handles arrow keys and other special navigation keys
func (e *TUIEditor) HandleArrowKey(direction rune) {
	// Only handle in text editing modes
	if e.mode != ModeEdit && e.mode != ModeInsert {
		return
	}
	
	switch direction {
	case 'U': // Arrow Up
		e.moveCursorUp()
	case 'D': // Arrow Down
		e.moveCursorDown()
	case 'L': // Arrow Left
		e.moveCursorBackward()
	case 'R': // Arrow Right
		e.moveCursorForward()
	case 'H': // Home key
		e.moveCursorToBeginningOfLine()
	case 'E': // End key
		e.moveCursorToEndOfLine()
	}
}

// ============================================
// Methods from jump.go
// ============================================

// Jump label characters in order of preference (home row first)
const jumpChars = "asdfghjklqwertyuiopzxcvbnm"

// startJump initiates jump mode with labels
func (e *TUIEditor) startJump(action JumpAction) {
	e.jumpAction = action
	e.assignJumpLabels()
	e.SetMode(ModeJump)
}
// assignJumpLabels assigns single-character labels to nodes and connections
func (e *TUIEditor) assignJumpLabels() {
	e.jumpLabels = make(map[int]rune)
	e.connectionLabels = make(map[int]rune)
	
	labelIndex := 0
	
	// Assign labels to nodes
	for _, node := range e.diagram.Nodes {
		if labelIndex < len(jumpChars) {
			e.jumpLabels[node.ID] = rune(jumpChars[labelIndex])
			labelIndex++
		} else {
			break
		}
	}
	
	// If in delete, edit, or hint mode, also assign labels to connections
	if e.jumpAction == JumpActionDelete || e.jumpAction == JumpActionEdit || e.jumpAction == JumpActionHint {
		// Use index-based iteration to ensure consistent ordering
		for i := 0; i < len(e.diagram.Connections); i++ {
			if labelIndex < len(jumpChars) {
				e.connectionLabels[i] = rune(jumpChars[labelIndex])
				labelIndex++
			} else {
				break
			}
		}
	}
}
// getJumpLabel returns the jump label for a node ID
func (e *TUIEditor) getJumpLabel(nodeID int) string {
	if label, ok := e.jumpLabels[nodeID]; ok {
		return string(label)
	}
	return ""
}
// clearJumpLabels clears all jump labels
func (e *TUIEditor) clearJumpLabels() {
	e.jumpLabels = make(map[int]rune)
	e.connectionLabels = make(map[int]rune)
	e.jumpAction = JumpActionSelect
}

// ============================================
// Methods from modes.go
// ============================================

// Mode represents the current editing mode
type Mode int

const (
	ModeNormal  Mode = iota // Normal navigation mode
	ModeInsert              // Inserting new nodes
	ModeEdit                // Editing existing node text
	ModeCommand             // Command input mode
	ModeJump                // Jump selection active
	ModeJSON                // JSON view mode
	ModeHintMenu            // Editing connection hints
	ModeHelp                // Help display mode
)

// String returns the mode name for display
func (m Mode) String() string {
	switch m {
	case ModeNormal:
		return "NORMAL"
	case ModeInsert:
		return "INSERT"
	case ModeEdit:
		return "EDIT"
	case ModeCommand:
		return "COMMAND"
	case ModeJump:
		return "JUMP"
	case ModeJSON:
		return "JSON"
	case ModeHintMenu:
		return "HINTS"
	case ModeHelp:
		return "HELP"
	default:
		return "UNKNOWN"
	}
}
// JumpAction represents what to do after a jump selection
type JumpAction int

const (
	JumpActionSelect     JumpAction = iota // Just select the node
	JumpActionEdit                          // Edit the selected node
	JumpActionDelete                        // Delete the selected node
	JumpActionConnectFrom                   // Start connection from this node
	JumpActionConnectTo                     // Complete connection to this node
	JumpActionHint                          // Edit hints for nodes and connections
)

// SetMode changes the editor mode
func (e *TUIEditor) SetMode(mode Mode) {
	e.mode = mode
	
	// Clear jump labels when leaving jump mode
	if mode != ModeJump {
		e.jumpLabels = make(map[int]rune)
	}
	
	// Clear text buffer when entering text modes
	if mode == ModeInsert || mode == ModeEdit {
		e.textBuffer = []rune{}
		e.cursorPos = 0
		e.cursorLine = 0
		e.cursorCol = 0
		
		// If editing existing node, load its text (support multi-line)
		if mode == ModeEdit && e.selected >= 0 {
			for _, node := range e.diagram.Nodes {
				if node.ID == e.selected {
					if len(node.Text) > 0 {
						// Load all lines, not just the first
						e.SetTextFromLines(node.Text)
					}
					break
				}
			}
		}
		
		// If editing existing connection, load its label
		if mode == ModeEdit && e.selectedConnection >= 0 && e.selectedConnection < len(e.diagram.Connections) {
			e.textBuffer = []rune(e.diagram.Connections[e.selectedConnection].Label)
			e.cursorPos = len(e.textBuffer)
		}
	}
}

// ============================================
// Methods from cursor_movement.go
// ============================================

// moveCursorToBeginningOfLine moves cursor to the beginning of the current line (Ctrl+A)
func (e *TUIEditor) moveCursorToBeginningOfLine() {
	if e.cursorPos == 0 {
		return
	}
	
	// Find the start of the current line
	newPos := e.cursorPos
	for newPos > 0 && e.textBuffer[newPos-1] != '\n' {
		newPos--
	}
	
	e.cursorPos = newPos
	e.updateCursorPosition()
}
// moveCursorToEndOfLine moves cursor to the end of the current line (Ctrl+E)
func (e *TUIEditor) moveCursorToEndOfLine() {
	if e.cursorPos >= len(e.textBuffer) {
		return
	}
	
	// Find the end of the current line
	newPos := e.cursorPos
	for newPos < len(e.textBuffer) && e.textBuffer[newPos] != '\n' {
		newPos++
	}
	
	e.cursorPos = newPos
	e.updateCursorPosition()
}
// moveCursorForward moves cursor forward one character (Ctrl+F)
func (e *TUIEditor) moveCursorForward() {
	if e.cursorPos < len(e.textBuffer) {
		e.cursorPos++
		e.updateCursorPosition()
	}
}
// moveCursorBackward moves cursor backward one character (Ctrl+B)
func (e *TUIEditor) moveCursorBackward() {
	if e.cursorPos > 0 {
		e.cursorPos--
		e.updateCursorPosition()
	}
}
// Additional useful cursor movements

// moveCursorWordForward moves cursor to the beginning of the next word (Alt+F in terminals)
func (e *TUIEditor) moveCursorWordForward() {
	if e.cursorPos >= len(e.textBuffer) {
		return
	}
	
	// Skip current word
	for e.cursorPos < len(e.textBuffer) && e.textBuffer[e.cursorPos] != ' ' && e.textBuffer[e.cursorPos] != '\n' {
		e.cursorPos++
	}
	
	// Skip spaces
	for e.cursorPos < len(e.textBuffer) && e.textBuffer[e.cursorPos] == ' ' {
		e.cursorPos++
	}
	
	e.updateCursorPosition()
}
// moveCursorWordBackward moves cursor to the beginning of the previous word (Alt+B in terminals)
func (e *TUIEditor) moveCursorWordBackward() {
	if e.cursorPos == 0 {
		return
	}
	
	// Move back one position
	e.cursorPos--
	
	// Skip spaces
	for e.cursorPos > 0 && e.textBuffer[e.cursorPos] == ' ' {
		e.cursorPos--
	}
	
	// Find beginning of word
	for e.cursorPos > 0 && e.textBuffer[e.cursorPos-1] != ' ' && e.textBuffer[e.cursorPos-1] != '\n' {
		e.cursorPos--
	}
	
	e.updateCursorPosition()
}
// moveCursorUp moves cursor up one line (Arrow Up)
func (e *TUIEditor) moveCursorUp() {
	e.moveUp()
}
// moveCursorDown moves cursor down one line (Arrow Down)
func (e *TUIEditor) moveCursorDown() {
	e.moveDown()
}

// ============================================
// Methods from text_editing.go
// ============================================

// deleteWordBackward deletes the previous word (Ctrl+W)
func (e *TUIEditor) deleteWordBackward() {
	if e.cursorPos == 0 {
		return
	}
	
	// Find the start of the previous word
	startPos := e.cursorPos - 1
	
	// Skip any trailing spaces
	for startPos >= 0 && e.textBuffer[startPos] == ' ' {
		startPos--
	}
	
	// Skip the word itself (non-space characters)
	for startPos >= 0 && e.textBuffer[startPos] != ' ' && e.textBuffer[startPos] != '\n' {
		startPos--
	}
	
	// startPos is now one position before the word start
	startPos++
	
	// Delete from startPos to cursorPos
	if startPos < e.cursorPos {
		e.textBuffer = append(e.textBuffer[:startPos], e.textBuffer[e.cursorPos:]...)
		e.cursorPos = startPos
		e.updateCursorPosition()
	}
}
// deleteToBeginningOfLine deletes from cursor to beginning of current line (Ctrl+U)
func (e *TUIEditor) deleteToBeginningOfLine() {
	if e.cursorPos == 0 {
		return
	}
	
	// Find the start of the current line
	lineStart := e.cursorPos
	for lineStart > 0 && e.textBuffer[lineStart-1] != '\n' {
		lineStart--
	}
	
	// Delete from lineStart to cursorPos
	if lineStart < e.cursorPos {
		e.textBuffer = append(e.textBuffer[:lineStart], e.textBuffer[e.cursorPos:]...)
		e.cursorPos = lineStart
		e.updateCursorPosition()
	}
}
// deleteToEndOfLine deletes from cursor to end of current line (Ctrl+K)
func (e *TUIEditor) deleteToEndOfLine() {
	if e.cursorPos >= len(e.textBuffer) {
		return
	}
	
	// Find the end of the current line
	lineEnd := e.cursorPos
	for lineEnd < len(e.textBuffer) && e.textBuffer[lineEnd] != '\n' {
		lineEnd++
	}
	
	// Delete from cursorPos to lineEnd
	if lineEnd > e.cursorPos {
		e.textBuffer = append(e.textBuffer[:e.cursorPos], e.textBuffer[lineEnd:]...)
		// cursorPos stays the same
		e.updateCursorPosition()
	}
}
// deleteWord deletes the word at cursor position (for future use)
func (e *TUIEditor) deleteWord() {
	if e.cursorPos >= len(e.textBuffer) {
		return
	}
	
	endPos := e.cursorPos
	
	// Skip any leading spaces
	for endPos < len(e.textBuffer) && e.textBuffer[endPos] == ' ' {
		endPos++
	}
	
	// Skip the word itself
	for endPos < len(e.textBuffer) && e.textBuffer[endPos] != ' ' && e.textBuffer[endPos] != '\n' {
		endPos++
	}
	
	// Delete from cursorPos to endPos
	if endPos > e.cursorPos {
		e.textBuffer = append(e.textBuffer[:e.cursorPos], e.textBuffer[endPos:]...)
		// cursorPos stays the same
		e.updateCursorPosition()
	}
}
// Helper to check if a rune is a word boundary
func isWordBoundary(r rune) bool {
	return unicode.IsSpace(r) || unicode.IsPunct(r)
}

// ============================================
// Methods from multiline.go
// ============================================

// splitIntoLines splits the text buffer into lines
func (e *TUIEditor) splitIntoLines() [][]rune {
	if len(e.textBuffer) == 0 {
		return [][]rune{{}}
	}
	
	lines := [][]rune{}
	currentLine := []rune{}
	
	for _, r := range e.textBuffer {
		if r == '\n' {
			lines = append(lines, currentLine)
			currentLine = []rune{}
		} else {
			currentLine = append(currentLine, r)
		}
	}
	
	// Add the last line
	lines = append(lines, currentLine)
	return lines
}
// updateCursorPosition updates line and column based on cursorPos
func (e *TUIEditor) updateCursorPosition() {
	if len(e.textBuffer) == 0 {
		e.cursorLine = 0
		e.cursorCol = 0
		return
	}
	
	line := 0
	col := 0
	
	for i := 0; i < e.cursorPos && i < len(e.textBuffer); i++ {
		if e.textBuffer[i] == '\n' {
			line++
			col = 0
		} else {
			col++
		}
	}
	
	e.cursorLine = line
	e.cursorCol = col
}
// getCursorPosFromLineCol calculates buffer position from line/column
func (e *TUIEditor) getCursorPosFromLineCol(line, col int) int {
	pos := 0
	currentLine := 0
	currentCol := 0
	
	for i, r := range e.textBuffer {
		if currentLine == line && currentCol == col {
			return i
		}
		
		if r == '\n' {
			if currentLine == line {
				// We're past the end of the target line
				return i
			}
			currentLine++
			currentCol = 0
		} else {
			currentCol++
		}
		pos = i + 1
	}
	
	return pos
}
// moveUp moves cursor up one line
func (e *TUIEditor) moveUp() {
	lines := e.splitIntoLines()
	
	if e.cursorLine > 0 {
		e.cursorLine--
		// Try to maintain column position
		if e.cursorCol > len(lines[e.cursorLine]) {
			e.cursorCol = len(lines[e.cursorLine])
		}
		e.cursorPos = e.getCursorPosFromLineCol(e.cursorLine, e.cursorCol)
	}
}
// moveDown moves cursor down one line
func (e *TUIEditor) moveDown() {
	lines := e.splitIntoLines()
	
	if e.cursorLine < len(lines)-1 {
		e.cursorLine++
		// Try to maintain column position
		if e.cursorCol > len(lines[e.cursorLine]) {
			e.cursorCol = len(lines[e.cursorLine])
		}
		e.cursorPos = e.getCursorPosFromLineCol(e.cursorLine, e.cursorCol)
	}
}
// insertNewline inserts a newline at cursor position
func (e *TUIEditor) insertNewline() {
	e.textBuffer = append(
		e.textBuffer[:e.cursorPos],
		append([]rune{'\n'}, e.textBuffer[e.cursorPos:]...)...,
	)
	e.cursorPos++
	e.updateCursorPosition()
}
// GetTextAsLines returns the current text buffer as lines
func (e *TUIEditor) GetTextAsLines() []string {
	if len(e.textBuffer) == 0 {
		return []string{""}
	}
	
	text := string(e.textBuffer)
	lines := strings.Split(text, "\n")
	
	// Ensure at least one line
	if len(lines) == 0 {
		return []string{""}
	}
	
	return lines
}
// SetTextFromLines sets the text buffer from lines
func (e *TUIEditor) SetTextFromLines(lines []string) {
	if len(lines) == 0 {
		e.textBuffer = []rune{}
		e.cursorPos = 0
		e.cursorLine = 0
		e.cursorCol = 0
		return
	}
	
	text := strings.Join(lines, "\n")
	e.textBuffer = []rune(text)
	e.cursorPos = len(e.textBuffer)
	e.updateCursorPosition()
}
// GetCursorInfo returns current cursor position info
func (e *TUIEditor) GetCursorInfo() (line, col int, lines []string) {
	return e.cursorLine, e.cursorCol, e.GetTextAsLines()
}
// IsMultilineEditKey checks if we should insert a newline (for future: Shift+Enter support)
func (e *TUIEditor) IsMultilineEditKey(key rune) bool {
	// For now, we'll use Alt+Enter (key code 30) or Ctrl+J (key code 10 with modifier)
	// In the future, we could detect Shift+Enter if the terminal supports it
	return false // Disabled for now - use explicit newline key binding
}
// HandleNewlineKey explicitly handles newline insertion
func (e *TUIEditor) HandleNewlineKey() {
	if e.mode == ModeEdit || e.mode == ModeInsert {
		e.insertNewline()
	}
}

// ============================================
// Methods from operations.go
// ============================================

// GetMode returns the current mode
func (e *TUIEditor) GetMode() Mode {
	return e.mode
}
// GetEddFrame returns Ed's current animation frame
func (e *TUIEditor) GetEddFrame() string {
	return e.edd.GetFrame(e.mode)
}
// GetJumpLabels returns the current jump labels
func (e *TUIEditor) GetJumpLabels() map[int]rune {
	return e.jumpLabels
}
// GetConnectionLabels returns the current connection jump labels
func (e *TUIEditor) GetConnectionLabels() map[int]rune {
	return e.connectionLabels
}
// GetJumpAction returns the current jump action
func (e *TUIEditor) GetJumpAction() JumpAction {
	return e.jumpAction
}
// GetSelectedNode returns the currently selected node ID
func (e *TUIEditor) GetSelectedNode() int {
	return e.selected
}
// GetNodePositions returns the last rendered node positions
func (e *TUIEditor) GetNodePositions() map[int]diagram.Point {
	return e.nodePositions
}
// GetConnectionPaths returns the last rendered connection paths
func (e *TUIEditor) GetConnectionPaths() map[int]diagram.Path {
	return e.connectionPaths
}
// GetTextBuffer returns the current text buffer (for display purposes)
func (e *TUIEditor) GetTextBuffer() []rune {
	return e.textBuffer
}
// IsContinuousConnect returns whether we're in continuous connection mode
func (e *TUIEditor) IsContinuousConnect() bool {
	return e.continuousConnect
}
// IsContinuousDelete returns whether we're in continuous delete mode
func (e *TUIEditor) IsContinuousDelete() bool {
	return e.continuousDelete
}
// StartAddNode begins adding a new node
func (e *TUIEditor) StartAddNode() {
	e.SetMode(ModeInsert)
	nodeID := e.AddNode([]string{""})
	e.selected = nodeID
	e.textBuffer = []rune{}
	e.cursorPos = 0
}
// StartConnect begins connection mode (single connection)
func (e *TUIEditor) StartConnect() {
	if len(e.diagram.Nodes) >= 2 {
		e.continuousConnect = false
		e.startJump(JumpActionConnectFrom)
	}
}
// StartContinuousConnect begins continuous connection mode (multiple connections)
func (e *TUIEditor) StartContinuousConnect() {
	if len(e.diagram.Nodes) >= 2 {
		e.continuousConnect = true
		e.startJump(JumpActionConnectFrom)
	}
}
// StartDelete begins delete mode (single deletion)
func (e *TUIEditor) StartDelete() {
	if len(e.diagram.Nodes) > 0 || len(e.diagram.Connections) > 0 {
		e.continuousDelete = false
		e.startJump(JumpActionDelete)
	}
}
// StartContinuousDelete begins continuous delete mode (multiple deletions)
func (e *TUIEditor) StartContinuousDelete() {
	if len(e.diagram.Nodes) > 0 || len(e.diagram.Connections) > 0 {
		e.continuousDelete = true
		e.startJump(JumpActionDelete)
	}
}
// StartEdit begins edit mode
func (e *TUIEditor) StartEdit() {
	if len(e.diagram.Nodes) > 0 {
		e.startJump(JumpActionEdit)
	}
}
// StartCommand enters command mode
func (e *TUIEditor) StartCommand() {
	e.SetMode(ModeCommand)
	e.commandBuffer = []rune{}
}
// StartHintEdit starts hint editing mode for nodes and connections
func (e *TUIEditor) StartHintEdit() {
	if len(e.diagram.Nodes) > 0 || len(e.diagram.Connections) > 0 {
		e.startJump(JumpActionHint)
	}
}
// HandleTextInput processes text input in insert/edit modes
func (e *TUIEditor) HandleTextInput(key rune) {
	// Delegate to the actual text handler
	e.handleTextKey(key)
}
// ToggleDiagramType switches between sequence and box diagram types
func (e *TUIEditor) ToggleDiagramType() {
	e.history.SaveState(e.diagram)
	currentType := e.diagram.Type
	if currentType == "" {
		currentType = "box"
	}
	
	if currentType == "sequence" {
		e.diagram.Type = "box"
	} else {
		e.diagram.Type = "sequence"
	}
}
// HandleJumpInput processes jump label selection for both nodes and connections
func (e *TUIEditor) HandleJumpInput(key rune) {
	// This is the public method that should handle both nodes and connections
	// It delegates to the internal handleJumpKey which has the full logic
	e.handleJumpKey(key)
}
// HandleJSONInput processes JSON view mode input
func (e *TUIEditor) HandleJSONInput(key rune) {
	// Delegate to the internal handler
	e.handleJSONKey(key)
}
// HandleCommandInput processes command mode input
func (e *TUIEditor) HandleCommandInput(key rune) {
	switch key {
	case 27: // ESC
		e.SetMode(ModeNormal)
	case 127, 8: // Backspace
		if len(e.commandBuffer) > 0 {
			e.commandBuffer = e.commandBuffer[:len(e.commandBuffer)-1]
		}
	default:
		if unicode.IsPrint(key) {
			e.commandBuffer = append(e.commandBuffer, key)
		}
	}
}
// GetCommand returns the current command buffer
func (e *TUIEditor) GetCommand() string {
	return string(e.commandBuffer)
}
// ClearCommand clears the command buffer
func (e *TUIEditor) ClearCommand() {
	e.commandBuffer = []rune{}
}
// AnimateEd advances Ed's animation
func (e *TUIEditor) AnimateEd() {
	e.edd.NextFrame()
}
// GetNodeCount returns the number of nodes
func (e *TUIEditor) GetNodeCount() int {
	return len(e.diagram.Nodes)
}
// GetConnectionCount returns the number of connections
func (e *TUIEditor) GetConnectionCount() int {
	return len(e.diagram.Connections)
}

// ============================================
// Methods from node_hints.go
// ============================================

// Available node styles in cycle order
var nodeStyles = []string{"rounded", "sharp", "double", "thick"}

// Available node colors in cycle order  
var nodeColors = []string{"", "red", "green", "yellow", "blue", "magenta", "cyan"}

// cycleNodeStyle cycles through available node styles
func (e *TUIEditor) cycleNodeStyle(nodeID int) {
	// Find the node
	var node *diagram.Node
	for i := range e.diagram.Nodes {
		if e.diagram.Nodes[i].ID == nodeID {
			node = &e.diagram.Nodes[i]
			break
		}
	}
	
	if node == nil {
		return
	}
	
	// Save state for undo
	e.history.SaveState(e.diagram)
	
	// Initialize hints if needed
	if node.Hints == nil {
		node.Hints = make(map[string]string)
	}
	
	// Get current style
	currentStyle := node.Hints["style"]
	
	// Find current index
	currentIndex := -1
	for i, style := range nodeStyles {
		if style == currentStyle {
			currentIndex = i
			break
		}
	}
	
	// Cycle to next style
	nextIndex := (currentIndex + 1) % len(nodeStyles)
	node.Hints["style"] = nodeStyles[nextIndex]
	
	// If it's the default (first) style and no other hints, remove the hints map
	if nodeStyles[nextIndex] == nodeStyles[0] && len(node.Hints) == 1 {
		delete(node.Hints, "style")
		if len(node.Hints) == 0 {
			node.Hints = nil
		}
	}
}
// cycleNodeColor cycles through available node colors
func (e *TUIEditor) cycleNodeColor(nodeID int) {
	// Find the node
	var node *diagram.Node
	for i := range e.diagram.Nodes {
		if e.diagram.Nodes[i].ID == nodeID {
			node = &e.diagram.Nodes[i]
			break
		}
	}
	
	if node == nil {
		return
	}
	
	// Save state for undo
	e.history.SaveState(e.diagram)
	
	// Initialize hints if needed
	if node.Hints == nil {
		node.Hints = make(map[string]string)
	}
	
	// Get current color
	currentColor := node.Hints["color"]
	
	// Find current index
	currentIndex := 0 // Default to no color (empty string)
	for i, color := range nodeColors {
		if color == currentColor {
			currentIndex = i
			break
		}
	}
	
	// Cycle to next color
	nextIndex := (currentIndex + 1) % len(nodeColors)
	
	if nodeColors[nextIndex] == "" {
		// Remove color hint
		delete(node.Hints, "color")
		if len(node.Hints) == 0 {
			node.Hints = nil
		}
	} else {
		node.Hints["color"] = nodeColors[nextIndex]
	}
}
// getNodeStyle returns the style hint for a node
func (e *TUIEditor) getNodeStyle(nodeID int) string {
	for _, node := range e.diagram.Nodes {
		if node.ID == nodeID {
			if node.Hints != nil {
				return node.Hints["style"]
			}
			return ""
		}
	}
	return ""
}
// getNodeColor returns the color hint for a node
func (e *TUIEditor) getNodeColor(nodeID int) string {
	for _, node := range e.diagram.Nodes {
		if node.ID == nodeID {
			if node.Hints != nil {
				return node.Hints["color"]
			}
			return ""
		}
	}
	return ""
}

// ============================================
// Methods from keys.go
// ============================================

// handleNormalKey processes keys in normal mode
func (e *TUIEditor) handleNormalKey(key rune) bool {
	// Debug logging
	if f, err := os.OpenFile("/tmp/edd_keys.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		fmt.Fprintf(f, "Normal mode key pressed: %c (%d), current diagram type: %s\n", key, key, e.diagram.Type)
		f.Close()
	}
	
	switch key {
	case 'q', 3: // q or Ctrl+C to quit
		return true
		
	case 27: // ESC - if we have a previous jump action, restart that jump mode
		if e.previousJumpAction != 0 {
			action := e.previousJumpAction
			e.previousJumpAction = 0  // Clear it
			e.startJump(action)  // Restart jump mode with the same action
		}
		
	case 'a', 'A': // Add new node (A is same as a since we're already in INSERT mode with continuation)
		e.previousJumpAction = 0  // Clear any previous jump action
		e.SetMode(ModeInsert)
		nodeID := e.AddNode([]string{""})
		e.selected = nodeID
		
	case 'c': // Connect nodes (single)
		if len(e.diagram.Nodes) >= 2 {
			e.continuousConnect = false
			e.selected = -1 // Clear any previous selection
			e.startJump(JumpActionConnectFrom)
		}
		
	case 'C': // Connect nodes (continuous)
		if len(e.diagram.Nodes) >= 2 {
			e.continuousConnect = true
			e.selected = -1 // Clear any previous selection
			e.startJump(JumpActionConnectFrom)
		}
		
	case 'd': // Delete node/connection (single)
		if len(e.diagram.Nodes) > 0 || len(e.diagram.Connections) > 0 {
			e.continuousDelete = false
			e.startJump(JumpActionDelete)
		}
		
	case 'D': // Delete node/connection (continuous)
		if len(e.diagram.Nodes) > 0 || len(e.diagram.Connections) > 0 {
			e.continuousDelete = true
			e.startJump(JumpActionDelete)
		}
		
	case 'e': // Edit node
		if len(e.diagram.Nodes) > 0 {
			e.startJump(JumpActionEdit)
		}
		
	case 'E': // Edit in external editor
		// This will be handled by the main loop since it needs file system access
		return false
		
	case 'H': // Edit hints for nodes and connections
		if len(e.diagram.Nodes) > 0 || len(e.diagram.Connections) > 0 {
			e.startJump(JumpActionHint)
		}
		
	case 'j': // Toggle JSON view
		e.SetMode(ModeJSON)
		e.jsonScrollOffset = 0  // Reset scroll when entering JSON mode
		
	case 't': // Toggle diagram type
		// Debug log
		if f, err := os.OpenFile("/tmp/edd_keys.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			fmt.Fprintf(f, "t key matched! Current type: %s\n", e.diagram.Type)
			f.Close()
		}
		
		e.history.SaveState(e.diagram)  // Save current state for undo
		// Handle empty type as "box" (default)
		currentType := e.diagram.Type
		if currentType == "" {
			currentType = "box"
		}
		
		if currentType == "sequence" {
			e.diagram.Type = "box"
		} else {
			e.diagram.Type = "sequence"
		}
		
		// Debug log after change
		if f, err := os.OpenFile("/tmp/edd_keys.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			fmt.Fprintf(f, "After toggle, new type: %s\n", e.diagram.Type)
			f.Close()
		}
		
	case 'u': // Undo
		e.Undo()
		
	case 18: // Ctrl+R for redo
		e.Redo()
		
	case '?', 'h': // Help
		e.SetMode(ModeHelp)
		
	case ':': // Command mode
		e.SetMode(ModeCommand)
		e.commandBuffer = []rune{}  // Start with empty, : is shown in prompt
	}
	
	return false
}
// handleTextKey processes keys in text input modes (Insert/Edit)
func (e *TUIEditor) handleTextKey(key rune) bool {
	switch key {
	case 27: // ESC - save and return to normal mode (or jump mode if we came from there)
		e.commitText()
		if e.previousJumpAction != 0 {
			action := e.previousJumpAction
			e.previousJumpAction = 0  // Clear it
			e.startJump(action)  // Restart jump mode with the same action
		} else {
			e.SetMode(ModeNormal)
		}
		
	case 127, 8: // Backspace
		if e.cursorPos > 0 {
			e.textBuffer = append(
				e.textBuffer[:e.cursorPos-1],
				e.textBuffer[e.cursorPos:]...,
			)
			e.cursorPos--
			e.updateCursorPosition()
		}
		
	case 14: // Ctrl+N - insert newline for multi-line editing
		e.insertNewline()
	
	case 23: // Ctrl+W - delete word backward
		e.deleteWordBackward()
	
	case 21: // Ctrl+U - delete to beginning of line
		e.deleteToBeginningOfLine()
	
	case 11: // Ctrl+K - delete to end of line
		e.deleteToEndOfLine()
	
	case 1: // Ctrl+A - move to beginning of line
		e.moveCursorToBeginningOfLine()
	
	case 5: // Ctrl+E - move to end of line
		e.moveCursorToEndOfLine()
	
	case 6: // Ctrl+F - move forward one character
		e.moveCursorForward()
	
	case 2: // Ctrl+B - move backward one character
		e.moveCursorBackward()
	
	case 16: // Ctrl+P - move up one line (previous)
		e.moveCursorUp()
	
	case 22: // Ctrl+V - move down one line (since Ctrl+N is for newline)
		e.moveCursorDown()
		
	case 13, 10: // Enter - commit text
		// Save the mode before committing (in case commit changes it)
		wasInsertMode := e.mode == ModeInsert
		
		e.commitText()
		
		// In INSERT mode, immediately start adding another node
		if wasInsertMode {
			// Create a new node and continue in insert mode
			nodeID := e.AddNode([]string{""})
			e.selected = nodeID
			e.textBuffer = []rune{}
			e.cursorPos = 0
			// Stay in INSERT mode (don't call SetMode as it clears the buffer)
			// e.mode is already ModeInsert
		} else {
			// In EDIT mode, return to normal
			e.SetMode(ModeNormal)
		}
		
	default:
		// Insert printable characters
		if unicode.IsPrint(key) {
			e.textBuffer = append(
				e.textBuffer[:e.cursorPos],
				append([]rune{key}, e.textBuffer[e.cursorPos:]...)...,
			)
			e.cursorPos++
			e.updateCursorPosition()
		}
	}
	
	return false
}
// handleCommandKey processes keys in command mode
func (e *TUIEditor) handleCommandKey(key rune) bool {
	switch key {
	case 27: // ESC - cancel command
		e.SetMode(ModeNormal)
		
	case 127, 8: // Backspace
		if len(e.commandBuffer) > 0 {
			e.commandBuffer = e.commandBuffer[:len(e.commandBuffer)-1]
		}
		
	case 13, 10: // Enter - execute command
		e.executeCommand(string(e.commandBuffer))
		e.SetMode(ModeNormal)
		
	default:
		// Add to command buffer
		if unicode.IsPrint(key) {
			e.commandBuffer = append(e.commandBuffer, key)
		}
	}
	
	return false
}
// handleJumpKey processes keys when jump labels are active
func (e *TUIEditor) handleJumpKey(key rune) bool {
	// ESC cancels jump
	if key == 27 {
		e.clearJumpLabels()
		e.continuousConnect = false // Exit continuous connect mode
		e.continuousDelete = false  // Exit continuous delete mode
		e.previousJumpAction = 0    // Clear the previous action since we're canceling
		e.selected = -1            // Clear selected node
		e.SetMode(ModeNormal)
		return false
	}
	
	// Look for matching node jump label
	for nodeID, label := range e.jumpLabels {
		if label == key {
			// Found match - execute jump action
			e.executeJumpAction(nodeID)
			return false
		}
	}
	
	// Look for matching connection jump label (in delete, edit, or hint mode)
	if e.jumpAction == JumpActionDelete || e.jumpAction == JumpActionEdit || e.jumpAction == JumpActionHint {
		// Log to file
		if f, err := os.OpenFile("/tmp/edd_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			fmt.Fprintf(f, "\n[%s] Looking for key '%c' in connection labels: %v\n", time.Now().Format("15:04:05"), key, e.connectionLabels)
			f.Close()
		}
		for connIndex, label := range e.connectionLabels {
			if label == key {
				// Log to file
				if f, err := os.OpenFile("/tmp/edd_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
					fmt.Fprintf(f, "Found match! Connection index %d has label '%c'\n", connIndex, label)
					f.Close()
				}
				if e.jumpAction == JumpActionDelete {
					// Delete the connection
					e.DeleteConnection(connIndex)
					
					// If in continuous delete mode, start another delete
					if e.continuousDelete && (len(e.diagram.Nodes) > 0 || len(e.diagram.Connections) > 0) {
						e.clearJumpLabels()
						e.startJump(JumpActionDelete)
					} else {
						// Normal mode - exit to normal
						e.continuousDelete = false
						e.clearJumpLabels()
						e.SetMode(ModeNormal)
					}
				} else if e.jumpAction == JumpActionEdit {
					// Edit the connection label
					e.previousJumpAction = e.jumpAction  // Save the action for ESC handling
					e.StartEditingConnection(connIndex)
				} else if e.jumpAction == JumpActionHint {
					// Enter hint menu for this connection
					e.previousJumpAction = e.jumpAction  // Save the action for ESC handling
					e.editingHintConn = connIndex
					// Log to file
					if f, err := os.OpenFile("/tmp/edd_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
						fmt.Fprintf(f, "Selected connection index %d for hint editing (pressed '%c')\n", connIndex, key)
						f.Close()
					}
					e.clearJumpLabels()
					e.SetMode(ModeHintMenu)
				}
				return false
			}
		}
	}
	
	// No match - handle based on current state
	if e.continuousConnect {
		// In continuous connect mode - just ignore invalid keys
		// User can still press a valid label or ESC to cancel
		return false
	}
	if e.continuousDelete {
		// In continuous delete mode - just ignore invalid keys
		return false
	}
	
	// For single-action modes, cancel jump on invalid key
	e.clearJumpLabels()
	e.SetMode(ModeNormal)
	return false
}
// commitText saves the current text buffer to the selected node or connection
func (e *TUIEditor) commitText() {
	// Check if we're editing a connection
	if e.selectedConnection >= 0 {
		// Connection labels are single line only
		text := strings.TrimSpace(string(e.textBuffer))
		// Connection labels can be empty (to clear them)
		e.UpdateConnectionLabel(e.selectedConnection, text)
		e.selectedConnection = -1
		return
	}
	
	// Otherwise we're editing a node - support multi-line
	if e.selected < 0 {
		return
	}
	
	// Get text as lines for multi-line support
	lines := e.GetTextAsLines()
	
	// Trim empty lines at the end
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	
	// In INSERT mode, we allow empty text (user might just press Enter to create empty nodes)
	if len(lines) == 0 && e.mode != ModeInsert {
		return
	}
	
	// If completely empty, use a single empty line
	if len(lines) == 0 {
		lines = []string{""}
	}
	
	// Update the node text with multiple lines
	e.UpdateNodeText(e.selected, lines)
}
// executeCommand executes a command mode command
func (e *TUIEditor) executeCommand(cmd string) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}
	
	switch parts[0] {
	case "q", "quit":
		// TODO: Implement quit with save check
		
	case "w", "write", "save":
		// TODO: Implement save
		
	case "wq":
		// TODO: Save and quit
		
	case "load", "open":
		// TODO: Load diagram
		
	case "new":
		// Clear diagram
		e.diagram = &diagram.Diagram{}
		e.selected = -1
		
	case "type":
		// Change diagram type
		if len(parts) > 1 {
			switch parts[1] {
			case "sequence", "seq":
				e.diagram.Type = string(diagram.DiagramTypeSequence)
				e.SaveHistory()
			case "flowchart", "flow", "":
				e.diagram.Type = string(diagram.DiagramTypeFlowchart)  // Empty means flowchart
				e.SaveHistory()
			default:
				// Unknown type, ignore
			}
		}
	}
}
// executeJumpAction executes the pending action after jump selection
func (e *TUIEditor) executeJumpAction(nodeID int) {
	// Save the action type before executing
	e.previousJumpAction = e.jumpAction
	
	switch e.jumpAction {
	case JumpActionSelect:
		e.selected = nodeID
		e.clearJumpLabels()
		e.SetMode(ModeNormal)
		
	case JumpActionEdit:
		e.selected = nodeID
		e.clearJumpLabels()
		e.SetMode(ModeEdit)
		return // Don't reset to normal mode
		
	case JumpActionDelete:
		e.DeleteNode(nodeID)
		if e.selected == nodeID {
			e.selected = -1
		}
		
		// If in continuous delete mode, start another delete
		if e.continuousDelete && (len(e.diagram.Nodes) > 0 || len(e.diagram.Connections) > 0) {
			e.clearJumpLabels()
			e.startJump(JumpActionDelete)
		} else {
			// Normal mode - exit to normal
			e.continuousDelete = false
			e.clearJumpLabels()
			e.SetMode(ModeNormal)
		}
		
	case JumpActionConnectFrom:
		e.selected = nodeID
		// Start second jump for target
		e.startJump(JumpActionConnectTo)
		return // Don't clear jump labels yet
		
	case JumpActionConnectTo:
		if e.selected >= 0 {
			e.AddConnection(e.selected, nodeID, "")
		}
		
		// If in continuous connect mode, behavior depends on diagram type
		if e.continuousConnect {
			if e.diagram.Type == string(diagram.DiagramTypeSequence) {
				// Sequence diagram: chain connections (TO becomes next FROM)
				e.selected = nodeID
				// Jump directly to selecting the next TO node
				e.startJump(JumpActionConnectTo)
			} else {
				// Flowchart: start fresh connection (select new FROM)
				e.selected = -1
				e.startJump(JumpActionConnectFrom)
			}
		} else {
			// Normal mode - exit to normal
			e.selected = -1
			e.clearJumpLabels()
			e.SetMode(ModeNormal)
		}
		
	case JumpActionHint:
		// Enter hint menu for this node
		e.editingHintNode = nodeID
		e.clearJumpLabels()
		e.SetMode(ModeHintMenu)
	}
}
// handleHelpKey processes keys in help mode
func (e *TUIEditor) handleHelpKey(key rune) bool {
	// Any key exits help mode
	e.SetMode(ModeNormal)
	return false
}
// handleJSONKey processes keys in JSON view mode
func (e *TUIEditor) handleJSONKey(key rune) bool {
	switch key {
	case 27, 'q', 'j': // ESC, q, or j to return to diagram view
		e.SetMode(ModeNormal)
		
	case 'E': // Edit in external editor (also works from JSON view)
		// This will be handled by the main loop
		return false
		
	case 'k', 'K': // vim-style up
		e.ScrollJSON(-1)
		
	case 'J': // vim-style down (capital J since j exits)
		e.ScrollJSON(1)
		
	case 'u', 21: // Page up (Ctrl+U in vim)
		e.ScrollJSON(-(e.height / 2))
		
	case 'd', 4: // Page down (Ctrl+D in vim)
		e.ScrollJSON(e.height / 2)
		
	case 'g': // Go to top
		e.jsonScrollOffset = 0
		
	case 'G': // Go to bottom
		e.jsonScrollOffset = 999999 // Will be clamped in renderJSON
	}
	
	return false
}

// ============================================
// Methods from hints.go
// ============================================

// HandleHintMenuInput processes input in hint menu mode
func (e *TUIEditor) HandleHintMenuInput(key rune) {
	// Check if we're editing a node or connection
	if e.editingHintNode >= 0 {
		e.handleNodeHintInput(key)
	} else if e.editingHintConn >= 0 {
		e.handleConnectionHintInput(key)
	} else {
		// Nothing selected, exit
		e.SetMode(ModeNormal)
	}
}
// handleNodeHintInput handles hint menu input for nodes
func (e *TUIEditor) handleNodeHintInput(key rune) {
	// Find the node
	var node *diagram.Node
	for i := range e.diagram.Nodes {
		if e.diagram.Nodes[i].ID == e.editingHintNode {
			node = &e.diagram.Nodes[i]
			break
		}
	}
	
	if node == nil {
		e.editingHintNode = -1
		e.SetMode(ModeNormal)
		return
	}
	
	// Initialize hints map if needed
	if node.Hints == nil {
		node.Hints = make(map[string]string)
	}
	
	isSequence := e.diagram.Type == string(diagram.DiagramTypeSequence)
	
	switch key {
	// Style options for nodes/boxes
	case 'a': // Rounded (default) for flowcharts, Box style for sequence
		if !isSequence {
			node.Hints["style"] = "rounded"
		} else {
			node.Hints["box-style"] = "rounded"
		}
		e.SaveHistory()
	case 'b': // Sharp for flowcharts, Sharp box for sequence
		if !isSequence {
			node.Hints["style"] = "sharp"
		} else {
			node.Hints["box-style"] = "sharp"
		}
		e.SaveHistory()
	case 'c': // Double for flowcharts, Double box for sequence
		if !isSequence {
			node.Hints["style"] = "double"
		} else {
			node.Hints["box-style"] = "double"
		}
		e.SaveHistory()
	case 'd': // Thick for flowcharts, Thick box for sequence
		if !isSequence {
			node.Hints["style"] = "thick"
		} else {
			node.Hints["box-style"] = "thick"
		}
		e.SaveHistory()
		
	// Color options
	case 'r': // Red
		node.Hints["color"] = "red"
		e.SaveHistory()
	case 'g': // Green
		node.Hints["color"] = "green"
		e.SaveHistory()
	case 'y': // Yellow
		node.Hints["color"] = "yellow"
		e.SaveHistory()
	case 'u': // Blue
		node.Hints["color"] = "blue"
		e.SaveHistory()
	case 'm': // Magenta
		node.Hints["color"] = "magenta"
		e.SaveHistory()
	case 'n': // Cyan
		node.Hints["color"] = "cyan"
		e.SaveHistory()
	case 'w': // Default (no color)
		delete(node.Hints, "color")
		e.SaveHistory()
		
	// Text style options (only for flowcharts)
	case 'o': // Toggle bold
		if !isSequence {
			if node.Hints["bold"] == "true" {
				delete(node.Hints, "bold")
			} else {
				node.Hints["bold"] = "true"
			}
			e.SaveHistory()
		}
	case 'i': // Toggle italic
		if !isSequence {
			if node.Hints["italic"] == "true" {
				delete(node.Hints, "italic")
			} else {
				node.Hints["italic"] = "true"
			}
			e.SaveHistory()
		}
	case 't': // Toggle text alignment (center/left)
		if !isSequence {
			if node.Hints["text-align"] == "center" {
				delete(node.Hints, "text-align") // Back to default (left)
			} else {
				node.Hints["text-align"] = "center"
			}
			e.SaveHistory()
		}
		
	// Shadow options (only for flowcharts)
	case 'z': // Shadow southeast
		if !isSequence {
			node.Hints["shadow"] = "southeast"
			if node.Hints["shadow-density"] == "" {
				node.Hints["shadow-density"] = "light"
			}
			e.SaveHistory()
		}
	case 'x': // No shadow
		if !isSequence {
			delete(node.Hints, "shadow")
			delete(node.Hints, "shadow-density")
			e.SaveHistory()
		}
	case 'l': // Shadow density for flowcharts only
		if !isSequence {
			if node.Hints["shadow-density"] == "light" {
				node.Hints["shadow-density"] = "medium"
			} else {
				node.Hints["shadow-density"] = "light"
			}
			e.SaveHistory()
		}
	
	// Lifeline style options (uppercase for sequence diagrams)
	case 'A': // Solid lifeline (default)
		if isSequence {
			delete(node.Hints, "lifeline-style") // Remove to use default (solid)
			e.SaveHistory()
		}
	case 'B': // Dashed lifeline
		if isSequence {
			node.Hints["lifeline-style"] = "dashed"
			e.SaveHistory()
		}
	case 'C': // Dotted lifeline
		if isSequence {
			node.Hints["lifeline-style"] = "dotted"
			e.SaveHistory()
		}
	case 'D': // Double lifeline
		if isSequence {
			node.Hints["lifeline-style"] = "double"
			e.SaveHistory()
		}
	
	// Lifeline color options (uppercase for sequence diagrams)
	case 'R': // Red lifeline
		if isSequence {
			node.Hints["lifeline-color"] = "red"
			e.SaveHistory()
		}
	case 'G': // Green lifeline
		if isSequence {
			node.Hints["lifeline-color"] = "green"
			e.SaveHistory()
		}
	case 'Y': // Yellow lifeline
		if isSequence {
			node.Hints["lifeline-color"] = "yellow"
			e.SaveHistory()
		}
	case 'U': // Blue lifeline
		if isSequence {
			node.Hints["lifeline-color"] = "blue"
			e.SaveHistory()
		}
	case 'M': // Magenta lifeline
		if isSequence {
			node.Hints["lifeline-color"] = "magenta"
			e.SaveHistory()
		}
	case 'N': // Cyan lifeline
		if isSequence {
			node.Hints["lifeline-color"] = "cyan"
			e.SaveHistory()
		}
	case 'W': // Default lifeline color (no color)
		if isSequence {
			delete(node.Hints, "lifeline-color")
			e.SaveHistory()
		}
		
	// Layout position hints (only for flowcharts)
	case '1': // Top-left
		if !isSequence {
			node.Hints["position"] = "top-left"
			e.SaveHistory()
		}
	case '2': // Top-center
		if !isSequence {
			node.Hints["position"] = "top-center"
			e.SaveHistory()
		}
	case '3': // Top-right
		if !isSequence {
			node.Hints["position"] = "top-right"
			e.SaveHistory()
		}
	case '4': // Middle-left
		if !isSequence {
			node.Hints["position"] = "middle-left"
			e.SaveHistory()
		}
	case '5': // Center
		if !isSequence {
			node.Hints["position"] = "center"
			e.SaveHistory()
		}
	case '6': // Middle-right
		if !isSequence {
			node.Hints["position"] = "middle-right"
			e.SaveHistory()
		}
	case '7': // Bottom-left
		if !isSequence {
			node.Hints["position"] = "bottom-left"
			e.SaveHistory()
		}
	case '8': // Bottom-center
		if !isSequence {
			node.Hints["position"] = "bottom-center"
			e.SaveHistory()
		}
	case '9': // Bottom-right
		if !isSequence {
			node.Hints["position"] = "bottom-right"
			e.SaveHistory()
		}
	case '0': // Clear position hint
		if !isSequence {
			delete(node.Hints, "position")
			e.SaveHistory()
		}
		
	case 27: // ESC - exit to normal mode or back to jump mode
		e.editingHintNode = -1
		if e.previousJumpAction != 0 {
			action := e.previousJumpAction
			e.previousJumpAction = 0  // Clear it
			e.startJump(action)  // Restart jump mode with the same action
		} else {
			e.SetMode(ModeNormal)
		}
	case 13, 10: // Enter - exit to normal mode
		e.editingHintNode = -1
		e.previousJumpAction = 0  // Clear the previous action
		e.SetMode(ModeNormal)
	}
}
// handleConnectionHintInput handles hint menu input for connections
func (e *TUIEditor) handleConnectionHintInput(key rune) {
	if e.editingHintConn < 0 || e.editingHintConn >= len(e.diagram.Connections) {
		// Invalid connection, exit
		e.editingHintConn = -1
		e.SetMode(ModeNormal)
		return
	}
	
	conn := &e.diagram.Connections[e.editingHintConn]
	
	// Initialize hints map if needed
	if conn.Hints == nil {
		conn.Hints = make(map[string]string)
	}
	
	isSequence := e.diagram.Type == string(diagram.DiagramTypeSequence)
	
	switch key {
	// Style options for connections
	case 'a': // Solid (default)
		delete(conn.Hints, "style") // Remove to use default
		e.SaveHistory()
	case 'b': // Dashed
		conn.Hints["style"] = "dashed"
		e.SaveHistory()
	case 'c': // Dotted
		conn.Hints["style"] = "dotted"
		e.SaveHistory()
	case 'd': // Double (only for flowcharts)
		if !isSequence {
			conn.Hints["style"] = "double"
			e.SaveHistory()
		}
		
	// Color options
	case 'r': // Red
		conn.Hints["color"] = "red"
		e.SaveHistory()
	case 'g': // Green
		conn.Hints["color"] = "green"
		e.SaveHistory()
	case 'y': // Yellow
		conn.Hints["color"] = "yellow"
		e.SaveHistory()
	case 'u': // Blue
		conn.Hints["color"] = "blue"
		e.SaveHistory()
	case 'm': // Magenta
		conn.Hints["color"] = "magenta"
		e.SaveHistory()
	case 'n': // Cyan
		conn.Hints["color"] = "cyan"
		e.SaveHistory()
	case 'w': // White/default
		delete(conn.Hints, "color") // Remove to use default
		e.SaveHistory()
		
	// Text style options
	case 'o': // Toggle bold
		if conn.Hints["bold"] == "true" {
			delete(conn.Hints, "bold")
		} else {
			conn.Hints["bold"] = "true"
		}
		e.SaveHistory()
	case 'i': // Toggle italic
		if conn.Hints["italic"] == "true" {
			delete(conn.Hints, "italic")
		} else {
			conn.Hints["italic"] = "true"
		}
		e.SaveHistory()
		
	// Flow direction hints (only for flowcharts)
	case 'f': // Cycle through flow directions
		if !isSequence {
			currentFlow := conn.Hints["flow"]
			switch currentFlow {
			case "right":
				conn.Hints["flow"] = "down"
			case "down":
				conn.Hints["flow"] = "left"
			case "left":
				conn.Hints["flow"] = "up"
			case "up":
				delete(conn.Hints, "flow") // Remove to go back to auto
			default:
				conn.Hints["flow"] = "right" // Start with right
			}
			e.SaveHistory()
		}
		
	case 27: // ESC - exit to normal mode or back to jump mode
		e.editingHintConn = -1
		if e.previousJumpAction != 0 {
			action := e.previousJumpAction
			e.previousJumpAction = 0  // Clear it
			e.startJump(action)  // Restart jump mode with the same action
		} else {
			e.SetMode(ModeNormal)
		}
	case 13, 10: // Enter - exit to normal mode
		e.editingHintConn = -1
		e.previousJumpAction = 0  // Clear the previous action
		e.SetMode(ModeNormal)
	}
}
// GetHintMenuDisplay returns the display string for hint menu
func (e *TUIEditor) GetHintMenuDisplay() string {
	if e.editingHintNode >= 0 {
		return e.getNodeHintMenuDisplay()
	} else if e.editingHintConn >= 0 {
		return e.getConnectionHintMenuDisplay()
	}
	return ""
}
// getNodeHintMenuDisplay returns the hint menu display for a node
func (e *TUIEditor) getNodeHintMenuDisplay() string {
	// Find the node
	var node *diagram.Node
	for i := range e.diagram.Nodes {
		if e.diagram.Nodes[i].ID == e.editingHintNode {
			node = &e.diagram.Nodes[i]
			break
		}
	}
	
	if node == nil {
		return ""
	}
	
	// Get current style, color, and shadow
	style := "rounded"
	if s, ok := node.Hints["style"]; ok {
		style = s
	}
	
	color := "default"
	if c, ok := node.Hints["color"]; ok {
		color = c
	}
	
	bold := "off"
	if b, ok := node.Hints["bold"]; ok && b == "true" {
		bold = "on"
	}
	
	italic := "off"
	if i, ok := node.Hints["italic"]; ok && i == "true" {
		italic = "on"
	}
	
	textAlign := "left"
	if a, ok := node.Hints["text-align"]; ok && a == "center" {
		textAlign = "center"
	}
	
	// Get node text
	nodeText := "Node"
	if len(node.Text) > 0 {
		nodeText = node.Text[0]
		if len(nodeText) > 20 {
			nodeText = nodeText[:20] + "..."
		}
	}
	
	// Different menu for sequence diagrams
	if e.diagram.Type == string(diagram.DiagramTypeSequence) {
		boxStyle := "rounded"
		if s, ok := node.Hints["box-style"]; ok {
			boxStyle = s
		}
		
		lifelineStyle := "solid"
		if s, ok := node.Hints["lifeline-style"]; ok {
			lifelineStyle = s
		}
		
		lifelineColor := "default"
		if c, ok := node.Hints["lifeline-color"]; ok {
			lifelineColor = c
		}
		
		return "\n" +
			"Participant: " + nodeText + " | box=" + boxStyle + "/" + color + " | lifeline=" + lifelineStyle + "/" + lifelineColor + "\n" +
			"Box: [a]Round [b]Sharp [c]Double [d]Thick | [r]Red [g]Green [y]Yellow [u]Blue [m]Magenta [n]Cyan [w]Clear\n" +
			"Line: [A]Solid [B]Dash [C]Dot [D]Double | [R]Red [G]Green [Y]Yellow [U]Blue [M]Magenta [N]Cyan [W]Clear\n" +
			"[ESC]Back [Enter]Done"
	}
	
	// Full menu for flowcharts
	return "\n" +
		"Node: " + nodeText + " | style=" + style + ", color=" + color + "\n" +
		"Style: [a]Rounded [b]Sharp [c]Double [d]Thick | " +
		"Color: [r]Red [g]Green [y]Yellow [u]Blue [m]Magenta [n]Cyan [w]Clear\n" +
		"Text: [o]Bold(" + bold + ") [i]Italic(" + italic + ") [t]Center(" + textAlign + ") | " +
		"Shadow: [z]Add [x]Remove [l]Density\n" +
		"Position: [1-9]Grid [0]Auto | [ESC]Back [Enter]Done"
}
// getConnectionHintMenuDisplay returns the hint menu display for a connection
func (e *TUIEditor) getConnectionHintMenuDisplay() string {
	if e.editingHintConn < 0 || e.editingHintConn >= len(e.diagram.Connections) {
		return ""
	}
	
	conn := &e.diagram.Connections[e.editingHintConn]
	
	// Get current style, color, and bold
	style := "solid"
	if s, ok := conn.Hints["style"]; ok {
		style = s
	}
	
	color := "default"
	if c, ok := conn.Hints["color"]; ok {
		color = c
	}
	
	bold := "off"
	if b, ok := conn.Hints["bold"]; ok && b == "true" {
		bold = "on"
	}
	
	italic := "off"
	if i, ok := conn.Hints["italic"]; ok && i == "true" {
		italic = "on"
	}
	
	flow := "auto"
	if f, ok := conn.Hints["flow"]; ok {
		flow = f
	}
	
	// Find connection info
	var fromText, toText string
	for _, node := range e.diagram.Nodes {
		if node.ID == conn.From && len(node.Text) > 0 {
			fromText = node.Text[0]
			if len(fromText) > 10 {
				fromText = fromText[:10] + "..."
			}
		}
		if node.ID == conn.To && len(node.Text) > 0 {
			toText = node.Text[0]
			if len(toText) > 10 {
				toText = toText[:10] + "..."
			}
		}
	}
	
	// Different menu for sequence diagrams
	if e.diagram.Type == string(diagram.DiagramTypeSequence) {
		return "\n" +
			"Message: " + fromText + " → " + toText + " | style=" + style + ", color=" + color + "\n" +
			"Style: [a]Solid [b]Dashed [c]Dotted | Color: [r]Red [g]Green [y]Yellow [u]Blue [m]Magenta [n]Cyan [w]Clear\n" +
			"Text: [o]Bold(" + bold + ") [i]Italic(" + italic + ") | [ESC]Back [Enter]Done"
	}
	
	// Full menu for flowcharts
	return "\n" +
		"Connection: " + fromText + " → " + toText + " | style=" + style + ", color=" + color + "\n" +
		"Style: [a]Solid [b]Dashed [c]Dotted [d]Double | " +
		"Color: [r]Red [g]Green [y]Yellow [u]Blue [m]Magenta [n]Cyan [w]Clear\n" +
		"Options: [o]Bold(" + bold + ") [i]Italic(" + italic + ") [f]Flow(" + flow + ") | [ESC]Back [Enter]Done"
}

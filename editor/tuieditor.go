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
	selected           int          // Currently selected node ID (-1 for none)
	selectedConnection int          // Currently selected connection index (-1 for none)
	jumpLabels         map[int]rune // Node ID -> jump label mapping
	connectionLabels   map[int]rune // Connection index -> jump label mapping
	insertionLabels    map[int]rune // Insertion point index -> jump label mapping
	jumpAction         JumpAction   // What to do after jump selection
	insertionPoint     int          // Where to insert new connection (for splice mode)
	continuousConnect  bool         // Whether to continue connecting after each connection
	continuousDelete   bool         // Whether to continue deleting after each deletion
	editingHintConn    int          // Connection being edited for hints (-1 for none)
	editingHintNode    int          // Node being edited for hints (-1 for none)
	previousJumpAction JumpAction   // Remember the jump action for ESC handling

	// Activation mode state
	activationStartConn int         // Connection where activation starts (-1 for none)
	activationStartFrom int         // Participant ID that will be activated

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
	nodePositions   map[int]diagram.Point // Node ID -> position from last render
	connectionPaths map[int]diagram.Path  // Connection index -> path from last render

	// JSON view state
	jsonScrollOffset int // Current scroll position in JSON view

	// Diagram view state
	diagramScrollOffset int  // Current vertical scroll position in diagram view
	diagramChanged      bool // Track if diagram was modified since last render

	// History management
	history *StructHistory // Undo/redo history (optimized struct-based)

	// Test-only field (not used in production)
	connectFrom int // Used by test helpers for connection tracking

	// Command mode results
	commandResult   string // Result message from last command
	exportFormat    string // Export format requested
	exportFilename  string // Export filename requested
	saveRequested   bool   // Save was requested
	saveFilename    string // Filename for save (optional)
	quitRequested   bool   // Quit was requested
	hasChanges      bool   // Track unsaved changes
}

// NewTUIEditor creates a new TUI editor instance
func NewTUIEditor(renderer DiagramRenderer) *TUIEditor {
	editor := &TUIEditor{
		diagram:             &diagram.Diagram{Type: "box"}, // Default to box diagram
		renderer:            renderer,
		mode:                ModeNormal,
		selected:            -1,
		selectedConnection:  -1,
		jumpLabels:          make(map[int]rune),
		connectionLabels:    make(map[int]rune),
		activationStartConn: -1,
		activationStartFrom: -1,
		textBuffer:          []rune{},
		commandBuffer:       []rune{},
		cursorPos:           0,
		edd:                 NewEddCharacter(),
		width:               80,
		height:              24,
		nodePositions:       make(map[int]diagram.Point),
		connectionPaths:     make(map[int]diagram.Path),
		continuousConnect:   false,
		continuousDelete:    false,
		editingHintConn:     -1, // Initialize to -1 (no connection being edited)
		editingHintNode:     -1, // Initialize to -1 (no node being edited)
		jsonScrollOffset:    0,
		diagramScrollOffset: 0,                    // Initialize diagram scroll offset
		history:             NewStructHistory(500), // 500 states max for extensive editing
		connectFrom:         -1,                   // Initialize test-only field
	}

	// Save initial empty state
	editor.history.SaveState(editor.diagram)

	return editor
}

// SetDiagram sets the diagram to edit
func (e *TUIEditor) SetDiagram(d *diagram.Diagram) {
	e.diagram = d
	// Don't auto-scroll when loading a diagram - start at the top
	e.diagramScrollOffset = 0
	e.diagramChanged = false
	// Save this as a new state in history
	e.history.SaveState(d)
}

// GetDiagram returns the current diagram
func (e *TUIEditor) GetDiagram() *diagram.Diagram {
	return e.diagram
}

// GetCommandResult returns and clears the command result
func (e *TUIEditor) GetCommandResult() string {
	result := e.commandResult
	e.commandResult = ""
	return result
}

// SetCommandResult sets the command result message
func (e *TUIEditor) SetCommandResult(result string) {
	e.commandResult = result
}

// GetExportRequest returns and clears any export request
func (e *TUIEditor) GetExportRequest() (format, filename string) {
	format = e.exportFormat
	filename = e.exportFilename
	e.exportFormat = ""
	e.exportFilename = ""
	return format, filename
}

// GetSaveRequest returns and clears any save request
func (e *TUIEditor) GetSaveRequest() (bool, string) {
	requested := e.saveRequested
	filename := e.saveFilename
	e.saveRequested = false
	e.saveFilename = ""
	return requested, filename
}

// GetQuitRequest returns and clears any quit request
func (e *TUIEditor) GetQuitRequest() bool {
	requested := e.quitRequested
	e.quitRequested = false
	return requested
}

// SetHasChanges sets the hasChanges flag
func (e *TUIEditor) SetHasChanges(changed bool) {
	e.hasChanges = changed
}

// SetTerminalSize updates the terminal dimensions
func (e *TUIEditor) SetTerminalSize(width, height int) {
	e.width = width
	e.height = height
}

// GetTerminalHeight returns the terminal height
func (e *TUIEditor) GetTerminalHeight() int {
	return e.height
}

// ScrollDiagram scrolls the diagram view by the given amount
func (e *TUIEditor) ScrollDiagram(delta int) {
	e.diagramScrollOffset += delta
	if e.diagramScrollOffset < 0 {
		e.diagramScrollOffset = 0
	}

	// If we're in jump mode, reassign labels for the new viewport
	if e.mode == ModeJump {
		e.assignJumpLabels()
	}
	// Note: Maximum scroll limit is handled in Render() based on actual content size
}

// ScrollToTop scrolls the diagram view to the top
func (e *TUIEditor) ScrollToTop() {
	e.diagramScrollOffset = 0

	// If we're in jump mode, reassign labels for the new viewport
	if e.mode == ModeJump {
		e.assignJumpLabels()
	}
}

// ScrollToBottom scrolls the diagram view to the bottom
func (e *TUIEditor) ScrollToBottom() {
	// Set a flag to scroll to bottom on next render
	// This is used when new content is added to show it immediately
	e.diagramChanged = true

	// Labels will be reassigned on next render when scroll position is updated
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
	// Debug: Print a marker to see if we're being called recursively
	// fmt.Fprintf(os.Stderr, "DEBUG: Render() called\n")

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
		// Set edit state based on what we're editing
		if e.mode == ModeEdit || e.mode == ModeInsert {
			if e.selectedConnection >= 0 {
				// Editing a connection label
				realRenderer.SetEditState(-1, "", 0) // Clear node edit state
				realRenderer.SetConnectionEditState(e.selectedConnection, string(e.textBuffer), e.cursorPos)
			} else {
				// Editing a node
				realRenderer.SetEditState(e.selected, string(e.textBuffer), e.cursorPos)
				realRenderer.SetConnectionEditState(-1, "", 0) // Clear connection edit state
			}
		} else {
			// Not editing anything
			realRenderer.SetEditState(-1, "", 0)
			realRenderer.SetConnectionEditState(-1, "", 0)
		}

		positions, output, err := realRenderer.RenderWithPositions(e.diagram)
		if err == nil && positions != nil {
			// Store node positions and connection paths for jump label rendering
			e.nodePositions = positions.Positions
			e.connectionPaths = positions.ConnectionPaths

			// Apply scroll offset if needed
			lines := strings.Split(output, "\n")
			totalLines := len(lines)
			visibleLines := e.height - 4 // Reserve space for status, Ed, etc. (reduced by 1 for extra line)

			// For sequence diagrams, find the header size (participant boxes)
			headerLines := 0
			if e.diagram.Type == "sequence" && len(lines) > 0 {
				// In sequence diagrams, participants are drawn at the top
				// They typically take up 3-5 lines (box with text inside)
				// Look for the first lifeline (vertical line) to determine where headers end
				for i, line := range lines {
					if strings.Contains(line, "│") && i > 0 {
						// Found a lifeline, headers include this line plus one more
						// to show the complete bottom of the boxes and connection to lifelines
						headerLines = i + 2 // Include the lifeline start and one more line
						break
					}
					// Safety check - headers shouldn't be more than 12 lines
					if i > 12 {
						headerLines = 7 // Default to 7 lines for headers
						break
					}
				}
			}

			// Check if content exceeds screen height
			if totalLines > visibleLines {
				maxScroll := totalLines - visibleLines

				// Auto-scroll to bottom if diagram changed (new content added)
				if e.diagramChanged {
					// Scroll to bottom to show new content
					e.diagramScrollOffset = maxScroll
					// Clear the changed flag after handling it
					e.diagramChanged = false
				}

				// Clamp scroll offset to valid range
				if e.diagramScrollOffset < 0 {
					e.diagramScrollOffset = 0
				} else if e.diagramScrollOffset > maxScroll {
					e.diagramScrollOffset = maxScroll
				}

				// Calculate visible window
				startLine := e.diagramScrollOffset
				endLine := startLine + visibleLines
				if endLine > totalLines {
					endLine = totalLines
				}

				// Debug log
				if f, err := os.OpenFile("/tmp/edd_scroll.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
					fmt.Fprintf(f, "Scroll: offset=%d, total=%d, visible=%d, max=%d, start=%d, end=%d, headers=%d\n",
						e.diagramScrollOffset, totalLines, visibleLines, maxScroll, startLine, endLine, headerLines)
					f.Close()
				}

				// Extract visible portion with sticky headers for sequence diagrams
				var scrolledLines []string
				if e.diagram.Type == "sequence" && headerLines > 0 && e.diagramScrollOffset > 0 {
					// Sticky headers are needed
					// Step 1: Include the header lines at the top
					// IMPORTANT: Make a copy to avoid modifying the original
					scrolledLines = make([]string, headerLines)
					copy(scrolledLines, lines[:headerLines])

					// Debug: Log what we have so far
					if false { // Set to true for debugging
						fmt.Fprintf(os.Stderr, "DEBUG: headerLines=%d, lines[:headerLines] has %d lines\n", headerLines, len(scrolledLines))
						for i, line := range scrolledLines {
							fmt.Fprintf(os.Stderr, "  %d: [%.60s...]\n", i, line)
						}
					}

					// Step 2: Add ONE separator line (but check if we already have one)
					needSeparator := true
					if len(scrolledLines) > 0 {
						lastLine := scrolledLines[len(scrolledLines)-1]
						// Check if the last line is already a pure separator (not a box border)
						// A pure separator would be mostly dashes with no box drawing corners
						trimmed := strings.TrimSpace(lastLine)
						if len(trimmed) > 0 && !strings.Contains(trimmed, "╰") && !strings.Contains(trimmed, "╯") &&
							strings.Count(trimmed, "─") == len([]rune(trimmed)) {
							needSeparator = false
						}
					}

					if needSeparator {
						// Make it slightly shorter than terminal width to avoid wrapping issues
						sepWidth := e.width - 1
						if sepWidth < 1 {
							sepWidth = 1
						}
						separator := strings.Repeat("─", sepWidth)
						scrolledLines = append(scrolledLines, separator)
					}

					// Step 3: Add the scrolled content, starting AFTER the headers
					// to avoid showing them twice
					contentStart := headerLines
					if startLine > headerLines {
						contentStart = startLine // We've scrolled past headers
					}

					// Calculate how much space we have left after headers and separator
					remainingSpace := visibleLines - headerLines - 1 // -1 for separator
					contentEnd := contentStart + remainingSpace
					if contentEnd > totalLines {
						contentEnd = totalLines
					}

					// Append the content
					if contentStart < contentEnd && contentStart < len(lines) {
						scrolledLines = append(scrolledLines, lines[contentStart:contentEnd]...)
					}
				} else {
					// Normal scrolling (not a sequence diagram or not scrolled past headers)
					if startLine < len(lines) && endLine <= len(lines) && startLine < endLine {
						scrolledLines = lines[startLine:endLine]
					}
				}
				output = strings.Join(scrolledLines, "\n")

				// Add scroll indicators
				if e.diagram.Type == "sequence" && headerLines > 0 && e.diagramScrollOffset > 0 {
					// For sticky headers, adjust the indicators
					// Count lines hidden above (everything before the displayed content)
					actualStart := startLine
					if actualStart < headerLines {
						actualStart = headerLines
					}
					scrollHiddenLines := actualStart - headerLines // Don't count the headers themselves
					if scrollHiddenLines > 0 {
						output = fmt.Sprintf("[↑ %d more lines above (headers pinned)]\n", scrollHiddenLines) + output
					}

					// Check if there are more lines below
					actualEnd := actualStart + (visibleLines - headerLines - 1)
					if actualEnd < totalLines {
						output = output + fmt.Sprintf("\n[↓ %d more lines below]", totalLines-actualEnd)
					}
				} else {
					// Normal scroll indicators (also for sequence diagrams when not using sticky headers)
					if startLine > 0 {
						output = fmt.Sprintf("[↑ %d more lines above]\n", startLine) + output
					}
					if endLine < totalLines {
						output = output + fmt.Sprintf("\n[↓ %d more lines below]", totalLines-endLine)
					}
				}
			} else {
				// Content fits on screen, reset scroll offset and clear changed flag
				e.diagramScrollOffset = 0
				e.diagramChanged = false
			}

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

	// Mark diagram as changed to trigger auto-scroll
	e.diagramChanged = true

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

// isReturnConnection checks if a connection from B to A is likely a return/response
// to an earlier connection from A to B (for auto-applying dashed style in sequence diagrams)
func (e *TUIEditor) isReturnConnection(from, to int) bool {
	// Only apply this logic to sequence diagrams
	if e.diagram.Type != "sequence" {
		return false
	}

	// Look backwards through existing connections for an unreturned call from 'to' to 'from'
	returnCount := 0
	for i := len(e.diagram.Connections) - 1; i >= 0; i-- {
		conn := e.diagram.Connections[i]

		// Found a call from the node we're returning to
		if conn.From == to && conn.To == from {
			// Check if this call already has a return (is it dashed?)
			if conn.Hints == nil || conn.Hints["style"] != "dashed" {
				// This looks like a call that needs a return
				return true
			}
		}

		// If we see a return from 'from' to 'to', increment counter
		// This helps handle multiple call-return pairs
		if conn.From == from && conn.To == to {
			if conn.Hints != nil && conn.Hints["style"] == "dashed" {
				returnCount++
			}
		}
	}

	return false
}

// hasOpenCallFrom checks if 'to' has an open call from 'from' that hasn't been responded to
func (e *TUIEditor) hasOpenCallFrom(from, to int) bool {
	// Only for sequence diagrams
	if e.diagram.Type != "sequence" {
		return false
	}

	// Track open calls using a simple approach:
	// Look through existing connections to see if 'to' has received from 'from'
	// without responding back yet

	callDepth := 0
	for _, conn := range e.diagram.Connections {
		if conn.From == from && conn.To == to {
			// Found a call from 'from' to 'to'
			callDepth++
		} else if conn.From == to && conn.To == from {
			// Found a response from 'to' back to 'from'
			callDepth--
		}
	}

	// If callDepth > 0, there are unanswered calls
	return callDepth > 0
}

// detectActivation determines if a connection should trigger activation/deactivation
// based on behavioral patterns (not keywords)
// Returns (shouldActivateTarget, shouldDeactivateSource)
func (e *TUIEditor) detectActivation(from, to int, label string) (bool, bool) {
	// Only apply to sequence diagrams
	if e.diagram.Type != "sequence" {
		return false, false
	}

	// Debug logging
	if f, err := os.OpenFile("/tmp/edd_activation.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		defer f.Close()
		fmt.Fprintf(f, "\n[%s] detectActivation: from=%d, to=%d, label=%s\n",
			time.Now().Format("15:04:05"), from, to, label)
		fmt.Fprintf(f, "  Total connections: %d\n", len(e.diagram.Connections))
	}

	// Find the most recent unresponded incoming REQUEST to 'from'
	// (not just any incoming message, but one that represents a request needing orchestration)
	var hasUnrespondedIncoming bool
	var incomingFrom int = -1
	var incomingIndex int = -1

	for i := len(e.diagram.Connections) - 1; i >= 0; i-- {
		if e.diagram.Connections[i].To == from {
			caller := e.diagram.Connections[i].From
			// Check if we've already responded to this caller
			hasResponded := false
			for j := i + 1; j < len(e.diagram.Connections); j++ {
				if e.diagram.Connections[j].From == from && e.diagram.Connections[j].To == caller {
					hasResponded = true
					break
				}
			}
			if !hasResponded {
				// Check if this is likely a request (not a response from a downstream call)
				isLikelyRequest := false

				// Special case: first node (Client) always makes requests, never responses
				if caller == 0 {
					isLikelyRequest = true
				} else {
					// Check if this is a response to an unresponded call from 'from' to 'caller'
					// We count all calls and responses to match request-response pairs
					hasUnrespondedCallBefore := false

					// Count interactions: calls from 'from' to 'caller' and responses back
					callCount := 0
					responseCount := 0

					// Count all calls and responses BEFORE this incoming at index i
					for j := 0; j < i; j++ {
						if e.diagram.Connections[j].From == from && e.diagram.Connections[j].To == caller {
							callCount++
						} else if e.diagram.Connections[j].From == caller && e.diagram.Connections[j].To == from {
							responseCount++
						}
					}

					// If there are more calls than responses, this incoming is likely completing a pair
					if callCount > responseCount {
						hasUnrespondedCallBefore = true
						if f, err := os.OpenFile("/tmp/edd_activation.log", os.O_APPEND|os.O_WRONLY, 0644); err == nil {
							fmt.Fprintf(f, "    Before index %d: %d calls from %d->%d, %d responses. This is a RESPONSE.\n",
								i, callCount, from, caller, responseCount)
							f.Close()
						}
					} else {
						if f, err := os.OpenFile("/tmp/edd_activation.log", os.O_APPEND|os.O_WRONLY, 0644); err == nil {
							fmt.Fprintf(f, "    Before index %d: %d calls from %d->%d, %d responses. This is a REQUEST.\n",
								i, callCount, from, caller, responseCount)
							f.Close()
						}
					}

					// If we don't have an unresponded call before this, it's likely a request
					if !hasUnrespondedCallBefore {
						isLikelyRequest = true
					}
				}

				if f, err := os.OpenFile("/tmp/edd_activation.log", os.O_APPEND|os.O_WRONLY, 0644); err == nil {
					fmt.Fprintf(f, "    Incoming from %d: isLikelyRequest=%v\n",
						caller, isLikelyRequest)
					f.Close()
				}

				if isLikelyRequest {
					hasUnrespondedIncoming = true
					incomingFrom = caller
					incomingIndex = i
					break
				}
			}
		}
	}

	// Count outgoing calls made AFTER receiving the unresponded incoming
	// (these are the calls made while processing the request)
	outgoingCallCount := 0
	if hasUnrespondedIncoming && incomingIndex >= 0 {
		for i := incomingIndex + 1; i < len(e.diagram.Connections); i++ {
			conn := e.diagram.Connections[i]
			if conn.From == from && conn.To != incomingFrom {
				outgoingCallCount++
			}
		}
	}

	// Log the detection state
	if f, err := os.OpenFile("/tmp/edd_activation.log", os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		fmt.Fprintf(f, "  Detection state: hasUnrespondedIncoming=%v, incomingFrom=%d, outgoingCallCount=%d\n",
			hasUnrespondedIncoming, incomingFrom, outgoingCallCount)
		fmt.Fprintf(f, "  Current target: to=%d, will be 2nd call=%v\n", to, (to != incomingFrom && outgoingCallCount == 1))
		f.Close()
	}

	// If this connection will be the 2nd downstream call while having an unresponded incoming
	if hasUnrespondedIncoming && to != incomingFrom && outgoingCallCount == 1 {
		// Find the first outgoing call AFTER the incoming request and retroactively mark it for activation
		for i := incomingIndex + 1; i < len(e.diagram.Connections); i++ {
			if e.diagram.Connections[i].From == from && e.diagram.Connections[i].To != incomingFrom {
				if e.diagram.Connections[i].Hints == nil {
					e.diagram.Connections[i].Hints = make(map[string]string)
				}
				e.diagram.Connections[i].Hints["activate_source"] = "true"
				if f, err := os.OpenFile("/tmp/edd_activation.log", os.O_APPEND|os.O_WRONLY, 0644); err == nil {
					fmt.Fprintf(f, "  ACTIVATED: Marked connection %d for activation\n", i)
					f.Close()
				}
				break
			}
		}
		return false, false
	}

	// Check for deactivation: when 'from' responds back to someone who called it
	// after having made 2+ downstream calls
	if hasUnrespondedIncoming && to == incomingFrom {
		// This is a response back to the caller
		// Check if 'from' made 2+ downstream calls AFTER receiving the request
		downstreamCalls := 0
		for i := incomingIndex + 1; i < len(e.diagram.Connections); i++ {
			conn := e.diagram.Connections[i]
			if conn.From == from && conn.To != incomingFrom {
				downstreamCalls++
			}
		}

		if downstreamCalls >= 2 {
			return false, true // Deactivate on this response
		}
	}

	return false, false
}


// AddConnection adds a connection between two nodes
func (e *TUIEditor) AddConnection(from, to int, label string) {
	// Debug logging
	if f, err := os.OpenFile("/tmp/edd_connections.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		fmt.Fprintf(f, "\n[%s] AddConnection called: from=%d, to=%d, label=%s\n",
			time.Now().Format("15:04:05"), from, to, label)
		fmt.Fprintf(f, "  Current connections count: %d\n", len(e.diagram.Connections))
		fmt.Fprintf(f, "  Diagram type: %s\n", e.diagram.Type)
		defer f.Close()
	}

	// In sequence diagrams, allow multiple messages between same participants
	// In flowcharts, check for duplicate connections
	if e.diagram.Type != string(diagram.DiagramTypeSequence) {
		// Check for duplicate connections in the same direction only
		for _, existing := range e.diagram.Connections {
			if existing.From == from && existing.To == to {
				// Connection already exists in this direction
				if f, err := os.OpenFile("/tmp/edd_connections.log", os.O_APPEND|os.O_WRONLY, 0644); err == nil {
					fmt.Fprintf(f, "  REJECTED: Duplicate connection in flowchart\n")
				}
				return
			}
		}
	}

	// Generate a unique ID for the new connection
	// Use the index as the ID (connections.length will be the new index)
	connID := len(e.diagram.Connections)

	// Make sure this ID isn't already used
	for _, existing := range e.diagram.Connections {
		if existing.ID >= connID {
			connID = existing.ID + 1
		}
	}

	if f, err := os.OpenFile("/tmp/edd_connections.log", os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		fmt.Fprintf(f, "  Generated ID: %d\n", connID)
	}

	conn := diagram.Connection{
		ID:    connID,
		From:  from,
		To:    to,
		Label: label,
		Arrow: true, // Sequence diagrams always have arrows
		Hints: make(map[string]string),
	}

	// Auto-detect and apply dashed style for return connections in sequence diagrams
	if e.isReturnConnection(from, to) {
		conn.Hints["style"] = "dashed"
	}

	// Auto-detect activation/deactivation
	if activate, deactivate := e.detectActivation(from, to, label); activate || deactivate {
		if activate {
			conn.Hints["activate"] = "true"
		}
		if deactivate {
			conn.Hints["deactivate"] = "true"
		}
	}

	e.diagram.Connections = append(e.diagram.Connections, conn)

	if f, err := os.OpenFile("/tmp/edd_connections.log", os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		fmt.Fprintf(f, "  SUCCESS: Connection added, new count: %d\n", len(e.diagram.Connections))
	}

	// Mark diagram as changed to trigger auto-scroll
	e.diagramChanged = true

	// Save to history after modification
	e.SaveHistory()
}

// InsertConnection inserts a connection at a specific index
func (e *TUIEditor) InsertConnection(index int, from, to int, label string) {
	// Validate index
	if index < 0 {
		index = 0
	}
	if index > len(e.diagram.Connections) {
		index = len(e.diagram.Connections)
	}

	// Generate a unique ID for the new connection
	connID := len(e.diagram.Connections)
	for _, existing := range e.diagram.Connections {
		if existing.ID >= connID {
			connID = existing.ID + 1
		}
	}

	conn := diagram.Connection{
		ID:    connID,
		From:  from,
		To:    to,
		Label: label,
		Arrow: true, // Sequence diagrams always have arrows
		Hints: make(map[string]string),
	}

	// Auto-detect and apply dashed style for return connections in sequence diagrams
	if e.isReturnConnection(from, to) {
		conn.Hints["style"] = "dashed"
	}

	// Auto-detect activation/deactivation
	if activate, deactivate := e.detectActivation(from, to, label); activate || deactivate {
		if activate {
			conn.Hints["activate"] = "true"
		}
		if deactivate {
			conn.Hints["deactivate"] = "true"
		}
	}

	// Insert at the specified position
	if index >= len(e.diagram.Connections) {
		// Append to end
		e.diagram.Connections = append(e.diagram.Connections, conn)
	} else {
		// Insert at position
		e.diagram.Connections = append(e.diagram.Connections[:index],
			append([]diagram.Connection{conn}, e.diagram.Connections[index:]...)...)
	}

	// Mark diagram as changed
	e.diagramChanged = true

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

// handleActivationSelection handles the two-step activation selection process
func (e *TUIEditor) handleActivationSelection(connIndex int) {
	if connIndex < 0 || connIndex >= len(e.diagram.Connections) {
		return
	}

	if e.activationStartConn == -1 {
		// First selection - this is the start of activation
		e.activationStartConn = connIndex
		e.activationStartFrom = e.diagram.Connections[connIndex].From

		// Re-assign labels to show only connections after this point
		e.assignActivationEndLabels()
		// Stay in jump mode to select the end point
	} else {
		// Second selection - this is the end of activation
		if connIndex > e.activationStartConn {
			// Apply activation from start to end
			e.applyActivation(e.activationStartConn, connIndex)
		}
		// Reset and exit
		e.activationStartConn = -1
		e.activationStartFrom = -1
		e.clearJumpLabels()
		e.SetMode(ModeNormal)
		e.SaveHistory()
	}
}

// assignActivationEndLabels assigns labels only to valid end points
func (e *TUIEditor) assignActivationEndLabels() {
	e.connectionLabels = make(map[int]rune)

	// Jump label characters in order of preference
	jumpChars := []rune("asdfjkl;ghqwertyuiopzxcvbnm,./1234567890")
	labelIndex := 0

	// Only assign labels to connections that:
	// 1. Come after the start point
	// 2. Are FROM the same participant that we're activating
	for i := e.activationStartConn + 1; i < len(e.diagram.Connections) && labelIndex < len(jumpChars); i++ {
		conn := e.diagram.Connections[i]
		// Only connections FROM the participant we're activating are valid end points
		if conn.From == e.activationStartFrom {
			e.connectionLabels[i] = jumpChars[labelIndex]
			labelIndex++
		}
	}
}

// deleteActivation removes all activation hints from a connection and its pair
func (e *TUIEditor) deleteActivation(connIndex int) {
	if connIndex < 0 || connIndex >= len(e.diagram.Connections) {
		return
	}

	conn := &e.diagram.Connections[connIndex]

	// Identify what participant this activation is for
	// activate_source means the To participant gets activated
	// activate also means the To participant gets activated
	// deactivate means the To participant gets deactivated
	var participantID int
	if conn.Hints != nil {
		if conn.Hints["activate_source"] == "true" {
			participantID = conn.To  // The recipient gets activated
		} else if conn.Hints["activate"] == "true" {
			participantID = conn.To  // The recipient gets activated
		} else if conn.Hints["deactivate"] == "true" {
			participantID = conn.To  // The recipient gets deactivated
		}
	}

	// Find and clear all activation hints for this participant
	// This includes both the start (activate_source) and end (deactivate) of the activation
	for i := range e.diagram.Connections {
		if e.diagram.Connections[i].Hints != nil {
			hints := e.diagram.Connections[i].Hints

			// Check if this connection is part of the same activation
			// All activation hints for the same participant should be cleared
			shouldClear := false
			if hints["activate_source"] == "true" && e.diagram.Connections[i].To == participantID {
				shouldClear = true
			}
			if hints["activate"] == "true" && e.diagram.Connections[i].To == participantID {
				shouldClear = true
			}
			if hints["deactivate"] == "true" && e.diagram.Connections[i].To == participantID {
				shouldClear = true
			}

			if shouldClear {
				// Remove activation-related hints
				delete(hints, "activate_source")
				delete(hints, "activate")
				delete(hints, "deactivate")

				// Clean up empty hints map
				if len(hints) == 0 {
					e.diagram.Connections[i].Hints = nil
				}
			}
		}
	}

	// Save to history
	e.SaveHistory()
}

// applyActivation applies activation hints between start and end connections
func (e *TUIEditor) applyActivation(startIdx, endIdx int) {
	if startIdx < 0 || endIdx >= len(e.diagram.Connections) || startIdx >= endIdx {
		return
	}

	// Mark the start connection to activate the source
	startConn := &e.diagram.Connections[startIdx]
	if startConn.Hints == nil {
		startConn.Hints = make(map[string]string)
	}
	startConn.Hints["activate_source"] = "true"

	// Mark the end connection to deactivate
	endConn := &e.diagram.Connections[endIdx]
	if endConn.Hints == nil {
		endConn.Hints = make(map[string]string)
	}
	endConn.Hints["deactivate"] = "true"
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

// HandleBacktab handles Shift+Tab to move to previous connection
func (e *TUIEditor) HandleBacktab() {
	if e.selectedConnection >= 0 && e.mode == ModeEdit && len(e.diagram.Connections) > 0 {
		// Save current index before committing (commitText clears it)
		currentIndex := e.selectedConnection

		// Commit current text first
		e.commitText()

		// Move to previous connection
		prevIndex := currentIndex - 1
		if prevIndex < 0 {
			// Wrap around to last connection
			prevIndex = len(e.diagram.Connections) - 1
		}

		e.selectedConnection = prevIndex
		conn := e.diagram.Connections[prevIndex]
		e.textBuffer = []rune(conn.Label)
		e.cursorPos = len(e.textBuffer)
		// Stay in EDIT mode
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

// Jump label characters in ergonomic order:
// - Home row (most comfortable): asdfjklgh
// - Upper row: qwertyuiop
// - Lower row: zxcvbnm
// - Numbers: 1234567890
// - Capitals (home row): ASDFJKLGH
// - Capitals (upper row): QWERTYUIOP
// - Capitals (lower row): ZXCVBNM
// - Number row shift chars: !@#$%^&*()
// Total: 69 characters
const jumpChars = "asdfjklghqwertyuiopzxcvbnm1234567890ASDFJKLGHQWERTYUIOPZXCVBNM!@#$%^&*()"

// startJump initiates jump mode with labels
func (e *TUIEditor) startJump(action JumpAction) {
	// Debug logging
	if f, err := os.OpenFile("/tmp/edd_labels.log", os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		fmt.Fprintf(f, "\n=== startJump called ===\n")
		fmt.Fprintf(f, "Action: %v, current mode: %v\n", action, e.mode)
		fmt.Fprintf(f, "Current scroll offset: %d\n", e.diagramScrollOffset)
		fmt.Fprintf(f, "Node positions available: %d\n", len(e.nodePositions))
		f.Close()
	}

	e.jumpAction = action
	e.assignJumpLabels()
	e.SetMode(ModeJump)
}

// isNodeVisible checks if a node is visible in the current viewport
func (e *TUIEditor) isNodeVisible(nodeID int) bool {
	pos, exists := e.nodePositions[nodeID]
	if !exists {
		// If we don't have position info yet, assume visible
		// This can happen when jump mode is entered before rendering
		// or when the renderer hasn't provided positions
		return true
	}

	// Check if the node is in the visible viewport
	// Calculate visible range based on scroll offset
	visibleStart := e.diagramScrollOffset
	visibleEnd := e.diagramScrollOffset + e.height - 4 // Account for UI elements

	// Special handling for sequence diagrams with sticky headers
	if e.diagram.Type == "sequence" && e.diagramScrollOffset > 0 {
		// When sticky headers are active, participants in the header area
		// remain visible until we've scrolled way past them
		if pos.Y < 7 {
			// In sequence diagrams, participants (at Y < 7) have lifelines that extend
			// through the entire diagram. They remain visible via sticky headers
			// as long as we're viewing any part of the diagram content.

			// Participants are visible via sticky headers as long as there's still
			// some diagram content in the viewport
			// Estimate total content height based on connections
			contentHeight := 7 // Header area
			for range e.diagram.Connections {
				// Each connection takes roughly 2-3 lines
				contentHeight += 2
			}

			// If we've scrolled past all content, participants aren't visible
			if e.diagramScrollOffset >= contentHeight {
				return false
			}

			// Otherwise participants are visible via sticky headers
			return true
		}
		// For non-participant nodes, adjust visible range for the space taken by headers
		visibleEnd = e.diagramScrollOffset + e.height - 4 - 8 // Subtract header space
	}

	// Check if node's Y position is within visible range
	// Account for node height (typically 3 lines for a box)
	nodeTop := pos.Y
	nodeBottom := pos.Y + 3 // Assume nodes are about 3 lines tall

	// Node is visible if any part of it is in the visible range
	return nodeBottom >= visibleStart && nodeTop < visibleEnd
}

// isConnectionVisible checks if a connection arrow is actually visible in the viewport
func (e *TUIEditor) isConnectionVisible(connIndex int) bool {
	if connIndex >= len(e.diagram.Connections) {
		return false
	}

	// Check if we have the path for this connection
	path, hasPath := e.connectionPaths[connIndex]
	if !hasPath || len(path.Points) == 0 {
		// If no path info, fall back to checking endpoints
		conn := e.diagram.Connections[connIndex]
		return e.isNodeVisible(conn.From) || e.isNodeVisible(conn.To)
	}

	// Get the middle point of the path (where the label would be placed)
	midPoint := path.Points[len(path.Points)/2]

	// Convert to viewport coordinates
	viewportY := e.TransformToViewport(midPoint.Y, false)

	// Check if this Y position is within the visible area
	// Terminal height minus status lines (4 lines reserved)
	visibleBottom := e.height - 4

	return viewportY >= 1 && viewportY <= visibleBottom
}

// assignJumpLabels assigns single-character labels to visible nodes and connections
func (e *TUIEditor) assignJumpLabels() {
	e.jumpLabels = make(map[int]rune)
	e.connectionLabels = make(map[int]rune)
	e.insertionLabels = make(map[int]rune) // Always clear insertion labels

	// Debug logging
	if f, err := os.OpenFile("/tmp/edd_labels.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		fmt.Fprintf(f, "\n=== assignJumpLabels called ===\n")
		fmt.Fprintf(f, "Scroll offset: %d, Mode: %v, Diagram type: %s\n", e.diagramScrollOffset, e.mode, e.diagram.Type)
		fmt.Fprintf(f, "JumpAction: %v, Total nodes: %d\n", e.jumpAction, len(e.diagram.Nodes))
		f.Close()
	}

	labelIndex := 0

	// Special handling for sequence diagrams in connect mode
	// Participants are always visible at the top (sticky header)
	if e.diagram.Type == "sequence" &&
	   (e.jumpAction == JumpActionConnectFrom || e.jumpAction == JumpActionConnectTo) {
		// Always assign labels to all participants
		for _, node := range e.diagram.Nodes {
			if labelIndex >= len(jumpChars) {
				break
			}
			e.jumpLabels[node.ID] = rune(jumpChars[labelIndex])
			if f, err := os.OpenFile("/tmp/edd_labels.log", os.O_APPEND|os.O_WRONLY, 0644); err == nil {
				fmt.Fprintf(f, "  Sequence participant %d -> label '%c' (always visible)\n", node.ID, jumpChars[labelIndex])
				f.Close()
			}
			labelIndex++
		}
	} else if e.jumpAction != JumpActionActivation && e.jumpAction != JumpActionDeleteActivation {
		// For activation modes, we only want connection labels, not node labels
		// Original logic - assign labels only to visible nodes
		for _, node := range e.diagram.Nodes {
			if labelIndex >= len(jumpChars) {
				break
			}

			isVisible := e.isNodeVisible(node.ID)
			if f, err := os.OpenFile("/tmp/edd_labels.log", os.O_APPEND|os.O_WRONLY, 0644); err == nil {
				pos, hasPos := e.nodePositions[node.ID]
				if hasPos {
					fmt.Fprintf(f, "  Node %d: pos.Y=%d, visible=%v, labelIndex=%d\n", node.ID, pos.Y, isVisible, labelIndex)
				} else {
					fmt.Fprintf(f, "  Node %d: no position, visible=%v, labelIndex=%d\n", node.ID, isVisible, labelIndex)
				}
				f.Close()
			}

			if isVisible {
				e.jumpLabels[node.ID] = rune(jumpChars[labelIndex])
				if f, err := os.OpenFile("/tmp/edd_labels.log", os.O_APPEND|os.O_WRONLY, 0644); err == nil {
					fmt.Fprintf(f, "    -> Assigned label '%c' to node %d\n", jumpChars[labelIndex], node.ID)
					f.Close()
				}
				labelIndex++
			}
		}
	}

	// If in delete, edit, hint, or activation mode, also assign labels to visible connections
	if e.jumpAction == JumpActionDelete || e.jumpAction == JumpActionEdit ||
	   e.jumpAction == JumpActionHint || e.jumpAction == JumpActionActivation {
		if f, err := os.OpenFile("/tmp/edd_labels.log", os.O_APPEND|os.O_WRONLY, 0644); err == nil {
			fmt.Fprintf(f, "Starting connection labeling at labelIndex=%d\n", labelIndex)
			f.Close()
		}

		// Use index-based iteration to ensure consistent ordering
		for i := 0; i < len(e.diagram.Connections); i++ {
			if labelIndex >= len(jumpChars) {
				break
			}

			isVisible := e.isConnectionVisible(i)
			if f, err := os.OpenFile("/tmp/edd_labels.log", os.O_APPEND|os.O_WRONLY, 0644); err == nil {
				fmt.Fprintf(f, "  Connection %d: visible=%v, labelIndex=%d\n", i, isVisible, labelIndex)
				f.Close()
			}

			if isVisible {
				e.connectionLabels[i] = rune(jumpChars[labelIndex])
				if f, err := os.OpenFile("/tmp/edd_labels.log", os.O_APPEND|os.O_WRONLY, 0644); err == nil {
					fmt.Fprintf(f, "    -> Assigned label '%c' to connection %d\n", jumpChars[labelIndex], i)
					f.Close()
				}
				labelIndex++
			}
		}
	}

	// For delete activation mode, show ONE label per activation span
	if e.jumpAction == JumpActionDeleteActivation {
		e.connectionLabels = make(map[int]rune)
		labelIndex = 0

		// Track which participants already have a label for their activation
		participantLabeled := make(map[int]bool)

		for i := 0; i < len(e.diagram.Connections) && labelIndex < len(jumpChars); i++ {
			conn := e.diagram.Connections[i]

			// Check if this connection starts an activation (activate_source or activate)
			if conn.Hints != nil {
				var participantID int
				isActivationStart := false

				if conn.Hints["activate_source"] == "true" {
					participantID = conn.To  // The recipient gets activated
					isActivationStart = true
				} else if conn.Hints["activate"] == "true" {
					participantID = conn.To  // The recipient gets activated
					isActivationStart = true
				}

				// Only assign a label to the first activation start for each participant
				if isActivationStart && !participantLabeled[participantID] && e.isConnectionVisible(i) {
					e.connectionLabels[i] = rune(jumpChars[labelIndex])
					participantLabeled[participantID] = true
					labelIndex++
				}
			}
		}
	} else if e.jumpAction == JumpActionInsertAt {
		// For insert mode, assign labels to insertion points (before each connection and after the last)
		if f, err := os.OpenFile("/tmp/edd_labels.log", os.O_APPEND|os.O_WRONLY, 0644); err == nil {
			fmt.Fprintf(f, "Starting insertion point labeling at labelIndex=%d\n", labelIndex)
			f.Close()
		}

		// Always add label for position 0 (before first connection)
		if labelIndex < len(jumpChars) {
			e.insertionLabels[0] = rune(jumpChars[labelIndex])
			labelIndex++
		}

		// Add labels for positions after each visible connection
		for i := 0; i < len(e.diagram.Connections); i++ {
			if labelIndex >= len(jumpChars) {
				break
			}
			// Only add labels for visible connections
			if e.isConnectionVisible(i) {
				e.insertionLabels[i+1] = rune(jumpChars[labelIndex])
				labelIndex++
			}
		}
	}

	// Log final labels assigned
	if f, err := os.OpenFile("/tmp/edd_labels.log", os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		fmt.Fprintf(f, "Final jump labels assigned: %d nodes, %d connections\n", len(e.jumpLabels), len(e.connectionLabels))
		for nodeID, label := range e.jumpLabels {
			fmt.Fprintf(f, "  Node %d -> '%c'\n", nodeID, label)
		}
		f.Close()
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
	e.insertionLabels = make(map[int]rune)
	e.jumpAction = JumpActionSelect
}

// ============================================
// Methods from modes.go
// ============================================

// Mode represents the current editing mode
type Mode int

const (
	ModeNormal   Mode = iota // Normal navigation mode
	ModeInsert               // Inserting new nodes
	ModeEdit                 // Editing existing node text
	ModeCommand              // Command input mode
	ModeJump                 // Jump selection active
	ModeJSON                 // JSON view mode
	ModeHintMenu             // Editing connection hints
	ModeHelp                 // Help display mode
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
	JumpActionSelect       JumpAction = iota // Just select the node
	JumpActionEdit                           // Edit the selected node
	JumpActionDelete                         // Delete the selected node
	JumpActionConnectFrom                    // Start connection from this node
	JumpActionConnectTo                      // Complete connection to this node
	JumpActionHint                           // Edit hints for nodes and connections
	JumpActionInsertAt                       // Select insertion point for new connection
	JumpActionActivation                     // Toggle activation on connections
	JumpActionDeleteActivation               // Delete activation from connections
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

// GetDiagramScrollOffset returns the current diagram scroll offset
func (e *TUIEditor) GetDiagramScrollOffset() int {
	return e.diagramScrollOffset
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

// GetInsertionLabels returns the current insertion point jump labels
func (e *TUIEditor) GetInsertionLabels() map[int]rune {
	return e.insertionLabels
}

// GetInsertionPoint returns the current insertion point (-1 if not set)
func (e *TUIEditor) GetInsertionPoint() int {
	return e.insertionPoint
}

// GetJumpAction returns the current jump action
func (e *TUIEditor) GetJumpAction() JumpAction {
	return e.jumpAction
}

// GetActivationStartConn returns the activation start connection index (-1 if not in activation mode)
func (e *TUIEditor) GetActivationStartConn() int {
	return e.activationStartConn
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
	if f, err := os.OpenFile("/tmp/edd_connect.log", os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		fmt.Fprintf(f, "  StartConnect called, nodes=%d\n", len(e.diagram.Nodes))
		f.Close()
	}
	if len(e.diagram.Nodes) >= 2 {
		e.continuousConnect = false
		e.startJump(JumpActionConnectFrom)
	} else {
		if f, err := os.OpenFile("/tmp/edd_connect.log", os.O_APPEND|os.O_WRONLY, 0644); err == nil {
			fmt.Fprintf(f, "  NOT ENOUGH NODES! Need >=2, have %d\n", len(e.diagram.Nodes))
			f.Close()
		}
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

// StartInsert starts connection insertion mode (for sequence diagrams)
func (e *TUIEditor) StartInsert() {
	// Only available for sequence diagrams with at least 2 nodes
	if e.diagram.Type == "sequence" && len(e.diagram.Nodes) >= 2 {
		e.continuousConnect = false
		e.startJump(JumpActionInsertAt)
	}
}

// StartContinuousInsert starts continuous connection insertion mode
func (e *TUIEditor) StartContinuousInsert() {
	// Only available for sequence diagrams with at least 2 nodes
	if e.diagram.Type == "sequence" && len(e.diagram.Nodes) >= 2 {
		e.continuousConnect = true
		e.startJump(JumpActionInsertAt)
	}
}

// StartActivationToggle starts activation toggle mode for connections
func (e *TUIEditor) StartActivationToggle() {
	// Only available for sequence diagrams with connections
	if e.diagram.Type == "sequence" && len(e.diagram.Connections) > 0 {
		// Reset activation selection state
		e.activationStartConn = -1
		e.activationStartFrom = -1
		e.startJump(JumpActionActivation)
	}
}

// StartActivationDelete starts deletion mode for activations
func (e *TUIEditor) StartActivationDelete() {
	// Only available for sequence diagrams with connections
	if e.diagram.Type == "sequence" && len(e.diagram.Connections) > 0 {
		e.startJump(JumpActionDeleteActivation)
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


// GetCommand returns the current command buffer
func (e *TUIEditor) GetCommand() string {
	return string(e.commandBuffer)
}

// ClearCommand clears the command buffer
func (e *TUIEditor) ClearCommand() {
	e.commandBuffer = []rune{}
	e.commandResult = ""
	e.saveRequested = false
	e.quitRequested = false
	e.exportFormat = ""
	e.exportFilename = ""
}

// ProcessCommand processes the completed command when Enter is pressed
func (e *TUIEditor) ProcessCommand() {
	cmd := strings.TrimSpace(string(e.commandBuffer))
	if cmd == "" {
		e.SetMode(ModeNormal)
		return
	}

	// Parse command
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		e.SetMode(ModeNormal)
		return
	}

	switch parts[0] {
	case "w", "write":
		// Save command
		e.saveRequested = true
		if len(parts) > 1 {
			e.saveFilename = parts[1]
		}
		e.hasChanges = false // Clear changes flag after save
		e.commandResult = "Saving..."
		e.SetMode(ModeNormal)

	case "wq":
		// Save and quit
		e.saveRequested = true
		if len(parts) > 1 {
			e.saveFilename = parts[1]
		}
		e.quitRequested = true
		e.hasChanges = false
		e.SetMode(ModeNormal)

	case "q", "quit":
		// Quit
		if e.hasChanges {
			// Has unsaved changes - require :q! to force quit
			e.commandResult = "Unsaved changes! Use :q! to force quit or :wq to save and quit"
			e.SetMode(ModeNormal)
		} else {
			e.quitRequested = true
			e.SetMode(ModeNormal)
		}

	case "q!":
		// Force quit without saving
		e.quitRequested = true
		e.SetMode(ModeNormal)

	case "export":
		// Export command
		if len(parts) < 2 {
			e.commandResult = "Usage: :export <format> [filename]"
		} else {
			e.exportFormat = parts[1]
			if len(parts) > 2 {
				e.exportFilename = parts[2]
			}
		}
		e.SetMode(ModeNormal)

	default:
		e.commandResult = "Unknown command: " + parts[0]
		e.SetMode(ModeNormal)
	}
}

// HasUnsavedChanges returns whether there are unsaved changes
func (e *TUIEditor) HasUnsavedChanges() bool {
	return e.hasChanges
}

// markAsModified marks the diagram as having unsaved changes
func (e *TUIEditor) markAsModified() {
	e.hasChanges = true
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

// TransformToViewport converts diagram Y coordinate to viewport Y coordinate
// This consolidates all coordinate transformation logic in one place
func (e *TUIEditor) TransformToViewport(diagramY int, hasScrollIndicator bool) int {
	scrollIndicatorLines := 0
	if hasScrollIndicator {
		scrollIndicatorLines = 1
	}

	// For sequence diagrams with sticky headers
	if e.diagram.Type == "sequence" && e.diagramScrollOffset > 0 {
		if diagramY < 7 {
			// Participant in sticky header area
			// Extra padding lines are added, so box appears at Y+2
			return diagramY + 2 + scrollIndicatorLines
		} else {
			// Content below headers
			headerLines := 8 // 7 for header + 1 for separator
			return headerLines + 1 + scrollIndicatorLines + (diagramY - e.diagramScrollOffset)
		}
	}

	// Normal scrolling (no sticky headers)
	return diagramY - e.diagramScrollOffset + 1 + scrollIndicatorLines
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
	// This is the SINGLE source of truth for normal mode key handling
	// Used by both the interactive loop and tests

	switch key {
	case 'q', 3: // q or Ctrl+C to quit
		return true

	case ':': // Enter command mode
		e.SetMode(ModeCommand)


	case 'a': // Add node
		e.StartAddNode()

	case 'c': // Connect (single)
		if f, err := os.OpenFile("/tmp/edd_connect.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			fmt.Fprintf(f, "\n[%s] 'c' key pressed\n", time.Now().Format("15:04:05"))
			fmt.Fprintf(f, "  Mode: %v, Nodes: %d, Connections: %d\n", e.mode, len(e.diagram.Nodes), len(e.diagram.Connections))
			f.Close()
		}
		e.StartConnect()

	case 'C': // Connect (continuous)
		if f, err := os.OpenFile("/tmp/edd_connect.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			fmt.Fprintf(f, "\n[%s] 'C' key pressed\n", time.Now().Format("15:04:05"))
			fmt.Fprintf(f, "  Mode: %v, Nodes: %d, Connections: %d\n", e.mode, len(e.diagram.Nodes), len(e.diagram.Connections))
			f.Close()
		}
		e.StartContinuousConnect()

	case 'd': // Delete (single)
		e.StartDelete()

	case 'D': // Delete (continuous)
		e.StartContinuousDelete()

	case 'e': // Edit
		e.StartEdit()

	case 'H': // Edit connection hints
		e.StartHintEdit()

	case 'i': // Insert connection (single)
		e.StartInsert()

	case 'I': // Insert connection (continuous)
		e.StartContinuousInsert()

	case 'v': // Toggle activation on connections
		e.StartActivationToggle()

	case 'V': // Delete activations (shift+v)
		e.StartActivationDelete()

	case 'J': // JSON view (capital J)
		e.SetMode(ModeJSON)

	case 't': // Toggle diagram type
		e.ToggleDiagramType()

	case 'u': // Undo
		e.Undo()

	case 18: // Ctrl+R for redo
		e.Redo()

	case 'j': // Scroll down line (vim-style)
		e.ScrollDiagram(5)

	case 'k': // Scroll up line (vim-style)
		e.ScrollDiagram(-5)

	case 21: // Ctrl+U - scroll up half page
		e.ScrollDiagram(-e.height / 2)

	case 4: // Ctrl+D - scroll down half page
		e.ScrollDiagram(e.height / 2)

	case 'g': // Go to top
		e.ScrollToTop()

	case 'G': // Go to bottom
		e.ScrollToBottom()


	case 27: // ESC - if we have a previous jump action, restart that jump mode
		if e.previousJumpAction != 0 {
			action := e.previousJumpAction
			e.previousJumpAction = 0 // Clear it
			e.startJump(action)      // Restart jump mode with the same action
		}
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
			e.previousJumpAction = 0 // Clear it
			e.startJump(action)      // Restart jump mode with the same action
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

	case 9: // Tab - move to next connection (without committing)
		if e.selectedConnection >= 0 && e.mode == ModeEdit {
			// Save current index before committing (commitText clears it)
			currentIndex := e.selectedConnection

			// Commit current text first
			e.commitText()

			// Move to next connection
			nextIndex := currentIndex + 1
			if nextIndex >= len(e.diagram.Connections) {
				// Wrap around to first connection
				nextIndex = 0
			}

			e.selectedConnection = nextIndex
			conn := e.diagram.Connections[nextIndex]
			e.textBuffer = []rune(conn.Label)
			e.cursorPos = len(e.textBuffer)
			// Stay in EDIT mode
		}

	case 13, 10: // Enter - commit text
		// Save the mode before committing (in case commit changes it)
		wasInsertMode := e.mode == ModeInsert
		wasEditingConnection := e.selectedConnection >= 0
		currentConnectionIndex := e.selectedConnection

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
		} else if wasEditingConnection && e.mode == ModeEdit {
			// If we were editing a connection, move to the next one
			nextIndex := currentConnectionIndex + 1
			if nextIndex < len(e.diagram.Connections) {
				// Select and edit the next connection
				e.selectedConnection = nextIndex
				conn := e.diagram.Connections[nextIndex]
				e.textBuffer = []rune(conn.Label)
				e.cursorPos = len(e.textBuffer)
				// Stay in EDIT mode
			} else {
				// No more connections, return to normal mode
				e.SetMode(ModeNormal)
			}
		} else {
			// In EDIT mode (for nodes), return to normal
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
		e.ProcessCommand()
		// ProcessCommand will handle setting the mode

	default:
		// Add to command buffer
		if unicode.IsPrint(key) {
			e.commandBuffer = append(e.commandBuffer, key)
		}
	}

	return false
}

// executeCommand processes command mode commands
func (e *TUIEditor) executeCommand(command string) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "w", "write":
		// Save command - handled by terminal layer
		e.saveRequested = true
		if len(parts) > 1 {
			e.saveFilename = parts[1]
		}
		e.commandResult = "Saving..."
	case "q", "quit":
		// Quit command - handled by terminal layer
		e.quitRequested = true
		e.commandResult = "Quitting..."
	case "wq":
		// Save and quit
		e.saveRequested = true
		e.quitRequested = true
		if len(parts) > 1 {
			e.saveFilename = parts[1]
		}
		e.commandResult = "Saving and quitting..."
	case "export", "exp", "e":
		// Export command - support various aliases
		if len(parts) < 2 {
			e.commandResult = "Usage: export <format> [filename|clipboard|clip]"
			return
		}
		format := parts[1]
		filename := ""
		if len(parts) > 2 {
			filename = parts[2]
		}
		e.exportDiagram(format, filename)
	default:
		e.commandResult = "Unknown command: " + parts[0]
	}
}

// exportDiagram exports the diagram to the specified format
func (e *TUIEditor) exportDiagram(format, filename string) {
	// Support format shortcuts
	switch format {
	case "m":
		format = "mermaid"
	case "p":
		format = "plantuml"
	case "a":
		format = "ascii"
	}

	// This will be implemented by the terminal layer
	// Set a command result that the terminal layer can check
	e.exportFormat = format
	e.exportFilename = filename
	e.commandResult = "Export requested: " + format
}

// handleJumpKey processes keys when jump labels are active
func (e *TUIEditor) handleJumpKey(key rune) bool {
	// ESC cancels jump
	if key == 27 {
		e.clearJumpLabels()
		e.continuousConnect = false // Exit continuous connect mode
		e.continuousDelete = false  // Exit continuous delete mode
		e.previousJumpAction = 0    // Clear the previous action since we're canceling
		e.selected = -1             // Clear selected node
		e.insertionPoint = -1       // Clear insertion point
		e.activationStartConn = -1  // Clear activation selection state
		e.activationStartFrom = -1
		e.SetMode(ModeNormal)
		return false
	}

	// Special case: 'V' in activation mode switches to delete activation mode
	if key == 'V' && e.jumpAction == JumpActionActivation {
		e.clearJumpLabels()
		e.activationStartConn = -1  // Clear activation selection state
		e.activationStartFrom = -1
		e.StartActivationDelete()
		return false
	}

	// Look for matching insertion point label (in insert mode)
	if e.jumpAction == JumpActionInsertAt {
		for insertPos, label := range e.insertionLabels {
			if label == key {
				// Save the insertion point and move to standard connect flow
				e.insertionPoint = insertPos
				e.startJump(JumpActionConnectFrom)
				return false
			}
		}
	}

	// Look for matching node jump label
	for nodeID, label := range e.jumpLabels {
		if label == key {
			// Found match - execute jump action
			e.executeJumpAction(nodeID)
			return false
		}
	}

	// Look for matching connection jump label (in delete, edit, hint, activation, or delete activation mode)
	if e.jumpAction == JumpActionDelete || e.jumpAction == JumpActionEdit ||
	   e.jumpAction == JumpActionHint || e.jumpAction == JumpActionActivation ||
	   e.jumpAction == JumpActionDeleteActivation {
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
					e.previousJumpAction = e.jumpAction // Save the action for ESC handling
					e.StartEditingConnection(connIndex)
				} else if e.jumpAction == JumpActionHint {
					// Enter hint menu for this connection
					e.previousJumpAction = e.jumpAction // Save the action for ESC handling
					e.editingHintConn = connIndex
					// Log to file
					if f, err := os.OpenFile("/tmp/edd_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
						fmt.Fprintf(f, "Selected connection index %d for hint editing (pressed '%c')\n", connIndex, key)
						f.Close()
					}
					e.clearJumpLabels()
					e.SetMode(ModeHintMenu)
				} else if e.jumpAction == JumpActionActivation {
					// Handle two-step activation selection
					e.handleActivationSelection(connIndex)
				} else if e.jumpAction == JumpActionDeleteActivation {
					// Delete activation from this connection
					e.deleteActivation(connIndex)
					e.clearJumpLabels()
					e.SetMode(ModeNormal)
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
			// Check if we have an insertion point set (from insert mode)
			if e.insertionPoint >= 0 {
				// Insert at the specified position
				e.InsertConnection(e.insertionPoint, e.selected, nodeID, "")
				// In continuous mode, increment insertion point to keep inserting at the same relative position
				// Otherwise clear it
				if e.continuousConnect {
					e.insertionPoint++
				} else {
					e.insertionPoint = -1
				}
			} else {
				// Normal append
				e.AddConnection(e.selected, nodeID, "")
			}
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
			e.insertionPoint = -1 // Also clear here for safety
			e.clearJumpLabels()
			e.SetMode(ModeNormal)
		}

	case JumpActionHint:
		// Enter hint menu for this node
		e.editingHintNode = nodeID
		e.clearJumpLabels()
		e.SetMode(ModeHintMenu)

	case JumpActionActivation:
		// This shouldn't be reached for nodes - activation is for connections
		// Just return to normal mode
		e.clearJumpLabels()
		e.SetMode(ModeNormal)
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
	case 27, 'q', 'J': // ESC, q, or J to return to diagram view
		e.SetMode(ModeNormal)

	case 'E': // Edit in external editor (also works from JSON view)
		// This will be handled by the main loop
		return false

	case 'k', 'K': // vim-style up
		e.ScrollJSON(-1)

	case 'j': // vim-style down
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
			e.previousJumpAction = 0 // Clear it
			e.startJump(action)      // Restart jump mode with the same action
		} else {
			e.SetMode(ModeNormal)
		}
	case 13, 10: // Enter - exit to normal mode
		e.editingHintNode = -1
		e.previousJumpAction = 0 // Clear the previous action
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
			e.previousJumpAction = 0 // Clear it
			e.startJump(action)      // Restart jump mode with the same action
		} else {
			e.SetMode(ModeNormal)
		}
	case 13, 10: // Enter - exit to normal mode
		e.editingHintConn = -1
		e.previousJumpAction = 0 // Clear the previous action
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

	// Build menu content
	var menuLines []string

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

		menuLines = []string{
			"Participant: " + nodeText + " | box=" + boxStyle + "/" + color + " | lifeline=" + lifelineStyle + "/" + lifelineColor,
			"Box: [a]Round [b]Sharp [c]Double [d]Thick | [r]Red [g]Green [y]Yellow [u]Blue [m]Magenta [n]Cyan [w]Clear",
			"Line: [A]Solid [B]Dash [C]Dot [D]Double | [R]Red [G]Green [Y]Yellow [U]Blue [M]Magenta [N]Cyan [W]Clear",
			"[ESC]Back [Enter]Done",
		}
	} else {
		// Full menu for flowcharts
		menuLines = []string{
			"Node: " + nodeText + " | style=" + style + ", color=" + color,
			"Style: [a]Rounded [b]Sharp [c]Double [d]Thick | Color: [r]Red [g]Green [y]Yellow [u]Blue [m]Magenta [n]Cyan [w]Clear",
			"Text: [o]Bold(" + bold + ") [i]Italic(" + italic + ") [t]Center(" + textAlign + ") | Shadow: [z]Add [x]Remove [l]Density",
			"Position: [1-9]Grid [0]Auto | [ESC]Back [Enter]Done",
		}
	}

	// Use absolute positioning to draw menu at bottom of screen
	var output strings.Builder

	// Move to bottom of screen (leave 5 lines for menu)
	startLine := e.height - len(menuLines) - 1

	// Clear the menu area and draw menu
	for i, line := range menuLines {
		// Move to position and clear line
		output.WriteString(fmt.Sprintf("\033[%d;1H\033[K", startLine+i))
		// Draw menu line with background color for visibility
		output.WriteString("\033[44m") // Blue background
		output.WriteString(line)
		// Pad to full width
		padding := e.width - len(line)
		if padding > 0 {
			output.WriteString(strings.Repeat(" ", padding))
		}
		output.WriteString("\033[0m") // Reset color
	}

	return output.String()
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

	// Build menu content
	var menuLines []string

	// Different menu for sequence diagrams
	if e.diagram.Type == string(diagram.DiagramTypeSequence) {
		menuLines = []string{
			"Message: " + fromText + " → " + toText + " | style=" + style + ", color=" + color,
			"Style: [a]Solid [b]Dashed [c]Dotted | Color: [r]Red [g]Green [y]Yellow [u]Blue [m]Magenta [n]Cyan [w]Clear",
			"Text: [o]Bold(" + bold + ") [i]Italic(" + italic + ") | [ESC]Back [Enter]Done",
		}
	} else {
		// Full menu for flowcharts
		menuLines = []string{
			"Connection: " + fromText + " → " + toText + " | style=" + style + ", color=" + color,
			"Style: [a]Solid [b]Dashed [c]Dotted [d]Double | Color: [r]Red [g]Green [y]Yellow [u]Blue [m]Magenta [n]Cyan [w]Clear",
			"Options: [o]Bold(" + bold + ") [i]Italic(" + italic + ") [f]Flow(" + flow + ") | [ESC]Back [Enter]Done",
		}
	}

	// Use absolute positioning to draw menu at bottom of screen
	var output strings.Builder

	// Move to bottom of screen (leave space for menu)
	startLine := e.height - len(menuLines) - 1

	// Clear the menu area and draw menu
	for i, line := range menuLines {
		// Move to position and clear line
		output.WriteString(fmt.Sprintf("\033[%d;1H\033[K", startLine+i))
		// Draw menu line with background color for visibility
		output.WriteString("\033[44m") // Blue background
		output.WriteString(line)
		// Pad to full width
		padding := e.width - len(line)
		if padding > 0 {
			output.WriteString(strings.Repeat(" ", padding))
		}
		output.WriteString("\033[0m") // Reset color
	}

	return output.String()
}

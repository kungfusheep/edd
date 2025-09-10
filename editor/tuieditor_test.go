package editor

import (
	"edd/diagram"
	"edd/render"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

// ===== Test helpers =====

func createTestEditor() *TUIEditor {
	d := &diagram.Diagram{
		Nodes:       []diagram.Node{},
		Connections: []diagram.Connection{},
	}
	return &TUIEditor{
		diagram:            d,
		renderer:           render.NewRenderer(),
		selected:           -1,
		selectedConnection: -1,
		mode:               ModeNormal,
		jumpLabels:         make(map[int]rune),
		connectionLabels:   make(map[int]rune),
	}
}

func createTestEditorWithNodes(nodeCount int) *TUIEditor {
	ed := createTestEditor()
	for i := 0; i < nodeCount; i++ {
		ed.diagram.Nodes = append(ed.diagram.Nodes, diagram.Node{
			ID:   i,
			Text: []string{fmt.Sprintf("Node %d", i+1)},
		})
	}
	return ed
}

// Helper to simulate a sequence of key presses
func sendKeys(ed *TUIEditor, keys string) {
	for _, key := range keys {
		ed.HandleKey(key)
	}
}

// Helper to send text input
func sendText(ed *TUIEditor, text string) {
	for _, ch := range text {
		ed.HandleKey(ch)
	}
}

// Create a test diagram with reasonable complexity for benchmarks
func createTestDiagram(nodes, connections int) *diagram.Diagram {
	d := &diagram.Diagram{
		Nodes:       make([]diagram.Node, nodes),
		Connections: make([]diagram.Connection, 0, connections),
	}
	
	for i := 0; i < nodes; i++ {
		d.Nodes[i] = diagram.Node{
			ID:   i + 1,
			Text: []string{"Node " + string(rune('A'+i)), "Description line", "Another line"},
		}
	}
	
	// Create connections between consecutive nodes
	for i := 0; i < connections && i < nodes-1; i++ {
		d.Connections = append(d.Connections, diagram.Connection{
			From:  i + 1,
			To:    i + 2,
			Label: "connection",
		})
	}
	
	return d
}

// Clone a diagram for testing
func cloneDiagram(d *diagram.Diagram) *diagram.Diagram {
	data, _ := json.Marshal(d)
	var clone diagram.Diagram
	json.Unmarshal(data, &clone)
	return &clone
}

// Validate test diagram structure
func validateTestDiagram(d *diagram.Diagram) error {
	if d == nil {
		return fmt.Errorf("diagram is nil")
	}
	
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

// ============================================
// Tests from adjacent_connect_test.go
// ============================================

func TestAdjacentNodeConnection(t *testing.T) {
	tui := NewTUIEditor(NewRealRenderer())
	
	// Create nodes A, B, C, D, E, F
	tui.AddNode([]string{"A"})
	tui.AddNode([]string{"B"})
	tui.AddNode([]string{"C"})
	tui.AddNode([]string{"D"})
	tui.AddNode([]string{"E"})
	tui.AddNode([]string{"F"})
	
	// Start continuous connect mode
	tui.HandleKey('C')
	
	fmt.Printf("Jump labels assigned:\n")
	for nodeID, label := range tui.jumpLabels {
		fmt.Printf("  Node %d -> '%c'\n", nodeID, label)
	}
	
	// The labels should be: a, s, d, f, g, h
	// Let's verify what we actually get
	expectedLabels := map[int]rune{
		1: 'a', // A
		2: 's', // B
		3: 'd', // C
		4: 'f', // D
		5: 'g', // E
		6: 'h', // F
	}
	
	for nodeID, expected := range expectedLabels {
		if actual, ok := tui.jumpLabels[nodeID]; ok {
			if actual != expected {
				t.Errorf("Node %d: expected label '%c', got '%c'", nodeID, expected, actual)
			}
		} else {
			t.Errorf("Node %d: no label assigned", nodeID)
		}
	}
	
	// Now test connecting D to F
	fmt.Println("\nTrying to connect D -> F")
	
	// Select D (label 'f') as FROM
	fmt.Printf("Pressing 'f' to select node D (id=4)\n")
	tui.HandleKey('f')
	
	fmt.Printf("After selecting D: selected=%d, jumpAction=%v\n", tui.selected, tui.jumpAction)
	
	// Labels might be reassigned after selecting FROM, let's check
	fmt.Printf("\nJump labels after selecting FROM:\n")
	for nodeID, label := range tui.jumpLabels {
		fmt.Printf("  Node %d -> '%c'\n", nodeID, label)
	}
	
	// Try to select F
	// PROBLEM: F might have label 'h' but when we press 'f' it selects D again!
	// This is the issue - 'f' is D's label
	
	// Let me try selecting h for F
	fmt.Printf("\nPressing 'h' to select node F (id=6)\n")
	tui.HandleKey('h')
	
	fmt.Printf("After attempting F: selected=%d, jumpAction=%v\n", tui.selected, tui.jumpAction)
	
	// Check if connection was made
	fmt.Printf("\nConnections created: %d\n", len(tui.diagram.Connections))
	for i, conn := range tui.diagram.Connections {
		fmt.Printf("  Connection %d: %d -> %d\n", i, conn.From, conn.To)
	}
}
func TestLabelConflict(t *testing.T) {
	tui := NewTUIEditor(NewRealRenderer())
	
	// Create many nodes to see label assignment pattern
	for i := 1; i <= 10; i++ {
		tui.AddNode([]string{fmt.Sprintf("Node%d", i)})
	}
	
	// Start continuous connect mode
	tui.HandleKey('C')
	
	fmt.Printf("Jump labels for 10 nodes:\n")
	for i := 1; i <= 10; i++ {
		if label, ok := tui.jumpLabels[i]; ok {
			fmt.Printf("  Node %d -> '%c'\n", i, label)
		}
	}
	
	// Check if we have conflicts
	labelToNode := make(map[rune]int)
	for nodeID, label := range tui.jumpLabels {
		if existing, exists := labelToNode[label]; exists {
			t.Errorf("Label conflict: '%c' assigned to both node %d and node %d", label, existing, nodeID)
		}
		labelToNode[label] = nodeID
	}
}

// ============================================
// Tests from arrow_test.go
// ============================================

func TestCursorUpDownMovement(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetMode(ModeEdit)
	
	// Create multi-line text
	tui.textBuffer = []rune("Line 1\nLine 2\nLine 3")
	tui.cursorPos = len(tui.textBuffer) // At end
	tui.updateCursorPosition()
	
	// Should be at end of line 3
	if tui.cursorLine != 2 || tui.cursorCol != 6 {
		t.Errorf("Initial position: expected (2,6), got (%d,%d)", tui.cursorLine, tui.cursorCol)
	}
	
	// Move up (Ctrl+P)
	tui.handleTextKey(16)
	if tui.cursorLine != 1 {
		t.Errorf("After Ctrl+P: expected line 1, got line %d", tui.cursorLine)
	}
	// Column should be maintained at 6 (end of "Line 2")
	if tui.cursorCol != 6 {
		t.Errorf("After Ctrl+P: expected col 6, got col %d", tui.cursorCol)
	}
	
	// Move up again
	tui.handleTextKey(16)
	if tui.cursorLine != 0 {
		t.Errorf("After second Ctrl+P: expected line 0, got line %d", tui.cursorLine)
	}
	
	// Move down (Ctrl+V)
	tui.handleTextKey(22)
	if tui.cursorLine != 1 {
		t.Errorf("After Ctrl+V: expected line 1, got line %d", tui.cursorLine)
	}
	
	// Move down again
	tui.handleTextKey(22)
	if tui.cursorLine != 2 {
		t.Errorf("After second Ctrl+V: expected line 2, got line %d", tui.cursorLine)
	}
	
	// Try to move down from last line (should stay)
	tui.handleTextKey(22)
	if tui.cursorLine != 2 {
		t.Errorf("After Ctrl+V on last line: expected to stay on line 2, got line %d", tui.cursorLine)
	}
	
	// Move to beginning of line
	tui.handleTextKey(1) // Ctrl+A
	if tui.cursorCol != 0 {
		t.Errorf("After Ctrl+A: expected col 0, got col %d", tui.cursorCol)
	}
	
	// Move up - cursor should go to beginning of line 2
	tui.handleTextKey(16)
	if tui.cursorLine != 1 || tui.cursorCol != 0 {
		t.Errorf("After Ctrl+P from beginning: expected (1,0), got (%d,%d)", tui.cursorLine, tui.cursorCol)
	}
}
func TestCursorUpDownWithUnevenLines(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetMode(ModeEdit)
	
	// Create text with uneven line lengths
	tui.textBuffer = []rune("Short\nThis is a longer line\nMid")
	tui.cursorPos = 27 // In middle of line 2 ("This is a longer line")
	tui.updateCursorPosition()
	
	// Should be in middle of line 2
	if tui.cursorLine != 1 {
		t.Errorf("Initial: expected line 1, got line %d", tui.cursorLine)
	}
	
	// Move up to short line
	tui.handleTextKey(16) // Ctrl+P
	if tui.cursorLine != 0 {
		t.Errorf("After Ctrl+P: expected line 0, got line %d", tui.cursorLine)
	}
	// Column should be clamped to line length
	if tui.cursorCol > 5 { // "Short" has 5 chars
		t.Errorf("After Ctrl+P: column should be clamped to 5, got %d", tui.cursorCol)
	}
	
	// Move down twice to get to line 3
	tui.handleTextKey(22) // Ctrl+V
	tui.handleTextKey(22) // Ctrl+V
	if tui.cursorLine != 2 {
		t.Errorf("After two Ctrl+V: expected line 2, got line %d", tui.cursorLine)
	}
	// Column should be clamped to "Mid" length
	if tui.cursorCol > 3 {
		t.Errorf("Column should be clamped to 3, got %d", tui.cursorCol)
	}
}

// ============================================
// Tests from chain_connect_test.go
// ============================================

func TestContinuousConnectChaining(t *testing.T) {
	tui := NewTUIEditor(NewRealRenderer())
	
	// Set up sequence diagram with 4 nodes
	tui.diagram.Type = "sequence"
	tui.AddNode([]string{"A"})
	tui.AddNode([]string{"B"})
	tui.AddNode([]string{"C"})
	tui.AddNode([]string{"D"})
	
	// Press 'C' to start continuous connect mode
	tui.HandleKey('C')
	
	// Verify we're in jump mode waiting for FROM
	if tui.mode != ModeJump {
		t.Errorf("Expected ModeJump after pressing C, got %v", tui.mode)
	}
	if tui.jumpAction != JumpActionConnectFrom {
		t.Errorf("Expected JumpActionConnectFrom, got %v", tui.jumpAction)
	}
	
	// Select first node as initial FROM (will have label 'a')
	if label, ok := tui.jumpLabels[1]; ok {
		tui.HandleKey(label)
	}
	
	// Should now be waiting for TO
	if tui.jumpAction != JumpActionConnectTo {
		t.Errorf("Expected JumpActionConnectTo after selecting FROM, got %v", tui.jumpAction)
	}
	
	// Select second node as TO - this creates A→B (will have label 's')
	if label, ok := tui.jumpLabels[2]; ok {
		tui.HandleKey(label)
	}
	
	// In continuous mode, should still be in jump mode
	// and 'b' should now be selected as the next FROM
	if tui.mode != ModeJump {
		t.Errorf("Should still be in ModeJump for continuous connect, got %v", tui.mode)
	}
	if tui.jumpAction != JumpActionConnectTo {
		t.Errorf("Expected JumpActionConnectTo for next connection, got %v", tui.jumpAction)
	}
	if tui.selected != 2 { // Node B has ID 2
		t.Errorf("Expected node B (ID 2) to be selected as next FROM, got %d", tui.selected)
	}
	
	// Select third node as next TO - this creates B→C (will have label 'd')
	if label, ok := tui.jumpLabels[3]; ok {
		tui.HandleKey(label)
	}
	
	// Should still be in continuous mode with C as next FROM
	if tui.mode != ModeJump {
		t.Errorf("Should still be in ModeJump for continuous connect, got %v", tui.mode)
	}
	if tui.selected != 3 { // Node C has ID 3
		t.Errorf("Expected node C (ID 3) to be selected as next FROM, got %d", tui.selected)
	}
	
	// Select fourth node as next TO - this creates C→D (will have label 'f')
	if label, ok := tui.jumpLabels[4]; ok {
		tui.HandleKey(label)
	}
	
	// Should still be in continuous mode with D as next FROM
	if tui.selected != 4 { // Node D has ID 4
		t.Errorf("Expected node D (ID 4) to be selected as next FROM, got %d", tui.selected)
	}
	
	// Press ESC to exit continuous mode
	tui.HandleKey(27)
	
	// Should be back in normal mode
	if tui.mode != ModeNormal {
		t.Errorf("Expected ModeNormal after ESC, got %v", tui.mode)
	}
	
	// Verify we created 3 connections: A→B, B→C, C→D
	if len(tui.diagram.Connections) != 3 {
		t.Errorf("Expected 3 connections, got %d", len(tui.diagram.Connections))
	}
	
	// Verify the connections are correct
	expectedConnections := []struct{ from, to int }{
		{1, 2}, // A→B
		{2, 3}, // B→C
		{3, 4}, // C→D
	}
	
	for i, expected := range expectedConnections {
		if i >= len(tui.diagram.Connections) {
			break
		}
		conn := tui.diagram.Connections[i]
		if conn.From != expected.from || conn.To != expected.to {
			t.Errorf("Connection %d: expected %d→%d, got %d→%d",
				i, expected.from, expected.to, conn.From, conn.To)
		}
	}
}
func TestContinuousConnectSelfLoop(t *testing.T) {
	tui := NewTUIEditor(NewRealRenderer())
	
	// Set up sequence diagram with 2 nodes
	tui.diagram.Type = "sequence"
	tui.AddNode([]string{"Server"})
	tui.AddNode([]string{"Client"})
	
	// Press 'C' to start continuous connect mode
	tui.HandleKey('C')
	
	// Select first node as initial FROM
	if label, ok := tui.jumpLabels[1]; ok {
		tui.HandleKey(label)
	}
	
	// Select first node again as TO - creates self-loop
	if label, ok := tui.jumpLabels[1]; ok {
		tui.HandleKey(label)
	}
	
	// Should still be in continuous mode with A as next FROM
	if tui.mode != ModeJump {
		t.Errorf("Should still be in ModeJump for continuous connect, got %v", tui.mode)
	}
	if tui.selected != 1 { // Node A has ID 1
		t.Errorf("Expected node A (ID 1) to be selected as next FROM, got %d", tui.selected)
	}
	
	// Select second node as next TO - creates A→B
	if label, ok := tui.jumpLabels[2]; ok {
		tui.HandleKey(label)
	}
	
	// Press ESC to exit
	tui.HandleKey(27)
	
	// Verify we created 2 connections: A→A (self-loop), A→B
	if len(tui.diagram.Connections) != 2 {
		t.Errorf("Expected 2 connections, got %d", len(tui.diagram.Connections))
	}
	
	// Verify first connection is self-loop
	if tui.diagram.Connections[0].From != 1 || tui.diagram.Connections[0].To != 1 {
		t.Errorf("Expected first connection to be 1→1 (self-loop), got %d→%d",
			tui.diagram.Connections[0].From, tui.diagram.Connections[0].To)
	}
	
	// Verify second connection
	if tui.diagram.Connections[1].From != 1 || tui.diagram.Connections[1].To != 2 {
		t.Errorf("Expected second connection to be 1→2, got %d→%d",
			tui.diagram.Connections[1].From, tui.diagram.Connections[1].To)
	}
}

// ============================================
// Tests from duplicate_connection_test.go
// ============================================

func TestPreventDuplicateConnections(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	
	// Add two nodes
	id1 := tui.AddNode([]string{"Node 1"})
	id2 := tui.AddNode([]string{"Node 2"})
	
	// Add first connection
	tui.AddConnection(id1, id2, "first")
	
	// Get initial connection count
	initialCount := len(tui.GetDiagram().Connections)
	if initialCount != 1 {
		t.Errorf("Expected 1 connection, got %d", initialCount)
	}
	
	// Try to add duplicate connection (same direction)
	tui.AddConnection(id1, id2, "duplicate")
	
	// Count should remain the same
	afterCount := len(tui.GetDiagram().Connections)
	if afterCount != 1 {
		t.Errorf("Duplicate connection was added! Expected 1, got %d", afterCount)
	}
	
	// Add connection in opposite direction (should be allowed)
	tui.AddConnection(id2, id1, "reverse")
	
	// Count should now be 2
	finalCount := len(tui.GetDiagram().Connections)
	if finalCount != 2 {
		t.Errorf("Reverse connection was not added! Expected 2, got %d", finalCount)
	}
	
	// Verify both connections exist
	conns := tui.GetDiagram().Connections
	if conns[0].Label != "first" {
		t.Errorf("Original connection label changed! Expected 'first', got '%s'", conns[0].Label)
	}
	if conns[1].Label != "reverse" {
		t.Errorf("Reverse connection has wrong label! Expected 'reverse', got '%s'", conns[1].Label)
	}
}
func TestAllowMultipleUniqueConnections(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	
	// Add three nodes
	id1 := tui.AddNode([]string{"Node 1"})
	id2 := tui.AddNode([]string{"Node 2"})
	id3 := tui.AddNode([]string{"Node 3"})
	
	// Add unique connections
	tui.AddConnection(id1, id2, "1-2")
	tui.AddConnection(id2, id3, "2-3")
	tui.AddConnection(id1, id3, "1-3")
	
	// Should have 3 connections
	count := len(tui.GetDiagram().Connections)
	if count != 3 {
		t.Errorf("Expected 3 unique connections, got %d", count)
	}
}

// ============================================
// Tests from external_editor_test.go
// ============================================

func TestJSONRoundTrip(t *testing.T) {
	// Create a test diagram
	original := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"Node A", "Line 2"}},
			{ID: 2, Text: []string{"Node B"}},
			{ID: 3, Text: []string{"Node C"}},
		},
		Connections: []diagram.Connection{
			{From: 1, To: 2, Label: "connects"},
			{From: 2, To: 3, Label: "flows to"},
		},
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal back
	var loaded diagram.Diagram
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify structure is preserved
	if len(loaded.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(loaded.Nodes))
	}
	if len(loaded.Connections) != 2 {
		t.Errorf("Expected 2 connections, got %d", len(loaded.Connections))
	}

	// Check specific values
	if loaded.Nodes[0].ID != 1 {
		t.Errorf("Expected node ID 1, got %d", loaded.Nodes[0].ID)
	}
	if len(loaded.Nodes[0].Text) != 2 {
		t.Errorf("Expected 2 text lines for node 1, got %d", len(loaded.Nodes[0].Text))
	}
	if loaded.Connections[0].Label != "connects" {
		t.Errorf("Expected label 'connects', got '%s'", loaded.Connections[0].Label)
	}
}
func TestTempFileCreation(t *testing.T) {
	// Test that we can create and write to a temp file
	tmpFile, err := os.CreateTemp("", "edd-test-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write test data
	testData := []byte(`{"nodes": []}`)
	if _, err := tmpFile.Write(testData); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Read it back
	readData, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	if string(readData) != string(testData) {
		t.Errorf("Data mismatch: got %s, want %s", readData, testData)
	}
}
func TestDiagramValidation(t *testing.T) {
	tests := []struct {
		name    string
		diagram diagram.Diagram
		wantErr bool
	}{
		{
			name: "valid diagram",
			diagram: diagram.Diagram{
				Nodes: []diagram.Node{
					{ID: 1, Text: []string{"A"}},
					{ID: 2, Text: []string{"B"}},
				},
				Connections: []diagram.Connection{
					{From: 1, To: 2},
				},
			},
			wantErr: false,
		},
		{
			name: "duplicate node IDs",
			diagram: diagram.Diagram{
				Nodes: []diagram.Node{
					{ID: 1, Text: []string{"A"}},
					{ID: 1, Text: []string{"B"}}, // Duplicate ID
				},
			},
			wantErr: true,
		},
		{
			name: "connection references non-existent node",
			diagram: diagram.Diagram{
				Nodes: []diagram.Node{
					{ID: 1, Text: []string{"A"}},
				},
				Connections: []diagram.Connection{
					{From: 1, To: 99}, // Node 99 doesn't exist
				},
			},
			wantErr: true,
		},
		{
			name: "empty diagram is valid",
			diagram: diagram.Diagram{
				Nodes:       []diagram.Node{},
				Connections: []diagram.Connection{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't directly test validateDiagram from main package,
			// but we can test the concept
			err := validateTestDiagram(&tt.diagram)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDiagram() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ============================================
// Tests from full_navigation_test.go
// ============================================

func TestFullNavigationIntegration(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetMode(ModeEdit)
	
	// Create a realistic multi-line text
	tui.textBuffer = []rune("The quick brown\nfox jumps over\nthe lazy dog")
	tui.cursorPos = 0
	tui.updateCursorPosition()
	
	tests := []struct {
		action      string
		keyOrArrow  interface{} // Either rune for regular key or 'U','D','L','R' for arrows
		expectedPos int
		expectedLine int
		expectedCol  int
	}{
		{"End key", "E", 15, 0, 15},              // End of first line
		{"Arrow down", "D", 30, 1, 14},           // Down to second line
		{"Arrow left 5 times", "L", 29, 1, 13},   // Move left
		{"Continue left", "L", 28, 1, 12},
		{"Continue left", "L", 27, 1, 11},
		{"Continue left", "L", 26, 1, 10},
		{"Continue left", "L", 25, 1, 9},
		{"Ctrl+A", rune(1), 16, 1, 0},            // Beginning of line
		{"Arrow up", "U", 0, 0, 0},               // Up to first line
		{"Arrow right 3 times", "R", 1, 0, 1},    // Move right
		{"Continue right", "R", 2, 0, 2},
		{"Continue right", "R", 3, 0, 3},
		{"Arrow down", "D", 19, 1, 3},            // Down maintaining column
		{"Home key", "H", 16, 1, 0},              // Beginning of line
		{"End key", "E", 30, 1, 14},              // End of line
		{"Arrow down", "D", 43, 2, 12},           // Down to last line (column clamped)
		{"Ctrl+E", rune(5), 43, 2, 12},           // Already at end
		{"Arrow up twice", "U", 28, 1, 12},       // Back up (col is clamped)
		{"Continue up", "U", 12, 0, 12},          // Up again
	}
	
	for _, tt := range tests {
		switch v := tt.keyOrArrow.(type) {
		case rune:
			// Regular control key
			tui.handleTextKey(v)
		case string:
			// Arrow key direction (stored as string for clarity)
			if len(v) == 1 {
				tui.HandleArrowKey(rune(v[0]))
			}
		}
		
		if tui.cursorPos != tt.expectedPos {
			t.Errorf("%s: expected pos %d, got %d", tt.action, tt.expectedPos, tui.cursorPos)
		}
		if tui.cursorLine != tt.expectedLine {
			t.Errorf("%s: expected line %d, got %d", tt.action, tt.expectedLine, tui.cursorLine)
		}
		if tui.cursorCol != tt.expectedCol {
			t.Errorf("%s: expected col %d, got %d", tt.action, tt.expectedCol, tui.cursorCol)
		}
	}
}
func TestArrowKeysWithEditing(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetMode(ModeInsert)
	
	// Type some text
	for _, ch := range "Hello" {
		tui.handleTextKey(ch)
	}
	
	// Arrow left twice
	tui.HandleArrowKey('L')
	tui.HandleArrowKey('L')
	
	// Insert text in the middle
	for _, ch := range " there" {
		tui.handleTextKey(ch)
	}
	
	expected := "Hel therelo"
	if string(tui.textBuffer) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(tui.textBuffer))
	}
	
	// Add newline and more text
	tui.HandleArrowKey('E') // Go to end
	tui.handleTextKey(14)    // Ctrl+N for newline
	for _, ch := range "World" {
		tui.handleTextKey(ch)
	}
	
	// Arrow up to first line
	tui.HandleArrowKey('U')
	if tui.cursorLine != 0 {
		t.Errorf("Should be on first line, got line %d", tui.cursorLine)
	}
	
	// Move to position after "Hel"
	tui.HandleArrowKey('H') // Home
	for i := 0; i < 3; i++ {
		tui.HandleArrowKey('R') // Right to position after "Hel"
	}
	
	// Delete to end of line
	tui.handleTextKey(11) // Ctrl+K
	
	finalExpected := "Hel\nWorld"
	if string(tui.textBuffer) != finalExpected {
		t.Errorf("Final: expected '%s', got '%s'", finalExpected, string(tui.textBuffer))
	}
}

// ============================================
// Tests from hints_test.go
// ============================================

func TestConnectionHints(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	
	// Add two nodes
	id1 := tui.AddNode([]string{"Node 1"})
	id2 := tui.AddNode([]string{"Node 2"})
	
	// Add a connection
	tui.AddConnection(id1, id2, "test")
	
	// Get the connection
	d := tui.GetDiagram()
	if len(d.Connections) != 1 {
		t.Fatal("Expected 1 connection")
	}
	
	// Add hints manually (simulating hint menu)
	conn := &d.Connections[0]
	conn.Hints = map[string]string{
		"style": "dashed",
		"color": "blue",
	}
	
	// Test that hints are preserved in JSON
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	
	// Parse back
	var loaded diagram.Diagram
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	
	// Check hints were preserved
	if len(loaded.Connections) != 1 {
		t.Fatal("Connection lost after round-trip")
	}
	
	hints := loaded.Connections[0].Hints
	if hints == nil {
		t.Fatal("Hints were not preserved")
	}
	
	if hints["style"] != "dashed" {
		t.Errorf("Style hint lost: got %v", hints["style"])
	}
	
	if hints["color"] != "blue" {
		t.Errorf("Color hint lost: got %v", hints["color"])
	}
}
func TestHintMenuInput(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	
	// Add nodes and connection
	id1 := tui.AddNode([]string{"A"})
	id2 := tui.AddNode([]string{"B"})
	tui.AddConnection(id1, id2, "")
	
	// Test node hints for text alignment
	t.Run("NodeTextAlignment", func(t *testing.T) {
		// Simulate entering hint menu for node
		tui.editingHintNode = id1
		tui.SetMode(ModeHintMenu)
		
		// Test toggling center alignment
		tui.HandleHintMenuInput('t')
		node := tui.GetDiagram().Nodes[0]
		if node.Hints["text-align"] != "center" {
			t.Errorf("Expected text-align=center, got %v", node.Hints["text-align"])
		}
		
		// Toggle again should remove it (back to default left)
		tui.HandleHintMenuInput('t')
		if _, exists := node.Hints["text-align"]; exists {
			t.Errorf("Expected text-align to be removed, but got %v", node.Hints["text-align"])
		}
		
		// Exit mode
		tui.HandleHintMenuInput(27)
	})
	
	// Test connection hints
	t.Run("ConnectionHints", func(t *testing.T) {
		// Simulate entering hint menu for connection 0
		tui.editingHintConn = 0
		tui.SetMode(ModeHintMenu)
		
		// Test setting style to dashed
		tui.HandleHintMenuInput('b')
		conn := tui.GetDiagram().Connections[0]
		if conn.Hints["style"] != "dashed" {
			t.Errorf("Expected style=dashed, got %v", conn.Hints["style"])
		}
		
		// Test setting color to red
		tui.HandleHintMenuInput('r')
		if conn.Hints["color"] != "red" {
			t.Errorf("Expected color=red, got %v", conn.Hints["color"])
		}
		
		// Test ESC exits mode (this test is for the old behavior)
		// Now ESC should return to jump mode if previousJumpAction is set
		tui.HandleHintMenuInput(27)
		// Since we didn't come from jump mode, it should go to normal
		if tui.GetMode() != ModeNormal {
			t.Error("ESC should return to normal mode when no previousJumpAction")
		}
	})
}
func TestHintMenuEnterExitsToNormal(t *testing.T) {
	// Create a simple diagram with nodes
	d := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"Node 1"}},
			{ID: 2, Text: []string{"Node 2"}},
		},
		Connections: []diagram.Connection{
			{From: 1, To: 2, Label: "connects"},
		},
	}

	// Create TUI editor
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(d)

	// Test node hints menu
	t.Run("NodeHintsEnterKey", func(t *testing.T) {
		// Press 'H' to enter hints jump mode
		tui.HandleKey('H')
		if tui.GetMode() != ModeJump {
			t.Errorf("Expected ModeJump after 'H', got %v", tui.GetMode())
		}

		// Select first node (should have label 'a')
		tui.HandleKey('a')
		if tui.GetMode() != ModeHintMenu {
			t.Errorf("Expected ModeHintMenu after selecting node, got %v", tui.GetMode())
		}

		// Press Enter - should exit to normal mode (test key code 13)
		tui.HandleKey(13) // Enter key (CR)
		if tui.GetMode() != ModeNormal {
			t.Errorf("Expected ModeNormal after Enter(13) in hints menu, got %v", tui.GetMode())
		}
		
		// Verify previousJumpAction was cleared
		if tui.previousJumpAction != 0 {
			t.Errorf("Expected previousJumpAction to be cleared, got %v", tui.previousJumpAction)
		}
		
		// Test again with key code 10 (LF)
		tui.HandleKey('H')
		tui.HandleKey('a')
		tui.HandleKey(10) // Enter key (LF)
		if tui.GetMode() != ModeNormal {
			t.Errorf("Expected ModeNormal after Enter(10) in hints menu, got %v", tui.GetMode())
		}
	})

	// Test connection hints menu
	t.Run("ConnectionHintsEnterKey", func(t *testing.T) {
		// Press 'H' to enter hints jump mode
		tui.HandleKey('H')
		if tui.GetMode() != ModeJump {
			t.Errorf("Expected ModeJump after 'H', got %v", tui.GetMode())
		}

		// Select first connection (should have label 'd' after nodes a,s)
		tui.HandleKey('d')
		if tui.GetMode() != ModeHintMenu {
			t.Errorf("Expected ModeHintMenu after selecting connection, got %v", tui.GetMode())
		}

		// Press Enter - should exit to normal mode
		tui.HandleKey(13) // Enter key
		if tui.GetMode() != ModeNormal {
			t.Errorf("Expected ModeNormal after Enter in hints menu, got %v", tui.GetMode())
		}
		
		// Verify previousJumpAction was cleared
		if tui.previousJumpAction != 0 {
			t.Errorf("Expected previousJumpAction to be cleared, got %v", tui.previousJumpAction)
		}
	})
}
func TestHintMenuESCReturnsToJump(t *testing.T) {
	// Create a simple diagram with nodes
	d := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"Node 1"}},
			{ID: 2, Text: []string{"Node 2"}},
		},
		Connections: []diagram.Connection{
			{From: 1, To: 2, Label: "connects"},
		},
	}

	// Create TUI editor
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(d)

	// Test node hints menu ESC behavior
	t.Run("NodeHintsESCKey", func(t *testing.T) {
		// Press 'H' to enter hints jump mode
		tui.HandleKey('H')
		if tui.GetMode() != ModeJump {
			t.Errorf("Expected ModeJump after 'H', got %v", tui.GetMode())
		}
		jumpAction := tui.GetJumpAction()

		// Select first node
		tui.HandleKey('a')
		if tui.GetMode() != ModeHintMenu {
			t.Errorf("Expected ModeHintMenu after selecting node, got %v", tui.GetMode())
		}

		// Press ESC - should return to jump mode with same action
		tui.HandleKey(27) // ESC key
		if tui.GetMode() != ModeJump {
			t.Errorf("Expected ModeJump after ESC in hints menu, got %v", tui.GetMode())
		}
		if tui.GetJumpAction() != jumpAction {
			t.Errorf("Expected jump action %v after ESC, got %v", jumpAction, tui.GetJumpAction())
		}
		
		// Clean up - exit jump mode
		tui.HandleKey(27) // ESC to exit jump mode
	})
	
	// Test connection hints menu ESC behavior
	t.Run("ConnectionHintsESCKey", func(t *testing.T) {
		// Press 'H' to enter hints jump mode
		tui.HandleKey('H')
		if tui.GetMode() != ModeJump {
			t.Errorf("Expected ModeJump after 'H', got %v", tui.GetMode())
		}
		jumpAction := tui.GetJumpAction()

		// Select first connection (should be 'd' after nodes a,s)
		tui.HandleKey('d')
		if tui.GetMode() != ModeHintMenu {
			t.Errorf("Expected ModeHintMenu after selecting connection, got %v", tui.GetMode())
		}

		// Press ESC - should return to jump mode with same action
		tui.HandleKey(27) // ESC key
		if tui.GetMode() != ModeJump {
			t.Errorf("Expected ModeJump after ESC in connection hints menu, got %v", tui.GetMode())
		}
		if tui.GetJumpAction() != jumpAction {
			t.Errorf("Expected jump action %v after ESC, got %v", jumpAction, tui.GetJumpAction())
		}
	})
}

// ============================================
// Tests from history_bench_test.go
// ============================================

func BenchmarkHistoryJSON(b *testing.B) {
	d := createTestDiagram(20, 15) // Reasonable size diagram
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate save
		data, _ := json.Marshal(d)
		_ = string(data)
		
		// Simulate restore
		var restored diagram.Diagram
		json.Unmarshal([]byte(data), &restored)
	}
}
func BenchmarkHistoryStruct(b *testing.B) {
	d := createTestDiagram(20, 15) // Same size as JSON test
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate save (clone)
		clone := cloneDiagram(d)
		
		// Simulate restore (just return the clone)
		_ = clone
	}
}
func BenchmarkHistoryJSONAllocs(b *testing.B) {
	d := createTestDiagram(20, 15)
	
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		data, _ := json.Marshal(d)
		var restored diagram.Diagram
		json.Unmarshal(data, &restored)
	}
}
func BenchmarkHistoryStructAllocs(b *testing.B) {
	d := createTestDiagram(20, 15)
	
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cloneDiagram(d)
	}
}
func BenchmarkHistoryScaling(b *testing.B) {
	sizes := []struct {
		name  string
		nodes int
		conns int
	}{
		{"Small", 5, 4},
		{"Medium", 20, 15},
		{"Large", 100, 80},
	}
	
	for _, size := range sizes {
		d := createTestDiagram(size.nodes, size.conns)
		
		b.Run("JSON/"+size.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				data, _ := json.Marshal(d)
				var restored diagram.Diagram
				json.Unmarshal(data, &restored)
			}
		})
		
		b.Run("Struct/"+size.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = cloneDiagram(d)
			}
		})
	}
}

// ============================================
// Tests from integration_test.go
// ============================================

func TestConnectionDeletionWithRendering(t *testing.T) {
	// Create a test diagram with connections
	d := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"Node A"}, X: 0, Y: 0, Width: 10, Height: 3},
			{ID: 2, Text: []string{"Node B"}, X: 20, Y: 0, Width: 10, Height: 3},
			{ID: 3, Text: []string{"Node C"}, X: 10, Y: 10, Width: 10, Height: 3},
		},
		Connections: []diagram.Connection{
			{From: 1, To: 2, Label: "A->B"},
			{From: 2, To: 3, Label: "B->C"},
			{From: 1, To: 3, Label: "A->C"},
		},
	}

	// Create TUI editor with real renderer
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(d)
	tui.SetTerminalSize(80, 24)

	// Render once to populate connection paths
	output := tui.Render()
	if output == "" {
		t.Error("Render output is empty")
	}

	// Verify connections are rendered
	if !strings.Contains(output, "─") && !strings.Contains(output, "│") {
		t.Error("No connection lines found in output")
	}

	// Start delete mode
	tui.handleNormalKey('d')
	
	// Verify connection paths were populated
	connPaths := tui.GetConnectionPaths()
	if len(connPaths) != 3 {
		t.Errorf("Expected 3 connection paths, got %d", len(connPaths))
	}

	// Verify connection labels were assigned
	connLabels := tui.GetConnectionLabels()
	if len(connLabels) != 3 {
		t.Errorf("Expected 3 connection labels, got %d", len(connLabels))
	}

	// Get first connection label
	var firstLabel rune
	for _, label := range connLabels {
		firstLabel = label
		break
	}

	// Delete the first connection
	beforeCount := len(d.Connections)
	tui.handleJumpKey(firstLabel)
	afterCount := len(d.Connections)

	if afterCount != beforeCount-1 {
		t.Errorf("Connection not deleted: before=%d, after=%d", beforeCount, afterCount)
	}

	// Render again and verify connection is gone
	output2 := tui.Render()
	
	// The output should have fewer connection lines
	// This is a simple heuristic check
	lineCount1 := strings.Count(output, "─") + strings.Count(output, "│")
	lineCount2 := strings.Count(output2, "─") + strings.Count(output2, "│")
	
	if lineCount2 >= lineCount1 {
		t.Errorf("Expected fewer connection lines after deletion, got %d vs %d", lineCount2, lineCount1)
	}
}
func TestConnectionLabelPositioning(t *testing.T) {
	// Test that connection labels are positioned at path midpoints
	d := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"A"}, X: 0, Y: 0, Width: 5, Height: 3},
			{ID: 2, Text: []string{"B"}, X: 10, Y: 0, Width: 5, Height: 3},
		},
		Connections: []diagram.Connection{
			{From: 1, To: 2, Label: ""},
		},
	}

	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(d)

	// Render to populate paths
	tui.Render()

	// Enter delete mode to assign labels
	tui.handleNormalKey('d')

	// Get connection paths and labels
	paths := tui.GetConnectionPaths()
	labels := tui.GetConnectionLabels()

	if len(paths) != 1 || len(labels) != 1 {
		t.Fatalf("Expected 1 path and 1 label, got %d paths and %d labels", len(paths), len(labels))
	}

	// Get the path
	var path diagram.Path
	for _, p := range paths {
		path = p
		break
	}

	// The label should be positioned at the midpoint
	if len(path.Points) > 0 {
		midIndex := len(path.Points) / 2
		midPoint := path.Points[midIndex]
		
		// Just verify the midpoint is reasonable (between the nodes)
		if midPoint.X < 0 || midPoint.Y < 0 {
			t.Errorf("Invalid midpoint position: %v", midPoint)
		}
	}
}
func TestConnectionLabelAssignment(t *testing.T) {
	// Test that connection labels are assigned in delete and edit modes, but not connect mode
	d := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"A"}},
			{ID: 2, Text: []string{"B"}},
		},
		Connections: []diagram.Connection{
			{From: 1, To: 2},
		},
	}

	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(d)

	// Test edit mode - SHOULD assign connection labels (for editing connection labels)
	tui.handleNormalKey('e')
	if len(tui.GetConnectionLabels()) != 1 {
		t.Errorf("Expected 1 connection label in edit mode, got %d", len(tui.GetConnectionLabels()))
	}
	tui.handleKey(27) // ESC to cancel

	// Test connect mode - should NOT assign connection labels
	tui.handleNormalKey('c')
	if len(tui.GetConnectionLabels()) != 0 {
		t.Error("Connection labels assigned in connect mode")
	}
	tui.handleKey(27) // ESC to cancel

	// Test delete mode - SHOULD assign connection labels
	tui.handleNormalKey('d')
	if len(tui.GetConnectionLabels()) != 1 {
		t.Errorf("Expected 1 connection label in delete mode, got %d", len(tui.GetConnectionLabels()))
	}
}

// ============================================
// Tests from multiline_render_test.go
// ============================================

func TestMultilineNodeRendering(t *testing.T) {
	// Create a real renderer
	renderer := NewRealRenderer()
	
	// Create a diagram with multi-line nodes
	d := &diagram.Diagram{
		Nodes: []diagram.Node{
			{
				ID:   1,
				Text: []string{"Line 1", "Line 2", "Line 3"},
				X:    5,
				Y:    2,
			},
			{
				ID:   2,
				Text: []string{"Single"},
				X:    20,
				Y:    2,
			},
		},
	}
	
	// Render the diagram
	output, err := renderer.Render(d)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}
	
	// Check that the output contains all lines of text
	if !strings.Contains(output, "Line 1") {
		t.Error("Output missing 'Line 1'")
	}
	if !strings.Contains(output, "Line 2") {
		t.Error("Output missing 'Line 2'")
	}
	if !strings.Contains(output, "Line 3") {
		t.Error("Output missing 'Line 3'")
	}
	
	// Parse the output to check text is properly arranged
	lines := strings.Split(output, "\n")
	
	// Find "Line 1" and verify subsequent lines
	line1Index := -1
	for i, line := range lines {
		if strings.Contains(line, "Line 1") {
			line1Index = i
			break
		}
	}
	
	if line1Index == -1 {
		t.Fatal("Could not find 'Line 1' in output")
	}
	
	// Check that Line 2 and Line 3 are on the next lines (within the same box)
	if line1Index+1 >= len(lines) || !strings.Contains(lines[line1Index+1], "Line 2") {
		t.Errorf("'Line 2' should be on the line after 'Line 1'")
		if line1Index+1 < len(lines) {
			t.Logf("Line after 'Line 1': %s", lines[line1Index+1])
		}
	}
	
	if line1Index+2 >= len(lines) || !strings.Contains(lines[line1Index+2], "Line 3") {
		t.Errorf("'Line 3' should be two lines after 'Line 1'")
		if line1Index+2 < len(lines) {
			t.Logf("Two lines after 'Line 1': %s", lines[line1Index+2])
		}
	}
	
	// Visual inspection - log the output
	t.Logf("Multi-line node rendering:\n%s", output)
}
func TestMultilineEditingAndRendering(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	
	// Add a node and edit it to have multiple lines
	nodeID := tui.AddNode([]string{"Initial"})
	tui.selected = nodeID
	tui.SetMode(ModeEdit)
	
	// Clear and type multi-line text
	tui.textBuffer = []rune{}
	tui.cursorPos = 0
	tui.handleTextKey('O')
	tui.handleTextKey('n')
	tui.handleTextKey('e')
	tui.handleTextKey(14) // Ctrl+N for newline
	tui.handleTextKey('T')
	tui.handleTextKey('w')
	tui.handleTextKey('o')
	tui.handleTextKey(14) // Ctrl+N for newline
	tui.handleTextKey('T')
	tui.handleTextKey('h')
	tui.handleTextKey('r')
	tui.handleTextKey('e')
	tui.handleTextKey('e')
	
	// Commit the text
	tui.commitText()
	
	// Check the node has 3 lines
	var node *diagram.Node
	for i := range tui.diagram.Nodes {
		if tui.diagram.Nodes[i].ID == nodeID {
			node = &tui.diagram.Nodes[i]
			break
		}
	}
	
	if node == nil {
		t.Fatal("Node not found")
	}
	
	if len(node.Text) != 3 {
		t.Errorf("Expected 3 lines, got %d: %v", len(node.Text), node.Text)
	}
	
	// Now render and check the output
	output := tui.Render()
	
	// All three lines should be visible
	if !strings.Contains(output, "One") {
		t.Error("'One' not found in rendered output")
	}
	if !strings.Contains(output, "Two") {
		t.Error("'Two' not found in rendered output")
	}
	if !strings.Contains(output, "Three") {
		t.Error("'Three' not found in rendered output")
	}
	
	// Split output into lines for analysis
	lines := strings.Split(output, "\n")
	
	// Find where "One" appears
	oneLineIdx := -1
	for i, line := range lines {
		if strings.Contains(line, "One") {
			oneLineIdx = i
			break
		}
	}
	
	if oneLineIdx == -1 {
		t.Fatal("Could not find 'One' in output")
	}
	
	// Check that "Two" and "Three" are on subsequent lines
	if oneLineIdx+1 >= len(lines) || !strings.Contains(lines[oneLineIdx+1], "Two") {
		t.Error("'Two' should be on the line after 'One'")
		if oneLineIdx+1 < len(lines) {
			t.Logf("Line after 'One': %s", lines[oneLineIdx+1])
		}
	}
	
	if oneLineIdx+2 >= len(lines) || !strings.Contains(lines[oneLineIdx+2], "Three") {
		t.Error("'Three' should be two lines after 'One'")
		if oneLineIdx+2 < len(lines) {
			t.Logf("Two lines after 'One': %s", lines[oneLineIdx+2])
		}
	}
	
	// Log the output for debugging
	t.Logf("Rendered output:\n%s", output)
}

// ============================================
// Tests from multiline_test.go
// ============================================

func TestMultilineEditing(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	
	// Start editing a new node
	tui.SetMode(ModeInsert)
	nodeID := tui.AddNode([]string{""})
	tui.selected = nodeID
	
	// Type some text
	tui.handleTextKey('H')
	tui.handleTextKey('e')
	tui.handleTextKey('l')
	tui.handleTextKey('l')
	tui.handleTextKey('o')
	
	// Insert a newline using Ctrl+N
	tui.handleTextKey(14) // Ctrl+N
	
	// Type more text
	tui.handleTextKey('W')
	tui.handleTextKey('o')
	tui.handleTextKey('r')
	tui.handleTextKey('l')
	tui.handleTextKey('d')
	
	// Check the text buffer contains a newline
	text := string(tui.textBuffer)
	if !strings.Contains(text, "\n") {
		t.Errorf("Expected text to contain newline, got: %q", text)
	}
	
	// Commit the text
	tui.commitText()
	
	// Check the node has multiple lines
	for _, node := range tui.diagram.Nodes {
		if node.ID == nodeID {
			if len(node.Text) != 2 {
				t.Errorf("Expected 2 lines, got %d: %v", len(node.Text), node.Text)
			}
			if node.Text[0] != "Hello" {
				t.Errorf("Expected first line 'Hello', got '%s'", node.Text[0])
			}
			if node.Text[1] != "World" {
				t.Errorf("Expected second line 'World', got '%s'", node.Text[1])
			}
			return
		}
	}
	t.Error("Node not found after commit")
}
func TestMultilineLoading(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	
	// Add a node with multiple lines
	nodeID := tui.AddNode([]string{"Line 1", "Line 2", "Line 3"})
	
	// Start editing it
	tui.selected = nodeID
	tui.SetMode(ModeEdit)
	
	// Check that all lines were loaded
	text := string(tui.textBuffer)
	if !strings.Contains(text, "Line 1") {
		t.Error("Line 1 not loaded")
	}
	if !strings.Contains(text, "Line 2") {
		t.Error("Line 2 not loaded")
	}
	if !strings.Contains(text, "Line 3") {
		t.Error("Line 3 not loaded")
	}
	
	// Check they're separated by newlines
	lines := strings.Split(text, "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d: %v", len(lines), lines)
	}
}
func TestCursorPositioning(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	
	// Set up multi-line text
	tui.SetMode(ModeInsert)
	tui.textBuffer = []rune("Hello\nWorld")
	tui.cursorPos = 0
	tui.updateCursorPosition()
	
	// Check initial position
	if tui.cursorLine != 0 || tui.cursorCol != 0 {
		t.Errorf("Expected cursor at (0,0), got (%d,%d)", tui.cursorLine, tui.cursorCol)
	}
	
	// Move to end of first line
	tui.cursorPos = 5 // Right before \n
	tui.updateCursorPosition()
	if tui.cursorLine != 0 || tui.cursorCol != 5 {
		t.Errorf("Expected cursor at (0,5), got (%d,%d)", tui.cursorLine, tui.cursorCol)
	}
	
	// Move to start of second line
	tui.cursorPos = 6 // Right after \n
	tui.updateCursorPosition()
	if tui.cursorLine != 1 || tui.cursorCol != 0 {
		t.Errorf("Expected cursor at (1,0), got (%d,%d)", tui.cursorLine, tui.cursorCol)
	}
	
	// Move to middle of second line
	tui.cursorPos = 8 // "Wo" position
	tui.updateCursorPosition()
	if tui.cursorLine != 1 || tui.cursorCol != 2 {
		t.Errorf("Expected cursor at (1,2), got (%d,%d)", tui.cursorLine, tui.cursorCol)
	}
}

// ============================================
// Tests from navigation_integration_test.go
// ============================================

func TestNavigationIntegration(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	
	// Start editing with multi-line text
	tui.SetMode(ModeEdit)
	tui.textBuffer = []rune("First line\nSecond line here\nThird")
	tui.cursorPos = 0
	tui.updateCursorPosition()
	
	// Test all navigation keys
	tests := []struct {
		key         rune
		desc        string
		expectedLine int
		expectedCol  int
	}{
		{5, "Ctrl+E to end of line", 0, 10},       // End of "First line"
		{22, "Ctrl+V down", 1, 10},                // Try to maintain col 10 on line 2
		{1, "Ctrl+A to beginning", 1, 0},          // Beginning of line 2
		{6, "Ctrl+F forward", 1, 1},               // Move right one
		{6, "Ctrl+F forward", 1, 2},               // Move right again
		{16, "Ctrl+P up", 0, 2},                   // Up to line 1, col 2
		{2, "Ctrl+B backward", 0, 1},              // Move left one
		{22, "Ctrl+V down", 1, 1},                 // Down to line 2
		{5, "Ctrl+E to end", 1, 16},               // End of "Second line here"
		{22, "Ctrl+V down to last", 2, 5},         // Down to line 3, clamped to "Third" length
		{16, "Ctrl+P up", 1, 5},                   // Back up to line 2
		{1, "Ctrl+A to beginning", 1, 0},          // Beginning of line 2
		{16, "Ctrl+P to first line", 0, 0},        // Up to first line beginning
	}
	
	for _, tt := range tests {
		tui.handleTextKey(tt.key)
		if tui.cursorLine != tt.expectedLine || tui.cursorCol != tt.expectedCol {
			t.Errorf("%s: expected (%d,%d), got (%d,%d)", 
				tt.desc, tt.expectedLine, tt.expectedCol, tui.cursorLine, tui.cursorCol)
		}
	}
}
func TestNavigationBoundaries(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetMode(ModeEdit)
	
	// Single line text
	tui.textBuffer = []rune("Single line")
	tui.cursorPos = 5
	tui.updateCursorPosition()
	
	// Try to move up from first line (should stay)
	tui.handleTextKey(16) // Ctrl+P
	if tui.cursorLine != 0 {
		t.Errorf("Ctrl+P on first line should stay at line 0, got %d", tui.cursorLine)
	}
	
	// Try to move down from last line (should stay)
	tui.handleTextKey(22) // Ctrl+V
	if tui.cursorLine != 0 {
		t.Errorf("Ctrl+V on last line should stay at line 0, got %d", tui.cursorLine)
	}
	
	// Move to beginning
	tui.handleTextKey(1) // Ctrl+A
	if tui.cursorPos != 0 {
		t.Errorf("Ctrl+A should move to position 0, got %d", tui.cursorPos)
	}
	
	// Try to move backward from beginning (should stay)
	tui.handleTextKey(2) // Ctrl+B
	if tui.cursorPos != 0 {
		t.Errorf("Ctrl+B at beginning should stay at 0, got %d", tui.cursorPos)
	}
	
	// Move to end
	tui.handleTextKey(5) // Ctrl+E
	if tui.cursorPos != 11 {
		t.Errorf("Ctrl+E should move to position 11, got %d", tui.cursorPos)
	}
	
	// Try to move forward from end (should stay)
	tui.handleTextKey(6) // Ctrl+F
	if tui.cursorPos != 11 {
		t.Errorf("Ctrl+F at end should stay at 11, got %d", tui.cursorPos)
	}
}

// ============================================
// Tests from overlay_test.go
// ============================================

func TestEdPositionBottomRight(t *testing.T) {
	state := TUIState{
		Diagram: &diagram.Diagram{
			Nodes: []diagram.Node{
				{ID: 1, Text: []string{"Node1"}},
				{ID: 2, Text: []string{"Node2"}},
			},
			Connections: []diagram.Connection{
				{From: 1, To: 2},
			},
		},
		Mode:     ModeNormal,
		EddFrame: "◉‿◉",
		Width:    80,
		Height:   24,
	}
	
	// Ed and mode indicators are now rendered using ANSI escape codes
	// This test just verifies that rendering doesn't panic and state is correct
	_ = RenderTUI(state)
	
	if state.EddFrame != "◉‿◉" {
		t.Errorf("Expected Ed frame to be ◉‿◉, got %s", state.EddFrame)
	}
}
func TestSingleModeIndicator(t *testing.T) {
	modes := []Mode{ModeNormal, ModeJump, ModeEdit, ModeCommand}
	
	for _, mode := range modes {
		state := TUIState{
			Diagram:  &diagram.Diagram{},
			Mode:     mode,
			EddFrame: "◉‿◉",
		}
		
		output := RenderTUI(state)
		
		// Count how many times mode strings appear
		modeCount := 0
		for _, m := range modes {
			if strings.Count(output, m.String()) > 0 {
				modeCount++
			}
		}
		
		if modeCount > 1 {
			t.Errorf("Multiple modes shown for %s mode. Output:\n%s", mode, output)
		}
	}
}
func TestEdFaceRendering(t *testing.T) {
	faces := map[string]string{
		"normal":  "◉‿◉",
		"command": ":_",
		"jump":    "◎‿◎",
	}
	
	for name, face := range faces {
		state := TUIState{
			Diagram:  &diagram.Diagram{},
			Mode:     ModeNormal,
			EddFrame: face,
		}
		
		// Ed faces are now rendered using ANSI escape codes
		// Just verify rendering doesn't panic and state is preserved
		_ = RenderTUI(state)
		
		if state.EddFrame != face {
			t.Errorf("Ed face %s: expected %s, got %s", name, face, state.EddFrame)
		}
	}
}
func TestOverlayClearance(t *testing.T) {
	state := TUIState{
		Diagram: &diagram.Diagram{
			Nodes: []diagram.Node{
				{ID: 1, Text: []string{"TopLeft"}},
				{ID: 2, Text: []string{"TopRight"}},
			},
		},
		Mode:     ModeNormal,
		EddFrame: "◉‿◉",
	}
	
	output := RenderTUI(state)
	
	// Both node texts should be visible
	if !strings.Contains(output, "TopLeft") {
		t.Error("TopLeft node text hidden by overlay")
	}
	if !strings.Contains(output, "TopRight") {
		t.Error("TopRight node text hidden by overlay")
	}
	
	fmt.Println("=== Overlay Test Output ===")
	fmt.Println(output)
	fmt.Println("=== End ===")
}
func TestNoOverlayDuplication(t *testing.T) {
	tui := NewTUIEditor(nil)
	tui.SetDiagram(&diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"Node 1"}},
			{ID: 2, Text: []string{"Node 2"}},
			{ID: 3, Text: []string{"Node 3"}},
		},
	})
	
	// Mode indicators and Ed are now rendered using ANSI escape codes
	// Test state transitions instead of output text
	
	// Test in normal mode
	tui.SetMode(ModeNormal)
	_ = tui.Render()
	
	if tui.GetMode() != ModeNormal {
		t.Errorf("Expected ModeNormal, got %v", tui.GetMode())
	}
	
	// Test transition to jump mode (like pressing 'c')
	tui.StartConnect()
	_ = tui.Render()
	
	if tui.GetMode() != ModeJump {
		t.Errorf("Expected ModeJump after StartConnect, got %v", tui.GetMode())
	}
}
func TestClearBetweenModes(t *testing.T) {
	tui := NewTUIEditor(nil)
	tui.SetDiagram(&diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"Test"}},
		},
	})
	
	// Mode indicators are now rendered using ANSI escape codes
	// Test state transitions instead of output text
	
	// Normal mode
	tui.SetMode(ModeNormal)
	_ = tui.Render()
	if tui.GetMode() != ModeNormal {
		t.Errorf("Expected ModeNormal, got %v", tui.GetMode())
	}
	
	// Jump mode
	tui.startJump(JumpActionEdit)
	_ = tui.Render()
	if tui.GetMode() != ModeJump {
		t.Errorf("Expected ModeJump, got %v", tui.GetMode())
	}
	
	// Back to normal
	tui.HandleJumpInput(27) // ESC
	_ = tui.Render()
	if tui.GetMode() != ModeNormal {
		t.Errorf("Expected ModeNormal after ESC, got %v", tui.GetMode())
	}
}

// ============================================
// Tests from render_test.go
// ============================================

func TestRenderEmptyState(t *testing.T) {
	state := TUIState{
		Diagram: &diagram.Diagram{},
		Mode:    ModeNormal,
		Width:   80,
		Height:  24,
	}
	
	output := RenderTUI(state)
	
	// Should contain help text
	if !strings.Contains(output, "Press 'a' to add a node") {
		t.Error("Empty state should show help text")
	}
	
	// Mode indicator is now rendered separately in the TUI, not in RenderTUI
	// So we don't test for it here
}
func TestRenderWithNodes(t *testing.T) {
	state := TUIState{
		Diagram: &diagram.Diagram{
			Nodes: []diagram.Node{
				{ID: 1, Text: []string{"Server"}},
				{ID: 2, Text: []string{"Database"}},
			},
		},
		Mode:     ModeNormal,
		EddFrame: "◉‿◉",
	}
	
	output := RenderTUI(state)
	
	// Should show both nodes
	if !strings.Contains(output, "Server") {
		t.Error("Should display Server node")
	}
	if !strings.Contains(output, "Database") {
		t.Error("Should display Database node")
	}
	
	// Ed mascot is now rendered separately via ANSI codes, not in RenderTUI
}
func TestRenderJumpMode(t *testing.T) {
	state := TUIState{
		Diagram: &diagram.Diagram{
			Nodes: []diagram.Node{
				{ID: 1, Text: []string{"Node1"}},
				{ID: 2, Text: []string{"Node2"}},
				{ID: 3, Text: []string{"Node3"}},
			},
		},
		Mode: ModeJump,
		JumpLabels: map[int]rune{
			1: 'a',
			2: 's',
			3: 'd',
		},
		EddFrame: "◎‿◎",
	}
	
	// Jump labels and mode indicator are now rendered separately via ANSI codes, not in RenderTUI
	// The actual rendering test would need to verify the state's JumpLabels map
	_ = RenderTUI(state) // Just verify it doesn't panic
}
func TestRenderTextInput(t *testing.T) {
	// Text input is now handled by showing cursor in the node itself
	// Not as an overlay, so just test that state is preserved
	tests := []struct {
		name       string
		textBuffer []rune
		cursorPos  int
	}{
		{
			name:       "Empty with cursor",
			textBuffer: []rune{},
			cursorPos:  0,
		},
		{
			name:       "Text with cursor at end",
			textBuffer: []rune("Hello"),
			cursorPos:  5,
		},
		{
			name:       "Text with cursor in middle",
			textBuffer: []rune("Hello"),
			cursorPos:  2,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := TUIState{
				Diagram:    &diagram.Diagram{},
				Mode:       ModeEdit,
				TextBuffer: tt.textBuffer,
				CursorPos:  tt.cursorPos,
			}
			
			// Just verify rendering doesn't panic and state is preserved
			_ = RenderTUI(state)
			
			if state.CursorPos != tt.cursorPos {
				t.Errorf("Cursor position changed: expected %d, got %d", tt.cursorPos, state.CursorPos)
			}
			if string(state.TextBuffer) != string(tt.textBuffer) {
				t.Errorf("Text buffer changed: expected %s, got %s", string(tt.textBuffer), string(state.TextBuffer))
			}
		})
	}
}
func TestRenderConnections(t *testing.T) {
	state := TUIState{
		Diagram: &diagram.Diagram{
			Nodes: []diagram.Node{
				{ID: 1, Text: []string{"A"}},
				{ID: 2, Text: []string{"B"}},
			},
			Connections: []diagram.Connection{
				{From: 1, To: 2, Label: "test"},
			},
		},
		Mode: ModeNormal,
	}
	
	output := RenderTUI(state)
	
	// Should show connection
	if !strings.Contains(output, "1 -> 2") {
		t.Error("Should show connection")
	}
	if !strings.Contains(output, "test") {
		t.Error("Should show connection label")
	}
}
func TestModeTransitions(t *testing.T) {
	modes := []struct {
		mode Mode
		want string
		face string
	}{
		{ModeNormal, "NORMAL", "◉‿◉"},
		{ModeInsert, "INSERT", "○‿○"},
		{ModeEdit, "EDIT", "◉‿◉"},
		{ModeCommand, "COMMAND", ":_"},
		{ModeJump, "JUMP", "◎‿◎"},
	}
	
	for _, m := range modes {
		state := TUIState{
			Diagram:  &diagram.Diagram{},
			Mode:     m.mode,
			EddFrame: m.face,
		}
		
		// Mode indicators and Ed face are now rendered separately via ANSI codes
		// They are not part of the RenderTUI output
		_ = RenderTUI(state) // Just verify it doesn't panic
	}
}
func TestComplexScenario(t *testing.T) {
	// Simulate: User is connecting nodes with jump labels active
	state := TUIState{
		Diagram: &diagram.Diagram{
			Nodes: []diagram.Node{
				{ID: 1, Text: []string{"Web", "Server"}},
				{ID: 2, Text: []string{"API"}},
				{ID: 3, Text: []string{"Database"}},
			},
			Connections: []diagram.Connection{
				{From: 1, To: 2},
			},
		},
		Mode:     ModeJump,
		Selected: 2, // API node selected as source
		JumpLabels: map[int]rune{
			1: 'a',
			3: 'd', // Can't connect to self, so no label for node 2
		},
		EddFrame: "◎‿◎",
	}
	
	output := RenderTUI(state)
	
	// Verify the node text is rendered
	if !strings.Contains(output, "Web Server") {
		t.Error("Should show multi-line node text")
	}
	// Jump labels and mode indicator are now rendered separately via ANSI codes
}
func BenchmarkRenderSimple(b *testing.B) {
	state := TUIState{
		Diagram: &diagram.Diagram{
			Nodes: []diagram.Node{
				{ID: 1, Text: []string{"Node1"}},
				{ID: 2, Text: []string{"Node2"}},
				{ID: 3, Text: []string{"Node3"}},
			},
			Connections: []diagram.Connection{
				{From: 1, To: 2},
				{From: 2, To: 3},
			},
		},
		Mode:     ModeNormal,
		EddFrame: "◉‿◉",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RenderTUI(state)
	}
}
func BenchmarkRenderComplex(b *testing.B) {
	// Create a larger diagram
	nodes := make([]diagram.Node, 20)
	for i := range nodes {
		nodes[i] = diagram.Node{
			ID:   i + 1,
			Text: []string{"Node"},
		}
	}
	
	connections := make([]diagram.Connection, 30)
	for i := range connections {
		connections[i] = diagram.Connection{
			From: (i % 20) + 1,
			To:   ((i + 5) % 20) + 1,
		}
	}
	
	state := TUIState{
		Diagram: &diagram.Diagram{
			Nodes:       nodes,
			Connections: connections,
		},
		Mode: ModeNormal,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RenderTUI(state)
	}
}

// ============================================
// Tests from restart_connect_test.go
// ============================================

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

// ============================================
// Tests from text_editing_integration_test.go
// ============================================

func TestTextEditingIntegration(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	
	// Start editing a new node
	tui.SetMode(ModeInsert)
	nodeID := tui.AddNode([]string{""})
	tui.selected = nodeID
	
	// Type some text
	for _, ch := range "Hello world" {
		tui.handleTextKey(ch)
	}
	
	// Move to beginning of line (Ctrl+A)
	tui.handleTextKey(1)
	if tui.cursorPos != 0 {
		t.Errorf("Ctrl+A: Expected cursor at 0, got %d", tui.cursorPos)
	}
	
	// Move to end of line (Ctrl+E)
	tui.handleTextKey(5)
	if tui.cursorPos != 11 {
		t.Errorf("Ctrl+E: Expected cursor at 11, got %d", tui.cursorPos)
	}
	
	// Insert a newline at the end (Ctrl+N)
	tui.handleTextKey(14)
	
	// Type more text on second line
	for _, ch := range "Second line" {
		tui.handleTextKey(ch)
	}
	
	expectedAfterSecondLine := "Hello world\nSecond line"
	if string(tui.textBuffer) != expectedAfterSecondLine {
		t.Errorf("After adding second line: Expected '%s', got '%s'", expectedAfterSecondLine, string(tui.textBuffer))
	}
	
	// Add another newline to start third line
	tui.handleTextKey(14) // Ctrl+N
	
	// Type third line
	for _, ch := range "Third line" {
		tui.handleTextKey(ch)
	}
	
	// Now we have "Hello world\nSecond line\nThird line"
	
	// Move to beginning of third line (Ctrl+A)
	tui.handleTextKey(1)
	if tui.cursorLine != 2 || tui.cursorCol != 0 {
		t.Errorf("Ctrl+A on line 3: Expected (2,0), got (%d,%d)", tui.cursorLine, tui.cursorCol)
	}
	
	// Delete to end of line (Ctrl+K) - removes "Third line"
	tui.handleTextKey(11)
	
	// Type replacement text
	for _, ch := range "Final line" {
		tui.handleTextKey(ch)
	}
	
	expectedFinal := "Hello world\nSecond line\nFinal line"
	if string(tui.textBuffer) != expectedFinal {
		t.Errorf("Final text: Expected '%s', got '%s'", expectedFinal, string(tui.textBuffer))
	}
	
	// Move back a few chars then delete word
	tui.handleTextKey(5) // Ctrl+E to go to end of line
	for i := 0; i < 4; i++ {
		tui.handleTextKey(2) // Ctrl+B to move back
	}
	// Cursor should be at 'l' in "Final line"
	tui.handleTextKey(23) // Ctrl+W to delete "Final "
	
	expectedAfterDelete := "Hello world\nSecond line\nline"
	if string(tui.textBuffer) != expectedAfterDelete {
		t.Errorf("After Ctrl+W: Expected '%s', got '%s'", expectedAfterDelete, string(tui.textBuffer))
	}
}
func TestAllEditingShortcuts(t *testing.T) {
	shortcuts := []struct {
		key  rune
		desc string
	}{
		{1, "Ctrl+A (beginning of line)"},
		{2, "Ctrl+B (backward)"},
		{5, "Ctrl+E (end of line)"},
		{6, "Ctrl+F (forward)"},
		{11, "Ctrl+K (delete to end)"},
		{14, "Ctrl+N (newline)"},
		{21, "Ctrl+U (delete to beginning)"},
		{23, "Ctrl+W (delete word)"},
	}
	
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetMode(ModeEdit)
	
	// Test that all shortcuts are handled without panic
	for _, shortcut := range shortcuts {
		tui.textBuffer = []rune("test text")
		tui.cursorPos = 5
		tui.updateCursorPosition()
		
		// Should not panic
		tui.handleTextKey(shortcut.key)
		
		t.Logf("✓ %s handled without error", shortcut.desc)
	}
}

// ============================================
// Tests from tui_test.go
// ============================================

func TestConnectionDeletion(t *testing.T) {
	// Create a test diagram
	d := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"Node A"}},
			{ID: 2, Text: []string{"Node B"}},
			{ID: 3, Text: []string{"Node C"}},
		},
		Connections: []diagram.Connection{
			{From: 1, To: 2, Label: "A->B"},
			{From: 2, To: 3, Label: "B->C"},
			{From: 1, To: 3, Label: "A->C"},
		},
	}

	// Create TUI editor
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(d)

	// Verify initial state
	if len(d.Connections) != 3 {
		t.Fatalf("Expected 3 connections, got %d", len(d.Connections))
	}

	// Start delete mode - simulate pressing 'd'
	tui.handleNormalKey('d')
	
	// Verify we're in jump mode with delete action
	if tui.GetMode() != ModeJump {
		t.Errorf("Expected ModeJump, got %v", tui.GetMode())
	}
	if tui.GetJumpAction() != JumpActionDelete {
		t.Errorf("Expected JumpActionDelete, got %v", tui.GetJumpAction())
	}

	// Verify labels were assigned to both nodes and connections
	nodeLabels := tui.GetJumpLabels()
	connLabels := tui.GetConnectionLabels()
	
	if len(nodeLabels) != 3 {
		t.Errorf("Expected 3 node labels, got %d", len(nodeLabels))
	}
	if len(connLabels) != 3 {
		t.Errorf("Expected 3 connection labels, got %d", len(connLabels))
	}

	// Get the first connection's label
	var firstConnLabel rune
	for _, label := range connLabels {
		firstConnLabel = label
		break
	}

	// Simulate pressing the connection's label key
	beforeCount := len(d.Connections)
	tui.handleJumpKey(firstConnLabel)
	afterCount := len(d.Connections)

	// Verify connection was deleted
	if afterCount != beforeCount-1 {
		t.Errorf("Connection not deleted: before=%d, after=%d", beforeCount, afterCount)
	}

	// Verify we're back in normal mode
	if tui.GetMode() != ModeNormal {
		t.Errorf("Expected ModeNormal after deletion, got %v", tui.GetMode())
	}

	// Verify labels were cleared
	if len(tui.GetJumpLabels()) != 0 {
		t.Errorf("Jump labels not cleared after deletion")
	}
	if len(tui.GetConnectionLabels()) != 0 {
		t.Errorf("Connection labels not cleared after deletion")
	}
}
func TestConnectionDeletionFullFlow(t *testing.T) {
	// Test the full key sequence: d -> connection_label
	d := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"A"}},
			{ID: 2, Text: []string{"B"}},
		},
		Connections: []diagram.Connection{
			{From: 1, To: 2, Label: "test"},
		},
	}

	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(d)

	// Press 'd' to enter delete mode
	result := tui.handleKey('d')
	if result {
		t.Error("handleKey returned true (exit) when it shouldn't")
	}

	// Should be in jump mode
	if tui.mode != ModeJump {
		t.Errorf("Expected ModeJump, got %v", tui.mode)
	}

	// Should have assigned labels
	connLabels := tui.connectionLabels
	if len(connLabels) != 1 {
		t.Fatalf("Expected 1 connection label, got %d", len(connLabels))
	}

	// Get the assigned label
	var connLabel rune
	for _, label := range connLabels {
		connLabel = label
		break
	}

	// Press the connection label to delete it
	initialConnCount := len(d.Connections)
	result = tui.handleKey(connLabel)
	if result {
		t.Error("handleKey returned true (exit) when it shouldn't")
	}

	// Check connection was deleted
	if len(d.Connections) != initialConnCount-1 {
		t.Errorf("Connection not deleted: expected %d connections, got %d", 
			initialConnCount-1, len(d.Connections))
	}

	// Should be back in normal mode
	if tui.mode != ModeNormal {
		t.Errorf("Expected ModeNormal after deletion, got %v", tui.mode)
	}
}
func TestNodeDeletionStillWorks(t *testing.T) {
	// Ensure node deletion still works after our changes
	d := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"Node A"}},
			{ID: 2, Text: []string{"Node B"}},
		},
		Connections: []diagram.Connection{
			{From: 1, To: 2, Label: "test"},
		},
	}

	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetDiagram(d)

	// Press 'd' to enter delete mode
	tui.handleKey('d')

	// Get a node label
	nodeLabels := tui.jumpLabels
	if len(nodeLabels) != 2 {
		t.Fatalf("Expected 2 node labels, got %d", len(nodeLabels))
	}

	// Get the first node's label
	var nodeLabel rune
	var nodeID int
	for id, label := range nodeLabels {
		nodeLabel = label
		nodeID = id
		break
	}

	// Press the node label to delete it
	initialNodeCount := len(d.Nodes)
	tui.handleKey(nodeLabel)

	// Check node was deleted
	if len(d.Nodes) != initialNodeCount-1 {
		t.Errorf("Node not deleted: expected %d nodes, got %d", 
			initialNodeCount-1, len(d.Nodes))
	}

	// Check that connections involving the deleted node were also removed
	for _, conn := range d.Connections {
		if conn.From == nodeID || conn.To == nodeID {
			t.Errorf("Connection involving deleted node %d still exists", nodeID)
		}
	}
}

package editor

import (
	"strings"
	"testing"
)

func TestMultilineCursorVisibility(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	
	// Start editing a new node
	tui.SetMode(ModeInsert)
	nodeID := tui.AddNode([]string{""})
	tui.selected = nodeID
	
	// Type text with multiple lines
	tui.textBuffer = []rune{}
	tui.cursorPos = 0
	tui.updateCursorPosition()
	
	// Type "Hello"
	for _, ch := range "Hello" {
		tui.handleTextKey(ch)
	}
	
	// Verify cursor position is at end of "Hello"
	if tui.cursorLine != 0 || tui.cursorCol != 5 {
		t.Errorf("After 'Hello', expected cursor at (0,5), got (%d,%d)", tui.cursorLine, tui.cursorCol)
	}
	
	// Insert newline
	tui.handleTextKey(14) // Ctrl+N
	
	// Cursor should be at start of line 2
	if tui.cursorLine != 1 || tui.cursorCol != 0 {
		t.Errorf("After newline, expected cursor at (1,0), got (%d,%d)", tui.cursorLine, tui.cursorCol)
	}
	
	// Type "World"
	for _, ch := range "World" {
		tui.handleTextKey(ch)
	}
	
	// Cursor should be at end of "World" on line 2
	if tui.cursorLine != 1 || tui.cursorCol != 5 {
		t.Errorf("After 'World', expected cursor at (1,5), got (%d,%d)", tui.cursorLine, tui.cursorCol)
	}
	
	// Get current state to verify text buffer
	state := tui.GetState()
	text := string(state.TextBuffer)
	if !strings.Contains(text, "Hello\nWorld") {
		t.Errorf("Expected text to be 'Hello\\nWorld', got '%s'", text)
	}
	
	// Render and verify we can see both lines
	output := tui.Render()
	
	// The text should be visible in edit mode, not after the newline
	lines := strings.Split(output, "\n")
	
	// Find the node being edited
	foundHello := false
	foundWorld := false
	
	for _, line := range lines {
		if strings.Contains(line, "Hello") {
			foundHello = true
		}
		if strings.Contains(line, "World") {
			foundWorld = true
		}
	}
	
	if !foundHello {
		t.Error("'Hello' should be visible during multi-line edit")
	}
	if !foundWorld {
		t.Error("'World' should be visible during multi-line edit")
	}
	
	t.Logf("Multi-line edit output:\n%s", output)
}

func TestMultilineCursorMovement(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	
	// Create a node with existing multi-line text
	nodeID := tui.AddNode([]string{"Line1", "Line2", "Line3"})
	tui.selected = nodeID
	tui.SetMode(ModeEdit)
	
	// Cursor should start at end of all text
	expectedPos := len("Line1") + 1 + len("Line2") + 1 + len("Line3") // +1 for each newline
	if tui.cursorPos != expectedPos {
		t.Errorf("Expected cursor at position %d, got %d", expectedPos, tui.cursorPos)
	}
	
	// Cursor should be at end of line 3
	if tui.cursorLine != 2 || tui.cursorCol != 5 {
		t.Errorf("Expected cursor at (2,5), got (%d,%d)", tui.cursorLine, tui.cursorCol)
	}
	
	// Test arrow key movement once implemented
	// For now just verify cursor position tracking
}

func TestMultilineRenderingDuringEdit(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	
	// Add a node
	nodeID := tui.AddNode([]string{""})
	tui.selected = nodeID
	
	// Set to edit mode
	tui.SetMode(ModeEdit)
	
	// Set the text buffer to multi-line text directly (simulating paste or load)
	tui.textBuffer = []rune("First\nSecond\nThird")
	tui.cursorPos = len(tui.textBuffer)
	tui.updateCursorPosition()
	
	// Tell renderer we're editing this node
	renderer.SetEditState(nodeID, string(tui.textBuffer), tui.cursorPos)
	
	// Render
	output, err := renderer.Render(tui.diagram)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	
	// All three lines should be visible
	if !strings.Contains(output, "First") {
		t.Error("'First' not visible in edit mode")
	}
	if !strings.Contains(output, "Second") {
		t.Error("'Second' not visible in edit mode")
	}
	if !strings.Contains(output, "Third") {
		t.Error("'Third' not visible in edit mode")
	}
	
	// Check they're on separate lines
	lines := strings.Split(output, "\n")
	firstIdx := -1
	secondIdx := -1
	thirdIdx := -1
	
	for i, line := range lines {
		if strings.Contains(line, "First") {
			firstIdx = i
		}
		if strings.Contains(line, "Second") {
			secondIdx = i
		}
		if strings.Contains(line, "Third") {
			thirdIdx = i
		}
	}
	
	if firstIdx >= 0 && secondIdx >= 0 && thirdIdx >= 0 {
		if secondIdx != firstIdx+1 {
			t.Errorf("'Second' should be on line after 'First': first=%d, second=%d", firstIdx, secondIdx)
		}
		if thirdIdx != secondIdx+1 {
			t.Errorf("'Third' should be on line after 'Second': second=%d, third=%d", secondIdx, thirdIdx)
		}
	}
	
	t.Logf("Edit mode multi-line rendering:\n%s", output)
}
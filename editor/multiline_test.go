package editor

import (
	"strings"
	"testing"
)

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
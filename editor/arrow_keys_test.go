package editor

import (
	"testing"
)

func TestArrowKeyNavigation(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetMode(ModeEdit)
	
	// Create multi-line text
	tui.textBuffer = []rune("First line\nSecond line\nThird line")
	tui.cursorPos = 0
	tui.updateCursorPosition()
	
	// Test arrow right
	tui.HandleArrowKey('R')
	if tui.cursorPos != 1 {
		t.Errorf("Arrow right: expected pos 1, got %d", tui.cursorPos)
	}
	
	// Test arrow down
	tui.HandleArrowKey('D')
	if tui.cursorLine != 1 {
		t.Errorf("Arrow down: expected line 1, got %d", tui.cursorLine)
	}
	
	// Test arrow left
	tui.HandleArrowKey('L')
	if tui.cursorCol != 0 {
		t.Errorf("Arrow left: expected col 0, got %d", tui.cursorCol)
	}
	
	// Test arrow up
	tui.HandleArrowKey('U')
	if tui.cursorLine != 0 {
		t.Errorf("Arrow up: expected line 0, got %d", tui.cursorLine)
	}
	
	// Test Home key
	tui.cursorPos = 5 // Middle of first line
	tui.updateCursorPosition()
	tui.HandleArrowKey('H')
	if tui.cursorPos != 0 {
		t.Errorf("Home key: expected pos 0, got %d", tui.cursorPos)
	}
	
	// Test End key
	tui.HandleArrowKey('E')
	if tui.cursorPos != 10 { // End of "First line"
		t.Errorf("End key: expected pos 10, got %d", tui.cursorPos)
	}
}

func TestArrowKeysWithEmptyText(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetMode(ModeEdit)
	
	// Empty text buffer
	tui.textBuffer = []rune{}
	tui.cursorPos = 0
	tui.updateCursorPosition()
	
	// None of these should crash
	tui.HandleArrowKey('U') // Up
	tui.HandleArrowKey('D') // Down
	tui.HandleArrowKey('L') // Left
	tui.HandleArrowKey('R') // Right
	tui.HandleArrowKey('H') // Home
	tui.HandleArrowKey('E') // End
	
	// Cursor should still be at 0
	if tui.cursorPos != 0 {
		t.Errorf("Cursor should remain at 0 with empty text, got %d", tui.cursorPos)
	}
}

func TestArrowKeysNotInEditMode(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetMode(ModeNormal) // Not in edit mode
	
	tui.textBuffer = []rune("Test")
	tui.cursorPos = 0
	
	// Arrow keys should not work in normal mode
	tui.HandleArrowKey('R')
	if tui.cursorPos != 0 {
		t.Errorf("Arrow keys should not work in normal mode")
	}
}
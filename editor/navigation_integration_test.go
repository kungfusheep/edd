package editor

import (
	"testing"
)

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
package editor

import (
	"testing"
)

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
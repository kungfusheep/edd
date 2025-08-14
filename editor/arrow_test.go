package editor

import (
	"testing"
)

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
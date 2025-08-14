package editor

import (
	"testing"
)

func TestCursorMovementToBeginningOfLine(t *testing.T) {
	tests := []struct {
		name           string
		initialText    string
		cursorPos      int
		expectedCursor int
		expectedLine   int
		expectedCol    int
	}{
		{
			name:           "move from middle to beginning",
			initialText:    "hello world",
			cursorPos:      6,
			expectedCursor: 0,
			expectedLine:   0,
			expectedCol:    0,
		},
		{
			name:           "move on second line",
			initialText:    "line1\nline2",
			cursorPos:      9, // at 'n' in line2
			expectedCursor: 6, // right after \n
			expectedLine:   1,
			expectedCol:    0,
		},
		{
			name:           "already at beginning",
			initialText:    "hello",
			cursorPos:      0,
			expectedCursor: 0,
			expectedLine:   0,
			expectedCol:    0,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := NewRealRenderer()
			tui := NewTUIEditor(renderer)
			tui.SetMode(ModeEdit)
			
			tui.textBuffer = []rune(tt.initialText)
			tui.cursorPos = tt.cursorPos
			tui.updateCursorPosition()
			
			tui.moveCursorToBeginningOfLine()
			
			if tui.cursorPos != tt.expectedCursor {
				t.Errorf("Expected cursor at %d, got %d", tt.expectedCursor, tui.cursorPos)
			}
			
			if tui.cursorLine != tt.expectedLine {
				t.Errorf("Expected line %d, got %d", tt.expectedLine, tui.cursorLine)
			}
			
			if tui.cursorCol != tt.expectedCol {
				t.Errorf("Expected col %d, got %d", tt.expectedCol, tui.cursorCol)
			}
		})
	}
}

func TestCursorMovementToEndOfLine(t *testing.T) {
	tests := []struct {
		name           string
		initialText    string
		cursorPos      int
		expectedCursor int
		expectedLine   int
		expectedCol    int
	}{
		{
			name:           "move from beginning to end",
			initialText:    "hello world",
			cursorPos:      0,
			expectedCursor: 11,
			expectedLine:   0,
			expectedCol:    11,
		},
		{
			name:           "move on first line of multi-line",
			initialText:    "line1\nline2",
			cursorPos:      2, // at 'n' in line1
			expectedCursor: 5, // before \n
			expectedLine:   0,
			expectedCol:    5,
		},
		{
			name:           "already at end",
			initialText:    "hello",
			cursorPos:      5,
			expectedCursor: 5,
			expectedLine:   0,
			expectedCol:    5,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := NewRealRenderer()
			tui := NewTUIEditor(renderer)
			tui.SetMode(ModeEdit)
			
			tui.textBuffer = []rune(tt.initialText)
			tui.cursorPos = tt.cursorPos
			tui.updateCursorPosition()
			
			tui.moveCursorToEndOfLine()
			
			if tui.cursorPos != tt.expectedCursor {
				t.Errorf("Expected cursor at %d, got %d", tt.expectedCursor, tui.cursorPos)
			}
			
			if tui.cursorLine != tt.expectedLine {
				t.Errorf("Expected line %d, got %d", tt.expectedLine, tui.cursorLine)
			}
			
			if tui.cursorCol != tt.expectedCol {
				t.Errorf("Expected col %d, got %d", tt.expectedCol, tui.cursorCol)
			}
		})
	}
}

func TestCursorMovementForwardBackward(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetMode(ModeEdit)
	
	tui.textBuffer = []rune("hello\nworld")
	tui.cursorPos = 5 // at \n
	tui.updateCursorPosition()
	
	// Move forward (should go to 'w')
	tui.moveCursorForward()
	if tui.cursorPos != 6 {
		t.Errorf("After forward, expected cursor at 6, got %d", tui.cursorPos)
	}
	if tui.cursorLine != 1 || tui.cursorCol != 0 {
		t.Errorf("After forward, expected (1,0), got (%d,%d)", tui.cursorLine, tui.cursorCol)
	}
	
	// Move backward (should go back to \n)
	tui.moveCursorBackward()
	if tui.cursorPos != 5 {
		t.Errorf("After backward, expected cursor at 5, got %d", tui.cursorPos)
	}
	if tui.cursorLine != 0 || tui.cursorCol != 5 {
		t.Errorf("After backward, expected (0,5), got (%d,%d)", tui.cursorLine, tui.cursorCol)
	}
	
	// Move backward again (should go to 'o')
	tui.moveCursorBackward()
	if tui.cursorPos != 4 {
		t.Errorf("After second backward, expected cursor at 4, got %d", tui.cursorPos)
	}
}

func TestCursorMovementWordForwardBackward(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetMode(ModeEdit)
	
	tui.textBuffer = []rune("hello world test")
	tui.cursorPos = 0
	tui.updateCursorPosition()
	
	// Move word forward (should go to 'w')
	tui.moveCursorWordForward()
	if tui.cursorPos != 6 {
		t.Errorf("After word forward, expected cursor at 6, got %d", tui.cursorPos)
	}
	
	// Move word forward again (should go to 't')
	tui.moveCursorWordForward()
	if tui.cursorPos != 12 {
		t.Errorf("After second word forward, expected cursor at 12, got %d", tui.cursorPos)
	}
	
	// Move word backward (should go to 'w')
	tui.moveCursorWordBackward()
	if tui.cursorPos != 6 {
		t.Errorf("After word backward, expected cursor at 6, got %d", tui.cursorPos)
	}
	
	// Move word backward again (should go to 'h')
	tui.moveCursorWordBackward()
	if tui.cursorPos != 0 {
		t.Errorf("After second word backward, expected cursor at 0, got %d", tui.cursorPos)
	}
}
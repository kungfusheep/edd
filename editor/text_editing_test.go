package editor

import (
	"testing"
)

func TestDeleteWordBackward(t *testing.T) {
	tests := []struct {
		name           string
		initialText    string
		cursorPos      int
		expectedText   string
		expectedCursor int
	}{
		{
			name:           "delete single word",
			initialText:    "hello world",
			cursorPos:      11, // at end
			expectedText:   "hello ",
			expectedCursor: 6,
		},
		{
			name:           "delete word with trailing space",
			initialText:    "hello world ",
			cursorPos:      12, // at end after space
			expectedText:   "hello ",
			expectedCursor: 6,
		},
		{
			name:           "delete word in middle",
			initialText:    "one two three",
			cursorPos:      7, // after "two"
			expectedText:   "one  three",
			expectedCursor: 4,
		},
		{
			name:           "delete at beginning does nothing",
			initialText:    "hello",
			cursorPos:      0,
			expectedText:   "hello",
			expectedCursor: 0,
		},
		{
			name:           "delete word after newline",
			initialText:    "line1\nword",
			cursorPos:      10, // after "word"
			expectedText:   "line1\n",
			expectedCursor: 6,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := NewRealRenderer()
			tui := NewTUIEditor(renderer)
			tui.SetMode(ModeEdit)
			
			tui.textBuffer = []rune(tt.initialText)
			tui.cursorPos = tt.cursorPos
			
			tui.deleteWordBackward()
			
			resultText := string(tui.textBuffer)
			if resultText != tt.expectedText {
				t.Errorf("Expected text '%s', got '%s'", tt.expectedText, resultText)
			}
			
			if tui.cursorPos != tt.expectedCursor {
				t.Errorf("Expected cursor at %d, got %d", tt.expectedCursor, tui.cursorPos)
			}
		})
	}
}

func TestDeleteToBeginningOfLine(t *testing.T) {
	tests := []struct {
		name           string
		initialText    string
		cursorPos      int
		expectedText   string
		expectedCursor int
	}{
		{
			name:           "delete from middle of line",
			initialText:    "hello world",
			cursorPos:      8, // at 'o' in world
			expectedText:   "rld",
			expectedCursor: 0,
		},
		{
			name:           "delete from end of line",
			initialText:    "hello world",
			cursorPos:      11,
			expectedText:   "",
			expectedCursor: 0,
		},
		{
			name:           "delete on second line",
			initialText:    "line1\nline2",
			cursorPos:      9, // at 'n' in line2
			expectedText:   "line1\ne2",
			expectedCursor: 6,
		},
		{
			name:           "delete at beginning does nothing",
			initialText:    "hello",
			cursorPos:      0,
			expectedText:   "hello",
			expectedCursor: 0,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := NewRealRenderer()
			tui := NewTUIEditor(renderer)
			tui.SetMode(ModeEdit)
			
			tui.textBuffer = []rune(tt.initialText)
			tui.cursorPos = tt.cursorPos
			
			tui.deleteToBeginningOfLine()
			
			resultText := string(tui.textBuffer)
			if resultText != tt.expectedText {
				t.Errorf("Expected text '%s', got '%s'", tt.expectedText, resultText)
			}
			
			if tui.cursorPos != tt.expectedCursor {
				t.Errorf("Expected cursor at %d, got %d", tt.expectedCursor, tui.cursorPos)
			}
		})
	}
}

func TestDeleteToEndOfLine(t *testing.T) {
	tests := []struct {
		name           string
		initialText    string
		cursorPos      int
		expectedText   string
		expectedCursor int
	}{
		{
			name:           "delete from middle of line",
			initialText:    "hello world",
			cursorPos:      6, // at 'w'
			expectedText:   "hello ",
			expectedCursor: 6,
		},
		{
			name:           "delete from beginning of line",
			initialText:    "hello world",
			cursorPos:      0,
			expectedText:   "",
			expectedCursor: 0,
		},
		{
			name:           "delete on first line of multi-line",
			initialText:    "line1\nline2",
			cursorPos:      2, // at 'n' in line1
			expectedText:   "li\nline2",
			expectedCursor: 2,
		},
		{
			name:           "delete at end does nothing",
			initialText:    "hello",
			cursorPos:      5,
			expectedText:   "hello",
			expectedCursor: 5,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := NewRealRenderer()
			tui := NewTUIEditor(renderer)
			tui.SetMode(ModeEdit)
			
			tui.textBuffer = []rune(tt.initialText)
			tui.cursorPos = tt.cursorPos
			
			tui.deleteToEndOfLine()
			
			resultText := string(tui.textBuffer)
			if resultText != tt.expectedText {
				t.Errorf("Expected text '%s', got '%s'", tt.expectedText, resultText)
			}
			
			if tui.cursorPos != tt.expectedCursor {
				t.Errorf("Expected cursor at %d, got %d", tt.expectedCursor, tui.cursorPos)
			}
		})
	}
}

func TestTextEditingWithNewlines(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetMode(ModeEdit)
	
	// Type multi-line text
	tui.textBuffer = []rune("first line\nsecond line\nthird line")
	tui.cursorPos = 18 // at 'l' in "line" of "second line"
	tui.updateCursorPosition()
	
	// Delete to beginning of line
	tui.deleteToBeginningOfLine()
	
	expected := "first line\nline\nthird line"
	if string(tui.textBuffer) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(tui.textBuffer))
	}
	
	// Cursor should be at beginning of second line
	if tui.cursorPos != 11 { // position right after first \n
		t.Errorf("Expected cursor at 11, got %d", tui.cursorPos)
	}
}
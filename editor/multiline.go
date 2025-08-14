package editor

import (
	"strings"
)

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
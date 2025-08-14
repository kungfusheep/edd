package editor

import (
	"unicode"
)

// deleteWordBackward deletes the previous word (Ctrl+W)
func (e *TUIEditor) deleteWordBackward() {
	if e.cursorPos == 0 {
		return
	}
	
	// Find the start of the previous word
	startPos := e.cursorPos - 1
	
	// Skip any trailing spaces
	for startPos >= 0 && e.textBuffer[startPos] == ' ' {
		startPos--
	}
	
	// Skip the word itself (non-space characters)
	for startPos >= 0 && e.textBuffer[startPos] != ' ' && e.textBuffer[startPos] != '\n' {
		startPos--
	}
	
	// startPos is now one position before the word start
	startPos++
	
	// Delete from startPos to cursorPos
	if startPos < e.cursorPos {
		e.textBuffer = append(e.textBuffer[:startPos], e.textBuffer[e.cursorPos:]...)
		e.cursorPos = startPos
		e.updateCursorPosition()
	}
}

// deleteToBeginningOfLine deletes from cursor to beginning of current line (Ctrl+U)
func (e *TUIEditor) deleteToBeginningOfLine() {
	if e.cursorPos == 0 {
		return
	}
	
	// Find the start of the current line
	lineStart := e.cursorPos
	for lineStart > 0 && e.textBuffer[lineStart-1] != '\n' {
		lineStart--
	}
	
	// Delete from lineStart to cursorPos
	if lineStart < e.cursorPos {
		e.textBuffer = append(e.textBuffer[:lineStart], e.textBuffer[e.cursorPos:]...)
		e.cursorPos = lineStart
		e.updateCursorPosition()
	}
}

// deleteToEndOfLine deletes from cursor to end of current line (Ctrl+K)
func (e *TUIEditor) deleteToEndOfLine() {
	if e.cursorPos >= len(e.textBuffer) {
		return
	}
	
	// Find the end of the current line
	lineEnd := e.cursorPos
	for lineEnd < len(e.textBuffer) && e.textBuffer[lineEnd] != '\n' {
		lineEnd++
	}
	
	// Delete from cursorPos to lineEnd
	if lineEnd > e.cursorPos {
		e.textBuffer = append(e.textBuffer[:e.cursorPos], e.textBuffer[lineEnd:]...)
		// cursorPos stays the same
		e.updateCursorPosition()
	}
}

// deleteWord deletes the word at cursor position (for future use)
func (e *TUIEditor) deleteWord() {
	if e.cursorPos >= len(e.textBuffer) {
		return
	}
	
	endPos := e.cursorPos
	
	// Skip any leading spaces
	for endPos < len(e.textBuffer) && e.textBuffer[endPos] == ' ' {
		endPos++
	}
	
	// Skip the word itself
	for endPos < len(e.textBuffer) && e.textBuffer[endPos] != ' ' && e.textBuffer[endPos] != '\n' {
		endPos++
	}
	
	// Delete from cursorPos to endPos
	if endPos > e.cursorPos {
		e.textBuffer = append(e.textBuffer[:e.cursorPos], e.textBuffer[endPos:]...)
		// cursorPos stays the same
		e.updateCursorPosition()
	}
}

// Helper to check if a rune is a word boundary
func isWordBoundary(r rune) bool {
	return unicode.IsSpace(r) || unicode.IsPunct(r)
}
package editor

// moveCursorToBeginningOfLine moves cursor to the beginning of the current line (Ctrl+A)
func (e *TUIEditor) moveCursorToBeginningOfLine() {
	if e.cursorPos == 0 {
		return
	}
	
	// Find the start of the current line
	newPos := e.cursorPos
	for newPos > 0 && e.textBuffer[newPos-1] != '\n' {
		newPos--
	}
	
	e.cursorPos = newPos
	e.updateCursorPosition()
}

// moveCursorToEndOfLine moves cursor to the end of the current line (Ctrl+E)
func (e *TUIEditor) moveCursorToEndOfLine() {
	if e.cursorPos >= len(e.textBuffer) {
		return
	}
	
	// Find the end of the current line
	newPos := e.cursorPos
	for newPos < len(e.textBuffer) && e.textBuffer[newPos] != '\n' {
		newPos++
	}
	
	e.cursorPos = newPos
	e.updateCursorPosition()
}

// moveCursorForward moves cursor forward one character (Ctrl+F)
func (e *TUIEditor) moveCursorForward() {
	if e.cursorPos < len(e.textBuffer) {
		e.cursorPos++
		e.updateCursorPosition()
	}
}

// moveCursorBackward moves cursor backward one character (Ctrl+B)
func (e *TUIEditor) moveCursorBackward() {
	if e.cursorPos > 0 {
		e.cursorPos--
		e.updateCursorPosition()
	}
}

// Additional useful cursor movements

// moveCursorWordForward moves cursor to the beginning of the next word (Alt+F in terminals)
func (e *TUIEditor) moveCursorWordForward() {
	if e.cursorPos >= len(e.textBuffer) {
		return
	}
	
	// Skip current word
	for e.cursorPos < len(e.textBuffer) && e.textBuffer[e.cursorPos] != ' ' && e.textBuffer[e.cursorPos] != '\n' {
		e.cursorPos++
	}
	
	// Skip spaces
	for e.cursorPos < len(e.textBuffer) && e.textBuffer[e.cursorPos] == ' ' {
		e.cursorPos++
	}
	
	e.updateCursorPosition()
}

// moveCursorWordBackward moves cursor to the beginning of the previous word (Alt+B in terminals)
func (e *TUIEditor) moveCursorWordBackward() {
	if e.cursorPos == 0 {
		return
	}
	
	// Move back one position
	e.cursorPos--
	
	// Skip spaces
	for e.cursorPos > 0 && e.textBuffer[e.cursorPos] == ' ' {
		e.cursorPos--
	}
	
	// Find beginning of word
	for e.cursorPos > 0 && e.textBuffer[e.cursorPos-1] != ' ' && e.textBuffer[e.cursorPos-1] != '\n' {
		e.cursorPos--
	}
	
	e.updateCursorPosition()
}

// moveCursorUp moves cursor up one line (Arrow Up)
func (e *TUIEditor) moveCursorUp() {
	e.moveUp()
}

// moveCursorDown moves cursor down one line (Arrow Down)
func (e *TUIEditor) moveCursorDown() {
	e.moveDown()
}
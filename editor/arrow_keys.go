package editor

// HandleArrowKey handles arrow keys and other special navigation keys
func (e *TUIEditor) HandleArrowKey(direction rune) {
	// Only handle in text editing modes
	if e.mode != ModeEdit && e.mode != ModeInsert {
		return
	}
	
	switch direction {
	case 'U': // Arrow Up
		e.moveCursorUp()
	case 'D': // Arrow Down
		e.moveCursorDown()
	case 'L': // Arrow Left
		e.moveCursorBackward()
	case 'R': // Arrow Right
		e.moveCursorForward()
	case 'H': // Home key
		e.moveCursorToBeginningOfLine()
	case 'E': // End key
		e.moveCursorToEndOfLine()
	}
}
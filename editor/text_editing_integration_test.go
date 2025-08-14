package editor

import (
	"testing"
)

func TestTextEditingIntegration(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	
	// Start editing a new node
	tui.SetMode(ModeInsert)
	nodeID := tui.AddNode([]string{""})
	tui.selected = nodeID
	
	// Type some text
	for _, ch := range "Hello world" {
		tui.handleTextKey(ch)
	}
	
	// Move to beginning of line (Ctrl+A)
	tui.handleTextKey(1)
	if tui.cursorPos != 0 {
		t.Errorf("Ctrl+A: Expected cursor at 0, got %d", tui.cursorPos)
	}
	
	// Move to end of line (Ctrl+E)
	tui.handleTextKey(5)
	if tui.cursorPos != 11 {
		t.Errorf("Ctrl+E: Expected cursor at 11, got %d", tui.cursorPos)
	}
	
	// Insert a newline at the end (Ctrl+N)
	tui.handleTextKey(14)
	
	// Type more text on second line
	for _, ch := range "Second line" {
		tui.handleTextKey(ch)
	}
	
	expectedAfterSecondLine := "Hello world\nSecond line"
	if string(tui.textBuffer) != expectedAfterSecondLine {
		t.Errorf("After adding second line: Expected '%s', got '%s'", expectedAfterSecondLine, string(tui.textBuffer))
	}
	
	// Add another newline to start third line
	tui.handleTextKey(14) // Ctrl+N
	
	// Type third line
	for _, ch := range "Third line" {
		tui.handleTextKey(ch)
	}
	
	// Now we have "Hello world\nSecond line\nThird line"
	
	// Move to beginning of third line (Ctrl+A)
	tui.handleTextKey(1)
	if tui.cursorLine != 2 || tui.cursorCol != 0 {
		t.Errorf("Ctrl+A on line 3: Expected (2,0), got (%d,%d)", tui.cursorLine, tui.cursorCol)
	}
	
	// Delete to end of line (Ctrl+K) - removes "Third line"
	tui.handleTextKey(11)
	
	// Type replacement text
	for _, ch := range "Final line" {
		tui.handleTextKey(ch)
	}
	
	expectedFinal := "Hello world\nSecond line\nFinal line"
	if string(tui.textBuffer) != expectedFinal {
		t.Errorf("Final text: Expected '%s', got '%s'", expectedFinal, string(tui.textBuffer))
	}
	
	// Move back a few chars then delete word
	tui.handleTextKey(5) // Ctrl+E to go to end of line
	for i := 0; i < 4; i++ {
		tui.handleTextKey(2) // Ctrl+B to move back
	}
	// Cursor should be at 'l' in "Final line"
	tui.handleTextKey(23) // Ctrl+W to delete "Final "
	
	expectedAfterDelete := "Hello world\nSecond line\nline"
	if string(tui.textBuffer) != expectedAfterDelete {
		t.Errorf("After Ctrl+W: Expected '%s', got '%s'", expectedAfterDelete, string(tui.textBuffer))
	}
}

func TestAllEditingShortcuts(t *testing.T) {
	shortcuts := []struct {
		key  rune
		desc string
	}{
		{1, "Ctrl+A (beginning of line)"},
		{2, "Ctrl+B (backward)"},
		{5, "Ctrl+E (end of line)"},
		{6, "Ctrl+F (forward)"},
		{11, "Ctrl+K (delete to end)"},
		{14, "Ctrl+N (newline)"},
		{21, "Ctrl+U (delete to beginning)"},
		{23, "Ctrl+W (delete word)"},
	}
	
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	tui.SetMode(ModeEdit)
	
	// Test that all shortcuts are handled without panic
	for _, shortcut := range shortcuts {
		tui.textBuffer = []rune("test text")
		tui.cursorPos = 5
		tui.updateCursorPosition()
		
		// Should not panic
		tui.handleTextKey(shortcut.key)
		
		t.Logf("✓ %s handled without error", shortcut.desc)
	}
}
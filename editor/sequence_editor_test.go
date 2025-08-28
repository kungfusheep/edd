package editor

import (
	"strings"
	"testing"
)

func TestSequenceDiagramInEditor(t *testing.T) {
	tui := NewTUIEditor(NewRealRenderer())
	
	// Set diagram type to sequence
	tui.HandleKey(':')
	for _, ch := range "type sequence" {
		tui.HandleKey(ch)
	}
	tui.HandleKey(13) // Enter
	
	if tui.diagram.Type != "sequence" {
		t.Errorf("Expected diagram type to be sequence, got %s", tui.diagram.Type)
	}
	
	// Add some participants
	tui.HandleKey('a') // Add node
	tui.textBuffer = []rune("Client")
	tui.commitText()
	tui.HandleKey(27) // ESC
	
	tui.HandleKey('a') // Add another node
	tui.textBuffer = []rune("Server")
	tui.commitText()
	tui.HandleKey(27) // ESC
	
	// Render and check output
	output := tui.Render()
	
	// Should contain participants
	if !strings.Contains(output, "Client") {
		t.Error("Output should contain Client participant")
	}
	if !strings.Contains(output, "Server") {
		t.Error("Output should contain Server participant")
	}
	
	// Should contain lifelines
	if !strings.Contains(output, "│") {
		t.Error("Output should contain lifelines")
	}
	
	// Add a connection
	if len(tui.diagram.Nodes) >= 2 {
		tui.HandleKey('c') // Connect mode
		tui.HandleKey('a') // Select first node
		tui.HandleKey('s') // Select second node
		
		// Check connection was added
		if len(tui.diagram.Connections) != 1 {
			t.Errorf("Expected 1 connection, got %d", len(tui.diagram.Connections))
		}
		
		// Render again
		output = tui.Render()
		
		// Should contain arrow
		if !strings.Contains(output, "─") || !strings.Contains(output, "▶") {
			t.Error("Output should contain message arrow")
		}
	}
}

func TestSequenceDiagramCommands(t *testing.T) {
	tui := NewTUIEditor(NewRealRenderer())
	
	// Test type command variations
	tests := []struct {
		command      string
		expectedType string
	}{
		{"type sequence", "sequence"},
		{"type seq", "sequence"},
		{"type flowchart", ""},
		{"type flow", ""},
	}
	
	for _, tt := range tests {
		tui.HandleKey(':')
		for _, ch := range tt.command {
			tui.HandleKey(ch)
		}
		tui.HandleKey(13) // Enter
		
		if tui.diagram.Type != tt.expectedType {
			t.Errorf("Command '%s': expected type '%s', got '%s'",
				tt.command, tt.expectedType, tui.diagram.Type)
		}
	}
}
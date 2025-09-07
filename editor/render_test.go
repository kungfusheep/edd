package editor

import (
	"edd/diagram"
	"strings"
	"testing"
)

// TestRenderEmptyState tests rendering with no diagram
func TestRenderEmptyState(t *testing.T) {
	state := TUIState{
		Diagram: &diagram.Diagram{},
		Mode:    ModeNormal,
		Width:   80,
		Height:  24,
	}
	
	output := RenderTUI(state)
	
	// Should contain help text
	if !strings.Contains(output, "Press 'a' to add a node") {
		t.Error("Empty state should show help text")
	}
	
	// Mode indicator is now rendered separately in the TUI, not in RenderTUI
	// So we don't test for it here
}

// TestRenderWithNodes tests rendering with nodes
func TestRenderWithNodes(t *testing.T) {
	state := TUIState{
		Diagram: &diagram.Diagram{
			Nodes: []diagram.Node{
				{ID: 1, Text: []string{"Server"}},
				{ID: 2, Text: []string{"Database"}},
			},
		},
		Mode:     ModeNormal,
		EddFrame: "◉‿◉",
	}
	
	output := RenderTUI(state)
	
	// Should show both nodes
	if !strings.Contains(output, "Server") {
		t.Error("Should display Server node")
	}
	if !strings.Contains(output, "Database") {
		t.Error("Should display Database node")
	}
	
	// Ed mascot is now rendered separately via ANSI codes, not in RenderTUI
}

// TestRenderJumpMode tests jump label rendering
func TestRenderJumpMode(t *testing.T) {
	state := TUIState{
		Diagram: &diagram.Diagram{
			Nodes: []diagram.Node{
				{ID: 1, Text: []string{"Node1"}},
				{ID: 2, Text: []string{"Node2"}},
				{ID: 3, Text: []string{"Node3"}},
			},
		},
		Mode: ModeJump,
		JumpLabels: map[int]rune{
			1: 'a',
			2: 's',
			3: 'd',
		},
		EddFrame: "◎‿◎",
	}
	
	// Jump labels and mode indicator are now rendered separately via ANSI codes, not in RenderTUI
	// The actual rendering test would need to verify the state's JumpLabels map
	_ = RenderTUI(state) // Just verify it doesn't panic
}

// TestRenderTextInput tests text input state
func TestRenderTextInput(t *testing.T) {
	// Text input is now handled by showing cursor in the node itself
	// Not as an overlay, so just test that state is preserved
	tests := []struct {
		name       string
		textBuffer []rune
		cursorPos  int
	}{
		{
			name:       "Empty with cursor",
			textBuffer: []rune{},
			cursorPos:  0,
		},
		{
			name:       "Text with cursor at end",
			textBuffer: []rune("Hello"),
			cursorPos:  5,
		},
		{
			name:       "Text with cursor in middle",
			textBuffer: []rune("Hello"),
			cursorPos:  2,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := TUIState{
				Diagram:    &diagram.Diagram{},
				Mode:       ModeEdit,
				TextBuffer: tt.textBuffer,
				CursorPos:  tt.cursorPos,
			}
			
			// Just verify rendering doesn't panic and state is preserved
			_ = RenderTUI(state)
			
			if state.CursorPos != tt.cursorPos {
				t.Errorf("Cursor position changed: expected %d, got %d", tt.cursorPos, state.CursorPos)
			}
			if string(state.TextBuffer) != string(tt.textBuffer) {
				t.Errorf("Text buffer changed: expected %s, got %s", string(tt.textBuffer), string(state.TextBuffer))
			}
		})
	}
}

// TestRenderConnections tests connection rendering
func TestRenderConnections(t *testing.T) {
	state := TUIState{
		Diagram: &diagram.Diagram{
			Nodes: []diagram.Node{
				{ID: 1, Text: []string{"A"}},
				{ID: 2, Text: []string{"B"}},
			},
			Connections: []diagram.Connection{
				{From: 1, To: 2, Label: "test"},
			},
		},
		Mode: ModeNormal,
	}
	
	output := RenderTUI(state)
	
	// Should show connection
	if !strings.Contains(output, "1 -> 2") {
		t.Error("Should show connection")
	}
	if !strings.Contains(output, "test") {
		t.Error("Should show connection label")
	}
}

// TestModeTransitions tests different mode indicators
func TestModeTransitions(t *testing.T) {
	modes := []struct {
		mode Mode
		want string
		face string
	}{
		{ModeNormal, "NORMAL", "◉‿◉"},
		{ModeInsert, "INSERT", "○‿○"},
		{ModeEdit, "EDIT", "◉‿◉"},
		{ModeCommand, "COMMAND", ":_"},
		{ModeJump, "JUMP", "◎‿◎"},
	}
	
	for _, m := range modes {
		state := TUIState{
			Diagram:  &diagram.Diagram{},
			Mode:     m.mode,
			EddFrame: m.face,
		}
		
		// Mode indicators and Ed face are now rendered separately via ANSI codes
		// They are not part of the RenderTUI output
		_ = RenderTUI(state) // Just verify it doesn't panic
	}
}

// TestComplexScenario tests a complex editing scenario
func TestComplexScenario(t *testing.T) {
	// Simulate: User is connecting nodes with jump labels active
	state := TUIState{
		Diagram: &diagram.Diagram{
			Nodes: []diagram.Node{
				{ID: 1, Text: []string{"Web", "Server"}},
				{ID: 2, Text: []string{"API"}},
				{ID: 3, Text: []string{"Database"}},
			},
			Connections: []diagram.Connection{
				{From: 1, To: 2},
			},
		},
		Mode:     ModeJump,
		Selected: 2, // API node selected as source
		JumpLabels: map[int]rune{
			1: 'a',
			3: 'd', // Can't connect to self, so no label for node 2
		},
		EddFrame: "◎‿◎",
	}
	
	output := RenderTUI(state)
	
	// Verify the node text is rendered
	if !strings.Contains(output, "Web Server") {
		t.Error("Should show multi-line node text")
	}
	// Jump labels and mode indicator are now rendered separately via ANSI codes
}

// Benchmark to establish baseline performance
func BenchmarkRenderSimple(b *testing.B) {
	state := TUIState{
		Diagram: &diagram.Diagram{
			Nodes: []diagram.Node{
				{ID: 1, Text: []string{"Node1"}},
				{ID: 2, Text: []string{"Node2"}},
				{ID: 3, Text: []string{"Node3"}},
			},
			Connections: []diagram.Connection{
				{From: 1, To: 2},
				{From: 2, To: 3},
			},
		},
		Mode:     ModeNormal,
		EddFrame: "◉‿◉",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RenderTUI(state)
	}
}

func BenchmarkRenderComplex(b *testing.B) {
	// Create a larger diagram
	nodes := make([]diagram.Node, 20)
	for i := range nodes {
		nodes[i] = diagram.Node{
			ID:   i + 1,
			Text: []string{"Node"},
		}
	}
	
	connections := make([]diagram.Connection, 30)
	for i := range connections {
		connections[i] = diagram.Connection{
			From: (i % 20) + 1,
			To:   ((i + 5) % 20) + 1,
		}
	}
	
	state := TUIState{
		Diagram: &diagram.Diagram{
			Nodes:       nodes,
			Connections: connections,
		},
		Mode: ModeNormal,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RenderTUI(state)
	}
}
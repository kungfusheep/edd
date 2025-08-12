package editor

import (
	"edd/core"
	"strings"
	"testing"
)

// TestRenderEmptyState tests rendering with no diagram
func TestRenderEmptyState(t *testing.T) {
	state := TUIState{
		Diagram: &core.Diagram{},
		Mode:    ModeNormal,
		Width:   80,
		Height:  24,
	}
	
	output := RenderTUI(state)
	
	// Should contain help text
	if !strings.Contains(output, "Press 'a' to add a node") {
		t.Error("Empty state should show help text")
	}
	
	// Should show mode indicator
	if !strings.Contains(output, "NORMAL") {
		t.Error("Should show NORMAL mode")
	}
}

// TestRenderWithNodes tests rendering with nodes
func TestRenderWithNodes(t *testing.T) {
	state := TUIState{
		Diagram: &core.Diagram{
			Nodes: []core.Node{
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
	
	// Should show Ed
	if !strings.Contains(output, "◉‿◉") {
		t.Error("Should display Ed mascot")
	}
}

// TestRenderJumpMode tests jump label rendering
func TestRenderJumpMode(t *testing.T) {
	state := TUIState{
		Diagram: &core.Diagram{
			Nodes: []core.Node{
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
	
	output := RenderTUI(state)
	
	// Should show jump labels
	if !strings.Contains(output, "[a]") {
		t.Error("Should show jump label [a]")
	}
	if !strings.Contains(output, "[s]") {
		t.Error("Should show jump label [s]")
	}
	if !strings.Contains(output, "[d]") {
		t.Error("Should show jump label [d]")
	}
	
	// Should show JUMP mode
	if !strings.Contains(output, "JUMP") {
		t.Error("Should show JUMP mode")
	}
}

// TestRenderTextInput tests text input rendering
func TestRenderTextInput(t *testing.T) {
	tests := []struct {
		name       string
		textBuffer []rune
		cursorPos  int
		want       string
	}{
		{
			name:       "Empty with cursor",
			textBuffer: []rune{},
			cursorPos:  0,
			want:       "Text: │",
		},
		{
			name:       "Text with cursor at end",
			textBuffer: []rune("Hello"),
			cursorPos:  5,
			want:       "Text: Hello│",
		},
		{
			name:       "Text with cursor in middle",
			textBuffer: []rune("Hello"),
			cursorPos:  2,
			want:       "Text: He│llo",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := TUIState{
				Diagram:    &core.Diagram{},
				Mode:       ModeEdit,
				TextBuffer: tt.textBuffer,
				CursorPos:  tt.cursorPos,
			}
			
			output := RenderTUI(state)
			
			if !strings.Contains(output, tt.want) {
				t.Errorf("Expected text input to show %q, got output:\n%s", tt.want, output)
			}
		})
	}
}

// TestRenderConnections tests connection rendering
func TestRenderConnections(t *testing.T) {
	state := TUIState{
		Diagram: &core.Diagram{
			Nodes: []core.Node{
				{ID: 1, Text: []string{"A"}},
				{ID: 2, Text: []string{"B"}},
			},
			Connections: []core.Connection{
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
			Diagram:  &core.Diagram{},
			Mode:     m.mode,
			EddFrame: m.face,
		}
		
		output := RenderTUI(state)
		
		if !strings.Contains(output, m.want) {
			t.Errorf("Mode %v should display %q", m.mode, m.want)
		}
		
		if !strings.Contains(output, m.face) {
			t.Errorf("Mode %v should show Ed face %q", m.mode, m.face)
		}
	}
}

// TestComplexScenario tests a complex editing scenario
func TestComplexScenario(t *testing.T) {
	// Simulate: User is connecting nodes with jump labels active
	state := TUIState{
		Diagram: &core.Diagram{
			Nodes: []core.Node{
				{ID: 1, Text: []string{"Web", "Server"}},
				{ID: 2, Text: []string{"API"}},
				{ID: 3, Text: []string{"Database"}},
			},
			Connections: []core.Connection{
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
	
	// Verify the complete state is rendered
	if !strings.Contains(output, "Web Server") {
		t.Error("Should show multi-line node text")
	}
	if !strings.Contains(output, "[a]") {
		t.Error("Should show jump label for node 1")
	}
	if !strings.Contains(output, "[d]") {
		t.Error("Should show jump label for node 3")
	}
	if !strings.Contains(output, "JUMP") {
		t.Error("Should indicate JUMP mode")
	}
}

// Benchmark to establish baseline performance
func BenchmarkRenderSimple(b *testing.B) {
	state := TUIState{
		Diagram: &core.Diagram{
			Nodes: []core.Node{
				{ID: 1, Text: []string{"Node1"}},
				{ID: 2, Text: []string{"Node2"}},
				{ID: 3, Text: []string{"Node3"}},
			},
			Connections: []core.Connection{
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
	nodes := make([]core.Node, 20)
	for i := range nodes {
		nodes[i] = core.Node{
			ID:   i + 1,
			Text: []string{"Node"},
		}
	}
	
	connections := make([]core.Connection, 30)
	for i := range connections {
		connections[i] = core.Connection{
			From: (i % 20) + 1,
			To:   ((i + 5) % 20) + 1,
		}
	}
	
	state := TUIState{
		Diagram: &core.Diagram{
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
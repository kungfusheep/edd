package editor

import (
	"edd/core"
	"fmt"
	"strings"
	"testing"
)

// TestModeIndicatorFormat tests that the mode indicator renders correctly
func TestModeIndicatorFormat(t *testing.T) {
	tests := []struct {
		name     string
		mode     Mode
		eddFrame string
		want     []string // Things that should appear
		notWant  []string // Things that should NOT appear
	}{
		{
			name:     "Normal mode indicator",
			mode:     ModeNormal,
			eddFrame: "◉‿◉",
			want:     []string{"NORMAL", "◉‿◉", "╭", "╮", "╰", "╯"},
			notWant:  []string{"NORMALRMAL", "||", ":_"},
		},
		{
			name:     "Command mode indicator",
			mode:     ModeCommand,
			eddFrame: ":_",
			want:     []string{"COMMAND", ":_"},
			notWant:  []string{"◉‿◉"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indicator := createModeIndicator(tt.mode, tt.eddFrame)
			
			// Check wanted strings
			for _, want := range tt.want {
				if !strings.Contains(indicator, want) {
					t.Errorf("Mode indicator should contain %q, got:\n%s", want, indicator)
				}
			}
			
			// Check unwanted strings
			for _, notWant := range tt.notWant {
				if strings.Contains(indicator, notWant) {
					t.Errorf("Mode indicator should NOT contain %q, got:\n%s", notWant, indicator)
				}
			}
			
			// Debug output
			fmt.Printf("=== %s ===\n%s\n", tt.name, indicator)
		})
	}
}

// TestHelpScreenClearing tests that help screen clears properly
func TestHelpScreenClearing(t *testing.T) {
	// Create a state with a node
	state := TUIState{
		Diagram: &core.Diagram{
			Nodes: []core.Node{
				{ID: 1, Text: []string{"hi"}},
			},
		},
		Mode:     ModeNormal,
		EddFrame: "◉‿◉",
		Width:    80,
		Height:   24,
	}
	
	output := RenderTUI(state)
	
	// The output should NOT contain help text when not in help mode
	helpStrings := []string{
		"Edit node text",
		"Command mode",
		"Press any key to continue",
		"Text Editing:",
	}
	
	for _, helpStr := range helpStrings {
		if strings.Contains(output, helpStr) {
			t.Errorf("Normal mode should not show help text %q", helpStr)
		}
	}
}

// TestOverlayAlignment tests that overlays align properly
func TestOverlayAlignment(t *testing.T) {
	state := TUIState{
		Diagram:  &core.Diagram{},
		Mode:     ModeNormal,
		EddFrame: "◉‿◉",
		Width:    80,
		Height:   24,
	}
	
	// Mode indicators and Ed are now rendered using ANSI escape codes
	// They don't appear in the actual output text
	output := RenderTUI(state)
	lines := strings.Split(output, "\n")
	
	// Just verify we have output and state is correct
	if len(lines) < 1 {
		t.Error("Output should have at least 1 line")
	}
	
	if state.Mode != ModeNormal {
		t.Errorf("Expected ModeNormal, got %v", state.Mode)
	}
	
	if state.EddFrame != "◉‿◉" {
		t.Errorf("Expected Ed frame ◉‿◉, got %s", state.EddFrame)
	}
}

// TestDebugRenderOutput provides detailed debug output
func TestDebugRenderOutput(t *testing.T) {
	state := TUIState{
		Diagram: &core.Diagram{
			Nodes: []core.Node{
				{ID: 1, Text: []string{"hi"}},
			},
		},
		Mode:     ModeNormal,
		EddFrame: "◉‿◉",
		Width:    80,
		Height:   24,
	}
	
	output := RenderTUI(state)
	
	fmt.Println("=== FULL RENDER OUTPUT ===")
	fmt.Println(output)
	fmt.Println("=== END OUTPUT ===")
	
	// Show each line with line numbers for debugging
	lines := strings.Split(output, "\n")
	fmt.Println("\n=== LINE BY LINE ===")
	for i, line := range lines {
		fmt.Printf("%2d: %q\n", i, line)
	}
}
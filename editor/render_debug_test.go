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
	
	output := RenderTUI(state)
	lines := strings.Split(output, "\n")
	
	// Check that we have enough lines
	if len(lines) < 5 {
		t.Errorf("Output should have at least 5 lines, got %d", len(lines))
	}
	
	// Check that mode indicator is properly positioned
	// It should be in the bottom-right corner (changed from top-right)
	foundIndicator := false
	for i, line := range lines {
		// Look for the box structure (now has color codes and different format)
		if strings.Contains(line, "╭────╮") || strings.Contains(line, "╭") && strings.Contains(line, "╮") {
			foundIndicator = true
			// Check next lines for proper box structure
			if i+1 < len(lines) && !strings.Contains(lines[i+1], "│") {
				t.Error("Mode indicator box not properly formed")
			}
			break
		}
	}
	
	if !foundIndicator {
		t.Error("Mode indicator box not found in output")
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
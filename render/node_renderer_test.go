package render

import (
	"edd/diagram"
	"strings"
	"testing"
)

func TestNodeRendererBasicStyles(t *testing.T) {
	tests := []struct {
		name      string
		style     string
		wantChars struct {
			topLeft     rune
			topRight    rune
			bottomLeft  rune
			bottomRight rune
		}
	}{
		{
			name:  "rounded style",
			style: "rounded",
			wantChars: struct {
				topLeft     rune
				topRight    rune
				bottomLeft  rune
				bottomRight rune
			}{'╭', '╮', '╰', '╯'},
		},
		{
			name:  "sharp style",
			style: "sharp",
			wantChars: struct {
				topLeft     rune
				topRight    rune
				bottomLeft  rune
				bottomRight rune
			}{'┌', '┐', '└', '┘'},
		},
		{
			name:  "double style",
			style: "double",
			wantChars: struct {
				topLeft     rune
				topRight    rune
				bottomLeft  rune
				bottomRight rune
			}{'╔', '╗', '╚', '╝'},
		},
		{
			name:  "thick style",
			style: "thick",
			wantChars: struct {
				topLeft     rune
				topRight    rune
				bottomLeft  rune
				bottomRight rune
			}{'┏', '┓', '┗', '┛'},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create canvas and renderer
			canvas := NewMatrixCanvas(20, 10)
			renderer := NewNodeRenderer(TerminalCapabilities{
				UnicodeLevel: UnicodeFull,
			})
			
			// Create node with style hint
			node := diagram.Node{
				ID:     1,
				Text:   []string{"Test"},
				X:      2,
				Y:      2,
				Width:  10,
				Height: 4,
				Hints: map[string]string{
					"style": tt.style,
				},
			}
			
			// Render the node
			err := renderer.RenderNode(canvas, node)
			if err != nil {
				t.Fatalf("Failed to render node: %v", err)
			}
			
			// Check corners
			if canvas.Get(diagram.Point{X: 2, Y: 2}) != tt.wantChars.topLeft {
				t.Errorf("Top-left corner: got %c, want %c", 
					canvas.Get(diagram.Point{X: 2, Y: 2}), tt.wantChars.topLeft)
			}
			if canvas.Get(diagram.Point{X: 11, Y: 2}) != tt.wantChars.topRight {
				t.Errorf("Top-right corner: got %c, want %c", 
					canvas.Get(diagram.Point{X: 11, Y: 2}), tt.wantChars.topRight)
			}
			if canvas.Get(diagram.Point{X: 2, Y: 5}) != tt.wantChars.bottomLeft {
				t.Errorf("Bottom-left corner: got %c, want %c", 
					canvas.Get(diagram.Point{X: 2, Y: 5}), tt.wantChars.bottomLeft)
			}
			if canvas.Get(diagram.Point{X: 11, Y: 5}) != tt.wantChars.bottomRight {
				t.Errorf("Bottom-right corner: got %c, want %c", 
					canvas.Get(diagram.Point{X: 11, Y: 5}), tt.wantChars.bottomRight)
			}
		})
	}
}

func TestNodeRendererFallback(t *testing.T) {
	// Test that invalid style falls back to default
	canvas := NewMatrixCanvas(20, 10)
	renderer := NewNodeRenderer(TerminalCapabilities{
		UnicodeLevel: UnicodeFull,
	})
	
	node := diagram.Node{
		ID:     1,
		Text:   []string{"Test"},
		X:      2,
		Y:      2,
		Width:  10,
		Height: 4,
		Hints: map[string]string{
			"style": "invalid-style",
		},
	}
	
	err := renderer.RenderNode(canvas, node)
	if err != nil {
		t.Fatalf("Failed to render node: %v", err)
	}
	
	// Should fall back to rounded (default)
	if canvas.Get(diagram.Point{X: 2, Y: 2}) != '╭' {
		t.Errorf("Should fall back to rounded style, got %c", canvas.Get(diagram.Point{X: 2, Y: 2}))
	}
}

func TestNodeRendererASCIIFallback(t *testing.T) {
	// Test that ASCII terminals get ASCII style
	canvas := NewMatrixCanvas(20, 10)
	renderer := NewNodeRenderer(TerminalCapabilities{
		UnicodeLevel: UnicodeNone,
	})
	
	node := diagram.Node{
		ID:     1,
		Text:   []string{"Test"},
		X:      2,
		Y:      2,
		Width:  10,
		Height: 4,
		Hints: map[string]string{
			"style": "rounded", // Should be ignored for ASCII
		},
	}
	
	err := renderer.RenderNode(canvas, node)
	if err != nil {
		t.Fatalf("Failed to render node: %v", err)
	}
	
	// Should use ASCII style
	if canvas.Get(diagram.Point{X: 2, Y: 2}) != '+' {
		t.Errorf("Should use ASCII style for ASCII terminal, got %c", canvas.Get(diagram.Point{X: 2, Y: 2}))
	}
}

func TestNodeRendererText(t *testing.T) {
	// Test that text is rendered correctly inside the box
	canvas := NewMatrixCanvas(20, 10)
	renderer := NewNodeRenderer(TerminalCapabilities{
		UnicodeLevel: UnicodeFull,
	})
	
	node := diagram.Node{
		ID:     1,
		Text:   []string{"Line 1", "Line 2"},
		X:      0,
		Y:      0,
		Width:  12,
		Height: 5,
	}
	
	err := renderer.RenderNode(canvas, node)
	if err != nil {
		t.Fatalf("Failed to render node: %v", err)
	}
	
	output := canvas.String()
	lines := strings.Split(output, "\n")
	
	// Check that text appears in the right place (line 1 at y=1, line 2 at y=2)
	// Text should be at x=2 (2 chars padding)
	if !strings.Contains(lines[1], "Line 1") {
		t.Errorf("Line 1 not found in correct position")
	}
	if !strings.Contains(lines[2], "Line 2") {
		t.Errorf("Line 2 not found in correct position")
	}
}

func TestNodeRendererColors(t *testing.T) {
	// Test that colors are applied when using ColoredMatrixCanvas
	canvas := NewColoredMatrixCanvas(20, 10)
	renderer := NewNodeRenderer(TerminalCapabilities{
		UnicodeLevel:  UnicodeFull,
		SupportsColor: true,
	})
	
	node := diagram.Node{
		ID:     1,
		Text:   []string{"Colored"},
		X:      2,
		Y:      2,
		Width:  10,
		Height: 4,
		Hints: map[string]string{
			"style": "double",
			"color": "blue",
		},
	}
	
	err := renderer.RenderNode(canvas, node)
	if err != nil {
		t.Fatalf("Failed to render node: %v", err)
	}
	
	// Check that the box uses double-line style
	if canvas.Get(diagram.Point{X: 2, Y: 2}) != '╔' {
		t.Errorf("Expected double-line top-left corner, got %c", canvas.Get(diagram.Point{X: 2, Y: 2}))
	}
	
	// The colored output should contain ANSI color codes
	coloredOutput := canvas.ColoredString()
	if !strings.Contains(coloredOutput, "\033[") {
		t.Errorf("Expected colored output to contain ANSI codes")
	}
}

func TestNodeRendererVisualRegression(t *testing.T) {
	// Visual regression test - ensure the output looks correct
	canvas := NewMatrixCanvas(15, 6)
	renderer := NewNodeRenderer(TerminalCapabilities{
		UnicodeLevel: UnicodeFull,
	})
	
	node := diagram.Node{
		ID:     1,
		Text:   []string{"Hello", "World"},
		X:      1,
		Y:      1,
		Width:  10,
		Height: 4,
		Hints: map[string]string{
			"style": "rounded",
		},
	}
	
	err := renderer.RenderNode(canvas, node)
	if err != nil {
		t.Fatalf("Failed to render node: %v", err)
	}
	
	expected := `               
 ╭────────╮    
 │ Hello  │    
 │ World  │    
 ╰────────╯    
               `
	
	actual := canvas.String()
	if actual != expected {
		t.Errorf("Visual regression failed.\nExpected:\n%s\nGot:\n%s", expected, actual)
		// Print with visible spaces for debugging
		t.Errorf("Expected (with dots for spaces):\n%s", strings.ReplaceAll(expected, " ", "·"))
		t.Errorf("Got (with dots for spaces):\n%s", strings.ReplaceAll(actual, " ", "·"))
	}
}
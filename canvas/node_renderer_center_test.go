package canvas

import (
	"edd/core"
	"strings"
	"testing"
)

func TestNodeRendererCenterText(t *testing.T) {
	tests := []struct {
		name     string
		node     core.Node
		hints    map[string]string
		expected []string
	}{
		{
			name: "center single line text",
			node: core.Node{
				X:      0,
				Y:      0,
				Width:  12,
				Height: 3,
				Text:   []string{"Hello"},
			},
			hints: map[string]string{
				"text-align": "center",
			},
			expected: []string{
				"╭──────────╮",
				"│  Hello   │",
				"╰──────────╯",
			},
		},
		{
			name: "center multiple lines",
			node: core.Node{
				X:      0,
				Y:      0,
				Width:  14,
				Height: 4,
				Text:   []string{"Hello", "World"},
			},
			hints: map[string]string{
				"text-align": "center",
			},
			expected: []string{
				"╭────────────╮",
				"│   Hello    │",
				"│   World    │",
				"╰────────────╯",
			},
		},
		{
			name: "center with different line lengths",
			node: core.Node{
				X:      0,
				Y:      0,
				Width:  16,
				Height: 5,
				Text:   []string{"Short", "Much Longer", "Mid"},
			},
			hints: map[string]string{
				"text-align": "center",
			},
			expected: []string{
				"╭──────────────╮",
				"│    Short     │",
				"│ Much Longer  │",
				"│     Mid      │",
				"╰──────────────╯",
			},
		},
		{
			name: "left-aligned by default",
			node: core.Node{
				X:      0,
				Y:      0,
				Width:  12,
				Height: 3,
				Text:   []string{"Hello"},
			},
			hints: map[string]string{}, // No text-align hint
			expected: []string{
				"╭──────────╮",
				"│ Hello    │",
				"╰──────────╯",
			},
		},
		{
			name: "center with bold",
			node: core.Node{
				X:      0,
				Y:      0,
				Width:  10,
				Height: 3,
				Text:   []string{"Test"},
			},
			hints: map[string]string{
				"text-align": "center",
				"bold":       "true",
			},
			expected: []string{
				"╭────────╮",
				"│  Test  │", // Should still be centered
				"╰────────╯",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create canvas and renderer
			canvas := NewMatrixCanvas(20, 10)
			renderer := NewNodeRenderer(TerminalCapabilities{
				UnicodeLevel: UnicodeFull,
			})

			// Set hints on the node
			if tt.hints != nil {
				tt.node.Hints = tt.hints
			}

			// Render the node
			err := renderer.RenderNode(canvas, tt.node)
			if err != nil {
				t.Fatalf("Failed to render node: %v", err)
			}

			// Get the rendered output
			output := canvas.String()
			lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

			// Check expected lines
			for i, expectedLine := range tt.expected {
				if i >= len(lines) {
					t.Errorf("Missing line %d: expected %q", i, expectedLine)
					continue
				}
				// Trim trailing spaces for comparison
				actualLine := strings.TrimRight(lines[i], " ")
				expectedLine = strings.TrimRight(expectedLine, " ")
				if actualLine != expectedLine {
					t.Errorf("Line %d mismatch:\nExpected: %q\nActual:   %q", i, expectedLine, actualLine)
				}
			}
		})
	}
}

func TestCenterTextAlignment(t *testing.T) {
	// Test the centering calculation specifically
	tests := []struct {
		name           string
		text           string
		nodeWidth      int
		expectedOffset int // Expected x offset from left border
	}{
		{
			name:           "exact fit",
			text:           "12345678",
			nodeWidth:      10, // 8 chars + 2 borders
			expectedOffset: 1,  // No centering needed
		},
		{
			name:           "small text in large node",
			text:           "Hi",
			nodeWidth:      10,                     // 8 chars available
			expectedOffset: 1 + (8-2)/2,            // 1 + 3 = 4
		},
		{
			name:           "odd spacing",
			text:           "Test",
			nodeWidth:      11,                     // 9 chars available
			expectedOffset: 1 + (9-4)/2,            // 1 + 2 = 3 (rounds down)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This mimics the calculation in drawText
			textWidth := len(tt.text)
			availableWidth := tt.nodeWidth - 2 // minus borders
			x := 1 // default padding
			if textWidth < availableWidth {
				x = 1 + (availableWidth-textWidth)/2
			}

			if x != tt.expectedOffset {
				t.Errorf("Expected x offset %d, got %d", tt.expectedOffset, x)
			}
		})
	}
}
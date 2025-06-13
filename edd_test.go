package main

import (
	"strings"
	"testing"
)

func TestCalculateNodeSize(t *testing.T) {
	tests := []struct {
		name      string
		text      []string
		expWidth  int
		expHeight int
	}{
		{
			name:      "single line text",
			text:      []string{"hello"},
			expWidth:  16, // Minimum width
			expHeight: 3,  // top + content + bottom
		},
		{
			name:      "multi-line text",
			text:      []string{"line1", "line2"},
			expWidth:  16, // Minimum width
			expHeight: 4,  // top + 2 content + bottom
		},
		{
			name:      "long text exceeds minimum",
			text:      []string{"this is a very long line of text"},
			expWidth:  38, // text length + padding + borders
			expHeight: 3,
		},
		{
			name:      "empty node",
			text:      []string{},
			expWidth:  16, // minimum width
			expHeight: 3, // minimum height
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			width, height := CalculateNodeSize(tt.text)
			if width != tt.expWidth || height != tt.expHeight {
				t.Errorf("CalculateNodeSize(%v) = (%d, %d), want (%d, %d)",
					tt.text, width, height, tt.expWidth, tt.expHeight)
			}
		})
	}
}

func TestDrawBox(t *testing.T) {
	tests := []struct {
		name     string
		node     Node
		expected string
	}{
		{
			name: "simple box with text",
			node: Node{X: 0, Y: 0, Width: 10, Height: 3, Text: []string{"test"}},
			expected: `╭────────╮
│  test  │
╰────────╯`,
		},
		{
			name: "multi-line text box",
			node: Node{X: 0, Y: 0, Width: 12, Height: 4, Text: []string{"line1", "line2"}},
			expected: `╭──────────╮
│  line1   │
│  line2   │
╰──────────╯`,
		},
		{
			name: "empty box",
			node: Node{X: 0, Y: 0, Width: 8, Height: 3, Text: []string{}},
			expected: `╭──────╮
│      │
╰──────╯`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canvas := NewCanvas(20, 10)
			canvas.DrawBox(tt.node)

			actual := strings.TrimSpace(canvas.String())
			expected := strings.TrimSpace(tt.expected)

			if actual != expected {
				t.Errorf("DrawBox failed\nExpected:\n%s\n\nActual:\n%s", expected, actual)
			}
		})
	}
}

func TestLayeredLayout(t *testing.T) {
	tests := []struct {
		name     string
		diagram  Diagram
		expected string
	}{
		{
			name: "linear chain",
			diagram: Diagram{
				Nodes: []Node{
					{ID: 0, Text: []string{"login"}},
					{ID: 1, Text: []string{"validate"}},
					{ID: 2, Text: []string{"dashboard"}},
				},
				Connections: []Connection{
					{From: 0, To: 1},
					{From: 1, To: 2},
				},
			},
			expected: `
╭──────────────╮    ╭──────────────╮    ╭──────────────╮
│    login     ├───▶│   validate   ├───▶│  dashboard   │
╰──────────────╯    ╰──────────────╯    ╰──────────────╯`,
		},
		{
			name: "branching pattern",
			diagram: Diagram{
				Nodes: []Node{
					{ID: 0, Text: []string{"auth"}},
					{ID: 1, Text: []string{"user dashboard"}},
					{ID: 2, Text: []string{"admin panel"}},
				},
				Connections: []Connection{
					{From: 0, To: 1},
					{From: 0, To: 2},
				},
			},
			expected: `
╭──────────────╮    ╭──────────────────╮
│     auth     ├─┬─▶│  user dashboard  │
╰──────────────╯ │  ╰──────────────────╯
                 │
                 │
                 │
                 │
                 │  ╭───────────────╮
                 ╰─▶│  admin panel  │
                    ╰───────────────╯`,
		},
		{
			name: "converging pattern",
			diagram: Diagram{
				Nodes: []Node{
					{ID: 0, Text: []string{"frontend"}},
					{ID: 1, Text: []string{"api"}},
					{ID: 2, Text: []string{"database"}},
				},
				Connections: []Connection{
					{From: 0, To: 2},
					{From: 1, To: 2},
				},
			},
			expected: `
╭──────────────╮    ╭──────────────╮
│   frontend   ├─┬─▶│   database   │
╰──────────────╯ │  ╰──────────────╯
                 │
                 │
                 │
                 │
╭──────────────╮ │
│     api      ├─╯
╰──────────────╯`,
		},
		{
			name: "diamond pattern (decision flow)",
			diagram: Diagram{
				Nodes: []Node{
					{ID: 0, Text: []string{"start"}},
					{ID: 1, Text: []string{"validate"}},
					{ID: 2, Text: []string{"success"}},
					{ID: 3, Text: []string{"error"}},
					{ID: 4, Text: []string{"end"}},
				},
				Connections: []Connection{
					{From: 0, To: 1},
					{From: 1, To: 2},
					{From: 1, To: 3},
					{From: 2, To: 4},
					{From: 3, To: 4},
				},
			},
			expected: `
╭──────────────╮    ╭──────────────╮    ╭──────────────╮    ╭──────────────╮
│    start     ├───▶│   validate   ├─┬─▶│   success    ├─┬─▶│     end      │
╰──────────────╯    ╰──────────────╯ │  ╰──────────────╯ │  ╰──────────────╯
                                     │                   │
                                     │                   │
                                     │                   │
                                     │                   │
                                     │  ╭──────────────╮ │
                                     ╰─▶│    error     ├─╯
                                        ╰──────────────╯`,
		},
		{
			name: "hub and spoke pattern",
			diagram: Diagram{
				Nodes: []Node{
					{ID: 0, Text: []string{"gateway"}},
					{ID: 1, Text: []string{"users"}},
					{ID: 2, Text: []string{"orders"}},
					{ID: 3, Text: []string{"inventory"}},
					{ID: 4, Text: []string{"billing"}},
				},
				Connections: []Connection{
					{From: 0, To: 1},
					{From: 0, To: 2},
					{From: 0, To: 3},
					{From: 0, To: 4},
				},
			},
			expected: `
╭──────────────╮    ╭──────────────╮
│   gateway    ├─┬─▶│    users     │
╰──────────────╯ │  ╰──────────────╯
                 │
                 │
                 │
                 │
                 │  ╭──────────────╮
                 ├─▶│    orders    │
                 │  ╰──────────────╯
                 │
                 │
                 │
                 │
                 │  ╭──────────────╮
                 ├─▶│  inventory   │
                 │  ╰──────────────╯
                 │
                 │
                 │
                 │
                 │  ╭──────────────╮
                 ╰─▶│   billing    │
                    ╰──────────────╯`,
		},
		{
			name: "pipeline with feedback",
			diagram: Diagram{
				Nodes: []Node{
					{ID: 0, Text: []string{"input"}},
					{ID: 1, Text: []string{"process"}},
					{ID: 2, Text: []string{"validate"}},
					{ID: 3, Text: []string{"output"}},
				},
				Connections: []Connection{
					{From: 0, To: 1},
					{From: 1, To: 2},
					{From: 2, To: 3},
					{From: 2, To: 1}, // feedback loop
				},
			},
			expected: `
╭──────────────╮    ╭──────────────╮    ╭──────────────╮    ╭──────────────╮
│    input     ├───▶│   process    ├───▶│   validate   ├───▶│    output    │
╰──────────────╯    ╰──────────────╯    ╰──────────────╯    ╰──────────────╯`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For layered layout, we need to calculate canvas size after layout
			layout := NewLayeredLayout()
			positioned := layout.CalculateLayout(tt.diagram.Nodes, tt.diagram.Connections)

			// Calculate required canvas size
			maxX, maxY := 0, 0
			for _, node := range positioned {
				if node.X+node.Width > maxX {
					maxX = node.X + node.Width
				}
				if node.Y+node.Height > maxY {
					maxY = node.Y + node.Height
				}
			}

			// Add space for connections
			canvas := NewCanvas(maxX+20, maxY+10) // More padding for routing
			canvas.Render(tt.diagram)

			// Compare with expected output
			actual := strings.TrimSpace(canvas.String())
			expected := strings.TrimSpace(tt.expected)

			// All tests should pass exactly as specified
			if actual != expected {
				t.Errorf("Test %s failed\nExpected:\n%s\n\nActual:\n%s", tt.name, expected, actual)
			}
		})
	}
}

func TestCanvasOperations(t *testing.T) {
	t.Run("set and get", func(t *testing.T) {
		canvas := NewCanvas(10, 10)
		canvas.Set(5, 5, 'X')
		if got := canvas.Get(5, 5); got != 'X' {
			t.Errorf("Get(5,5) = %c, want X", got)
		}
	})

	t.Run("out of bounds", func(t *testing.T) {
		canvas := NewCanvas(10, 10)
		canvas.Set(15, 15, 'X') // Should not panic
		if got := canvas.Get(15, 15); got != ' ' {
			t.Errorf("Get(15,15) = %c, want space", got)
		}
	})

	t.Run("clear", func(t *testing.T) {
		canvas := NewCanvas(5, 5)
		canvas.Set(2, 2, 'X')
		canvas.Clear()
		if got := canvas.Get(2, 2); got != ' ' {
			t.Errorf("After Clear(), Get(2,2) = %c, want space", got)
		}
	})
}


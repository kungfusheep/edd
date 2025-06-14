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
			expHeight: 3,  // minimum height
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
                 │  ╭──────────────╮
                 ├─▶│    orders    │
                 │  ╰──────────────╯
                 │
                 │  ╭──────────────╮
                 ├─▶│  inventory   │
                 │  ╰──────────────╯
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
					{From: 2, To: 0}, // feedback loop
				},
			},
			expected: `
╭──────────────╮    ╭──────────────╮    ╭──────────────╮    ╭──────────────╮
│    input     ├───▶│   process    ├◀──▶┤   validate   ├───▶│    output    │
╰───────▲──────╯    ╰──────────────╯    ╰───────┬──────╯    ╰──────────────╯
        │                                       │
        │                                       │
        ╰───────────────────────────────────────╯`,
		},
		{
			name: "feedback loop with collision avoidance",
			diagram: Diagram{
				Nodes: []Node{
					{ID: 0, Text: []string{"1"}},
					{ID: 1, Text: []string{"2"}},
					{ID: 2, Text: []string{"3"}},
					{ID: 3, Text: []string{"4"}},
					{ID: 4, Text: []string{"5"}},
					{ID: 5, Text: []string{"6"}},
				},
				Connections: []Connection{
					{From: 0, To: 1}, // 1 -> 2
					{From: 0, To: 2}, // 1 -> 3
					{From: 0, To: 3}, // 1 -> 4
					{From: 3, To: 4}, // 4 -> 5
					{From: 2, To: 5}, // 3 -> 6
					{From: 5, To: 0}, // 6 -> 1 (feedback)
				},
			},
			expected: `
╭──────────────╮    ╭──────────────╮
│      1       ├─┬─▶│      2       │
╰──────────────╯ │  ╰──────────────╯
        ▲        │
        │        │  ╭──────────────╮    ╭──────────────╮
        │        ├─▶│      3       ├───▶│      6       │
        │        │  ╰──────────────╯    ╰──────────────╯
        │        │                               ┬
        │        │  ╭──────────────╮    ╭─────────┼──╮
        │        ╰─▶│      4       ├───▶│      5  │  │
        │           ╰──────────────╯    ╰─────────┼──╯
        │                                        │
        │                                        │
        │                                        │
        ╰────────────────────────────────────────╯`,
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

			// Add space for connections and feedback loops
			canvas := NewCanvas(maxX+20, maxY+15) // Extra padding for feedback routing
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

// func TestEditor(t *testing.T) {
// 	t.Run("new editor starts in normal mode", func(t *testing.T) {
// 		editor := NewEditor()
// 		if editor.mode != ModeNormal {
// 			t.Errorf("Expected ModeNormal, got %v", editor.mode)
// 		}
// 		if editor.currentNode != -1 {
// 			t.Errorf("Expected no selected node (-1), got %d", editor.currentNode)
// 		}
// 		if editor.nextNodeID != 0 {
// 			t.Errorf("Expected nextNodeID 0, got %d", editor.nextNodeID)
// 		}
// 	})
//
// 	t.Run("mode string representation", func(t *testing.T) {
// 		tests := []struct {
// 			mode Mode
// 			want string
// 		}{
// 			{ModeNormal, "NORMAL"},
// 			{ModeInsert, "INSERT"},
// 			{ModeConnect, "CONNECT"},
// 			{ModeCommand, "COMMAND"},
// 		}
// 		for _, tt := range tests {
// 			if got := tt.mode.String(); got != tt.want {
// 				t.Errorf("Mode(%d).String() = %q, want %q", tt.mode, got, tt.want)
// 			}
// 		}
// 	})
//
// 	t.Run("set mode", func(t *testing.T) {
// 		editor := NewEditor()
// 		editor.SetMode(ModeInsert)
// 		if editor.mode != ModeInsert {
// 			t.Errorf("Expected ModeInsert, got %v", editor.mode)
// 		}
// 	})
//
// 	t.Run("add node", func(t *testing.T) {
// 		editor := NewEditor()
// 		nodeID := editor.AddNode([]string{"test"})
//
// 		if nodeID != 0 {
// 			t.Errorf("Expected first node ID to be 0, got %d", nodeID)
// 		}
// 		if len(editor.diagram.Nodes) != 1 {
// 			t.Errorf("Expected 1 node, got %d", len(editor.diagram.Nodes))
// 		}
// 		if editor.currentNode != 0 {
// 			t.Errorf("Expected current node to be 0, got %d", editor.currentNode)
// 		}
// 		if editor.nextNodeID != 1 {
// 			t.Errorf("Expected next node ID to be 1, got %d", editor.nextNodeID)
// 		}
//
// 		node := editor.diagram.Nodes[0]
// 		if node.ID != 0 {
// 			t.Errorf("Expected node ID 0, got %d", node.ID)
// 		}
// 		if len(node.Text) != 1 || node.Text[0] != "test" {
// 			t.Errorf("Expected node text ['test'], got %v", node.Text)
// 		}
// 	})
// }

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

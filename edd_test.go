package main

import (
	"strings"
	"testing"
)

func TestCalculateNodeSize(t *testing.T) {
	tests := []struct {
		name         string
		text         []string
		wantWidth    int
		wantHeight   int
	}{
		{
			name:       "single line text",
			text:       []string{"hello"},
			wantWidth:  NodeMinWidth, // Should use minimum width
			wantHeight: 3,
		},
		{
			name:       "multi-line text",
			text:       []string{"hello", "world"},
			wantWidth:  NodeMinWidth,
			wantHeight: 4,
		},
		{
			name:       "long text exceeds minimum",
			text:       []string{"this is a longer text"},
			wantWidth:  27, // 21 chars + 4 padding + 2 borders
			wantHeight: 3,
		},
		{
			name:       "empty node",
			text:       []string{},
			wantWidth:  NodeMinWidth,
			wantHeight: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			width, height := CalculateNodeSize(tt.text)
			if width != tt.wantWidth {
				t.Errorf("width = %d, want %d", width, tt.wantWidth)
			}
			if height != tt.wantHeight {
				t.Errorf("height = %d, want %d", height, tt.wantHeight)
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
			node: Node{
				ID:     1,
				X:      0,
				Y:      0,
				Width:  16,
				Height: 3,
				Text:   []string{"hello"},
			},
			expected: `╭──────────────╮
│    hello     │
╰──────────────╯`,
		},
		{
			name: "multi-line text box",
			node: Node{
				ID:     1,
				X:      0,
				Y:      0,
				Width:  16,
				Height: 4,
				Text:   []string{"hello", "world"},
			},
			expected: `╭──────────────╮
│    hello     │
│    world     │
╰──────────────╯`,
		},
		{
			name: "empty box",
			node: Node{
				ID:     1,
				X:      0,
				Y:      0,
				Width:  16,
				Height: 3,
				Text:   []string{},
			},
			expected: `╭──────────────╮
│              │
╰──────────────╯`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canvas := NewCanvas(20, 10)
			canvas.DrawBox(tt.node)
			
			// Extract the relevant portion of the canvas
			lines := strings.Split(canvas.String(), "\n")
			var trimmedLines []string
			for i := 0; i < tt.node.Height; i++ {
				trimmedLines = append(trimmedLines, strings.TrimRight(lines[i], " "))
			}
			actual := strings.Join(trimmedLines, "\n")
			
			// Normalize expected
			expected := strings.TrimSpace(tt.expected)
			
			if actual != expected {
				t.Errorf("DrawBox() output mismatch\nwant:\n%s\ngot:\n%s", expected, actual)
			}
		})
	}
}

func TestComplexDiagrams(t *testing.T) {
	tests := []struct {
		name     string
		diagram  Diagram
		expected string
	}{
		{
			name: "horizontal chain",
			diagram: Diagram{
				Nodes: []Node{
					{ID: 0, X: 0, Y: 0, Width: 16, Height: 3, Text: []string{"a"}},
					{ID: 1, X: 24, Y: 0, Width: 16, Height: 3, Text: []string{"b"}},
					{ID: 2, X: 48, Y: 0, Width: 16, Height: 3, Text: []string{"c"}},
				},
				Connections: []Connection{
					{From: 0, To: 1},
					{From: 1, To: 2},
				},
			},
			expected: `
╭──────────────╮        ╭──────────────╮        ╭──────────────╮
│      a       ├───────▶│      b       ├───────▶│      c       │
╰──────────────╯        ╰──────────────╯        ╰──────────────╯`,
		},
		{
			name: "multiple connections from one node",
			diagram: Diagram{
				Nodes: []Node{
					{ID: 0, X: 0, Y: 0, Width: 16, Height: 3, Text: []string{"a"}},
					{ID: 1, X: 24, Y: 0, Width: 16, Height: 3, Text: []string{"b"}},
					{ID: 2, X: 24, Y: 11, Width: 16, Height: 3, Text: []string{"c"}},
				},
				Connections: []Connection{
					{From: 0, To: 1},
					{From: 0, To: 2},
				},
			},
			expected: `
╭──────────────╮        ╭──────────────╮
│      a       ├───┬───▶│      b       │
╰──────────────╯   │    ╰──────────────╯
                   │    
                   │    
                   │    ╭──────────────╮
                   └───▶│      c       │
                        ╰──────────────╯`,
		},
		{
			name: "bidirectional connection",
			diagram: Diagram{
				Nodes: []Node{
					{ID: 0, X: 0, Y: 0, Width: 16, Height: 3, Text: []string{"a"}},
					{ID: 1, X: 24, Y: 0, Width: 16, Height: 3, Text: []string{"b"}},
				},
				Connections: []Connection{
					{From: 0, To: 1},
					{From: 1, To: 0},
				},
			},
			expected: `
╭──────────────╮        ╭──────────────╮
│      a       ├───────▶│      b       │
│              │◀───────┤              │
╰──────────────╯        ╰──────────────╯`,
		},
		{
			name: "hub pattern",
			diagram: Diagram{
				Nodes: []Node{
					{ID: 0, X: 24, Y: 6, Width: 16, Height: 3, Text: []string{"hub"}},
					{ID: 1, X: 0, Y: 0, Width: 16, Height: 3, Text: []string{"a"}},
					{ID: 2, X: 48, Y: 0, Width: 16, Height: 3, Text: []string{"b"}},
					{ID: 3, X: 0, Y: 12, Width: 16, Height: 3, Text: []string{"c"}},
					{ID: 4, X: 48, Y: 12, Width: 16, Height: 3, Text: []string{"d"}},
				},
				Connections: []Connection{
					{From: 0, To: 1},
					{From: 0, To: 2},
					{From: 0, To: 3},
					{From: 0, To: 4},
				},
			},
			expected: `
╭──────────────╮        ╭──────────────╮        ╭──────────────╮
│      a       │◀──╮    │     hub      ├───────▶│      b       │
╰──────────────╯   │    ╰──────┬───────╯        ╰──────────────╯
                   │            │
                   ╰────────────┤
                                │
╭──────────────╮                │                ╭──────────────╮
│      c       │◀───────────────┴───────────────▶│      d       │
╰──────────────╯                                 ╰──────────────╯`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate canvas size needed
			maxX, maxY := 0, 0
			for _, node := range tt.diagram.Nodes {
				if node.X+node.Width > maxX {
					maxX = node.X + node.Width
				}
				if node.Y+node.Height > maxY {
					maxY = node.Y + node.Height
				}
			}
			
			canvas := NewCanvas(maxX+10, maxY+5) // Add some padding
			canvas.Render(tt.diagram)
			
			// Compare with expected output
			actual := canvas.String()
			expected := strings.TrimSpace(tt.expected)
			
			// For now, just print both for visual comparison
			// We'll implement proper comparison once routing is done
			t.Logf("Expected:\n%s\n\nActual:\n%s", expected, actual)
			
			// Debug: print connection count
			// t.Logf("Connections in diagram: %d", len(tt.diagram.Connections))
		})
	}
}

func TestBidirectionalRouting(t *testing.T) {
	// Simple test to debug bidirectional connections
	nodes := []Node{
		{ID: 0, X: 0, Y: 0, Width: 16, Height: 3, Text: []string{"a"}},
		{ID: 1, X: 24, Y: 0, Width: 16, Height: 3, Text: []string{"b"}},
	}
	connections := []Connection{
		{From: 0, To: 1},
		{From: 1, To: 0},
	}
	
	plan := PlanRouting(nodes, connections)
	
	t.Logf("Number of connections in plan: %d", len(plan.Connections))
	for key, path := range plan.Connections {
		t.Logf("Connection %s has %d points", key, len(path))
		if len(path) > 0 {
			t.Logf("  First point: (%d,%d) '%c'", path[0].X, path[0].Y, path[0].Rune)
			t.Logf("  Last point: (%d,%d) '%c'", path[len(path)-1].X, path[len(path)-1].Y, path[len(path)-1].Rune)
		}
	}
	
	// Test rendering
	canvas := NewCanvas(50, 10)
	canvas.Render(Diagram{Nodes: nodes, Connections: connections})
	t.Logf("Rendered:\n%s", canvas.String())
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
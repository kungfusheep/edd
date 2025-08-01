package core

import (
	"testing"
)

func TestDirection(t *testing.T) {
	tests := []struct {
		name     string
		dir      Direction
		expected string
		opposite Direction
	}{
		{"North", North, "North", South},
		{"East", East, "East", West},
		{"South", South, "South", North},
		{"West", West, "West", East},
		{"Invalid", Direction(99), "Unknown", Direction(99)},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.dir.String(); got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
			if got := tt.dir.Opposite(); got != tt.opposite {
				t.Errorf("Opposite() = %v, want %v", got, tt.opposite)
			}
		})
	}
}

func TestNode(t *testing.T) {
	node := Node{
		ID:     1,
		Text:   []string{"Test", "Node"},
		X:      10,
		Y:      20,
		Width:  8,
		Height: 4,
	}
	
	t.Run("Center", func(t *testing.T) {
		center := node.Center()
		if center.X != 14 || center.Y != 22 {
			t.Errorf("Center() = %v, want (14, 22)", center)
		}
	})
	
	t.Run("Contains", func(t *testing.T) {
		tests := []struct {
			point    Point
			contains bool
		}{
			{Point{10, 20}, true},  // Top-left corner
			{Point{17, 23}, true},  // Bottom-right corner (exclusive)
			{Point{14, 22}, true},  // Center
			{Point{9, 20}, false},  // Just outside left
			{Point{18, 20}, false}, // Just outside right
			{Point{10, 19}, false}, // Just outside top
			{Point{10, 24}, false}, // Just outside bottom
		}
		
		for _, tt := range tests {
			if got := node.Contains(tt.point); got != tt.contains {
				t.Errorf("Contains(%v) = %v, want %v", tt.point, got, tt.contains)
			}
		}
	})
}

func TestPath(t *testing.T) {
	t.Run("Empty path", func(t *testing.T) {
		p := Path{}
		if !p.IsEmpty() {
			t.Error("IsEmpty() = false, want true")
		}
		if p.Length() != 0 {
			t.Errorf("Length() = %d, want 0", p.Length())
		}
	})
	
	t.Run("Non-empty path", func(t *testing.T) {
		p := Path{
			Points: []Point{{0, 0}, {1, 0}, {2, 0}},
			Cost:   3,
		}
		if p.IsEmpty() {
			t.Error("IsEmpty() = true, want false")
		}
		if p.Length() != 3 {
			t.Errorf("Length() = %d, want 3", p.Length())
		}
	})
}

func TestBounds(t *testing.T) {
	b := Bounds{
		Min: Point{10, 20},
		Max: Point{50, 40},
	}
	
	t.Run("Dimensions", func(t *testing.T) {
		if w := b.Width(); w != 40 {
			t.Errorf("Width() = %d, want 40", w)
		}
		if h := b.Height(); h != 20 {
			t.Errorf("Height() = %d, want 20", h)
		}
	})
	
	t.Run("Contains", func(t *testing.T) {
		tests := []struct {
			point    Point
			contains bool
		}{
			{Point{10, 20}, true},  // Min corner (inclusive)
			{Point{49, 39}, true},  // Just inside max corner
			{Point{50, 40}, false}, // Max corner (exclusive)
			{Point{30, 30}, true},  // Middle
			{Point{9, 20}, false},  // Just outside left
			{Point{10, 19}, false}, // Just outside top
		}
		
		for _, tt := range tests {
			if got := b.Contains(tt.point); got != tt.contains {
				t.Errorf("Contains(%v) = %v, want %v", tt.point, got, tt.contains)
			}
		}
	})
}

func TestDiagramTypes(t *testing.T) {
	// Test that our JSON-tagged types work correctly
	diagram := Diagram{
		Nodes: []Node{
			{ID: 0, Text: []string{"Node A"}},
			{ID: 1, Text: []string{"Node B"}},
		},
		Connections: []Connection{
			{From: 0, To: 1},
		},
		Metadata: Metadata{
			Name:    "Test Diagram",
			Created: "2024-01-01",
			Version: "1.0",
		},
	}
	
	if len(diagram.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(diagram.Nodes))
	}
	if len(diagram.Connections) != 1 {
		t.Errorf("Expected 1 connection, got %d", len(diagram.Connections))
	}
}
package canvas

import (
	"edd/core"
	"testing"
)

func TestPreserveCornersMode(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		height   int
		paths    []core.Path
		expected string
	}{
		{
			name:   "simple box with preserve corners",
			width:  5,
			height: 3,
			paths: []core.Path{
				{Points: []core.Point{
					{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 2}, {X: 0, Y: 2}, {X: 0, Y: 0},
				}},
			},
			expected: `┌───┐
│   │
└───┘`,
		},
		{
			name:   "overlapping boxes preserve corners",
			width:  7,
			height: 4,
			paths: []core.Path{
				{Points: []core.Point{
					{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 2}, {X: 0, Y: 2}, {X: 0, Y: 0},
				}},
				{Points: []core.Point{
					{X: 2, Y: 1}, {X: 6, Y: 1}, {X: 6, Y: 3}, {X: 2, Y: 3}, {X: 2, Y: 1},
				}},
			},
			expected: `┌───┐  
│ ┌─┼─┐
└─┼─┘ │
  └───┘`,
		},
		{
			name:   "L-shaped path",
			width:  5,
			height: 4,
			paths: []core.Path{
				{Points: []core.Point{
					{X: 0, Y: 0}, {X: 0, Y: 3}, {X: 4, Y: 3},
				}},
			},
			expected: `│    
│    
│    
└──▶ `,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create canvas and renderer
			c := NewMatrixCanvas(tt.width, tt.height)
			renderer := NewPathRenderer(TerminalCapabilities{
				UnicodeLevel: UnicodeFull,
			})
			
			// Enable preserve corners mode
			renderer.SetRenderMode(RenderModePreserveCorners)
			
			// Draw all paths
			for _, path := range tt.paths {
				hasArrow := false
				// Check if last segment should have arrow (simple heuristic)
				if len(path.Points) > 1 {
					last := path.Points[len(path.Points)-1]
					// Only add arrow for non-closed paths
					if path.Points[0] != last {
						hasArrow = true
					}
				}
				
				err := renderer.RenderPath(c, path, hasArrow)
				if err != nil {
					t.Fatalf("Failed to render path: %v", err)
				}
			}
			
			// Check the output
			output := c.String()
			if output != tt.expected {
				t.Errorf("Unexpected output:\nGot:\n%s\nExpected:\n%s", output, tt.expected)
			}
		})
	}
}

func TestPreserveCornersVsStandardMode(t *testing.T) {
	// Test that shows the difference between standard and preserve corners mode
	paths := []core.Path{
		{Points: []core.Point{
			{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 2}, {X: 0, Y: 2}, {X: 0, Y: 0},
		}},
		{Points: []core.Point{
			{X: 2, Y: 0}, {X: 2, Y: 2},
		}},
	}
	
	// Test with standard mode
	t.Run("standard mode creates junction", func(t *testing.T) {
		c := NewMatrixCanvas(5, 3)
		renderer := NewPathRenderer(TerminalCapabilities{UnicodeLevel: UnicodeFull})
		renderer.SetRenderMode(RenderModeStandard)
		
		for _, path := range paths {
			renderer.RenderPath(c, path, false)
		}
		
		// In standard mode with a 2-point vertical line:
		// - The line doesn't include its endpoint, so bottom junction isn't created
		// - Top junction is created where the line starts
		expected := `┌─┼─┐
│ │ │
└───┘`
		
		if output := c.String(); output != expected {
			t.Errorf("Standard mode output incorrect:\nGot:\n%s\nExpected:\n%s", output, expected)
		}
	})
	
	// Test with preserve corners mode
	t.Run("preserve corners mode keeps corners", func(t *testing.T) {
		c := NewMatrixCanvas(5, 3)
		renderer := NewPathRenderer(TerminalCapabilities{UnicodeLevel: UnicodeFull})
		renderer.SetRenderMode(RenderModePreserveCorners)
		
		for _, path := range paths {
			renderer.RenderPath(c, path, false)
		}
		
		// In preserve corners mode with a 2-point vertical line:
		// - The line doesn't include its endpoint, so bottom junction isn't created
		// - Top junction is created where the line starts
		expected := `┌─┼─┐
│ │ │
└───┘`
		
		if output := c.String(); output != expected {
			t.Errorf("Preserve corners mode output incorrect:\nGot:\n%s\nExpected:\n%s", output, expected)
		}
	})
}
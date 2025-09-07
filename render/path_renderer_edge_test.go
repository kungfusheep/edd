package render

import (
	"strings"
	"testing"

	"edd/diagram"
)

func TestPathRenderer_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		caps     TerminalCapabilities
		path     diagram.Path
		hasArrow bool
		width    int
		height   int
		expected string
	}{
		{
			name: "empty path",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: diagram.Path{
				Points: []diagram.Point{},
			},
			hasArrow: false,
			width:    5,
			height:   5,
			expected: `     
     
     
     
     `,
		},
		{
			name: "single point without arrow",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: diagram.Path{
				Points: []diagram.Point{{X: 2, Y: 2}},
			},
			hasArrow: false,
			width:    5,
			height:   5,
			expected: `     
     
     
     
     `,
		},
		{
			name: "single point with arrow",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: diagram.Path{
				Points: []diagram.Point{{X: 2, Y: 2}},
			},
			hasArrow: true,
			width:    5,
			height:   5,
			expected: `     
     
  •  
     
     `,
		},
		{
			name: "crossing paths horizontal/vertical",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: diagram.Path{}, // We'll render two paths
			hasArrow: false,
			width:    7,
			height:   5,
		},
		{
			name: "path with junction at corner",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: diagram.Path{}, // We'll render two paths
			hasArrow: false,
			width:    7,
			height:   5,
		},
		{
			name: "zero-length horizontal line",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: diagram.Path{
				Points: []diagram.Point{{X: 2, Y: 2}, {X: 2, Y: 2}},
			},
			hasArrow: false,
			width:    5,
			height:   5,
			expected: `     
     
     
     
     `,
		},
		{
			name: "leftward arrow",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: diagram.Path{
				Points: []diagram.Point{{X: 4, Y: 2}, {X: 1, Y: 2}},
			},
			hasArrow: true,
			width:    6,
			height:   4,
			expected: `      
      
  ◀── 
      `,
		},
		{
			name: "upward arrow",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: diagram.Path{
				Points: []diagram.Point{{X: 2, Y: 3}, {X: 2, Y: 1}},
			},
			hasArrow: true,
			width:    5,
			height:   5,
			expected: `     
     
  ▲  
  │  
     `,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip special tests that need custom handling
			if tt.name == "crossing paths horizontal/vertical" {
				// Test crossing paths
				c := NewMatrixCanvas(tt.width, tt.height)
				renderer := NewPathRenderer(tt.caps)
				
				// Render horizontal line
				path1 := diagram.Path{Points: []diagram.Point{{X: 1, Y: 2}, {X: 5, Y: 2}}}
				renderer.RenderPath(c, path1, false)
				
				// Render vertical line crossing it
				path2 := diagram.Path{Points: []diagram.Point{{X: 3, Y: 0}, {X: 3, Y: 4}}}
				renderer.RenderPath(c, path2, false)
				
				expected := `   │   
   │   
 ──┼─  
   │   
       `
				
				got := c.String()
				want := strings.TrimPrefix(expected, "\n")
				if got != want {
					t.Errorf("Crossing paths mismatch\nGot:\n%s\nWant:\n%s", got, want)
				}
				return
			}
			
			if tt.name == "path with junction at corner" {
				// Test junction at a corner
				c := NewMatrixCanvas(tt.width, tt.height)
				renderer := NewPathRenderer(tt.caps)
				
				// Render L-shaped path
				path1 := diagram.Path{Points: []diagram.Point{{X: 1, Y: 1}, {X: 3, Y: 1}, {X: 3, Y: 3}}}
				renderer.RenderPath(c, path1, false)
				
				// Render line that intersects at the corner
				path2 := diagram.Path{Points: []diagram.Point{{X: 3, Y: 0}, {X: 3, Y: 2}}}
				renderer.RenderPath(c, path2, false)
				
				expected := `   │   
 ──┤   
   │   
   │   
       `
				
				got := c.String()
				want := strings.TrimPrefix(expected, "\n")
				if got != want {
					t.Errorf("Junction at corner mismatch\nGot:\n%s\nWant:\n%s", got, want)
				}
				return
			}
			
			// Regular test case
			c := NewMatrixCanvas(tt.width, tt.height)
			renderer := NewPathRenderer(tt.caps)
			
			err := renderer.RenderPath(c, tt.path, tt.hasArrow)
			if err != nil {
				t.Fatalf("RenderPath failed: %v", err)
			}
			
			got := c.String()
			want := strings.TrimPrefix(tt.expected, "\n")
			
			if got != want {
				t.Errorf("Path rendering mismatch\nGot:\n%s\nWant:\n%s", got, want)
			}
		})
	}
}

func TestPathRenderer_TerminalFallback(t *testing.T) {
	// Test a complex path with different terminal capabilities
	path := diagram.Path{
		Points: []diagram.Point{
			{X: 1, Y: 1},
			{X: 4, Y: 1},
			{X: 4, Y: 3},
			{X: 2, Y: 3},
			{X: 2, Y: 2},
		},
	}
	
	tests := []struct {
		name     string
		caps     TerminalCapabilities
		expected string
	}{
		{
			name: "full unicode",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			expected: `        
 ───┐   
  │ │   
  └─┘   `,
		},
		{
			name: "ASCII only",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeNone},
			expected: `        
 ---+   
  | |   
  +-+   `,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewMatrixCanvas(8, 4)
			renderer := NewPathRenderer(tt.caps)
			// Use preserve corners mode for cleaner output
			renderer.SetRenderMode(RenderModePreserveCorners)
			
			err := renderer.RenderPath(c, path, false)
			if err != nil {
				t.Fatalf("RenderPath failed: %v", err)
			}
			
			got := c.String()
			want := strings.TrimPrefix(tt.expected, "\n")
			
			if got != want {
				t.Errorf("Fallback rendering mismatch\nGot:\n%s\nWant:\n%s", got, want)
			}
		})
	}
}
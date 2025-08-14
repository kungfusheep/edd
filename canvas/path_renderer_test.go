package canvas

import (
	"strings"
	"testing"

	"edd/canvas"
	"edd/core"
)

func TestPathRenderer_RenderPath(t *testing.T) {
	tests := []struct {
		name     string
		caps     TerminalCapabilities
		path     core.Path
		hasArrow bool
		width    int
		height   int
		expected string
	}{
		{
			name: "horizontal line with unicode",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: core.Path{
				Points: []core.Point{{X: 1, Y: 1}, {X: 5, Y: 1}},
			},
			hasArrow: false,
			width:    7,
			height:   3,
			expected: `       
 ────  
       `,
		},
		{
			name: "horizontal line with arrow",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: core.Path{
				Points: []core.Point{{X: 1, Y: 1}, {X: 5, Y: 1}},
			},
			hasArrow: true,
			width:    7,
			height:   3,
			expected: `       
 ───▶  
       `,
		},
		{
			name: "vertical line with unicode",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: core.Path{
				Points: []core.Point{{X: 2, Y: 0}, {X: 2, Y: 3}},
			},
			hasArrow: false,
			width:    5,
			height:   4,
			expected: `  │  
  │  
  │  
     `,
		},
		{
			name: "L-shaped path with corner",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: core.Path{
				Points: []core.Point{{X: 1, Y: 1}, {X: 4, Y: 1}, {X: 4, Y: 3}},
			},
			hasArrow: false,
			width:    6,
			height:   4,
			expected: `      
 ───┐ 
    │ 
    │ `,
		},
		{
			name: "ASCII horizontal line",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeNone},
			path: core.Path{
				Points: []core.Point{{X: 1, Y: 1}, {X: 5, Y: 1}},
			},
			hasArrow: false,
			width:    7,
			height:   3,
			expected: `       
 ----  
       `,
		},
		{
			name: "ASCII L-shaped path",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeNone},
			path: core.Path{
				Points: []core.Point{{X: 1, Y: 1}, {X: 4, Y: 1}, {X: 4, Y: 3}},
			},
			hasArrow: false,
			width:    6,
			height:   4,
			expected: `      
 ---+ 
    | 
    | `,
		},
		{
			name: "complex path with multiple turns",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: core.Path{
				Points: []core.Point{
					{X: 1, Y: 1},
					{X: 3, Y: 1},
					{X: 3, Y: 3},
					{X: 5, Y: 3},
					{X: 5, Y: 1},
				},
			},
			hasArrow: false,
			width:    7,
			height:   5,
			expected: `       
 ──┐ │ 
   │ │ 
   └─┘ 
       `,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create canvas
			c := canvas.NewMatrixCanvas(tt.width, tt.height)
			
			// Create renderer
			renderer := NewPathRenderer(tt.caps)
			// Use preserve corners mode for paths that expect clean corners
			if strings.Contains(tt.name, "path") || strings.Contains(tt.name, "shaped") {
				renderer.SetRenderMode(RenderModePreserveCorners)
			}
			
			// Render path
			err := renderer.RenderPath(c, tt.path, tt.hasArrow)
			if err != nil {
				t.Fatalf("RenderPath failed: %v", err)
			}
			
			// Compare output
			got := c.String()
			want := strings.TrimPrefix(tt.expected, "\n")
			
			if got != want {
				t.Errorf("Path rendering mismatch\nGot:\n%s\nWant:\n%s", got, want)
				// Show difference
				gotLines := strings.Split(got, "\n")
				wantLines := strings.Split(want, "\n")
				for i := 0; i < len(gotLines) && i < len(wantLines); i++ {
					if gotLines[i] != wantLines[i] {
						t.Errorf("Line %d differs:\nGot:  %q\nWant: %q", i, gotLines[i], wantLines[i])
					}
				}
			}
		})
	}
}

func TestJunctionResolver(t *testing.T) {
	jr := NewJunctionResolver()
	
	tests := []struct {
		name     string
		existing rune
		newLine  rune
		expected rune
	}{
		// Basic intersections
		{"horizontal meets vertical", '─', '│', '┼'},
		{"vertical meets horizontal", '│', '─', '┼'},
		
		// Corner junctions
		{"horizontal meets top-left corner", '─', '┌', '┬'},
		{"vertical meets top-left corner", '│', '┌', '├'},
		
		// ASCII intersections
		{"ASCII horizontal meets vertical", '-', '|', '+'},
		{"ASCII vertical meets horizontal", '|', '-', '+'},
		
		// Same character
		{"same horizontal", '─', '─', '─'},
		{"same vertical", '│', '│', '│'},
		
		// Unknown combinations
		{"unknown combo keeps existing", 'A', '─', 'A'},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jr.Resolve(tt.existing, tt.newLine)
			if got != tt.expected {
				t.Errorf("Resolve(%q, %q) = %q, want %q", 
					tt.existing, tt.newLine, got, tt.expected)
			}
		})
	}
}

func TestLineStyleSelection(t *testing.T) {
	tests := []struct {
		name     string
		caps     TerminalCapabilities
		wantHoriz rune
		wantArrow rune
	}{
		{
			"full unicode",
			TerminalCapabilities{UnicodeLevel: UnicodeFull},
			'─',
			'▶',
		},
		{
			"extended unicode",
			TerminalCapabilities{UnicodeLevel: UnicodeExtended},
			'─',
			'▶',
		},
		{
			"basic unicode",
			TerminalCapabilities{UnicodeLevel: UnicodeBasic},
			'-',
			'>',
		},
		{
			"ASCII only",
			TerminalCapabilities{UnicodeLevel: UnicodeNone},
			'-',
			'>',
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			style := selectLineStyle(tt.caps)
			if style.Horizontal != tt.wantHoriz {
				t.Errorf("Horizontal = %q, want %q", style.Horizontal, tt.wantHoriz)
			}
			if style.ArrowRight != tt.wantArrow {
				t.Errorf("ArrowRight = %q, want %q", style.ArrowRight, tt.wantArrow)
			}
		})
	}
}
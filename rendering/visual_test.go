package rendering

import (
	"fmt"
	"strings"
	"testing"

	"edd/canvas"
	"edd/core"
)

// TestVisualRendering creates visual examples of path rendering
func TestVisualRendering(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
		paths  []struct {
			points   []core.Point
			hasArrow bool
		}
		description string
	}{
		{
			name:   "Simple Box",
			width:  20,
			height: 10,
			paths: []struct {
				points   []core.Point
				hasArrow bool
			}{
				{[]core.Point{{2, 2}, {15, 2}, {15, 7}, {2, 7}, {2, 2}}, false},
			},
			description: "A simple rectangular box",
		},
		{
			name:   "Multiple Boxes with Arrows",
			width:  30,
			height: 12,
			paths: []struct {
				points   []core.Point
				hasArrow bool
			}{
				// Box 1
				{[]core.Point{{2, 2}, {10, 2}, {10, 5}, {2, 5}, {2, 2}}, false},
				// Box 2
				{[]core.Point{{15, 2}, {23, 2}, {23, 5}, {15, 5}, {15, 2}}, false},
				// Box 3
				{[]core.Point{{8, 7}, {18, 7}, {18, 10}, {8, 10}, {8, 7}}, false},
				// Arrows between Box 1 and Box 2
				{[]core.Point{{10, 3}, {15, 3}}, true},
				{[]core.Point{{10, 4}, {15, 4}}, true},
				// Arrows from boxes to Box 3 (using orthogonal paths)
				{[]core.Point{{6, 5}, {6, 7}, {8, 7}}, true},
				{[]core.Point{{19, 5}, {19, 7}}, true},
			},
			description: "Three boxes connected with arrows",
		},
		{
			name:   "Complex Path with Multiple Turns",
			width:  25,
			height: 15,
			paths: []struct {
				points   []core.Point
				hasArrow bool
			}{
				{[]core.Point{
					{2, 2}, {10, 2}, {10, 5}, {15, 5}, {15, 2}, {20, 2},
					{20, 10}, {15, 10}, {15, 7}, {10, 7}, {10, 10}, {5, 10},
					{5, 5}, {2, 5}, {2, 2},
				}, false},
			},
			description: "A complex path with many turns forming an intricate shape",
		},
		{
			name:   "Line Intersections and Junctions",
			width:  20,
			height: 10,
			paths: []struct {
				points   []core.Point
				hasArrow bool
			}{
				// Horizontal lines
				{[]core.Point{{2, 2}, {18, 2}}, false},
				{[]core.Point{{2, 5}, {18, 5}}, false},
				{[]core.Point{{2, 8}, {18, 8}}, false},
				// Vertical lines
				{[]core.Point{{5, 1}, {5, 9}}, false},
				{[]core.Point{{10, 1}, {10, 9}}, false},
				{[]core.Point{{15, 1}, {15, 9}}, false},
			},
			description: "Grid pattern demonstrating line intersections",
		},
		{
			name:   "Directed Graph",
			width:  25,
			height: 12,
			paths: []struct {
				points   []core.Point
				hasArrow bool
			}{
				// Nodes (small boxes)
				{[]core.Point{{3, 2}, {7, 2}, {7, 4}, {3, 4}, {3, 2}}, false},
				{[]core.Point{{15, 2}, {19, 2}, {19, 4}, {15, 4}, {15, 2}}, false},
				{[]core.Point{{3, 7}, {7, 7}, {7, 9}, {3, 9}, {3, 7}}, false},
				{[]core.Point{{15, 7}, {19, 7}, {19, 9}, {15, 9}, {15, 7}}, false},
				// Directed edges
				{[]core.Point{{7, 3}, {15, 3}}, true},
				{[]core.Point{{5, 4}, {5, 7}}, true},
				{[]core.Point{{17, 4}, {17, 7}}, true},
				{[]core.Point{{7, 8}, {15, 8}}, true},
			},
			description: "A directed graph with four nodes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test both Unicode and ASCII modes
			for _, unicodeMode := range []bool{true, false} {
				modeName := "Unicode"
				caps := canvas.TerminalCapabilities{UnicodeLevel: canvas.UnicodeFull}
				if !unicodeMode {
					modeName = "ASCII"
					caps = canvas.TerminalCapabilities{UnicodeLevel: canvas.UnicodeNone}
				}

				c := canvas.NewMatrixCanvas(tt.width, tt.height)
				renderer := canvas.NewPathRenderer(caps)
				// Use preserve corners mode for better box appearance
				renderer.SetRenderMode(canvas.RenderModePreserveCorners)

				// Draw all paths
				for _, path := range tt.paths {
					err := renderer.RenderPath(c, core.Path{Points: path.points}, path.hasArrow)
					if err != nil {
						t.Errorf("Failed to render path: %v", err)
					}
				}

				// Display the result
				fmt.Printf("\n=== %s (%s mode) ===\n", tt.name, modeName)
				fmt.Printf("Description: %s\n", tt.description)
				fmt.Printf("Size: %dx%d\n", tt.width, tt.height)
				fmt.Println(strings.Repeat("-", tt.width))
				fmt.Print(c.String())
				fmt.Println(strings.Repeat("-", tt.width))
			}
		})
	}
}

// TestOverlappingPaths tests junction resolution with overlapping paths
func TestOverlappingPaths(t *testing.T) {
	c := canvas.NewMatrixCanvas(15, 10)
	renderer := canvas.NewPathRenderer(canvas.TerminalCapabilities{UnicodeLevel: canvas.UnicodeFull})
	// Use standard mode to test junction resolution
	// (preserve corners mode would keep separate boxes visually distinct)

	// Create overlapping boxes that share edges
	paths := []core.Path{
		// Box 1
		{Points: []core.Point{{2, 2}, {8, 2}, {8, 5}, {2, 5}, {2, 2}}},
		// Box 2 (overlaps right edge of Box 1)
		{Points: []core.Point{{8, 2}, {12, 2}, {12, 5}, {8, 5}, {8, 2}}},
		// Box 3 (overlaps bottom edge of Box 1 and 2)
		{Points: []core.Point{{2, 5}, {12, 5}, {12, 8}, {2, 8}, {2, 5}}},
	}

	fmt.Println("\n=== Overlapping Boxes Test ===")
	fmt.Println("Three boxes with shared edges demonstrating junction resolution")
	
	for _, path := range paths {
		err := renderer.RenderPath(c, path, false)
		if err != nil {
			t.Errorf("Failed to render path: %v", err)
		}
	}

	fmt.Println(strings.Repeat("-", 15))
	fmt.Print(c.String())
	fmt.Println(strings.Repeat("-", 15))

	// Verify corners were drawn (later paths overwrite earlier ones)
	// The point (8,2) will be a corner from the second box
	if char := c.Get(core.Point{X: 8, Y: 2}); char != '┌' && char != '┬' && char != '+' {
		t.Errorf("Expected corner or T-junction at (8,2), got %c", char)
	}
	
	// The point (8,5) will be a T-junction (bottom) where box 3's top edge meets the shared edge
	if char := c.Get(core.Point{X: 8, Y: 5}); char != '┴' && char != '+' {
		t.Errorf("Expected T-junction (bottom) at (8,5), got %c", char)
	}
}
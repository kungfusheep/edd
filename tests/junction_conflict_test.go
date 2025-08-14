package tests

import (
	"edd/canvas"
	"edd/core"
	"edd/validation"
	"testing"
)

// TestJunctionConflict demonstrates the design conflict between validation
// requirements and visual requirements for connections to boxes.
func TestJunctionConflict(t *testing.T) {
	// Create a simple two-box diagram
	nodes := []core.Node{
		{
			ID:     1,
			X:      2,
			Y:      2,
			Width:  7,  // "Hello" + padding
			Height: 3,
			Text:   []string{"Hello"},
		},
		{
			ID:     2,
			X:      15,
			Y:      2,
			Width:  7,  // "World" + padding
			Height: 3,
			Text:   []string{"World"},
		},
	}

	// Connection from right edge of box 1 to left edge of box 2
	// According to getConnectionPoint, this creates:
	// - Start: (8, 3) which is X + Width - 1 = 2 + 7 - 1 = 8
	// - End: (15, 3) which is X = 15
	path := core.Path{
		Points: []core.Point{
			{X: 8, Y: 3},   // Right edge of "Hello" box
			{X: 15, Y: 3},  // Left edge of "World" box
		},
	}

	// Render the diagram
	c := canvas.NewMatrixCanvas(30, 10)
	caps := canvas.TerminalCapabilities{UnicodeLevel: canvas.UnicodeFull}
	renderer := canvas.NewPathRenderer(caps)

	// First draw the boxes
	for _, node := range nodes {
		// Draw box borders
		for x := node.X; x < node.X+node.Width; x++ {
			c.Set(core.Point{X: x, Y: node.Y}, '─')
			c.Set(core.Point{X: x, Y: node.Y + node.Height - 1}, '─')
		}
		for y := node.Y; y < node.Y+node.Height; y++ {
			c.Set(core.Point{X: node.X, Y: y}, '│')
			c.Set(core.Point{X: node.X + node.Width - 1, Y: y}, '│')
		}
		// Corners
		c.Set(core.Point{X: node.X, Y: node.Y}, '┌')
		c.Set(core.Point{X: node.X + node.Width - 1, Y: node.Y}, '┐')
		c.Set(core.Point{X: node.X, Y: node.Y + node.Height - 1}, '└')
		c.Set(core.Point{X: node.X + node.Width - 1, Y: node.Y + node.Height - 1}, '┘')
		
		// Draw text
		c.DrawText(node.X+1, node.Y+1, node.Text[0])
	}

	// Now draw the connection with arrow
	renderer.RenderPath(c, path, true)

	output := c.String()
	t.Logf("Rendered diagram:\n%s", output)

	// Analyze the specific characters at connection points
	startChar := c.Get(core.Point{X: 8, Y: 3})
	endChar := c.Get(core.Point{X: 15, Y: 3})
	
	t.Logf("Character at start point (8,3): %c", startChar)
	t.Logf("Character at end point (15,3): %c", endChar)

	// The issue: we get junction characters where we don't want them
	// Expected: clean arrow connection
	// Actual: junction characters at box edges
	
	// Let's also check with the validator
	validator := validation.NewLineValidator()
	errors := validator.Validate(output)
	
	if len(errors) > 0 {
		t.Logf("Validation errors:")
		for _, err := range errors {
			t.Logf("  %s", err)
		}
	}

	// Demonstrate the conflict:
	// 1. If we place connection at edge, we get unwanted junctions
	// 2. If we place connection outside edge, validator complains about disconnected lines
}

// TestConnectionPointOptions demonstrates different approaches to solving the conflict
func TestConnectionPointOptions(t *testing.T) {
	tests := []struct {
		name        string
		approach    string
		startPoint  core.Point
		endPoint    core.Point
		description string
	}{
		{
			name:     "Current: At Edge",
			approach: "edge",
			startPoint: core.Point{X: 8, Y: 3},  // Right edge of left box
			endPoint:   core.Point{X: 15, Y: 3}, // Left edge of right box
			description: "Connection points at box edges - causes junction characters",
		},
		{
			name:     "Option A: One Step Outside",
			approach: "outside",
			startPoint: core.Point{X: 9, Y: 3},  // One step outside right edge
			endPoint:   core.Point{X: 14, Y: 3}, // One step outside left edge
			description: "Connection points outside boxes - needs connection stubs",
		},
		{
			name:     "Option B: Smart Start/End",
			approach: "smart",
			startPoint: core.Point{X: 8, Y: 3},  // At edge
			endPoint:   core.Point{X: 15, Y: 3}, // At edge
			description: "Special handling for first/last segments to avoid junctions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := canvas.NewMatrixCanvas(30, 7)
			
			// Draw simplified boxes (just the connection points)
			// Left box edge at x=8
			for y := 1; y <= 5; y++ {
				c.Set(core.Point{X: 8, Y: y}, '│')
			}
			// Right box edge at x=15
			for y := 1; y <= 5; y++ {
				c.Set(core.Point{X: 15, Y: y}, '│')
			}

			// Draw the connection based on approach
			path := core.Path{Points: []core.Point{tt.startPoint, tt.endPoint}}
			
			caps := canvas.TerminalCapabilities{UnicodeLevel: canvas.UnicodeFull}
			renderer := canvas.NewPathRenderer(caps)
			renderer.RenderPath(c, path, true)

			output := c.String()
			t.Logf("%s:\n%s", tt.description, output)
			t.Logf("Result:\n%s\n", output)
		})
	}
}
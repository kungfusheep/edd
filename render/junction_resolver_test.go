package render

import (
	"testing"
)

func TestJunctionResolver_ArrowMeetsLine(t *testing.T) {
	jr := NewJunctionResolver()
	
	tests := []struct {
		name     string
		existing rune
		newLine  rune
		want     rune
	}{
		// Triangular arrows meet lines
		{"Right arrow meets vertical", '│', '▶', '├'},
		{"Vertical meets right arrow", '▶', '│', '▶'},  // Arrow is preserved when it's existing
		{"Left arrow meets vertical", '│', '◀', '┤'},
		{"Down arrow meets horizontal", '─', '▼', '┬'},
		{"Up arrow meets horizontal", '─', '▲', '┴'},
		
		// Traditional arrows meet lines
		{"Right arrow → meets vertical", '│', '→', '├'},
		{"Left arrow ← meets vertical", '│', '←', '┤'},
		{"Down arrow ↓ meets horizontal", '─', '↓', '┬'},
		{"Up arrow ↑ meets horizontal", '─', '↑', '┴'},
		
		// ASCII arrows meet lines
		{"ASCII > meets vertical", '|', '>', '+'},
		{"ASCII < meets vertical", '|', '<', '+'},
		{"ASCII v meets horizontal", '-', 'v', '+'},
		{"ASCII ^ meets horizontal", '-', '^', '+'},
		
		// Arrow protection - arrows should not be overridden
		{"Existing arrow not overridden by line", '▶', '│', '▶'},
		{"Existing arrow not overridden by cross", '▼', '─', '▼'},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jr.Resolve(tt.existing, tt.newLine)
			if got != tt.want {
				t.Errorf("Resolve(%c, %c) = %c, want %c", tt.existing, tt.newLine, got, tt.want)
			}
		})
	}
}

func TestJunctionResolver_ArrowMeetsCorner(t *testing.T) {
	jr := NewJunctionResolver()
	
	tests := []struct {
		name     string
		existing rune
		newLine  rune
		want     rune
	}{
		// Right arrow meets corners
		{"Right arrow meets top-left corner", '┌', '▶', '├'},
		{"Right arrow meets bottom-left corner", '└', '▶', '├'},
		{"Right arrow meets top-right corner", '┐', '▶', '┼'},
		{"Right arrow meets bottom-right corner", '┘', '▶', '┼'},
		
		// Down arrow meets corners
		{"Down arrow meets top-left corner", '┌', '▼', '┬'},
		{"Down arrow meets top-right corner", '┐', '▼', '┬'},
		{"Down arrow meets bottom corners", '└', '▼', '┼'},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jr.Resolve(tt.existing, tt.newLine)
			if got != tt.want {
				t.Errorf("Resolve(%c, %c) = %c, want %c", tt.existing, tt.newLine, got, tt.want)
			}
		})
	}
}

func TestJunctionResolver_ArrowMeetsArrow(t *testing.T) {
	jr := NewJunctionResolver()
	
	tests := []struct {
		name     string
		existing rune
		newLine  rune
		want     rune
	}{
		// Perpendicular arrows form crosses
		{"Right meets down", '▶', '▼', '┼'},
		{"Down meets right", '▼', '▶', '┼'},
		{"Left meets up", '◀', '▲', '┼'},
		
		// Same direction arrows - existing is preserved
		{"Same arrow", '▶', '▶', '▶'},
		{"Different arrow same direction", '▶', '→', '▶'},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jr.Resolve(tt.existing, tt.newLine)
			if got != tt.want {
				t.Errorf("Resolve(%c, %c) = %c, want %c", tt.existing, tt.newLine, got, tt.want)
			}
		})
	}
}
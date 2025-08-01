package canvas

import (
	"edd/core"
	"fmt"
	"strings"
	"testing"
)

// TestValidator provides validation utilities for canvas tests.
type TestValidator struct {
	t *testing.T
}

// NewTestValidator creates a validator for canvas tests.
func NewTestValidator(t *testing.T) *TestValidator {
	return &TestValidator{t: t}
}

// ValidateMatrix checks that the canvas matrix contains valid character combinations.
// This is a simplified version - in production we'd use the full LineValidator.
func (v *TestValidator) ValidateMatrix(matrix [][]rune) {
	errors := v.checkCharacterAdjacency(matrix)
	for _, err := range errors {
		v.t.Error(err)
	}
}

// checkCharacterAdjacency validates that adjacent characters are compatible.
func (v *TestValidator) checkCharacterAdjacency(matrix [][]rune) []string {
	var errors []string
	
	for y := 0; y < len(matrix); y++ {
		for x := 0; x < len(matrix[y]); x++ {
			char := matrix[y][x]
			if char == ' ' || isAlphaNum(char) {
				continue
			}
			
			// Check basic adjacency rules
			if x > 0 {
				west := matrix[y][x-1]
				if !isValidAdjacent(char, west, 'W') {
					errors = append(errors, fmt.Sprintf("(%d,%d): %c cannot have %c to the west", x, y, char, west))
				}
			}
			
			if x < len(matrix[y])-1 {
				east := matrix[y][x+1]
				if !isValidAdjacent(char, east, 'E') {
					errors = append(errors, fmt.Sprintf("(%d,%d): %c cannot have %c to the east", x, y, char, east))
				}
			}
			
			if y > 0 && x < len(matrix[y-1]) {
				north := matrix[y-1][x]
				if !isValidAdjacent(char, north, 'N') {
					errors = append(errors, fmt.Sprintf("(%d,%d): %c cannot have %c to the north", x, y, char, north))
				}
			}
			
			if y < len(matrix)-1 && x < len(matrix[y+1]) {
				south := matrix[y+1][x]
				if !isValidAdjacent(char, south, 'S') {
					errors = append(errors, fmt.Sprintf("(%d,%d): %c cannot have %c to the south", x, y, char, south))
				}
			}
		}
	}
	
	return errors
}

// isValidAdjacent checks if two characters can be adjacent in the given direction.
// This is a simplified version for testing.
func isValidAdjacent(from, to rune, dir rune) bool {
	if to == ' ' || isAlphaNum(to) {
		return true
	}
	
	// Simplified rules for testing
	switch from {
	case '─':
		switch dir {
		case 'E', 'W':
			return to == '─' || to == '├' || to == '┤' || to == '┬' || to == '┴' || to == '┼' ||
				   to == '╭' || to == '╮' || to == '╰' || to == '╯' || to == '▶' || to == '◀'
		case 'N', 'S':
			return to == ' ' || to == '▲' || to == '▼'
		}
		
	case '│':
		switch dir {
		case 'N', 'S':
			return to == '│' || to == '├' || to == '┤' || to == '┬' || to == '┴' || to == '┼' ||
				   to == '╭' || to == '╮' || to == '╰' || to == '╯' || to == '▲' || to == '▼'
		case 'E', 'W':
			return to == ' ' || to == '▶' || to == '◀'
		}
		
	case '╭', '╮', '╰', '╯':
		// Box corners - simplified rules
		return true
		
	case '├', '┤', '┬', '┴', '┼':
		// Junctions - simplified rules
		return true
		
	case '▶', '◀', '▲', '▼':
		// Arrows - simplified rules
		return true
		
	case '+':
		// ASCII box corner - can connect to - and |
		switch dir {
		case 'E', 'W':
			return to == '-' || to == '+' || to == ' '
		case 'N', 'S':
			return to == '|' || to == '+' || to == ' '
		}
		
	case '-':
		// ASCII horizontal line
		switch dir {
		case 'E', 'W':
			return to == '-' || to == '+' || to == ' '
		case 'N', 'S':
			return to == ' '
		}
		
	case '|':
		// ASCII vertical line
		switch dir {
		case 'N', 'S':
			return to == '|' || to == '+' || to == ' '
		case 'E', 'W':
			return to == ' '
		}
	}
	
	return false
}

func isAlphaNum(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

// AssertCanvasEquals checks if the canvas output matches expected string.
func (v *TestValidator) AssertCanvasEquals(canvas Canvas, expected string) {
	actual := canvas.String()
	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)
	
	if actual != expected {
		v.t.Errorf("Canvas output mismatch:\nExpected:\n%s\n\nActual:\n%s", expected, actual)
		
		// Show diff
		expectedLines := strings.Split(expected, "\n")
		actualLines := strings.Split(actual, "\n")
		
		for i := 0; i < len(expectedLines) || i < len(actualLines); i++ {
			if i >= len(expectedLines) {
				v.t.Errorf("Extra line %d: %q", i+1, actualLines[i])
			} else if i >= len(actualLines) {
				v.t.Errorf("Missing line %d: %q", i+1, expectedLines[i])
			} else if expectedLines[i] != actualLines[i] {
				v.t.Errorf("Line %d differs:\n  Expected: %q\n  Actual:   %q", i+1, expectedLines[i], actualLines[i])
			}
		}
	}
}

// AssertCharAt verifies a character at a specific position.
func (v *TestValidator) AssertCharAt(canvas Canvas, p core.Point, expected rune) {
	actual := canvas.Get(p)
	if actual != expected {
		v.t.Errorf("Character at (%d,%d): expected %c, got %c", p.X, p.Y, expected, actual)
	}
}

// CreateTestCanvas creates a canvas with some test content for debugging.
func CreateTestCanvas(width, height int) Canvas {
	// This will be replaced with actual canvas implementation
	return nil
}

// DrawTestPattern draws a test pattern on the canvas for visual debugging.
func DrawTestPattern(canvas Canvas) {
	width, height := canvas.Size()
	
	// Draw border
	for x := 0; x < width; x++ {
		canvas.Set(core.Point{X: x, Y: 0}, '─')
		canvas.Set(core.Point{X: x, Y: height - 1}, '─')
	}
	
	for y := 0; y < height; y++ {
		canvas.Set(core.Point{X: 0, Y: y}, '│')
		canvas.Set(core.Point{X: width - 1, Y: y}, '│')
	}
	
	// Corners
	canvas.Set(core.Point{X: 0, Y: 0}, '╭')
	canvas.Set(core.Point{X: width - 1, Y: 0}, '╮')
	canvas.Set(core.Point{X: 0, Y: height - 1}, '╰')
	canvas.Set(core.Point{X: width - 1, Y: height - 1}, '╯')
}
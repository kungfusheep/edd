package validation

import (
	"fmt"
	"strings"
)

// LineValidator validates that rendered diagrams follow proper line drawing rules.
// It checks that adjacent characters are compatible according to box-drawing logic.
type LineValidator struct {
	// Track validation errors
	errors []ValidationError
	// Options
	allowASCII    bool // Allow ASCII characters (-, |, +) mixed with Unicode
	strictMode    bool // Enforce stricter rules (no mixed styles)
	validateBoxes bool // Check that boxes are properly closed
}

// ValidationError represents a validation error with location information.
type ValidationError struct {
	X, Y    int
	Char    rune
	Context string
	Message string
}

// NewLineValidator creates a new validator with default settings.
func NewLineValidator() *LineValidator {
	return &LineValidator{
		allowASCII:    true,
		strictMode:    false,
		validateBoxes: true,
	}
}

// SetStrictMode enables or disables strict validation.
func (v *LineValidator) SetStrictMode(strict bool) {
	v.strictMode = strict
}

// Validate checks a rendered diagram for line drawing errors.
func (v *LineValidator) Validate(diagram string) []ValidationError {
	v.errors = nil
	
	// Convert to 2D grid
	lines := strings.Split(strings.TrimRight(diagram, "\n"), "\n")
	grid := make([][]rune, len(lines))
	for i, line := range lines {
		grid[i] = []rune(line)
	}
	
	// Check each character
	for y := 0; y < len(grid); y++ {
		for x := 0; x < len(grid[y]); x++ {
			char := grid[y][x]
			if char == ' ' || isAlphaNum(char) {
				continue // Skip spaces and text
			}
			
			// Check adjacency rules for this character
			v.checkCharacter(grid, x, y, char)
		}
	}
	
	return v.errors
}

// checkCharacter validates a single character's adjacency rules.
func (v *LineValidator) checkCharacter(grid [][]rune, x, y int, char rune) {
	// Get adjacent characters
	north := v.getChar(grid, x, y-1)
	south := v.getChar(grid, x, y+1)
	east := v.getChar(grid, x+1, y)
	west := v.getChar(grid, x-1, y)
	
	// Check based on character type
	switch char {
	case '─', '-':
		// Horizontal line - must connect horizontally
		if !v.canConnectHorizontal(west) && west != ' ' && !isAlphaNum(west) {
			v.addError(x, y, char, fmt.Sprintf("west=%c", west), 
				"Horizontal line cannot connect to %c on the west", west)
		}
		if !v.canConnectHorizontal(east) && east != ' ' && !isAlphaNum(east) {
			v.addError(x, y, char, fmt.Sprintf("east=%c", east),
				"Horizontal line cannot connect to %c on the east", east)
		}
		// Should not connect vertically (except for junctions)
		if v.strictMode {
			if v.isVerticalOnly(north) {
				v.addError(x, y, char, fmt.Sprintf("north=%c", north),
					"Horizontal line should not have vertical line above")
			}
			if v.isVerticalOnly(south) {
				v.addError(x, y, char, fmt.Sprintf("south=%c", south),
					"Horizontal line should not have vertical line below")
			}
		}
		
	case '│', '|':
		// Vertical line - must connect vertically
		if !v.canConnectVertical(north) && north != ' ' && !isAlphaNum(north) {
			v.addError(x, y, char, fmt.Sprintf("north=%c", north),
				"Vertical line cannot connect to %c on the north", north)
		}
		if !v.canConnectVertical(south) && south != ' ' && !isAlphaNum(south) {
			v.addError(x, y, char, fmt.Sprintf("south=%c", south),
				"Vertical line cannot connect to %c on the south", south)
		}
		// Should not connect horizontally (except for junctions)
		if v.strictMode {
			if v.isHorizontalOnly(west) {
				v.addError(x, y, char, fmt.Sprintf("west=%c", west),
					"Vertical line should not have horizontal line to the west")
			}
			if v.isHorizontalOnly(east) {
				v.addError(x, y, char, fmt.Sprintf("east=%c", east),
					"Vertical line should not have horizontal line to the east")
			}
		}
		
	case '┌', '╭':
		// Top-left corner - connects right and down
		if !v.canConnectHorizontal(east) && east != ' ' && !isAlphaNum(east) {
			v.addError(x, y, char, fmt.Sprintf("east=%c", east),
				"Top-left corner must connect horizontally to the east")
		}
		if !v.canConnectVertical(south) && south != ' ' && !isAlphaNum(south) {
			v.addError(x, y, char, fmt.Sprintf("south=%c", south),
				"Top-left corner must connect vertically to the south")
		}
		
	case '┐', '╮':
		// Top-right corner - connects left and down
		if !v.canConnectHorizontal(west) && west != ' ' && !isAlphaNum(west) {
			v.addError(x, y, char, fmt.Sprintf("west=%c", west),
				"Top-right corner must connect horizontally to the west")
		}
		if !v.canConnectVertical(south) && south != ' ' && !isAlphaNum(south) {
			v.addError(x, y, char, fmt.Sprintf("south=%c", south),
				"Top-right corner must connect vertically to the south")
		}
		
	case '└', '╰':
		// Bottom-left corner - connects right and up
		if !v.canConnectHorizontal(east) && east != ' ' && !isAlphaNum(east) {
			v.addError(x, y, char, fmt.Sprintf("east=%c", east),
				"Bottom-left corner must connect horizontally to the east")
		}
		if !v.canConnectVertical(north) && north != ' ' && !isAlphaNum(north) {
			v.addError(x, y, char, fmt.Sprintf("north=%c", north),
				"Bottom-left corner must connect vertically to the north")
		}
		
	case '┘', '╯':
		// Bottom-right corner - connects left and up
		if !v.canConnectHorizontal(west) && west != ' ' && !isAlphaNum(west) {
			v.addError(x, y, char, fmt.Sprintf("west=%c", west),
				"Bottom-right corner must connect horizontally to the west")
		}
		if !v.canConnectVertical(north) && north != ' ' && !isAlphaNum(north) {
			v.addError(x, y, char, fmt.Sprintf("north=%c", north),
				"Bottom-right corner must connect vertically to the north")
		}
		
	case '├':
		// Tee right - connects in three directions (not west)
		if !v.canConnectVertical(north) && north != ' ' && !isAlphaNum(north) {
			v.addError(x, y, char, fmt.Sprintf("north=%c", north),
				"Tee-right must connect vertically to the north")
		}
		if !v.canConnectVertical(south) && south != ' ' && !isAlphaNum(south) {
			v.addError(x, y, char, fmt.Sprintf("south=%c", south),
				"Tee-right must connect vertically to the south")
		}
		if !v.canConnectHorizontal(east) && east != ' ' && !isAlphaNum(east) {
			v.addError(x, y, char, fmt.Sprintf("east=%c", east),
				"Tee-right must connect horizontally to the east")
		}
		
	case '┤':
		// Tee left - connects in three directions (not east)
		if !v.canConnectVertical(north) && north != ' ' && !isAlphaNum(north) {
			v.addError(x, y, char, fmt.Sprintf("north=%c", north),
				"Tee-left must connect vertically to the north")
		}
		if !v.canConnectVertical(south) && south != ' ' && !isAlphaNum(south) {
			v.addError(x, y, char, fmt.Sprintf("south=%c", south),
				"Tee-left must connect vertically to the south")
		}
		if !v.canConnectHorizontal(west) && west != ' ' && !isAlphaNum(west) {
			v.addError(x, y, char, fmt.Sprintf("west=%c", west),
				"Tee-left must connect horizontally to the west")
		}
		
	case '┬':
		// Tee down - connects in three directions (not north)
		if !v.canConnectHorizontal(west) && west != ' ' && !isAlphaNum(west) {
			v.addError(x, y, char, fmt.Sprintf("west=%c", west),
				"Tee-down must connect horizontally to the west")
		}
		if !v.canConnectHorizontal(east) && east != ' ' && !isAlphaNum(east) {
			v.addError(x, y, char, fmt.Sprintf("east=%c", east),
				"Tee-down must connect horizontally to the east")
		}
		if !v.canConnectVertical(south) && south != ' ' && !isAlphaNum(south) {
			v.addError(x, y, char, fmt.Sprintf("south=%c", south),
				"Tee-down must connect vertically to the south")
		}
		
	case '┴':
		// Tee up - connects in three directions (not south)
		if !v.canConnectHorizontal(west) && west != ' ' && !isAlphaNum(west) {
			v.addError(x, y, char, fmt.Sprintf("west=%c", west),
				"Tee-up must connect horizontally to the west")
		}
		if !v.canConnectHorizontal(east) && east != ' ' && !isAlphaNum(east) {
			v.addError(x, y, char, fmt.Sprintf("east=%c", east),
				"Tee-up must connect horizontally to the east")
		}
		if !v.canConnectVertical(north) && north != ' ' && !isAlphaNum(north) {
			v.addError(x, y, char, fmt.Sprintf("north=%c", north),
				"Tee-up must connect vertically to the north")
		}
		
	case '┼', '+':
		// Cross - connects in all four directions
		if !v.canConnectHorizontal(west) && west != ' ' && !isAlphaNum(west) {
			v.addError(x, y, char, fmt.Sprintf("west=%c", west),
				"Cross must connect horizontally to the west")
		}
		if !v.canConnectHorizontal(east) && east != ' ' && !isAlphaNum(east) {
			v.addError(x, y, char, fmt.Sprintf("east=%c", east),
				"Cross must connect horizontally to the east")
		}
		if !v.canConnectVertical(north) && north != ' ' && !isAlphaNum(north) {
			v.addError(x, y, char, fmt.Sprintf("north=%c", north),
				"Cross must connect vertically to the north")
		}
		if !v.canConnectVertical(south) && south != ' ' && !isAlphaNum(south) {
			v.addError(x, y, char, fmt.Sprintf("south=%c", south),
				"Cross must connect vertically to the south")
		}
		
	case '▶', '>', '▷':
		// Right arrow - should have horizontal line to the west
		if !v.canConnectHorizontal(west) && west != ' ' && !isAlphaNum(west) {
			v.addError(x, y, char, fmt.Sprintf("west=%c", west),
				"Right arrow should have horizontal line to the west")
		}
		
	case '◀', '<', '◁':
		// Left arrow - should have horizontal line to the east
		if !v.canConnectHorizontal(east) && east != ' ' && !isAlphaNum(east) {
			v.addError(x, y, char, fmt.Sprintf("east=%c", east),
				"Left arrow should have horizontal line to the east")
		}
		
	case '▲', '^', '△':
		// Up arrow - should have vertical line to the south
		if !v.canConnectVertical(south) && south != ' ' && !isAlphaNum(south) {
			v.addError(x, y, char, fmt.Sprintf("south=%c", south),
				"Up arrow should have vertical line to the south")
		}
		
	case '▼', 'v', 'V', '▽':
		// Down arrow - should have vertical line to the north
		if !v.canConnectVertical(north) && north != ' ' && !isAlphaNum(north) {
			v.addError(x, y, char, fmt.Sprintf("north=%c", north),
				"Down arrow should have vertical line to the north")
		}
	}
}

// canConnectHorizontal checks if a character can connect horizontally.
func (v *LineValidator) canConnectHorizontal(char rune) bool {
	switch char {
	case '─', '━', '—':
		return true
	case '-':
		return v.allowASCII
	case '├', '┤', '┬', '┴', '┼':
		return true
	case '╭', '╮', '╰', '╯':
		return true
	case '┌', '┐', '└', '┘':
		return true
	case '+':
		return v.allowASCII
	case '▶', '◀', '>', '<', '▷', '◁':
		return true
	default:
		return false
	}
}

// canConnectVertical checks if a character can connect vertically.
func (v *LineValidator) canConnectVertical(char rune) bool {
	switch char {
	case '│', '┃', '｜':
		return true
	case '|':
		return v.allowASCII
	case '├', '┤', '┬', '┴', '┼':
		return true
	case '╭', '╮', '╰', '╯':
		return true
	case '┌', '┐', '└', '┘':
		return true
	case '+':
		return v.allowASCII
	case '▲', '▼', '^', 'v', 'V', '△', '▽':
		return true
	default:
		return false
	}
}

// isHorizontalOnly checks if a character is purely horizontal (no vertical component).
func (v *LineValidator) isHorizontalOnly(char rune) bool {
	return (char == '─' || char == '-') && !v.canConnectVertical(char)
}

// isVerticalOnly checks if a character is purely vertical (no horizontal component).
func (v *LineValidator) isVerticalOnly(char rune) bool {
	return (char == '│' || char == '|') && !v.canConnectHorizontal(char)
}

// getChar safely gets a character from the grid.
func (v *LineValidator) getChar(grid [][]rune, x, y int) rune {
	if y < 0 || y >= len(grid) {
		return ' '
	}
	if x < 0 || x >= len(grid[y]) {
		return ' '
	}
	return grid[y][x]
}

// addError adds a validation error.
func (v *LineValidator) addError(x, y int, char rune, context, format string, args ...interface{}) {
	v.errors = append(v.errors, ValidationError{
		X:       x,
		Y:       y,
		Char:    char,
		Context: context,
		Message: fmt.Sprintf(format, args...),
	})
}

// isAlphaNum checks if a rune is alphanumeric.
func isAlphaNum(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

// String formats validation errors as a string.
func (e ValidationError) String() string {
	return fmt.Sprintf("(%d,%d) '%c' [%s]: %s", e.X, e.Y, e.Char, e.Context, e.Message)
}
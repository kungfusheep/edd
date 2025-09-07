package render

// JunctionResolver determines the appropriate junction character when lines intersect.
type JunctionResolver struct {
	// Maps existing character + new line type to junction character
	junctions map[junctionKey]rune
}

type junctionKey struct {
	existing rune
	newLine  rune
}

// NewJunctionResolver creates a new junction resolver with default mappings.
func NewJunctionResolver() *JunctionResolver {
	jr := &JunctionResolver{
		junctions: make(map[junctionKey]rune),
	}
	jr.initializeJunctions()
	return jr
}

// Resolve determines the appropriate character when a new line intersects an existing character.
func (jr *JunctionResolver) Resolve(existing, newLine rune) rune {
	// If the existing character is the same as the new line, no change needed
	if existing == newLine {
		return existing
	}
	
	// Don't override existing arrows unless we're specifically handling arrow junctions
	if isArrowChar(existing) && !isArrowChar(newLine) {
		return existing
	}
	
	// Special case: when two arrows point in the same direction, preserve the existing arrow
	if isArrowChar(existing) && isArrowChar(newLine) {
		if arrowsPointSameDirection(existing, newLine) {
			return existing
		}
	}
	
	
	// Special handling for corners being placed where lines exist
	// This improves aesthetics for simple paths
	if isSimpleLineChar(existing) && isCornerChar(newLine) {
		// Check if this creates a valid junction pattern
		if junction, ok := jr.junctions[junctionKey{existing, newLine}]; ok {
			// If we have a specific junction mapping, use it
			return junction
		}
		// Otherwise, prefer the corner for aesthetics
		return newLine
	}
	
	// Look up the appropriate junction
	if junction, ok := jr.junctions[junctionKey{existing, newLine}]; ok {
		return junction
	}
	
	// Special case: if new character is an arrow, it should take precedence
	if isArrowChar(newLine) {
		return newLine
	}
	
	// Default behavior: if we can't resolve, use a cross
	if isLineChar(existing) && isLineChar(newLine) && !isArrowChar(existing) && !isArrowChar(newLine) {
		return '┼' // or '+' for ASCII
	}
	
	// Otherwise, keep the existing character
	return existing
}

// initializeJunctions sets up the junction mappings.
func (jr *JunctionResolver) initializeJunctions() {
	// Unicode box-drawing junctions
	// Horizontal line meets vertical line
	jr.junctions[junctionKey{'─', '│'}] = '┼'
	jr.junctions[junctionKey{'│', '─'}] = '┼'
	
	// Horizontal line meets corners
	jr.junctions[junctionKey{'─', '┌'}] = '┬'
	jr.junctions[junctionKey{'─', '┐'}] = '┬'
	jr.junctions[junctionKey{'─', '└'}] = '┴'
	jr.junctions[junctionKey{'─', '┘'}] = '┴'
	
	// Corners meet horizontal line (reverse case)
	jr.junctions[junctionKey{'┌', '─'}] = '┬'
	jr.junctions[junctionKey{'┐', '─'}] = '┬'
	jr.junctions[junctionKey{'└', '─'}] = '┴'
	jr.junctions[junctionKey{'┘', '─'}] = '┴'
	
	// Vertical line meets corners
	jr.junctions[junctionKey{'│', '┌'}] = '├'
	jr.junctions[junctionKey{'│', '└'}] = '├'
	jr.junctions[junctionKey{'│', '┐'}] = '┤'
	jr.junctions[junctionKey{'│', '┘'}] = '┤'
	
	// Corners meet vertical line (reverse case)
	jr.junctions[junctionKey{'┌', '│'}] = '├'
	jr.junctions[junctionKey{'└', '│'}] = '├'
	jr.junctions[junctionKey{'┐', '│'}] = '┤'
	jr.junctions[junctionKey{'┘', '│'}] = '┤'
	
	// Horizontal line meets T-junctions
	jr.junctions[junctionKey{'─', '├'}] = '┼'
	jr.junctions[junctionKey{'─', '┤'}] = '┼'
	jr.junctions[junctionKey{'─', '┬'}] = '┬'
	jr.junctions[junctionKey{'─', '┴'}] = '┴'
	
	// Vertical line meets T-junctions
	jr.junctions[junctionKey{'│', '┬'}] = '┼'
	jr.junctions[junctionKey{'│', '┴'}] = '┼'
	jr.junctions[junctionKey{'│', '├'}] = '├'
	jr.junctions[junctionKey{'│', '┤'}] = '┤'
	
	// T-junctions meet lines (reverse cases)
	jr.junctions[junctionKey{'┬', '│'}] = '┼'
	jr.junctions[junctionKey{'┴', '│'}] = '┼'
	jr.junctions[junctionKey{'├', '│'}] = '├'  // vertical line already part of left T
	jr.junctions[junctionKey{'┤', '│'}] = '┤'  // vertical line already part of right T
	jr.junctions[junctionKey{'┬', '─'}] = '┬'  // horizontal line already part of top T
	jr.junctions[junctionKey{'┴', '─'}] = '┴'  // horizontal line already part of bottom T
	jr.junctions[junctionKey{'├', '─'}] = '┼'
	jr.junctions[junctionKey{'┤', '─'}] = '┼'
	
	// ASCII junctions (everything becomes +)
	jr.junctions[junctionKey{'-', '|'}] = '+'
	jr.junctions[junctionKey{'|', '-'}] = '+'
	jr.junctions[junctionKey{'-', '+'}] = '+'
	jr.junctions[junctionKey{'|', '+'}] = '+'
	jr.junctions[junctionKey{'+', '-'}] = '+'
	jr.junctions[junctionKey{'+', '|'}] = '+'
	
	// Corner to corner junctions
	// When two corners meet, we need to analyze which directions they cover
	// ┌ (top-left): right and down
	// ┐ (top-right): left and down  
	// └ (bottom-left): right and up
	// ┘ (bottom-right): left and up
	
	// Same corners = keep the corner
	jr.junctions[junctionKey{'┌', '┌'}] = '┌'
	jr.junctions[junctionKey{'┐', '┐'}] = '┐'
	jr.junctions[junctionKey{'└', '└'}] = '└'
	jr.junctions[junctionKey{'┘', '┘'}] = '┘'
	
	// Adjacent corners (share one direction) = T-junction
	jr.junctions[junctionKey{'┌', '┐'}] = '┬' // both have down, add left+right
	jr.junctions[junctionKey{'┐', '┌'}] = '┬' // both have down, add left+right
	jr.junctions[junctionKey{'└', '┘'}] = '┴' // both have up, add left+right
	jr.junctions[junctionKey{'┘', '└'}] = '┴' // both have up, add left+right
	jr.junctions[junctionKey{'┌', '└'}] = '├' // both have right, add up+down
	jr.junctions[junctionKey{'└', '┌'}] = '├' // both have right, add up+down
	jr.junctions[junctionKey{'┐', '┘'}] = '┤' // both have left, add up+down
	jr.junctions[junctionKey{'┘', '┐'}] = '┤' // both have left, add up+down
	
	// Opposite corners (no shared directions) = cross
	jr.junctions[junctionKey{'┌', '┘'}] = '┼'
	jr.junctions[junctionKey{'┘', '┌'}] = '┼'
	jr.junctions[junctionKey{'┐', '└'}] = '┼'
	jr.junctions[junctionKey{'└', '┐'}] = '┼'
	
	// Arrow meets line junctions (triangular arrows)
	// Right arrow meets vertical line
	jr.junctions[junctionKey{'│', '▶'}] = '├'  // Vertical line with arrow extending right
	jr.junctions[junctionKey{'▶', '│'}] = '├'  // Arrow exists, vertical line crosses it
	// Left arrow meets vertical line
	jr.junctions[junctionKey{'│', '◀'}] = '┤'  // Vertical line with arrow extending left
	jr.junctions[junctionKey{'◀', '│'}] = '┤'  // Arrow exists, vertical line crosses it
	// Down arrow meets horizontal line
	jr.junctions[junctionKey{'─', '▼'}] = '┬'
	jr.junctions[junctionKey{'▼', '─'}] = '┬'
	// Up arrow meets horizontal line
	jr.junctions[junctionKey{'─', '▲'}] = '┴'
	jr.junctions[junctionKey{'▲', '─'}] = '┴'
	
	// Arrow meets line junctions (traditional arrows)
	jr.junctions[junctionKey{'│', '→'}] = '├'
	jr.junctions[junctionKey{'→', '│'}] = '├'
	jr.junctions[junctionKey{'│', '←'}] = '┤'
	jr.junctions[junctionKey{'←', '│'}] = '┤'
	jr.junctions[junctionKey{'─', '↓'}] = '┬'
	jr.junctions[junctionKey{'↓', '─'}] = '┬'
	jr.junctions[junctionKey{'─', '↑'}] = '┴'
	jr.junctions[junctionKey{'↑', '─'}] = '┴'
	
	// ASCII arrow meets line junctions
	jr.junctions[junctionKey{'|', '>'}] = '+'
	jr.junctions[junctionKey{'>', '|'}] = '+'
	jr.junctions[junctionKey{'|', '<'}] = '+'
	jr.junctions[junctionKey{'<', '|'}] = '+'
	jr.junctions[junctionKey{'-', 'v'}] = '+'
	jr.junctions[junctionKey{'v', '-'}] = '+'
	jr.junctions[junctionKey{'-', '^'}] = '+'
	jr.junctions[junctionKey{'^', '-'}] = '+'
	
	// Arrow meets corner junctions (triangular arrows)
	// Right arrow meets corners
	jr.junctions[junctionKey{'┌', '▶'}] = '├'
	jr.junctions[junctionKey{'└', '▶'}] = '├'
	jr.junctions[junctionKey{'┐', '▶'}] = '┼'
	jr.junctions[junctionKey{'┘', '▶'}] = '┼'
	// Left arrow meets corners
	jr.junctions[junctionKey{'┌', '◀'}] = '┼'
	jr.junctions[junctionKey{'└', '◀'}] = '┼'
	jr.junctions[junctionKey{'┐', '◀'}] = '┤'
	jr.junctions[junctionKey{'┘', '◀'}] = '┤'
	// Down arrow meets corners
	jr.junctions[junctionKey{'┌', '▼'}] = '┬'
	jr.junctions[junctionKey{'┐', '▼'}] = '┬'
	jr.junctions[junctionKey{'└', '▼'}] = '┼'
	jr.junctions[junctionKey{'┘', '▼'}] = '┼'
	// Up arrow meets corners
	jr.junctions[junctionKey{'┌', '▲'}] = '┼'
	jr.junctions[junctionKey{'┐', '▲'}] = '┼'
	jr.junctions[junctionKey{'└', '▲'}] = '┴'
	jr.junctions[junctionKey{'┘', '▲'}] = '┴'
	
	// Arrow meets T-junction (triangular arrows)
	// Right arrow meets T-junctions
	jr.junctions[junctionKey{'┬', '▶'}] = '┼'
	jr.junctions[junctionKey{'┴', '▶'}] = '┼'
	jr.junctions[junctionKey{'├', '▶'}] = '├'  // already a left T
	jr.junctions[junctionKey{'┤', '▶'}] = '┼'
	// Left arrow meets T-junctions
	jr.junctions[junctionKey{'┬', '◀'}] = '┼'
	jr.junctions[junctionKey{'┴', '◀'}] = '┼'
	jr.junctions[junctionKey{'├', '◀'}] = '┼'
	jr.junctions[junctionKey{'┤', '◀'}] = '┤'  // already a right T
	// Down arrow meets T-junctions
	jr.junctions[junctionKey{'┬', '▼'}] = '┬'  // already a top T
	jr.junctions[junctionKey{'┴', '▼'}] = '┼'
	jr.junctions[junctionKey{'├', '▼'}] = '┼'
	jr.junctions[junctionKey{'┤', '▼'}] = '┼'
	// Up arrow meets T-junctions
	jr.junctions[junctionKey{'┬', '▲'}] = '┼'
	jr.junctions[junctionKey{'┴', '▲'}] = '┴'  // already a bottom T
	jr.junctions[junctionKey{'├', '▲'}] = '┼'
	jr.junctions[junctionKey{'┤', '▲'}] = '┼'
	
	// Arrow meets arrow (perpendicular arrows form crosses)
	// Horizontal arrows meet vertical arrows
	jr.junctions[junctionKey{'▶', '▼'}] = '┼'
	jr.junctions[junctionKey{'▶', '▲'}] = '┼'
	jr.junctions[junctionKey{'◀', '▼'}] = '┼'
	jr.junctions[junctionKey{'◀', '▲'}] = '┼'
	jr.junctions[junctionKey{'▼', '▶'}] = '┼'
	jr.junctions[junctionKey{'▼', '◀'}] = '┼'
	jr.junctions[junctionKey{'▲', '▶'}] = '┼'
	jr.junctions[junctionKey{'▲', '◀'}] = '┼'
}

// isLineChar checks if a character is a line drawing character.
func isLineChar(r rune) bool {
	// Unicode box-drawing characters
	if r >= '─' && r <= '╿' {
		return true
	}
	// ASCII line characters
	return r == '-' || r == '|' || r == '+'
}

// isArrowChar checks if a character is an arrow character.
func isArrowChar(r rune) bool {
	// Triangular arrows
	return r == '▲' || r == '▼' || r == '◀' || r == '▶' ||
		// Traditional arrows
		r == '↑' || r == '↓' || r == '←' || r == '→' ||
		// ASCII arrows
		r == '^' || r == 'v' || r == '<' || r == '>'
}

// IsJunctionChar checks if a character is a junction (T-junction or cross).
func IsJunctionChar(r rune) bool {
	// Unicode junctions
	return r == '┼' || r == '├' || r == '┤' || r == '┬' || r == '┴' ||
		// ASCII junction
		r == '+'
}

// isCornerChar checks if a character is a corner character.
func isCornerChar(r rune) bool {
	return r == '┌' || r == '┐' || r == '└' || r == '┘' ||
		// ASCII corners (+ can be a corner in ASCII mode)
		r == '+'
}

// isSimpleLineChar checks if a character is a simple horizontal or vertical line.
func isSimpleLineChar(r rune) bool {
	return r == '─' || r == '│' || r == '-' || r == '|'
}

// arrowsPointSameDirection checks if two arrow characters point in the same direction
func arrowsPointSameDirection(a, b rune) bool {
	// Right-pointing arrows
	if (a == '▶' || a == '→' || a == '>') && (b == '▶' || b == '→' || b == '>') {
		return true
	}
	// Left-pointing arrows
	if (a == '◀' || a == '←' || a == '<') && (b == '◀' || b == '←' || b == '<') {
		return true
	}
	// Up-pointing arrows
	if (a == '▲' || a == '↑' || a == '^') && (b == '▲' || b == '↑' || b == '^') {
		return true
	}
	// Down-pointing arrows
	if (a == '▼' || a == '↓' || a == 'v') && (b == '▼' || b == '↓' || b == 'v') {
		return true
	}
	return false
}
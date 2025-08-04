package canvas

// CharacterMerger handles the merging of two characters at the same position
type CharacterMerger struct {
	mergeMap map[mergePair]rune
}

type mergePair struct {
	existing rune
	new      rune
}

// NewCharacterMerger creates a merger with standard box-drawing merge rules
func NewCharacterMerger() *CharacterMerger {
	m := &CharacterMerger{
		mergeMap: make(map[mergePair]rune),
	}
	m.initializeMergeRules()
	return m
}

// Merge combines two characters according to box-drawing rules
func (m *CharacterMerger) Merge(existing, new rune) rune {
	// If empty, use the new character
	if existing == ' ' || existing == '\x00' {
		return new
	}
	
	// If same character, no change needed
	if existing == new {
		return existing
	}
	
	// Check the merge map
	if merged, ok := m.mergeMap[mergePair{existing, new}]; ok {
		return merged
	}
	
	// Check reverse order (merging should be commutative)
	if merged, ok := m.mergeMap[mergePair{new, existing}]; ok {
		return merged
	}
	
	// Default: keep existing character
	return existing
}

// initializeMergeRules sets up the character merge mappings
func (m *CharacterMerger) initializeMergeRules() {
	// Basic line intersections
	m.mergeMap[mergePair{'─', '│'}] = '┼'  // horizontal + vertical = cross
	m.mergeMap[mergePair{'│', '─'}] = '┼'
	
	// Corner + line = T-junction
	// Top-left corner
	m.mergeMap[mergePair{'┌', '─'}] = '┬'  // becomes top T
	m.mergeMap[mergePair{'┌', '│'}] = '├'  // becomes left T
	
	// Top-right corner  
	m.mergeMap[mergePair{'┐', '─'}] = '┬'  // becomes top T
	m.mergeMap[mergePair{'┐', '│'}] = '┤'  // becomes right T
	
	// Bottom-left corner
	m.mergeMap[mergePair{'└', '─'}] = '┴'  // becomes bottom T
	m.mergeMap[mergePair{'└', '│'}] = '├'  // becomes left T
	
	// Bottom-right corner
	m.mergeMap[mergePair{'┘', '─'}] = '┴'  // becomes bottom T
	m.mergeMap[mergePair{'┘', '│'}] = '┤'  // becomes right T
	
	// T-junction + line = cross
	m.mergeMap[mergePair{'┬', '│'}] = '┼'
	m.mergeMap[mergePair{'┴', '│'}] = '┼'
	m.mergeMap[mergePair{'├', '─'}] = '┼'
	m.mergeMap[mergePair{'┤', '─'}] = '┼'
	
	// Corner + corner combinations
	m.mergeMap[mergePair{'┌', '┘'}] = '┼'  // opposite corners = cross
	m.mergeMap[mergePair{'┐', '└'}] = '┼'
	m.mergeMap[mergePair{'┌', '┐'}] = '┬'  // adjacent corners = T
	m.mergeMap[mergePair{'└', '┘'}] = '┴'
	m.mergeMap[mergePair{'┌', '└'}] = '├'
	m.mergeMap[mergePair{'┐', '┘'}] = '┤'
	
	// ASCII fallbacks
	m.mergeMap[mergePair{'-', '|'}] = '+'
	m.mergeMap[mergePair{'|', '-'}] = '+'
	m.mergeMap[mergePair{'+', '-'}] = '+'
	m.mergeMap[mergePair{'+', '|'}] = '+'
}

// isHorizontalChar checks if a character is a horizontal line.
func isHorizontalChar(r rune) bool {
	switch r {
	case '─', '━', '-', '═':
		return true
	case '┌', '┐', '└', '┘', '├', '┤', '┬', '┴', '┼':
		return true
	case '╔', '╗', '╚', '╝', '╠', '╣', '╦', '╩', '╬':
		return true
	case '╭', '╮', '╰', '╯':
		return true
	case '+':
		return true
	default:
		return false
	}
}

// isVerticalChar checks if a character is a vertical line.
func isVerticalChar(r rune) bool {
	switch r {
	case '│', '┃', '|', '║':
		return true
	case '┌', '┐', '└', '┘', '├', '┤', '┬', '┴', '┼':
		return true
	case '╔', '╗', '╚', '╝', '╠', '╣', '╦', '╩', '╬':
		return true
	case '╭', '╮', '╰', '╯':
		return true
	case '+':
		return true
	default:
		return false
	}
}


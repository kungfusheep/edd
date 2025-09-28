package render

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
	
	// Shadow characters should always be overwritten by non-space characters
	// This allows connection lines to pass through shadows
	if isShadow(existing) && new != ' ' {
		return new
	}
	
	// Arrow preservation: arrows should never be overwritten
	if isArrow(existing) {
		return existing
	}
	if isArrow(new) {
		return new
	}
	
	// Check the merge map
	if merged, ok := m.mergeMap[mergePair{existing, new}]; ok {
		return merged
	}
	
	// Check reverse order (merging should be commutative)
	if merged, ok := m.mergeMap[mergePair{new, existing}]; ok {
		return merged
	}
	
	// Text should overwrite line-drawing characters
	if isText(new) && isLineDrawing(existing) {
		return new
	}
	
	// Default: keep existing character
	return existing
}

// isShadow checks if a character is a shadow character
func isShadow(r rune) bool {
	return r == '░' || r == '▒' || r == '▓'
}

// isArrow checks if a character is an arrow
func isArrow(r rune) bool {
	return r == '▶' || r == '◀' || r == '▲' || r == '▼' ||
	       r == '>' || r == '<' || r == '^' || r == 'v' ||
	       r == '→' || r == '←' || r == '↑' || r == '↓'
}

// isText checks if a character is regular text (letters, numbers, etc.)
func isText(r rune) bool {
	// Consider alphanumeric, spaces, and common punctuation as text
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || 
	       (r >= '0' && r <= '9') || r == ' ' || r == '.' || 
	       r == ',' || r == '!' || r == '?' || r == ':' || 
	       r == ';' || r == '(' || r == ')' || r == '[' || 
	       r == ']' || r == '{' || r == '}' || r == '\'' || 
	       r == '"' || r == '-' || r == '_' || r == '/' || 
	       r == '\\' || r == '@' || r == '#' || r == '$' || 
	       r == '%' || r == '&' || r == '*' || r == '+' || 
	       r == '=' || r == '~' || r == '`'
}

// isLineDrawing checks if a character is a line-drawing character
func isLineDrawing(r rune) bool {
	// Box drawing characters are in the range U+2500 to U+257F
	return (r >= '─' && r <= '╿') || r == '│' || r == '─' ||
	       r == '┌' || r == '┐' || r == '└' || r == '┘' ||
	       r == '├' || r == '┤' || r == '┬' || r == '┴' ||
	       r == '┼' || r == '╭' || r == '╮' || r == '╯' ||
	       r == '╰' || r == '┆' || r == '┊' || r == '╌' ||
	       r == '╎' || r == '·'
}

// initializeMergeRules sets up the character merge mappings
func (m *CharacterMerger) initializeMergeRules() {
	// Basic line intersections
	m.mergeMap[mergePair{'─', '│'}] = '┼'  // horizontal + vertical = cross
	m.mergeMap[mergePair{'│', '─'}] = '┼'
	
	// For sequence diagrams: prefer branches when arrows meet lifelines
	// These will be handled by explicit branch placement if needed
	
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
	
	// T-junction + corner combinations (for paths that turn after branching)
	m.mergeMap[mergePair{'┴', '┐'}] = '┼'  // bottom T + top-right corner = cross
	m.mergeMap[mergePair{'┴', '┌'}] = '┼'  // bottom T + top-left corner = cross
	m.mergeMap[mergePair{'┬', '┘'}] = '┼'  // top T + bottom-right corner = cross
	m.mergeMap[mergePair{'┬', '└'}] = '┼'  // top T + bottom-left corner = cross
	m.mergeMap[mergePair{'├', '┐'}] = '┼'  // left T + top-right corner = cross
	m.mergeMap[mergePair{'├', '┘'}] = '┼'  // left T + bottom-right corner = cross
	m.mergeMap[mergePair{'┤', '┌'}] = '┼'  // right T + top-left corner = cross
	m.mergeMap[mergePair{'┤', '└'}] = '┼'  // right T + bottom-left corner = cross
	
	// Line + branch = branch (for when we explicitly want to place a branch)
	m.mergeMap[mergePair{'│', '├'}] = '├'  // vertical + left branch = left branch
	m.mergeMap[mergePair{'│', '┤'}] = '┤'  // vertical + right branch = right branch
	m.mergeMap[mergePair{'─', '┬'}] = '┬'  // horizontal + top branch = top branch
	m.mergeMap[mergePair{'─', '┴'}] = '┴'  // horizontal + bottom branch = bottom branch
	
	// Rounded corners - treat similar to regular corners
	// When a rounded corner is placed on a line, it should remain as the corner
	m.mergeMap[mergePair{'─', '╮'}] = '╮'  // horizontal + rounded top-right = keep corner
	m.mergeMap[mergePair{'─', '╭'}] = '╭'  // horizontal + rounded top-left = keep corner
	m.mergeMap[mergePair{'─', '╯'}] = '╯'  // horizontal + rounded bottom-right = keep corner
	m.mergeMap[mergePair{'─', '╰'}] = '╰'  // horizontal + rounded bottom-left = keep corner
	m.mergeMap[mergePair{'│', '╮'}] = '╮'  // vertical + rounded top-right = keep corner
	m.mergeMap[mergePair{'│', '╭'}] = '╭'  // vertical + rounded top-left = keep corner
	m.mergeMap[mergePair{'│', '╯'}] = '╯'  // vertical + rounded bottom-right = keep corner
	m.mergeMap[mergePair{'│', '╰'}] = '╰'  // vertical + rounded bottom-left = keep corner
	
	// Rounded corners replacing crosses (for DrawSmartPath)
	m.mergeMap[mergePair{'┼', '╮'}] = '╮'  // cross + rounded corner = keep corner
	m.mergeMap[mergePair{'┼', '╭'}] = '╭'
	m.mergeMap[mergePair{'┼', '╯'}] = '╯'
	m.mergeMap[mergePair{'┼', '╰'}] = '╰'
	
	// ASCII fallbacks
	m.mergeMap[mergePair{'-', '|'}] = '+'
	m.mergeMap[mergePair{'|', '-'}] = '+'
	m.mergeMap[mergePair{'+', '-'}] = '+'
	m.mergeMap[mergePair{'+', '|'}] = '+'
}


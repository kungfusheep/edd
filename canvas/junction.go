package canvas

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

// hasConnection checks if a character has a connection in the given direction.
func hasConnection(r rune, dir rune) bool {
	switch dir {
	case 'N': // North
		switch r {
		case '│', '┃', '|', '║':
			return true
		case '┌', '┐', '├', '┤', '┬', '┼':
			return true
		case '╔', '╗', '╠', '╣', '╦', '╬':
			return true
		case '╭', '╮':
			return true
		case '+':
			return true
		}
	case 'S': // South
		switch r {
		case '│', '┃', '|', '║':
			return true
		case '└', '┘', '├', '┤', '┴', '┼':
			return true
		case '╚', '╝', '╠', '╣', '╩', '╬':
			return true
		case '╰', '╯':
			return true
		case '+':
			return true
		}
	case 'E': // East
		switch r {
		case '─', '━', '-', '═':
			return true
		case '└', '┌', '├', '┬', '┴', '┼':
			return true
		case '╚', '╔', '╠', '╦', '╩', '╬':
			return true
		case '╰', '╭':
			return true
		case '+':
			return true
		}
	case 'W': // West
		switch r {
		case '─', '━', '-', '═':
			return true
		case '┘', '┐', '┤', '┬', '┴', '┼':
			return true
		case '╝', '╗', '╣', '╦', '╩', '╬':
			return true
		case '╯', '╮':
			return true
		case '+':
			return true
		}
	}
	return false
}

// resolveJunction determines the appropriate junction character based on connections.
func resolveJunction(north, south, east, west bool) rune {
	// All four directions
	if north && south && east && west {
		return '┼'
	}
	
	// Three directions
	if north && south && east {
		return '├'
	}
	if north && south && west {
		return '┤'
	}
	if north && east && west {
		return '┴'
	}
	if south && east && west {
		return '┬'
	}
	
	// Corners (two directions)
	if north && east {
		return '└'
	}
	if north && west {
		return '┘'
	}
	if south && east {
		return '┌'
	}
	if south && west {
		return '┐'
	}
	
	// Straight lines
	if north && south {
		return '│'
	}
	if east && west {
		return '─'
	}
	
	// Single direction or none - shouldn't happen in junction context
	return ' '
}

// resolveJunctionAt calculates the appropriate junction character at a position.
// The isDrawingVertical parameter indicates if we're currently drawing a vertical line.
func (c *MatrixCanvas) resolveJunctionAt(x, y int, isDrawingVertical bool) rune {
	// Check connections in all four directions
	hasNorth := false
	hasSouth := false
	hasEast := false
	hasWest := false
	
	// If we're drawing a vertical line, we have north/south connections
	if isDrawingVertical {
		hasNorth = true
		hasSouth = true
	} else {
		// Drawing horizontal line, we have east/west connections
		hasEast = true
		hasWest = true
	}
	
	// Check existing connections
	// North
	if y > 0 {
		char := c.matrix[y-1][x]
		if hasConnection(char, 'S') { // Character above must connect south
			hasNorth = true
		}
	}
	
	// South
	if y < c.height-1 {
		char := c.matrix[y+1][x]
		if hasConnection(char, 'N') { // Character below must connect north
			hasSouth = true
		}
	}
	
	// East
	if x < c.width-1 {
		char := c.matrix[y][x+1]
		if hasConnection(char, 'W') { // Character to right must connect west
			hasEast = true
		}
	}
	
	// West
	if x > 0 {
		char := c.matrix[y][x-1]
		if hasConnection(char, 'E') { // Character to left must connect east
			hasWest = true
		}
	}
	
	return resolveJunction(hasNorth, hasSouth, hasEast, hasWest)
}
package canvas

import (
	"edd/core"
	"errors"
	"fmt"
	"strings"
)

// Common errors
var (
	ErrOutOfBounds = errors.New("position out of bounds")
	ErrInvalidSize = errors.New("invalid canvas size")
	ErrNilCanvas   = errors.New("canvas is nil")
)

// MatrixCanvas implements a rune matrix-based canvas with high-level drawing primitives.
//
// Thread Safety:
// MatrixCanvas is NOT thread-safe for writes. All write operations (Set, Draw*, Clear)
// must be synchronized externally if used from multiple goroutines. Read operations
// (Get, Size, Matrix, String) are safe for concurrent access as long as no writes
// are happening simultaneously.
//
// Common synchronization patterns:
//
//	// Using a mutex:
//	var mu sync.Mutex
//	mu.Lock()
//	canvas.DrawBox(10, 10, 20, 20, DefaultBoxStyle)
//	mu.Unlock()
//
//	// Using channels for serialization:
//	type canvasOp func(*MatrixCanvas)
//	ops := make(chan canvasOp)
//	go func() {
//	    for op := range ops {
//	        op(canvas)
//	    }
//	}()
//
// Performance Characteristics:
//   - Set/Get: O(1)
//   - DrawBox: O(width + height)
//   - DrawLine: O(max(|x2-x1|, |y2-y1|))
//   - String: O(width × height)
//   - Clear: O(width × height)
//
// Coordinate System:
//   - Origin (0,0) is top-left
//   - X increases rightward
//   - Y increases downward
//   - All coordinates are in character cells
//
// Unicode Support:
//   - Full support for Unicode box-drawing characters
//   - Proper handling of wide characters (CJK, emoji)
//   - Zero-width character support (combining marks)
//   - Automatic junction resolution at line intersections
type MatrixCanvas struct {
	matrix [][]rune
	width  int
	height int
	merger *CharacterMerger
}

// NewMatrixCanvas creates a new canvas with the specified dimensions.
func NewMatrixCanvas(width, height int) *MatrixCanvas {
	if width <= 0 || height <= 0 {
		return nil
	}
	
	// Initialize matrix
	matrix := make([][]rune, height)
	for y := 0; y < height; y++ {
		matrix[y] = make([]rune, width)
		for x := 0; x < width; x++ {
			matrix[y][x] = ' '
		}
	}
	
	return &MatrixCanvas{
		matrix: matrix,
		width:  width,
		height: height,
		merger: NewCharacterMerger(),
	}
}

// Size returns the width and height of the canvas.
func (c *MatrixCanvas) Size() (width, height int) {
	return c.width, c.height
}

// Matrix returns direct access to the underlying rune matrix.
// This allows tools like LineValidator to work directly with the matrix.
func (c *MatrixCanvas) Matrix() [][]rune {
	return c.matrix
}

// Get returns the character at the given position.
// Returns ' ' (space) if position is out of bounds.
func (c *MatrixCanvas) Get(p core.Point) rune {
	if p.X < 0 || p.X >= c.width || p.Y < 0 || p.Y >= c.height {
		return ' '
	}
	return c.matrix[p.Y][p.X]
}

// Set places a character at the given position.
// Returns error if position is out of bounds.
// Uses the character merger to properly handle box-drawing character intersections.
func (c *MatrixCanvas) Set(p core.Point, char rune) error {
	if p.X < 0 || p.X >= c.width || p.Y < 0 || p.Y >= c.height {
		return ErrOutOfBounds
	}
	existing := c.matrix[p.Y][p.X]
	merged := c.merger.Merge(existing, char)
	
	// Debug merging
	// if existing != ' ' && existing != merged {
	//     fmt.Printf("Merge at (%d,%d): %c + %c = %c\n", p.X, p.Y, existing, char, merged)
	// }
	
	c.matrix[p.Y][p.X] = merged
	return nil
}

// Clear resets the canvas to all spaces.
func (c *MatrixCanvas) Clear() {
	for y := 0; y < c.height; y++ {
		for x := 0; x < c.width; x++ {
			c.matrix[y][x] = ' '
		}
	}
}

// String returns the canvas as a string with newlines.
func (c *MatrixCanvas) String() string {
	// Pre-calculate capacity for efficiency
	capacity := c.height * (c.width + 1) // +1 for newlines
	var sb strings.Builder
	sb.Grow(capacity)
	
	for y := 0; y < c.height; y++ {
		// Process each character, converting null bytes to spaces
		for x := 0; x < c.width; x++ {
			r := c.matrix[y][x]
			if r == '\x00' {
				// Wide character continuation - render as space
				sb.WriteRune(' ')
			} else {
				sb.WriteRune(r)
			}
		}
		if y < c.height-1 {
			sb.WriteRune('\n')
		}
	}
	
	return sb.String()
}

// DrawBox draws a rectangle with the specified style.
func (c *MatrixCanvas) DrawBox(x, y, width, height int, style BoxStyle) error {
	// Validate bounds
	if x < 0 || y < 0 || width <= 0 || height <= 0 {
		return fmt.Errorf("invalid box dimensions")
	}
	if x+width > c.width || y+height > c.height {
		return fmt.Errorf("box exceeds canvas bounds")
	}
	
	// Top line
	c.Set(core.Point{x, y}, style.TopLeft)
	for i := 1; i < width-1; i++ {
		c.Set(core.Point{x + i, y}, style.Horizontal)
	}
	c.Set(core.Point{x + width - 1, y}, style.TopRight)
	
	// Vertical lines
	for i := 1; i < height-1; i++ {
		c.Set(core.Point{x, y + i}, style.Vertical)
		c.Set(core.Point{x + width - 1, y + i}, style.Vertical)
	}
	
	// Bottom line
	c.Set(core.Point{x, y + height - 1}, style.BottomLeft)
	for i := 1; i < width-1; i++ {
		c.Set(core.Point{x + i, y + height - 1}, style.Horizontal)
	}
	c.Set(core.Point{x + width - 1, y + height - 1}, style.BottomRight)
	
	return nil
}

// DrawHorizontalLine draws a horizontal line.
func (c *MatrixCanvas) DrawHorizontalLine(x1, y, x2 int, char rune) error {
	if y < 0 || y >= c.height {
		return ErrOutOfBounds
	}
	
	// Ensure x1 <= x2
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	
	// Clip to canvas bounds
	if x1 < 0 {
		x1 = 0
	}
	if x2 >= c.width {
		x2 = c.width - 1
	}
	
	for x := x1; x <= x2; x++ {
		existing := c.matrix[y][x]
		c.matrix[y][x] = c.merger.Merge(existing, char)
	}
	
	return nil
}

// DrawVerticalLine draws a vertical line.
func (c *MatrixCanvas) DrawVerticalLine(x, y1, y2 int, char rune) error {
	if x < 0 || x >= c.width {
		return ErrOutOfBounds
	}
	
	// Ensure y1 <= y2
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	
	// Clip to canvas bounds
	if y1 < 0 {
		y1 = 0
	}
	if y2 >= c.height {
		y2 = c.height - 1
	}
	
	for y := y1; y <= y2; y++ {
		existing := c.matrix[y][x]
		c.matrix[y][x] = c.merger.Merge(existing, char)
	}
	
	return nil
}

// DrawLine draws a line between two points using Bresenham's algorithm.
func (c *MatrixCanvas) DrawLine(p1, p2 core.Point, char rune) error {
	// Bresenham's line algorithm
	dx := abs(p2.X - p1.X)
	dy := abs(p2.Y - p1.Y)
	
	x, y := p1.X, p1.Y
	
	xInc := 1
	if p1.X > p2.X {
		xInc = -1
	}
	
	yInc := 1
	if p1.Y > p2.Y {
		yInc = -1
	}
	
	// Draw the line
	if dx > dy {
		err := dx / 2
		for x != p2.X {
			c.setClipped(x, y, char)
			err -= dy
			if err < 0 {
				y += yInc
				err += dx
			}
			x += xInc
		}
	} else {
		err := dy / 2
		for y != p2.Y {
			c.setClipped(x, y, char)
			err -= dx
			if err < 0 {
				x += xInc
				err += dy
			}
			y += yInc
		}
	}
	
	// Draw the final point
	c.setClipped(p2.X, p2.Y, char)
	
	return nil
}

// DrawText renders text at the specified position.
func (c *MatrixCanvas) DrawText(x, y int, text string) error {
	if y < 0 || y >= c.height {
		return ErrOutOfBounds
	}
	
	// Track current x position for wide characters
	currentX := x
	
	for _, r := range text {
		width := UnicodeWidth(r)
		
		// Skip zero-width characters
		if width == 0 {
			continue
		}
		
		// Check if character fits completely
		if width == 2 && currentX >= 0 && currentX+1 >= c.width {
			// Wide character doesn't fully fit, skip it
			break
		}
		
		// Place the character if within bounds
		if currentX >= 0 && currentX < c.width {
			c.matrix[y][currentX] = r
			
			// For wide characters, mark the next cell
			if width == 2 && currentX+1 < c.width {
				c.matrix[y][currentX+1] = '\x00' // Null byte marks continuation
			}
		}
		
		currentX += width
		
		// Stop if we've gone past the canvas
		if currentX >= c.width {
			break
		}
	}
	
	return nil
}


// DrawSmartPath draws a path with automatic corner selection.
func (c *MatrixCanvas) DrawSmartPath(points []core.Point) error {
	if len(points) < 2 {
		return fmt.Errorf("path must have at least 2 points")
	}
	
	for i := 0; i < len(points)-1; i++ {
		p1 := points[i]
		p2 := points[i+1]
		
		// Draw segments
		if p1.Y == p2.Y {
			// Horizontal segment
			c.DrawHorizontalLine(p1.X, p1.Y, p2.X, '─')
		} else if p1.X == p2.X {
			// Vertical segment
			c.DrawVerticalLine(p1.X, p1.Y, p2.Y, '│')
		} else {
			// Diagonal - not supported in smart path, draw as line
			c.DrawLine(p1, p2, '*')
		}
		
		// Add corners at joints (except first and last)
		if i > 0 && i < len(points)-1 {
			prev := points[i-1]
			curr := points[i]
			next := points[i+1]
			
			corner := c.selectCorner(prev, curr, next)
			c.Set(curr, corner)
		}
	}
	
	return nil
}

// selectCorner chooses the appropriate corner character based on direction.
func (c *MatrixCanvas) selectCorner(prev, curr, next core.Point) rune {
	fromDir := getDirection(prev, curr)
	toDir := getDirection(curr, next)
	
	// Select corner based on from and to directions
	switch {
	case fromDir == 'E' && toDir == 'S', fromDir == 'N' && toDir == 'W':
		return '╮'
	case fromDir == 'E' && toDir == 'N', fromDir == 'S' && toDir == 'W':
		return '╯'
	case fromDir == 'W' && toDir == 'S', fromDir == 'N' && toDir == 'E':
		return '╭'
	case fromDir == 'W' && toDir == 'N', fromDir == 'S' && toDir == 'E':
		return '╰'
	default:
		// Junction or straight line
		if fromDir == toDir {
			if fromDir == 'E' || fromDir == 'W' {
				return '─'
			}
			return '│'
		}
		return '┼' // Cross for complex cases
	}
}

// getDirection returns the direction from p1 to p2.
func getDirection(p1, p2 core.Point) rune {
	if p2.X > p1.X {
		return 'E'
	} else if p2.X < p1.X {
		return 'W'
	} else if p2.Y > p1.Y {
		return 'S'
	} else {
		return 'N'
	}
}

// setClipped sets a character with bounds checking (no error).
func (c *MatrixCanvas) setClipped(x, y int, char rune) {
	if x >= 0 && x < c.width && y >= 0 && y < c.height {
		c.matrix[y][x] = char
	}
}

// abs returns the absolute value of an integer.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
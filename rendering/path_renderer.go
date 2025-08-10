package rendering

import (
	"edd/canvas"
	"edd/core"
	"fmt"
)

// PathRenderMode controls how paths are rendered, particularly junction behavior.
type PathRenderMode int

const (
	// RenderModeStandard creates junctions when lines cross (default).
	RenderModeStandard PathRenderMode = iota
	// RenderModePreserveCorners keeps corner characters intact when possible.
	RenderModePreserveCorners
)

// PathRenderer renders paths with appropriate line characters based on terminal capabilities.
type PathRenderer struct {
	caps       TerminalCapabilities
	style      LineStyle
	junction   *JunctionResolver
	renderMode PathRenderMode
}

// LineStyle represents the visual style for rendering lines.
type LineStyle struct {
	// Basic line characters
	Horizontal rune
	Vertical   rune
	
	// Corner characters
	TopLeft     rune
	TopRight    rune
	BottomLeft  rune
	BottomRight rune
	
	// Junction characters
	Cross        rune // ┼
	TeeUp        rune // ┴
	TeeDown      rune // ┬
	TeeLeft      rune // ┤
	TeeRight     rune // ├
	
	// Arrow characters
	ArrowUp    rune
	ArrowDown  rune
	ArrowLeft  rune
	ArrowRight rune
}

// NewPathRenderer creates a new path renderer with the given capabilities.
func NewPathRenderer(caps TerminalCapabilities) *PathRenderer {
	return &PathRenderer{
		caps:       caps,
		style:      selectLineStyle(caps),
		junction:   NewJunctionResolver(),
		renderMode: RenderModeStandard, // Default to standard behavior
	}
}

// SetRenderMode changes how paths are rendered.
func (r *PathRenderer) SetRenderMode(mode PathRenderMode) {
	r.renderMode = mode
}

// RenderPath draws a path on the canvas with appropriate line characters.
func (r *PathRenderer) RenderPath(canvas canvas.Canvas, path core.Path, hasArrow bool) error {
	return r.RenderPathWithOptions(canvas, path, hasArrow, false)
}

// RenderPathWithOptions draws a path with additional rendering options.
func (r *PathRenderer) RenderPathWithOptions(canvas canvas.Canvas, path core.Path, hasArrow bool, isConnection bool) error {
	if path.IsEmpty() {
		return nil
	}
	
	points := path.Points
	
	// Handle single point
	if len(points) == 1 {
		if hasArrow {
			return canvas.Set(points[0], '•')
		}
		return nil
	}
	
	// Check if this is a closed path (first and last points are the same)
	isClosed := len(points) > 2 && points[0] == points[len(points)-1]
	if isClosed {
		// Remove the duplicate last point to avoid rendering issues
		points = points[:len(points)-1]
	}
	
	// Phase 1: Identify all corner positions
	corners := r.identifyCorners(points, isClosed)
	
	// Phase 2: Draw all segments, skipping corner positions
	for i := 0; i < len(points)-1; i++ {
		from := points[i]
		to := points[i+1]
		
		// For the last segment, check if we need an arrow
		isLastSegment := (i == len(points)-2)
		drawArrowOnSegment := isLastSegment && hasArrow && !isClosed
		
		// For the first segment of a connection, check if we need a corner character
		if i == 0 && isConnection && !isClosed {
			existing := canvas.Get(from)
			// Only place corner if we're starting from a clean box edge (not a junction or branch)
			if existing == '│' || existing == '─' {
				// Place appropriate corner character based on direction
				dx := to.X - from.X
				dy := to.Y - from.Y
				
				var cornerChar rune
				if existing == '│' && dy == 0 {
					// Horizontal movement from vertical edge
					if dx > 0 {
						cornerChar = r.style.BottomLeft  // └ merges with │ to make ├
					} else {
						cornerChar = r.style.BottomRight // ┘ merges with │ to make ┤
					}
				} else if existing == '─' && dx == 0 {
					// Vertical movement from horizontal edge
					if dy > 0 {
						cornerChar = r.style.TopLeft     // ┌ points down, merges with ─ to make ┬
					} else {
						cornerChar = r.style.BottomLeft  // └ points up, merges with ─ to make ┴
					}
				}
				
				if cornerChar != 0 {
					canvas.Set(from, cornerChar)
					// Skip first point when drawing to preserve corner
					if err := r.drawSegmentSkippingCornersWithOptions(canvas, from, to, corners, drawArrowOnSegment, true); err != nil {
						return err
					}
				} else {
					// No corner needed, draw normally
					if err := r.drawSegmentSkippingCorners(canvas, from, to, corners, drawArrowOnSegment); err != nil {
						return err
					}
				}
			} else {
				// Not a clean box edge, draw normally
				if err := r.drawSegmentSkippingCorners(canvas, from, to, corners, drawArrowOnSegment); err != nil {
					return err
				}
			}
		} else {
			// Not first segment, draw normally
			if err := r.drawSegmentSkippingCorners(canvas, from, to, corners, drawArrowOnSegment); err != nil {
				return err
			}
		}
	}
	
	// For closed paths, draw the closing segment
	if isClosed && len(points) > 2 {
		from := points[len(points)-1]
		to := points[0]
		if err := r.drawSegmentSkippingCorners(canvas, from, to, corners, false); err != nil {
			return err
		}
	}
	
	// Phase 3: Place all corners
	for pos, corner := range corners {
		canvas.Set(pos, corner)
	}
	
	return nil
}

// placeStartBranch places a corner character at the start of a connection
// This corner will merge with the box edge (─ or │) to create a branch character (├, ┤, ┬, ┴)
func (r *PathRenderer) placeStartBranch(canvas canvas.Canvas, from, to core.Point) {
	dx := to.X - from.X
	dy := to.Y - from.Y
	
	// Determine which character to use based on direction and what's already there
	existing := canvas.Get(from)
	
	// Don't place anything if:
	// - It's already a cross
	// - It's already a branch character
	// - It's a line in the same direction we're going (horizontal line when moving horizontally)
	if existing == '┼' || existing == '├' || existing == '┤' || existing == '┬' || existing == '┴' {
		return
	}
	
	// Don't place a corner if we're continuing in the same direction as existing line
	if (dy == 0 && existing == '─') || (dx == 0 && existing == '│') {
		// We're going in the same direction as the existing line, no branch needed
		return
	}
	
	var cornerChar rune
	
	// Only place corner characters when connecting to perpendicular box edges
	if dy == 0 && existing == '│' {
		// Moving horizontally from a vertical edge (box side)
		if dx > 0 {
			// Moving right from left edge - use └ to merge with │ to get ├
			cornerChar = r.style.BottomLeft
		} else {
			// Moving left from right edge - use ┘ to merge with │ to get ┤  
			cornerChar = r.style.BottomRight
		}
	} else if dx == 0 && existing == '─' {
		// Moving vertically from a horizontal edge (box top/bottom)
		if dy > 0 {
			// Moving down from top edge - use ┌ to merge with ─ to get ┬
			cornerChar = r.style.TopLeft
		} else {
			// Moving up from bottom edge - use └ to merge with ─ to get ┴
			cornerChar = r.style.BottomLeft
		}
	}
	// If existing is space or something else, don't place anything
	// The normal line drawing will handle it
	
	if cornerChar != 0 {
		canvas.Set(from, cornerChar)
	}
}

// drawSegmentSkippingCorners draws a line segment while skipping any positions marked as corners
// If skipFirst is true, skip drawing at the first point (used for connection starts)
func (r *PathRenderer) drawSegmentSkippingCorners(canvas canvas.Canvas, from, to core.Point, corners map[core.Point]rune, drawArrow bool) error {
	return r.drawSegmentSkippingCornersWithOptions(canvas, from, to, corners, drawArrow, false)
}

func (r *PathRenderer) drawSegmentSkippingCornersWithOptions(canvas canvas.Canvas, from, to core.Point, corners map[core.Point]rune, drawArrow bool, skipFirst bool) error {
	dx := to.X - from.X
	dy := to.Y - from.Y
	
	// Horizontal line
	if dy == 0 {
		step := 1
		if dx < 0 {
			step = -1
		}
		
		for x := from.X; x != to.X+step; x += step {
			p := core.Point{X: x, Y: from.Y}
			
			// Skip the first point if requested
			if skipFirst && x == from.X {
				continue
			}
			
			// Skip if this is a corner position
			if _, isCorner := corners[p]; isCorner {
				continue
			}
			
			// Handle endpoint with arrow
			if x == to.X && drawArrow {
				arrowChar := r.style.ArrowRight
				if step < 0 {
					arrowChar = r.style.ArrowLeft
				}
				// Debug: print when placing arrows
				//fmt.Printf("Placing arrow %c at (%d,%d)\n", arrowChar, p.X, p.Y)
				canvas.Set(p, arrowChar)
			} else {
				canvas.Set(p, r.style.Horizontal)
			}
		}
		return nil
	}
	
	// Vertical line
	if dx == 0 {
		step := 1
		if dy < 0 {
			step = -1
		}
		
		for y := from.Y; y != to.Y+step; y += step {
			p := core.Point{X: from.X, Y: y}
			
			// Skip the first point if requested
			if skipFirst && y == from.Y {
				continue
			}
			
			// Skip if this is a corner position
			if _, isCorner := corners[p]; isCorner {
				continue
			}
			
			// Handle endpoint with arrow
			if y == to.Y && drawArrow {
				arrowChar := r.style.ArrowDown
				if step < 0 {
					arrowChar = r.style.ArrowUp
				}
				canvas.Set(p, arrowChar)
			} else {
				canvas.Set(p, r.style.Vertical)
			}
		}
		return nil
	}
	
	// Diagonal lines not supported in terminal rendering
	return fmt.Errorf("diagonal lines not supported: from (%d,%d) to (%d,%d)", from.X, from.Y, to.X, to.Y)
}

// drawSegmentInclusive draws a line segment including the endpoint.
// For multi-segment paths, we skip the start point if it's a potential corner location.
func (r *PathRenderer) drawSegmentInclusive(canvas canvas.Canvas, from, to core.Point, drawArrow bool) error {
	dx := to.X - from.X
	dy := to.Y - from.Y
	
	// Horizontal line
	if dy == 0 {
		step := 1
		if dx < 0 {
			step = -1
		}
		
		for x := from.X; ; x += step {
			p := core.Point{X: x, Y: from.Y}
			
			// If this is the endpoint and we need to draw an arrow, draw only the arrow
			if x == to.X {
				if drawArrow {
					arrowChar := r.style.ArrowRight
					if step < 0 {
						arrowChar = r.style.ArrowLeft
					}
					// Check for existing content and resolve junctions if needed
					if existing := canvas.Get(p); existing != ' ' && existing != 0 {
						if junction := r.junction.Resolve(existing, arrowChar); junction != 0 {
							canvas.Set(p, junction)
						} else {
							canvas.Set(p, arrowChar)
						}
					} else {
						canvas.Set(p, arrowChar)
					}
				} else {
					// No arrow - draw the line segment
					if existing := canvas.Get(p); existing != ' ' && existing != 0 {
						if junction := r.junction.Resolve(existing, r.style.Horizontal); junction != 0 {
							canvas.Set(p, junction)
						}
					} else {
					canvas.Set(p, r.style.Horizontal)
					}
				}
				break
			} else {
				// Not the endpoint - draw the line
				if existing := canvas.Get(p); existing != ' ' && existing != 0 {
					if junction := r.junction.Resolve(existing, r.style.Horizontal); junction != 0 {
						canvas.Set(p, junction)
					}
				} else {
					canvas.Set(p, r.style.Horizontal)
				}
			}
		}
	}
	
	// Vertical line
	if dx == 0 {
		step := 1
		if dy < 0 {
			step = -1
		}
		
		for y := from.Y; ; y += step {
			p := core.Point{X: from.X, Y: y}
			
			// If this is the endpoint and we need to draw an arrow, draw only the arrow
			if y == to.Y {
				if drawArrow {
					arrowChar := r.style.ArrowDown
					if step < 0 {
						arrowChar = r.style.ArrowUp
					}
					// Check for existing content and resolve junctions if needed
					if existing := canvas.Get(p); existing != ' ' && existing != 0 {
						if junction := r.junction.Resolve(existing, arrowChar); junction != 0 {
							canvas.Set(p, junction)
						} else {
							canvas.Set(p, arrowChar)
						}
					} else {
						canvas.Set(p, arrowChar)
					}
				} else {
					// No arrow - draw the line segment
					if existing := canvas.Get(p); existing != ' ' && existing != 0 {
						if junction := r.junction.Resolve(existing, r.style.Vertical); junction != 0 {
							canvas.Set(p, junction)
						}
					} else {
					canvas.Set(p, r.style.Vertical)
					}
				}
				break
			} else {
				// Not the endpoint - draw the line
				if existing := canvas.Get(p); existing != ' ' && existing != 0 {
					if junction := r.junction.Resolve(existing, r.style.Vertical); junction != 0 {
						canvas.Set(p, junction)
					}
				} else {
					canvas.Set(p, r.style.Vertical)
				}
			}
		}
	}
	
	return nil
}

// drawSegmentForClosedPath draws a line segment for closed paths, skipping both endpoints to leave room for corners.
func (r *PathRenderer) drawSegmentForClosedPath(canvas canvas.Canvas, from, to core.Point) error {
	dx := to.X - from.X
	dy := to.Y - from.Y
	
	// Horizontal line
	if dy == 0 {
		step := 1
		if dx < 0 {
			step = -1
		}
		
		// Skip both start and end points for corners
		startX := from.X + step
		endX := to.X
		
		for x := startX; x != endX; x += step {
			p := core.Point{X: x, Y: from.Y}
			
			// Check if there's already a character here (junction)
			if existing := canvas.Get(p); existing != ' ' && existing != 0 {
				if junction := r.junction.Resolve(existing, r.style.Horizontal); junction != 0 {
					canvas.Set(p, junction)
				}
			} else {
				canvas.Set(p, r.style.Horizontal)
			}
		}
	}
	
	// Vertical line
	if dx == 0 {
		step := 1
		if dy < 0 {
			step = -1
		}
		
		// Skip both start and end points for corners
		startY := from.Y + step
		endY := to.Y
		
		for y := startY; y != endY; y += step {
			p := core.Point{X: from.X, Y: y}
			
			// Check if there's already a character here (junction)
			if existing := canvas.Get(p); existing != ' ' && existing != 0 {
				if junction := r.junction.Resolve(existing, r.style.Vertical); junction != 0 {
					canvas.Set(p, junction)
				}
			} else {
				canvas.Set(p, r.style.Vertical)
			}
		}
	}
	
	return nil
}

// drawSegmentWithOptions draws a line segment with options to skip start/end points.
func (r *PathRenderer) drawSegmentWithOptions(canvas canvas.Canvas, from, to core.Point, drawArrow bool, skipStart bool) error {
	dx := to.X - from.X
	dy := to.Y - from.Y
	
	// Horizontal line
	if dy == 0 {
		step := 1
		if dx < 0 {
			step = -1
		}
		
		startX := from.X
		if skipStart {
			startX += step
		}
		
		for x := startX; x != to.X; x += step {
			p := core.Point{X: x, Y: from.Y}
			
			// Check if we should draw an arrow at the end
			if drawArrow && x == to.X-step {
				arrowChar := r.style.ArrowRight
				if step < 0 {
					arrowChar = r.style.ArrowLeft
				}
				// Check for existing character and resolve junction
				if existing := canvas.Get(p); existing != ' ' && existing != 0 {
					if junction := r.junction.Resolve(existing, arrowChar); junction != 0 {
						canvas.Set(p, junction)
					} else {
						canvas.Set(p, arrowChar)
					}
				} else {
					canvas.Set(p, arrowChar)
				}
			} else {
				// Check if there's already a character here (junction)
				if existing := canvas.Get(p); existing != ' ' && existing != 0 {
					if junction := r.junction.Resolve(existing, r.style.Horizontal); junction != 0 {
						canvas.Set(p, junction)
					}
				} else {
					canvas.Set(p, r.style.Horizontal)
				}
			}
		}
	}
	
	// Vertical line
	if dx == 0 {
		step := 1
		if dy < 0 {
			step = -1
		}
		
		startY := from.Y
		if skipStart {
			startY += step
		}
		
		for y := startY; y != to.Y; y += step {
			p := core.Point{X: from.X, Y: y}
			
			// Check if we should draw an arrow at the end
			if drawArrow && y == to.Y-step {
				arrowChar := r.style.ArrowDown
				if step < 0 {
					arrowChar = r.style.ArrowUp
				}
				// Check for existing character and resolve junction
				if existing := canvas.Get(p); existing != ' ' && existing != 0 {
					if junction := r.junction.Resolve(existing, arrowChar); junction != 0 {
						canvas.Set(p, junction)
					} else {
						canvas.Set(p, arrowChar)
					}
				} else {
					canvas.Set(p, arrowChar)
				}
			} else {
				// Check if there's already a character here (junction)
				if existing := canvas.Get(p); existing != ' ' && existing != 0 {
				if junction := r.junction.Resolve(existing, r.style.Vertical); junction != 0 {
					canvas.Set(p, junction)
				}
				} else {
					canvas.Set(p, r.style.Vertical)
				}
			}
		}
	}
	
	return nil
}

// drawSegment draws a line segment between two points.
func (r *PathRenderer) drawSegment(canvas canvas.Canvas, from, to core.Point, drawArrow bool) error {
	return r.drawSegmentWithOptions(canvas, from, to, drawArrow, false)
}

// getCornerChar determines the appropriate corner character for a turn in the path.
func (r *PathRenderer) getCornerChar(prev, current, next core.Point) rune {
	// Determine incoming and outgoing directions
	dxIn := current.X - prev.X
	dyIn := current.Y - prev.Y
	dxOut := next.X - current.X
	dyOut := next.Y - current.Y
	
	// Normalize to directions (-1, 0, 1)
	if dxIn != 0 {
		dxIn = dxIn / abs(dxIn)
	}
	if dyIn != 0 {
		dyIn = dyIn / abs(dyIn)
	}
	if dxOut != 0 {
		dxOut = dxOut / abs(dxOut)
	}
	if dyOut != 0 {
		dyOut = dyOut / abs(dyOut)
	}
	
	// Determine which corner to use based on the turn direction
	// The key insight: we need to determine which two sides of the corner are "open"
	// (i.e., have lines extending from them)
	
	// Check all four corner cases
	// ┌ (BottomLeft): lines extend right and down
	if (dxIn < 0 && dyOut > 0) || (dyIn < 0 && dxOut > 0) {
		return r.style.BottomLeft
	}
	// ┐ (BottomRight): lines extend left and down
	if (dxIn > 0 && dyOut > 0) || (dyIn < 0 && dxOut < 0) {
		return r.style.BottomRight
	}
	// └ (TopLeft): lines extend right and up  
	if (dxIn < 0 && dyOut < 0) || (dyIn > 0 && dxOut > 0) {
		return r.style.TopLeft
	}
	// ┘ (TopRight): lines extend left and up
	if (dxIn > 0 && dyOut < 0) || (dyIn > 0 && dxOut < 0) {
		return r.style.TopRight
	}
	
	return 0 // No corner needed (straight line)
}




// identifyCorners analyzes a path and returns a map of corner positions to their characters
func (r *PathRenderer) identifyCorners(points []core.Point, isClosed bool) map[core.Point]rune {
	corners := make(map[core.Point]rune)
	
	// Need at least 3 points for a corner
	if len(points) < 3 {
		return corners
	}
	
	// Determine the range of points to check for corners
	startIdx := 1 // Skip first point unless closed
	endIdx := len(points) - 1 // Skip last point unless closed
	
	if isClosed {
		startIdx = 0
		endIdx = len(points)
	}
	
	for i := startIdx; i < endIdx; i++ {
		var prev, current, next core.Point
		
		if isClosed {
			// Handle wrap-around for closed paths
			prev = points[(i-1+len(points))%len(points)]
			current = points[i%len(points)]
			next = points[(i+1)%len(points)]
		} else {
			// For open paths, we already skip first and last
			prev = points[i-1]
			current = points[i]
			next = points[i+1]
		}
		
		// Check if this point is a corner
		if corner := r.getCornerChar(prev, current, next); corner != 0 {
			corners[current] = corner
		}
	}
	
	return corners
}

// isLShaped checks if 3 points form an L shape (one horizontal and one vertical segment)
func isLShaped(points []core.Point) bool {
	if len(points) != 3 {
		return false
	}
	
	// Check if first segment is horizontal/vertical and second is perpendicular
	dx1 := points[1].X - points[0].X
	dy1 := points[1].Y - points[0].Y
	dx2 := points[2].X - points[1].X
	dy2 := points[2].Y - points[1].Y
	
	// One segment horizontal, one vertical
	return (dx1 == 0 && dy1 != 0 && dx2 != 0 && dy2 == 0) ||
	       (dx1 != 0 && dy1 == 0 && dx2 == 0 && dy2 != 0)
}

// selectLineStyle chooses the appropriate line style based on terminal capabilities.
func selectLineStyle(caps TerminalCapabilities) LineStyle {
	switch caps.UnicodeLevel {
	case UnicodeFull, UnicodeExtended:
		return LineStyle{
			// Basic lines
			Horizontal: '─',
			Vertical:   '│',
			// Corners (named from box perspective, not line direction)
			// TopLeft = top-left corner of a box = └ (lines go right and up)
			// TopRight = top-right corner of a box = ┘ (lines go left and up)
			// BottomLeft = bottom-left corner of a box = ┌ (lines go right and down)
			// BottomRight = bottom-right corner of a box = ┐ (lines go left and down)
			TopLeft:     '└',
			TopRight:    '┘',
			BottomLeft:  '┌',
			BottomRight: '┐',
			// Junctions
			Cross:    '┼',
			TeeUp:    '┴',
			TeeDown:  '┬',
			TeeLeft:  '┤',
			TeeRight: '├',
			// Arrows (triangular for better alignment)
			ArrowUp:    '▲',
			ArrowDown:  '▼',
			ArrowLeft:  '◀',
			ArrowRight: '▶',
		}
	case UnicodeBasic:
		return LineStyle{
			// Basic lines
			Horizontal: '-',
			Vertical:   '|',
			// Corners
			TopLeft:     '+',
			TopRight:    '+',
			BottomLeft:  '+',
			BottomRight: '+',
			// Junctions
			Cross:    '+',
			TeeUp:    '+',
			TeeDown:  '+',
			TeeLeft:  '+',
			TeeRight: '+',
			// Arrows (ASCII for basic Unicode)
			ArrowUp:    '^',
			ArrowDown:  'v',
			ArrowLeft:  '<',
			ArrowRight: '>',
		}
	default: // UnicodeNone (ASCII)
		return LineStyle{
			// Basic lines
			Horizontal: '-',
			Vertical:   '|',
			// Corners
			TopLeft:     '+',
			TopRight:    '+',
			BottomLeft:  '+',
			BottomRight: '+',
			// Junctions
			Cross:    '+',
			TeeUp:    '+',
			TeeDown:  '+',
			TeeLeft:  '+',
			TeeRight: '+',
			// Arrows
			ArrowUp:    '^',
			ArrowDown:  'v',
			ArrowLeft:  '<',
			ArrowRight: '>',
		}
	}
}

// abs returns the absolute value of an integer.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// isJunctionChar checks if a character is a junction character.
func isJunctionChar(r rune) bool {
	return r == '┼' || r == '┴' || r == '┬' || r == '┤' || r == '├' || r == '+'
}
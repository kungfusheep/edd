package render

import (
	"edd/diagram"
	"edd/layout"
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
	hintStyle  string // Current hint style (solid, dashed, dotted, double)
	hintColor  string // Current hint color
	hintBold   bool   // Current hint bold setting
	hintItalic bool   // Current hint italic setting
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
func (r *PathRenderer) RenderPath(canvas Canvas, path diagram.Path, hasArrow bool) error {
	return r.RenderPathWithOptions(canvas, path, hasArrow, false)
}

// RenderPathWithHints draws a path with visual hints applied.
func (r *PathRenderer) RenderPathWithHints(canvas Canvas, path diagram.Path, hasArrow bool, hints map[string]string) error {
	// Save current style
	oldStyle := r.style
	oldHintStyle := r.hintStyle
	oldHintColor := r.hintColor
	oldHintBold := r.hintBold
	oldHintItalic := r.hintItalic
	
	// Apply hints
	if hints != nil {
		if style, ok := hints["style"]; ok {
			r.hintStyle = style
			r.applyHintStyle(style)
		}
		if color, ok := hints["color"]; ok {
			r.hintColor = color
		}
		if bold, ok := hints["bold"]; ok && bold == "true" {
			r.hintBold = true
		}
		if italic, ok := hints["italic"]; ok && italic == "true" {
			r.hintItalic = true
		}
	}
	
	// Render the path (color, bold, and italic will be applied via setWithColor method)
	err := r.RenderPathWithOptions(canvas, path, hasArrow, true)
	
	// Restore original style
	r.style = oldStyle
	r.hintStyle = oldHintStyle
	r.hintColor = oldHintColor
	r.hintBold = oldHintBold
	r.hintItalic = oldHintItalic
	
	return err
}

// setWithColor sets a character on the canvas, applying color and style if the canvas supports it
func (r *PathRenderer) setWithColor(canvas Canvas, p diagram.Point, char rune) error {
	// If we have color, bold, or italic, use SetWithColorAndStyle
	if r.hintColor != "" || r.hintBold || r.hintItalic {
		// Build style string
		style := ""
		if r.hintBold && r.hintItalic {
			style = "bold+italic"
		} else if r.hintBold {
			style = "bold"
		} else if r.hintItalic {
			style = "italic"
		}
		
		// Try to set with color and style if the canvas supports it
		if coloredCanvas, ok := canvas.(*ColoredMatrixCanvas); ok {
			return coloredCanvas.SetWithColorAndStyle(p, char, r.hintColor, style)
		}
		// Also check if it's a type that supports SetWithColorAndStyle method (like offsetCanvas)
		if styleSetter, ok := canvas.(interface {
			SetWithColorAndStyle(diagram.Point, rune, string, string) error
		}); ok {
			return styleSetter.SetWithColorAndStyle(p, char, r.hintColor, style)
		}
		// Fall back to just color if available
		if r.hintColor != "" {
			if colorSetter, ok := canvas.(interface {
				SetWithColor(diagram.Point, rune, string) error
			}); ok {
				return colorSetter.SetWithColor(p, char, r.hintColor)
			}
		}
	}
	// Fall back to regular set
	return canvas.Set(p, char)
}

// applyHintStyle modifies the line style based on the hint.
func (r *PathRenderer) applyHintStyle(style string) {
	switch style {
	case "dashed":
		if r.caps.UnicodeLevel >= UnicodeBasic {
			r.style.Horizontal = '╌' // Box drawing light dashed
			r.style.Vertical = '╎'   // Box drawing light dashed vertical
		} else {
			r.style.Horizontal = '-'
			r.style.Vertical = '|'
		}
	case "dotted":
		if r.caps.UnicodeLevel >= UnicodeBasic {
			r.style.Horizontal = '·' // Middle dot
			r.style.Vertical = '·'   // Middle dot
		} else {
			r.style.Horizontal = '.'
			r.style.Vertical = '.'
		}
	case "double":
		if r.caps.UnicodeLevel >= UnicodeFull {
			r.style.Horizontal = '═' // Box drawing double horizontal
			r.style.Vertical = '║'   // Box drawing double vertical
			// Update corners for double lines
			r.style.TopLeft = '╚'     // Box drawing double up and right
			r.style.TopRight = '╝'    // Box drawing double up and left
			r.style.BottomLeft = '╔'  // Box drawing double down and right
			r.style.BottomRight = '╗' // Box drawing double down and left
		} else {
			r.style.Horizontal = '='
			r.style.Vertical = '|'
		}
	// "solid" or default - no change needed
	}
}

// getLineChar returns the appropriate character for a line segment based on style hints.
// For dashed and dotted styles, this creates the pattern effect.
func (r *PathRenderer) getLineChar(horizontal bool, position int) rune {
	switch r.hintStyle {
	case "dashed":
		// For dashed, use the dashed characters consistently
		if horizontal {
			return r.style.Horizontal
		}
		return r.style.Vertical
	case "dotted":
		// For dotted, space out the dots
		if position%2 == 0 {
			if horizontal {
				return r.style.Horizontal
			}
			return r.style.Vertical
		}
		return ' '
	default:
		// Solid or double - use the normal style
		if horizontal {
			return r.style.Horizontal
		}
		return r.style.Vertical
	}
}

// RenderPathWithOptions draws a path with additional rendering options.
func (r *PathRenderer) RenderPathWithOptions(canvas Canvas, path diagram.Path, hasArrow bool, isConnection bool) error {
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
		
		// For the first segment of a connection from a box edge, handle branch creation
		if i == 0 && isConnection && !isClosed {
			existing := canvas.Get(from)
			// Check if we're starting from a box edge (could be clean or already a branch)
			if existing == '│' || existing == '─' || existing == '├' || existing == '┤' || existing == '┬' || existing == '┴' {
				// Don't draw the line at the first point - let it stay as box edge
				// This prevents │ + ─ = ┼ when we want │ + ─ = ├
				if err := r.drawSegmentSkippingCornersWithOptions(canvas, from, to, corners, drawArrowOnSegment, true); err != nil {
					return err
				}
				
				// After drawing, place appropriate branch character at start
				dx := to.X - from.X
				dy := to.Y - from.Y
				var branchChar rune
				
				if existing == '│' && dy == 0 {
					// Horizontal from vertical edge
					if dx > 0 {
						branchChar = r.style.TeeRight  // ├
					} else {
						branchChar = r.style.TeeLeft   // ┤
					}
				} else if existing == '─' && dx == 0 {
					// Vertical from horizontal edge  
					if dy > 0 {
						branchChar = r.style.TeeDown   // ┬
					} else {
						branchChar = r.style.TeeUp     // ┴
					}
				}
				
				if branchChar != 0 {
					r.setWithColor(canvas, from, branchChar)
				}
			} else {
				// Not a clean edge, draw normally
				if err := r.drawSegmentSkippingCorners(canvas, from, to, corners, drawArrowOnSegment); err != nil {
					return err
				}
			}
		} else {
			// Not first segment or not a connection, draw normally
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
		r.setWithColor(canvas, pos, corner)
	}
	
	// Phase 4: For multi-segment non-arrow paths, ensure the final endpoint is drawn
	// (it's not a corner and segments exclude endpoints)
	if !hasArrow && !isClosed && len(points) > 2 {
		lastPoint := points[len(points)-1]
		// Only draw if it's not already a corner
		if _, isCorner := corners[lastPoint]; !isCorner {
			// Determine character based on the direction of the last segment
			secondLast := points[len(points)-2]
			if lastPoint.Y == secondLast.Y {
				// Horizontal segment
				r.setWithColor(canvas, lastPoint, r.style.Horizontal)
			} else if lastPoint.X == secondLast.X {
				// Vertical segment
				r.setWithColor(canvas, lastPoint, r.style.Vertical)
			}
		}
	}
	
	return nil
}

// placeStartBranch places a corner character at the start of a connection
// This corner will merge with the box edge (─ or │) to create a branch character (├, ┤, ┬, ┴)
func (r *PathRenderer) placeStartBranch(canvas Canvas, from, to diagram.Point) {
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
func (r *PathRenderer) drawSegmentSkippingCorners(canvas Canvas, from, to diagram.Point, corners map[diagram.Point]rune, drawArrow bool) error {
	return r.drawSegmentSkippingCornersWithOptions(canvas, from, to, corners, drawArrow, false)
}

func (r *PathRenderer) drawSegmentSkippingCornersWithOptions(canvas Canvas, from, to diagram.Point, corners map[diagram.Point]rune, drawArrow bool, skipFirst bool) error {
	dx := to.X - from.X
	dy := to.Y - from.Y
	
	// Horizontal line
	if dy == 0 {
		step := 1
		if dx < 0 {
			step = -1
		}
		
		// Always stop one before the endpoint (endpoint is exclusive)
		endX := to.X - step
		
		for x := from.X; x != endX+step; x += step {
			p := diagram.Point{X: x, Y: from.Y}
			
			// Skip the first point if requested
			if skipFirst && x == from.X {
				continue
			}
			
			// Skip if this is a corner position
			if _, isCorner := corners[p]; isCorner {
				continue
			}
			
			
			// Handle endpoint with arrow
			if x == endX && drawArrow {
				arrowChar := r.style.ArrowRight
				if step < 0 {
					arrowChar = r.style.ArrowLeft
				}
				// Debug: print when placing arrows
				//fmt.Printf("Placing arrow %c at (%d,%d)\n", arrowChar, p.X, p.Y)
				r.setWithColor(canvas, p, arrowChar)
			} else {
				r.setWithColor(canvas, p, r.style.Horizontal)
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
		
		// Always stop one before the endpoint (endpoint is exclusive)
		endY := to.Y - step
		
		for y := from.Y; y != endY+step; y += step {
			p := diagram.Point{X: from.X, Y: y}
			
			// Skip the first point if requested
			if skipFirst && y == from.Y {
				continue
			}
			
			// Skip if this is a corner position
			if _, isCorner := corners[p]; isCorner {
				continue
			}
			
			// Handle endpoint with arrow
			if y == endY && drawArrow {
				arrowChar := r.style.ArrowDown
				if step < 0 {
					arrowChar = r.style.ArrowUp
				}
				r.setWithColor(canvas, p, arrowChar)
			} else {
				r.setWithColor(canvas, p, r.style.Vertical)
			}
		}
		return nil
	}
	
	// Diagonal lines not supported in terminal rendering
	return fmt.Errorf("diagonal lines not supported: from (%d,%d) to (%d,%d)", from.X, from.Y, to.X, to.Y)
}

// drawSegmentInclusive draws a line segment including the endpoint.
// For multi-segment paths, we skip the start point if it's a potential corner location.
func (r *PathRenderer) drawSegmentInclusive(canvas Canvas, from, to diagram.Point, drawArrow bool) error {
	dx := to.X - from.X
	dy := to.Y - from.Y
	
	// Horizontal line
	if dy == 0 {
		step := 1
		if dx < 0 {
			step = -1
		}
		
		for x := from.X; ; x += step {
			p := diagram.Point{X: x, Y: from.Y}
			
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
			p := diagram.Point{X: from.X, Y: y}
			
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
func (r *PathRenderer) drawSegmentForClosedPath(canvas Canvas, from, to diagram.Point) error {
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
			p := diagram.Point{X: x, Y: from.Y}
			
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
			p := diagram.Point{X: from.X, Y: y}
			
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
func (r *PathRenderer) drawSegmentWithOptions(canvas Canvas, from, to diagram.Point, drawArrow bool, skipStart bool) error {
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
			p := diagram.Point{X: x, Y: from.Y}
			
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
			p := diagram.Point{X: from.X, Y: y}
			
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
func (r *PathRenderer) drawSegment(canvas Canvas, from, to diagram.Point, drawArrow bool) error {
	return r.drawSegmentWithOptions(canvas, from, to, drawArrow, false)
}

// getCornerChar determines the appropriate corner character for a turn in the path.
func (r *PathRenderer) getCornerChar(prev, current, next diagram.Point) rune {
	// Determine incoming and outgoing directions
	dxIn := current.X - prev.X
	dyIn := current.Y - prev.Y
	dxOut := next.X - current.X
	dyOut := next.Y - current.Y
	
	// Normalize to directions (-1, 0, 1)
	if dxIn != 0 {
		dxIn = dxIn / layout.Abs(dxIn)
	}
	if dyIn != 0 {
		dyIn = dyIn / layout.Abs(dyIn)
	}
	if dxOut != 0 {
		dxOut = dxOut / layout.Abs(dxOut)
	}
	if dyOut != 0 {
		dyOut = dyOut / layout.Abs(dyOut)
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
func (r *PathRenderer) identifyCorners(points []diagram.Point, isClosed bool) map[diagram.Point]rune {
	corners := make(map[diagram.Point]rune)
	
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
		var prev, current, next diagram.Point
		
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
func isLShaped(points []diagram.Point) bool {
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


// isJunctionChar checks if a character is a junction character.
func isJunctionChar(r rune) bool {
	return r == '┼' || r == '┴' || r == '┬' || r == '┤' || r == '├' || r == '+'
}
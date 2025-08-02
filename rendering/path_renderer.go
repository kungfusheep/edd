package rendering

import (
	"edd/canvas"
	"edd/core"
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
			// Draw a dot or small arrow
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
	
	// Special handling for simple L-shaped paths (3 points forming an L)
	if !isClosed && len(points) == 3 && isLShaped(points) {
		// Draw the two segments without the middle point
		if err := r.drawSegment(canvas, points[0], points[1], false); err != nil {
			return err
		}
		// For the second segment of L-shape, include the endpoint
		if hasArrow {
			if err := r.drawSegmentInclusive(canvas, points[1], points[2], hasArrow); err != nil {
				return err
			}
		} else {
			// Draw up to but not including the last point, then add it separately
			if err := r.drawSegment(canvas, points[1], points[2], false); err != nil {
				return err
			}
			// Add the last point
			if points[1].X == points[2].X {
				canvas.Set(points[2], r.style.Vertical)
			} else {
				canvas.Set(points[2], r.style.Horizontal)
			}
		}
		// Place the corner at the middle point
		if corner := r.getCornerChar(points[0], points[1], points[2]); corner != 0 {
			canvas.Set(points[1], corner)
		}
		return nil
	}
	
	// For connection paths, mark endpoints for special handling
	var connectionEndpoints map[core.Point]bool
	if isConnection && len(points) >= 2 {
		connectionEndpoints = map[core.Point]bool{
			points[0]:              true,
			points[len(points)-1]:  true,
		}
	}
	
	// Draw each segment
	numSegments := len(points) - 1
	if isClosed {
		// For closed paths, we need one more segment to close the loop
		numSegments = len(points)
	}
	
	for i := 0; i < numSegments; i++ {
		current := points[i]
		next := points[(i+1)%len(points)] // Use modulo to wrap around for closed paths
		
		// Determine if this is the last segment
		isLastSegment := (i == numSegments-1)
		
		// Draw the segment
		// Special handling based on path type:
		// - For simple 2-point paths with arrows: include endpoint so arrow is at the end
		// - For simple 2-point paths without arrows: exclude endpoint (traditional line drawing)
		// - For multi-segment paths: last segment includes endpoint, others exclude it
		// - For closed paths: all segments exclude endpoints to avoid double-drawing
		if isLastSegment && !isClosed && (len(points) > 2 || hasArrow) {
			// Multi-segment open path or arrow path: include endpoint on last segment
			if err := r.drawSegmentInclusive(canvas, current, next, hasArrow); err != nil {
				return err
			}
		} else {
			// Simple path, intermediate segment, or closed path: exclude endpoint
			if isClosed && len(points) > 2 {
				// For closed paths with corners, skip both endpoints to leave room for corners
				if err := r.drawSegmentForClosedPath(canvas, current, next); err != nil {
					return err
				}
			} else {
				// For other cases, use normal segment drawing
				if err := r.drawSegment(canvas, current, next, isLastSegment && hasArrow && !isClosed); err != nil {
					return err
				}
			}
		}
	}
	
	// Draw corners where direction changes
	cornerCount := len(points) - 1
	if isClosed {
		// For closed paths, every point can be a corner
		cornerCount = len(points)
	}
	
	for i := 0; i < cornerCount; i++ {
		// Skip corners for the first and last points of open paths
		if !isClosed && (i == 0 || i == len(points)-1) {
			continue
		}
		
		// Get the three points involved in the corner
		var prev, current, next core.Point
		if isClosed {
			prev = points[(i+len(points)-1)%len(points)]
			current = points[i]
			next = points[(i+1)%len(points)]
		} else {
			if i == 0 || i >= len(points)-1 {
				continue
			}
			prev = points[i-1]
			current = points[i]
			next = points[i+1]
		}
		
		if corner := r.getCornerChar(prev, current, next); corner != 0 {
			// Check render mode and existing character
			if r.renderMode == RenderModePreserveCorners {
				// In preserve corners mode, corners override simple lines but not junctions
				if existing := canvas.Get(current); existing == ' ' || existing == 0 || 
					existing == r.style.Horizontal || existing == r.style.Vertical {
					canvas.Set(current, corner)
				} else if IsJunctionChar(existing) && isClosed {
					// For closed paths (like boxes), corners should override junctions too
					canvas.Set(current, corner)
				}
				// Otherwise keep existing character (e.g., existing corners)
			} else {
				// Standard mode: resolve junctions
				if existing := canvas.Get(current); existing != ' ' && existing != 0 {
					if junction := r.junction.Resolve(existing, corner); junction != 0 {
						canvas.Set(current, junction)
					} else {
						canvas.Set(current, corner)
					}
				} else {
					canvas.Set(current, corner)
				}
			}
		}
	}
	
	// Handle connection endpoints specially - use T-junctions instead of crosses
	if isConnection && connectionEndpoints != nil {
		for point := range connectionEndpoints {
			existing := canvas.Get(point)
			
			// Skip arrow characters - they should not be modified
			if existing == '▶' || existing == '◀' || existing == '▲' || existing == '▼' ||
			   existing == '>' || existing == '<' || existing == '^' || existing == 'v' {
				continue
			}
			
			// Check for junction characters that indicate a connection meeting a box edge
			if existing == '┼' || existing == '├' || existing == '┤' || existing == '┬' || existing == '┴' {
				// This is a junction created by our connection meeting a box edge
				// Determine which direction the connection goes
				isStart := point == points[0]
				var direction core.Point
				
				if isStart && len(points) > 1 {
					direction = points[1]
				} else if !isStart && len(points) > 1 {
					direction = points[len(points)-2]
				}
				
				// Only handle start points - endpoints should keep their arrows
				if isStart {
					// Replace cross junctions with appropriate T-junctions at start points
					if existing == '┼' {
						// Cross junction - determine which T-junction to use
						if direction.X > point.X {
							canvas.Set(point, '├') // Connection goes right
						} else if direction.X < point.X {
							canvas.Set(point, '┤') // Connection goes left
						} else if direction.Y > point.Y {
							canvas.Set(point, '┬') // Connection goes down
						} else if direction.Y < point.Y {
							canvas.Set(point, '┴') // Connection goes up
						}
					}
					// Other junctions (├, ┤, ┬, ┴) are already correct T-junctions
				}
				// Don't modify endpoints - they may have arrows
			}
			// Note: Corners (┌, ┐, └, ┘) are already correct T-junctions
		}
	}
	
	return nil
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
			// Corners
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
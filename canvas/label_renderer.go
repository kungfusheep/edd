package canvas

import (
	"edd/core"
	"edd/utils"
	"strings"
)

// LabelRenderer handles the rendering of connection labels on paths
type LabelRenderer struct {
	maxLabelLength int // Maximum length before truncation
}

// NewLabelRenderer creates a new label renderer
func NewLabelRenderer() *LabelRenderer {
	return &LabelRenderer{
		maxLabelLength: 10, // Default max length
	}
}

// LabelPosition represents where along a path a label should be placed
type LabelPosition int

const (
	LabelAuto LabelPosition = iota // Automatically choose best position
	LabelStart                      // Near the start of the connection
	LabelMiddle                     // At the midpoint (default)
	LabelEnd                        // Near the end of the connection
)

// Segment represents a line segment in a path
type Segment struct {
	Start        core.Point
	End          core.Point
	IsHorizontal bool
	IsVertical   bool
}

// RenderLabel renders a label on a path at the specified position
func (lr *LabelRenderer) RenderLabel(c Canvas, path core.Path, label string, position LabelPosition) {
	if label == "" || len(path.Points) < 2 {
		return
	}

	// Format the label
	formattedLabel := lr.formatLabel(label)

	// Find the best segment for the label (prefer horizontal segments)
	segment := lr.findBestSegmentForLabel(path, formattedLabel, position)
	if segment == nil {
		// If no suitable segment found, try with relaxed constraints
		segment = lr.findAnySegmentForLabel(path, formattedLabel, position)
		if segment == nil {
			return // Still no suitable segment
		}
	}
	
	// Render the label inline on the segment
	lr.renderInlineLabel(c, segment, formattedLabel)
}

// findBestSegmentForLabel finds the best segment in the path to place a label
func (lr *LabelRenderer) findBestSegmentForLabel(path core.Path, label string, position LabelPosition) *Segment {
	if len(path.Points) < 2 {
		return nil
	}

	labelLen := len(label)
	minSegmentLen := labelLen + 2 // Need space for label plus minimal padding

	// Combine consecutive segments in the same direction
	segments := lr.combineConsecutiveSegments(path)
	
	// Find the longest suitable segment
	var bestSegment *Segment
	var bestLength int
	
	for _, seg := range segments {
		var segLen int
		if seg.IsHorizontal {
			segLen = utils.Abs(seg.End.X - seg.Start.X)
		} else if seg.IsVertical {
			segLen = utils.Abs(seg.End.Y - seg.Start.Y)
		} else {
			continue // Skip diagonal segments
		}
		
		// Prefer longer segments and horizontal over vertical
		if segLen >= minSegmentLen {
			if bestSegment == nil || segLen > bestLength || (segLen == bestLength && seg.IsHorizontal && !bestSegment.IsHorizontal) {
				tempSeg := seg
				bestSegment = &tempSeg
				bestLength = segLen
			}
		}
	}

	return bestSegment
}

// combineConsecutiveSegments combines consecutive path segments that go in the same direction
func (lr *LabelRenderer) combineConsecutiveSegments(path core.Path) []Segment {
	if len(path.Points) < 2 {
		return nil
	}
	
	segments := []Segment{}
	currentStart := path.Points[0]
	currentEnd := path.Points[1]
	currentDirection := getSegmentDirection(currentStart, currentEnd)
	
	for i := 2; i < len(path.Points); i++ {
		nextPoint := path.Points[i]
		nextDirection := getSegmentDirection(currentEnd, nextPoint)
		
		// If direction changes, save the current segment and start a new one
		if nextDirection != currentDirection {
			seg := Segment{
				Start:        currentStart,
				End:          currentEnd,
				IsHorizontal: currentStart.Y == currentEnd.Y,
				IsVertical:   currentStart.X == currentEnd.X,
			}
			segments = append(segments, seg)
			
			currentStart = currentEnd
			currentEnd = nextPoint
			currentDirection = nextDirection
		} else {
			// Same direction, extend the current segment
			currentEnd = nextPoint
		}
	}
	
	// Add the last segment
	seg := Segment{
		Start:        currentStart,
		End:          currentEnd,
		IsHorizontal: currentStart.Y == currentEnd.Y,
		IsVertical:   currentStart.X == currentEnd.X,
	}
	segments = append(segments, seg)
	
	return segments
}

// getSegmentDirection returns a simple direction indicator for a segment
func getSegmentDirection(from, to core.Point) string {
	dx := to.X - from.X
	dy := to.Y - from.Y
	
	if dx > 0 && dy == 0 {
		return "right"
	} else if dx < 0 && dy == 0 {
		return "left"
	} else if dx == 0 && dy > 0 {
		return "down"
	} else if dx == 0 && dy < 0 {
		return "up"
	}
	return "diagonal"
}

// findAnySegmentForLabel finds any segment with relaxed constraints
func (lr *LabelRenderer) findAnySegmentForLabel(path core.Path, label string, position LabelPosition) *Segment {
	if len(path.Points) < 2 {
		return nil
	}

	// Try to find ANY horizontal or vertical segment, even if it's short
	for i := 0; i < len(path.Points)-1; i++ {
		start := path.Points[i]
		end := path.Points[i+1]
		
		seg := Segment{
			Start:        start,
			End:          end,
			IsHorizontal: start.Y == end.Y,
			IsVertical:   start.X == end.X,
		}
		
		// Accept any horizontal or vertical segment
		if seg.IsHorizontal || seg.IsVertical {
			return &seg
		}
	}

	return nil
}

// renderInlineLabel renders a label inline on a segment, replacing the line characters
func (lr *LabelRenderer) renderInlineLabel(c Canvas, segment *Segment, label string) {
	if segment.IsHorizontal {
		lr.renderHorizontalInlineLabel(c, segment, label)
	} else if segment.IsVertical {
		lr.renderVerticalInlineLabel(c, segment, label)
	}
}

// renderHorizontalInlineLabel renders a label inline on a horizontal segment
func (lr *LabelRenderer) renderHorizontalInlineLabel(c Canvas, segment *Segment, label string) {
	labelLen := len(label)
	segmentLen := utils.Abs(segment.End.X - segment.Start.X)
	
	// Calculate where to place the label (centered on the segment)
	minX := min(segment.Start.X, segment.End.X)
	maxX := max(segment.Start.X, segment.End.X)
	labelStart := minX + (segmentLen - labelLen) / 2
	
	// Ensure label fits within segment
	if labelStart < minX {
		labelStart = minX + 1 // Leave space at the start
	}
	if labelStart + labelLen > maxX {
		labelStart = maxX - labelLen - 1 // Leave space at the end
	}
	
	// If the segment is too short for the label, just place it at the start
	if segmentLen < labelLen + 2 {
		labelStart = minX + 1
	}
	
	// The label itself - use direct matrix access to overwrite the line
	// We can't use c.Set because it merges characters and won't overwrite lines with text
	y := segment.Start.Y
	
	// Try to get direct matrix access
	var matrix [][]rune
	var xOffset, yOffset int
	
	// Check if we can access the canvas as a MatrixCanvas for direct access
	if mc, ok := c.(*MatrixCanvas); ok {
		matrix = mc.Matrix()
		xOffset = 0
		yOffset = 0
	} else if oc, ok := c.(interface{
		Matrix() [][]rune
		Offset() core.Point
	}); ok {
		// Handle offsetCanvas case
		matrix = oc.Matrix()
		offset := oc.Offset()
		xOffset = -offset.X
		yOffset = -offset.Y
	}
	
	if matrix != nil {
		// Direct matrix access to force overwrite
		actualY := y + yOffset
		if actualY >= 0 && actualY < len(matrix) {
			for i, ch := range label {
				actualX := labelStart + i + xOffset
				if actualX >= 0 && actualX < len(matrix[actualY]) {
					matrix[actualY][actualX] = ch
				}
			}
		}
	} else {
		// Fallback to normal Set (won't work properly with lines)
		for i, ch := range label {
			pos := core.Point{X: labelStart + i, Y: y}
			if pos.X >= minX && pos.X <= maxX {
				c.Set(pos, ch)
			}
		}
	}
}

// formatLabel formats the label text, truncating if necessary
func (lr *LabelRenderer) formatLabel(label string) string {
	label = strings.TrimSpace(label)
	
	// Truncate if too long
	if len(label) > lr.maxLabelLength {
		label = label[:lr.maxLabelLength-2] + ".."
	}

	// Add brackets around the label
	return "[" + label + "]"
}

// renderVerticalInlineLabel renders a label inline on a vertical segment
func (lr *LabelRenderer) renderVerticalInlineLabel(c Canvas, segment *Segment, label string) {
	// For vertical segments, we'll render the label vertically
	labelLen := len(label)
	segmentLen := utils.Abs(segment.End.Y - segment.Start.Y)
	
	// Calculate where to place the label (centered on the segment)
	minY := min(segment.Start.Y, segment.End.Y)
	maxY := max(segment.Start.Y, segment.End.Y)
	labelStart := minY + (segmentLen - labelLen) / 2
	
	// Ensure label fits within segment
	if labelStart < minY {
		labelStart = minY + 1 // Leave space at the start
	}
	if labelStart + labelLen > maxY {
		labelStart = maxY - labelLen - 1 // Leave space at the end
	}
	
	// If the segment is too short for the label, just place it at the start
	if segmentLen < labelLen + 2 {
		labelStart = minY + 1
	}
	
	// The label itself (rendered vertically) - use direct matrix access
	x := segment.Start.X
	
	// Try to get direct matrix access
	var matrix [][]rune
	var xOffset, yOffset int
	
	// Check if we can access the canvas as a MatrixCanvas for direct access
	if mc, ok := c.(*MatrixCanvas); ok {
		matrix = mc.Matrix()
		xOffset = 0
		yOffset = 0
	} else if oc, ok := c.(interface{
		Matrix() [][]rune
		Offset() core.Point
	}); ok {
		// Handle offsetCanvas case
		matrix = oc.Matrix()
		offset := oc.Offset()
		xOffset = -offset.X
		yOffset = -offset.Y
	}
	
	if matrix != nil && len(matrix) > 0 {
		// Direct matrix access to force overwrite
		actualX := x + xOffset
		if actualX >= 0 && actualX < len(matrix[0]) {
			for i, ch := range label {
				actualY := labelStart + i + yOffset
				if actualY >= 0 && actualY < len(matrix) {
					matrix[actualY][actualX] = ch
				}
			}
		}
	} else {
		// Fallback to normal Set (won't work properly with lines)
		for i, ch := range label {
			pos := core.Point{X: x, Y: labelStart + i}
			if pos.Y >= minY && pos.Y <= maxY {
				c.Set(pos, ch)
			}
		}
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
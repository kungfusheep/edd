package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

// Design constants for balanced, aesthetic spacing
const (
	NodePadding       = 2  // Padding inside boxes
	NodeMinWidth      = 16 // Minimum width for aesthetic consistency
	NodeGap           = 8  // Space between nodes
	TrunkOffset       = 3  // Distance from node to vertical routing trunk
	ConnectionPadding = 2  // Space around connection lines
)

// Node represents a box in the diagram
type Node struct {
	ID     int      `json:"id"`
	X, Y   int      `json:"-"`    // Top-left position (calculated, not saved)
	Width  int      `json:"-"`    // Calculated from text + padding (not saved)
	Height int      `json:"-"`    // Calculated from text lines + padding (not saved)
	Text   []string `json:"text"` // Lines of text
}

// Connection represents a directed edge between nodes
type Connection struct {
	From int     `json:"from"` // Source node ID
	To   int     `json:"to"`   // Target node ID
	Path []Point `json:"-"`    // The actual route coordinates (calculated, not saved)
}

// Point represents a coordinate in the canvas
type Point struct {
	X, Y int
	Rune rune // The character to draw at this point
}

// Diagram holds all nodes and connections
type Diagram struct {
	Nodes       []Node       `json:"nodes"`
	Connections []Connection `json:"connections"`
}

// SavedDiagram represents the JSON structure for .edd files
type SavedDiagram struct {
	Diagram  `json:",inline"` // Embed the diagram directly
	Metadata DiagramMetadata  `json:"metadata"`
}

// DiagramMetadata holds diagram metadata
type DiagramMetadata struct {
	Name    string `json:"name"`
	Created string `json:"created"`
}

// Canvas represents the drawing surface
type Canvas struct {
	width, height int
	cells         []rune
}

// NewCanvas creates a new canvas with given dimensions
func NewCanvas(width, height int) *Canvas {
	canvas := &Canvas{
		width:  width,
		height: height,
		cells:  make([]rune, width*height),
	}
	canvas.Clear()
	return canvas
}

// Clear fills the canvas with spaces
func (c *Canvas) Clear() {
	for i := range c.cells {
		c.cells[i] = ' '
	}
}

// Resize changes the canvas dimensions
func (c *Canvas) Resize(width, height int) {
	c.width = width
	c.height = height
	c.cells = make([]rune, width*height)
	c.Clear()
}

// Set places a rune at the given position
func (c *Canvas) Set(x, y int, r rune) {
	if x >= 0 && x < c.width && y >= 0 && y < c.height {
		idx := y*c.width + x
		existing := c.cells[idx]
		// Apply combination rules if there's already a character
		if existing != ' ' && existing != 0 {
			c.cells[idx] = combineCharacters(existing, r)
		} else {
			c.cells[idx] = r
		}
	}
}

// Get returns the rune at the given position
func (c *Canvas) Get(x, y int) rune {
	if x >= 0 && x < c.width && y >= 0 && y < c.height {
		return c.cells[y*c.width+x]
	}
	return ' '
}

// String converts the canvas to a string representation
func (c *Canvas) String() string {
	var sb strings.Builder
	for y := 0; y < c.height; y++ {
		// Build the line
		var line strings.Builder
		for x := 0; x < c.width; x++ {
			line.WriteRune(c.cells[y*c.width+x])
		}
		// Trim trailing spaces from each line
		lineStr := strings.TrimRight(line.String(), " ")
		sb.WriteString(lineStr)
		if y < c.height-1 {
			sb.WriteRune('\n')
		}
	}
	return sb.String()
}

// combineCharacters merges two line-drawing characters into appropriate junction
func combineCharacters(existing, new rune) rune {
	// If either is a special character (arrow, box content), keep the new one
	if isSpecialChar(new) {
		return new
	}
	if isSpecialChar(existing) {
		return existing
	}

	// Build combination key
	key := string([]rune{existing, new})
	// Also check reverse combination
	reverseKey := string([]rune{new, existing})

	// Combination rules
	combinations := map[string]rune{
		// Horizontal meets vertical
		"─│": '┼', "│─": '┼',

		// Corners meet lines
		"╰│": '├', "│╰": '├', // Bottom-left corner + vertical = left T
		"╮│": '┤', "│╮": '┤', // Top-right corner + vertical = right T
		"╭│": '├', "│╭": '├', // Top-left corner + vertical = left T
		"╯│": '┤', "│╯": '┤', // Bottom-right corner + vertical = right T

		"╰─": '┴', "─╰": '┴', // Bottom-left corner + horizontal = bottom T
		"╯─": '┴', "─╯": '┴', // Bottom-right corner + horizontal = bottom T
		"╮─": '┬', "─╮": '┬', // Top-right corner + horizontal = top T
		"╭─": '┬', "─╭": '┬', // Top-left corner + horizontal = top T

		// T-junctions meet lines
		"├─": '┼', "─├": '┼',
		"┤─": '┼', "─┤": '┼',
		"┬│": '┼', "│┬": '┼',
		"┴│": '┼', "│┴": '┼',

		// Same character = keep it
		"──": '─', "││": '│',
		"╭╭": '╭', "╮╮": '╮',
		"╯╯": '╯', "╰╰": '╰',
	}

	if combined, ok := combinations[key]; ok {
		return combined
	}
	if combined, ok := combinations[reverseKey]; ok {
		return combined
	}

	// Default: keep the new character
	return new
}

// isSpecialChar returns true for characters that shouldn't be combined
func isSpecialChar(r rune) bool {
	switch r {
	case '▶', '▲', '▼', '◀': // Arrows
		return true
	case '├', '┤', '┬', '┴', '┼': // Already junctions
		return true
	default:
		// Check if it's a letter/number (box content)
		return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
	}
}

// DrawBox draws a node as a box with Unicode characters and rounded corners
func (c *Canvas) DrawBox(node Node) {
	// Top border with rounded corners
	c.Set(node.X, node.Y, '╭')
	for x := node.X + 1; x < node.X+node.Width-1; x++ {
		c.Set(x, node.Y, '─')
	}
	c.Set(node.X+node.Width-1, node.Y, '╮')

	// Middle lines with text
	for i := 0; i < node.Height-2; i++ {
		c.Set(node.X, node.Y+i+1, '│')
		c.Set(node.X+node.Width-1, node.Y+i+1, '│')

		// Draw text if available, centered
		if i < len(node.Text) {
			text := node.Text[i]
			// Calculate centering
			textLen := len(text)
			availableWidth := node.Width - 2*NodePadding - 2 // -2 for borders
			if textLen <= availableWidth {
				startX := node.X + 1 + NodePadding + (availableWidth-textLen)/2
				for j, ch := range text {
					c.Set(startX+j, node.Y+i+1, ch)
				}
			}
		}
	}

	// Bottom border with rounded corners
	c.Set(node.X, node.Y+node.Height-1, '╰')
	for x := node.X + 1; x < node.X+node.Width-1; x++ {
		c.Set(x, node.Y+node.Height-1, '─')
	}
	c.Set(node.X+node.Width-1, node.Y+node.Height-1, '╯')
}

// CalculateNodeSize determines the width and height for a node based on its text
func CalculateNodeSize(text []string) (width, height int) {
	maxLen := 0
	for _, line := range text {
		if len(line) > maxLen {
			maxLen = len(line)
		}
	}

	// Width: text + padding on both sides + borders
	width = maxLen + 2*NodePadding + 2
	if width < NodeMinWidth {
		width = NodeMinWidth
	}

	// Height: text lines + top and bottom borders
	height = len(text) + 2
	if height < 3 { // Minimum height for a box
		height = 3
	}

	return width, height
}

// ConnectionGroup represents connections that share routing space
type ConnectionGroup struct {
	From     int   // Source node ID
	Targets  []int // Target node IDs
	TrunkX   int   // X position of shared vertical trunk
	ExitY    int   // Y position where connections exit the source
	ExitSide Side  // Which side of the source node
}

// Side represents which side of a node a connection attaches to
type Side int

const (
	SideRight Side = iota
	SideBottom
	SideLeft
	SideTop
)

// RoutingPlan contains the overall routing strategy for all connections
type RoutingPlan struct {
	Groups      []ConnectionGroup
	Connections map[string][]Point // Key is "from,to"
}

// PlanRouting analyzes all connections and creates an optimal routing plan
func PlanRouting(nodes []Node, connections []Connection) RoutingPlan {
	plan := RoutingPlan{
		Connections: make(map[string][]Point),
	}

	// Group connections by source node
	sourceGroups := make(map[int][]int)
	for _, conn := range connections {
		sourceGroups[conn.From] = append(sourceGroups[conn.From], conn.To)
	}

	// Create connection groups with trunk positions
	for sourceID, targets := range sourceGroups {
		source := findNode(nodes, sourceID)
		group := ConnectionGroup{
			From:    sourceID,
			Targets: targets,
		}

		// Determine exit side and trunk position based on targets
		if len(targets) == 1 {
			// Single connection - route directly
			target := findNode(nodes, targets[0])
			group.ExitSide = determineExitSide(source, target)
		} else {
			// Multiple connections - check if it's a hub pattern
			// Hub pattern: connections go in different directions
			isHub := false
			var sides []Side
			for _, targetID := range targets {
				target := findNode(nodes, targetID)
				side := determineExitSide(source, target)
				sides = append(sides, side)
			}

			// Check if we have connections going to different sides
			firstSide := sides[0]
			for _, side := range sides[1:] {
				if side != firstSide {
					isHub = true
					break
				}
			}

			if isHub {
				// Hub pattern - each connection routes independently
				group.ExitSide = -1 // Special value to indicate hub routing
			} else {
				// All connections go same direction - use shared trunk
				group.ExitSide = firstSide
				if firstSide == SideRight {
					group.TrunkX = source.X + source.Width + TrunkOffset
				} else if firstSide == SideLeft {
					group.TrunkX = source.X - TrunkOffset
				}
				group.ExitY = source.Y + source.Height/2
			}
		}

		plan.Groups = append(plan.Groups, group)
	}

	// First pass: detect bidirectional connections
	bidirectional := make(map[string]bool)
	for i, conn := range connections {
		for j, other := range connections {
			if i != j && conn.From == other.To && conn.To == other.From {
				key1 := fmt.Sprintf("%d,%d", conn.From, conn.To)
				key2 := fmt.Sprintf("%d,%d", other.From, other.To)
				bidirectional[key1] = true
				bidirectional[key2] = true
			}
		}
	}

	// Route each connection using group information
	for _, conn := range connections {
		key := fmt.Sprintf("%d,%d", conn.From, conn.To)
		reverseKey := fmt.Sprintf("%d,%d", conn.To, conn.From)
		isBidirectional := bidirectional[key] || bidirectional[reverseKey]
		plan.Connections[key] = routeConnectionWithGroups(nodes, conn, plan, isBidirectional)
	}

	return plan
}

// findNode finds a node by ID
func findNode(nodes []Node, id int) Node {
	for _, n := range nodes {
		if n.ID == id {
			return n
		}
	}
	return Node{}
}

// determineExitSide figures out which side of the source to exit from
func determineExitSide(from, to Node) Side {
	// Simple heuristic: exit from the side closest to target
	if to.X > from.X+from.Width {
		return SideRight
	} else if to.X+to.Width < from.X {
		return SideLeft
	} else if to.Y > from.Y+from.Height {
		return SideBottom
	} else {
		return SideTop
	}
}

// routeConnectionWithGroups creates the path for a single connection using group information
func routeConnectionWithGroups(nodes []Node, conn Connection, plan RoutingPlan, isBidirectional bool) []Point {
	from := findNode(nodes, conn.From)
	to := findNode(nodes, conn.To)

	// Handle self-connections (loops)
	if conn.From == conn.To {
		return routeSelfConnection(from)
	}

	// Handle bidirectional connections
	if isBidirectional && from.Y == to.Y {
		// For bidirectional horizontal connections, offset one of them
		if conn.From < conn.To {
			// First connection uses the first content line
			if from.X < to.X {
				return routeHorizontalAtLine(from, to, from.Y+1) // First content line
			} else {
				return routeHorizontalLeft(from, to)
			}
		} else {
			// Second connection - going left, use second content line
			var path []Point

			if from.X > to.X {
				// Going left - use second content line if box is tall enough
				if from.Height > 3 {
					// Multi-line box - use second content line
					startY := from.Y + from.Height - 2 // Second to last line (second content line)

					// Place │ in the content area (13 spaces from left for a 16-wide box)
					sourceContentPos := from.X + from.Width - 3

					// Place ┤ on the target box right border
					targetBorderX := to.X + to.Width - 1

					// Draw the path going left
					path = append(path, Point{X: sourceContentPos, Y: startY, Rune: '│'})
					path = append(path, Point{X: sourceContentPos - 1, Y: startY, Rune: '◀'})

					// Horizontal line going left from source to target
					for x := sourceContentPos - 2; x > targetBorderX; x-- {
						path = append(path, Point{X: x, Y: startY, Rune: '─'})
					}

					// End junction on target box border
					path = append(path, Point{X: targetBorderX, Y: startY, Rune: '┤'})
				} else {
					// Single content line - place on bottom border
					startY := from.Y + from.Height - 1
					startX := from.X
					endX := to.X + to.Width - 1

					path = append(path, Point{X: startX, Y: startY, Rune: '┤'})
					path = append(path, Point{X: startX - 1, Y: startY, Rune: '◀'})
					for x := startX - 2; x > endX; x-- {
						path = append(path, Point{X: x, Y: startY, Rune: '─'})
					}
					path = append(path, Point{X: endX, Y: startY, Rune: '┤'})
				}
			}
			return path
		}
	}

	// Find the group this connection belongs to
	var group *ConnectionGroup
	for i := range plan.Groups {
		if plan.Groups[i].From == conn.From {
			group = &plan.Groups[i]
			break
		}
	}

	// If multiple targets, check the routing strategy
	if group != nil && len(group.Targets) > 1 {
		if group.ExitSide == -1 {
			// Hub pattern - use diagonal routing for all connections
			// since they go in different directions
			return routeDiagonalSimple(from, to)
		} else {
			// Shared trunk routing
			return routeWithSharedTrunk(from, to, group, nodes)
		}
	}

	// Simple routing for single connections
	if from.Y == to.Y && from.X < to.X {
		// Horizontal
		return routeHorizontalSimple(from, to)
	} else if from.Y == to.Y && from.X > to.X {
		// Horizontal but going left
		return routeHorizontalLeft(from, to)
	} else if from.X == to.X && from.Y != to.Y {
		// Vertical alignment
		return routeVerticalSimple(from, to)
	}

	// Default to diagonal routing
	return routeDiagonalSimple(from, to)
}

// routeWithSharedTrunk routes a connection that shares a trunk with others
func routeWithSharedTrunk(from, to Node, group *ConnectionGroup, nodes []Node) []Point {
	var path []Point

	exitX := from.X + from.Width - 1
	exitY := from.Y + from.Height/2

	// Determine if this is the first connection (needs the junction)
	isFirst := to.ID == group.Targets[0]

	if isFirst {
		// First connection draws the exit junction and initial trunk
		path = append(path, Point{X: exitX, Y: exitY, Rune: '├'})

		// Horizontal to trunk
		for x := exitX + 1; x < group.TrunkX; x++ {
			path = append(path, Point{X: x, Y: exitY, Rune: '─'})
		}

		// Add split junction if there are more connections
		if len(group.Targets) > 1 {
			path = append(path, Point{X: group.TrunkX, Y: exitY, Rune: '┬'})
		}
	}

	// Now route from trunk to target
	if to.Y == from.Y {
		// Same level - continue horizontally
		for x := group.TrunkX + 1; x < to.X; x++ {
			path = append(path, Point{X: x, Y: exitY, Rune: '─'})
		}
		path = append(path, Point{X: to.X - 1, Y: to.Y + to.Height/2, Rune: '▶'})
	} else {
		// Different level - go down from trunk
		startY := exitY
		if !isFirst {
			startY = exitY + 1 // Start below the horizontal line
		}

		// Vertical segment
		targetY := to.Y + to.Height/2
		if targetY > startY {
			// Going down
			for y := startY; y < targetY; y++ {
				path = append(path, Point{X: group.TrunkX, Y: y, Rune: '│'})
			}
			// Turn towards target
			path = append(path, Point{X: group.TrunkX, Y: targetY, Rune: '╰'})
		} else {
			// Going up
			for y := startY; y > targetY; y-- {
				path = append(path, Point{X: group.TrunkX, Y: y, Rune: '│'})
			}
			// Turn towards target
			path = append(path, Point{X: group.TrunkX, Y: targetY, Rune: '╭'})
		}

		// Horizontal to target
		for x := group.TrunkX + 1; x < to.X; x++ {
			path = append(path, Point{X: x, Y: targetY, Rune: '─'})
		}
		path = append(path, Point{X: to.X - 1, Y: targetY, Rune: '▶'})
	}

	return path
}

// routeHorizontalSimple creates a simple horizontal path
func routeHorizontalSimple(from, to Node) []Point {
	var path []Point

	startX := from.X + from.Width - 1
	startY := from.Y + from.Height/2
	endX := to.X
	endY := to.Y + to.Height/2

	// Exit junction
	path = append(path, Point{X: startX, Y: startY, Rune: '├'})

	// Horizontal line
	for x := startX + 1; x < endX; x++ {
		path = append(path, Point{X: x, Y: startY, Rune: '─'})
	}

	// Arrow should be just before the target box
	if endX > 0 {
		path = append(path, Point{X: endX - 1, Y: endY, Rune: '▶'})
	}

	return path
}

// routeHorizontalAtLine creates a horizontal path at a specific Y line
func routeHorizontalAtLine(from, to Node, lineY int) []Point {
	var path []Point

	startX := from.X + from.Width - 1
	endX := to.X

	// Exit junction
	path = append(path, Point{X: startX, Y: lineY, Rune: '├'})

	// Horizontal line
	for x := startX + 1; x < endX; x++ {
		path = append(path, Point{X: x, Y: lineY, Rune: '─'})
	}

	// Arrow should be just before the target box
	if endX > 0 {
		path = append(path, Point{X: endX - 1, Y: lineY, Rune: '▶'})
	}

	return path
}

// routeBidirectional handles connections that go both ways between nodes
func routeBidirectional(from, to Node, isUpper bool) []Point {
	var path []Point

	// For horizontal bidirectional connections at same Y level
	if from.Y == to.Y {
		// Both connections use the same Y line (middle of the box)
		startY := from.Y + 1

		if from.X < to.X {
			// Going right
			startX := from.X + from.Width - 1
			endX := to.X

			path = append(path, Point{X: startX, Y: startY, Rune: '├'})
			for x := startX + 1; x < endX; x++ {
				path = append(path, Point{X: x, Y: startY, Rune: '─'})
			}
			path = append(path, Point{X: endX - 1, Y: startY, Rune: '▶'})
		} else {
			// Going left
			startX := from.X
			endX := to.X + to.Width - 1

			path = append(path, Point{X: startX, Y: startY, Rune: '┤'})
			path = append(path, Point{X: startX - 1, Y: startY, Rune: '◀'})
			for x := startX - 2; x > endX; x-- {
				path = append(path, Point{X: x, Y: startY, Rune: '─'})
			}
			path = append(path, Point{X: endX, Y: startY, Rune: '├'})
		}
	}

	return path
}

// routeBackwardBelowWithContext creates a backward path that goes below all boxes to avoid collisions
func routeBackwardBelowWithContext(from, to Node, allNodes []Node) []Point {
	var path []Point

	// Exit from bottom center of source box
	startX := from.X + from.Width/2
	startY := from.Y + from.Height - 1

	// Enter at bottom center of target box (1 line below the box)
	endX := to.X + to.Width/2
	endY := to.Y + to.Height

	// Find the maximum Y position of all boxes to route below them
	maxY := 0
	for _, node := range allNodes {
		nodeBottom := node.Y + node.Height
		if nodeBottom > maxY {
			maxY = nodeBottom
		}
	}

	// Route 3 lines below the lowest box to ensure clearance
	routeY := maxY + 3

	// Exit from bottom
	path = append(path, Point{X: startX, Y: startY, Rune: '┬'})

	// Go down
	for y := startY + 1; y < routeY; y++ {
		path = append(path, Point{X: startX, Y: y, Rune: '│'})
	}

	// Turn left at the bottom
	if endX < startX {
		path = append(path, Point{X: startX, Y: routeY, Rune: '╯'})
		// Go left
		for x := startX - 1; x > endX; x-- {
			path = append(path, Point{X: x, Y: routeY, Rune: '─'})
		}
	} else {
		// Turn right if target is to the right
		path = append(path, Point{X: startX, Y: routeY, Rune: '╰'})
		// Go right
		for x := startX + 1; x < endX; x++ {
			path = append(path, Point{X: x, Y: routeY, Rune: '─'})
		}
	}

	// Turn up towards target
	path = append(path, Point{X: endX, Y: routeY, Rune: '╰'})

	// Go up to target
	for y := routeY - 1; y > endY; y-- {
		path = append(path, Point{X: endX, Y: y, Rune: '│'})
	}

	// Enter target from bottom
	path = append(path, Point{X: endX, Y: endY, Rune: '▲'})

	return path
}

// routeBackwardBelow creates a backward path that goes below boxes to avoid collisions
func routeBackwardBelow(from, to Node) []Point {
	var path []Point

	// Exit from bottom center of source box
	startX := from.X + from.Width/2
	startY := from.Y + from.Height - 1

	// Enter at bottom center of target box
	endX := to.X + to.Width/2
	endY := to.Y + to.Height - 1

	// Route below all boxes - use a safe Y that should be below any reasonable layout
	routeY := 20 // Fixed Y position that should be below most diagrams

	// Exit from bottom
	path = append(path, Point{X: startX, Y: startY, Rune: '┬'})

	// Go down
	for y := startY + 1; y < routeY; y++ {
		path = append(path, Point{X: startX, Y: y, Rune: '│'})
	}

	// Turn left at the bottom
	if endX < startX {
		path = append(path, Point{X: startX, Y: routeY, Rune: '╯'})
		// Go left
		for x := startX - 1; x > endX; x-- {
			path = append(path, Point{X: x, Y: routeY, Rune: '─'})
		}
	} else {
		// Turn right if target is to the right
		path = append(path, Point{X: startX, Y: routeY, Rune: '╰'})
		// Go right
		for x := startX + 1; x < endX; x++ {
			path = append(path, Point{X: x, Y: routeY, Rune: '─'})
		}
	}

	// Turn up towards target
	path = append(path, Point{X: endX, Y: routeY, Rune: '╰'})

	// Go up to target
	for y := routeY - 1; y > endY; y-- {
		path = append(path, Point{X: endX, Y: y, Rune: '│'})
	}

	// Enter target from bottom
	path = append(path, Point{X: endX, Y: endY, Rune: '▲'})

	return path
}

// routeHorizontalLeft creates a horizontal path going left
func routeHorizontalLeft(from, to Node) []Point {
	var path []Point

	startX := from.X
	startY := from.Y + from.Height/2
	endX := to.X + to.Width - 1
	endY := to.Y + to.Height/2

	// For backward connections, we need to route around boxes
	// If nodes are on the same line and close together, do simple routing
	if startY == endY && startX-endX < 20 {
		// Exit junction on left
		path = append(path, Point{X: startX, Y: startY, Rune: '┤'})

		// Horizontal line
		for x := startX - 1; x > endX+1; x-- {
			path = append(path, Point{X: x, Y: startY, Rune: '─'})
		}

		// Arrow pointing left at the target
		path = append(path, Point{X: endX + 1, Y: endY, Rune: '◀'})
	} else {
		// Multi-hop backward connection - route below the boxes
		// Start by going down from left side
		path = append(path, Point{X: startX, Y: startY, Rune: '┤'})
		path = append(path, Point{X: startX - 1, Y: startY, Rune: '─'})
		path = append(path, Point{X: startX - 2, Y: startY, Rune: '╮'})

		// Go down below the boxes
		routeY := from.Y + from.Height + 2 // 2 lines below the box
		for y := startY + 1; y < routeY; y++ {
			path = append(path, Point{X: startX - 2, Y: y, Rune: '│'})
		}

		// Turn left at the bottom
		path = append(path, Point{X: startX - 2, Y: routeY, Rune: '╯'})

		// Go left below all boxes
		targetX := to.X + to.Width/2
		for x := startX - 3; x > targetX; x-- {
			path = append(path, Point{X: x, Y: routeY, Rune: '─'})
		}

		// Turn up towards target
		path = append(path, Point{X: targetX, Y: routeY, Rune: '╰'})

		// Go up to target
		targetY := to.Y + to.Height - 1
		for y := routeY - 1; y > targetY; y-- {
			path = append(path, Point{X: targetX, Y: y, Rune: '│'})
		}

		// Enter target from bottom
		path = append(path, Point{X: targetX, Y: targetY, Rune: '▲'})
	}

	return path
}

// routeDiagonalSimple creates a simple L-shaped path
func routeDiagonalSimple(from, to Node) []Point {
	var path []Point

	// Debug - uncomment to trace routing
	// fmt.Printf("DIAGONAL: from=(%d,%d,%dx%d) to=(%d,%d,%dx%d)\n",
	//     from.X, from.Y, from.Width, from.Height, to.X, to.Y, to.Width, to.Height)

	// Exit from the side closest to target
	if to.X > from.X+from.Width {
		// Target is to the right - exit right, go right then up/down
		startX := from.X + from.Width - 1
		startY := from.Y + from.Height/2
		targetX := to.X
		targetY := to.Y + to.Height/2

		// Exit junction
		path = append(path, Point{X: startX, Y: startY, Rune: '├'})

		// Go right to a point between nodes
		trunkX := from.X + from.Width + TrunkOffset
		for x := startX + 1; x <= trunkX; x++ {
			path = append(path, Point{X: x, Y: startY, Rune: '─'})
		}

		// Turn up or down
		if targetY < startY {
			// Going up
			path = append(path, Point{X: trunkX, Y: startY, Rune: '╰'})
			for y := startY - 1; y > targetY; y-- {
				path = append(path, Point{X: trunkX, Y: y, Rune: '│'})
			}
			path = append(path, Point{X: trunkX, Y: targetY, Rune: '╭'})
		} else if targetY > startY {
			// Going down
			path = append(path, Point{X: trunkX, Y: startY, Rune: '╭'})
			for y := startY + 1; y < targetY; y++ {
				path = append(path, Point{X: trunkX, Y: y, Rune: '│'})
			}
			path = append(path, Point{X: trunkX, Y: targetY, Rune: '╰'})
		}

		// Go to target
		for x := trunkX + 1; x < targetX-1; x++ {
			path = append(path, Point{X: x, Y: targetY, Rune: '─'})
		}
		// Arrow should be just before the target box
		if targetX-1 > trunkX {
			// There's space between corner and target - place arrow just before target
			path = append(path, Point{X: targetX - 1, Y: targetY, Rune: '▶'})
		} else {
			// Corner is adjacent to target - replace corner with arrow
			path[len(path)-1] = Point{X: trunkX, Y: targetY, Rune: '▶'}
		}

	} else if to.X+to.Width < from.X {
		// Target is to the left - exit left
		startX := from.X
		startY := from.Y + from.Height/2
		targetX := to.X + to.Width - 1
		targetY := to.Y + to.Height/2

		// Exit junction
		path = append(path, Point{X: startX, Y: startY, Rune: '┤'})

		// Go left to trunk position
		trunkX := from.X - TrunkOffset
		for x := startX - 1; x >= trunkX; x-- {
			path = append(path, Point{X: x, Y: startY, Rune: '─'})
		}

		// Turn up or down
		if targetY < startY {
			// Going up
			path = append(path, Point{X: trunkX, Y: startY, Rune: '╯'})
			for y := startY - 1; y > targetY; y-- {
				path = append(path, Point{X: trunkX, Y: y, Rune: '│'})
			}
			path = append(path, Point{X: trunkX, Y: targetY, Rune: '╮'})
		} else if targetY > startY {
			// Going down
			path = append(path, Point{X: trunkX, Y: startY, Rune: '╮'})
			for y := startY + 1; y < targetY; y++ {
				path = append(path, Point{X: trunkX, Y: y, Rune: '│'})
			}
			path = append(path, Point{X: trunkX, Y: targetY, Rune: '╯'})
		}

		// Go to target
		for x := trunkX - 1; x > targetX+to.Width; x-- {
			path = append(path, Point{X: x, Y: targetY, Rune: '─'})
		}
		// Arrow should be just after the target box (at the right border)
		if targetX+to.Width < trunkX {
			// There's space between corner and target - place arrow just after target
			path = append(path, Point{X: to.X + to.Width, Y: targetY, Rune: '◀'})
		} else {
			// Corner is adjacent to target - replace corner with arrow
			path[len(path)-1] = Point{X: trunkX, Y: targetY, Rune: '◀'}
		}

	} else {
		// Target is above or below - exit from top or bottom
		if to.Y < from.Y {
			// Exit top
			startX := from.X + from.Width/2
			startY := from.Y

			path = append(path, Point{X: startX, Y: startY, Rune: '┴'})
			// TODO: Implement vertical routing
		} else {
			// Exit bottom
			startX := from.X + from.Width/2
			startY := from.Y + from.Height - 1

			path = append(path, Point{X: startX, Y: startY, Rune: '┬'})
			// TODO: Implement vertical routing
			// For now, just exit from the side for self-connections
			if from.X == to.X && from.Y == to.Y {
				// Self-connection - exit from right side
				return routeHorizontalSimple(from, to)
			}
		}
	}

	return path
}

// DrawConnection draws a connection path
func (c *Canvas) DrawConnection(path []Point) {
	for _, p := range path {
		c.Set(p.X, p.Y, p.Rune)
	}
}

// Render draws the entire diagram to the canvas using layered layout
func (c *Canvas) Render(diagram Diagram) {
	// Use new layered layout system
	layout := NewLayeredLayout()
	positioned := layout.CalculateLayout(diagram.Nodes, diagram.Connections)

	// Calculate required canvas size based on positioned nodes
	maxX, maxY := 0, 0
	for _, node := range positioned {
		if node.X+node.Width > maxX {
			maxX = node.X + node.Width
		}
		if node.Y+node.Height > maxY {
			maxY = node.Y + node.Height
		}
	}

	// Auto-resize canvas to match current terminal size (throttled check)
	// This will be handled by the Editor, not the Canvas directly

	// Draw all nodes in their calculated positions
	for _, node := range positioned {
		c.DrawBox(node)
	}

	// Draw connections using simple orthogonal routing
	nodeMap := make(map[int]Node)
	for _, node := range positioned {
		nodeMap[node.ID] = node
	}

	for _, conn := range diagram.Connections {
		from := nodeMap[conn.From]
		to := nodeMap[conn.To]
		path := SimpleOrthogonalRouteWithContext(from, to, positioned)
		c.DrawConnection(path)
	}
}

// routeSelfConnection creates a loop from a node back to itself
func routeSelfConnection(node Node) []Point {
	var path []Point

	// Exit from the right side at content line
	exitX := node.X + node.Width - 1
	exitY := node.Y + 1 // Content line (Y=1 for height=3 box)

	// Loop coordinates based on expected output
	rightX := exitX + 2   // Two spaces to the right
	bottomY := node.Y + 4 // Two lines below the box
	leftX := node.X + 2   // Two spaces from left edge
	upY := node.Y + 3     // One line below the box

	// Exit junction
	path = append(path, Point{X: exitX, Y: exitY, Rune: '├'})

	// Go right
	path = append(path, Point{X: exitX + 1, Y: exitY, Rune: '─'})
	path = append(path, Point{X: rightX, Y: exitY, Rune: '╮'})

	// Go down
	for y := exitY + 1; y < bottomY; y++ {
		path = append(path, Point{X: rightX, Y: y, Rune: '│'})
	}

	// Go left (bottom of loop)
	path = append(path, Point{X: rightX, Y: bottomY, Rune: '╯'})
	for x := rightX - 1; x > leftX; x-- {
		path = append(path, Point{X: x, Y: bottomY, Rune: '─'})
	}

	// Go up to complete loop
	path = append(path, Point{X: leftX, Y: bottomY, Rune: '╰'})
	path = append(path, Point{X: leftX, Y: upY, Rune: '▲'})

	return path
}

// routeVerticalSimple creates a vertical path between aligned nodes
func routeVerticalSimple(from, to Node) []Point {
	var path []Point

	// Both nodes have same X alignment (from.X == to.X)
	centerX := from.X + from.Width/2

	if from.Y < to.Y {
		// Going down
		startY := from.Y + from.Height - 1 // Bottom edge
		endY := to.Y                       // Top edge of target

		// Exit from bottom
		path = append(path, Point{X: centerX, Y: startY, Rune: '┬'})

		// Vertical line down
		for y := startY + 1; y < endY; y++ {
			path = append(path, Point{X: centerX, Y: y, Rune: '│'})
		}

		// Arrow pointing down to target
		path = append(path, Point{X: centerX, Y: endY - 1, Rune: '▼'})

	} else {
		// Going up
		startY := from.Y             // Top edge
		endY := to.Y + to.Height - 1 // Bottom edge of target

		// Exit from top
		path = append(path, Point{X: centerX, Y: startY, Rune: '┴'})

		// Vertical line up
		for y := startY - 1; y > endY; y-- {
			path = append(path, Point{X: centerX, Y: y, Rune: '│'})
		}

		// Arrow pointing up to target
		path = append(path, Point{X: centerX, Y: endY + 1, Rune: '▲'})
	}

	return path
}

// ========================================
// NEW LAYERED LAYOUT SYSTEM
// ========================================

// Layer represents a horizontal layer in the diagram
type Layer struct {
	Nodes []int // Node IDs in this layer
	Y     int   // Y position of this layer
}

// LayeredLayout performs automatic layout using layered approach
type LayeredLayout struct {
	Layers       []Layer
	NodeSpacing  int // Horizontal space between nodes
	LayerSpacing int // Vertical space between layers
	NodeWidth    int // Standard node width
	NodeHeight   int // Standard node height
}

// NewLayeredLayout creates a layout with reasonable defaults
func NewLayeredLayout() *LayeredLayout {
	return &LayeredLayout{
		NodeSpacing:  4, // 4 chars between nodes
		LayerSpacing: 1, // 1 line between layers
		NodeWidth:    12,
		NodeHeight:   3,
	}
}

// CalculateLayout performs topological sort and assigns positions using column-based approach
func (l *LayeredLayout) CalculateLayout(nodes []Node, connections []Connection) []Node {
	// Step 1: Calculate box sizes based on text content
	result := make([]Node, len(nodes))
	nodeMap := make(map[int]*Node)
	for i := range nodes {
		nodeMap[nodes[i].ID] = &result[i]
		result[i] = nodes[i] // Copy original node data
		// Calculate dynamic size
		width, height := CalculateNodeSize(result[i].Text)
		result[i].Width = width
		result[i].Height = height
	}

	// Step 2: Find connected components
	components := l.findConnectedComponents(nodes, connections)

	// Step 3: Layout each component separately and position side-by-side
	currentX := 0
	for _, component := range components {
		componentWidth := l.layoutComponent(component, connections, nodeMap, currentX)
		currentX += componentWidth + 4 // Add spacing between components
	}

	return result
}

// findConnectedComponents identifies separate subgraphs using DFS
func (l *LayeredLayout) findConnectedComponents(nodes []Node, connections []Connection) [][]int {
	// Build bidirectional adjacency list (undirected graph for component detection)
	adjacent := make(map[int][]int)
	for _, node := range nodes {
		adjacent[node.ID] = []int{}
	}
	for _, conn := range connections {
		adjacent[conn.From] = append(adjacent[conn.From], conn.To)
		adjacent[conn.To] = append(adjacent[conn.To], conn.From)
	}

	// Find components using DFS
	visited := make(map[int]bool)
	components := [][]int{}

	for _, node := range nodes {
		if !visited[node.ID] {
			component := []int{}
			stack := []int{node.ID}

			for len(stack) > 0 {
				current := stack[len(stack)-1]
				stack = stack[:len(stack)-1]

				if visited[current] {
					continue
				}
				visited[current] = true
				component = append(component, current)

				for _, neighbor := range adjacent[current] {
					if !visited[neighbor] {
						stack = append(stack, neighbor)
					}
				}
			}

			// Sort component by ID for consistent ordering
			sort.Slice(component, func(i, j int) bool {
				return component[i] < component[j]
			})

			components = append(components, component)
		}
	}

	return components
}

// layoutComponent lays out a single connected component at the given X offset
func (l *LayeredLayout) layoutComponent(componentNodes []int, connections []Connection, nodeMap map[int]*Node, startX int) int {
	// Filter connections for this component only
	componentSet := make(map[int]bool)
	for _, nodeID := range componentNodes {
		componentSet[nodeID] = true
	}

	componentConnections := []Connection{}
	for _, conn := range connections {
		if componentSet[conn.From] && componentSet[conn.To] {
			componentConnections = append(componentConnections, conn)
		}
	}

	// Use the original layout algorithm for this component
	return l.layoutSingleComponent(componentNodes, componentConnections, nodeMap, startX)
}

// layoutSingleComponent applies the original algorithm to a single component
func (l *LayeredLayout) layoutSingleComponent(nodeIDs []int, connections []Connection, nodeMap map[int]*Node, startX int) int {
	// Build adjacency list for topological sort
	graph := make(map[int][]int)
	inDegree := make(map[int]int)
	backwardEdges := make(map[string]bool)

	for _, nodeID := range nodeIDs {
		graph[nodeID] = []int{}
		inDegree[nodeID] = 0
	}

	// First pass: identify backward edges
	for _, conn := range connections {
		if conn.To < conn.From {
			key := fmt.Sprintf("%d->%d", conn.From, conn.To)
			backwardEdges[key] = true
		}
	}

	// Second pass: build graph excluding backward edges
	for _, conn := range connections {
		key := fmt.Sprintf("%d->%d", conn.From, conn.To)
		if !backwardEdges[key] {
			graph[conn.From] = append(graph[conn.From], conn.To)
			inDegree[conn.To]++
		}
	}

	// Topological sort to determine columns
	columns := [][]int{}
	remaining := make(map[int]bool)
	for _, nodeID := range nodeIDs {
		remaining[nodeID] = true
	}

	for len(remaining) > 0 {
		// Find nodes with no incoming edges
		candidates := []int{}
		for nodeID := range remaining {
			if inDegree[nodeID] == 0 {
				candidates = append(candidates, nodeID)
			}
		}

		if len(candidates) == 0 {
			// Cycle detected - break it
			for nodeID := range remaining {
				candidates = append(candidates, nodeID)
				break
			}
		}

		// Sort by ID for consistent ordering
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i] < candidates[j]
		})

		columns = append(columns, candidates)

		// Remove these nodes and update in-degrees
		for _, nodeID := range candidates {
			delete(remaining, nodeID)
			for _, neighbor := range graph[nodeID] {
				inDegree[neighbor]--
			}
		}
	}

	// Calculate column widths
	columnWidths := make([]int, len(columns))
	for colIdx, column := range columns {
		maxWidth := 0
		for _, nodeID := range column {
			node := nodeMap[nodeID]
			if node.Width > maxWidth {
				maxWidth = node.Width
			}
		}
		columnWidths[colIdx] = maxWidth
	}

	// Assign X positions to columns (offset by startX)
	columnStartX := make([]int, len(columns))
	currentX := startX
	for colIdx := range columns {
		columnStartX[colIdx] = currentX
		currentX += columnWidths[colIdx] + l.NodeSpacing
	}

	// Assign Y positions within each column, considering connections
	for colIdx, column := range columns {
		// Check if this column has 1-to-1 connections (should align)
		// vs hub-and-spoke connections (should distribute)
		shouldAlign := make(map[int]bool)

		if colIdx > 0 {
			// Count outgoing connections from previous column nodes
			prevColumnOutgoing := make(map[int]int)
			for _, conn := range connections {
				for _, prevNodeID := range columns[colIdx-1] {
					if conn.From == prevNodeID {
						prevColumnOutgoing[prevNodeID]++
					}
				}
			}

			// Check each node in current column
			for _, nodeID := range column {
				for _, conn := range connections {
					if conn.To == nodeID {
						sourceNodeID := conn.From
						// Only align if source has exactly 1 outgoing connection
						if prevColumnOutgoing[sourceNodeID] == 1 {
							shouldAlign[nodeID] = true
						}
						break
					}
				}
			}
		}

		// Assign positions
		unalignedIndex := 0
		for _, nodeID := range column {
			node := nodeMap[nodeID]
			node.X = columnStartX[colIdx]

			if shouldAlign[nodeID] {
				// Find the source node's Y position for alignment
				for _, conn := range connections {
					if conn.To == nodeID {
						sourceNode := nodeMap[conn.From]
						node.Y = sourceNode.Y
						break
					}
				}
			} else {
				// Use vertical distribution
				node.Y = unalignedIndex * (node.Height + l.LayerSpacing)
				unalignedIndex++
			}
		}
	}

	// Return the width of this component
	return currentX - startX
}

func SimpleOrthogonalRouteWithContext(from, to Node, allNodes []Node) []Point {
	// Check if this is a backward connection
	if to.X+to.Width < from.X {
		// Target is to the left of source
		// If it's a multi-hop backward connection, route from bottom
		if from.X-(to.X+to.Width) > 20 {
			return routeBackwardBelowWithContext(from, to, allNodes)
		}
		// Otherwise use simple left routing
		return routeHorizontalLeft(from, to)
	}

	var path []Point

	// Exit from center-right of source box border
	startX := from.X + from.Width - 1 // On the border, not past it
	startY := from.Y + from.Height/2

	// Enter at center-left of target
	endX := to.X - 1
	endY := to.Y + to.Height/2

	// Simple L-shaped route
	if startY == endY {
		// Same level - check for box collisions on horizontal path
		hasCollision := false
		for _, node := range allNodes {
			if node.ID == from.ID || node.ID == to.ID {
				continue // Skip source and target nodes
			}
			// Check if the horizontal line at startY intersects with this box
			if startY >= node.Y && startY < node.Y+node.Height &&
				startX < node.X+node.Width && endX > node.X {
				hasCollision = true
				break
			}
		}

		if !hasCollision {
			// No collision - straight horizontal
			path = append(path, Point{X: startX, Y: startY, Rune: '├'})
			for x := startX + 1; x < endX; x++ {
				path = append(path, Point{X: x, Y: startY, Rune: '─'})
			}
			path = append(path, Point{X: endX, Y: endY, Rune: '▶'})
		} else {
			// Collision detected - route below the obstructing boxes
			maxY := 0
			for _, node := range allNodes {
				nodeBottom := node.Y + node.Height
				if nodeBottom > maxY {
					maxY = nodeBottom
				}
			}
			safeY := maxY + 1

			// Go down to safe Y
			path = append(path, Point{X: startX, Y: startY, Rune: '├'})
			path = append(path, Point{X: startX + 1, Y: startY, Rune: '─'})
			path = append(path, Point{X: startX + 2, Y: startY, Rune: '╮'})
			for y := startY + 1; y < safeY; y++ {
				path = append(path, Point{X: startX + 2, Y: y, Rune: '│'})
			}
			path = append(path, Point{X: startX + 2, Y: safeY, Rune: '╰'})

			// Horizontal to target at safe Y
			for x := startX + 3; x < endX; x++ {
				path = append(path, Point{X: x, Y: safeY, Rune: '─'})
			}

			// Go up to target Y
			path = append(path, Point{X: endX, Y: safeY, Rune: '╮'})
			for y := safeY - 1; y > endY; y-- {
				path = append(path, Point{X: endX, Y: y, Rune: '│'})
			}
			path = append(path, Point{X: endX, Y: endY, Rune: '▶'})
		}
	} else {
		// Different levels - L shape
		midX := startX + 2 // Small horizontal segment

		// Exit horizontally
		path = append(path, Point{X: startX, Y: startY, Rune: '├'})
		path = append(path, Point{X: startX + 1, Y: startY, Rune: '─'})

		// Turn down/up
		if endY > startY {
			// Going down
			path = append(path, Point{X: midX, Y: startY, Rune: '╮'})
			for y := startY + 1; y < endY; y++ {
				path = append(path, Point{X: midX, Y: y, Rune: '│'})
			}
			path = append(path, Point{X: midX, Y: endY, Rune: '╰'})
		} else {
			// Going up
			path = append(path, Point{X: midX, Y: startY, Rune: '╮'})
			for y := startY - 1; y > endY; y-- {
				path = append(path, Point{X: midX, Y: y, Rune: '│'})
			}
			path = append(path, Point{X: midX, Y: endY, Rune: '╰'})
		}

		// Horizontal to target
		for x := midX + 1; x < endX; x++ {
			path = append(path, Point{X: x, Y: endY, Rune: '─'})
		}
		path = append(path, Point{X: endX, Y: endY, Rune: '▶'})
	}

	return path
}

func SimpleOrthogonalRoute(from, to Node) []Point {
	// Check if this is a backward connection
	if to.X+to.Width < from.X {
		// Target is to the left of source
		// If it's a multi-hop backward connection, route from bottom
		if from.X-(to.X+to.Width) > 20 {
			return routeBackwardBelow(from, to)
		}
		// Otherwise use simple left routing
		return routeHorizontalLeft(from, to)
	}

	var path []Point

	// Exit from center-right of source box border
	startX := from.X + from.Width - 1 // On the border, not past it
	startY := from.Y + from.Height/2

	// Enter at center-left of target
	endX := to.X - 1
	endY := to.Y + to.Height/2

	// Simple L-shaped route
	if startY == endY {
		// Same level - straight horizontal
		path = append(path, Point{X: startX, Y: startY, Rune: '├'})
		for x := startX + 1; x < endX; x++ {
			path = append(path, Point{X: x, Y: startY, Rune: '─'})
		}
		path = append(path, Point{X: endX, Y: endY, Rune: '▶'})
	} else {
		// Different levels - L shape
		midX := startX + 2 // Small horizontal segment

		// Exit horizontally
		path = append(path, Point{X: startX, Y: startY, Rune: '├'})
		path = append(path, Point{X: startX + 1, Y: startY, Rune: '─'})

		// Turn down/up
		if endY > startY {
			// Going down
			path = append(path, Point{X: midX, Y: startY, Rune: '╮'})
			for y := startY + 1; y < endY; y++ {
				path = append(path, Point{X: midX, Y: y, Rune: '│'})
			}
			path = append(path, Point{X: midX, Y: endY, Rune: '╰'})
		} else {
			// Going up
			path = append(path, Point{X: midX, Y: startY, Rune: '╯'})
			for y := startY - 1; y > endY; y-- {
				path = append(path, Point{X: midX, Y: y, Rune: '│'})
			}
			path = append(path, Point{X: midX, Y: endY, Rune: '╮'})
		}

		// Horizontal to target
		for x := midX + 1; x < endX; x++ {
			path = append(path, Point{X: x, Y: endY, Rune: '─'})
		}
		path = append(path, Point{X: endX, Y: endY, Rune: '▶'})
	}

	return path
}

// ========================================
// MODAL EDITING SYSTEM
// ========================================

// Mode represents the current editing mode
type Mode int

const (
	ModeStartPage Mode = iota
	ModeNormal
	ModeInsert
	ModeInsertSelect // Selecting node to edit
	ModeCommand
	ModeSelectFrom    // Selecting source node for connection
	ModeSelectTo      // Selecting target node for connection
	ModeDelete        // Selecting node to delete
	ModeDeleteConfirm // Confirming node deletion
	ModeLoadConfirm   // Confirming file load (overwrite warning)
	ModeSaveConfirm   // Confirming file save (overwrite warning)
	ModeHelp          // Showing help dialogue
)

// String returns the mode name for display
func (m Mode) String() string {
	switch m {
	case ModeStartPage:
		return "WELCOME"
	case ModeNormal:
		return "NORMAL (help: ?)"
	case ModeInsert:
		return "INSERT"
	case ModeInsertSelect:
		return "SELECT TO EDIT"
	case ModeCommand:
		return "COMMAND"
	case ModeSelectFrom:
		return "SELECT FROM"
	case ModeSelectTo:
		return "SELECT TO"
	case ModeDelete:
		return "DELETE"
	case ModeDeleteConfirm:
		return "DELETE CONFIRM"
	case ModeLoadConfirm:
		return "LOAD CONFIRM"
	case ModeSaveConfirm:
		return "SAVE CONFIRM"
	case ModeHelp:
		return "HELP"
	default:
		return "UNKNOWN"
	}
}

// Editor represents the interactive diagram editor
type Editor struct {
	diagram          Diagram
	mode             Mode
	currentNode      int // ID of currently selected node
	nextNodeID       int // Next available node ID
	canvas           *Canvas
	modeIndicator    *Canvas       // Small canvas for mode animation
	eddCharacter     *EddCharacter // The living edd character
	animationRunning bool          // Whether the living animation is active

	// Jump selection state
	jumpActive       bool
	jumpLabels       map[int]rune // Map from node ID to jump label
	connectionLabels map[int]rune // Map from connection index to jump label
	connectionFrom   int          // Source node for connection (when in SelectTo mode)

	// Insert mode cursor position
	cursorPos int // Character position within current node's text

	// File management
	currentFilename string // Current .edd file being edited

	// Command mode
	commandBuffer    string     // Command being typed in command mode
	pendingFilename  string     // Filename waiting for load confirmation
	startupAnimation *Animation // The startup animation
	animationFrame   int        // Current animation frame
	animationTimer   int        // Timer for animation frame timing
	isAnimating      bool       // Whether animation is playing
}

// EddCharacter represents the living animated character
type EddCharacter struct {
	currentFrame    int
	frameCount      int
	idleAnimations  map[Mode][]string // Idle animation frames per mode
	transitionAnim  []string          // Current transition animation
	isTransitioning bool
	blinkTimer      int
	lookTimer       int
	lookDirection   int // -1 left, 0 center, 1 right
	tableFlipFrames int // Countdown for table flip animation
}

// NewEddCharacter creates a new living ed character
func NewEddCharacter() *EddCharacter {
	ed := &EddCharacter{
		currentFrame:    0,
		frameCount:      0,
		idleAnimations:  make(map[Mode][]string),
		isTransitioning: false,
		blinkTimer:      0,
		lookTimer:       0,
		lookDirection:   0,
	}

	// Define idle animations for each mode - meet "ed"!
	ed.idleAnimations[ModeNormal] = []string{
		"◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "-‿ -", "◉‿ ◉", // Occasional blink with smile
		"◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉",
		"◉‿ ◉", "◉‿ ◉", "⊙‿ ⊙", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", // Occasional look around
	}

	ed.idleAnimations[ModeInsert] = []string{
		"○‿ ○", "○‿ ○", "○‿ ○", "○‿ ○", "○‿ ○", "○‿ ○", // Wide alert eyes
		"-‿ ○", "-‿ -", "○‿ -", "○‿ ○", "○‿ ○", "-‿ ○", // Out-of-sync blinks
		"○‿ ○", "-‿ -", "○‿ ○", "○‿ ○", "◉‿ ◉", "○‿ ○", // Full blink and focus burst
	}

	ed.idleAnimations[ModeInsertSelect] = []string{
		"◉‿ ◉", "⊙‿ ⊙", "◉‿ ◉", "⊙‿ ⊙", "◉‿ ◉", "⊙‿ ⊙", // Contemplative selection - what to edit?
		"⊙‿ ⊙", "-‿ -", "⊙‿ ⊙", "◉‿ ◉", "⊙‿ ⊙", "◉‿ ◉", // Thoughtful scanning
	}

	ed.idleAnimations[ModeSelectFrom] = []string{
		"◉‿ ◉", "◎‿ ◎", "◉‿ ◉", "◎‿ ◎", "◉‿ ◉", "◎‿ ◎", // Eyes darting between nodes
		"◎‿ ◎", "-‿ -", "◎‿ ◎", "◎‿ ◎", "◎‿ ◎", "◎‿ ◎", // Quick blink while scanning
	}

	ed.idleAnimations[ModeSelectTo] = []string{
		"◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", // Focused on target selection
		"◉‿ ◉", "◉‿ ◉", "-‿ -", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", // Focused with quick blink
	}

	ed.idleAnimations[ModeDelete] = []string{
		"◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", // Serious focused look
		"◉‿ ◉", "◉‿ ◉", ">‿ <", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", // Squinting concentration
	}

	ed.idleAnimations[ModeDeleteConfirm] = []string{
		"◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", // Serious, waiting
		"◉‿ ◉", "◉‿ ◉", ">‿ <", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", // Focused concentration
	}

	ed.idleAnimations[ModeHelp] = []string{
		"◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", // Scholarly contemplation
		"◎‿ ◎", "◎‿ ◎", "◉‿ ◉", "◉‿ ◉", "◎‿ ◎", "◉‿ ◉", // Academic focus (glasses effect)
		"◉‿ ◉", "◉‿ ◉", "-‿ -", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", // Wise pondering blink
		"◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "⊙‿ ⊙", "◉‿ ◉", "◉‿ ◉", // Deep thought
	}

	ed.idleAnimations[ModeCommand] = []string{
		":_", ":_", ":|", ":|", ":_", ":_", ":_", ":_", // Cursor blink
		":_", ":_", ":_", ":?", ":_", ":_", ":_", ":_", // Occasional thinking
	}

	return ed
}

// getTerminalSize returns the current terminal dimensions
func getTerminalSize() (width, height int) {
	// Use stty to get terminal size (matches our existing stty usage pattern)
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	output, err := cmd.Output()
	if err != nil {
		return 80, 30 // fallback to reasonable defaults
	}

	parts := strings.Fields(string(output))
	if len(parts) != 2 {
		return 80, 30 // fallback on parse error
	}

	// stty returns "rows columns"
	if h, err := strconv.Atoi(parts[0]); err == nil {
		height = h
	} else {
		height = 30
	}

	if w, err := strconv.Atoi(parts[1]); err == nil {
		width = w
	} else {
		width = 80
	}

	// Reserve space for ed character and padding
	height = height - 7 // Just leave room for ed and some padding
	if height < 10 {
		height = 10 // Minimum usable height
	}

	return width, height
}

// NewEditor creates a new editor instance
func NewEditor() *Editor {
	width, height := getTerminalSize()
	editor := &Editor{
		diagram: Diagram{
			Nodes:       []Node{},
			Connections: []Connection{},
		},
		mode:             ModeStartPage,
		currentNode:      -1, // No node selected initially
		nextNodeID:       0,
		canvas:           NewCanvas(width, height),
		modeIndicator:    NewCanvas(18, 5),  // Wide enough for table flip animation
		eddCharacter:     NewEddCharacter(), // Meet ed!
		animationRunning: false,
		jumpActive:       false,
		jumpLabels:       make(map[int]rune),
		connectionLabels: make(map[int]rune),
		connectionFrom:   -1,
		cursorPos:        0,
		currentFilename:  "",
		commandBuffer:    "",
		pendingFilename:  "",
		startupAnimation: nil,
		animationFrame:   0,
		animationTimer:   0,
		isAnimating:      false,
	}

	return editor
}

// ResizeBuffer resizes the canvas to match current terminal size
func (e *Editor) ResizeBuffer() {
	width, height := getTerminalSize()
	e.canvas.Resize(width, height)
}

// positionCursor moves the terminal cursor to the correct position within the current node
func (e *Editor) positionCursor() {
	if e.mode != ModeInsert || e.currentNode < 0 {
		return
	}

	// Find the current node and get its positioned coordinates
	layout := NewLayeredLayout()
	positioned := layout.CalculateLayout(e.diagram.Nodes, e.diagram.Connections)

	var currentNode *Node
	for _, node := range positioned {
		if node.ID == e.currentNode {
			currentNode = &node
			break
		}
	}

	if currentNode == nil {
		return
	}

	// Get current text
	var currentText string
	if len(currentNode.Text) > 0 {
		currentText = currentNode.Text[0]
	}
	textLength := len(currentText)

	// Ensure cursor position is within bounds
	if e.cursorPos > textLength {
		e.cursorPos = textLength
	}

	// Calculate text start position using same logic as DrawBox
	availableWidth := currentNode.Width - 2*NodePadding - 2 // -2 for borders
	textStartX := currentNode.X + 1 + NodePadding + (availableWidth-textLength)/2
	textY := currentNode.Y + 1

	// Position cursor at the current cursor position within the centered text
	cursorX := textStartX + e.cursorPos
	cursorY := textY

	// Move cursor and show it
	fmt.Printf("\033[%d;%dH\033[?25h", cursorY+1, cursorX+1)
}

// saveDiagram saves the current diagram to a .edd file
func (e *Editor) saveDiagram(filename string) error {
	// Create saved diagram with metadata - no conversion needed!
	saved := SavedDiagram{
		Diagram: e.diagram, // Direct assignment, JSON tags handle the rest
		Metadata: DiagramMetadata{
			Name:    strings.TrimSuffix(filename, ".edd"),
			Created: time.Now().Format("2006-01-02 15:04:05"),
		},
	}

	// Marshal to JSON with indentation for readability
	jsonData, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal diagram: %v", err)
	}

	// Write to file
	err = os.WriteFile(filename, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	// Update current filename
	e.currentFilename = filename
	return nil
}

// loadDiagram loads a diagram from a .edd file
func (e *Editor) loadDiagram(filename string) error {
	// Read file
	jsonData, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	// Parse JSON
	var saved SavedDiagram
	err = json.Unmarshal(jsonData, &saved)
	if err != nil {
		return fmt.Errorf("failed to parse diagram: %v", err)
	}

	// Load diagram data
	e.diagram = saved.Diagram

	// Recalculate node sizes and positions since they weren't saved
	for i := range e.diagram.Nodes {
		node := &e.diagram.Nodes[i]
		// Recalculate width and height from text
		node.Width, node.Height = CalculateNodeSize(node.Text)
	}

	// Update editor state
	e.currentFilename = filename
	e.currentNode = -1 // Reset current node selection
	e.cursorPos = 0

	// Update nextNodeID to avoid conflicts
	maxID := -1
	for _, node := range e.diagram.Nodes {
		if node.ID > maxID {
			maxID = node.ID
		}
	}
	e.nextNodeID = maxID + 1

	return nil
}

// hasContent returns true if the diagram has any nodes or connections
func (e *Editor) hasContent() bool {
	return len(e.diagram.Nodes) > 0 || len(e.diagram.Connections) > 0
}

// fileExists returns true if the file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// SetMode changes the current editing mode
func (e *Editor) SetMode(mode Mode) {
	oldMode := e.mode
	e.mode = mode

	// Handle living character transition
	if oldMode != mode {
		e.eddCharacter.TransitionToMode(mode)

		// Start opening animation when entering start page
		if mode == ModeStartPage && oldMode != ModeStartPage {
			e.startStartupAnimation()
		}
	}

	e.UpdateLivingModeIndicator()
}

// startStartupAnimation begins the startup animation sequence
func (e *Editor) startStartupAnimation() {
	anim := CreateStartupAnimation()
	e.startupAnimation = &anim
	e.animationFrame = 0
	e.isAnimating = true
	e.animationTimer = 0
}

// updateStartupAnimation advances the startup animation
func (e *Editor) updateStartupAnimation() {
	if !e.isAnimating || e.startupAnimation == nil {
		return
	}

	// Each ticker is 200ms, so increment timer
	e.animationTimer += 200

	// Check if it's time to advance to next frame
	if e.animationFrame < len(e.startupAnimation.Frames) {
		currentFrame := e.startupAnimation.Frames[e.animationFrame]
		if e.animationTimer >= currentFrame.Delay {
			e.animationFrame++
			e.animationTimer = 0
			e.Render() // Render the new frame
		}
	}
}

// StartLivingAnimation begins the continuous character animation
func (e *Editor) StartLivingAnimation() {
	if e.animationRunning {
		return // Already running
	}

	e.animationRunning = true
	go e.livingAnimationLoop()
}

// StopLivingAnimation stops the continuous character animation
func (e *Editor) StopLivingAnimation() {
	e.animationRunning = false
}

// livingAnimationLoop runs the continuous character animation
func (e *Editor) livingAnimationLoop() {
	for e.animationRunning {
		e.eddCharacter.NextFrame(e.mode)
		// Note: UpdateLivingModeIndicator is called by main render loop
		// to avoid unsynchronized updates and flashing
		time.Sleep(400 * time.Millisecond) // ~2.5 FPS for subtle animation
	}
}

// DisplayCharacter shows the current character state (disabled in editor mode)
func (e *Editor) DisplayCharacter() {
	// Don't display here - the main Render() handles it
}

// UpdateLivingModeIndicator updates the indicator with the living character
func (e *Editor) UpdateLivingModeIndicator() {
	e.modeIndicator.Clear()

	// Handle table flip animation separately
	if e.eddCharacter.tableFlipFrames > 0 {
		e.drawTableFlipAnimation()
		return
	}

	// Get current living frame
	face := e.eddCharacter.GetCurrentFrame(e.mode)
	cursor := ""
	connections := e.eddCharacter.GetConnections(e.mode)

	// Draw the living edd character
	e.drawLivingEdd(face, cursor, connections)

	// Add mode label
	text := e.mode.String()
	startX := 2
	for i, ch := range text {
		e.modeIndicator.Set(startX+i, 4, ch)
	}
}

// drawLivingEdd draws the living character with current expression
func (e *Editor) drawLivingEdd(face, cursor string, connections map[string]rune) {
	// Draw box
	e.modeIndicator.Set(2, 1, '╭')
	e.modeIndicator.Set(7, 1, '╮')
	e.modeIndicator.Set(2, 3, '╰')
	e.modeIndicator.Set(7, 3, '╯')
	for x := 3; x < 7; x++ {
		e.modeIndicator.Set(x, 1, '─')
		e.modeIndicator.Set(x, 3, '─')
	}
	e.modeIndicator.Set(2, 2, '│')
	e.modeIndicator.Set(7, 2, '│')

	// Handle different face formats
	faceRunes := []rune(face)

	if len(face) >= 2 && face[0] == ':' {
		// Command mode format ":_", ":|", ":?"
		e.modeIndicator.Set(3, 2, ':')
		if len(faceRunes) > 1 {
			e.modeIndicator.Set(4, 2, faceRunes[1])
		}
		// Add eyes for command mode
		e.modeIndicator.Set(2, 2, '◉')
		e.modeIndicator.Set(6, 2, '◉')
	} else {
		// Regular face format "◉‿◉", "◉▿◉|", etc.
		// Split by cursor character if present
		mainFace := face
		cursorChar := ""

		if strings.Contains(face, "|") {
			parts := strings.Split(face, "|")
			mainFace = parts[0]
			if len(parts) > 1 {
				cursorChar = "|" + parts[1]
			} else {
				cursorChar = "|"
			}
		} else if strings.Contains(face, " ") && len(face) > 3 {
			// Handle space-separated cursor
			if len(faceRunes) > 3 && faceRunes[3] == ' ' {
				mainFace = string(faceRunes[:3])
				cursorChar = string(faceRunes[3:])
			}
		}

		// Draw the main face (up to 4 characters to accommodate "◉‿ ◉")
		mainFaceRunes := []rune(mainFace)
		for i, ch := range mainFaceRunes {
			if i < 4 { // Allow 4 characters for the face
				e.modeIndicator.Set(3+i, 2, ch)
			}
		}

		// Draw cursor if present
		if cursorChar != "" {
			cursorRunes := []rune(cursorChar)
			for i, ch := range cursorRunes {
				e.modeIndicator.Set(6+i, 2, ch)
			}
		}
	}

	// Add connections if any
	for pos, char := range connections {
		switch pos {
		case "left":
			e.modeIndicator.Set(1, 2, char)
		case "right":
			e.modeIndicator.Set(8, 2, char)
		case "up":
			e.modeIndicator.Set(4, 0, char)
		case "down":
			e.modeIndicator.Set(4, 4, char)
		case "cursor":
			e.modeIndicator.Set(9, 2, char) // Cursor right of the box
		}
	}
}

// drawTableFlipAnimation draws ed doing a table flip with the table outside his box
func (e *Editor) drawTableFlipAnimation() {
	frames := e.eddCharacter.tableFlipFrames

	// Always draw ed's box first
	e.modeIndicator.Set(2, 1, '╭')
	e.modeIndicator.Set(7, 1, '╮')
	e.modeIndicator.Set(2, 3, '╰')
	e.modeIndicator.Set(7, 3, '╯')
	for x := 3; x < 7; x++ {
		e.modeIndicator.Set(x, 1, '─')
		e.modeIndicator.Set(x, 3, '─')
	}
	e.modeIndicator.Set(2, 2, '│')
	e.modeIndicator.Set(7, 2, '│')

	switch frames {
	case 5:
		// Realization - narrowing eyes
		e.modeIndicator.Set(3, 2, '>')
		e.modeIndicator.Set(4, 2, '‿')
		e.modeIndicator.Set(5, 2, ' ')
		e.modeIndicator.Set(6, 2, '<')
		// Table sitting peacefully to the right
		e.modeIndicator.Set(10, 3, '┬')
		e.modeIndicator.Set(11, 3, '─')
		e.modeIndicator.Set(12, 3, '─')
		e.modeIndicator.Set(13, 3, '┬')

	case 4:
		// Building rage - wide angry eyes
		e.modeIndicator.Set(3, 2, '◉')
		e.modeIndicator.Set(4, 2, 'Д')
		e.modeIndicator.Set(5, 2, ' ')
		e.modeIndicator.Set(6, 2, '◉')
		// Arms starting to rise
		e.modeIndicator.Set(1, 2, '/')
		e.modeIndicator.Set(8, 2, '\\')
		// Table still there
		e.modeIndicator.Set(10, 3, '┬')
		e.modeIndicator.Set(11, 3, '─')
		e.modeIndicator.Set(12, 3, '─')
		e.modeIndicator.Set(13, 3, '┬')

	case 3:
		// Full rage - arms up, grabbing table
		e.modeIndicator.Set(3, 2, '◉')
		e.modeIndicator.Set(4, 2, 'Д')
		e.modeIndicator.Set(5, 2, '◉')
		// Arms raised high
		e.modeIndicator.Set(0, 1, '(')
		e.modeIndicator.Set(1, 1, '╯')
		e.modeIndicator.Set(8, 1, '╯')
		e.modeIndicator.Set(9, 1, ')')
		// Table being grabbed
		e.modeIndicator.Set(10, 2, '┬')
		e.modeIndicator.Set(11, 2, '─')
		e.modeIndicator.Set(12, 2, '─')
		e.modeIndicator.Set(13, 2, '┬')

	case 2:
		// Mid flip - maximum effort
		e.modeIndicator.Set(3, 2, '>')
		e.modeIndicator.Set(4, 2, 'Д')
		e.modeIndicator.Set(5, 2, '<')
		// Arms in throwing motion
		e.modeIndicator.Set(0, 0, '(')
		e.modeIndicator.Set(1, 0, '╯')
		e.modeIndicator.Set(8, 0, '╯')
		e.modeIndicator.Set(9, 0, ')')
		// Table mid-flip
		e.modeIndicator.Set(10, 1, '︵')
		e.modeIndicator.Set(11, 1, ' ')
		e.modeIndicator.Set(12, 1, '┻')
		e.modeIndicator.Set(13, 1, '━')
		e.modeIndicator.Set(14, 1, '┻')

	case 1:
		// Satisfied - table gone, ed happy
		e.modeIndicator.Set(3, 2, '◉')
		e.modeIndicator.Set(4, 2, '‿')
		e.modeIndicator.Set(5, 2, ' ')
		e.modeIndicator.Set(6, 2, '◉')
		// Arms down, dusting off hands
		e.modeIndicator.Set(1, 2, '‾')
		e.modeIndicator.Set(8, 2, '‾')
		// Table far away
		e.modeIndicator.Set(12, 0, '︵')
		e.modeIndicator.Set(14, 0, '┻')
		e.modeIndicator.Set(15, 0, '┻')
	}

	// Mode label
	text := "TABLE FLIP!"
	startX := 2
	for i, ch := range text {
		if startX+i < 15 { // Don't overflow canvas
			e.modeIndicator.Set(startX+i, 4, ch)
		}
	}
}

// EddCharacter methods for living animation

// NextFrame advances to the next frame in the current mode's animation
func (edd *EddCharacter) NextFrame(mode Mode) {
	// Handle table flip countdown
	if edd.tableFlipFrames > 0 {
		edd.tableFlipFrames--
		return
	}

	if edd.isTransitioning {
		return // Don't advance during transitions
	}

	frames := edd.idleAnimations[mode]
	if len(frames) > 0 {
		edd.currentFrame = (edd.currentFrame + 1) % len(frames)
	}
}

// GetCurrentFrame returns the current animation frame for the given mode
func (edd *EddCharacter) GetCurrentFrame(mode Mode) string {
	// Table flip is handled separately in drawTableFlipAnimation
	// Don't return table flip frames here

	frames := edd.idleAnimations[mode]
	if len(frames) == 0 {
		return "◉_◉" // Default
	}
	return frames[edd.currentFrame]
}

// TriggerTableFlip starts the table flip animation sequence
func (edd *EddCharacter) TriggerTableFlip() {
	edd.tableFlipFrames = 5 // Will show table flip for next 5 frames
}

// GetConnections returns connection indicators for the current mode
func (edd *EddCharacter) GetConnections(mode Mode) map[string]rune {
	connections := make(map[string]rune)

	if mode == ModeSelectFrom || mode == ModeSelectTo {
		// Animate connection lines extending/retracting
		intensity := edd.currentFrame % 4
		if intensity >= 1 {
			connections["left"] = '◀'
		}
		if intensity >= 2 {
			connections["right"] = '▶'
		}
		if intensity >= 3 {
			connections["up"] = '▲'
		}
	} else if mode == ModeInsert {
		// Blinking cursor outside the box
		if edd.currentFrame%4 < 2 {
			connections["cursor"] = '|'
		}
	}

	return connections
}

// TransitionToMode handles smooth transitions between mode states
func (edd *EddCharacter) TransitionToMode(newMode Mode) {
	// Reset frame counter for new mode
	edd.currentFrame = 0
	edd.isTransitioning = false // Simple transition for now
}

// UpdateModeIndicator updates the mode indicator with current mode animation
func (e *Editor) UpdateModeIndicator() {
	e.modeIndicator.Clear()

	switch e.mode {
	case ModeNormal:
		e.DrawNormalModeIndicator()
	case ModeInsert:
		e.DrawInsertModeIndicator()
	case ModeCommand:
		e.DrawCommandModeIndicator()
	}
}

// DrawNormalModeIndicator shows a calm, centered box
func (e *Editor) DrawNormalModeIndicator() {
	// Centered, composed face
	e.modeIndicator.Set(2, 1, '╭')
	e.modeIndicator.Set(7, 1, '╮')
	e.modeIndicator.Set(2, 3, '╰')
	e.modeIndicator.Set(7, 3, '╯')
	for x := 3; x < 7; x++ {
		e.modeIndicator.Set(x, 1, '─')
		e.modeIndicator.Set(x, 3, '─')
	}
	e.modeIndicator.Set(2, 2, '│')
	e.modeIndicator.Set(7, 2, '│')

	// Calm eyes and expression: ◉_◉
	e.modeIndicator.Set(3, 2, '◉')
	e.modeIndicator.Set(4, 2, '_')
	e.modeIndicator.Set(6, 2, '◉')

	// Mode label
	text := "NORMAL"
	for i, ch := range text {
		e.modeIndicator.Set(1+i, 4, ch)
	}
}

// DrawInsertModeIndicator shows an active, typing box
func (e *Editor) DrawInsertModeIndicator() {
	// Same box but more energetic expression
	e.modeIndicator.Set(2, 1, '╭')
	e.modeIndicator.Set(7, 1, '╮')
	e.modeIndicator.Set(2, 3, '╰')
	e.modeIndicator.Set(7, 3, '╯')
	for x := 3; x < 7; x++ {
		e.modeIndicator.Set(x, 1, '─')
		e.modeIndicator.Set(x, 3, '─')
	}
	e.modeIndicator.Set(2, 2, '│')
	e.modeIndicator.Set(7, 2, '│')

	// Focused typing expression: ◉▿◉
	e.modeIndicator.Set(3, 2, '◉')
	e.modeIndicator.Set(4, 2, '▿')
	e.modeIndicator.Set(6, 2, '◉')

	// Blinking cursor effect
	e.modeIndicator.Set(8, 2, '|')

	// Mode label
	text := "INSERT"
	for i, ch := range text {
		e.modeIndicator.Set(1+i, 4, ch)
	}
}

// DrawConnectModeIndicator shows a box with connection lines
func (e *Editor) DrawConnectModeIndicator() {
	// Box with connection arms extending
	e.modeIndicator.Set(2, 1, '╭')
	e.modeIndicator.Set(7, 1, '╮')
	e.modeIndicator.Set(2, 3, '╰')
	e.modeIndicator.Set(7, 3, '╯')
	for x := 3; x < 7; x++ {
		e.modeIndicator.Set(x, 1, '─')
		e.modeIndicator.Set(x, 3, '─')
	}
	e.modeIndicator.Set(2, 2, '│')
	e.modeIndicator.Set(7, 2, '│')

	// Searching/connecting expression: ⊙‿⊙
	e.modeIndicator.Set(3, 2, '⊙')
	e.modeIndicator.Set(4, 2, '‿')
	e.modeIndicator.Set(6, 2, '⊙')

	// Connection lines extending outward
	e.modeIndicator.Set(1, 2, '◀') // Left connection
	e.modeIndicator.Set(8, 2, '▶') // Right connection
	e.modeIndicator.Set(4, 0, '▲') // Up connection

	// Mode label
	text := "CONNECT"
	for i, ch := range text {
		e.modeIndicator.Set(0+i, 4, ch)
	}
}

// DrawCommandModeIndicator shows a command prompt style
func (e *Editor) DrawCommandModeIndicator() {
	// Command line style box
	e.modeIndicator.Set(1, 1, '╭')
	e.modeIndicator.Set(8, 1, '╮')
	e.modeIndicator.Set(1, 3, '╰')
	e.modeIndicator.Set(8, 3, '╯')
	for x := 2; x < 8; x++ {
		e.modeIndicator.Set(x, 1, '─')
		e.modeIndicator.Set(x, 3, '─')
	}
	e.modeIndicator.Set(1, 2, '│')
	e.modeIndicator.Set(8, 2, '│')

	// Command prompt: :
	e.modeIndicator.Set(2, 2, ':')
	e.modeIndicator.Set(3, 2, '_')

	// Thinking expression
	e.modeIndicator.Set(5, 2, '◉')
	e.modeIndicator.Set(7, 2, '◉')

	// Mode label
	text := "COMMAND"
	for i, ch := range text {
		e.modeIndicator.Set(0+i, 4, ch)
	}
}

// PlayModeTransition creates a smooth animation between modes
func (e *Editor) PlayModeTransition(fromMode, toMode Mode) {
	frames := []AnimationFrame{}

	// Transition effects based on mode change
	if fromMode == ModeNormal && toMode == ModeInsert {
		// Normal → Insert: Eyes get focused, cursor appears
		frames = e.CreateNormalToInsertTransition()
	} else if fromMode == ModeInsert && toMode == ModeNormal {
		// Insert → Normal: Cursor fades, eyes relax
		frames = e.CreateInsertToNormalTransition()
	} else if toMode == ModeSelectFrom || toMode == ModeSelectTo {
		// Any → Connect: Lines extend outward
		frames = e.CreateConnectModeTransition()
	} else {
		// Generic transition: quick blink
		frames = e.CreateGenericTransition()
	}

	// Play the transition
	anim := Animation{Frames: frames, Loop: false}
	PlayAnimation(anim)
}

// CreateNormalToInsertTransition creates the calm → focused transition
func (e *Editor) CreateNormalToInsertTransition() []AnimationFrame {
	frames := []AnimationFrame{}

	// Frame 1: Normal state
	canvas1 := NewCanvas(15, 5)
	e.drawBoxWithFace(canvas1, "◉_◉", "")
	frames = append(frames, AnimationFrame{Canvas: canvas1, Delay: 200})

	// Frame 2: Eyes widening
	canvas2 := NewCanvas(15, 5)
	e.drawBoxWithFace(canvas2, "⊙_⊙", "")
	frames = append(frames, AnimationFrame{Canvas: canvas2, Delay: 100})

	// Frame 3: Insert mode - focused
	canvas3 := NewCanvas(15, 5)
	e.drawBoxWithFace(canvas3, "◉▿◉", "|")
	frames = append(frames, AnimationFrame{Canvas: canvas3, Delay: 200})

	return frames
}

// CreateInsertToNormalTransition creates the focused → calm transition
func (e *Editor) CreateInsertToNormalTransition() []AnimationFrame {
	frames := []AnimationFrame{}

	// Frame 1: Insert state with cursor
	canvas1 := NewCanvas(15, 5)
	e.drawBoxWithFace(canvas1, "◉▿◉", "|")
	frames = append(frames, AnimationFrame{Canvas: canvas1, Delay: 150})

	// Frame 2: Cursor fading
	canvas2 := NewCanvas(15, 5)
	e.drawBoxWithFace(canvas2, "◉▿◉", "")
	frames = append(frames, AnimationFrame{Canvas: canvas2, Delay: 100})

	// Frame 3: Eyes relaxing
	canvas3 := NewCanvas(15, 5)
	e.drawBoxWithFace(canvas3, "◉_◉", "")
	frames = append(frames, AnimationFrame{Canvas: canvas3, Delay: 200})

	return frames
}

// CreateConnectModeTransition creates connection lines extending
func (e *Editor) CreateConnectModeTransition() []AnimationFrame {
	frames := []AnimationFrame{}

	// Frame 1: Normal box
	canvas1 := NewCanvas(15, 5)
	e.drawBoxWithFace(canvas1, "⊙‿⊙", "")
	frames = append(frames, AnimationFrame{Canvas: canvas1, Delay: 100})

	// Frame 2: Lines starting to extend
	canvas2 := NewCanvas(15, 5)
	e.drawBoxWithFace(canvas2, "⊙‿⊙", "")
	canvas2.Set(8, 2, '▶')
	frames = append(frames, AnimationFrame{Canvas: canvas2, Delay: 100})

	// Frame 3: Full connection mode
	canvas3 := NewCanvas(15, 5)
	e.drawBoxWithFace(canvas3, "⊙‿⊙", "")
	canvas3.Set(1, 2, '◀')
	canvas3.Set(8, 2, '▶')
	canvas3.Set(4, 0, '▲')
	frames = append(frames, AnimationFrame{Canvas: canvas3, Delay: 200})

	return frames
}

// CreateGenericTransition creates a simple blink transition
func (e *Editor) CreateGenericTransition() []AnimationFrame {
	frames := []AnimationFrame{}

	// Blink effect
	canvas1 := NewCanvas(15, 5)
	e.drawBoxWithFace(canvas1, "-_-", "")
	frames = append(frames, AnimationFrame{Canvas: canvas1, Delay: 100})

	canvas2 := NewCanvas(15, 5)
	e.drawBoxWithFace(canvas2, "◉_◉", "")
	frames = append(frames, AnimationFrame{Canvas: canvas2, Delay: 100})

	return frames
}

// Helper function to draw a box with face and optional cursor
func (e *Editor) drawBoxWithFace(canvas *Canvas, face string, cursor string) {
	// Draw box
	canvas.Set(2, 1, '╭')
	canvas.Set(7, 1, '╮')
	canvas.Set(2, 3, '╰')
	canvas.Set(7, 3, '╯')
	for x := 3; x < 7; x++ {
		canvas.Set(x, 1, '─')
		canvas.Set(x, 3, '─')
	}
	canvas.Set(2, 2, '│')
	canvas.Set(7, 2, '│')

	// Add face
	for i, ch := range face {
		canvas.Set(3+i, 2, ch)
	}

	// Add cursor if provided
	if cursor != "" {
		for i, ch := range cursor {
			canvas.Set(8+i, 2, ch)
		}
	}
}

// AddNode creates a new node with the given text
func (e *Editor) AddNode(text []string) int {
	width, height := CalculateNodeSize(text)
	node := Node{
		ID:     e.nextNodeID,
		Text:   text,
		Width:  width,
		Height: height,
		// X, Y will be set by layout algorithm
	}
	e.diagram.Nodes = append(e.diagram.Nodes, node)
	nodeID := e.nextNodeID
	e.nextNodeID++
	e.currentNode = nodeID
	return nodeID
}

// printDebugInfo outputs the current graph structure for debugging
func (e *Editor) printDebugInfo() {
	fmt.Println("\n=== GRAPH DEBUG INFO ===")
	fmt.Printf("Mode: %s\n", e.mode)
	fmt.Printf("Current Node: %d\n", e.currentNode)
	fmt.Printf("Next Node ID: %d\n", e.nextNodeID)

	fmt.Println("\nNodes:")
	for _, node := range e.diagram.Nodes {
		fmt.Printf("  Node %d: pos(%d,%d) size(%dx%d) text=%q\n",
			node.ID, node.X, node.Y, node.Width, node.Height,
			strings.Join(node.Text, " "))
	}

	fmt.Println("\nConnections:")
	for i, conn := range e.diagram.Connections {
		fmt.Printf("  [%d] %d -> %d", i, conn.From, conn.To)
		if len(conn.Path) > 0 {
			fmt.Printf(" (path length: %d)", len(conn.Path))
		}
		fmt.Println()
	}

	// Show positioned nodes from layout
	fmt.Println("\nPositioned Nodes (from layout):")
	layout := NewLayeredLayout()
	positioned := layout.CalculateLayout(e.diagram.Nodes, e.diagram.Connections)
	for _, node := range positioned {
		fmt.Printf("  Node %d: pos(%d,%d) size(%dx%d) text=%q\n",
			node.ID, node.X, node.Y, node.Width, node.Height, strings.Join(node.Text, " "))
	}

	fmt.Println("\n=== END DEBUG INFO ===")
	fmt.Println("\nCopy the above output when reporting issues!")
}

// ========================================
// JUMP SELECTION SYSTEM
// ========================================

// startJump activates jump mode and assigns labels to nodes
func (e *Editor) startJump() {
	if len(e.diagram.Nodes) == 0 && (e.mode != ModeDelete || len(e.diagram.Connections) == 0) {
		return // No nodes/connections to select
	}

	e.jumpActive = true
	e.jumpLabels = make(map[int]rune)
	e.connectionLabels = make(map[int]rune)

	// Generate labels using home row keys first, then other letters
	labels := "asdfghjklqwertyuiopzxcvbnm"
	labelIndex := 0

	// Assign labels to nodes
	for _, node := range e.diagram.Nodes {
		if labelIndex < len(labels) {
			e.jumpLabels[node.ID] = rune(labels[labelIndex])
			labelIndex++
		}
	}

	// In delete mode, also assign labels to connections
	if e.mode == ModeDelete {
		for i := range e.diagram.Connections {
			if labelIndex < len(labels) {
				e.connectionLabels[i] = rune(labels[labelIndex])
				labelIndex++
			}
		}
	}
}

// stopJump deactivates jump mode
func (e *Editor) stopJump() {
	e.jumpActive = false
	e.jumpLabels = make(map[int]rune)
	e.connectionLabels = make(map[int]rune)
}

// handleJumpSelection processes jump character selection
func (e *Editor) handleJumpSelection(key rune) bool {
	// Find node with this label
	for nodeID, label := range e.jumpLabels {
		if label == key {
			return e.selectNode(nodeID)
		}
	}

	// Find connection with this label (only in delete mode)
	if e.mode == ModeDelete {
		for connIndex, label := range e.connectionLabels {
			if label == key {
				return e.selectConnection(connIndex)
			}
		}
	}

	return false // Key not found
}

// selectNode handles node selection based on current mode
func (e *Editor) selectNode(nodeID int) bool {
	switch e.mode {
	case ModeSelectFrom:
		// Selected source node for connection
		e.connectionFrom = nodeID
		e.SetMode(ModeSelectTo)
		e.startJump() // Start new jump for target selection
		return false

	case ModeSelectTo:
		// Selected target node, create connection
		if e.connectionFrom >= 0 {
			conn := Connection{From: e.connectionFrom, To: nodeID}
			e.diagram.Connections = append(e.diagram.Connections, conn)
			fmt.Printf("Connected node %d to node %d\n", e.connectionFrom, nodeID)
		}
		// Stay in connect mode - go back to FROM selection
		e.SetMode(ModeSelectFrom)
		e.connectionFrom = -1
		e.startJump() // Start new jump for next connection
		return false

	case ModeInsertSelect:
		// Selected node to edit - enter insert mode
		e.currentNode = nodeID
		// Set cursor position at end of existing text
		for _, node := range e.diagram.Nodes {
			if node.ID == nodeID && len(node.Text) > 0 {
				e.cursorPos = len(node.Text[0])
				break
			}
		}
		e.SetMode(ModeInsert)
		e.stopJump()
		return false

	case ModeDelete:
		// Selected node to delete - prompt for confirmation
		return e.confirmDelete(nodeID)

	default:
		// Just select the node
		e.currentNode = nodeID
		e.stopJump()
		e.SetMode(ModeNormal)
		return false
	}
}

// selectConnection handles connection selection for deletion
func (e *Editor) selectConnection(connIndex int) bool {
	if e.mode == ModeDelete && connIndex < len(e.diagram.Connections) {
		// Directly delete the connection (no confirmation needed for connections)
		e.eddCharacter.TriggerTableFlip()
		e.deleteConnection(connIndex)
		// Stay in delete mode - restart jump selection
		e.startJump()
	}
	return false
}

// deleteConnection removes a connection by index
func (e *Editor) deleteConnection(connIndex int) {
	if connIndex >= 0 && connIndex < len(e.diagram.Connections) {
		e.diagram.Connections = append(e.diagram.Connections[:connIndex], e.diagram.Connections[connIndex+1:]...)
	}
}

// confirmDelete handles delete confirmation for a node
func (e *Editor) confirmDelete(nodeID int) bool {
	// Keep jump active but store the node to delete
	e.connectionFrom = nodeID // Reuse this field to store pending delete
	e.stopJump()              // Stop showing jump labels but keep node highlighted

	// Switch to delete confirm mode - this triggers ed's table flip animation
	e.SetMode(ModeDeleteConfirm)

	return false
}

// deleteNode removes a node and all its connections
func (e *Editor) deleteNode(nodeID int) {
	// Remove the node
	for i, node := range e.diagram.Nodes {
		if node.ID == nodeID {
			e.diagram.Nodes = append(e.diagram.Nodes[:i], e.diagram.Nodes[i+1:]...)
			break
		}
	}

	// Remove all connections involving this node
	var remainingConnections []Connection
	for _, conn := range e.diagram.Connections {
		if conn.From != nodeID && conn.To != nodeID {
			remainingConnections = append(remainingConnections, conn)
		}
	}
	e.diagram.Connections = remainingConnections

	// Update current node if it was deleted
	if e.currentNode == nodeID {
		e.currentNode = -1
	}
}

// renderJumpLabels overlays jump labels on nodes and connections
func (e *Editor) renderJumpLabels() {
	// Get positioned nodes
	layout := NewLayeredLayout()
	positioned := layout.CalculateLayout(e.diagram.Nodes, e.diagram.Connections)

	// Print labels directly after the canvas
	fmt.Print("\033[s") // Save cursor position

	// Render node labels
	for _, node := range positioned {
		if e.mode == ModeDeleteConfirm && node.ID == e.connectionFrom {
			// Show y/N prompt for the node being deleted
			labelX := node.X + 1
			labelY := node.Y
			fmt.Printf("\033[%d;%dH\033[33my/N?\033[0m", labelY+1, labelX+1)
		} else if e.mode == ModeSelectTo && node.ID == e.connectionFrom {
			// Mark selected "from" node in green
			labelX := node.X + 1
			labelY := node.Y
			fmt.Printf("\033[%d;%dH\033[32mFROM\033[0m", labelY+1, labelX+1)
		} else if label, exists := e.jumpLabels[node.ID]; exists {
			// Position cursor and print yellow label
			labelX := node.X + 1
			labelY := node.Y
			fmt.Printf("\033[%d;%dH\033[33m%c\033[0m", labelY+1, labelX+1, label)
		}
	}

	// Render connection labels (only in delete mode)
	if e.mode == ModeDelete {
		// Create node map for connection rendering
		nodeMap := make(map[int]Node)
		for _, node := range positioned {
			nodeMap[node.ID] = node
		}

		for i, conn := range e.diagram.Connections {
			if label, exists := e.connectionLabels[i]; exists {
				fromNode := nodeMap[conn.From]
				toNode := nodeMap[conn.To]

				// Route the connection to find the arrow position
				path := SimpleOrthogonalRouteWithContext(fromNode, toNode, positioned)

				if len(path) >= 2 {
					// Place label near the arrow (which is at the end of the path)
					// Use the second-to-last point to avoid overlapping with arrow
					labelPoint := path[len(path)-2]

					// Adjust position based on arrow direction
					labelX := labelPoint.X
					labelY := labelPoint.Y

					// If the arrow is horizontal, place label above/below
					if len(path) >= 3 {
						prevPoint := path[len(path)-3]
						if prevPoint.Y == labelPoint.Y {
							// Horizontal approach to arrow
							labelY = labelPoint.Y - 1 // Place above the line
							if labelY < 0 {
								labelY = labelPoint.Y + 1 // Place below if no room above
							}
						} else {
							// Vertical approach to arrow
							labelX = labelPoint.X - 2 // Place to the left
							if labelX < 0 {
								labelX = labelPoint.X + 2 // Place to the right if no room
							}
						}
					}

					// Print red label for connection
					fmt.Printf("\033[%d;%dH\033[31m%c\033[0m", labelY+1, labelX+1, label)
				}
			}
		}
	}

	fmt.Print("\033[u") // Restore cursor position
}

// ========================================
// ANIMATION SYSTEM
// ========================================

// AnimationFrame represents a single frame of animation
type AnimationFrame struct {
	Canvas *Canvas
	Delay  int // milliseconds
}

// Animation represents a sequence of frames
type Animation struct {
	Frames []AnimationFrame
	Loop   bool
}

// PlayAnimation renders animation frames with timing
func PlayAnimation(anim Animation) {
	PlayAnimationWithLimit(anim, -1) // No limit by default
}

// PlayAnimationWithLimit renders animation frames with optional cycle limit
func PlayAnimationWithLimit(anim Animation, maxCycles int) {
	cycles := 0
	for i := 0; i < len(anim.Frames); i++ {
		frame := anim.Frames[i]

		// Clear screen and show frame
		fmt.Print("\033[2J\033[H") // Clear screen, move cursor to top
		fmt.Print(frame.Canvas.String())

		// Wait for specified delay
		if frame.Delay > 0 {
			time.Sleep(time.Duration(frame.Delay) * time.Millisecond)
		}

		// Loop back to start if needed
		if anim.Loop && i == len(anim.Frames)-1 {
			cycles++
			if maxCycles > 0 && cycles >= maxCycles {
				break
			}
			i = -1 // Will be incremented to 0
		}
	}
}

// CreateStartupAnimation creates the box-morphing startup sequence
func CreateStartupAnimation() Animation {
	width, height := 40, 15
	frames := []AnimationFrame{}

	// Frame 1: Scattered pieces
	canvas1 := NewCanvas(width, height)
	canvas1.Set(5, 3, '╭')
	canvas1.Set(25, 8, '╮')
	canvas1.Set(15, 12, '╰')
	canvas1.Set(30, 2, '╯')
	canvas1.Set(10, 7, '─')
	canvas1.Set(20, 5, '│')
	frames = append(frames, AnimationFrame{Canvas: canvas1, Delay: 300})

	// Frame 2: Pieces moving closer
	canvas2 := NewCanvas(width, height)
	canvas2.Set(12, 5, '╭')
	canvas2.Set(22, 6, '╮')
	canvas2.Set(13, 9, '╰')
	canvas2.Set(21, 8, '╯')
	canvas2.Set(15, 7, '─')
	canvas2.Set(18, 6, '│')
	frames = append(frames, AnimationFrame{Canvas: canvas2, Delay: 300})

	// Frame 3: Almost formed box
	canvas3 := NewCanvas(width, height)
	canvas3.Set(15, 6, '╭')
	canvas3.Set(20, 6, '╮')
	canvas3.Set(15, 8, '╰')
	canvas3.Set(20, 8, '╯')
	for x := 16; x < 20; x++ {
		canvas3.Set(x, 6, '─')
		canvas3.Set(x, 8, '─')
	}
	canvas3.Set(15, 7, '│')
	canvas3.Set(20, 7, '│')
	frames = append(frames, AnimationFrame{Canvas: canvas3, Delay: 400})

	// Frame 4: Complete box with eyes appearing
	canvas4 := NewCanvas(width, height)
	canvas4.Set(15, 6, '╭')
	canvas4.Set(20, 6, '╮')
	canvas4.Set(15, 8, '╰')
	canvas4.Set(20, 8, '╯')
	for x := 16; x < 20; x++ {
		canvas4.Set(x, 6, '─')
		canvas4.Set(x, 8, '─')
	}
	canvas4.Set(15, 7, '│')
	canvas4.Set(20, 7, '│')
	// Add eyes
	canvas4.Set(16, 7, '◉')
	canvas4.Set(19, 7, '◉')
	frames = append(frames, AnimationFrame{Canvas: canvas4, Delay: 500})

	// Frame 5: Happy face!
	canvas5 := NewCanvas(width, height)
	canvas5.Set(15, 6, '╭')
	canvas5.Set(20, 6, '╮')
	canvas5.Set(15, 8, '╰')
	canvas5.Set(20, 8, '╯')
	for x := 16; x < 20; x++ {
		canvas5.Set(x, 6, '─')
		canvas5.Set(x, 8, '─')
	}
	canvas5.Set(15, 7, '│')
	canvas5.Set(20, 7, '│')
	// Happy face: ◉‿◉
	canvas5.Set(16, 7, '◉')
	canvas5.Set(17, 7, '‿')
	canvas5.Set(19, 7, '◉')
	frames = append(frames, AnimationFrame{Canvas: canvas5, Delay: 800})

	// Frame 6: Add "edd" text below
	canvas6 := NewCanvas(width, height)
	canvas6.Set(15, 6, '╭')
	canvas6.Set(20, 6, '╮')
	canvas6.Set(15, 8, '╰')
	canvas6.Set(20, 8, '╯')
	for x := 16; x < 20; x++ {
		canvas6.Set(x, 6, '─')
		canvas6.Set(x, 8, '─')
	}
	canvas6.Set(15, 7, '│')
	canvas6.Set(20, 7, '│')
	canvas6.Set(16, 7, '◉')
	canvas6.Set(17, 7, '‿')
	canvas6.Set(19, 7, '◉')

	// "edd" text
	text := "edd - elegant diagram drawer"
	startX := (width - len(text)) / 2
	for i, ch := range text {
		canvas6.Set(startX+i, 10, ch)
	}
	frames = append(frames, AnimationFrame{Canvas: canvas6, Delay: 1000})

	return Animation{Frames: frames, Loop: false}
}

// CreateThinkingAnimation creates a "processing" animation
func CreateThinkingAnimation() Animation {
	width, height := 20, 10
	frames := []AnimationFrame{}

	// Thinking faces cycle
	faces := []string{"⊙_⊙", "◉_◉", "⊙_⊙", "-_-"}

	for _, face := range faces {
		canvas := NewCanvas(width, height)

		// Draw box
		canvas.Set(8, 3, '╭')
		canvas.Set(12, 3, '╮')
		canvas.Set(8, 5, '╰')
		canvas.Set(12, 5, '╯')
		for x := 9; x < 12; x++ {
			canvas.Set(x, 3, '─')
			canvas.Set(x, 5, '─')
		}
		canvas.Set(8, 4, '│')
		canvas.Set(12, 4, '│')

		// Add face
		for i, ch := range face {
			canvas.Set(9+i, 4, ch)
		}

		frames = append(frames, AnimationFrame{Canvas: canvas, Delay: 400})
	}

	return Animation{Frames: frames, Loop: true}
}

// CreateSuccessAnimation creates a celebration animation
func CreateSuccessAnimation() Animation {
	width, height := 30, 12
	frames := []AnimationFrame{}

	// Success sequence
	celebrations := []struct {
		face string
		arms string
	}{
		{"◉‿◉", ""},
		{"◉‿◉", "\\o/"},
		{"◉▿◉", "\\o/"},
		{"◉‿◉", "\\o/"},
	}

	for _, celeb := range celebrations {
		canvas := NewCanvas(width, height)

		// Center the box
		centerX := width / 2

		// Draw box
		canvas.Set(centerX-2, 4, '╭')
		canvas.Set(centerX+2, 4, '╮')
		canvas.Set(centerX-2, 6, '╰')
		canvas.Set(centerX+2, 6, '╯')
		for x := centerX - 1; x < centerX+2; x++ {
			canvas.Set(x, 4, '─')
			canvas.Set(x, 6, '─')
		}
		canvas.Set(centerX-2, 5, '│')
		canvas.Set(centerX+2, 5, '│')

		// Add face
		for i, ch := range celeb.face {
			canvas.Set(centerX-1+i, 5, ch)
		}

		// Add celebration arms if present
		if celeb.arms != "" {
			for i, ch := range celeb.arms {
				canvas.Set(centerX-1+i, 3, ch)
			}
		}

		frames = append(frames, AnimationFrame{Canvas: canvas, Delay: 300})
	}

	return Animation{Frames: frames, Loop: false}
}

// ========================================
// SIMPLE TCELL UI
// ========================================

// RunEditor starts the main editor interface
func (e *Editor) RunEditor() error {
	// Ensure cursor is restored on any exit
	defer func() {
		fmt.Print("\033[?25h") // Show cursor
	}()

	// Start living character animation
	e.StartLivingAnimation()
	defer e.StopLivingAnimation()

	// Set terminal to raw mode for single-key input
	if err := e.setRawMode(); err != nil {
		return err
	}
	defer e.restoreTerminal()

	// Set up signal handling for window resize and interrupts
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGWINCH, syscall.SIGINT, syscall.SIGTERM)

	// Initial render
	e.Render()

	// Animation and input loop
	ticker := time.NewTicker(200 * time.Millisecond) // 5 FPS for smooth animation
	defer ticker.Stop()

	inputChan := make(chan rune, 1)
	errChan := make(chan error, 1)

	// Start non-blocking input reader
	go func() {
		for {
			key, err := e.readKey()
			if err != nil {
				errChan <- err
				return
			}
			inputChan <- key
		}
	}()

	// Initial render to show the start page animation
	e.Render()

	for {
		select {
		case key := <-inputChan:
			if e.handleKey(key) {
				return nil // Exit requested
			}
			e.Render()
		case err := <-errChan:
			return err
		case <-ticker.C:
			// Regular animation refresh
			if e.mode == ModeStartPage && e.isAnimating {
				e.updateStartupAnimation()
			} else if e.mode != ModeStartPage {
				e.Render()
			}
		case sig := <-sigChan:
			switch sig {
			case syscall.SIGWINCH:
				// Terminal was resized - update canvas size
				e.ResizeBuffer()
				e.Render()
			case syscall.SIGINT, syscall.SIGTERM:
				// Graceful shutdown on interrupt
				return nil
			}
		}
	}
}

// setRawMode puts terminal into raw mode for single-key input
func (e *Editor) setRawMode() error {
	cmd := exec.Command("stty", "-echo", "-icanon")
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// restoreTerminal restores normal terminal mode
func (e *Editor) restoreTerminal() {
	// Show cursor and restore terminal
	fmt.Print("\033[?25h") // Show cursor
	cmd := exec.Command("stty", "echo", "icanon")
	cmd.Stdin = os.Stdin
	cmd.Run()
}

// readKey reads a single key press
func (e *Editor) readKey() (rune, error) {
	buffer := make([]byte, 1)
	_, err := os.Stdin.Read(buffer)
	if err != nil {
		return 0, err
	}
	return rune(buffer[0]), nil
}

// handleKey processes single key presses
func (e *Editor) handleKey(key rune) bool {
	// Handle jump mode first
	if e.jumpActive {
		return e.handleJumpKey(key)
	}

	switch e.mode {
	case ModeStartPage:
		return e.handleStartPageKey(key)
	case ModeNormal:
		return e.handleNormalKey(key)
	case ModeInsert:
		return e.handleInsertKey(key)
	case ModeInsertSelect, ModeSelectFrom, ModeSelectTo, ModeDelete:
		return e.handleSelectKey(key)
	case ModeDeleteConfirm:
		return e.handleDeleteConfirmKey(key)
	case ModeLoadConfirm:
		return e.handleLoadConfirmKey(key)
	case ModeSaveConfirm:
		return e.handleSaveConfirmKey(key)
	case ModeHelp:
		return e.handleHelpKey(key)
	case ModeCommand:
		return e.handleCommandKey(key)
	}
	return false
}

// handleStartPageKey processes keys in start page mode
func (e *Editor) handleStartPageKey(key rune) bool {
	// Cancel animation on any key press
	if e.isAnimating {
		e.isAnimating = false
	}

	switch key {
	case 'q', 3: // q or Ctrl+C
		return true // Exit
	case 'n': // New diagram
		e.SetMode(ModeNormal)
	case 'l': // Load diagram
		e.SetMode(ModeCommand)
		e.commandBuffer = "r "
	case '?': // Show help
		e.SetMode(ModeHelp)
	}
	return false
}

// handleNormalKey processes keys in normal mode
func (e *Editor) handleNormalKey(key rune) bool {
	switch key {
	case 'q', 3: // q or Ctrl+C
		return true // Exit
	case 'Q': // Debug quit - print graph structure
		e.printDebugInfo()
		return true // Exit after debug
	case 'a':
		e.SetMode(ModeInsert)
		nodeID := e.AddNode([]string{""}) // Create empty node for editing
		e.currentNode = nodeID
		e.cursorPos = 0 // Start at beginning of empty node
	case 'i':
		// Edit existing node - show jump menu for selection
		if len(e.diagram.Nodes) > 0 {
			e.SetMode(ModeInsertSelect)
			e.startJump()
		}
	case 'c':
		e.SetMode(ModeSelectFrom)
		e.startJump()
	case 'd':
		e.SetMode(ModeDelete)
		e.startJump()
	case 'r': // Resize buffer to fit terminal
		e.ResizeBuffer()
	case '?': // Show help
		e.SetMode(ModeHelp)
	case ':': // Enter command mode
		e.SetMode(ModeCommand)
		e.commandBuffer = ":"
	}
	return false
}

// handleInsertKey processes keys in insert mode
func (e *Editor) handleInsertKey(key rune) bool {
	switch key {
	case 27: // ESC
		e.SetMode(ModeNormal)
	case 127, 8: // Backspace or Delete
		e.handleBackspace()
	case 13, 10: // Enter
		// Create another new node and stay in insert mode
		nodeID := e.AddNode([]string{""})
		e.currentNode = nodeID
		e.cursorPos = 0 // Start at beginning of new node
	case 3: // Ctrl+C
		return true
	default:
		if key >= 32 && key <= 126 { // Printable characters
			e.addCharToCurrentNode(key)
		}
	}
	return false
}

// handleSelectKey processes keys in selection modes
func (e *Editor) handleSelectKey(key rune) bool {
	switch key {
	case 27: // ESC
		e.stopJump()
		e.SetMode(ModeNormal)
		e.connectionFrom = -1
	case 3: // Ctrl+C
		return true
	}
	return false
}

// handleDeleteConfirmKey processes y/N confirmation for delete
func (e *Editor) handleDeleteConfirmKey(key rune) bool {
	switch key {
	case 'y', 'Y':
		// Confirm delete
		if e.connectionFrom >= 0 {
			// Trigger table flip animation sequence
			e.eddCharacter.TriggerTableFlip()
			e.deleteNode(e.connectionFrom)
		}
		// Stay in delete mode after deletion - restart jump selection
		e.SetMode(ModeDelete)
		e.connectionFrom = -1
		e.startJump()
	case 'n', 'N', 27: // N, n, or ESC to cancel
		// Cancel delete - go back to delete mode jump selection
		e.SetMode(ModeDelete)
		e.connectionFrom = -1
		e.startJump()
	case 3: // Ctrl+C
		return true
	}
	return false
}

// handleLoadConfirmKey processes y/N confirmation for file load
func (e *Editor) handleLoadConfirmKey(key rune) bool {
	switch key {
	case 'y', 'Y':
		// Confirm load - proceed with overwriting current diagram
		if e.pendingFilename != "" {
			err := e.loadDiagram(e.pendingFilename)
			if err != nil {
				fmt.Printf("\nError loading: %v", err)
				time.Sleep(2 * time.Second)
			} else {
				fmt.Printf("\nLoaded %s", e.pendingFilename)
				time.Sleep(1 * time.Second)
			}
			e.pendingFilename = ""
		}
		e.SetMode(ModeNormal)
	case 'n', 'N', 27: // N, n, or ESC to cancel
		// Cancel load
		e.pendingFilename = ""
		e.SetMode(ModeNormal)
	case 3: // Ctrl+C
		return true
	}
	return false
}

// handleSaveConfirmKey processes y/N confirmation for file save
func (e *Editor) handleSaveConfirmKey(key rune) bool {
	switch key {
	case 'y', 'Y':
		// Confirm save - proceed with overwriting existing file
		if e.pendingFilename != "" {
			err := e.saveDiagram(e.pendingFilename)
			if err != nil {
				fmt.Printf("\nError saving: %v", err)
				time.Sleep(2 * time.Second)
			} else {
				fmt.Printf("\nSaved to %s", e.pendingFilename)
				time.Sleep(1 * time.Second)
			}
			e.pendingFilename = ""
		}
		e.SetMode(ModeNormal)
	case 'n', 'N', 27: // N, n, or ESC to cancel
		// Cancel save
		e.pendingFilename = ""
		e.SetMode(ModeNormal)
	case 3: // Ctrl+C
		return true
	}
	return false
}

// handleHelpKey processes keys in help mode
func (e *Editor) handleHelpKey(key rune) bool {
	switch key {
	case 27, '?', 'q', 3: // ESC, ?, q, or Ctrl+C to exit help
		e.SetMode(ModeNormal)
	}
	return false
}

// handleCommandKey processes keys in command mode
func (e *Editor) handleCommandKey(key rune) bool {
	switch key {
	case 27: // ESC
		e.SetMode(ModeNormal)
		e.commandBuffer = ""
	case 13, 10: // Enter - execute command
		result := e.executeCommand(e.commandBuffer)
		if result {
			return true // Exit requested
		}
		// Only return to normal mode if we're not in a confirmation mode
		if e.mode != ModeLoadConfirm && e.mode != ModeSaveConfirm {
			e.SetMode(ModeNormal)
			e.commandBuffer = ""
		} else {
			// In confirm mode, clear command buffer but stay in confirm mode
			e.commandBuffer = ""
		}
	case 127, 8: // Backspace
		if len(e.commandBuffer) > 1 { // Keep the ':'
			e.commandBuffer = e.commandBuffer[:len(e.commandBuffer)-1]
		}
	case 3: // Ctrl+C
		return true
	default:
		if key >= 32 && key <= 126 { // Printable characters
			e.commandBuffer += string(key)
		}
	}
	return false
}

// executeCommand processes and executes vim-style commands
func (e *Editor) executeCommand(command string) bool {
	// Remove leading ':'
	if strings.HasPrefix(command, ":") {
		command = command[1:]
	}

	// Split command and arguments
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "w", "write":
		// Save command
		if len(args) == 0 {
			// :w - save to current file
			if e.currentFilename == "" {
				// No current file, need filename
				// For now, just show an error
				fmt.Print("\nNo filename specified")
				time.Sleep(1 * time.Second)
				return false
			}
			err := e.saveDiagram(e.currentFilename)
			if err != nil {
				fmt.Printf("\nError saving: %v", err)
				time.Sleep(2 * time.Second)
			}
		} else {
			// :w filename - save to specific file
			filename := args[0]
			if !strings.HasSuffix(filename, ".edd") {
				filename += ".edd"
			}

			// Check if file exists and ask for confirmation
			if fileExists(filename) {
				// Store filename and ask for confirmation
				e.pendingFilename = filename
				e.SetMode(ModeSaveConfirm)
				return false
			} else {
				// Safe to save directly
				err := e.saveDiagram(filename)
				if err != nil {
					fmt.Printf("\nError saving: %v", err)
					time.Sleep(2 * time.Second)
				} else {
					fmt.Printf("\nSaved to %s", filename)
					time.Sleep(1 * time.Second)
				}
			}
		}
	case "r", "read":
		// Load command
		if len(args) == 0 {
			fmt.Print("\nNo filename specified")
			time.Sleep(1 * time.Second)
			return false
		}

		filename := args[0]
		if !strings.HasSuffix(filename, ".edd") {
			filename += ".edd"
		}

		// Check if we need to confirm overwrite
		if e.hasContent() {
			// Store filename and ask for confirmation
			e.pendingFilename = filename
			e.SetMode(ModeLoadConfirm)
			return false
		} else {
			// Safe to load directly
			err := e.loadDiagram(filename)
			if err != nil {
				fmt.Printf("\nError loading: %v", err)
				time.Sleep(2 * time.Second)
			} else {
				fmt.Printf("\nLoaded %s", filename)
				time.Sleep(1 * time.Second)
			}
		}
	case "q", "quit":
		// Quit command
		return true
	case "wq":
		// Save and quit
		if e.currentFilename != "" {
			err := e.saveDiagram(e.currentFilename)
			if err != nil {
				fmt.Printf("\nError saving: %v", err)
				time.Sleep(2 * time.Second)
				return false
			}
		}
		return true
	default:
		fmt.Printf("\nUnknown command: %s", cmd)
		time.Sleep(1 * time.Second)
	}

	return false
}

// handleJumpKey processes keys when ace-jump is active
func (e *Editor) handleJumpKey(key rune) bool {
	switch key {
	case 27: // ESC
		e.stopJump()
		e.SetMode(ModeNormal)
		e.connectionFrom = -1
	case 3: // Ctrl+C
		return true
	default:
		// Try to select node with this key
		return e.handleJumpSelection(key)
	}
	return false
}

// promptForNodeText shows prompt for node creation
func (e *Editor) promptForNodeText() {
	fmt.Println("Enter node text:")
}

// promptForConnection shows prompt for connection creation
func (e *Editor) promptForConnection() {
	fmt.Println("Enter connection (from,to):")
	for i, node := range e.diagram.Nodes {
		fmt.Printf("  %d: %s\n", node.ID, strings.Join(node.Text, " "))
		_ = i
	}
}

// handleConnectionInput processes connection input like "0,1"
func (e *Editor) handleConnectionInput(input string) {
	parts := strings.Split(input, ",")
	if len(parts) != 2 {
		fmt.Println("Invalid format. Use: from,to (e.g., 0,1)")
		return
	}

	var from, to int
	if _, err := fmt.Sscanf(parts[0], "%d", &from); err != nil {
		fmt.Println("Invalid from node ID")
		return
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &to); err != nil {
		fmt.Println("Invalid to node ID")
		return
	}

	// Add the connection
	conn := Connection{From: from, To: to}
	e.diagram.Connections = append(e.diagram.Connections, conn)
	fmt.Printf("Connected node %d to node %d\n", from, to)
}

// addCharToCurrentNode adds a character to the current node's text at cursor position
func (e *Editor) addCharToCurrentNode(ch rune) {
	if e.currentNode >= 0 {
		// Find the node with the current ID
		for i := range e.diagram.Nodes {
			if e.diagram.Nodes[i].ID == e.currentNode {
				// Initialize text if empty
				if len(e.diagram.Nodes[i].Text) == 0 {
					e.diagram.Nodes[i].Text = []string{""}
				}

				text := e.diagram.Nodes[i].Text[0]

				// Insert character at cursor position
				if e.cursorPos >= len(text) {
					// Append to end
					e.diagram.Nodes[i].Text[0] = text + string(ch)
				} else {
					// Insert in middle
					e.diagram.Nodes[i].Text[0] = text[:e.cursorPos] + string(ch) + text[e.cursorPos:]
				}

				// Move cursor forward
				e.cursorPos++

				// Recalculate node size
				width, height := CalculateNodeSize(e.diagram.Nodes[i].Text)
				e.diagram.Nodes[i].Width = width
				e.diagram.Nodes[i].Height = height
				break
			}
		}
	}
}

// handleBackspace removes character before cursor position
func (e *Editor) handleBackspace() {
	if e.currentNode >= 0 && e.cursorPos > 0 {
		// Find the node with the current ID
		for i := range e.diagram.Nodes {
			if e.diagram.Nodes[i].ID == e.currentNode {
				if len(e.diagram.Nodes[i].Text) > 0 && len(e.diagram.Nodes[i].Text[0]) > 0 {
					text := e.diagram.Nodes[i].Text[0]

					// Remove character before cursor
					if e.cursorPos <= len(text) {
						e.diagram.Nodes[i].Text[0] = text[:e.cursorPos-1] + text[e.cursorPos:]
						e.cursorPos-- // Move cursor back
					}

					// Recalculate node size
					width, height := CalculateNodeSize(e.diagram.Nodes[i].Text)
					e.diagram.Nodes[i].Width = width
					e.diagram.Nodes[i].Height = height
				}
				break
			}
		}
	}
}

// Render draws the interface simply
func (e *Editor) Render() {
	// Clear screen and hide cursor
	fmt.Print("\033[2J\033[H\033[?25l")

	if e.mode == ModeHelp {
		// Show help screen instead of diagram
		e.renderHelp()
	} else if e.mode == ModeStartPage {
		// Show start page instead of diagram
		e.renderStartPage()
	} else {
		// Clear canvas
		e.canvas.Clear()

		// Render diagram
		e.canvas.Render(e.diagram)

		// Display main canvas
		fmt.Print(e.canvas.String())

		// Add jump labels if active or in delete confirm mode (overlay on top)
		if e.jumpActive || e.mode == ModeDeleteConfirm {
			e.renderJumpLabels()
		}

		fmt.Print("\n\n\n")
	}

	// Update and display ed character with color (but not on start page)
	if e.mode != ModeStartPage {
		e.UpdateLivingModeIndicator()
		fmt.Print("\033[33m") // Yellow
		fmt.Print(e.modeIndicator.String())
		fmt.Print("\033[0m") // Reset
	}

	// Display command buffer if in command mode
	if e.mode == ModeCommand {
		fmt.Printf("\n%s", e.commandBuffer)
	}

	// Display load confirmation if in load confirm mode
	if e.mode == ModeLoadConfirm {
		fmt.Printf("\nOverwrite current diagram with %s? (y/N): ", e.pendingFilename)
	}

	// Display save confirmation if in save confirm mode
	if e.mode == ModeSaveConfirm {
		fmt.Printf("\nOverwrite existing file %s? (y/N): ", e.pendingFilename)
	}

	// Position cursor if in insert mode
	e.positionCursor()
}

// renderHelp displays the help dialogue
func (e *Editor) renderHelp() {
	fmt.Print("\033[36m") // Cyan color for help text

	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println("                     EDD - Elegant Diagram Drawer")
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("📚 MODES & NAVIGATION:")
	fmt.Println("  Normal Mode:")
	fmt.Println("    a      - Add new node (enter insert mode)")
	fmt.Println("    i      - Edit existing node (select node to edit)")
	fmt.Println("    c      - Connect nodes (enter select mode)")
	fmt.Println("    d      - Delete nodes/connections")
	fmt.Println("    r      - Resize buffer to fit terminal")
	fmt.Println("    :      - Enter command mode")
	fmt.Println("    ?      - Show this help (you are here!)")
	fmt.Println("    q      - Quit application")
	fmt.Println("    Q      - Debug quit (show graph structure)")
	fmt.Println()
	fmt.Println("  Insert Mode:")
	fmt.Println("    Type   - Add text to current node")
	fmt.Println("    Enter  - Create another new node")
	fmt.Println("    ESC    - Return to normal mode")
	fmt.Println()
	fmt.Println("  Connect Mode:")
	fmt.Println("    Letter - Select node (yellow labels)")
	fmt.Println("           - First: source node (FROM)")
	fmt.Println("           - Second: target node (TO)")
	fmt.Println("           - Stays in connect mode for multiple connections")
	fmt.Println("    ESC    - Return to normal mode")
	fmt.Println()
	fmt.Println("  Delete Mode:")
	fmt.Println("    Letter - Select node (yellow) or connection (red)")
	fmt.Println("           - Nodes: y/N confirmation required")
	fmt.Println("           - Connections: immediate deletion")
	fmt.Println("           - Stays in delete mode for multiple deletions")
	fmt.Println("    ESC    - Return to normal mode")
	fmt.Println()
	fmt.Println("  Command Mode:")
	fmt.Println("    :w filename - Save diagram to .edd file")
	fmt.Println("    :w          - Save to current file")
	fmt.Println("    :r filename - Load diagram from .edd file")
	fmt.Println("    :q          - Quit application")
	fmt.Println("    :wq         - Save and quit")
	fmt.Println("    ESC         - Return to normal mode")
	fmt.Println()
	fmt.Println("🎨 FEATURES:")
	fmt.Println("  • Unicode box drawing with rounded corners")
	fmt.Println("  • Automatic layout and routing")
	fmt.Println("  • Jump-based selection (like ace-jump)")
	fmt.Println("  • Living character 'ed' with mode-specific animations")
	fmt.Println("  • Auto-resizing terminal support")
	fmt.Println("  • Save/load .edd files")
	fmt.Println()
	fmt.Println("💡 USAGE:")
	fmt.Println("  Start: ./edd [filename.edd]")
	fmt.Println("  Load existing diagram or start fresh")
	fmt.Println()
	fmt.Println("Press ESC, ?, or q to return to normal mode")

	fmt.Print("\033[0m") // Reset color
	fmt.Print("\n\n")
}

// renderStartPage displays the welcoming start page
func (e *Editor) renderStartPage() {
	width, height := getTerminalSize()

	// Ed character (center individually)
	edLines := []string{
		" ╭────╮",
		" │◉‿◉ │",
		" ╰────╯",
		"",
		"edd - elegant diagram drawer",
	}

	// Menu commands only (for block alignment)
	commands := []string{
		"n          Start a new diagram",
		"l          Load existing diagram",
		"?          Show help",
		"q          Quit",
	}

	// Calculate total height for vertical centering
	totalHeight := len(edLines) + 1 + 1 + 1 + len(commands) + 1 + 1 // ed + gap + quickstart + gap + commands + gap + tip
	startY := (height - totalHeight) / 2
	if startY < 0 {
		startY = 0
	}

	// Print vertical spacing
	for i := 0; i < startY; i++ {
		fmt.Println()
	}

	// Print ed character (each line centered)
	for _, line := range edLines {
		lineOffset := (width - utf8.RuneCountInString(line)) / 2
		if lineOffset < 0 {
			lineOffset = 0
		}

		if line == "edd - elegant diagram drawer" {
			fmt.Printf("%*s\033[36m%s\033[0m\n", lineOffset, "", line)
		} else {
			fmt.Printf("%*s%s\n", lineOffset, "", line)
		}
	}

	// Gap
	fmt.Println()

	// Find longest command for block positioning
	maxCommandLen := 0
	for _, cmd := range commands {
		if len(cmd) > maxCommandLen {
			maxCommandLen = len(cmd)
		}
	}

	// Center the command block
	commandBlockOffset := (width - maxCommandLen) / 2
	if commandBlockOffset < 0 {
		commandBlockOffset = 0
	}

	// Center QUICK START above the command block
	quickStartOffset := commandBlockOffset + (maxCommandLen-len("QUICK START"))/2
	fmt.Printf("%*s\033[1m%s\033[0m\n", quickStartOffset, "", "QUICK START")
	fmt.Println()

	// Print commands (left-aligned within the centered block)
	for _, cmd := range commands {
		fmt.Printf("%*s\033[36m%s\033[0m\n", commandBlockOffset, "", cmd)
	}

	fmt.Println()

	// Center TIP independently
	tip := "TIP: You can also start with ./edd filename.edd"
	tipOffset := (width - len(tip)) / 2
	fmt.Printf("%*s\033[90m%s\033[0m\n", tipOffset, "", tip)
}

// renderStartMenu displays the quick start menu
func (e *Editor) renderStartMenu() {
	// Get terminal dimensions for centering
	width, height := getTerminalSize()

	// Calculate vertical centering - show content in center
	startY := (height - 10) / 2 // 10 lines of content
	if startY < 0 {
		startY = 0
	}

	// Print empty lines to center vertically
	for i := 0; i < startY; i++ {
		fmt.Println()
	}

	// Center the title
	title := "QUICK START"
	titleX := (width - len(title)) / 2
	fmt.Printf("%*s\033[1m%s\033[0m\n\n", titleX, "", title)

	// Commands block - find the longest line for consistent alignment
	commands := []string{
		"n          Start a new diagram",
		"l          Load existing diagram",
		"?          Show help",
		"q          Quit",
	}

	// Find longest command line
	maxLen := 0
	for _, cmd := range commands {
		if len(cmd) > maxLen {
			maxLen = len(cmd)
		}
	}

	// Center the command block as a unit
	blockX := (width - maxLen) / 2
	for _, cmd := range commands {
		fmt.Printf("%*s\033[36m%s\033[0m\n", blockX, "", cmd)
	}

	fmt.Println()
	fmt.Println()

	// Center the tip
	tip := "TIP: You can also start with ./edd filename.edd"
	tipX := (width - len(tip)) / 2
	fmt.Printf("%*s\033[90m%s\033[0m\n", tipX, "", tip)
}

// renderFinalStartScreen displays the final start screen
func (e *Editor) renderFinalStartScreen() {
	width, height := getTerminalSize()

	content := []string{
		"╭────╮",
		"│◉‿ ◉│",
		"╰────╯",
		"",
		"edd - elegant diagram drawer",
		"",
		"QUICK START",
		"",
		"n          Start a new diagram",
		"l          Load existing diagram",
		"?          Show help",
		"q          Quit",
		"",
		"TIP: You can also start with ./edd filename.edd",
	}

	// Center vertically
	startY := (height - len(content)) / 2
	if startY < 0 {
		startY = 0
	}

	for i := 0; i < startY; i++ {
		fmt.Println()
	}

	// Print each line centered
	for i, line := range content {
		startX := (width - len(line)) / 2
		if startX < 0 {
			startX = 0
		}

		if i <= 2 { // Ed character
			fmt.Printf("%*s\033[33m%s\033[0m\n", startX, "", line)
		} else if i == 4 { // Tagline
			fmt.Printf("%*s\033[36m%s\033[0m\n", startX, "", line)
		} else if i == 6 { // "QUICK START"
			fmt.Printf("%*s\033[1m%s\033[0m\n", startX, "", line)
		} else if i >= 8 && i <= 11 { // Commands
			fmt.Printf("%*s\033[36m%s\033[0m\n", startX, "", line)
		} else if i == 13 { // TIP
			fmt.Printf("%*s\033[90m%s\033[0m\n", startX, "", line)
		} else {
			fmt.Printf("%*s%s\n", startX, "", line)
		}
	}
}

// testStartPageLayout tests the layout at a specific terminal size
func testStartPageLayout(width, height int) {
	// Static content
	allContent := []string{
		"╭────╮",
		"│ ◉‿◉ │",
		"╰────╯",
		"",
		"edd - elegant diagram drawer",
		"",
		"QUICK START",
		"",
		"n          Start a new diagram",
		"l          Load existing diagram",
		"?          Show help",
		"q          Quit",
		"",
		"TIP: You can also start with ./edd filename.edd",
	}

	// Center the entire block vertically
	totalHeight := len(allContent)
	startY := (height - totalHeight) / 2
	if startY < 0 {
		startY = 0
	}

	// Print vertical spacing
	for i := 0; i < startY; i++ {
		fmt.Println()
	}

	// Print all content centered horizontally
	for _, line := range allContent {
		startX := (width - len(line)) / 2
		if startX < 0 {
			startX = 0
		}

		fmt.Printf("%*s%s\n", startX, "", line)
	}
}

// testStartPageSizes tests the layout at different terminal sizes
func testStartPageSizes() {
	sizes := []struct{ w, h int }{
		{80, 24},  // Small terminal
		{120, 30}, // Medium terminal
		{160, 40}, // Large terminal
		{60, 20},  // Very small
	}

	for i, size := range sizes {
		fmt.Printf("=== TEST %d: %dx%d ===\n", i+1, size.w, size.h)
		testStartPageLayout(size.w, size.h)
		fmt.Printf("\n")
	}
}

func main() {
	// Uncomment this line to test the start page layout:
	// testStartPageSizes(); return

	// Start the editor
	editor := NewEditor()

	// Check for command line argument (filename to load)
	if len(os.Args) > 1 {
		filename := os.Args[1]
		if !strings.HasSuffix(filename, ".edd") {
			filename += ".edd"
		}

		fmt.Printf("Loading %s...\n", filename)
		err := editor.loadDiagram(filename)
		if err != nil {
			fmt.Printf("Error loading file: %v\n", err)
			fmt.Println("Starting with empty diagram...")
		} else {
			fmt.Printf("Loaded %s successfully\n", filename)
		}

		// Bypass start page when loading from command line
		editor.SetMode(ModeNormal)
	}

	// fmt.Println("\nStarting editor...")
	// time.Sleep(1 * time.Second)

	err := editor.RunEditor()
	if err != nil {
		fmt.Printf("Editor error: %v\n", err)
	}

	fmt.Println("\nGoodbye from ed! 👋")
}

package main

import (
	"fmt"
	"strings"
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
	ID     int
	X, Y   int      // Top-left position
	Width  int      // Calculated from text + padding
	Height int      // Calculated from text lines + padding
	Text   []string // Lines of text
}

// Connection represents a directed edge between nodes
type Connection struct {
	From, To int     // Node IDs
	Path     []Point // The actual route coordinates
}

// Point represents a coordinate in the canvas
type Point struct {
	X, Y int
	Rune rune // The character to draw at this point
}

// Diagram holds all nodes and connections
type Diagram struct {
	Nodes       []Node
	Connections []Connection
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
		"╰│": '├', "│╰": '├',  // Bottom-left corner + vertical = left T
		"╮│": '┤', "│╮": '┤',  // Top-right corner + vertical = right T
		"╭│": '├', "│╭": '├',  // Top-left corner + vertical = left T
		"╯│": '┤', "│╯": '┤',  // Bottom-right corner + vertical = right T
		
		"╰─": '┴', "─╰": '┴',  // Bottom-left corner + horizontal = bottom T
		"╯─": '┴', "─╯": '┴',  // Bottom-right corner + horizontal = bottom T
		"╮─": '┬', "─╮": '┬',  // Top-right corner + horizontal = top T
		"╭─": '┬', "─╭": '┬',  // Top-left corner + horizontal = top T
		
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
	From        int          // Source node ID
	Targets     []int        // Target node IDs
	TrunkX      int          // X position of shared vertical trunk
	ExitY       int          // Y position where connections exit the source
	ExitSide    Side         // Which side of the source node
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
				return routeHorizontalAtLine(from, to, from.Y + 1) // First content line
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


// routeHorizontalLeft creates a horizontal path going left
func routeHorizontalLeft(from, to Node) []Point {
	var path []Point
	
	startX := from.X
	startY := from.Y + from.Height/2
	endX := to.X + to.Width - 1
	endY := to.Y + to.Height/2
	
	// Exit junction on left
	path = append(path, Point{X: startX, Y: startY, Rune: '┤'})
	
	// Horizontal line
	for x := startX - 1; x > endX + 1; x-- {
		path = append(path, Point{X: x, Y: startY, Rune: '─'})
	}
	
	// Arrow pointing left at the target
	path = append(path, Point{X: endX + 1, Y: endY, Rune: '◀'})
	
	return path
}

// routeDiagonalSimple creates a simple L-shaped path
func routeDiagonalSimple(from, to Node) []Point {
	var path []Point
	
	// Debug - uncomment to trace routing
	// fmt.Printf("DIAGONAL: from=(%d,%d,%dx%d) to=(%d,%d,%dx%d)\n", 
	//     from.X, from.Y, from.Width, from.Height, to.X, to.Y, to.Width, to.Height)
	
	// Exit from the side closest to target
	if to.X > from.X + from.Width {
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
		for x := trunkX + 1; x < targetX - 1; x++ {
			path = append(path, Point{X: x, Y: targetY, Rune: '─'})
		}
		// Arrow should be just before the target box  
		if targetX - 1 > trunkX {
			// There's space between corner and target - place arrow just before target
			path = append(path, Point{X: targetX - 1, Y: targetY, Rune: '▶'})
		} else {
			// Corner is adjacent to target - replace corner with arrow
			path[len(path)-1] = Point{X: trunkX, Y: targetY, Rune: '▶'}
		}
		
	} else if to.X + to.Width < from.X {
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
		for x := trunkX - 1; x > targetX + to.Width; x-- {
			path = append(path, Point{X: x, Y: targetY, Rune: '─'})
		}
		// Arrow should be just after the target box (at the right border)
		if targetX + to.Width < trunkX {
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
		path := SimpleOrthogonalRoute(from, to)
		c.DrawConnection(path)
	}
}

// routeSelfConnection creates a loop from a node back to itself
func routeSelfConnection(node Node) []Point {
	var path []Point
	
	// Exit from the right side at content line
	exitX := node.X + node.Width - 1
	exitY := node.Y + 1  // Content line (Y=1 for height=3 box)
	
	// Loop coordinates based on expected output
	rightX := exitX + 2     // Two spaces to the right
	bottomY := node.Y + 4   // Two lines below the box
	leftX := node.X + 2     // Two spaces from left edge
	upY := node.Y + 3       // One line below the box
	
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
		startY := from.Y + from.Height - 1  // Bottom edge
		endY := to.Y  // Top edge of target
		
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
		startY := from.Y  // Top edge
		endY := to.Y + to.Height - 1  // Bottom edge of target
		
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
	Layers    []Layer
	NodeSpacing int // Horizontal space between nodes
	LayerSpacing int // Vertical space between layers
	NodeWidth int   // Standard node width
	NodeHeight int  // Standard node height
}

// NewLayeredLayout creates a layout with reasonable defaults
func NewLayeredLayout() *LayeredLayout {
	return &LayeredLayout{
		NodeSpacing:  4, // 4 chars between nodes
		LayerSpacing: 4, // 4 lines between layers  
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
	
	// Step 2: Build adjacency list for topological sort
	graph := make(map[int][]int)
	inDegree := make(map[int]int)
	
	for _, node := range nodes {
		graph[node.ID] = []int{}
		inDegree[node.ID] = 0
	}
	
	for _, conn := range connections {
		graph[conn.From] = append(graph[conn.From], conn.To)
		inDegree[conn.To]++
	}
	
	// Step 3: Topological sort to determine columns (left-to-right dependency levels)
	columns := [][]int{}
	remaining := make(map[int]bool)
	for _, node := range nodes {
		remaining[node.ID] = true
	}
	
	for len(remaining) > 0 {
		// Find nodes with no incoming edges (can be placed in current column)
		currentColumn := []int{}
		for nodeID := range remaining {
			if inDegree[nodeID] == 0 {
				currentColumn = append(currentColumn, nodeID)
			}
		}
		
		if len(currentColumn) == 0 {
			// Cycle detected - break it by taking any remaining node
			for nodeID := range remaining {
				currentColumn = append(currentColumn, nodeID)
				break
			}
		}
		
		columns = append(columns, currentColumn)
		
		// Remove these nodes and update in-degrees
		for _, nodeID := range currentColumn {
			delete(remaining, nodeID)
			for _, neighbor := range graph[nodeID] {
				inDegree[neighbor]--
			}
		}
	}
	
	// Step 4: Calculate column widths and positions
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
	
	// Step 5: Assign X positions based on columns
	currentX := 0
	columnStartX := make([]int, len(columns))
	for colIdx := range columns {
		columnStartX[colIdx] = currentX
		currentX += columnWidths[colIdx] + l.NodeSpacing
	}
	
	// Step 6: Assign Y positions within each column (center-aligned, evenly distributed)
	for colIdx, column := range columns {
		if len(column) == 1 {
			// Single node - center vertically
			nodeID := column[0]
			node := nodeMap[nodeID]
			node.X = columnStartX[colIdx]
			node.Y = 0 // Start at top for now
		} else {
			// Multiple nodes - distribute vertically
			for nodeIdx, nodeID := range column {
				node := nodeMap[nodeID]
				node.X = columnStartX[colIdx]
				node.Y = nodeIdx * (node.Height + l.LayerSpacing)
			}
		}
	}
	
	return result
}

// SimpleOrthogonalRoute creates clean orthogonal paths
func SimpleOrthogonalRoute(from, to Node) []Point {
	var path []Point
	
	// Exit from center-right of source box border
	startX := from.X + from.Width - 1  // On the border, not past it
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

func main() {
	fmt.Println("edd - elegant diagram drawer")
}
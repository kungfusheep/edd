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
		c.cells[y*c.width+x] = r
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
		for x := 0; x < c.width; x++ {
			sb.WriteRune(c.cells[y*c.width+x])
		}
		if y < c.height-1 {
			sb.WriteRune('\n')
		}
	}
	return sb.String()
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
			// Multiple connections - need shared trunk
			group.ExitSide = SideRight // Default for now
			group.TrunkX = source.X + source.Width + TrunkOffset
			group.ExitY = source.Y + source.Height/2
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
	
	// Check for bidirectional connection
	if isBidirectional {
		// This is part of a bidirectional pair
		// Route upper connection for smaller ID, lower for larger
		return routeBidirectional(from, to, conn.From < conn.To)
	}
	
	// Find the group this connection belongs to
	var group *ConnectionGroup
	for i := range plan.Groups {
		if plan.Groups[i].From == conn.From {
			group = &plan.Groups[i]
			break
		}
	}
	
	// If multiple targets, we need trunk routing
	if group != nil && len(group.Targets) > 1 {
		return routeWithSharedTrunk(from, to, group, nodes)
	}
	
	// Simple routing for single connections
	if from.Y == to.Y && from.X < to.X {
		// Horizontal
		return routeHorizontalSimple(from, to)
	} else if from.Y == to.Y && from.X > to.X {
		// Horizontal but going left
		return routeHorizontalLeft(from, to)
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

// routeBidirectional handles connections that go both ways between nodes
func routeBidirectional(from, to Node, isUpper bool) []Point {
	var path []Point
	
	// For horizontal bidirectional connections at same Y level
	if from.Y == to.Y && from.X < to.X {
		// Both connections between same two nodes, going right/left
		if isUpper {
			// Connection from left to right on line 1
			startX := from.X + from.Width - 1
			startY := from.Y + 1
			endX := to.X
			
			path = append(path, Point{X: startX, Y: startY, Rune: '├'})
			for x := startX + 1; x < endX; x++ {
				path = append(path, Point{X: x, Y: startY, Rune: '─'})
			}
			path = append(path, Point{X: endX - 1, Y: startY, Rune: '▶'})
		} else {
			// Connection from right to left on line 2
			startX := to.X + to.Width - 1
			startY := to.Y + 2  
			endX := from.X
			
			path = append(path, Point{X: startX, Y: startY, Rune: '┤'})
			for x := startX - 1; x > endX; x-- {
				path = append(path, Point{X: x, Y: startY, Rune: '─'})
			}
			path = append(path, Point{X: endX, Y: startY, Rune: '◀'})
		}
	} else if from.Y == to.Y && from.X > to.X {
		// Both connections between same two nodes, but this one goes left
		if isUpper {
			// Connection from right to left on line 1
			startX := from.X
			startY := from.Y + 1
			endX := to.X + to.Width - 1
			
			path = append(path, Point{X: startX, Y: startY, Rune: '┤'})
			for x := startX - 1; x > endX; x-- {
				path = append(path, Point{X: x, Y: startY, Rune: '─'})
			}
			path = append(path, Point{X: endX + 1, Y: startY, Rune: '◀'})
		} else {
			// Connection from left to right on line 2
			startX := to.X + to.Width - 1
			startY := to.Y + 2
			endX := from.X
			
			path = append(path, Point{X: startX, Y: startY, Rune: '├'})
			for x := startX + 1; x < endX; x++ {
				path = append(path, Point{X: x, Y: startY, Rune: '─'})
			}
			path = append(path, Point{X: endX - 1, Y: startY, Rune: '▶'})
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
	
	// Arrow pointing left
	path = append(path, Point{X: startX - 1, Y: startY, Rune: '◀'})
	
	// Horizontal line
	for x := endX + 1; x < startX - 1; x++ {
		path = append(path, Point{X: x, Y: startY, Rune: '─'})
	}
	
	// Entry junction on right
	path = append(path, Point{X: endX, Y: endY, Rune: '├'})
	
	return path
}

// routeDiagonalSimple creates a simple L-shaped path
func routeDiagonalSimple(from, to Node) []Point {
	var path []Point
	
	// For now, just create placeholder paths
	// We'll implement proper routing after we see the test results
	
	return path
}

// DrawConnection draws a connection path
func (c *Canvas) DrawConnection(path []Point) {
	for _, p := range path {
		c.Set(p.X, p.Y, p.Rune)
	}
}

// Render draws the entire diagram to the canvas
func (c *Canvas) Render(diagram Diagram) {
	// Plan all routing first
	plan := PlanRouting(diagram.Nodes, diagram.Connections)
	
	// First draw all nodes
	for _, node := range diagram.Nodes {
		c.DrawBox(node)
	}
	
	// Then draw connections (which will overwrite box borders where needed)
	for _, path := range plan.Connections {
		c.DrawConnection(path)
	}
}

func main() {
	fmt.Println("edd - elegant diagram drawer")
}
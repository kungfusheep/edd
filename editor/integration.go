package editor

import (
	"edd/diagram"
	"edd/layout"
	"edd/pathfinding"
	"edd/render"
	"fmt"
	"os"
	"strings"
)

// RealRenderer wraps our actual modular renderer for the TUI
type RealRenderer struct {
	mainRenderer *render.Renderer  // The actual refactored renderer
	layout     diagram.LayoutEngine
	pathfinder diagram.PathFinder
	router     *pathfinding.Router
	capabilities render.TerminalCapabilities
	pathRenderer *render.PathRenderer
	nodeRenderer *render.NodeRenderer
	labelRenderer *render.LabelRenderer
	
	// Edit state for rendering cursor in nodes
	editingNodeID int
	editText      string
	cursorPos     int

	// Edit state for rendering cursor in connection labels
	EditingConnectionID int // Made public for debugging
	EditConnectionText  string
	EditConnectionCursorPos int
}

// SetEditState sets the current editing state for in-node text editing
func (r *RealRenderer) SetEditState(nodeID int, text string, cursorPos int) {
	r.editingNodeID = nodeID
	r.editText = text
	r.cursorPos = cursorPos

	// Debug log
	if f, err := os.OpenFile("/tmp/edd_edit_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		fmt.Fprintf(f, "SetEditState: nodeID=%d, text='%s', cursorPos=%d\n", nodeID, text, cursorPos)
		f.Close()
	}
}

// SetConnectionEditState sets the current editing state for connection label editing
func (r *RealRenderer) SetConnectionEditState(connectionID int, text string, cursorPos int) {
	r.EditingConnectionID = connectionID
	r.EditConnectionText = text
	r.EditConnectionCursorPos = cursorPos

	// Debug log
	if f, err := os.OpenFile("/tmp/edd_edit_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		fmt.Fprintf(f, "SetConnectionEditState: connectionID=%d, text='%s', cursorPos=%d\n", connectionID, text, cursorPos)
		f.Close()
	}
}

// GetEditingNodeID returns the node being edited
func (r *RealRenderer) GetEditingNodeID() int {
	return r.editingNodeID
}

// GetEditText returns the text being edited
func (r *RealRenderer) GetEditText() string {
	return r.editText
}

// GetCursorPos returns the cursor position
func (r *RealRenderer) GetCursorPos() int {
	return r.cursorPos
}

// NewRealRenderer creates a renderer using our actual modules
func NewRealRenderer() *RealRenderer {
	// Use the actual refactored renderer that supports colors and proper separation
	mainRenderer := render.NewRenderer()

	// Keep the old structure for compatibility but delegate to the real renderer
	// Default to vertical layout for flowcharts
	// Note: The actual layout will be chosen based on diagram type in RenderWithPositions
	layoutEngine := layout.NewVerticalLayout()
	
	// Use smart pathfinder with caching
	pathfinder := pathfinding.NewSmartPathFinder(pathfinding.PathCost{
		StraightCost:  10,
		TurnCost:      100,
		ProximityCost: 0,
		DirectionBias: 0,
	})
	cachedPathfinder := pathfinding.NewCachedPathFinder(pathfinder, 100)
	
	// Create router
	router := pathfinding.NewRouter(cachedPathfinder)
	
	// Terminal capabilities
	caps := render.TerminalCapabilities{
		UnicodeLevel: render.UnicodeFull,
		SupportsColor: true,
	}
	
	return &RealRenderer{
		mainRenderer:  mainRenderer,  // Store the real renderer
		layout:        layoutEngine,
		pathfinder:    cachedPathfinder,
		router:        router,
		capabilities:  caps,
		pathRenderer:  render.NewPathRenderer(caps),
		nodeRenderer:  render.NewNodeRenderer(caps),
		labelRenderer: render.NewLabelRenderer(),
		editingNodeID: -1,
		EditingConnectionID: -1,
	}
}

// NodePositions stores the last rendered node positions and connection paths
type NodePositions struct {
	Positions        map[int]diagram.Point // Node ID -> position
	ConnectionPaths  map[int]diagram.Path  // Connection index -> path
	Offset           diagram.Point         // Canvas offset used during rendering
}

// Render implements the diagram.Renderer interface for TUI
func (r *RealRenderer) Render(d *diagram.Diagram) (string, error) {
	// Delegate to RenderWithPositions and discard positions
	_, output, err := r.RenderWithPositions(d)
	return output, err
}

// RenderWithPositions renders and returns node positions for jump labels
func (r *RealRenderer) RenderWithPositions(d *diagram.Diagram) (*NodePositions, string, error) {
	if d == nil || len(d.Nodes) == 0 {
		return &NodePositions{Positions: make(map[int]diagram.Point)}, "", nil
	}

	// Check if this is a sequence diagram - use old rendering path for now
	if d.Type == "sequence" {
		return r.renderSequenceWithPositions(d)
	}

	// For flowcharts/box diagrams, delegate to the main renderer
	// Get the flowchart renderer from the main renderer
	flowchartRenderer := r.mainRenderer.GetFlowchartRenderer()
	if flowchartRenderer == nil {
		return &NodePositions{Positions: make(map[int]diagram.Point)}, "", fmt.Errorf("no flowchart renderer available")
	}

	// Set edit state
	flowchartRenderer.SetEditState(r.editingNodeID, r.editText, r.cursorPos)

	// Render and get positions
	positions, paths, output, err := flowchartRenderer.RenderWithPositions(d)
	if err != nil {
		return nil, "", err
	}

	// Convert to NodePositions format
	nodePos := &NodePositions{
		Positions:       positions,
		ConnectionPaths: paths,
		Offset:          diagram.Point{X: 0, Y: 0},
	}

	return nodePos, output, nil
}


// renderSequenceWithPositions renders a sequence diagram and returns positions
func (r *RealRenderer) renderSequenceWithPositions(d *diagram.Diagram) (*NodePositions, string, error) {
	// If we're editing, create a copy of the diagram with the edited text
	renderDiagram := d
	if r.editingNodeID >= 0 {
		// Make a shallow copy of the diagram
		tempDiagram := *d
		// Make a copy of the nodes slice
		tempNodes := make([]diagram.Node, len(d.Nodes))
		copy(tempNodes, d.Nodes)
		tempDiagram.Nodes = tempNodes

		// Update the text of the node being edited
		for i := range tempDiagram.Nodes {
			if tempDiagram.Nodes[i].ID == r.editingNodeID {
				// Replace the node's text with the edit buffer text
				lines := strings.Split(r.editText, "\n")
				tempDiagram.Nodes[i].Text = lines
				break
			}
		}
		renderDiagram = &tempDiagram
	} else if r.EditingConnectionID >= 0 {
		// If we're editing a connection label, create a copy with edited text
		tempDiagram := *d
		// Make a copy of the connections slice
		tempConnections := make([]diagram.Connection, len(d.Connections))
		copy(tempConnections, d.Connections)
		tempDiagram.Connections = tempConnections

		// Update the label of the connection being edited with cursor
		if r.EditingConnectionID < len(tempDiagram.Connections) {
			// Build the label with cursor
			labelWithCursor := r.buildLabelWithCursor(r.EditConnectionText, r.EditConnectionCursorPos)
			tempDiagram.Connections[r.EditingConnectionID].Label = labelWithCursor
		}
		renderDiagram = &tempDiagram
	}
	
	// Create sequence renderer
	seqRenderer := render.NewSequenceRenderer(r.capabilities)
	
	// Get bounds
	width, height := seqRenderer.GetBounds(renderDiagram)
	if width <= 0 || height <= 0 {
		return nil, "", fmt.Errorf("invalid bounds: %dx%d", width, height)
	}
	
	// Check if we need color support
	needsColor := render.HasColorHints(renderDiagram)
	
	// Create appropriate canvas
	var c render.Canvas
	var coloredCanvas *render.ColoredMatrixCanvas
	if needsColor && r.capabilities.SupportsColor {
		coloredCanvas = render.NewColoredMatrixCanvas(width, height)
		c = coloredCanvas
	} else {
		c = render.NewMatrixCanvas(width, height)
	}
	
	// Render the sequence diagram (with edited text if applicable)
	if err := seqRenderer.RenderToCanvas(renderDiagram, c); err != nil {
		return nil, "", fmt.Errorf("failed to render sequence diagram: %w", err)
	}
	
	// Compute positions for jump labels
	seqLayout := layout.NewSequenceLayout()
	positionData := seqLayout.ComputePositions(d)
	
	// Collect positions for editor
	positions := &NodePositions{
		Positions:       make(map[int]diagram.Point),
		ConnectionPaths: make(map[int]diagram.Path),
	}
	
	// Add participant positions
	for nodeID, pos := range positionData.Participants {
		positions.Positions[nodeID] = diagram.Point{X: pos.X, Y: pos.Y}
	}
	
	// Add message paths
	for i, msg := range positionData.Messages {
		if i < len(d.Connections) {
			positions.ConnectionPaths[i] = diagram.Path{
				Points: []diagram.Point{
					{X: msg.FromX, Y: msg.Y},
					{X: msg.ToX, Y: msg.Y},
				},
			}
		}
	}
	
	// Get the output string
	var output string
	if coloredCanvas != nil {
		// Use colored output if we have a colored canvas
		output = coloredCanvas.ColoredString()
	} else {
		// Regular output
		output = c.String()
	}
	
	return positions, output, nil
}

// Helper functions from renderer.go
func calculateNodeDimensions(nodes []diagram.Node) []diagram.Node {
	result := make([]diagram.Node, len(nodes))
	copy(result, nodes)
	
	for i := range result {
		maxWidth := 0
		for _, line := range result[i].Text {
			if len(line) > maxWidth {
				maxWidth = len(line)
			}
		}
		// Minimum width of 8 characters for empty nodes
		if maxWidth < 4 {
			maxWidth = 4
		}
		result[i].Width = maxWidth + 4  // padding
		result[i].Height = len(result[i].Text) + 2  // borders
	}
	
	return result
}

func calculateBounds(nodes []diagram.Node, paths map[int]diagram.Path) diagram.Bounds {
	if len(nodes) == 0 {
		return diagram.Bounds{Min: diagram.Point{X: 0, Y: 0}, Max: diagram.Point{X: 80, Y: 24}}
	}
	
	minX, minY := nodes[0].X, nodes[0].Y
	maxX, maxY := nodes[0].X+nodes[0].Width, nodes[0].Y+nodes[0].Height
	
	for _, node := range nodes[1:] {
		if node.X < minX { minX = node.X }
		if node.Y < minY { minY = node.Y }
		if node.X+node.Width > maxX { maxX = node.X + node.Width }
		if node.Y+node.Height > maxY { maxY = node.Y + node.Height }
	}
	
	for _, path := range paths {
		for _, point := range path.Points {
			if point.X < minX { minX = point.X }
			if point.Y < minY { minY = point.Y }
			if point.X > maxX { maxX = point.X }
			if point.Y > maxY { maxY = point.Y }
		}
	}
	
	padding := 2
	return diagram.Bounds{
		Min: diagram.Point{X: minX - padding, Y: minY - padding},
		Max: diagram.Point{X: maxX + padding, Y: maxY + padding},
	}
}

// drawSimpleBox draws a basic box without any special styling
func drawSimpleBox(c render.Canvas, node diagram.Node, style render.NodeStyle) {
	// Top border
	c.Set(diagram.Point{X: node.X, Y: node.Y}, style.TopLeft)
	for x := node.X + 1; x < node.X + node.Width - 1; x++ {
		c.Set(diagram.Point{X: x, Y: node.Y}, style.Horizontal)
	}
	c.Set(diagram.Point{X: node.X + node.Width - 1, Y: node.Y}, style.TopRight)
	
	// Side borders
	for y := node.Y + 1; y < node.Y + node.Height - 1; y++ {
		c.Set(diagram.Point{X: node.X, Y: y}, style.Vertical)
		c.Set(diagram.Point{X: node.X + node.Width - 1, Y: y}, style.Vertical)
	}
	
	// Bottom border
	c.Set(diagram.Point{X: node.X, Y: node.Y + node.Height - 1}, style.BottomLeft)
	for x := node.X + 1; x < node.X + node.Width - 1; x++ {
		c.Set(diagram.Point{X: x, Y: node.Y + node.Height - 1}, style.Horizontal)
	}
	c.Set(diagram.Point{X: node.X + node.Width - 1, Y: node.Y + node.Height - 1}, style.BottomRight)
}

// buildLabelWithCursor builds a label string with cursor for direct inclusion
func (r *RealRenderer) buildLabelWithCursor(text string, cursorPos int) string {
	runes := []rune(text)
	maxVisibleChars := 15 // Allow more chars for sequence diagrams

	var displayText []rune
	var adjustedCursorPos int

	if len(runes) <= maxVisibleChars {
		// Text fits, use it all
		displayText = runes
		adjustedCursorPos = cursorPos
	} else {
		// Text too long, need to window it
		// Keep cursor visible by showing a window around it
		if cursorPos <= 8 {
			// Cursor near start, show beginning
			displayText = runes[:maxVisibleChars]
			adjustedCursorPos = cursorPos
		} else if cursorPos >= len(runes) - 7 {
			// Cursor near end, show end
			start := len(runes) - maxVisibleChars
			displayText = runes[start:]
			adjustedCursorPos = cursorPos - start
		} else {
			// Cursor in middle, center window on cursor
			start := cursorPos - 7
			if start < 0 {
				start = 0
			}
			end := start + maxVisibleChars
			if end > len(runes) {
				end = len(runes)
				start = end - maxVisibleChars
			}
			displayText = runes[start:end]
			adjustedCursorPos = cursorPos - start
		}
	}

	// Now build string with cursor
	if adjustedCursorPos >= 0 && adjustedCursorPos <= len(displayText) {
		before := string(displayText[:adjustedCursorPos])
		after := ""
		if adjustedCursorPos < len(displayText) {
			after = string(displayText[adjustedCursorPos:])
		}
		return before + "█" + after
	}
	// Shouldn't happen but failsafe
	return string(displayText) + "█"
}

func renderLabelWithCursor(labelRenderer *render.LabelRenderer, c render.Canvas, path diagram.Path, text string, cursorPos int) {
	// The label renderer truncates at 10 chars total (including brackets it adds)
	// So content can be 8 chars: [12345678] plus .. for truncation: [123456..]
	// When editing, we need to show the cursor, which takes 1 char
	// So we can show at most 7 chars of actual text plus cursor

	runes := []rune(text)
	maxVisibleChars := 7 // Room for 7 chars + 1 cursor in the 8-char content space

	var displayText []rune
	var adjustedCursorPos int

	if len(runes) <= maxVisibleChars {
		// Text fits, use it all
		displayText = runes
		adjustedCursorPos = cursorPos
	} else {
		// Text too long, need to window it
		// Keep cursor visible by showing a window around it
		if cursorPos <= 4 {
			// Cursor near start, show beginning
			displayText = runes[:maxVisibleChars]
			adjustedCursorPos = cursorPos
		} else if cursorPos >= len(runes) - 3 {
			// Cursor near end, show end
			start := len(runes) - maxVisibleChars
			displayText = runes[start:]
			adjustedCursorPos = cursorPos - start
		} else {
			// Cursor in middle, center window on cursor
			start := cursorPos - 3
			if start < 0 {
				start = 0
			}
			end := start + maxVisibleChars
			if end > len(runes) {
				end = len(runes)
				start = end - maxVisibleChars
			}
			displayText = runes[start:end]
			adjustedCursorPos = cursorPos - start
		}
	}

	// Now build string with cursor
	var labelWithCursor string
	if adjustedCursorPos >= 0 && adjustedCursorPos <= len(displayText) {
		before := string(displayText[:adjustedCursorPos])
		after := ""
		if adjustedCursorPos < len(displayText) {
			after = string(displayText[adjustedCursorPos:])
		}
		labelWithCursor = before + "█" + after
	} else {
		// Shouldn't happen but failsafe
		labelWithCursor = string(displayText) + "█"
	}

	// Use the standard label renderer
	// It will add brackets and won't truncate since we pre-sized the content
	labelRenderer.RenderLabel(c, path, labelWithCursor, render.LabelMiddle)
}

// Removed - no longer needed since we use the standard label renderer

func renderNodeWithEdit(c render.Canvas, node diagram.Node, nodeRenderer *render.NodeRenderer, isEditing bool, editText string, cursorPos int) {
	// If not editing, use NodeRenderer to draw with styles
	if !isEditing {
		nodeRenderer.RenderNode(c, node)
		return
	}
	
	// When editing, draw a simple box without special styles
	// (to avoid visual noise during editing)
	style := render.NodeStyles["sharp"] // Use sharp style for editing
	drawSimpleBox(c, node, style)
	
	// Draw the edit text with cursor, handling multi-line
	// Split text by newlines
	// Debug log
	if f, err := os.OpenFile("/tmp/edd_edit_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		fmt.Fprintf(f, "renderNodeWithEdit: nodeID=%d, editText='%s', cursorPos=%d\n", node.ID, editText, cursorPos)
		f.Close()
	}
	lines := strings.Split(editText, "\n")
	
	for lineIdx, line := range lines {
		y := node.Y + 1 + lineIdx
		x := node.X + 2
		
		// Don't draw lines outside the box
		if y >= node.Y+node.Height-1 {
			break
		}
		
		// Draw each character of this line
		for i, ch := range line {
			if x+i < node.X+node.Width-2 {
				c.Set(diagram.Point{X: x + i, Y: y}, ch)
			}
		}
	}
}

// offsetCanvas implementation (from renderer.go)
type offsetCanvas struct {
	canvas render.Canvas
	offset diagram.Point
}

func newOffsetCanvas(c render.Canvas, offset diagram.Point) *offsetCanvas {
	return &offsetCanvas{
		canvas: c,
		offset: offset,
	}
}

func (oc *offsetCanvas) Set(p diagram.Point, char rune) error {
	translated := diagram.Point{
		X: p.X - oc.offset.X,
		Y: p.Y - oc.offset.Y,
	}
	return oc.canvas.Set(translated, char)
}

// SetWithColor sets a character with color if the underlying canvas supports it
func (oc *offsetCanvas) SetWithColor(p diagram.Point, char rune, color string) error {
	translated := diagram.Point{
		X: p.X - oc.offset.X,
		Y: p.Y - oc.offset.Y,
	}
	// Try to set with color if the underlying canvas supports it
	if coloredCanvas, ok := oc.canvas.(*render.ColoredMatrixCanvas); ok {
		return coloredCanvas.SetWithColor(translated, char, color)
	}
	// Fall back to regular set
	return oc.canvas.Set(translated, char)
}

// SetWithColorAndStyle sets a character with color and style if the underlying canvas supports it
func (oc *offsetCanvas) SetWithColorAndStyle(p diagram.Point, char rune, color string, style string) error {
	translated := diagram.Point{
		X: p.X - oc.offset.X,
		Y: p.Y - oc.offset.Y,
	}
	// Try to set with color and style if the underlying canvas supports it
	if coloredCanvas, ok := oc.canvas.(*render.ColoredMatrixCanvas); ok {
		return coloredCanvas.SetWithColorAndStyle(translated, char, color, style)
	}
	// Fall back to regular set with color
	return oc.SetWithColor(p, char, color)
}

func (oc *offsetCanvas) Get(p diagram.Point) rune {
	translated := diagram.Point{
		X: p.X - oc.offset.X,
		Y: p.Y - oc.offset.Y,
	}
	return oc.canvas.Get(translated)
}

func (oc *offsetCanvas) Size() (int, int) {
	return oc.canvas.Size()
}

func (oc *offsetCanvas) Clear() {
	oc.canvas.Clear()
}

func (oc *offsetCanvas) String() string {
	return oc.canvas.String()
}

func (oc *offsetCanvas) Matrix() [][]rune {
	if mc, ok := oc.canvas.(*render.MatrixCanvas); ok {
		return mc.Matrix()
	}
	if cc, ok := oc.canvas.(*render.ColoredMatrixCanvas); ok {
		return cc.Matrix()
	}
	return nil
}

func (oc *offsetCanvas) Offset() diagram.Point {
	return oc.offset
}
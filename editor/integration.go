package editor

import (
	"edd/core"
	"edd/layout"
	"edd/pathfinding"
	"edd/connections"
	"edd/canvas"
	"edd/rendering"
	"fmt"
	"strings"
)

// RealRenderer wraps our actual modular renderer for the TUI
type RealRenderer struct {
	mainRenderer *rendering.Renderer  // The actual refactored renderer
	layout     core.LayoutEngine
	pathfinder core.PathFinder
	router     *connections.Router
	capabilities canvas.TerminalCapabilities
	pathRenderer *canvas.PathRenderer
	nodeRenderer *canvas.NodeRenderer
	labelRenderer *canvas.LabelRenderer
	
	// Edit state for rendering cursor in nodes
	editingNodeID int
	editText      string
	cursorPos     int
}

// SetEditState sets the current editing state for in-node text editing
func (r *RealRenderer) SetEditState(nodeID int, text string, cursorPos int) {
	r.editingNodeID = nodeID
	r.editText = text
	r.cursorPos = cursorPos
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
	mainRenderer := rendering.NewRenderer()
	
	// Keep the old structure for compatibility but delegate to the real renderer
	// Use simple layout
	layoutEngine := layout.NewSimpleLayout()
	
	// Use smart pathfinder with caching
	pathfinder := pathfinding.NewSmartPathFinder(pathfinding.PathCost{
		StraightCost:  10,
		TurnCost:      100,
		ProximityCost: 0,
		DirectionBias: 0,
	})
	cachedPathfinder := pathfinding.NewCachedPathFinder(pathfinder, 100)
	
	// Create router
	router := connections.NewRouter(cachedPathfinder)
	
	// Terminal capabilities
	caps := canvas.TerminalCapabilities{
		UnicodeLevel: canvas.UnicodeFull,
		SupportsColor: true,
	}
	
	return &RealRenderer{
		mainRenderer:  mainRenderer,  // Store the real renderer
		layout:        layoutEngine,
		pathfinder:    cachedPathfinder,
		router:        router,
		capabilities:  caps,
		pathRenderer:  canvas.NewPathRenderer(caps),
		nodeRenderer:  canvas.NewNodeRenderer(caps),
		labelRenderer: canvas.NewLabelRenderer(),
		editingNodeID: -1,
	}
}

// NodePositions stores the last rendered node positions and connection paths
type NodePositions struct {
	Positions        map[int]core.Point // Node ID -> position
	ConnectionPaths  map[int]core.Path  // Connection index -> path
	Offset           core.Point         // Canvas offset used during rendering
}

// Render implements the core.Renderer interface for TUI
func (r *RealRenderer) Render(diagram *core.Diagram) (string, error) {
	// Use the main renderer which properly handles colors and diagram types
	if r.mainRenderer != nil {
		return r.mainRenderer.Render(diagram)
	}
	// Fallback to old implementation if needed
	positions, output, err := r.RenderWithPositions(diagram)
	_ = positions // Will be used by TUI for jump labels
	return output, err
}

// RenderWithPositions renders and returns node positions for jump labels
func (r *RealRenderer) RenderWithPositions(diagram *core.Diagram) (*NodePositions, string, error) {
	if diagram == nil || len(diagram.Nodes) == 0 {
		return &NodePositions{Positions: make(map[int]core.Point)}, "", nil
	}
	
	// Check if this is a sequence diagram
	if diagram.Type == "sequence" {
		return r.renderSequenceWithPositions(diagram)
	}
	
	// Calculate node dimensions
	nodes := calculateNodeDimensions(diagram.Nodes)
	
	// Layout
	layoutNodes, err := r.layout.Layout(nodes, diagram.Connections)
	
	// After layout, adjust dimensions for editing node if needed
	if r.editingNodeID >= 0 {
		for i := range layoutNodes {
			if layoutNodes[i].ID == r.editingNodeID {
				// Split edit text into lines for multi-line support
				lines := strings.Split(r.editText, "\n")
				
				// Recalculate width - find the longest line
				maxWidth := 0
				for _, line := range lines {
					lineWidth := len([]rune(line))
					if lineWidth > maxWidth {
						maxWidth = lineWidth
					}
				}
				minWidth := maxWidth + 4  // text + padding
				if minWidth < 8 {
					minWidth = 8
				}
				layoutNodes[i].Width = minWidth
				
				// Recalculate height for multi-line text
				layoutNodes[i].Height = len(lines) + 2  // lines + borders
				
				break
			}
		}
	}
	if err != nil {
		return nil, "", fmt.Errorf("layout failed: %w", err)
	}
	
	// Route connections
	paths, err := r.router.RouteConnections(diagram.Connections, layoutNodes)
	if err != nil {
		return nil, "", fmt.Errorf("routing failed: %w", err)
	}
	
	// Calculate bounds
	bounds := calculateBounds(layoutNodes, paths)
	
	// Check if any nodes or connections have color or style hints
	hasColors := false
	for _, node := range layoutNodes {
		if node.Hints != nil {
			if _, hasColor := node.Hints["color"]; hasColor {
				hasColors = true
				break
			}
			if _, hasBold := node.Hints["bold"]; hasBold {
				hasColors = true  // We use the colored canvas for styles too
				break
			}
		}
	}
	if !hasColors {
		for _, conn := range diagram.Connections {
			if conn.Hints != nil {
				if _, hasColor := conn.Hints["color"]; hasColor {
					hasColors = true
					break
				}
				if _, hasBold := conn.Hints["bold"]; hasBold {
					hasColors = true  // We use the colored canvas for styles too
					break
				}
			}
		}
	}
	
	// Create appropriate canvas type
	var c canvas.Canvas
	var coloredCanvas *canvas.ColoredMatrixCanvas
	if hasColors {
		coloredCanvas = canvas.NewColoredMatrixCanvas(bounds.Width(), bounds.Height())
		c = coloredCanvas
	} else {
		c = canvas.NewMatrixCanvas(bounds.Width(), bounds.Height())
	}
	
	// Create offset canvas for negative coordinates
	offsetCanvas := newOffsetCanvas(c, bounds.Min)
	
	// Track node positions and connection paths (adjusted for canvas offset)
	positions := &NodePositions{
		Positions:       make(map[int]core.Point),
		ConnectionPaths: make(map[int]core.Path),
		Offset:          bounds.Min,
	}
	for _, node := range layoutNodes {
		// Store the canvas-relative position (after offset adjustment)
		positions.Positions[node.ID] = core.Point{
			X: node.X - bounds.Min.X,
			Y: node.Y - bounds.Min.Y,
		}
	}
	
	// Store connection paths (adjusted for offset)
	for i, path := range paths {
		adjustedPath := core.Path{
			Points: make([]core.Point, len(path.Points)),
		}
		for j, point := range path.Points {
			adjustedPath.Points[j] = core.Point{
				X: point.X - bounds.Min.X,
				Y: point.Y - bounds.Min.Y,
			}
		}
		positions.ConnectionPaths[i] = adjustedPath
	}
	
	// Render shadows first (so connections can overwrite them)
	for _, node := range layoutNodes {
		r.nodeRenderer.RenderShadowOnly(offsetCanvas, node)
	}
	
	// Render nodes
	for _, node := range layoutNodes {
		// Check if this node is being edited
		isEditing := node.ID == r.editingNodeID
		var editText string
		var cursorPos int
		if isEditing {
			editText = r.editText
			cursorPos = r.cursorPos
		}
		renderNodeWithEdit(offsetCanvas, node, r.nodeRenderer, isEditing, editText, cursorPos)
	}
	
	// Apply arrows to connections
	arrowConfig := connections.NewArrowConfig()
	connectionsWithArrows := connections.ApplyArrowConfig(diagram.Connections, paths, arrowConfig)
	
	// Render connections with hints
	for i, cwa := range connectionsWithArrows {
		hasArrow := cwa.ArrowType == connections.ArrowEnd || cwa.ArrowType == connections.ArrowBoth
		
		// Check if this connection has hints
		if i < len(diagram.Connections) && diagram.Connections[i].Hints != nil && len(diagram.Connections[i].Hints) > 0 {
			// Use RenderPathWithHints to apply visual hints
			r.pathRenderer.RenderPathWithHints(offsetCanvas, cwa.Path, hasArrow, diagram.Connections[i].Hints)
		} else {
			// Render normally
			r.pathRenderer.RenderPathWithOptions(offsetCanvas, cwa.Path, hasArrow, true)
		}
	}
	
	// Render labels
	for i, conn := range diagram.Connections {
		if conn.Label != "" && i < len(connectionsWithArrows) {
			r.labelRenderer.RenderLabel(offsetCanvas, connectionsWithArrows[i].Path, conn.Label, canvas.LabelMiddle)
		}
	}
	
	// Get the output string
	var output string
	if coloredCanvas != nil {
		// Use colored output if we have a colored canvas
		output = coloredCanvas.ColoredString()
	} else {
		// Regular output
		if mc, ok := c.(*canvas.MatrixCanvas); ok {
			output = mc.String()
		} else {
			output = c.String()
		}
	}
	
	return positions, output, nil
}

// renderSequenceWithPositions renders a sequence diagram and returns positions
func (r *RealRenderer) renderSequenceWithPositions(diagram *core.Diagram) (*NodePositions, string, error) {
	// Create sequence renderer
	seqRenderer := rendering.NewSequenceRenderer(r.capabilities)
	
	// Get bounds
	width, height := seqRenderer.GetBounds(diagram)
	if width <= 0 || height <= 0 {
		return nil, "", fmt.Errorf("invalid bounds: %dx%d", width, height)
	}
	
	// Check if we need color support
	needsColor := rendering.HasColorHints(diagram)
	
	// Create appropriate canvas
	var c canvas.Canvas
	var coloredCanvas *canvas.ColoredMatrixCanvas
	if needsColor && r.capabilities.SupportsColor {
		coloredCanvas = canvas.NewColoredMatrixCanvas(width, height)
		c = coloredCanvas
	} else {
		c = canvas.NewMatrixCanvas(width, height)
	}
	
	// Render the sequence diagram
	if err := seqRenderer.RenderToCanvas(diagram, c); err != nil {
		return nil, "", fmt.Errorf("failed to render sequence diagram: %w", err)
	}
	
	// Compute positions for jump labels
	seqLayout := layout.NewSequenceLayout()
	positionData := seqLayout.ComputePositions(diagram)
	
	// Collect positions for editor
	positions := &NodePositions{
		Positions:       make(map[int]core.Point),
		ConnectionPaths: make(map[int]core.Path),
	}
	
	// Add participant positions
	for nodeID, pos := range positionData.Participants {
		positions.Positions[nodeID] = core.Point{X: pos.X, Y: pos.Y}
	}
	
	// Add message paths
	for i, msg := range positionData.Messages {
		if i < len(diagram.Connections) {
			positions.ConnectionPaths[i] = core.Path{
				Points: []core.Point{
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
func calculateNodeDimensions(nodes []core.Node) []core.Node {
	result := make([]core.Node, len(nodes))
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

func calculateBounds(nodes []core.Node, paths map[int]core.Path) core.Bounds {
	if len(nodes) == 0 {
		return core.Bounds{Min: core.Point{X: 0, Y: 0}, Max: core.Point{X: 80, Y: 24}}
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
	return core.Bounds{
		Min: core.Point{X: minX - padding, Y: minY - padding},
		Max: core.Point{X: maxX + padding, Y: maxY + padding},
	}
}

// drawSimpleBox draws a basic box without any special styling
func drawSimpleBox(c canvas.Canvas, node core.Node, style canvas.NodeStyle) {
	// Top border
	c.Set(core.Point{X: node.X, Y: node.Y}, style.TopLeft)
	for x := node.X + 1; x < node.X + node.Width - 1; x++ {
		c.Set(core.Point{X: x, Y: node.Y}, style.Horizontal)
	}
	c.Set(core.Point{X: node.X + node.Width - 1, Y: node.Y}, style.TopRight)
	
	// Side borders
	for y := node.Y + 1; y < node.Y + node.Height - 1; y++ {
		c.Set(core.Point{X: node.X, Y: y}, style.Vertical)
		c.Set(core.Point{X: node.X + node.Width - 1, Y: y}, style.Vertical)
	}
	
	// Bottom border
	c.Set(core.Point{X: node.X, Y: node.Y + node.Height - 1}, style.BottomLeft)
	for x := node.X + 1; x < node.X + node.Width - 1; x++ {
		c.Set(core.Point{X: x, Y: node.Y + node.Height - 1}, style.Horizontal)
	}
	c.Set(core.Point{X: node.X + node.Width - 1, Y: node.Y + node.Height - 1}, style.BottomRight)
}

func renderNodeWithEdit(c canvas.Canvas, node core.Node, nodeRenderer *canvas.NodeRenderer, isEditing bool, editText string, cursorPos int) {
	// If not editing, use NodeRenderer to draw with styles
	if !isEditing {
		nodeRenderer.RenderNode(c, node)
		return
	}
	
	// When editing, draw a simple box without special styles
	// (to avoid visual noise during editing)
	style := canvas.NodeStyles["sharp"] // Use sharp style for editing
	drawSimpleBox(c, node, style)
	
	// Draw text with cursor if editing
	if isEditing {
		// Draw the edit text with cursor, handling multi-line
		// Split text by newlines
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
					c.Set(core.Point{X: x + i, Y: y}, ch)
				}
			}
		}
	} else {
		// Draw normal text
		for i, line := range node.Text {
			y := node.Y + 1 + i
			x := node.X + 2
			for j, ch := range line {
				if x+j < node.X+node.Width-2 {
					c.Set(core.Point{X: x + j, Y: y}, ch)
				}
			}
		}
	}
}

// offsetCanvas implementation (from renderer.go)
type offsetCanvas struct {
	canvas canvas.Canvas
	offset core.Point
}

func newOffsetCanvas(c canvas.Canvas, offset core.Point) *offsetCanvas {
	return &offsetCanvas{
		canvas: c,
		offset: offset,
	}
}

func (oc *offsetCanvas) Set(p core.Point, char rune) error {
	translated := core.Point{
		X: p.X - oc.offset.X,
		Y: p.Y - oc.offset.Y,
	}
	return oc.canvas.Set(translated, char)
}

// SetWithColor sets a character with color if the underlying canvas supports it
func (oc *offsetCanvas) SetWithColor(p core.Point, char rune, color string) error {
	translated := core.Point{
		X: p.X - oc.offset.X,
		Y: p.Y - oc.offset.Y,
	}
	// Try to set with color if the underlying canvas supports it
	if coloredCanvas, ok := oc.canvas.(*canvas.ColoredMatrixCanvas); ok {
		return coloredCanvas.SetWithColor(translated, char, color)
	}
	// Fall back to regular set
	return oc.canvas.Set(translated, char)
}

// SetWithColorAndStyle sets a character with color and style if the underlying canvas supports it
func (oc *offsetCanvas) SetWithColorAndStyle(p core.Point, char rune, color string, style string) error {
	translated := core.Point{
		X: p.X - oc.offset.X,
		Y: p.Y - oc.offset.Y,
	}
	// Try to set with color and style if the underlying canvas supports it
	if coloredCanvas, ok := oc.canvas.(*canvas.ColoredMatrixCanvas); ok {
		return coloredCanvas.SetWithColorAndStyle(translated, char, color, style)
	}
	// Fall back to regular set with color
	return oc.SetWithColor(p, char, color)
}

func (oc *offsetCanvas) Get(p core.Point) rune {
	translated := core.Point{
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
	if mc, ok := oc.canvas.(*canvas.MatrixCanvas); ok {
		return mc.Matrix()
	}
	if cc, ok := oc.canvas.(*canvas.ColoredMatrixCanvas); ok {
		return cc.Matrix()
	}
	return nil
}

func (oc *offsetCanvas) Offset() core.Point {
	return oc.offset
}
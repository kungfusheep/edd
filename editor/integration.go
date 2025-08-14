package editor

import (
	"edd/core"
	"edd/layout"
	"edd/pathfinding"
	"edd/connections"
	"edd/canvas"
	"fmt"
	"strings"
)

// RealRenderer wraps our actual modular renderer for the TUI
type RealRenderer struct {
	layout     core.LayoutEngine
	pathfinder core.PathFinder
	router     *connections.Router
	capabilities canvas.TerminalCapabilities
	pathRenderer *canvas.PathRenderer
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
		layout:        layoutEngine,
		pathfinder:    cachedPathfinder,
		router:        router,
		capabilities:  caps,
		pathRenderer:  canvas.NewPathRenderer(caps),
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
	positions, output, err := r.RenderWithPositions(diagram)
	_ = positions // Will be used by TUI for jump labels
	return output, err
}

// RenderWithPositions renders and returns node positions for jump labels
func (r *RealRenderer) RenderWithPositions(diagram *core.Diagram) (*NodePositions, string, error) {
	if diagram == nil || len(diagram.Nodes) == 0 {
		return &NodePositions{Positions: make(map[int]core.Point)}, "", nil
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
	
	// Create canvas
	c := canvas.NewMatrixCanvas(bounds.Width(), bounds.Height())
	
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
		renderNodeWithEdit(offsetCanvas, node, r.pathRenderer, isEditing, editText, cursorPos)
	}
	
	// Apply arrows to connections
	arrowConfig := connections.NewArrowConfig()
	connectionsWithArrows := connections.ApplyArrowConfig(diagram.Connections, paths, arrowConfig)
	
	// Render connections with hints
	for _, cwa := range connectionsWithArrows {
		hasArrow := cwa.ArrowType == connections.ArrowEnd || cwa.ArrowType == connections.ArrowBoth
		
		// TODO: Implement style and color rendering based on hints
		// For now, just render normally
		r.pathRenderer.RenderPathWithOptions(offsetCanvas, cwa.Path, hasArrow, true)
	}
	
	// Render labels
	for i, conn := range diagram.Connections {
		if conn.Label != "" && i < len(connectionsWithArrows) {
			r.labelRenderer.RenderLabel(offsetCanvas, connectionsWithArrows[i].Path, conn.Label, canvas.LabelMiddle)
		}
	}
	
	return positions, c.String(), nil
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

func renderNodeWithEdit(c canvas.Canvas, node core.Node, pathRenderer *canvas.PathRenderer, isEditing bool, editText string, cursorPos int) {
	// Draw box
	boxPath := core.Path{
		Points: []core.Point{
			{X: node.X, Y: node.Y},
			{X: node.X + node.Width - 1, Y: node.Y},
			{X: node.X + node.Width - 1, Y: node.Y + node.Height - 1},
			{X: node.X, Y: node.Y + node.Height - 1},
			{X: node.X, Y: node.Y},
		},
	}
	pathRenderer.RenderPath(c, boxPath, false)
	
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
	canvas *canvas.MatrixCanvas
	offset core.Point
}

func newOffsetCanvas(c *canvas.MatrixCanvas, offset core.Point) *offsetCanvas {
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
	return oc.canvas.Matrix()
}

func (oc *offsetCanvas) Offset() core.Point {
	return oc.offset
}
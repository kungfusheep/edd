package editor

import (
	"edd/core"
	"edd/layout"
	"edd/pathfinding"
	"edd/connections"
	"edd/rendering"
	"edd/canvas"
	"fmt"
)

// RealRenderer wraps our actual modular renderer for the TUI
type RealRenderer struct {
	layout     core.LayoutEngine
	pathfinder core.PathFinder
	router     *connections.Router
	capabilities rendering.TerminalCapabilities
	pathRenderer *rendering.PathRenderer
	labelRenderer *rendering.LabelRenderer
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
	caps := rendering.TerminalCapabilities{
		UnicodeLevel: rendering.UnicodeFull,
		SupportsColor: true,
	}
	
	return &RealRenderer{
		layout:        layoutEngine,
		pathfinder:    cachedPathfinder,
		router:        router,
		capabilities:  caps,
		pathRenderer:  rendering.NewPathRenderer(caps),
		labelRenderer: rendering.NewLabelRenderer(),
	}
}

// Render implements the core.Renderer interface for TUI
func (r *RealRenderer) Render(diagram *core.Diagram) (string, error) {
	if diagram == nil || len(diagram.Nodes) == 0 {
		return "", nil
	}
	
	// Calculate node dimensions
	nodes := calculateNodeDimensions(diagram.Nodes)
	
	// Layout
	layoutNodes, err := r.layout.Layout(nodes, diagram.Connections)
	if err != nil {
		return "", fmt.Errorf("layout failed: %w", err)
	}
	
	// Route connections
	paths, err := r.router.RouteConnections(diagram.Connections, layoutNodes)
	if err != nil {
		return "", fmt.Errorf("routing failed: %w", err)
	}
	
	// Calculate bounds
	bounds := calculateBounds(layoutNodes, paths)
	
	// Create canvas
	c := canvas.NewMatrixCanvas(bounds.Width(), bounds.Height())
	
	// Create offset canvas for negative coordinates
	offsetCanvas := newOffsetCanvas(c, bounds.Min)
	
	// Render nodes
	for _, node := range layoutNodes {
		renderNode(offsetCanvas, node, r.pathRenderer)
	}
	
	// Apply arrows to connections
	arrowConfig := connections.NewArrowConfig()
	connectionsWithArrows := connections.ApplyArrowConfig(diagram.Connections, paths, arrowConfig)
	
	// Render connections
	for _, cwa := range connectionsWithArrows {
		hasArrow := cwa.ArrowType == connections.ArrowEnd || cwa.ArrowType == connections.ArrowBoth
		r.pathRenderer.RenderPathWithOptions(offsetCanvas, cwa.Path, hasArrow, true)
	}
	
	// Render labels
	for i, conn := range diagram.Connections {
		if conn.Label != "" && i < len(connectionsWithArrows) {
			r.labelRenderer.RenderLabel(offsetCanvas, connectionsWithArrows[i].Path, conn.Label, rendering.LabelMiddle)
		}
	}
	
	return c.String(), nil
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

func renderNode(c canvas.Canvas, node core.Node, pathRenderer *rendering.PathRenderer) {
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
	
	// Draw text
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
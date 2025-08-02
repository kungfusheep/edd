package main

import (
	"edd/canvas"
	"edd/connections"
	"edd/core"
	"edd/layout"
	"edd/pathfinding"
	"edd/rendering"
	"fmt"
)

// Renderer orchestrates the diagram rendering pipeline.
// It coordinates layout, canvas creation, node rendering, and connection routing.
type Renderer struct {
	layout       core.LayoutEngine
	pathfinder   core.PathFinder
	router       *connections.Router
	capabilities rendering.TerminalCapabilities // Cached to avoid repeated detection
	pathRenderer *rendering.PathRenderer         // Reused across renders
}

// NewRenderer creates a new renderer with sensible defaults.
func NewRenderer() *Renderer {
	// Use simple layout by default
	layoutEngine := layout.NewSimpleLayout()
	
	// Use smart pathfinder with good defaults
	pathfinder := pathfinding.NewSmartPathFinder(pathfinding.PathCost{
		StraightCost:  10,
		TurnCost:      20,
		ProximityCost: -5, // Prefer paths that hug obstacles
	})
	
	// Add caching for performance. Cache size of 100 handles most diagrams
	// efficiently without excessive memory usage (100 * ~1KB per path = ~100KB)
	cachedPathfinder := pathfinding.NewCachedPathFinder(pathfinder, 100)
	
	// Create router with pathfinder
	router := connections.NewRouter(cachedPathfinder)
	
	// Detect terminal capabilities once
	caps := detectTerminalCapabilities()
	
	return &Renderer{
		layout:       layoutEngine,
		pathfinder:   cachedPathfinder,
		router:       router,
		capabilities: caps,
		pathRenderer: rendering.NewPathRenderer(caps),
	}
}

// calculateNodeDimensions determines the width and height of nodes based on their text content.
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
		
		// Add padding: 2 chars for borders + 2 chars for internal padding
		result[i].Width = maxWidth + 4
		// Height: number of lines + 2 for borders
		result[i].Height = len(result[i].Text) + 2
	}
	
	return result
}

// calculateBounds determines the canvas size needed to fit all nodes.
func calculateBounds(nodes []core.Node) core.Bounds {
	if len(nodes) == 0 {
		return core.Bounds{Min: core.Point{X: 0, Y: 0}, Max: core.Point{X: 10, Y: 10}}
	}
	
	minX, minY := nodes[0].X, nodes[0].Y
	maxX, maxY := nodes[0].X+nodes[0].Width, nodes[0].Y+nodes[0].Height
	
	for _, node := range nodes[1:] {
		if node.X < minX {
			minX = node.X
		}
		if node.Y < minY {
			minY = node.Y
		}
		if node.X+node.Width > maxX {
			maxX = node.X + node.Width
		}
		if node.Y+node.Height > maxY {
			maxY = node.Y + node.Height
		}
	}
	
	// Add some padding around the diagram
	padding := 2
	return core.Bounds{
		Min: core.Point{X: minX - padding, Y: minY - padding},
		Max: core.Point{X: maxX + padding, Y: maxY + padding},
	}
}

// renderNode draws a single node on the canvas.
func (r *Renderer) renderNode(c *canvas.MatrixCanvas, node core.Node) error {
	// Draw the box
	boxPath := core.Path{
		Points: []core.Point{
			{X: node.X, Y: node.Y},
			{X: node.X + node.Width - 1, Y: node.Y},
			{X: node.X + node.Width - 1, Y: node.Y + node.Height - 1},
			{X: node.X, Y: node.Y + node.Height - 1},
			{X: node.X, Y: node.Y},
		},
	}
	
	// Use the cached path renderer to draw the box
	r.pathRenderer.RenderPath(c, boxPath, false)
	
	// Draw the text inside the box
	for i, line := range node.Text {
		y := node.Y + 1 + i
		x := node.X + 2 // 2 chars padding from left border
		
		for j, ch := range line {
			if x+j < node.X+node.Width-2 { // Keep text within borders
				c.Set(core.Point{X: x + j, Y: y}, ch)
			}
		}
	}
	
	return nil
}

// detectTerminalCapabilities returns the current terminal's capabilities.
func detectTerminalCapabilities() rendering.TerminalCapabilities {
	// For now, return a simple default. In the future, this could
	// actually detect the terminal type and capabilities.
	return rendering.TerminalCapabilities{
		UnicodeLevel: rendering.UnicodeFull,
		SupportsColor: true,
	}
}

// adjustPathEndpoints adjusts the start and end points of a path to avoid
// junction conflicts with node borders. It moves the endpoints one character
// outside the node boxes.
func adjustPathEndpoints(path core.Path, fromNode, toNode core.Node) core.Path {
	if len(path.Points) < 2 {
		return path
	}
	
	// Copy the path
	adjusted := core.Path{
		Points: make([]core.Point, len(path.Points)),
		Cost:   path.Cost,
	}
	copy(adjusted.Points, path.Points)
	
	// Adjust start point
	start := &adjusted.Points[0]
	if len(path.Points) > 1 {
		next := path.Points[1]
		if next.X > start.X {
			// Moving right - start one char to the right
			start.X++
		} else if next.X < start.X {
			// Moving left - start one char to the left  
			start.X--
		} else if next.Y > start.Y {
			// Moving down - start one char down
			start.Y++
		} else if next.Y < start.Y {
			// Moving up - start one char up
			start.Y--
		}
	}
	
	// Adjust end point
	end := &adjusted.Points[len(adjusted.Points)-1]
	if len(path.Points) > 1 {
		prev := path.Points[len(path.Points)-2]
		if prev.X > end.X {
			// Coming from right - end one char to the right
			end.X++
		} else if prev.X < end.X {
			// Coming from left - end one char to the left
			end.X--
		} else if prev.Y > end.Y {
			// Coming from below - end one char down  
			end.Y++
		} else if prev.Y < end.Y {
			// Coming from above - end one char up
			end.Y--
		}
	}
	
	return adjusted
}

// Render orchestrates the complete rendering pipeline for a diagram.
// It performs layout, creates a canvas, renders nodes, routes connections,
// and returns the final ASCII/Unicode output.
func (r *Renderer) Render(diagram *core.Diagram) (string, error) {
	// Step 1: Calculate node dimensions from their text content
	nodes := calculateNodeDimensions(diagram.Nodes)
	
	// Step 2: Run layout algorithm to position nodes
	layoutNodes, err := r.layout.Layout(nodes, diagram.Connections)
	if err != nil {
		return "", fmt.Errorf("layout failed: %w", err)
	}
	
	// Step 3: Calculate bounds and create canvas
	bounds := calculateBounds(layoutNodes)
	c := canvas.NewMatrixCanvas(bounds.Width(), bounds.Height())
	
	// Step 4: Render all nodes
	for _, node := range layoutNodes {
		if err := r.renderNode(c, node); err != nil {
			return "", fmt.Errorf("failed to render node %d: %w", node.ID, err)
		}
	}
	
	// Step 5: Route connections between nodes
	paths, err := r.router.RouteConnections(diagram.Connections, layoutNodes)
	if err != nil {
		return "", fmt.Errorf("connection routing failed: %w", err)
	}
	
	// Step 5.5: Handle self-loops specially
	for i, conn := range diagram.Connections {
		if conn.From == conn.To {
			// Handle self-loops
			for _, node := range layoutNodes {
				if node.ID == conn.From {
					paths[i] = connections.HandleSelfLoops(conn, &node)
					break
				}
			}
		}
	}
	
	// Step 6: Create arrow configuration
	arrowConfig := connections.NewArrowConfig()
	// For now, use default arrow configuration
	// Future: Could be customized based on diagram metadata
	
	// Step 7: Apply arrow configuration to connections
	connectionsWithArrows := connections.ApplyArrowConfig(diagram.Connections, paths, arrowConfig)
	
	// Step 8: Render all connections (using cached pathRenderer)
	for _, cwa := range connectionsWithArrows {
		hasArrow := cwa.ArrowType == connections.ArrowEnd || cwa.ArrowType == connections.ArrowBoth
		r.pathRenderer.RenderPath(c, cwa.Path, hasArrow)
	}
	
	// Step 9: Convert canvas to string output
	return c.String(), nil
}
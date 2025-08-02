package main

import (
	"edd/canvas"
	"edd/connections"
	"edd/core"
	"edd/layout"
	"edd/pathfinding"
	"edd/rendering"
	"fmt"
	"os"
)

// Renderer orchestrates the diagram rendering pipeline.
// It coordinates layout, canvas creation, node rendering, and connection routing.
type Renderer struct {
	layout       core.LayoutEngine
	pathfinder   core.PathFinder
	router       *connections.Router
	capabilities rendering.TerminalCapabilities // Cached to avoid repeated detection
	pathRenderer *rendering.PathRenderer         // Reused across renders
	validator    *LineValidator                  // Optional output validator
	debugMode    bool                            // Enable debug obstacle visualization
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
		validator:    nil, // Validator is optional, enabled via SetValidator
	}
}

// SetValidator enables output validation with the given validator.
func (r *Renderer) SetValidator(v *LineValidator) {
	r.validator = v
}

// EnableValidation enables output validation with default settings.
func (r *Renderer) EnableValidation() {
	r.validator = NewLineValidator()
}

// EnableDebug enables debug mode to show obstacle visualization.
func (r *Renderer) EnableDebug() {
	r.debugMode = true
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

// createBoxEdgeJunction creates a T-junction at the box edge where a connection meets it.
func (r *Renderer) createBoxEdgeJunction(c *canvas.MatrixCanvas, connectionPoint core.Point, node core.Node, isStart bool) {
	// Determine which edge of the box the connection point is near
	cx, cy := connectionPoint.X, connectionPoint.Y
	
	// Check each edge
	if cx == node.X-1 && cy >= node.Y && cy < node.Y+node.Height {
		// Connection is to the left of the box
		edgePoint := core.Point{X: node.X, Y: cy}
		existing := c.Get(edgePoint)
		if existing == '│' {
			c.Set(edgePoint, '├') // T-junction pointing right
		}
	} else if cx == node.X+node.Width && cy >= node.Y && cy < node.Y+node.Height {
		// Connection is to the right of the box
		edgePoint := core.Point{X: node.X+node.Width-1, Y: cy}
		existing := c.Get(edgePoint)
		if existing == '│' {
			c.Set(edgePoint, '┤') // T-junction pointing left
		}
	} else if cy == node.Y-1 && cx >= node.X && cx < node.X+node.Width {
		// Connection is above the box
		edgePoint := core.Point{X: cx, Y: node.Y}
		existing := c.Get(edgePoint)
		if existing == '─' {
			c.Set(edgePoint, '┬') // T-junction pointing down
		}
	} else if cy == node.Y+node.Height && cx >= node.X && cx < node.X+node.Width {
		// Connection is below the box
		edgePoint := core.Point{X: cx, Y: node.Y+node.Height-1}
		existing := c.Get(edgePoint)
		if existing == '─' {
			c.Set(edgePoint, '┴') // T-junction pointing up
		}
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
	
	// Step 5.1: Debug mode - visualize obstacles if enabled
	if r.debugMode {
		return r.renderDebugObstacles(layoutNodes, diagram.Connections, paths, bounds), nil
	}
	
	// Step 5.5: Self-loops are already handled by the router, no need to override
	
	// Step 6: Create arrow configuration
	arrowConfig := connections.NewArrowConfig()
	// For now, use default arrow configuration
	// Future: Could be customized based on diagram metadata
	
	// Step 7: Apply arrow configuration to connections
	connectionsWithArrows := connections.ApplyArrowConfig(diagram.Connections, paths, arrowConfig)
	
	// Step 7.5: Create a map of node IDs to nodes for quick lookup
	nodeMap := make(map[int]core.Node)
	for _, node := range layoutNodes {
		nodeMap[node.ID] = node
	}
	
	// Step 8: Render all connections (no endpoint adjustment needed)
	for i, cwa := range connectionsWithArrows {
		hasArrow := cwa.ArrowType == connections.ArrowEnd || cwa.ArrowType == connections.ArrowBoth
		
		// Use RenderPathWithOptions to enable connection endpoint handling
		r.pathRenderer.RenderPathWithOptions(c, cwa.Path, hasArrow, true)
		
		// Create T-junctions at box edges where connections start/end
		if len(cwa.Path.Points) >= 2 {
			// Handle start point junction
			conn := diagram.Connections[i]
			if fromNode, ok := nodeMap[conn.From]; ok {
				r.createBoxEdgeJunction(c, cwa.Path.Points[0], fromNode, true)
			}
			
			// Handle end point junction (but not if there's an arrow there)
			if toNode, ok := nodeMap[conn.To]; ok && !hasArrow {
				endPoint := cwa.Path.Points[len(cwa.Path.Points)-1]
				r.createBoxEdgeJunction(c, endPoint, toNode, false)
			}
		}
	}
	
	// Step 9: Convert canvas to string output
	output := c.String()
	
	// Step 10: Validate output if validator is enabled
	if r.validator != nil {
		errors := r.validator.Validate(output)
		if len(errors) > 0 {
			// Log validation errors but don't fail the render
			// In production, you might want to return these as warnings
			fmt.Fprintf(os.Stderr, "Warning: Output validation found %d issues:\n", len(errors))
			for i, err := range errors {
				if i >= 5 {
					fmt.Fprintf(os.Stderr, "  ... and %d more\n", len(errors)-5)
					break
				}
				fmt.Fprintf(os.Stderr, "  %s\n", err)
			}
		}
	}
	
	return output, nil
}

// renderDebugObstacles creates a debug visualization showing obstacles and paths.
func (r *Renderer) renderDebugObstacles(layoutNodes []core.Node, connections []core.Connection, paths map[int]core.Path, bounds core.Bounds) string {
	// Import the debug visualizer
	debugViz := pathfinding.NewDebugVisualizer(bounds.Width(), bounds.Height())
	
	// Add all nodes to the visualization
	for i, node := range layoutNodes {
		label := rune('A' + i)
		if i >= 26 {
			label = rune('0' + (i - 26))
		}
		debugViz.AddNode(node, label)
	}
	
	// Create the obstacle function used by the router to show exactly what it sees
	for i, conn := range connections {
		if path, exists := paths[i]; exists && len(path.Points) >= 2 {
			// Get the obstacle function that was used for this connection
			obstacles := r.createObstaclesForConnection(layoutNodes, conn.From, conn.To)
			
			// Add obstacles to visualization (but only for first connection to avoid clutter)
			if i == 0 {
				debugViz.AddObstacles(obstacles, 'o')
			}
			
			// Add the path
			pathMarker := '*'
			if i > 0 {
				pathMarker = rune('1' + (i % 9)) // Use numbers for multiple paths
			}
			debugViz.AddPath(path, rune(pathMarker))
		}
	}
	
	result := fmt.Sprintf("DEBUG OBSTACLE VISUALIZATION\n")
	result += fmt.Sprintf("===========================\n\n")
	result += debugViz.String()
	
	// Add analysis for each connection
	for i, conn := range connections {
		if path, exists := paths[i]; exists {
			obstacles := r.createObstaclesForConnection(layoutNodes, conn.From, conn.To)
			analysis := debugViz.AnalyzePath(path, layoutNodes, obstacles)
			result += fmt.Sprintf("\nConnection %d (%d -> %d):\n", i, conn.From, conn.To)
			result += analysis
		}
	}
	
	return result
}

// createObstaclesForConnection creates the same obstacle function used by the router.
func (r *Renderer) createObstaclesForConnection(nodes []core.Node, sourceID, targetID int) func(core.Point) bool {
	// This should match the logic in connections.createObstaclesFunction
	return func(p core.Point) bool {
		for _, node := range nodes {
			// For source and target nodes, only block the interior (not edges)
			if node.ID == sourceID || node.ID == targetID {
				// Allow points on the edge but not inside
				if p.X > node.X && p.X < node.X+node.Width-1 &&
				   p.Y > node.Y && p.Y < node.Y+node.Height-1 {
					return true
				}
				// Add virtual obstacles around source/target for aesthetic control
				if r.isInVirtualObstacleZone(p, node, sourceID, targetID) {
					return true
				}
				continue
			}
			
			// For other nodes, block with padding using proper boundaries
			padding := 2
			if p.X >= node.X-padding && p.X < node.X+node.Width+padding &&
			   p.Y >= node.Y-padding && p.Y < node.Y+node.Height+padding {
				return true
			}
			
			// Add virtual obstacles around other nodes for aesthetic control
			if r.isInVirtualObstacleZone(p, node, sourceID, targetID) {
				return true
			}
		}
		return false
	}
}

// isInVirtualObstacleZone mirrors the logic from connections/router.go
func (r *Renderer) isInVirtualObstacleZone(p core.Point, node core.Node, sourceID, targetID int) bool {
	// Define approach zone parameters
	const (
		approachZoneSize = 1 // Size of the approach zone around each box
	)
	
	// For the current connection's source/target nodes, allow more flexible access
	if node.ID == sourceID || node.ID == targetID {
		// Allow the connection to work but still prevent extremely close diagonal approaches
		
		// Allow points that are the exact connection points (1 unit away from box edges)
		connectionPoints := []core.Point{
			{X: node.X - 1, Y: node.Y + node.Height/2},     // Left connection point
			{X: node.X + node.Width, Y: node.Y + node.Height/2}, // Right connection point
			{X: node.X + node.Width/2, Y: node.Y - 1},      // Top connection point
			{X: node.X + node.Width/2, Y: node.Y + node.Height}, // Bottom connection point
		}
		
		for _, cp := range connectionPoints {
			if p.X == cp.X && p.Y == cp.Y {
				return false // Allow exact connection points
			}
		}
		
		// For source/target nodes, only block very close diagonal approaches
		if abs(p.X - (node.X + node.Width/2)) <= 1 && abs(p.Y - (node.Y + node.Height/2)) <= 1 {
			// Very close to node center - check if it's a diagonal approach that we should block
			if p.X != node.X + node.Width/2 && p.Y != node.Y + node.Height/2 {
				return true // Block diagonal approaches very close to the node
			}
		}
		
		return false // Otherwise allow for source/target nodes
	}
	
	// For OTHER nodes (not involved in this connection), create approach corridors
	// These virtual obstacles force orthogonal approaches while preserving access corridors
	
	// Calculate approach zone boundaries  
	leftBoundary := node.X - approachZoneSize
	rightBoundary := node.X + node.Width + approachZoneSize - 1
	topBoundary := node.Y - approachZoneSize  
	bottomBoundary := node.Y + node.Height + approachZoneSize - 1
	
	// Check if point is within the approach zone
	if p.X < leftBoundary || p.X > rightBoundary || 
	   p.Y < topBoundary || p.Y > bottomBoundary {
		return false // Outside approach zone - no obstacles
	}
	
	// Create orthogonal approach corridors to each side of the box
	// Allow corridors that lead directly to connection points
	
	// Horizontal corridors (left and right approaches)
	midY := node.Y + node.Height/2
	if p.Y == midY {
		// Allow direct horizontal approach corridors at box center height
		if p.X == node.X-1 || p.X == node.X+node.Width {
			return false // Allow connection points
		}
		if (p.X < node.X-1 && p.X >= leftBoundary) || 
		   (p.X > node.X+node.Width && p.X <= rightBoundary) {
			return false // Allow horizontal approach corridors
		}
	}
	
	// Vertical corridors (top and bottom approaches)  
	midX := node.X + node.Width/2
	if p.X == midX {
		// Allow direct vertical approach corridors at box center width
		if p.Y == node.Y-1 || p.Y == node.Y+node.Height {
			return false // Allow connection points
		}
		if (p.Y < node.Y-1 && p.Y >= topBoundary) ||
		   (p.Y > node.Y+node.Height && p.Y <= bottomBoundary) {
			return false // Allow vertical approach corridors
		}
	}
	
	// If we reach here, the point is in the approach zone but not in any allowed corridor
	// Block it to force use of the orthogonal approach corridors
	return true
}

// abs returns the absolute value of an integer (helper function).
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
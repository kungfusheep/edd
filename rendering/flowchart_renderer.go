package rendering

import (
	"edd/canvas"
	"edd/connections"
	"edd/core"
	"edd/layout"
	"edd/obstacles"
	"edd/pathfinding"
	"fmt"
)

// FlowchartRenderer handles rendering of flowchart diagrams
type FlowchartRenderer struct {
	layout        core.LayoutEngine
	pathfinder    core.PathFinder
	router        *connections.Router
	capabilities  canvas.TerminalCapabilities
	pathRenderer  *canvas.PathRenderer
	nodeRenderer  *canvas.NodeRenderer
	labelRenderer *canvas.LabelRenderer
	debugMode     bool
	showObstacles bool
}

// NewFlowchartRenderer creates a new flowchart diagram renderer
func NewFlowchartRenderer(caps canvas.TerminalCapabilities) *FlowchartRenderer {
	// Use simple layout by default
	layoutEngine := layout.NewSimpleLayout()
	
	// Use smart pathfinder with good defaults
	pathfinder := pathfinding.NewSmartPathFinder(pathfinding.PathCost{
		StraightCost:  10,
		TurnCost:      100,  // Very high cost to strongly discourage turns
		ProximityCost: 0,    // Neutral - don't hug or avoid walls
		DirectionBias: 0,    // No bias - treat horizontal and vertical equally for symmetry
	})
	
	// Add caching for performance. Cache size of 100 handles most diagrams
	// efficiently without excessive memory usage (100 * ~1KB per path = ~100KB)
	cachedPathfinder := pathfinding.NewCachedPathFinder(pathfinder, 100)
	
	// Create router with pathfinder
	router := connections.NewRouter(cachedPathfinder)
	
	return &FlowchartRenderer{
		layout:        layoutEngine,
		pathfinder:    cachedPathfinder,
		router:        router,
		capabilities:  caps,
		pathRenderer:  canvas.NewPathRenderer(caps),
		nodeRenderer:  canvas.NewNodeRenderer(caps),
		labelRenderer: canvas.NewLabelRenderer(),
		debugMode:     false,
		showObstacles: false,
	}
}

// CanRender returns true if this renderer can handle the given diagram type.
func (r *FlowchartRenderer) CanRender(diagramType core.DiagramType) bool {
	// Flowchart is the default type (empty string or "flowchart")
	return diagramType == core.DiagramTypeFlowchart || diagramType == ""
}

// Render renders the flowchart diagram and returns the string output.
func (r *FlowchartRenderer) Render(diagram *core.Diagram) (string, error) {
	if diagram == nil {
		return "", fmt.Errorf("diagram is nil")
	}
	
	// Step 1: Calculate node dimensions from their text content
	nodes := CalculateNodeDimensions(diagram.Nodes)
	
	// Step 2: Run layout algorithm to position nodes
	layoutNodes, err := r.layout.Layout(nodes, diagram.Connections)
	if err != nil {
		return "", fmt.Errorf("layout failed: %w", err)
	}
	
	// Step 3: Set up port manager (after layout is complete)
	// Always use port manager for better connection routing
	portWidth := 1 // Default port width
	portManager := obstacles.NewPortManager(layoutNodes, portWidth)
	r.router.SetPortManager(portManager)
	
	// Step 4: Route connections between nodes
	paths, err := r.router.RouteConnections(diagram.Connections, layoutNodes)
	if err != nil {
		return "", fmt.Errorf("connection routing failed: %w", err)
	}
	
	// Step 5: Calculate bounds and create canvas
	bounds := CalculateBounds(layoutNodes, paths)
	
	// Check if we need colors
	needsColor := HasColorHints(diagram)
	
	// Create appropriate canvas type
	c := CreateCanvas(bounds.Width(), bounds.Height(), needsColor)
	
	// Create offset canvas that handles coordinate translation
	offsetCanvas := NewOffsetCanvas(c, bounds.Min)
	
	// Step 5.1: Debug mode - visualize obstacles if enabled
	if r.debugMode {
		return r.renderDebugObstacles(layoutNodes, diagram.Connections, paths, bounds), nil
	}
	
	// Step 6: Render the diagram components
	if err := r.renderToCanvas(diagram, layoutNodes, paths, offsetCanvas); err != nil {
		return "", fmt.Errorf("failed to render to canvas: %w", err)
	}
	
	// Step 7: Convert canvas to string output
	var output string
	if coloredCanvas, ok := c.(*canvas.ColoredMatrixCanvas); ok {
		// Use colored output if we have a colored canvas
		output = coloredCanvas.ColoredString()
	} else {
		// Regular output
		output = c.String()
	}
	
	return output, nil
}

// GetBounds returns the required canvas size for the diagram
func (r *FlowchartRenderer) GetBounds(diagram *core.Diagram) (width, height int) {
	// Calculate node dimensions
	nodes := CalculateNodeDimensions(diagram.Nodes)
	
	// Run layout to get positions
	layoutNodes, err := r.layout.Layout(nodes, diagram.Connections)
	if err != nil {
		// Return a default size on error
		return 80, 24
	}
	
	// Route connections to get paths
	paths, err := r.router.RouteConnections(diagram.Connections, layoutNodes)
	if err != nil {
		// Just use node bounds if routing fails
		bounds := CalculateBounds(layoutNodes, nil)
		return bounds.Width(), bounds.Height()
	}
	
	// Calculate full bounds including paths
	bounds := CalculateBounds(layoutNodes, paths)
	return bounds.Width(), bounds.Height()
}

// renderToCanvas performs the actual rendering to the canvas
func (r *FlowchartRenderer) renderToCanvas(diagram *core.Diagram, layoutNodes []core.Node, paths map[int]core.Path, offsetCanvas canvas.Canvas) error {
	// Step 1: Render shadows first (so connections can overwrite them)
	for _, node := range layoutNodes {
		r.nodeRenderer.RenderShadowOnly(offsetCanvas, node)
	}
	
	// Step 2: Render nodes (boxes and text) before connections
	// This allows connections to properly connect to node edges
	for _, node := range layoutNodes {
		if err := r.nodeRenderer.RenderNode(offsetCanvas, node); err != nil {
			return fmt.Errorf("failed to render node %d: %w", node.ID, err)
		}
	}
	
	// Step 3: Create arrow configuration
	arrowConfig := connections.NewArrowConfig()
	// For now, use default arrow configuration
	// Future: Could be customized based on diagram metadata
	
	// Step 4: Apply arrow configuration to connections
	connectionsWithArrows := connections.ApplyArrowConfig(diagram.Connections, paths, arrowConfig)
	
	// Step 5: Render all connections
	// Note: connectionsWithArrows maintains the same order as diagram.Connections
	for i, cwa := range connectionsWithArrows {
		hasArrow := cwa.ArrowType == connections.ArrowEnd || cwa.ArrowType == connections.ArrowBoth
		
		// Check if this connection has hints
		if i < len(diagram.Connections) && diagram.Connections[i].Hints != nil && len(diagram.Connections[i].Hints) > 0 {
			// Use RenderPathWithHints to apply visual hints
			r.pathRenderer.RenderPathWithHints(offsetCanvas, cwa.Path, hasArrow, diagram.Connections[i].Hints)
		} else {
			// Use RenderPathWithOptions to enable connection endpoint handling
			r.pathRenderer.RenderPathWithOptions(offsetCanvas, cwa.Path, hasArrow, true)
		}
	}
	
	// Step 6: Render connection labels after all paths are drawn
	// This ensures labels are placed on top of the lines
	for i, conn := range diagram.Connections {
		if conn.Label != "" && i < len(connectionsWithArrows) {
			r.labelRenderer.RenderLabel(offsetCanvas, connectionsWithArrows[i].Path, conn.Label, canvas.LabelMiddle)
		}
	}
	
	// Step 7: Show virtual obstacles if enabled
	if r.showObstacles {
		r.renderObstacleDots(offsetCanvas, layoutNodes, diagram.Connections, paths)
	}
	
	return nil
}

// EnableDebug enables debug mode to show obstacle visualization.
func (r *FlowchartRenderer) EnableDebug() {
	r.debugMode = true
}

// EnableObstacleVisualization enables showing virtual obstacles as dots in standard rendering
func (r *FlowchartRenderer) EnableObstacleVisualization() {
	r.showObstacles = true
}

// GetRouter returns the router instance for external configuration
func (r *FlowchartRenderer) GetRouter() *connections.Router {
	return r.router
}

// SetRouterType sets the type of router to use
func (r *FlowchartRenderer) SetRouterType(routerType connections.RouterType) {
	r.router.SetRouterType(routerType)
}

// renderDebugObstacles creates a debug visualization showing obstacles and paths.
func (r *FlowchartRenderer) renderDebugObstacles(layoutNodes []core.Node, connections []core.Connection, paths map[int]core.Path, bounds core.Bounds) string {
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
	
	// Show port information (always available now)
	var portInfo string
	if r.router.GetObstacleManager() != nil {
		portInfo = r.renderPortDebugInfo(layoutNodes, paths)
		
		// Add port corridors to visualization
		for _, path := range paths {
			if path.Metadata != nil {
				// Show source port
				if sourcePort, ok := path.Metadata["sourcePort"].(obstacles.Port); ok {
					debugViz.AddPoint(sourcePort.Point, 'P')
				}
				// Show target port
				if targetPort, ok := path.Metadata["targetPort"].(obstacles.Port); ok {
					debugViz.AddPoint(targetPort.Point, 'P')
				}
			}
		}
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
	
	// Add port information if available
	if portInfo != "" {
		result += fmt.Sprintf("\n\nPORT INFORMATION\n")
		result += fmt.Sprintf("================\n")
		result += portInfo
	}
	
	// Add analysis for each connection
	for i, conn := range connections {
		if path, exists := paths[i]; exists {
			// Use the same obstacle function that was used for routing this specific connection
			var obstacles func(core.Point) bool
			if r.router != nil && r.router.GetObstacleManager() != nil {
				obstacles = r.router.GetObstacleManager().GetObstacleFuncForConnection(layoutNodes, conn)
			} else {
				obstacles = r.createObstaclesForConnection(layoutNodes, conn.From, conn.To)
			}
			analysis := debugViz.AnalyzePath(path, layoutNodes, obstacles)
			result += fmt.Sprintf("\nConnection %d (%d -> %d):\n", i, conn.From, conn.To)
			result += analysis
		}
	}
	
	return result
}

// createObstaclesForConnection creates the same obstacle function used by the router.
func (r *FlowchartRenderer) createObstaclesForConnection(nodes []core.Node, sourceID, targetID int) func(core.Point) bool {
	// Use the same obstacle checker as the router for consistency
	if r.router != nil && r.router.GetObstacleManager() != nil {
		// Create a dummy connection to get the proper obstacle function
		dummyConn := core.Connection{
			ID:   -1,
			From: sourceID,
			To:   targetID,
		}
		return r.router.GetObstacleManager().GetObstacleFuncForConnection(nodes, dummyConn)
	}
	
	// Fallback implementation
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
func (r *FlowchartRenderer) isInVirtualObstacleZone(p core.Point, node core.Node, sourceID, targetID int) bool {
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
		dx := p.X - (node.X + node.Width/2)
		dy := p.Y - (node.Y + node.Height/2)
		if dx < 0 { dx = -dx }
		if dy < 0 { dy = -dy }
		if dx <= 1 && dy <= 1 {
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

// renderPortDebugInfo generates debug information about port usage
func (r *FlowchartRenderer) renderPortDebugInfo(nodes []core.Node, paths map[int]core.Path) string {
	var result string
	
	// Get the port manager through the obstacle manager
	obstacleManager := r.router.GetObstacleManager()
	if obstacleManager == nil {
		return "No obstacle manager available\n"
	}
	
	// For each node, show port information
	for _, node := range nodes {
		result += fmt.Sprintf("\nNode %d (%dx%d at %d,%d):\n", 
			node.ID, node.Width, node.Height, node.X, node.Y)
		
		// Show ports on each edge
		edges := []obstacles.EdgeSide{
			obstacles.North,
			obstacles.East,
			obstacles.South,
			obstacles.West,
		}
		
		for _, edge := range edges {
			edgeName := getEdgeName(edge)
			result += fmt.Sprintf("  %s edge: ", edgeName)
			
			// Get available and occupied ports info
			// Since we can't directly access the port manager, we'll extract from paths
			portsOnEdge := extractPortsFromPaths(node.ID, edge, paths)
			
			if len(portsOnEdge) > 0 {
				for i, portInfo := range portsOnEdge {
					if i > 0 {
						result += ", "
					}
					result += fmt.Sprintf("Port at (%d,%d) used by conn %d", 
						portInfo.point.X, portInfo.point.Y, portInfo.connID)
				}
			} else {
				result += "No ports in use"
			}
			result += "\n"
		}
	}
	
	// Show port usage for each connection
	result += "\nConnection Port Usage:\n"
	for connID, path := range paths {
		if path.Metadata != nil {
			if sourcePort, ok := path.Metadata["sourcePort"].(obstacles.Port); ok {
				if targetPort, ok2 := path.Metadata["targetPort"].(obstacles.Port); ok2 {
					result += fmt.Sprintf("  Connection %d: ", connID)
					result += fmt.Sprintf("Source node %d %s edge (%d,%d) -> ", 
						sourcePort.NodeID, getEdgeName(sourcePort.Edge), 
						sourcePort.Point.X, sourcePort.Point.Y)
					result += fmt.Sprintf("Target node %d %s edge (%d,%d)\n", 
						targetPort.NodeID, getEdgeName(targetPort.Edge),
						targetPort.Point.X, targetPort.Point.Y)
				}
			}
		}
	}
	
	return result
}

// renderObstacleDots adds dots to show virtual obstacles on the canvas
func (r *FlowchartRenderer) renderObstacleDots(c canvas.Canvas, nodes []core.Node, connections []core.Connection, paths map[int]core.Path) {
	// Get the obstacle manager to access virtual obstacles
	obstacleManager := r.router.GetObstacleManager()
	if obstacleManager == nil {
		return
	}
	
	// Create a test obstacle function to probe all points
	// We'll use a dummy connection to get the general obstacle map
	obstacleFunc := obstacleManager.GetObstacleFunc(nodes, -1)
	
	// Get canvas bounds
	width, height := c.Size()
	
	// Check each point on the canvas
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			p := core.Point{X: x, Y: y}
			
			// Skip if there's already content at this position
			if c.Get(p) != ' ' && c.Get(p) != 0 {
				continue
			}
			
			// Check if this point is a virtual obstacle
			if obstacleFunc(p) {
				// Check if it's a physical obstacle (node body)
				isPhysical := false
				for _, node := range nodes {
					if x >= node.X && x < node.X+node.Width &&
					   y >= node.Y && y < node.Y+node.Height {
						isPhysical = true
						break
					}
				}
				
				// Only show virtual obstacles, not physical ones
				if !isPhysical {
					c.Set(p, '·') // Use middle dot for virtual obstacles
				}
			}
		}
	}
	
	// Now render path start and end points with distinct markers
	// We'll mark points near the start/end to avoid overwriting edge characters
	markedPoints := make(map[core.Point]bool)
	for _, path := range paths {
		if len(path.Points) >= 2 {
			// Mark start point vicinity
			start := path.Points[0]
			for _, neighbor := range []core.Point{
				{X: start.X - 1, Y: start.Y},
				{X: start.X + 1, Y: start.Y},
				{X: start.X, Y: start.Y - 1},
				{X: start.X, Y: start.Y + 1},
			} {
				if !markedPoints[neighbor] && (c.Get(neighbor) == ' ' || c.Get(neighbor) == '·') {
					c.Set(neighbor, 'S') // S for start
					markedPoints[neighbor] = true
					break
				}
			}
			
			// Mark end point vicinity  
			end := path.Points[len(path.Points)-1]
			for _, neighbor := range []core.Point{
				{X: end.X - 1, Y: end.Y},
				{X: end.X + 1, Y: end.Y},
				{X: end.X, Y: end.Y - 1},
				{X: end.X, Y: end.Y + 1},
			} {
				if !markedPoints[neighbor] && (c.Get(neighbor) == ' ' || c.Get(neighbor) == '·') {
					c.Set(neighbor, 'E') // E for end
					markedPoints[neighbor] = true
					break
				}
			}
		}
	}
}

// Helper to extract port information from paths
func extractPortsFromPaths(nodeID int, edge obstacles.EdgeSide, paths map[int]core.Path) []struct{
	point core.Point
	connID int
} {
	var ports []struct{
		point core.Point
		connID int
	}
	
	for connID, path := range paths {
		if path.Metadata != nil {
			// Check source port
			if sourcePort, ok := path.Metadata["sourcePort"].(obstacles.Port); ok {
				if sourcePort.NodeID == nodeID && sourcePort.Edge == edge {
					ports = append(ports, struct{
						point core.Point
						connID int
					}{sourcePort.Point, connID})
				}
			}
			// Check target port
			if targetPort, ok := path.Metadata["targetPort"].(obstacles.Port); ok {
				if targetPort.NodeID == nodeID && targetPort.Edge == edge {
					ports = append(ports, struct{
						point core.Point
						connID int
					}{targetPort.Point, connID})
				}
			}
		}
	}
	
	return ports
}

// Helper to get edge name
func getEdgeName(edge obstacles.EdgeSide) string {
	switch edge {
	case obstacles.North:
		return "North"
	case obstacles.East:
		return "East"
	case obstacles.South:
		return "South"
	case obstacles.West:
		return "West"
	default:
		return "Unknown"
	}
}
package render

import (
	"edd/diagram"
	"edd/layout"
	"edd/pathfinding"
	"fmt"
	"strings"
)

// FlowchartRenderer handles rendering of flowchart diagrams
type FlowchartRenderer struct {
	layout        diagram.LayoutEngine
	pathfinder    diagram.PathFinder
	router        *pathfinding.Router
	capabilities  TerminalCapabilities
	pathRenderer  *PathRenderer
	nodeRenderer  *NodeRenderer
	labelRenderer *LabelRenderer
	debugMode     bool
	showObstacles bool

	// Edit state for cursor display
	editingNodeID int
	editText      string
	cursorPos     int
}

// NewFlowchartRenderer creates a new flowchart diagram renderer
func NewFlowchartRenderer(caps TerminalCapabilities) *FlowchartRenderer {
	// Use vertical layout for flowcharts (top-to-bottom flow)
	layoutEngine := layout.NewVerticalLayout()
	
	// Use smart pathfinder with good defaults
	pathfinder := pathfinding.NewSmartPathFinder(pathfinding.PathCost{
		StraightCost:          10,
		TurnCost:              1000, // Extremely high cost - avoid turns at almost any cost
		ProximityCost:         0,    // Neutral - don't hug or avoid walls
		DirectionBias:         0,    // No bias - treat horizontal and vertical equally for symmetry
		InitialDirectionBonus: 50,   // Strong preference to continue in initial direction
	})
	
	// Add caching for performance. Cache size of 100 handles most diagrams
	// efficiently without excessive memory usage (100 * ~1KB per path = ~100KB)
	cachedPathfinder := pathfinding.NewCachedPathFinder(pathfinder, 100)
	
	// Create router with pathfinder
	router := pathfinding.NewRouter(cachedPathfinder)
	
	return &FlowchartRenderer{
		layout:        layoutEngine,
		pathfinder:    cachedPathfinder,
		router:        router,
		capabilities:  caps,
		pathRenderer:  NewPathRenderer(caps),
		nodeRenderer:  NewNodeRenderer(caps),
		labelRenderer: NewLabelRenderer(),
		debugMode:     false,
		showObstacles: false,
	}
}

// CanRender returns true if this renderer can handle the given diagram type.
func (r *FlowchartRenderer) CanRender(diagramType diagram.DiagramType) bool {
	// Flowchart handles: empty string (default), "flowchart", and "box" (legacy name)
	return diagramType == diagram.DiagramTypeFlowchart || diagramType == "" || diagramType == "box"
}

// Render renders the flowchart diagram and returns the string output.
func (r *FlowchartRenderer) Render(d *diagram.Diagram) (string, error) {
	if d == nil {
		return "", fmt.Errorf("diagram is nil")
	}

	// Step 1: Calculate node dimensions from their text content
	nodes := CalculateNodeDimensions(d.Nodes)

	// Step 2: Choose layout based on diagram hints
	layoutEngine := r.layout // Default to vertical
	flowDirection := pathfinding.FlowVertical

	if d.Hints != nil {
		if layoutHint := d.Hints["layout"]; layoutHint == "horizontal" {
			layoutEngine = layout.NewHorizontalLayout()
			flowDirection = pathfinding.FlowHorizontal
		}
	}

	// Step 3: Run layout algorithm to position nodes
	layoutNodes, err := layoutEngine.Layout(nodes, d.Connections)
	if err != nil {
		return "", fmt.Errorf("layout failed: %w", err)
	}

	// Step 3.1: Adjust dimensions for node being edited (so box grows in real-time)
	if r.editingNodeID >= 0 {
		for i := range layoutNodes {
			if layoutNodes[i].ID == r.editingNodeID {
				// Split edit text into lines
				lines := strings.Split(r.editText, "\n")

				// Recalculate width - find the longest line
				maxWidth := 0
				for _, line := range lines {
					lineWidth := len([]rune(line)) + 1 // +1 for cursor character
					if lineWidth > maxWidth {
						maxWidth = lineWidth
					}
				}
				minWidth := maxWidth + 4 // text + padding
				if minWidth < 8 {
					minWidth = 8
				}
				if minWidth > layoutNodes[i].Width {
					layoutNodes[i].Width = minWidth
				}

				// Recalculate height for multi-line text
				minHeight := len(lines) + 2 // lines + borders
				if minHeight > layoutNodes[i].Height {
					layoutNodes[i].Height = minHeight
				}

				break
			}
		}
	}

	// Step 3.2: Set flow direction on the router
	if areaRouter := r.router.GetAreaRouter(); areaRouter != nil {
		areaRouter.SetFlowDirection(flowDirection)
	}
	
	// Step 3: Set up port manager (after layout is complete)
	// Always use port manager for better connection routing
	portWidth := 1 // Default port width
	portManager := pathfinding.NewPortManager(layoutNodes, portWidth)
	r.router.SetPortManager(portManager)
	
	// Step 4: Route connections between nodes
	paths, err := r.router.RouteConnections(d.Connections, layoutNodes)
	if err != nil {
		return "", fmt.Errorf("connection routing failed: %w", err)
	}
	
	// Step 5: Calculate bounds and create canvas
	bounds := CalculateBounds(layoutNodes, paths)
	
	// Check if we need colors
	needsColor := HasColorHints(d)
	
	// Create appropriate canvas type
	c := CreateCanvas(bounds.Width(), bounds.Height(), needsColor)
	
	// Create offset canvas that handles coordinate translation
	offsetCanvas := NewOffsetCanvas(c, bounds.Min)
	
	// Step 5.1: Debug mode - visualize obstacles if enabled
	// TODO: renderDebugObstacles is currently disabled as it references non-existent methods
	/*
	if r.debugMode {
		return r.renderDebugObstacles(layoutNodes, d.Connections, paths, bounds), nil
	}
	*/
	
	// Step 6: Render the diagram components
	if err := r.renderToCanvas(d, layoutNodes, paths, offsetCanvas); err != nil {
		return "", fmt.Errorf("failed to render to canvas: %w", err)
	}
	
	
	// Step 7: Convert canvas to string output
	var output string
	if coloredCanvas, ok := c.(*ColoredMatrixCanvas); ok {
		// Use colored output if we have a colored canvas
		output = coloredCanvas.ColoredString()
	} else {
		// Regular output
		output = c.String()
	}
	
	return output, nil
}

// RenderWithPositions renders the diagram and returns node positions and connection paths
// This is needed by the TUI for jump label positioning
func (r *FlowchartRenderer) RenderWithPositions(d *diagram.Diagram) (map[int]diagram.Point, map[int]diagram.Path, string, error) {
	// Use the same rendering logic as Render()
	output, err := r.Render(d)
	if err != nil {
		return nil, nil, "", err
	}

	// We need to re-layout to get positions (TODO: optimize by caching)
	nodes := CalculateNodeDimensions(d.Nodes)

	// Choose layout based on hints
	var layoutEngine diagram.LayoutEngine
	var flowDirection pathfinding.FlowDirection

	if d.Hints != nil && d.Hints["layout"] == "horizontal" {
		layoutEngine = layout.NewHorizontalLayout()
		flowDirection = pathfinding.FlowHorizontal
	} else {
		layoutEngine = r.layout
		flowDirection = pathfinding.FlowVertical
	}

	layoutNodes, err := layoutEngine.Layout(nodes, d.Connections)
	if err != nil {
		return nil, nil, output, nil // Return output even if we can't get positions
	}

	// Set flow direction on router for proper pathfinding
	if areaRouter := r.router.GetAreaRouter(); areaRouter != nil {
		areaRouter.SetFlowDirection(flowDirection)
	}

	// Note: Dimension adjustment for editing happens in Render() now
	// so we don't need to duplicate it here

	// Route connections to get paths
	paths, err := r.router.RouteConnections(d.Connections, layoutNodes)
	if err != nil {
		return nil, nil, output, nil // Return output even if routing fails
	}

	// Calculate bounds to get the offset
	bounds := CalculateBounds(layoutNodes, paths)

	// Build position maps with offset applied (to match rendered coordinates)
	positions := make(map[int]diagram.Point)
	for _, node := range layoutNodes {
		positions[node.ID] = diagram.Point{
			X: node.X - bounds.Min.X,
			Y: node.Y - bounds.Min.Y,
		}
	}

	// Adjust paths to account for offset as well
	adjustedPaths := make(map[int]diagram.Path)
	for i, path := range paths {
		adjustedPath := diagram.Path{
			Points:   make([]diagram.Point, len(path.Points)),
			Cost:     path.Cost,
			Metadata: path.Metadata,
		}
		for j, point := range path.Points {
			adjustedPath.Points[j] = diagram.Point{
				X: point.X - bounds.Min.X,
				Y: point.Y - bounds.Min.Y,
			}
		}
		adjustedPaths[i] = adjustedPath
	}

	return positions, adjustedPaths, output, nil
}

// GetBounds returns the required canvas size for the diagram
func (r *FlowchartRenderer) GetBounds(d *diagram.Diagram) (width, height int) {
	// Calculate node dimensions
	nodes := CalculateNodeDimensions(d.Nodes)
	
	// Run layout to get positions
	layoutNodes, err := r.layout.Layout(nodes, d.Connections)
	if err != nil {
		// Return a default size on error
		return 80, 24
	}
	
	// Route connections to get paths
	paths, err := r.router.RouteConnections(d.Connections, layoutNodes)
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
func (r *FlowchartRenderer) renderToCanvas(d *diagram.Diagram, layoutNodes []diagram.Node, paths map[int]diagram.Path, offsetCanvas Canvas) error {
	// Step 1: Render shadows first (so connections can overwrite them)
	for _, node := range layoutNodes {
		r.nodeRenderer.RenderShadowOnly(offsetCanvas, node)
	}
	
	// Step 2: Render nodes (boxes and text) before connections
	// This allows connections to properly connect to node edges
	for _, node := range layoutNodes {
		// Check if this node is being edited
		isEditing := node.ID == r.editingNodeID
		if isEditing {
			if err := r.nodeRenderer.RenderNodeWithEdit(offsetCanvas, node, true, r.editText, r.cursorPos); err != nil {
				return fmt.Errorf("failed to render node %d: %w", node.ID, err)
			}
		} else {
			if err := r.nodeRenderer.RenderNode(offsetCanvas, node); err != nil {
				return fmt.Errorf("failed to render node %d: %w", node.ID, err)
			}
		}
	}
	
	// Step 3: Create arrow configuration
	arrowConfig := pathfinding.NewArrowConfig()
	// For now, use default arrow configuration
	// Future: Could be customized based on diagram metadata
	
	// Step 4: Apply arrow configuration to connections
	connectionsWithArrows := pathfinding.ApplyArrowConfig(d.Connections, paths, arrowConfig)
	
	// Step 5: Render all connections
	// Note: connectionsWithArrows may not maintain the same order as d.Connections if some connections failed to route
	for _, cwa := range connectionsWithArrows {
		hasArrow := cwa.ArrowType == pathfinding.ArrowEnd || cwa.ArrowType == pathfinding.ArrowBoth
		
		// Check if this connection has hints - use the connection from cwa, not d.Connections[i]
		if cwa.Connection.Hints != nil && len(cwa.Connection.Hints) > 0 {
			// Use RenderPathWithHints to apply visual hints
			r.pathRenderer.RenderPathWithHints(offsetCanvas, cwa.Path, hasArrow, cwa.Connection.Hints)
		} else {
			// Use RenderPathWithOptions to enable connection endpoint handling
			r.pathRenderer.RenderPathWithOptions(offsetCanvas, cwa.Path, hasArrow, true)
		}
	}
	
	// Step 6: Render connection labels after all paths are drawn
	// This ensures labels are placed on top of the lines
	for _, cwa := range connectionsWithArrows {
		if cwa.Connection.Label != "" {
			r.labelRenderer.RenderLabel(offsetCanvas, cwa.Path, cwa.Connection.Label, LabelMiddle)
		}
	}
	
	// Step 7: Show virtual obstacles if enabled
	if r.showObstacles {
		r.renderObstacleDots(offsetCanvas, layoutNodes, d.Connections, paths)
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
func (r *FlowchartRenderer) GetRouter() *pathfinding.Router {
	return r.router
}

// SetRouterType sets the type of router to use
func (r *FlowchartRenderer) SetRouterType(routerType pathfinding.RouterType) {
	r.router.SetRouterType(routerType)
}

// SetEditState sets the editing state for cursor display
func (r *FlowchartRenderer) SetEditState(nodeID int, text string, cursorPos int) {
	r.editingNodeID = nodeID
	r.editText = text
	r.cursorPos = cursorPos
}

// renderDebugObstacles creates a debug visualization showing obstacles and paths.
// TODO: This function is currently unused and references non-existent methods
/*
func (r *FlowchartRenderer) renderDebugObstacles(layoutNodes []diagram.Node, connections []diagram.Connection, paths map[int]diagram.Path, bounds diagram.Bounds) string {
	// Import the debug visualizer
	debugViz := NewDebugVisualizer(bounds.Width(), bounds.Height())
	
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
				if sourcePort, ok := path.Metadata["sourcePort"].(pathfinding.Port); ok {
					debugViz.AddPoint(sourcePort.Point, 'P')
				}
				// Show target port
				if targetPort, ok := path.Metadata["targetPort"].(pathfinding.Port); ok {
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
			var obstacles func(diagram.Point) bool
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
*/


// renderObstacleDots adds dots to show virtual obstacles on the canvas
func (r *FlowchartRenderer) renderObstacleDots(c Canvas, nodes []diagram.Node, connections []diagram.Connection, paths map[int]diagram.Path) {
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
			p := diagram.Point{X: x, Y: y}
			
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
	markedPoints := make(map[diagram.Point]bool)
	for _, path := range paths {
		if len(path.Points) >= 2 {
			// Mark start point vicinity
			start := path.Points[0]
			for _, neighbor := range []diagram.Point{
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
			for _, neighbor := range []diagram.Point{
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


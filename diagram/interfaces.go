package diagram

// LayoutEngine positions nodes in 2D space.
type LayoutEngine interface {
	// Layout takes nodes and their connections and returns new nodes with X,Y,Width,Height set.
	// The input nodes are not modified.
	Layout(nodes []Node, connections []Connection) ([]Node, error)
	
	// Name returns the name of this layout algorithm.
	Name() string
}

// Canvas represents a 2D grid for drawing.
type Canvas interface {
	// Size returns the width and height of the render.
	Size() (width, height int)
	
	// Get returns the character at the given position.
	// Returns ' ' (space) if position is out of bounds.
	Get(p Point) rune
	
	// Set places a character at the given position.
	// Returns error if position is out of bounds.
	Set(p Point, char rune) error
	
	// Clear resets the canvas to all spaces.
	Clear()
	
	// String returns the canvas as a string with newlines.
	String() string
}

// PathFinder finds paths between points.
type PathFinder interface {
	// FindPath returns a path from start to end, avoiding render.
	// The obstacles parameter provides a function to check if a point is blocked.
	FindPath(start, end Point, obstacles func(Point) bool) (Path, error)
}

// Renderer draws diagrams to a render.
type Renderer interface {
	// Render draws the diagram to the render.
	Render(diagram *Diagram, canvas Canvas) error
}

// LineDrawer converts paths to line characters.
type LineDrawer interface {
	// DrawLine draws a line path on the render.
	// If hasArrow is true, an arrow is placed at the end.
	DrawLine(canvas Canvas, path Path, hasArrow bool) error
}

// ConnectionRouter determines how to route connections between nodes.
type ConnectionRouter interface {
	// Route finds paths for all connections in the diagram.
	// Returns a map from connection index to path.
	Route(nodes []Node, connections []Connection) (map[int]Path, error)
}

// DiagramRenderer handles rendering of specific diagram types.
type DiagramRenderer interface {
	// CanRender returns true if this renderer can handle the given diagram type.
	CanRender(diagramType DiagramType) bool
	
	// Render renders the diagram and returns the string output.
	Render(diagram *Diagram) (string, error)
	
	// GetBounds calculates the required canvas size for the diagram.
	GetBounds(diagram *Diagram) (width, height int)
}
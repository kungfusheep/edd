// Package core contains the fundamental types used throughout the edd diagram renderer.
package diagram

// Point represents a 2D coordinate in the render.
type Point struct {
	X, Y int
}

// Direction represents a cardinal direction.
type Direction int

const (
	North Direction = iota
	East
	South
	West
)

// String returns the string representation of a Direction.
func (d Direction) String() string {
	switch d {
	case North:
		return "North"
	case East:
		return "East"
	case South:
		return "South"
	case West:
		return "West"
	default:
		return "Unknown"
	}
}

// Opposite returns the opposite direction.
func (d Direction) Opposite() Direction {
	switch d {
	case North:
		return South
	case East:
		return West
	case South:
		return North
	case West:
		return East
	default:
		return d
	}
}

// Node represents a box in the diagram.
type Node struct {
	ID     int               `json:"id"`
	Text   []string          `json:"text"`
	Hints  map[string]string `json:"hints,omitempty"` // Visual hints (style, color, etc.)
	X      int               `json:"-"` // Set by layout engine
	Y      int               `json:"-"` // Set by layout engine
	Width  int               `json:"-"` // Calculated from text
	Height int               `json:"-"` // Calculated from text
}

// Center returns the center point of the node.
func (n Node) Center() Point {
	return Point{
		X: n.X + n.Width/2,
		Y: n.Y + n.Height/2,
	}
}

// Contains checks if a point is inside the node.
func (n Node) Contains(p Point) bool {
	return p.X >= n.X && p.X < n.X+n.Width &&
		p.Y >= n.Y && p.Y < n.Y+n.Height
}

// Connection represents a directed edge between nodes.
type Connection struct {
	ID    int               `json:"id,omitempty"`    // Unique connection identifier  
	From  int               `json:"from"`            // Source node ID
	To    int               `json:"to"`              // Target node ID
	Arrow bool              `json:"arrow,omitempty"` // Whether this connection should have an arrow
	Label string            `json:"label,omitempty"` // Optional label for the connection
	Hints map[string]string `json:"hints,omitempty"` // Visual hints (style, color, etc.)
}

// DiagramType represents the type of diagram
type DiagramType string

// Diagram type constants
const (
	DiagramTypeFlowchart DiagramType = ""         // Default/empty is flowchart for backwards compatibility
	DiagramTypeSequence  DiagramType = "sequence" // UML sequence diagram
)

// Diagram represents a complete diagram with nodes and pathfinding.
type Diagram struct {
	Type        string            `json:"type,omitempty"`      // Diagram type: "sequence", "flowchart", etc.
	Nodes       []Node            `json:"nodes"`
	Connections []Connection      `json:"connections"`
	Metadata    Metadata          `json:"metadata,omitempty"`
	Hints       map[string]string `json:"hints,omitempty"`     // Diagram-level hints (layout, title, etc.)
}

// GetType returns the diagram type as a DiagramType constant
func (d *Diagram) GetType() DiagramType {
	if d.Type == "" {
		return DiagramTypeFlowchart
	}
	return DiagramType(d.Type)
}

// IsSequence returns true if this is a sequence diagram
func (d *Diagram) IsSequence() bool {
	return d.GetType() == DiagramTypeSequence
}

// IsFlowchart returns true if this is a flowchart diagram
func (d *Diagram) IsFlowchart() bool {
	return d.GetType() == DiagramTypeFlowchart || d.Type == ""
}

// Clone creates a deep copy of the diagram
func (d *Diagram) Clone() *Diagram {
	if d == nil {
		return nil
	}
	
	clone := &Diagram{
		Type:        d.Type,
		Nodes:       make([]Node, len(d.Nodes)),
		Connections: make([]Connection, len(d.Connections)),
		Metadata:    d.Metadata, // Metadata is a simple struct, can be copied directly
	}

	// Deep copy diagram-level hints map if it exists
	if d.Hints != nil {
		clone.Hints = make(map[string]string)
		for k, v := range d.Hints {
			clone.Hints[k] = v
		}
	}
	
	// Deep copy nodes (need to copy the Text slice and Hints map)
	for i, node := range d.Nodes {
		textCopy := make([]string, len(node.Text))
		copy(textCopy, node.Text)
		clone.Nodes[i] = Node{
			ID:     node.ID,
			Text:   textCopy,
			X:      node.X,
			Y:      node.Y,
			Width:  node.Width,
			Height: node.Height,
		}
		// Deep copy hints map if it exists
		if node.Hints != nil {
			clone.Nodes[i].Hints = make(map[string]string)
			for k, v := range node.Hints {
				clone.Nodes[i].Hints[k] = v
			}
		}
	}
	
	// Deep copy connections (need to copy Hints map)
	for i, conn := range d.Connections {
		clone.Connections[i] = Connection{
			ID:    conn.ID,
			From:  conn.From,
			To:    conn.To,
			Arrow: conn.Arrow,
			Label: conn.Label,
		}
		// Deep copy hints map if it exists
		if conn.Hints != nil {
			clone.Connections[i].Hints = make(map[string]string)
			for k, v := range conn.Hints {
				clone.Connections[i].Hints[k] = v
			}
		}
	}
	
	return clone
}

// Metadata contains optional diagram metadata.
type Metadata struct {
	Name    string `json:"name,omitempty"`
	Created string `json:"created,omitempty"`
	Version string `json:"version,omitempty"`
}

// Path represents a route through the render.
type Path struct {
	Points   []Point
	Cost     int                    // Used by pathfinding algorithms
	Metadata map[string]interface{} // Optional metadata (e.g., port information)
}

// Length returns the number of points in the path.
func (p Path) Length() int {
	return len(p.Points)
}

// IsEmpty returns true if the path has no points.
func (p Path) IsEmpty() bool {
	return len(p.Points) == 0
}

// Bounds represents a rectangular area.
type Bounds struct {
	Min, Max Point
}

// Width returns the width of the bounds.
func (b Bounds) Width() int {
	return b.Max.X - b.Min.X
}

// Height returns the height of the bounds.
func (b Bounds) Height() int {
	return b.Max.Y - b.Min.Y
}

// Contains checks if a point is within the bounds.
func (b Bounds) Contains(p Point) bool {
	return p.X >= b.Min.X && p.X < b.Max.X &&
		p.Y >= b.Min.Y && p.Y < b.Max.Y
}
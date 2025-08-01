// Package core contains the fundamental types used throughout the edd diagram renderer.
package core

// Point represents a 2D coordinate in the canvas.
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
	ID     int      `json:"id"`
	Text   []string `json:"text"`
	X      int      `json:"-"` // Set by layout engine
	Y      int      `json:"-"` // Set by layout engine
	Width  int      `json:"-"` // Calculated from text
	Height int      `json:"-"` // Calculated from text
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
	From int `json:"from"` // Source node ID
	To   int `json:"to"`   // Target node ID
}

// Diagram represents a complete diagram with nodes and connections.
type Diagram struct {
	Nodes       []Node       `json:"nodes"`
	Connections []Connection `json:"connections"`
	Metadata    Metadata     `json:"metadata,omitempty"`
}

// Metadata contains optional diagram metadata.
type Metadata struct {
	Name    string `json:"name,omitempty"`
	Created string `json:"created,omitempty"`
	Version string `json:"version,omitempty"`
}

// Path represents a route through the canvas.
type Path struct {
	Points []Point
	Cost   int // Used by pathfinding algorithms
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
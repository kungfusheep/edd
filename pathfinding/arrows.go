package pathfinding

import (
	"edd/diagram"
	"fmt"
)

// ArrowType represents the type of arrow to use for a connection.
type ArrowType int

const (
	// ArrowNone indicates no arrow
	ArrowNone ArrowType = iota
	// ArrowEnd indicates an arrow at the end of the connection
	ArrowEnd
	// ArrowStart indicates an arrow at the start of the connection
	ArrowStart
	// ArrowBoth indicates arrows at both ends
	ArrowBoth
)

// ArrowConfig configures arrow placement for pathfinding.
type ArrowConfig struct {
	// DefaultType is the default arrow type for connections
	DefaultType ArrowType
	// TypeOverrides allows specific connections to have different arrow types
	TypeOverrides map[string]ArrowType // key is "fromID->toID" as formatted string
}

// NewArrowConfig creates a new arrow configuration with sensible defaults.
func NewArrowConfig() *ArrowConfig {
	return &ArrowConfig{
		DefaultType:   ArrowEnd,
		TypeOverrides: make(map[string]ArrowType),
	}
}

// GetArrowType returns the arrow type for a specific connection.
func (ac *ArrowConfig) GetArrowType(conn diagram.Connection) ArrowType {
	// Check for overrides
	key := fmt.Sprintf("%d->%d", conn.From, conn.To)
	if arrowType, exists := ac.TypeOverrides[key]; exists {
		return arrowType
	}
	
	// Default behavior: all connections have arrows unless arrow is explicitly false
	// This change: treating missing arrow field as true (default arrow)
	return ac.DefaultType
}

// SetArrowType sets the arrow type for a specific connection.
func (ac *ArrowConfig) SetArrowType(from, to int, arrowType ArrowType) {
	key := fmt.Sprintf("%d->%d", from, to)
	ac.TypeOverrides[key] = arrowType
}

// ShouldDrawArrowAtEnd returns true if an arrow should be drawn at the end of the path.
func (ac *ArrowConfig) ShouldDrawArrowAtEnd(conn diagram.Connection) bool {
	arrowType := ac.GetArrowType(conn)
	return arrowType == ArrowEnd || arrowType == ArrowBoth
}

// ShouldDrawArrowAtStart returns true if an arrow should be drawn at the start of the path.
func (ac *ArrowConfig) ShouldDrawArrowAtStart(conn diagram.Connection) bool {
	arrowType := ac.GetArrowType(conn)
	return arrowType == ArrowStart || arrowType == ArrowBoth
}

// ConnectionWithArrow represents a connection path with arrow configuration.
type ConnectionWithArrow struct {
	Connection diagram.Connection
	Path       diagram.Path
	ArrowType  ArrowType
}

// ApplyArrowConfig applies arrow configuration to routed pathfinding.
func ApplyArrowConfig(connections []diagram.Connection, paths map[int]diagram.Path, config *ArrowConfig) []ConnectionWithArrow {
	result := make([]ConnectionWithArrow, 0, len(connections))
	
	for i, conn := range connections {
		if path, exists := paths[i]; exists {
			result = append(result, ConnectionWithArrow{
				Connection: conn,
				Path:       path,
				ArrowType:  config.GetArrowType(conn),
			})
		}
	}
	
	return result
}
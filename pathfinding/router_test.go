package pathfinding

import (
	"edd/diagram"
	"edd/geometry"
	"testing"
)

func TestRouter_RouteConnection(t *testing.T) {
	// Create a simple pathfinder
	pf := NewAStarPathFinder(PathCost{
		StraightCost: 10,
		TurnCost: 5,
	})
	router := NewRouter(pf)
	
	// Define test nodes
	nodes := []diagram.Node{
		{ID: 1, X: 5, Y: 5, Width: 10, Height: 5},
		{ID: 2, X: 25, Y: 5, Width: 10, Height: 5},
		{ID: 3, X: 15, Y: 20, Width: 10, Height: 5},
	}
	
	tests := []struct {
		name    string
		conn    diagram.Connection
		wantErr bool
		minLen  int // minimum expected path length
	}{
		{
			name:    "simple horizontal connection",
			conn:    diagram.Connection{From: 1, To: 2},
			wantErr: false,
			minLen:  2,
		},
		{
			name:    "diagonal connection",
			conn:    diagram.Connection{From: 1, To: 3},
			wantErr: false,
			minLen:  2,
		},
		{
			name:    "reverse connection",
			conn:    diagram.Connection{From: 2, To: 1},
			wantErr: false,
			minLen:  2,
		},
		{
			name:    "non-existent source",
			conn:    diagram.Connection{From: 99, To: 2},
			wantErr: true,
		},
		{
			name:    "non-existent target",
			conn:    diagram.Connection{From: 1, To: 99},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := router.RouteConnection(tt.conn, nodes)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("RouteConnection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if len(path.Points) < tt.minLen {
					t.Errorf("Path too short: got %d points, want at least %d", 
						len(path.Points), tt.minLen)
				}
				
				// Verify path starts and ends near the nodes
				if len(path.Points) > 0 {
					start := path.Points[0]
					end := path.Points[len(path.Points)-1]
					
					// Check start point is adjacent to source node
					sourceNode := findNode(nodes, tt.conn.From)
					if !isAdjacentToNode(start, sourceNode) {
						t.Errorf("Path start point %v is not adjacent to source node %v", 
							start, sourceNode)
					}
					
					// Check end point is adjacent to target node
					targetNode := findNode(nodes, tt.conn.To)
					if !isAdjacentToNode(end, targetNode) {
						t.Errorf("Path end point %v is not adjacent to target node %v", 
							end, targetNode)
					}
				}
			}
		})
	}
}

func TestRouter_RouteConnections(t *testing.T) {
	pf := NewAStarPathFinder(PathCost{
		StraightCost: 10,
		TurnCost: 5,
	})
	router := NewRouter(pf)
	
	nodes := []diagram.Node{
		{ID: 1, X: 5, Y: 5, Width: 10, Height: 5},
		{ID: 2, X: 25, Y: 5, Width: 10, Height: 5},
		{ID: 3, X: 15, Y: 20, Width: 10, Height: 5},
	}
	
	connections := []diagram.Connection{
		{From: 1, To: 2},
		{From: 2, To: 3},
		{From: 3, To: 1},
	}
	
	paths, err := router.RouteConnections(connections, nodes)
	if err != nil {
		t.Fatalf("RouteConnections() error = %v", err)
	}
	
	// Verify we got paths for all connections
	if len(paths) != len(connections) {
		t.Errorf("Got %d paths, want %d", len(paths), len(connections))
	}
	
	// Verify each path
	for i, conn := range connections {
		path, exists := paths[i]
		if !exists {
			t.Errorf("No path for connection %d", i)
			continue
		}
		
		if len(path.Points) < 2 {
			t.Errorf("Path %d too short: %d points", i, len(path.Points))
		}
		
		// Verify path connects the right nodes
		sourceNode := findNode(nodes, conn.From)
		targetNode := findNode(nodes, conn.To)
		
		if !isAdjacentToNode(path.Points[0], sourceNode) {
			t.Errorf("Path %d doesn't start at source node", i)
		}
		
		if !isAdjacentToNode(path.Points[len(path.Points)-1], targetNode) {
			t.Errorf("Path %d doesn't end at target node", i)
		}
	}
}

func TestGetConnectionPoint(t *testing.T) {
	tests := []struct {
		name     string
		fromNode diagram.Node
		toNode   diagram.Node
		wantSide string // "left", "right", "top", "bottom"
	}{
		{
			name:     "horizontal right",
			fromNode: diagram.Node{X: 10, Y: 10, Width: 10, Height: 10},
			toNode:   diagram.Node{X: 30, Y: 10, Width: 10, Height: 10},
			wantSide: "right",
		},
		{
			name:     "horizontal left",
			fromNode: diagram.Node{X: 30, Y: 10, Width: 10, Height: 10},
			toNode:   diagram.Node{X: 10, Y: 10, Width: 10, Height: 10},
			wantSide: "left",
		},
		{
			name:     "vertical down",
			fromNode: diagram.Node{X: 10, Y: 10, Width: 10, Height: 10},
			toNode:   diagram.Node{X: 10, Y: 30, Width: 10, Height: 10},
			wantSide: "bottom",
		},
		{
			name:     "vertical up",
			fromNode: diagram.Node{X: 10, Y: 30, Width: 10, Height: 10},
			toNode:   diagram.Node{X: 10, Y: 10, Width: 10, Height: 10},
			wantSide: "top",
		},
		{
			name:     "diagonal prefers horizontal",
			fromNode: diagram.Node{X: 10, Y: 10, Width: 10, Height: 10},
			toNode:   diagram.Node{X: 25, Y: 20, Width: 10, Height: 10},
			wantSide: "right", // dx=15, dy=10, so horizontal wins
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			point := getConnectionPoint(&tt.fromNode, &tt.toNode)
			
			switch tt.wantSide {
			case "left":
				if point.X != tt.fromNode.X {
					t.Errorf("Expected left side connection, got point %v", point)
				}
			case "right":
				if point.X != tt.fromNode.X+tt.fromNode.Width-1 {
					t.Errorf("Expected right side connection, got point %v", point)
				}
			case "top":
				if point.Y != tt.fromNode.Y {
					t.Errorf("Expected top side connection, got point %v", point)
				}
			case "bottom":
				if point.Y != tt.fromNode.Y+tt.fromNode.Height-1 {
					t.Errorf("Expected bottom side connection, got point %v", point)
				}
			}
		})
	}
}

// Helper functions

func findNode(nodes []diagram.Node, id int) *diagram.Node {
	for i := range nodes {
		if nodes[i].ID == id {
			return &nodes[i]
		}
	}
	return nil
}

func isAdjacentToNode(point diagram.Point, node *diagram.Node) bool {
	if node == nil {
		return false
	}
	
	// Check if point is on any edge of the node (with 1 pixel tolerance)
	tolerance := 1
	
	// On left or right edge
	if (geometry.Abs(point.X-node.X) <= tolerance || geometry.Abs(point.X-(node.X+node.Width)) <= tolerance) &&
		point.Y >= node.Y-tolerance && point.Y <= node.Y+node.Height+tolerance {
		return true
	}
	
	// On top or bottom edge
	if (geometry.Abs(point.Y-node.Y) <= tolerance || geometry.Abs(point.Y-(node.Y+node.Height)) <= tolerance) &&
		point.X >= node.X-tolerance && point.X <= node.X+node.Width+tolerance {
		return true
	}
	
	return false
}
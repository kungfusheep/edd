package connections

import (
	"edd/core"
	"fmt"
	"math"
)

// BundleThreshold is the minimum number of connections to trigger bundling
const BundleThreshold = 4

// BundleSpacing is the spacing between bundled connections
const BundleSpacing = 1

// BundleConnections creates bundled paths for groups with many connections.
// It routes the middle connection optimally and creates parallel paths for others.
func BundleConnections(group ConnectionGroup, nodes []core.Node, router *Router) (map[int]core.Path, error) {
	if len(group.Connections) < BundleThreshold {
		// Not enough connections for bundling
		return OptimizeGroupedPaths(group, nodes, router)
	}
	
	paths := make(map[int]core.Path)
	
	// Find source and target nodes
	var sourceNode, targetNode *core.Node
	for i := range nodes {
		if nodes[i].ID == group.Connections[0].From {
			sourceNode = &nodes[i]
		}
		if nodes[i].ID == group.Connections[0].To {
			targetNode = &nodes[i]
		}
	}
	
	if sourceNode == nil || targetNode == nil {
		return nil, ErrNodeNotFound
	}
	
	// Route the middle connection optimally
	middleIdx := len(group.Connections) / 2
	
	// Get connection points
	sourcePoint := getConnectionPoint(sourceNode, targetNode)
	targetPoint := getConnectionPoint(targetNode, sourceNode)
	
	// Route middle path
	obstacles := createObstaclesFunctionWithPadding(nodes, sourceNode.ID, targetNode.ID, 2)
	middlePath, err := router.pathFinder.FindPath(sourcePoint, targetPoint, obstacles)
	if err != nil {
		return nil, err
	}
	
	// Store middle path
	paths[group.Indices[middleIdx]] = middlePath
	
	// Create parallel paths for other connections
	for i, connIdx := range group.Indices {
		if i == middleIdx {
			continue // Already routed
		}
		
		// Calculate offset from middle
		offset := (i - middleIdx) * BundleSpacing
		
		// Create parallel path
		parallelPath := createParallelPath(middlePath, offset)
		paths[connIdx] = parallelPath
	}
	
	return paths, nil
}

// createParallelPath creates a path parallel to the reference path with given offset.
func createParallelPath(referencePath core.Path, offset int) core.Path {
	if len(referencePath.Points) < 2 {
		return referencePath
	}
	
	points := make([]core.Point, len(referencePath.Points))
	
	for i, point := range referencePath.Points {
		if i == 0 || i == len(referencePath.Points)-1 {
			// Keep start and end points the same
			points[i] = point
			continue
		}
		
		// Calculate direction of the segment
		var dx, dy int
		if i < len(referencePath.Points)-1 {
			dx = referencePath.Points[i+1].X - point.X
			dy = referencePath.Points[i+1].Y - point.Y
		} else {
			dx = point.X - referencePath.Points[i-1].X
			dy = point.Y - referencePath.Points[i-1].Y
		}
		
		// Calculate perpendicular offset
		if dx == 0 {
			// Vertical segment - offset horizontally
			points[i] = core.Point{
				X: point.X + offset,
				Y: point.Y,
			}
		} else if dy == 0 {
			// Horizontal segment - offset vertically
			points[i] = core.Point{
				X: point.X,
				Y: point.Y + offset,
			}
		} else {
			// Diagonal - offset perpendicular to direction
			length := math.Sqrt(float64(dx*dx + dy*dy))
			perpX := int(float64(-dy) / length * float64(offset))
			perpY := int(float64(dx) / length * float64(offset))
			
			points[i] = core.Point{
				X: point.X + perpX,
				Y: point.Y + perpY,
			}
		}
	}
	
	return core.Path{Points: points, Cost: referencePath.Cost}
}

// shouldBundle determines if a connection group should be bundled.
func shouldBundle(group ConnectionGroup) bool {
	return len(group.Connections) >= BundleThreshold
}

// ErrNodeNotFound is returned when a required node cannot be found
var ErrNodeNotFound = fmt.Errorf("node not found")
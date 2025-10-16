package pathfinding

import (
	"container/heap"
	"edd/diagram"
	"edd/layout"
	"fmt"
	"math"
)

// AStarNode represents a state in the A* search.
type AStarNode struct {
	Point            diagram.Point
	GCost            int        // Cost from start
	HCost            int        // Heuristic cost to goal
	FCost            int        // GCost + HCost
	Parent           *AStarNode
	Direction        Direction // Direction we entered this node from
	InitialDirection Direction // The first direction taken from start
	Index            int       // Index in the heap
}

// NodeQueue is a priority queue for A* nodes.
type NodeQueue []*AStarNode

func (nq NodeQueue) Len() int           { return len(nq) }
func (nq NodeQueue) Less(i, j int) bool {
	// Primary sort by FCost
	if nq[i].FCost != nq[j].FCost {
		return nq[i].FCost < nq[j].FCost
	}
	
	// Tie-breaker 1: Prefer nodes closer to goal (lower HCost)
	if nq[i].HCost != nq[j].HCost {
		return nq[i].HCost < nq[j].HCost
	}
	
	// Tie-breaker 2: For equal costs, use position-based ordering
	// This ensures deterministic and symmetric behavior
	// Prefer the node that balances X and Y distances to create symmetric paths
	return symmetricOrder(nq[i].Point, nq[j].Point)
}

// symmetricOrder provides a deterministic ordering that promotes symmetry
func symmetricOrder(p1, p2 diagram.Point) bool {
	// Order by sum of coordinates first (promotes diagonal movement)
	sum1 := p1.X + p1.Y
	sum2 := p2.X + p2.Y
	if sum1 != sum2 {
		return sum1 < sum2
	}
	
	// Then by X coordinate
	if p1.X != p2.X {
		return p1.X < p2.X
	}
	
	// Finally by Y coordinate
	return p1.Y < p2.Y
}
func (nq NodeQueue) Swap(i, j int) {
	nq[i], nq[j] = nq[j], nq[i]
	nq[i].Index = i
	nq[j].Index = j
}

func (nq *NodeQueue) Push(x interface{}) {
	n := len(*nq)
	node := x.(*AStarNode)
	node.Index = n
	*nq = append(*nq, node)
}

func (nq *NodeQueue) Pop() interface{} {
	old := *nq
	n := len(old)
	node := old[n-1]
	old[n-1] = nil  // avoid memory leak
	node.Index = -1 // for safety
	*nq = old[0 : n-1]
	return node
}

// PointKey is used for efficient map lookups.
type PointKey struct {
	X, Y int
}

// AStarPathFinder implements A* pathfinding for diagrams.
type AStarPathFinder struct {
	costs    PathCost
	maxNodes int // Maximum nodes to explore (safety limit)
}

// NewAStarPathFinder creates a new A* path finder with the given cost model.
func NewAStarPathFinder(costs PathCost) *AStarPathFinder {
	return &AStarPathFinder{
		costs:    costs,
		maxNodes: 50000, // Default safety limit
	}
}

// FindPath finds an optimal path from start to end avoiding render.
func (a *AStarPathFinder) FindPath(start, end diagram.Point, obstacles func(diagram.Point) bool) (diagram.Path, error) {
	if start == end {
		return diagram.Path{Points: []diagram.Point{start}, Cost: 0}, nil
	}
	
	// Check if start or end is blocked
	if obstacles != nil {
		if obstacles(start) {
			return diagram.Path{}, fmt.Errorf("start point is blocked")
		}
		if obstacles(end) {
			return diagram.Path{}, fmt.Errorf("end point is blocked")
		}
	}
	
	// Initialize open and closed sets
	openSet := &NodeQueue{}
	heap.Init(openSet)
	closedSet := make(map[PointKey]bool)
	nodeMap := make(map[PointKey]*AStarNode)
	
	// Create start node
	startNode := &AStarNode{
		Point:     start,
		GCost:     0,
		HCost:     a.heuristic(start, end, DirNone),
		Direction: DirNone,
	}
	startNode.FCost = startNode.GCost + startNode.HCost
	
	heap.Push(openSet, startNode)
	nodeMap[PointKey{start.X, start.Y}] = startNode
	
	nodesExplored := 0
	
	// Main A* loop
	for openSet.Len() > 0 {
		// Safety check
		nodesExplored++
		if nodesExplored > a.maxNodes {
			return diagram.Path{}, fmt.Errorf("pathfinding exceeded node limit")
		}
		
		// Get node with lowest F cost
		current := heap.Pop(openSet).(*AStarNode)
		currentKey := PointKey{current.Point.X, current.Point.Y}
		
		// Check if we reached the goal
		if current.Point == end {
			return a.reconstructPath(current), nil
		}
		
		// Move to closed set
		closedSet[currentKey] = true
		
		// Explore neighbors using symmetric ordering to prevent directional bias
		for _, neighbor := range GetNeighborsSymmetric(current.Point, end) {
			neighborKey := PointKey{neighbor.X, neighbor.Y}
			
			// Skip if in closed set
			if closedSet[neighborKey] {
				continue
			}
			
			// Skip if obstacle
			if obstacles != nil && obstacles(neighbor) {
				continue
			}
			
			// Calculate costs
			dir := GetDirection(current.Point, neighbor)

			// Determine initial direction (propagate from parent or set for first move)
			var initialDir Direction
			if current.Parent == nil {
				// This is the first move from start
				initialDir = dir
			} else {
				// Propagate initial direction from parent
				initialDir = current.InitialDirection
			}

			tentativeGCost := a.calculateGCost(current, neighbor, dir, obstacles)

			// Check if we've seen this node before
			existingNode, exists := nodeMap[neighborKey]

			if !exists {
				// New node
				newNode := &AStarNode{
					Point:            neighbor,
					GCost:            tentativeGCost,
					HCost:            a.heuristic(neighbor, end, dir),
					Parent:           current,
					Direction:        dir,
					InitialDirection: initialDir,
				}
				newNode.FCost = newNode.GCost + newNode.HCost

				heap.Push(openSet, newNode)
				nodeMap[neighborKey] = newNode
			} else if tentativeGCost < existingNode.GCost {
				// Found a better path to existing node
				existingNode.GCost = tentativeGCost
				existingNode.FCost = existingNode.GCost + existingNode.HCost
				existingNode.Parent = current
				existingNode.Direction = dir
				existingNode.InitialDirection = initialDir

				// Fix heap ordering
				heap.Fix(openSet, existingNode.Index)
			}
		}
	}
	
	return diagram.Path{}, fmt.Errorf("no path found")
}

// heuristic calculates the estimated cost to reach the goal.
func (a *AStarPathFinder) heuristic(current, goal diagram.Point, currentDir Direction) int {
	// Manhattan distance
	dx := layout.Abs(goal.X - current.X)
	dy := layout.Abs(goal.Y - current.Y)
	distance := dx + dy
	
	// Base cost using straight movement
	h := distance * a.costs.StraightCost
	
	// Add minimum turn cost if we'll need at least one turn
	// This makes the heuristic more accurate without breaking admissibility
	if dx > 0 && dy > 0 {
		h += a.costs.TurnCost
	}
	
	// Add a tiny tie-breaker to prefer paths that move toward the goal
	// This reduces node expansion without affecting optimality
	tieBreaker := (dx + dy) / 1000
	
	return h + tieBreaker
}

// calculateGCost calculates the cost to move from current to next.
func (a *AStarPathFinder) calculateGCost(current *AStarNode, next diagram.Point, nextDir Direction, obstacles func(diagram.Point) bool) int {
	// Base movement cost
	cost := a.costs.StraightCost

	// Add turn cost if changing direction
	if current.Parent != nil && current.Direction != DirNone && current.Direction != nextDir {
		cost += a.costs.TurnCost

		// Extra penalty for turning again shortly after a previous turn (jitter penalty)
		// This prevents zigzag patterns like: down, left 1 cell, down, right 1 cell
		if current.Parent.Parent != nil && current.Parent.Direction != current.Direction {
			// We turned at parent, and now turning again - add extra penalty
			cost += a.costs.TurnCost // Double the turn cost for quick direction reversals
		}
	}

	// Apply initial direction bonus
	// Encourage continuing in the initial direction taken from the start
	if a.costs.InitialDirectionBonus != 0 && current.InitialDirection != DirNone {
		if nextDir == current.InitialDirection {
			// Reduce cost when moving in the initial direction
			bonus := a.costs.InitialDirectionBonus
			if bonus < cost {
				cost -= bonus
			}
		}
	}

	// Apply proximity cost/bonus based on obstacles
	if a.costs.ProximityCost != 0 && obstacles != nil {
		adjacentObstacles := a.countAdjacentObstacles(next, obstacles)
		if adjacentObstacles > 0 {
			// ProximityCost > 0: avoid walls (increase cost)
			// ProximityCost < 0: hug walls (decrease cost)
			// Give extra weight to corners (2+ adjacent walls)
			weight := adjacentObstacles
			if adjacentObstacles >= 2 {
				weight = adjacentObstacles * 2 // Double bonus for corners
			}
			proximityCost := (a.costs.ProximityCost * weight) / 4
			cost += proximityCost
		}
	}

	// Direction bias (prefer horizontal/vertical based on setting)
	if a.costs.DirectionBias != 0 {
		if a.costs.DirectionBias > 0 && (nextDir == DirEast || nextDir == DirWest) {
			// Prefer horizontal movement
			bias := layout.Abs(a.costs.DirectionBias)
			if bias < cost {
				cost -= bias
			}
		} else if a.costs.DirectionBias < 0 && (nextDir == DirNorth || nextDir == DirSouth) {
			// Prefer vertical movement
			bias := layout.Abs(a.costs.DirectionBias)
			if bias < cost {
				cost -= bias
			}
		}
	}

	return current.GCost + cost
}

// reconstructPath builds the final path from the goal node.
func (a *AStarPathFinder) reconstructPath(goalNode *AStarNode) diagram.Path {
	points := []diagram.Point{}
	totalCost := goalNode.GCost
	
	// Walk backwards from goal to start
	current := goalNode
	for current != nil {
		points = append([]diagram.Point{current.Point}, points...)
		current = current.Parent
	}
	
	return diagram.Path{
		Points: points,
		Cost:   totalCost,
	}
}

// countAdjacentObstacles counts how many adjacent cells contain render.
func (a *AStarPathFinder) countAdjacentObstacles(p diagram.Point, obstacles func(diagram.Point) bool) int {
	count := 0
	// Check all 4 cardinal directions
	neighbors := GetNeighbors(p)
	for _, neighbor := range neighbors {
		if obstacles(neighbor) {
			count++
		}
	}
	return count
}

// SetMaxNodes sets the maximum number of nodes to explore.
func (a *AStarPathFinder) SetMaxNodes(max int) {
	a.maxNodes = max
}

// FindPathToArea finds an optimal path from start to the edge of a target area.
// The target area is defined by a rectangle (node bounds).
// The path will terminate at the first point on the edge of the area that is not blocked.
func (a *AStarPathFinder) FindPathToArea(start diagram.Point, targetNode diagram.Node, obstacles func(diagram.Point) bool) (diagram.Path, error) {
	// Check if start is blocked
	if obstacles != nil && obstacles(start) {
		return diagram.Path{}, fmt.Errorf("start point is blocked")
	}
	
	// Check if start is already at the target edge
	if isAtNodeEdge(start, targetNode) {
		return diagram.Path{Points: []diagram.Point{start}, Cost: 0}, nil
	}
	
	// Initialize open and closed sets
	openSet := &NodeQueue{}
	heap.Init(openSet)
	closedSet := make(map[PointKey]bool)
	nodeMap := make(map[PointKey]*AStarNode)
	
	// Create start node
	startNode := &AStarNode{
		Point:     start,
		GCost:     0,
		HCost:     a.heuristicToArea(start, targetNode, DirNone),
		Direction: DirNone,
	}
	startNode.FCost = startNode.GCost + startNode.HCost
	
	heap.Push(openSet, startNode)
	nodeMap[PointKey{start.X, start.Y}] = startNode
	
	nodesExplored := 0
	
	// Main A* loop
	for openSet.Len() > 0 {
		// Safety check
		nodesExplored++
		if nodesExplored > a.maxNodes {
			return diagram.Path{}, fmt.Errorf("pathfinding exceeded node limit")
		}
		
		// Get node with lowest F cost
		current := heap.Pop(openSet).(*AStarNode)
		currentKey := PointKey{current.Point.X, current.Point.Y}
		
		// Check if we reached the target area edge AND it's not blocked
		// IMPORTANT: We must respect virtual obstacles even at the target edge
		if isAtNodeEdge(current.Point, targetNode) && (obstacles == nil || !obstacles(current.Point)) {
			return a.reconstructPath(current), nil
		}

		// Move to closed set
		closedSet[currentKey] = true

		// Explore neighbors using symmetric ordering to prevent directional bias
		// Use center of target node as goal for symmetric exploration
		targetCenter := diagram.Point{
			X: targetNode.X + targetNode.Width/2,
			Y: targetNode.Y + targetNode.Height/2,
		}
		for _, neighbor := range GetNeighborsSymmetric(current.Point, targetCenter) {
			neighborKey := PointKey{neighbor.X, neighbor.Y}
			
			// Skip if in closed set
			if closedSet[neighborKey] {
				continue
			}
			
			// Skip if obstacle
			// Virtual obstacles should block even at target edges - that's the whole point!
			if obstacles != nil && obstacles(neighbor) {
				// Debug: log when obstacles block
				// fmt.Printf("    Blocked at (%d,%d)\n", neighbor.X, neighbor.Y)
				continue
			}
			
			// Calculate costs
			dir := GetDirection(current.Point, neighbor)

			// Determine initial direction (propagate from parent or set for first move)
			var initialDir Direction
			if current.Parent == nil {
				// This is the first move from start
				initialDir = dir
			} else {
				// Propagate initial direction from parent
				initialDir = current.InitialDirection
			}

			tentativeGCost := a.calculateGCost(current, neighbor, dir, obstacles)

			// Check if we've seen this node before
			existingNode, exists := nodeMap[neighborKey]

			if !exists {
				// New node
				newNode := &AStarNode{
					Point:            neighbor,
					GCost:            tentativeGCost,
					HCost:            a.heuristicToArea(neighbor, targetNode, dir),
					Parent:           current,
					Direction:        dir,
					InitialDirection: initialDir,
				}
				newNode.FCost = newNode.GCost + newNode.HCost

				heap.Push(openSet, newNode)
				nodeMap[neighborKey] = newNode
			} else if tentativeGCost < existingNode.GCost {
				// Found a better path to existing node
				existingNode.GCost = tentativeGCost
				existingNode.FCost = existingNode.GCost + existingNode.HCost
				existingNode.Parent = current
				existingNode.Direction = dir
				existingNode.InitialDirection = initialDir

				// Fix heap ordering
				heap.Fix(openSet, existingNode.Index)
			}
		}
	}
	
	return diagram.Path{}, fmt.Errorf("no path found to target area")
}

// heuristicToArea calculates the estimated cost to reach the target area.
func (a *AStarPathFinder) heuristicToArea(current diagram.Point, targetNode diagram.Node, currentDir Direction) int {
	// Calculate minimum Manhattan distance to any edge of the target
	minDist := math.MaxInt32
	
	// Check distance to each edge
	// Top edge
	if current.Y < targetNode.Y {
		dist := layout.Abs(targetNode.Y - current.Y) + layout.Abs(current.X - (targetNode.X + targetNode.Width/2))
		if dist < minDist {
			minDist = dist
		}
	}
	
	// Bottom edge
	if current.Y > targetNode.Y + targetNode.Height - 1 {
		dist := layout.Abs(current.Y - (targetNode.Y + targetNode.Height - 1)) + layout.Abs(current.X - (targetNode.X + targetNode.Width/2))
		if dist < minDist {
			minDist = dist
		}
	}
	
	// Left edge
	if current.X < targetNode.X {
		dist := layout.Abs(targetNode.X - current.X) + layout.Abs(current.Y - (targetNode.Y + targetNode.Height/2))
		if dist < minDist {
			minDist = dist
		}
	}
	
	// Right edge
	if current.X > targetNode.X + targetNode.Width - 1 {
		dist := layout.Abs(current.X - (targetNode.X + targetNode.Width - 1)) + layout.Abs(current.Y - (targetNode.Y + targetNode.Height/2))
		if dist < minDist {
			minDist = dist
		}
	}
	
	// If we're inside or at the edge, distance is 0
	if minDist == math.MaxInt32 {
		minDist = 0
	}
	
	// Base cost using straight movement
	h := minDist * a.costs.StraightCost
	
	// Add minimum turn cost if we'll need at least one turn
	if minDist > 0 && currentDir != DirNone {
		// Simplified turn estimation
		h += a.costs.TurnCost / 2
	}
	
	return h
}

// isAtNodeEdge checks if a point is exactly at the edge of a node (not inside).
func isAtNodeEdge(p diagram.Point, node diagram.Node) bool {
	// Check if on the perimeter of the node
	onVerticalEdge := (p.X == node.X-1 || p.X == node.X+node.Width) && 
	                  p.Y >= node.Y-1 && p.Y <= node.Y+node.Height
	onHorizontalEdge := (p.Y == node.Y-1 || p.Y == node.Y+node.Height) && 
	                    p.X >= node.X-1 && p.X <= node.X+node.Width
	
	return onVerticalEdge || onHorizontalEdge
}
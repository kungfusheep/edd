package pathfinding

import (
	"container/heap"
	"edd/core"
	"fmt"
)

// AStarNode represents a state in the A* search.
type AStarNode struct {
	Point     core.Point
	GCost     int        // Cost from start
	HCost     int        // Heuristic cost to goal
	FCost     int        // GCost + HCost
	Parent    *AStarNode
	Direction Direction  // Direction we entered this node from
	Index     int        // Index in the heap
}

// NodeQueue is a priority queue for A* nodes.
type NodeQueue []*AStarNode

func (nq NodeQueue) Len() int           { return len(nq) }
func (nq NodeQueue) Less(i, j int) bool { return nq[i].FCost < nq[j].FCost }
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

// FindPath finds an optimal path from start to end avoiding obstacles.
func (a *AStarPathFinder) FindPath(start, end core.Point, obstacles func(core.Point) bool) (core.Path, error) {
	if start == end {
		return core.Path{Points: []core.Point{start}, Cost: 0}, nil
	}
	
	// Check if start or end is blocked
	if obstacles != nil {
		if obstacles(start) {
			return core.Path{}, fmt.Errorf("start point is blocked")
		}
		if obstacles(end) {
			return core.Path{}, fmt.Errorf("end point is blocked")
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
		HCost:     a.heuristic(start, end, None),
		Direction: None,
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
			return core.Path{}, fmt.Errorf("pathfinding exceeded node limit")
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
		
		// Explore neighbors
		for _, neighbor := range GetNeighbors(current.Point) {
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
			tentativeGCost := a.calculateGCost(current, neighbor, dir, obstacles)
			
			// Check if we've seen this node before
			existingNode, exists := nodeMap[neighborKey]
			
			if !exists {
				// New node
				newNode := &AStarNode{
					Point:     neighbor,
					GCost:     tentativeGCost,
					HCost:     a.heuristic(neighbor, end, dir),
					Parent:    current,
					Direction: dir,
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
				
				// Fix heap ordering
				heap.Fix(openSet, existingNode.Index)
			}
		}
	}
	
	return core.Path{}, fmt.Errorf("no path found")
}

// heuristic calculates the estimated cost to reach the goal.
func (a *AStarPathFinder) heuristic(current, goal core.Point, currentDir Direction) int {
	// Manhattan distance
	dx := abs(goal.X - current.X)
	dy := abs(goal.Y - current.Y)
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
func (a *AStarPathFinder) calculateGCost(current *AStarNode, next core.Point, nextDir Direction, obstacles func(core.Point) bool) int {
	// Base movement cost
	cost := a.costs.StraightCost
	
	// Add turn cost if changing direction
	if current.Parent != nil && current.Direction != None && current.Direction != nextDir {
		cost += a.costs.TurnCost
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
		if a.costs.DirectionBias > 0 && (nextDir == East || nextDir == West) {
			// Prefer horizontal movement
			bias := abs(a.costs.DirectionBias)
			if bias < cost {
				cost -= bias
			}
		} else if a.costs.DirectionBias < 0 && (nextDir == North || nextDir == South) {
			// Prefer vertical movement
			bias := abs(a.costs.DirectionBias)
			if bias < cost {
				cost -= bias
			}
		}
	}
	
	return current.GCost + cost
}

// reconstructPath builds the final path from the goal node.
func (a *AStarPathFinder) reconstructPath(goalNode *AStarNode) core.Path {
	points := []core.Point{}
	totalCost := goalNode.GCost
	
	// Walk backwards from goal to start
	current := goalNode
	for current != nil {
		points = append([]core.Point{current.Point}, points...)
		current = current.Parent
	}
	
	return core.Path{
		Points: points,
		Cost:   totalCost,
	}
}

// countAdjacentObstacles counts how many adjacent cells contain obstacles.
func (a *AStarPathFinder) countAdjacentObstacles(p core.Point, obstacles func(core.Point) bool) int {
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
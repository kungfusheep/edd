package diagram

// AreaPathFinder extends PathFinder with area-based routing capability
type AreaPathFinder interface {
	PathFinder
	// FindPathToArea finds a path from start point to the edge of a target area (node)
	FindPathToArea(start Point, targetNode Node, obstacles func(Point) bool) (Path, error)
}
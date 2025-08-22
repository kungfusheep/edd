package geometry

// Abs returns the absolute value of an integer.
func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Min returns the minimum of two integers.
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Max returns the maximum of two integers.
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ManhattanDistance calculates the Manhattan distance between two points.
func ManhattanDistance(x1, y1, x2, y2 int) int {
	return Abs(x2-x1) + Abs(y2-y1)
}

// IsHorizontal returns true if the line from (x1,y1) to (x2,y2) is more horizontal than vertical.
func IsHorizontal(x1, y1, x2, y2 int) bool {
	return Abs(x2-x1) > Abs(y2-y1)
}

// IsVertical returns true if the line from (x1,y1) to (x2,y2) is more vertical than horizontal.
func IsVertical(x1, y1, x2, y2 int) bool {
	return Abs(y2-y1) > Abs(x2-x1)
}
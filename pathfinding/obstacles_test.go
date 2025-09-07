package pathfinding

import (
	"edd/diagram"
	"testing"
)

func TestRectangleObstacle(t *testing.T) {
	rect := RectangleObstacle{
		X:       10,
		Y:       10,
		Width:   5,
		Height:  5,
		Padding: 1,
	}
	
	tests := []struct {
		point    diagram.Point
		contains bool
	}{
		// Inside rectangle
		{diagram.Point{12, 12}, true},
		{diagram.Point{10, 10}, true},
		{diagram.Point{14, 14}, true},
		
		// In padding area
		{diagram.Point{9, 10}, true},
		{diagram.Point{15, 12}, true},
		{diagram.Point{12, 9}, true},
		{diagram.Point{12, 15}, true},
		
		// Outside
		{diagram.Point{8, 10}, false},
		{diagram.Point{16, 12}, false},
		{diagram.Point{12, 8}, false},
		{diagram.Point{12, 16}, false},
		{diagram.Point{0, 0}, false},
		{diagram.Point{100, 100}, false},
	}
	
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if rect.Contains(tt.point) != tt.contains {
				t.Errorf("Contains(%v) = %v, want %v", tt.point, rect.Contains(tt.point), tt.contains)
			}
		})
	}
}

func TestCreateNodeObstacleChecker(t *testing.T) {
	nodes := []diagram.Node{
		{ID: 1, X: 10, Y: 10, Width: 5, Height: 5},
		{ID: 2, X: 20, Y: 20, Width: 10, Height: 3},
	}
	
	checker := CreateNodeObstacleChecker(nodes, 1)
	
	tests := []struct {
		point      diagram.Point
		isObstacle bool
	}{
		// First node area
		{diagram.Point{12, 12}, true},
		{diagram.Point{9, 10}, true}, // padding
		
		// Second node area
		{diagram.Point{25, 21}, true},
		{diagram.Point{19, 20}, true}, // padding
		
		// Free space
		{diagram.Point{0, 0}, false},
		{diagram.Point{17, 17}, false},
		{diagram.Point{50, 50}, false},
	}
	
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if checker(tt.point) != tt.isObstacle {
				t.Errorf("checker(%v) = %v, want %v", tt.point, checker(tt.point), tt.isObstacle)
			}
		})
	}
}

func TestCreateBoundsObstacleChecker(t *testing.T) {
	bounds := diagram.Bounds{
		Min: diagram.Point{X: 0, Y: 0},
		Max: diagram.Point{X: 100, Y: 50},
	}
	
	checker := CreateBoundsObstacleChecker(bounds)
	
	tests := []struct {
		point      diagram.Point
		isObstacle bool
	}{
		// Inside bounds
		{diagram.Point{50, 25}, false},
		{diagram.Point{0, 0}, false},
		{diagram.Point{99, 49}, false},
		
		// Outside bounds
		{diagram.Point{-1, 25}, true},
		{diagram.Point{100, 25}, true},
		{diagram.Point{50, -1}, true},
		{diagram.Point{50, 50}, true},
		{diagram.Point{-10, -10}, true},
		{diagram.Point{200, 200}, true},
	}
	
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if checker(tt.point) != tt.isObstacle {
				t.Errorf("checker(%v) = %v, want %v", tt.point, checker(tt.point), tt.isObstacle)
			}
		})
	}
}

func TestCombineObstacleCheckers(t *testing.T) {
	// Create a bounds checker
	boundsChecker := CreateBoundsObstacleChecker(diagram.Bounds{
		Min: diagram.Point{X: 0, Y: 0},
		Max: diagram.Point{X: 50, Y: 50},
	})
	
	// Create a node checker
	nodeChecker := CreateNodeObstacleChecker([]diagram.Node{
		{X: 20, Y: 20, Width: 10, Height: 10},
	}, 0)
	
	// Combine them
	combined := CombineObstacleCheckers(boundsChecker, nodeChecker)
	
	tests := []struct {
		point      diagram.Point
		isObstacle bool
		reason     string
	}{
		{diagram.Point{10, 10}, false, "inside bounds, not in node"},
		{diagram.Point{25, 25}, true, "inside node"},
		{diagram.Point{-5, 25}, true, "outside bounds"},
		{diagram.Point{60, 25}, true, "outside bounds"},
		{diagram.Point{25, -5}, true, "outside bounds"},
		{diagram.Point{25, 60}, true, "outside bounds"},
	}
	
	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			if combined(tt.point) != tt.isObstacle {
				t.Errorf("combined(%v) = %v, want %v", tt.point, combined(tt.point), tt.isObstacle)
			}
		})
	}
}

func TestPathObstacle(t *testing.T) {
	path := PathObstacle{
		Points: []diagram.Point{
			{X: 10, Y: 10},
			{X: 20, Y: 10},
			{X: 20, Y: 20},
		},
		Thickness: 1,
	}
	
	tests := []struct {
		point    diagram.Point
		contains bool
		reason   string
	}{
		// On horizontal segment
		{diagram.Point{15, 10}, true, "on horizontal line"},
		{diagram.Point{10, 10}, true, "at start point"},
		{diagram.Point{20, 10}, true, "at corner"},
		
		// On vertical segment
		{diagram.Point{20, 15}, true, "on vertical line"},
		{diagram.Point{20, 20}, true, "at end point"},
		
		// Within thickness
		{diagram.Point{15, 9}, true, "within thickness of horizontal"},
		{diagram.Point{15, 11}, true, "within thickness of horizontal"},
		{diagram.Point{19, 15}, true, "within thickness of vertical"},
		{diagram.Point{21, 15}, true, "within thickness of vertical"},
		
		// Outside
		{diagram.Point{15, 8}, false, "outside thickness"},
		{diagram.Point{15, 12}, false, "outside thickness"},
		{diagram.Point{25, 15}, false, "beyond path"},
		{diagram.Point{5, 5}, false, "far from path"},
	}
	
	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			if path.Contains(tt.point) != tt.contains {
				t.Errorf("Contains(%v) = %v, want %v", tt.point, path.Contains(tt.point), tt.contains)
			}
		})
	}
}

func TestCreateRegionObstacleChecker(t *testing.T) {
	regions := []Region{
		{
			Bounds:     diagram.Bounds{Min: diagram.Point{X: 10, Y: 10}, Max: diagram.Point{X: 20, Y: 20}},
			IsObstacle: true,
		},
		{
			Bounds:     diagram.Bounds{Min: diagram.Point{X: 30, Y: 30}, Max: diagram.Point{X: 40, Y: 40}},
			IsObstacle: false, // This is a high-cost region, not an obstacle
			Cost:       50,
		},
	}
	
	checker := CreateRegionObstacleChecker(regions)
	
	tests := []struct {
		point      diagram.Point
		isObstacle bool
		reason     string
	}{
		{diagram.Point{15, 15}, true, "inside obstacle region"},
		{diagram.Point{35, 35}, false, "inside cost region (not obstacle)"},
		{diagram.Point{5, 5}, false, "outside all regions"},
		{diagram.Point{25, 25}, false, "between regions"},
		{diagram.Point{10, 10}, true, "on obstacle boundary"},
		{diagram.Point{20, 20}, false, "just outside obstacle (exclusive max)"},
	}
	
	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			if checker(tt.point) != tt.isObstacle {
				t.Errorf("checker(%v) = %v, want %v", tt.point, checker(tt.point), tt.isObstacle)
			}
		})
	}
}

func TestGetRegionCost(t *testing.T) {
	regions := []Region{
		{
			Bounds:     diagram.Bounds{Min: diagram.Point{X: 10, Y: 10}, Max: diagram.Point{X: 20, Y: 20}},
			IsObstacle: true,
			Cost:       0, // Obstacles don't add cost
		},
		{
			Bounds:     diagram.Bounds{Min: diagram.Point{X: 30, Y: 30}, Max: diagram.Point{X: 40, Y: 40}},
			IsObstacle: false,
			Cost:       50,
		},
		{
			Bounds:     diagram.Bounds{Min: diagram.Point{X: 35, Y: 35}, Max: diagram.Point{X: 45, Y: 45}},
			IsObstacle: false,
			Cost:       30, // Overlaps with previous region
		},
	}
	
	tests := []struct {
		point        diagram.Point
		expectedCost int
		reason       string
	}{
		{diagram.Point{5, 5}, 0, "outside all regions"},
		{diagram.Point{15, 15}, 0, "inside obstacle (no cost)"},
		{diagram.Point{35, 35}, 80, "inside two overlapping cost regions"},
		{diagram.Point{32, 32}, 50, "inside single cost region"},
		{diagram.Point{42, 42}, 30, "inside only second cost region"},
	}
	
	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			cost := GetRegionCost(tt.point, regions)
			if cost != tt.expectedCost {
				t.Errorf("GetRegionCost(%v) = %d, want %d", tt.point, cost, tt.expectedCost)
			}
		})
	}
}
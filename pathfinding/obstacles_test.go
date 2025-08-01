package pathfinding

import (
	"edd/core"
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
		point    core.Point
		contains bool
	}{
		// Inside rectangle
		{core.Point{12, 12}, true},
		{core.Point{10, 10}, true},
		{core.Point{14, 14}, true},
		
		// In padding area
		{core.Point{9, 10}, true},
		{core.Point{15, 12}, true},
		{core.Point{12, 9}, true},
		{core.Point{12, 15}, true},
		
		// Outside
		{core.Point{8, 10}, false},
		{core.Point{16, 12}, false},
		{core.Point{12, 8}, false},
		{core.Point{12, 16}, false},
		{core.Point{0, 0}, false},
		{core.Point{100, 100}, false},
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
	nodes := []core.Node{
		{ID: 1, X: 10, Y: 10, Width: 5, Height: 5},
		{ID: 2, X: 20, Y: 20, Width: 10, Height: 3},
	}
	
	checker := CreateNodeObstacleChecker(nodes, 1)
	
	tests := []struct {
		point      core.Point
		isObstacle bool
	}{
		// First node area
		{core.Point{12, 12}, true},
		{core.Point{9, 10}, true}, // padding
		
		// Second node area
		{core.Point{25, 21}, true},
		{core.Point{19, 20}, true}, // padding
		
		// Free space
		{core.Point{0, 0}, false},
		{core.Point{17, 17}, false},
		{core.Point{50, 50}, false},
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
	bounds := core.Bounds{
		Min: core.Point{X: 0, Y: 0},
		Max: core.Point{X: 100, Y: 50},
	}
	
	checker := CreateBoundsObstacleChecker(bounds)
	
	tests := []struct {
		point      core.Point
		isObstacle bool
	}{
		// Inside bounds
		{core.Point{50, 25}, false},
		{core.Point{0, 0}, false},
		{core.Point{99, 49}, false},
		
		// Outside bounds
		{core.Point{-1, 25}, true},
		{core.Point{100, 25}, true},
		{core.Point{50, -1}, true},
		{core.Point{50, 50}, true},
		{core.Point{-10, -10}, true},
		{core.Point{200, 200}, true},
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
	boundsChecker := CreateBoundsObstacleChecker(core.Bounds{
		Min: core.Point{X: 0, Y: 0},
		Max: core.Point{X: 50, Y: 50},
	})
	
	// Create a node checker
	nodeChecker := CreateNodeObstacleChecker([]core.Node{
		{X: 20, Y: 20, Width: 10, Height: 10},
	}, 0)
	
	// Combine them
	combined := CombineObstacleCheckers(boundsChecker, nodeChecker)
	
	tests := []struct {
		point      core.Point
		isObstacle bool
		reason     string
	}{
		{core.Point{10, 10}, false, "inside bounds, not in node"},
		{core.Point{25, 25}, true, "inside node"},
		{core.Point{-5, 25}, true, "outside bounds"},
		{core.Point{60, 25}, true, "outside bounds"},
		{core.Point{25, -5}, true, "outside bounds"},
		{core.Point{25, 60}, true, "outside bounds"},
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
		Points: []core.Point{
			{X: 10, Y: 10},
			{X: 20, Y: 10},
			{X: 20, Y: 20},
		},
		Thickness: 1,
	}
	
	tests := []struct {
		point    core.Point
		contains bool
		reason   string
	}{
		// On horizontal segment
		{core.Point{15, 10}, true, "on horizontal line"},
		{core.Point{10, 10}, true, "at start point"},
		{core.Point{20, 10}, true, "at corner"},
		
		// On vertical segment
		{core.Point{20, 15}, true, "on vertical line"},
		{core.Point{20, 20}, true, "at end point"},
		
		// Within thickness
		{core.Point{15, 9}, true, "within thickness of horizontal"},
		{core.Point{15, 11}, true, "within thickness of horizontal"},
		{core.Point{19, 15}, true, "within thickness of vertical"},
		{core.Point{21, 15}, true, "within thickness of vertical"},
		
		// Outside
		{core.Point{15, 8}, false, "outside thickness"},
		{core.Point{15, 12}, false, "outside thickness"},
		{core.Point{25, 15}, false, "beyond path"},
		{core.Point{5, 5}, false, "far from path"},
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
			Bounds:     core.Bounds{Min: core.Point{X: 10, Y: 10}, Max: core.Point{X: 20, Y: 20}},
			IsObstacle: true,
		},
		{
			Bounds:     core.Bounds{Min: core.Point{X: 30, Y: 30}, Max: core.Point{X: 40, Y: 40}},
			IsObstacle: false, // This is a high-cost region, not an obstacle
			Cost:       50,
		},
	}
	
	checker := CreateRegionObstacleChecker(regions)
	
	tests := []struct {
		point      core.Point
		isObstacle bool
		reason     string
	}{
		{core.Point{15, 15}, true, "inside obstacle region"},
		{core.Point{35, 35}, false, "inside cost region (not obstacle)"},
		{core.Point{5, 5}, false, "outside all regions"},
		{core.Point{25, 25}, false, "between regions"},
		{core.Point{10, 10}, true, "on obstacle boundary"},
		{core.Point{20, 20}, false, "just outside obstacle (exclusive max)"},
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
			Bounds:     core.Bounds{Min: core.Point{X: 10, Y: 10}, Max: core.Point{X: 20, Y: 20}},
			IsObstacle: true,
			Cost:       0, // Obstacles don't add cost
		},
		{
			Bounds:     core.Bounds{Min: core.Point{X: 30, Y: 30}, Max: core.Point{X: 40, Y: 40}},
			IsObstacle: false,
			Cost:       50,
		},
		{
			Bounds:     core.Bounds{Min: core.Point{X: 35, Y: 35}, Max: core.Point{X: 45, Y: 45}},
			IsObstacle: false,
			Cost:       30, // Overlaps with previous region
		},
	}
	
	tests := []struct {
		point        core.Point
		expectedCost int
		reason       string
	}{
		{core.Point{5, 5}, 0, "outside all regions"},
		{core.Point{15, 15}, 0, "inside obstacle (no cost)"},
		{core.Point{35, 35}, 80, "inside two overlapping cost regions"},
		{core.Point{32, 32}, 50, "inside single cost region"},
		{core.Point{42, 42}, 30, "inside only second cost region"},
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
package pathfinding

import (
	"edd/diagram"
	"testing"
)

func TestArrowConfig(t *testing.T) {
	config := NewArrowConfig()
	
	// Test default arrow type
	conn1 := diagram.Connection{From: 1, To: 2}
	if config.GetArrowType(conn1) != ArrowEnd {
		t.Errorf("Default arrow type should be ArrowEnd")
	}
	
	// Test setting specific arrow type
	config.SetArrowType(1, 2, ArrowBoth)
	if config.GetArrowType(conn1) != ArrowBoth {
		t.Errorf("Arrow type should be ArrowBoth after override")
	}
	
	// Test different connection still uses default
	conn2 := diagram.Connection{From: 2, To: 3}
	if config.GetArrowType(conn2) != ArrowEnd {
		t.Errorf("Different connection should still use default type")
	}
}

func TestArrowConfig_ShouldDrawArrow(t *testing.T) {
	config := NewArrowConfig()
	
	tests := []struct {
		name      string
		from      int
		to        int
		arrowType ArrowType
		wantEnd   bool
		wantStart bool
	}{
		{
			name:      "no arrow",
			from:      1,
			to:        2,
			arrowType: ArrowNone,
			wantEnd:   false,
			wantStart: false,
		},
		{
			name:      "end arrow only",
			from:      1,
			to:        2,
			arrowType: ArrowEnd,
			wantEnd:   true,
			wantStart: false,
		},
		{
			name:      "start arrow only",
			from:      1,
			to:        2,
			arrowType: ArrowStart,
			wantEnd:   false,
			wantStart: true,
		},
		{
			name:      "both arrows",
			from:      1,
			to:        2,
			arrowType: ArrowBoth,
			wantEnd:   true,
			wantStart: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.SetArrowType(tt.from, tt.to, tt.arrowType)
			conn := diagram.Connection{From: tt.from, To: tt.to}
			
			if got := config.ShouldDrawArrowAtEnd(conn); got != tt.wantEnd {
				t.Errorf("ShouldDrawArrowAtEnd() = %v, want %v", got, tt.wantEnd)
			}
			
			if got := config.ShouldDrawArrowAtStart(conn); got != tt.wantStart {
				t.Errorf("ShouldDrawArrowAtStart() = %v, want %v", got, tt.wantStart)
			}
		})
	}
}

func TestApplyArrowConfig(t *testing.T) {
	config := NewArrowConfig()
	config.SetArrowType(1, 2, ArrowBoth)
	config.SetArrowType(2, 3, ArrowNone)
	
	connections := []diagram.Connection{
		{From: 1, To: 2},
		{From: 2, To: 3},
		{From: 3, To: 1}, // Uses default
	}
	
	paths := map[int]diagram.Path{
		0: {Points: []diagram.Point{{0, 0}, {10, 0}}},
		1: {Points: []diagram.Point{{10, 0}, {10, 10}}},
		2: {Points: []diagram.Point{{10, 10}, {0, 0}}},
	}
	
	result := ApplyArrowConfig(connections, paths, config)
	
	if len(result) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(result))
	}
	
	// Check arrow types
	expectedTypes := []ArrowType{ArrowBoth, ArrowNone, ArrowEnd}
	for i, expected := range expectedTypes {
		if result[i].ArrowType != expected {
			t.Errorf("Result %d: ArrowType = %v, want %v", i, result[i].ArrowType, expected)
		}
	}
}
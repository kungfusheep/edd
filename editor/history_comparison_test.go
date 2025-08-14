package editor

import (
	"edd/core"
	"testing"
	"time"
)

func TestHistoryPerformanceComparison(t *testing.T) {
	// Create a test diagram
	diagram := &core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"Node 1", "Line 2", "Line 3"}},
			{ID: 2, Text: []string{"Node 2", "Description"}},
			{ID: 3, Text: []string{"Node 3"}},
		},
		Connections: []core.Connection{
			{From: 1, To: 2, Label: "connects"},
			{From: 2, To: 3, Label: "flows"},
		},
	}
	
	// Test JSON-based history (SimpleHistory)
	jsonHistory := NewSimpleHistory(50)
	start := time.Now()
	for i := 0; i < 1000; i++ {
		jsonHistory.SaveState(diagram)
	}
	jsonTime := time.Since(start)
	
	// Test struct-based history (StructHistory) 
	structHistory := NewStructHistory(50)
	start = time.Now()
	for i := 0; i < 1000; i++ {
		structHistory.SaveState(diagram)
	}
	structTime := time.Since(start)
	
	// Calculate speedup
	speedup := float64(jsonTime) / float64(structTime)
	
	t.Logf("JSON-based:   %v for 1000 saves", jsonTime)
	t.Logf("Struct-based: %v for 1000 saves", structTime)
	t.Logf("Speedup:      %.1fx faster", speedup)
	
	// The struct-based should be significantly faster (at least 5x in practice)
	if speedup < 5 {
		t.Errorf("Expected struct-based to be at least 5x faster, got %.1fx", speedup)
	}
}
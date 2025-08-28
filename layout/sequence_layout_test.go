package layout

import (
	"edd/core"
	"testing"
)

func TestSequenceLayoutBasic(t *testing.T) {
	layout := NewSequenceLayout()
	
	diagram := &core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"User"}},
			{ID: 2, Text: []string{"Server"}},
			{ID: 3, Text: []string{"Database"}},
		},
		Connections: []core.Connection{
			{From: 1, To: 2, Label: "request"},
			{From: 2, To: 3, Label: "query"},
			{From: 3, To: 2, Label: "result"},
			{From: 2, To: 1, Label: "response"},
		},
	}
	
	layout.Layout(diagram)
	
	// Check participants are positioned horizontally
	if diagram.Nodes[0].Y != layout.TopMargin {
		t.Errorf("First participant should be at top margin, got Y=%d", diagram.Nodes[0].Y)
	}
	
	if diagram.Nodes[1].X <= diagram.Nodes[0].X+diagram.Nodes[0].Width {
		t.Error("Second participant should be to the right of first")
	}
	
	if diagram.Nodes[2].X <= diagram.Nodes[1].X+diagram.Nodes[1].Width {
		t.Error("Third participant should be to the right of second")
	}
	
	// Check all participants are at same Y
	for i := 1; i < len(diagram.Nodes); i++ {
		if diagram.Nodes[i].Y != diagram.Nodes[0].Y {
			t.Errorf("All participants should be at same Y level")
		}
	}
	
	// Check connections have position hints
	for i, conn := range diagram.Connections {
		if conn.Hints == nil {
			t.Errorf("Connection %d should have hints", i)
			continue
		}
		if _, ok := conn.Hints["y-position"]; !ok {
			t.Errorf("Connection %d missing y-position hint", i)
		}
		if _, ok := conn.Hints["from-x"]; !ok {
			t.Errorf("Connection %d missing from-x hint", i)
		}
		if _, ok := conn.Hints["to-x"]; !ok {
			t.Errorf("Connection %d missing to-x hint", i)
		}
	}
}

func TestSequenceLayoutWithHints(t *testing.T) {
	layout := NewSequenceLayout()
	
	diagram := &core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"Actor"}, Hints: map[string]string{"node-type": "actor"}},
			{ID: 2, Text: []string{"System"}, Hints: map[string]string{"node-type": "participant"}},
			{ID: 3, Text: []string{"Other"}, Hints: map[string]string{"node-type": "other"}}, // Should be ignored
		},
	}
	
	layout.Layout(diagram)
	
	// Check that only actor and participant are positioned
	if diagram.Nodes[0].X != layout.LeftMargin {
		t.Error("Actor should be positioned")
	}
	if diagram.Nodes[1].X <= diagram.Nodes[0].X {
		t.Error("Participant should be positioned")
	}
}

func TestSequenceLayoutBounds(t *testing.T) {
	layout := NewSequenceLayout()
	
	diagram := &core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"A"}},
			{ID: 2, Text: []string{"B"}},
		},
		Connections: []core.Connection{
			{From: 1, To: 2},
			{From: 2, To: 1},
			{From: 1, To: 2},
		},
	}
	
	layout.Layout(diagram)
	width, height := layout.GetDiagramBounds(diagram)
	
	if width <= 0 {
		t.Error("Width should be positive")
	}
	if height <= 0 {
		t.Error("Height should be positive")
	}
	
	// Height should account for messages
	expectedMinHeight := layout.TopMargin + layout.ParticipantHeight + (3 * layout.MessageSpacing)
	if height < expectedMinHeight {
		t.Errorf("Height should be at least %d, got %d", expectedMinHeight, height)
	}
}
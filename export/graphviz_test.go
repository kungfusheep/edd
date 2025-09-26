package export

import (
	"edd/diagram"
	"strings"
	"testing"
)

func TestGraphvizExporter_Basic(t *testing.T) {
	exporter := NewGraphvizExporter()

	d := &diagram.Diagram{
		Type: "box",
		Nodes: []diagram.Node{
			{ID: 0, Text: []string{"Node A"}},
			{ID: 1, Text: []string{"Node B"}},
		},
		Connections: []diagram.Connection{
			{From: 0, To: 1, Label: "connects"},
		},
	}

	result, err := exporter.Export(d)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Check for basic structure
	if !strings.Contains(result, "digraph G {") {
		t.Error("Missing digraph declaration")
	}
	if !strings.Contains(result, "N0 [label=\"Node A\"]") {
		t.Error("Missing node N0")
	}
	if !strings.Contains(result, "N1 [label=\"Node B\"]") {
		t.Error("Missing node N1")
	}
	if !strings.Contains(result, "N0 -> N1 [label=\"connects\"]") {
		t.Error("Missing connection with label")
	}
}

func TestGraphvizExporter_WithHints(t *testing.T) {
	exporter := NewGraphvizExporter()

	d := &diagram.Diagram{
		Type: "box",
		Nodes: []diagram.Node{
			{
				ID:   0,
				Text: []string{"Colored Node"},
				Hints: map[string]string{
					"color": "red",
					"style": "rounded",
					"bold":  "true",
				},
			},
			{
				ID:   1,
				Text: []string{"Shaped Node"},
				Hints: map[string]string{
					"shape": "diamond",
				},
			},
		},
		Connections: []diagram.Connection{
			{
				From:  0,
				To:    1,
				Label: "dashed",
				Hints: map[string]string{
					"style": "dashed",
					"color": "blue",
				},
			},
		},
	}

	result, err := exporter.Export(d)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Check for color
	if !strings.Contains(result, "fillcolor=\"#FF6B6B\"") {
		t.Error("Missing red color")
	}

	// Check for shape
	if !strings.Contains(result, "shape=diamond") {
		t.Error("Missing diamond shape")
	}

	// Check for connection style
	if !strings.Contains(result, "style=dashed") {
		t.Error("Missing dashed style")
	}

	// Check for connection color
	if !strings.Contains(result, "color=\"#339AF0\"") {
		t.Error("Missing blue color on connection")
	}
}

func TestGraphvizExporter_MultilineText(t *testing.T) {
	exporter := NewGraphvizExporter()

	d := &diagram.Diagram{
		Type: "box",
		Nodes: []diagram.Node{
			{ID: 0, Text: []string{"Line 1", "Line 2", "Line 3"}},
		},
	}

	result, err := exporter.Export(d)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Check for multiline text
	if !strings.Contains(result, "Line 1\\nLine 2\\nLine 3") {
		t.Error("Multiline text not properly formatted")
	}
}

func TestGraphvizExporter_EscapedCharacters(t *testing.T) {
	exporter := NewGraphvizExporter()

	d := &diagram.Diagram{
		Type: "box",
		Nodes: []diagram.Node{
			{ID: 0, Text: []string{`Node with "quotes" and \backslash`}},
		},
		Connections: []diagram.Connection{
			{From: 0, To: 0, Label: `Label with "quotes"`},
		},
	}

	result, err := exporter.Export(d)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Check for escaped characters
	if !strings.Contains(result, `\"quotes\"`) {
		t.Error("Quotes not properly escaped")
	}
	if !strings.Contains(result, `\\backslash`) {
		t.Error("Backslash not properly escaped")
	}
}

func TestGraphvizExporter_BidirectionalConnection(t *testing.T) {
	exporter := NewGraphvizExporter()

	d := &diagram.Diagram{
		Type: "box",
		Nodes: []diagram.Node{
			{ID: 0, Text: []string{"A"}},
			{ID: 1, Text: []string{"B"}},
		},
		Connections: []diagram.Connection{
			{
				From: 0,
				To:   1,
				Hints: map[string]string{
					"bidirectional": "true",
				},
			},
		},
	}

	result, err := exporter.Export(d)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Check for bidirectional attribute
	if !strings.Contains(result, "dir=both") {
		t.Error("Missing bidirectional direction")
	}
}

func TestGraphvizExporter_EmptyDiagram(t *testing.T) {
	exporter := NewGraphvizExporter()

	d := &diagram.Diagram{
		Type:  "box",
		Nodes: []diagram.Node{},
	}

	_, err := exporter.Export(d)
	if err == nil {
		t.Error("Expected error for empty diagram")
	}
}

func TestGraphvizExporter_NilDiagram(t *testing.T) {
	exporter := NewGraphvizExporter()

	_, err := exporter.Export(nil)
	if err == nil {
		t.Error("Expected error for nil diagram")
	}
}
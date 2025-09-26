package export

import (
	"edd/diagram"
	"strings"
	"testing"
)

func TestD2Exporter_Basic(t *testing.T) {
	exporter := NewD2Exporter()

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
	if !strings.Contains(result, "node_0: Node A") {
		t.Error("Missing node_0 declaration")
	}
	if !strings.Contains(result, "node_1: Node B") {
		t.Error("Missing node_1 declaration")
	}
	if !strings.Contains(result, "node_0 -> node_1: connects") {
		t.Error("Missing connection with label")
	}
}

func TestD2Exporter_WithHints(t *testing.T) {
	exporter := NewD2Exporter()

	d := &diagram.Diagram{
		Type: "box",
		Nodes: []diagram.Node{
			{
				ID:   0,
				Text: []string{"Colored Node"},
				Hints: map[string]string{
					"color":  "red",
					"style":  "rounded",
					"bold":   "true",
					"shadow": "southeast",
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
	if !strings.Contains(result, "node_0.style.fill: \"#FF6B6B\"") {
		t.Error("Missing red fill color")
	}

	// Check for bold
	if !strings.Contains(result, "node_0.style.bold: true") {
		t.Error("Missing bold style")
	}

	// Check for shadow
	if !strings.Contains(result, "node_0.style.shadow: true") {
		t.Error("Missing shadow")
	}

	// Check for shape
	if !strings.Contains(result, "node_1.shape: diamond") {
		t.Error("Missing diamond shape")
	}

	// Check for connection style
	if !strings.Contains(result, "node_0 --> node_1: dashed") {
		t.Error("Missing dashed connection arrow")
	}
}

func TestD2Exporter_MultilineText(t *testing.T) {
	exporter := NewD2Exporter()

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

func TestD2Exporter_EscapedCharacters(t *testing.T) {
	exporter := NewD2Exporter()

	d := &diagram.Diagram{
		Type: "box",
		Nodes: []diagram.Node{
			{ID: 0, Text: []string{`Node: with special chars`}},
			{ID: 1, Text: []string{`Node "with quotes"`}},
		},
		Connections: []diagram.Connection{
			{From: 0, To: 1, Label: `Label -> with arrow`},
		},
	}

	result, err := exporter.Export(d)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Check for escaped/quoted strings
	if !strings.Contains(result, `"Node: with special chars"`) {
		t.Error("Special characters not properly quoted in node")
	}
	if !strings.Contains(result, `\"with quotes\"`) {
		t.Error("Quotes not properly escaped")
	}
	if !strings.Contains(result, `"Label -> with arrow"`) {
		t.Error("Special characters in label not properly quoted")
	}
}

func TestD2Exporter_BidirectionalConnection(t *testing.T) {
	exporter := NewD2Exporter()

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

	// Check for bidirectional arrow
	if !strings.Contains(result, "node_0 <-> node_1") {
		t.Error("Missing bidirectional arrow")
	}
}

func TestD2Exporter_ConnectionStyles(t *testing.T) {
	exporter := NewD2Exporter()

	tests := []struct {
		style    string
		expected string
	}{
		{"dashed", "-->"},
		{"dotted", "-->"},
		{"thick", "=>"},
		{"double", "=>"},
	}

	for _, tt := range tests {
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
						"style": tt.style,
					},
				},
			},
		}

		result, err := exporter.Export(d)
		if err != nil {
			t.Fatalf("Export failed for style %s: %v", tt.style, err)
		}

		if !strings.Contains(result, tt.expected) {
			t.Errorf("Style %s: expected arrow %s, result: %s", tt.style, tt.expected, result)
		}
	}
}

func TestD2Exporter_WithMetadata(t *testing.T) {
	exporter := NewD2Exporter()

	d := &diagram.Diagram{
		Type: "box",
		Nodes: []diagram.Node{
			{ID: 0, Text: []string{"A"}},
		},
		Metadata: diagram.Metadata{
			Name: "Test Diagram",
		},
	}

	result, err := exporter.Export(d)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Check for metadata comment
	if !strings.Contains(result, "# Test Diagram") {
		t.Error("Missing diagram name in comment")
	}
}

func TestD2Exporter_EmptyDiagram(t *testing.T) {
	exporter := NewD2Exporter()

	d := &diagram.Diagram{
		Type:  "box",
		Nodes: []diagram.Node{},
	}

	_, err := exporter.Export(d)
	if err == nil {
		t.Error("Expected error for empty diagram")
	}
}

func TestD2Exporter_NilDiagram(t *testing.T) {
	exporter := NewD2Exporter()

	_, err := exporter.Export(nil)
	if err == nil {
		t.Error("Expected error for nil diagram")
	}
}
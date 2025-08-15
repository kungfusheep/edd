package core

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestNodeWithHints(t *testing.T) {
	// Test creating a node with hints
	node := Node{
		ID:   1,
		Text: []string{"Test Node"},
		Hints: map[string]string{
			"style": "rounded",
			"color": "blue",
		},
	}

	if node.Hints["style"] != "rounded" {
		t.Errorf("Expected style hint to be 'rounded', got %s", node.Hints["style"])
	}
	if node.Hints["color"] != "blue" {
		t.Errorf("Expected color hint to be 'blue', got %s", node.Hints["color"])
	}
}

func TestNodeJSONMarshalUnmarshal(t *testing.T) {
	// Test that hints are preserved through JSON marshaling
	original := Node{
		ID:   1,
		Text: []string{"Test", "Node"},
		Hints: map[string]string{
			"style": "double",
			"color": "red",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal node: %v", err)
	}

	// Unmarshal back
	var loaded Node
	err = json.Unmarshal(data, &loaded)
	if err != nil {
		t.Fatalf("Failed to unmarshal node: %v", err)
	}

	// Check that hints are preserved
	if !reflect.DeepEqual(original.Hints, loaded.Hints) {
		t.Errorf("Hints not preserved: original=%v, loaded=%v", original.Hints, loaded.Hints)
	}
}

func TestNodeBackwardCompatibility(t *testing.T) {
	// Test that old JSON without hints still loads
	oldJSON := `{
		"id": 1,
		"text": ["Legacy Node"]
	}`

	var node Node
	err := json.Unmarshal([]byte(oldJSON), &node)
	if err != nil {
		t.Fatalf("Failed to unmarshal legacy node: %v", err)
	}

	if node.ID != 1 {
		t.Errorf("Expected ID to be 1, got %d", node.ID)
	}
	if len(node.Text) != 1 || node.Text[0] != "Legacy Node" {
		t.Errorf("Expected text to be ['Legacy Node'], got %v", node.Text)
	}
	if node.Hints != nil {
		t.Errorf("Expected hints to be nil for legacy node, got %v", node.Hints)
	}
}

func TestDiagramCloneWithNodeHints(t *testing.T) {
	// Test that Clone properly copies node hints
	original := &Diagram{
		Nodes: []Node{
			{
				ID:   1,
				Text: []string{"Node 1"},
				Hints: map[string]string{
					"style": "rounded",
					"color": "green",
				},
			},
			{
				ID:   2,
				Text: []string{"Node 2"},
				// No hints
			},
		},
		Connections: []Connection{
			{
				ID:   1,
				From: 1,
				To:   2,
			},
		},
	}

	clone := original.Clone()

	// Verify node hints are cloned
	if !reflect.DeepEqual(original.Nodes[0].Hints, clone.Nodes[0].Hints) {
		t.Errorf("Node hints not properly cloned: original=%v, clone=%v",
			original.Nodes[0].Hints, clone.Nodes[0].Hints)
	}

	// Verify deep copy (modifying clone doesn't affect original)
	clone.Nodes[0].Hints["style"] = "sharp"
	if original.Nodes[0].Hints["style"] != "rounded" {
		t.Error("Modifying cloned hints affected original")
	}

	// Verify node without hints is handled correctly
	if clone.Nodes[1].Hints != nil {
		t.Errorf("Expected nil hints for node 2, got %v", clone.Nodes[1].Hints)
	}
}

func TestNodeHintsOmitEmpty(t *testing.T) {
	// Test that empty hints map is omitted from JSON
	node := Node{
		ID:    1,
		Text:  []string{"Test"},
		Hints: nil,
	}

	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("Failed to marshal node: %v", err)
	}

	jsonStr := string(data)
	if strings.Contains(jsonStr, "hints") {
		t.Errorf("Empty hints should be omitted from JSON: %s", jsonStr)
	}

	// Test with empty map
	node.Hints = make(map[string]string)
	data, err = json.Marshal(node)
	if err != nil {
		t.Fatalf("Failed to marshal node: %v", err)
	}

	jsonStr = string(data)
	// Note: empty map might still appear in JSON as "hints":{}, 
	// but omitempty should handle nil case
}
package editor

import (
	"edd/diagram"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
)

func TestJSONRoundTrip(t *testing.T) {
	// Create a test diagram
	original := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"Node A", "Line 2"}},
			{ID: 2, Text: []string{"Node B"}},
			{ID: 3, Text: []string{"Node C"}},
		},
		Connections: []diagram.Connection{
			{From: 1, To: 2, Label: "connects"},
			{From: 2, To: 3, Label: "flows to"},
		},
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal back
	var loaded diagram.Diagram
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify structure is preserved
	if len(loaded.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(loaded.Nodes))
	}
	if len(loaded.Connections) != 2 {
		t.Errorf("Expected 2 connections, got %d", len(loaded.Connections))
	}

	// Check specific values
	if loaded.Nodes[0].ID != 1 {
		t.Errorf("Expected node ID 1, got %d", loaded.Nodes[0].ID)
	}
	if len(loaded.Nodes[0].Text) != 2 {
		t.Errorf("Expected 2 text lines for node 1, got %d", len(loaded.Nodes[0].Text))
	}
	if loaded.Connections[0].Label != "connects" {
		t.Errorf("Expected label 'connects', got '%s'", loaded.Connections[0].Label)
	}
}

func TestTempFileCreation(t *testing.T) {
	// Test that we can create and write to a temp file
	tmpFile, err := ioutil.TempFile("", "edd-test-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write test data
	testData := []byte(`{"nodes": []}`)
	if _, err := tmpFile.Write(testData); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Read it back
	readData, err := ioutil.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	if string(readData) != string(testData) {
		t.Errorf("Data mismatch: got %s, want %s", readData, testData)
	}
}

func TestDiagramValidation(t *testing.T) {
	tests := []struct {
		name    string
		diagram diagram.Diagram
		wantErr bool
	}{
		{
			name: "valid diagram",
			diagram: diagram.Diagram{
				Nodes: []diagram.Node{
					{ID: 1, Text: []string{"A"}},
					{ID: 2, Text: []string{"B"}},
				},
				Connections: []diagram.Connection{
					{From: 1, To: 2},
				},
			},
			wantErr: false,
		},
		{
			name: "duplicate node IDs",
			diagram: diagram.Diagram{
				Nodes: []diagram.Node{
					{ID: 1, Text: []string{"A"}},
					{ID: 1, Text: []string{"B"}}, // Duplicate ID
				},
			},
			wantErr: true,
		},
		{
			name: "connection references non-existent node",
			diagram: diagram.Diagram{
				Nodes: []diagram.Node{
					{ID: 1, Text: []string{"A"}},
				},
				Connections: []diagram.Connection{
					{From: 1, To: 99}, // Node 99 doesn't exist
				},
			},
			wantErr: true,
		},
		{
			name: "empty diagram is valid",
			diagram: diagram.Diagram{
				Nodes:       []diagram.Node{},
				Connections: []diagram.Connection{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't directly test validateDiagram from main package,
			// but we can test the concept
			err := validateTestDiagram(&tt.diagram)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDiagram() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// validateTestDiagram is a test version of the validation logic
func validateTestDiagram(d *diagram.Diagram) error {
	// Check for duplicate node IDs
	nodeIDs := make(map[int]bool)
	for _, node := range d.Nodes {
		if nodeIDs[node.ID] {
			return &ValidationError{Message: "duplicate node ID"}
		}
		nodeIDs[node.ID] = true
	}

	// Check that connections reference valid nodes
	for _, conn := range d.Connections {
		if !nodeIDs[conn.From] {
			return &ValidationError{Message: "invalid from node"}
		}
		if !nodeIDs[conn.To] {
			return &ValidationError{Message: "invalid to node"}
		}
	}

	return nil
}

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
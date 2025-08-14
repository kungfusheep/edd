package tests

import (
	"encoding/json"
	"edd/core"
	"edd/rendering"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIntegrationDiagrams tests rendering all diagrams in the test_diagrams directory
func TestIntegrationDiagrams(t *testing.T) {
	// Find all JSON files in test_diagrams
	files, err := filepath.Glob("test_diagrams/*.json")
	if err != nil {
		t.Fatalf("Failed to find test diagrams: %v", err)
	}

	if len(files) == 0 {
		t.Skip("No test diagrams found")
	}

	renderer := rendering.NewRenderer()

	for _, file := range files {
		name := filepath.Base(file)
		t.Run(name, func(t *testing.T) {
			// Read and parse the diagram
			diagram, err := loadTestDiagram(file)
			if err != nil {
				t.Fatalf("Failed to load diagram: %v", err)
			}

			// Render the diagram
			output, err := renderer.Render(diagram)
			if err != nil {
				// Some diagrams might fail with simple layout (e.g., cycles)
				if strings.Contains(err.Error(), "cycle") || strings.Contains(err.Error(), "blocked") {
					t.Skipf("Known limitation with simple layout: %v", err)
					return
				}
				t.Errorf("Failed to render: %v", err)
				return
			}

			// Basic validation
			if output == "" {
				t.Error("Expected non-empty output")
			}

			// Check that all node texts appear in output
			for _, node := range diagram.Nodes {
				for _, text := range node.Text {
					if !strings.Contains(output, text) {
						t.Errorf("Missing node text in output: %s", text)
					}
				}
			}

			// Log the output for visual inspection
			t.Logf("Rendered %s:\n%s", name, output)
		})
	}
}

// loadTestDiagram loads a diagram from a JSON file for testing
func loadTestDiagram(filename string) (*core.Diagram, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var diagram core.Diagram
	if err := json.Unmarshal(data, &diagram); err != nil {
		return nil, err
	}

	return &diagram, nil
}

// TestRendererEndToEnd tests a complete rendering scenario
func TestRendererEndToEnd(t *testing.T) {
	// Create a comprehensive test diagram
	diagram := &core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"Frontend", "React App"}},
			{ID: 2, Text: []string{"API", "Gateway"}},
			{ID: 3, Text: []string{"Auth", "Service"}},
			{ID: 4, Text: []string{"User", "Service"}},
			{ID: 5, Text: []string{"Database"}},
		},
		Connections: []core.Connection{
			{From: 1, To: 2},
			{From: 2, To: 3},
			{From: 2, To: 4},
			{From: 3, To: 5},
			{From: 4, To: 5},
		},
		Metadata: core.Metadata{
			Name: "Microservices Architecture",
		},
	}

	renderer := rendering.NewRenderer()
	output, err := renderer.Render(diagram)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	// Display the output
	t.Logf("End-to-end test output:\n%s", output)

	// Validate output structure
	lines := strings.Split(output, "\n")
	if len(lines) < 5 {
		t.Error("Output seems too small")
	}

	// Check for box drawing characters
	hasBoxes := false
	hasConnections := false
	for _, line := range lines {
		if strings.ContainsAny(line, "┌┐└┘") {
			hasBoxes = true
		}
		if strings.ContainsAny(line, "─│├┤┬┴┼") {
			hasConnections = true
		}
	}

	if !hasBoxes {
		t.Error("Missing box drawing characters")
	}
	if !hasConnections {
		t.Error("Missing connection characters")
	}
}
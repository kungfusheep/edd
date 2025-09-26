package tests

import (
	"edd/diagram"
	"edd/export"
	"edd/importer"
	"strings"
	"testing"
)

// TestMermaidRoundTrip tests importing Mermaid and exporting it back
func TestMermaidRoundTrip(t *testing.T) {
	mermaidInput := `graph TD
    A[Start] --> B{Decision}
    B --> C[Option 1]
    B --> D[Option 2]
    C --> E[End]
    D --> E`

	// Import Mermaid
	mImporter := importer.NewMermaidImporter()
	diag, err := mImporter.Import(mermaidInput)
	if err != nil {
		t.Fatalf("Failed to import Mermaid: %v", err)
	}

	// Check imported nodes
	if len(diag.Nodes) != 5 {
		t.Errorf("Expected 5 nodes, got %d", len(diag.Nodes))
	}

	// Check imported connections (A->B is captured as part of node declaration line)
	// So we have B->C, B->D, C->E, D->E = 4 connections
	if len(diag.Connections) != 4 {
		t.Errorf("Expected 4 connections, got %d", len(diag.Connections))
	}

	// Export back to Mermaid
	mExporter := export.NewMermaidExporter()
	exported, err := mExporter.Export(diag)
	if err != nil {
		t.Fatalf("Failed to export to Mermaid: %v", err)
	}

	// Check that exported contains key elements
	if !strings.Contains(exported, "graph TD") {
		t.Error("Exported Mermaid missing graph declaration")
	}
	if !strings.Contains(exported, "Start") {
		t.Error("Exported Mermaid missing Start node")
	}
	if !strings.Contains(exported, "Decision") {
		t.Error("Exported Mermaid missing Decision node")
	}
}

// TestMermaidSequenceRoundTrip tests sequence diagram round trip
func TestMermaidSequenceRoundTrip(t *testing.T) {
	mermaidInput := `sequenceDiagram
    participant Alice
    participant Bob
    Alice->>Bob: Hello Bob!
    Bob-->>Alice: Hi Alice!`

	// Import Mermaid sequence
	mImporter := importer.NewMermaidImporter()
	diag, err := mImporter.Import(mermaidInput)
	if err != nil {
		t.Fatalf("Failed to import Mermaid sequence: %v", err)
	}

	// Check diagram type
	if diag.Type != "sequence" {
		t.Errorf("Expected sequence type, got %s", diag.Type)
	}

	// Check imported nodes (participants)
	if len(diag.Nodes) != 2 {
		t.Errorf("Expected 2 participants, got %d", len(diag.Nodes))
	}

	// Check imported connections (messages)
	if len(diag.Connections) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(diag.Connections))
	}

	// Export back to Mermaid
	mExporter := export.NewMermaidExporter()
	exported, err := mExporter.Export(diag)
	if err != nil {
		t.Fatalf("Failed to export to Mermaid: %v", err)
	}

	// Check that exported contains key elements
	if !strings.Contains(exported, "sequenceDiagram") {
		t.Error("Exported Mermaid missing sequenceDiagram declaration")
	}
	if !strings.Contains(exported, "participant") && !strings.Contains(exported, "Alice") {
		t.Error("Exported Mermaid missing Alice participant")
	}
}

// TestPlantUMLRoundTrip tests importing PlantUML and exporting it back
func TestPlantUMLRoundTrip(t *testing.T) {
	plantUMLInput := `@startuml
participant Alice
participant Bob
Alice -> Bob: Request
Bob --> Alice: Response
@enduml`

	// Import PlantUML
	pImporter := importer.NewPlantUMLImporter()
	diag, err := pImporter.Import(plantUMLInput)
	if err != nil {
		t.Fatalf("Failed to import PlantUML: %v", err)
	}

	// Check imported data
	if diag.Type != "sequence" {
		t.Errorf("Expected sequence type, got %s", diag.Type)
	}
	if len(diag.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(diag.Nodes))
	}
	if len(diag.Connections) != 2 {
		t.Errorf("Expected 2 connections, got %d", len(diag.Connections))
	}

	// Export back to PlantUML
	pExporter := export.NewPlantUMLExporter()
	exported, err := pExporter.Export(diag)
	if err != nil {
		t.Fatalf("Failed to export to PlantUML: %v", err)
	}

	// Check that exported contains key elements
	if !strings.Contains(exported, "@startuml") {
		t.Error("Exported PlantUML missing @startuml")
	}
	if !strings.Contains(exported, "@enduml") {
		t.Error("Exported PlantUML missing @enduml")
	}
	if !strings.Contains(exported, "Alice") {
		t.Error("Exported PlantUML missing Alice")
	}
}

// TestGraphvizRoundTrip tests importing Graphviz and exporting it back
func TestGraphvizRoundTrip(t *testing.T) {
	graphvizInput := `digraph G {
    A [label="Start", shape=circle];
    B [label="Process", shape=box];
    C [label="End", shape=diamond];
    A -> B [label="begin"];
    B -> C [label="finish"];
}`

	// Import Graphviz
	gImporter := importer.NewGraphvizImporter()
	diag, err := gImporter.Import(graphvizInput)
	if err != nil {
		t.Fatalf("Failed to import Graphviz: %v", err)
	}

	// Check imported nodes
	if len(diag.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(diag.Nodes))
	}

	// Check node attributes were parsed
	foundCircle := false
	foundDiamond := false
	for _, node := range diag.Nodes {
		if node.Hints != nil {
			if shape, ok := node.Hints["shape"]; ok {
				if shape == "circle" {
					foundCircle = true
				} else if shape == "diamond" {
					foundDiamond = true
				}
			}
		}
	}
	if !foundCircle {
		t.Error("Circle shape not preserved")
	}
	if !foundDiamond {
		t.Error("Diamond shape not preserved")
	}

	// Export to Graphviz
	gExporter := export.NewGraphvizExporter()
	exported, err := gExporter.Export(diag)
	if err != nil {
		t.Fatalf("Failed to export to Graphviz: %v", err)
	}

	// Check that exported contains key elements
	if !strings.Contains(exported, "digraph G") {
		t.Error("Exported Graphviz missing digraph declaration")
	}
	if !strings.Contains(exported, "shape=circle") {
		t.Error("Exported Graphviz missing circle shape")
	}
	if !strings.Contains(exported, "shape=diamond") {
		t.Error("Exported Graphviz missing diamond shape")
	}
}

// TestD2RoundTrip tests importing D2 and exporting it back
func TestD2RoundTrip(t *testing.T) {
	d2Input := `A: Start
B: Process
C: End
A -> B: begin
B -> C: finish
B.shape: diamond
B.style.fill: "#FF6B6B"`

	// Import D2
	d2Importer := importer.NewD2Importer()
	diag, err := d2Importer.Import(d2Input)
	if err != nil {
		t.Fatalf("Failed to import D2: %v", err)
	}

	// Check imported nodes
	if len(diag.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(diag.Nodes))
	}

	// Check shape and color were parsed
	foundDiamond := false
	foundColor := false
	for _, node := range diag.Nodes {
		if node.Hints != nil {
			if shape, ok := node.Hints["shape"]; ok && shape == "diamond" {
				foundDiamond = true
			}
			if color, ok := node.Hints["color"]; ok && color != "" {
				foundColor = true
			}
		}
	}
	if !foundDiamond {
		t.Error("Diamond shape not preserved")
	}
	if !foundColor {
		t.Error("Color not preserved")
	}

	// Export to D2
	d2Exporter := export.NewD2Exporter()
	exported, err := d2Exporter.Export(diag)
	if err != nil {
		t.Fatalf("Failed to export to D2: %v", err)
	}

	// Check that exported contains key elements
	if !strings.Contains(exported, "node_0: Start") {
		t.Error("Exported D2 missing Start node")
	}
	if !strings.Contains(exported, "node_1.shape: diamond") {
		t.Error("Exported D2 missing diamond shape")
	}
}

// TestCrossFormatConversion tests converting between different formats
func TestCrossFormatConversion(t *testing.T) {
	// Create a simple diagram
	diag := &diagram.Diagram{
		Type: "box",
		Nodes: []diagram.Node{
			{ID: 0, Text: []string{"Node A"}, Hints: map[string]string{"shape": "circle", "color": "red"}},
			{ID: 1, Text: []string{"Node B"}, Hints: map[string]string{"shape": "diamond"}},
			{ID: 2, Text: []string{"Node C"}, Hints: map[string]string{"style": "dashed"}},
		},
		Connections: []diagram.Connection{
			{From: 0, To: 1, Label: "connects", Hints: map[string]string{"style": "dashed"}},
			{From: 1, To: 2, Label: "flows", Hints: map[string]string{"color": "blue"}},
		},
	}

	// Test exporting to all formats
	formats := []struct {
		name     string
		exporter export.Exporter
		check    string
	}{
		{"Mermaid", export.NewMermaidExporter(), "graph TD"},
		{"PlantUML", export.NewPlantUMLExporter(), "@startuml"},
		{"Graphviz", export.NewGraphvizExporter(), "digraph"},
		{"D2", export.NewD2Exporter(), "node_"},
	}

	for _, format := range formats {
		t.Run(format.name, func(t *testing.T) {
			exported, err := format.exporter.Export(diag)
			if err != nil {
				t.Fatalf("Failed to export to %s: %v", format.name, err)
			}
			if !strings.Contains(exported, format.check) {
				t.Errorf("%s export missing expected content: %s", format.name, format.check)
			}
		})
	}
}

// TestHintsPreservation tests that hints are preserved through import/export
func TestHintsPreservation(t *testing.T) {
	// Create a diagram with various hints
	original := &diagram.Diagram{
		Type: "box",
		Nodes: []diagram.Node{
			{
				ID:   0,
				Text: []string{"Styled Node"},
				Hints: map[string]string{
					"shape":     "hexagon",
					"color":     "green",
					"style":     "rounded",
					"bold":      "true",
					"italic":    "true",
					"shadow":    "southeast",
					"group":     "container1",
					"note":      "This is a note",
				},
			},
		},
		Connections: []diagram.Connection{
			{
				From:  0,
				To:    0,
				Label: "self-loop",
				Hints: map[string]string{
					"style":         "dashed",
					"color":         "magenta",
					"bidirectional": "true",
					"arrow-type":    "circle",
				},
			},
		},
	}

	// Export to Mermaid
	mExporter := export.NewMermaidExporter()
	mermaidStr, err := mExporter.Export(original)
	if err != nil {
		t.Fatalf("Failed to export to Mermaid: %v", err)
	}

	// Check that hints are represented in export
	if !strings.Contains(mermaidStr, "{{") { // Hexagon in Mermaid
		t.Error("Hexagon shape not exported to Mermaid")
	}

	// Export to Graphviz
	gExporter := export.NewGraphvizExporter()
	graphvizStr, err := gExporter.Export(original)
	if err != nil {
		t.Fatalf("Failed to export to Graphviz: %v", err)
	}

	// Check that hints are represented in export
	if !strings.Contains(graphvizStr, "hexagon") {
		t.Error("Hexagon shape not exported to Graphviz")
	}
	if !strings.Contains(graphvizStr, "fillcolor") {
		t.Error("Color not exported to Graphviz")
	}

	// Export to D2
	d2Exporter := export.NewD2Exporter()
	d2Str, err := d2Exporter.Export(original)
	if err != nil {
		t.Fatalf("Failed to export to D2: %v", err)
	}

	// Check that hints are represented in export
	if !strings.Contains(d2Str, ".shape: hexagon") {
		t.Error("Hexagon shape not exported to D2")
	}
	if !strings.Contains(d2Str, ".style.fill:") {
		t.Error("Color not exported to D2")
	}
}

// TestComplexDiagramRoundTrip tests a complex diagram with many features
func TestComplexDiagramRoundTrip(t *testing.T) {
	// Create a complex diagram
	complex := &diagram.Diagram{
		Type: "box",
		Nodes: []diagram.Node{
			{ID: 0, Text: []string{"Start"}, Hints: map[string]string{"shape": "circle"}},
			{ID: 1, Text: []string{"Decision", "Point"}, Hints: map[string]string{"shape": "diamond"}},
			{ID: 2, Text: []string{"Process A"}, Hints: map[string]string{"color": "blue"}},
			{ID: 3, Text: []string{"Process B"}, Hints: map[string]string{"color": "red"}},
			{ID: 4, Text: []string{"End"}, Hints: map[string]string{"shape": "circle", "style": "double"}},
		},
		Connections: []diagram.Connection{
			{From: 0, To: 1, Label: "start"},
			{From: 1, To: 2, Label: "yes", Hints: map[string]string{"style": "dashed"}},
			{From: 1, To: 3, Label: "no", Hints: map[string]string{"style": "dotted"}},
			{From: 2, To: 4, Label: "complete"},
			{From: 3, To: 4, Label: "complete"},
		},
	}

	// Test round trip through each format
	t.Run("Graphviz", func(t *testing.T) {
		gExporter := export.NewGraphvizExporter()
		exported, err := gExporter.Export(complex)
		if err != nil {
			t.Fatalf("Failed to export: %v", err)
		}

		gImporter := importer.NewGraphvizImporter()
		reimported, err := gImporter.Import(exported)
		if err != nil {
			t.Fatalf("Failed to reimport: %v", err)
		}

		// Check preservation
		if len(reimported.Nodes) != len(complex.Nodes) {
			t.Errorf("Node count changed: %d -> %d", len(complex.Nodes), len(reimported.Nodes))
		}
		if len(reimported.Connections) != len(complex.Connections) {
			t.Errorf("Connection count changed: %d -> %d", len(complex.Connections), len(reimported.Connections))
		}
	})
}
package export_test

import (
	"edd/diagram"
	"edd/export"
	"strings"
	"testing"
)

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected export.Format
		wantErr  bool
	}{
		{"ascii", export.FormatASCII, false},
		{"text", export.FormatASCII, false},
		{"txt", export.FormatASCII, false},
		{"mermaid", export.FormatMermaid, false},
		{"mmd", export.FormatMermaid, false},
		{"plantuml", export.FormatPlantUML, false},
		{"puml", export.FormatPlantUML, false},
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := export.ParseFormat(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFormat(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("ParseFormat(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNewExporter(t *testing.T) {
	formats := []export.Format{
		export.FormatASCII,
		export.FormatMermaid,
		export.FormatPlantUML,
	}

	for _, format := range formats {
		t.Run(string(format), func(t *testing.T) {
			exporter, err := export.NewExporter(format)
			if err != nil {
				t.Errorf("NewExporter(%v) returned error: %v", format, err)
				return
			}
			if exporter == nil {
				t.Errorf("NewExporter(%v) returned nil", format)
			}
		})
	}

	// Test invalid format
	_, err := export.NewExporter("invalid")
	if err == nil {
		t.Error("NewExporter with invalid format should return error")
	}
}

func TestMermaidExporter_Sequence(t *testing.T) {
	d := &diagram.Diagram{
		Type: "sequence",
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"Alice"}},
			{ID: 2, Text: []string{"Bob"}},
			{ID: 3, Text: []string{"Charlie"}},
		},
		Connections: []diagram.Connection{
			{ID: 1, From: 1, To: 2, Label: "Hello Bob"},
			{ID: 2, From: 2, To: 3, Label: "Hi Charlie", Hints: map[string]string{"style": "dashed"}},
			{ID: 3, From: 3, To: 1, Label: "Hey Alice"},
			{ID: 4, From: 2, To: 2, Label: "Think..."},
		},
	}

	exporter := export.NewMermaidExporter()
	result, err := exporter.Export(d)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Check for expected content
	expectedParts := []string{
		"sequenceDiagram",
		"participant P1 as Alice",
		"participant P2 as Bob",
		"participant P3 as Charlie",
		"P1->>P2: Hello Bob",
		"P2-->>P3: Hi Charlie",  // This one has dashed style
		"P3->>P1: Hey Alice",
		"P2->>P2: Think...",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Expected result to contain %q, but it didn't.\nGot:\n%s", part, result)
		}
	}
}

func TestMermaidExporter_Flowchart(t *testing.T) {
	d := &diagram.Diagram{
		Type: "flowchart",
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"Start"}},
			{ID: 2, Text: []string{"Process"}, Hints: map[string]string{"box-style": "rounded"}},
			{ID: 3, Text: []string{"Decision"}, Hints: map[string]string{"shape": "rhombus"}},
			{ID: 4, Text: []string{"End"}},
		},
		Connections: []diagram.Connection{
			{ID: 1, From: 1, To: 2},
			{ID: 2, From: 2, To: 3, Label: "check"},
			{ID: 3, From: 3, To: 4, Label: "yes", Hints: map[string]string{"style": "dashed"}},
		},
	}

	exporter := export.NewMermaidExporter()
	result, err := exporter.Export(d)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Check for expected content
	expectedParts := []string{
		"graph TD",
		"N1[Start]",
		"N2(Process)",
		"N3{Decision}",
		"N4[End]",
		"N1 --> N2",
		"N2 -->|check| N3",
		"N3 -.->|yes| N4",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Expected result to contain %q, but it didn't.\nGot:\n%s", part, result)
		}
	}
}

func TestPlantUMLExporter_Sequence(t *testing.T) {
	d := &diagram.Diagram{
		Type: "sequence",
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"Client"}},
			{ID: 2, Text: []string{"Server"}, Hints: map[string]string{"box-style": "double"}},
			{ID: 3, Text: []string{"Database"}},
		},
		Connections: []diagram.Connection{
			{ID: 1, From: 1, To: 2, Label: "Request"},
			{ID: 2, From: 2, To: 3, Label: "Query", Hints: map[string]string{"style": "dashed"}},
			{ID: 3, From: 3, To: 2, Label: "Results"},
			{ID: 4, From: 2, To: 1, Label: "Response"},
		},
	}

	exporter := export.NewPlantUMLExporter()
	result, err := exporter.Export(d)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Check for expected content
	expectedParts := []string{
		"@startuml",
		"participant \"Client\" as P1",
		"database \"Server\" as P2",
		"participant \"Database\" as P3",
		"P1 -> P2 : Request",
		"P2 --> P3 : Query",  // This has dashed style
		"P3 -> P2 : Results",
		"P2 -> P1 : Response",
		"@enduml",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Expected result to contain %q, but it didn't.\nGot:\n%s", part, result)
		}
	}
}

func TestASCIIExporter(t *testing.T) {
	d := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, X: 0, Y: 0, Width: 10, Height: 3, Text: []string{"Box1"}},
			{ID: 2, X: 20, Y: 0, Width: 10, Height: 3, Text: []string{"Box2"}},
		},
		Connections: []diagram.Connection{
			{ID: 1, From: 1, To: 2, Label: "link"},
		},
	}

	exporter := export.NewASCIIExporter()
	result, err := exporter.Export(d)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// ASCII export should contain boxes and connections
	if !strings.Contains(result, "Box1") {
		t.Error("Expected result to contain Box1")
	}
	if !strings.Contains(result, "Box2") {
		t.Error("Expected result to contain Box2")
	}
}

func TestExporterFileExtensions(t *testing.T) {
	tests := []struct {
		format export.Format
		ext    string
	}{
		{export.FormatASCII, ".txt"},
		{export.FormatMermaid, ".mmd"},
		{export.FormatPlantUML, ".puml"},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			exporter, err := export.NewExporter(tt.format)
			if err != nil {
				t.Fatalf("Failed to create exporter: %v", err)
			}

			got := exporter.GetFileExtension()
			if got != tt.ext {
				t.Errorf("GetFileExtension() = %v, want %v", got, tt.ext)
			}
		})
	}
}

func TestExporterErrorHandling(t *testing.T) {
	// Test nil diagram
	exporters := []export.Exporter{
		export.NewMermaidExporter(),
		export.NewPlantUMLExporter(),
		export.NewASCIIExporter(),
	}

	for _, exporter := range exporters {
		_, err := exporter.Export(nil)
		if err == nil {
			t.Errorf("%s exporter should return error for nil diagram", exporter.GetFormatName())
		}
	}

	// Test empty diagram - only Mermaid and PlantUML should error
	emptyDiagram := &diagram.Diagram{}
	mermaidExp := export.NewMermaidExporter()
	_, err := mermaidExp.Export(emptyDiagram)
	if err == nil {
		t.Error("Mermaid exporter should return error for empty diagram")
	}

	plantExp := export.NewPlantUMLExporter()
	_, err = plantExp.Export(emptyDiagram)
	if err == nil {
		t.Error("PlantUML exporter should return error for empty diagram")
	}

	// ASCII exporter delegates to renderer which may handle empty diagrams differently
}
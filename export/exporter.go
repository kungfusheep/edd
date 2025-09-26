// Package export provides functionality to export diagrams to various text-based formats
package export

import (
	"edd/diagram"
	"fmt"
)

// Format represents an export format
type Format string

const (
	// FormatASCII exports to ASCII/Unicode art (default edd format)
	FormatASCII Format = "ascii"
	// FormatMermaid exports to Mermaid diagram syntax
	FormatMermaid Format = "mermaid"
	// FormatPlantUML exports to PlantUML syntax
	FormatPlantUML Format = "plantuml"
	// FormatJSON exports to JSON (edd native format)
	FormatJSON Format = "json"
	// FormatGraphviz exports to Graphviz DOT syntax
	FormatGraphviz Format = "graphviz"
	// FormatD2 exports to D2 syntax
	FormatD2 Format = "d2"
)

// Exporter interface for different export formats
type Exporter interface {
	// Export converts a diagram to the target format
	Export(d *diagram.Diagram) (string, error)
	// GetFileExtension returns the recommended file extension for this format
	GetFileExtension() string
	// GetFormatName returns a human-readable name for this format
	GetFormatName() string
}

// NewExporter creates an exporter for the specified format
func NewExporter(format Format) (Exporter, error) {
	switch format {
	case FormatASCII:
		return NewASCIIExporter(), nil
	case FormatMermaid:
		return NewMermaidExporter(), nil
	case FormatPlantUML:
		return NewPlantUMLExporter(), nil
	case FormatJSON:
		return NewJSONExporter(), nil
	case FormatGraphviz:
		return NewGraphvizExporter(), nil
	case FormatD2:
		return NewD2Exporter(), nil
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// ParseFormat converts a string to a Format
func ParseFormat(s string) (Format, error) {
	switch s {
	case "ascii", "text", "txt":
		return FormatASCII, nil
	case "mermaid", "mmd":
		return FormatMermaid, nil
	case "plantuml", "puml":
		return FormatPlantUML, nil
	case "json":
		return FormatJSON, nil
	case "graphviz", "dot", "gv":
		return FormatGraphviz, nil
	case "d2":
		return FormatD2, nil
	default:
		return "", fmt.Errorf("unknown format: %s", s)
	}
}

// GetAvailableFormats returns a list of all available export formats
func GetAvailableFormats() []Format {
	return []Format{
		FormatASCII,
		FormatMermaid,
		FormatPlantUML,
		FormatJSON,
		FormatGraphviz,
		FormatD2,
	}
}

// GetFormatDescriptions returns human-readable descriptions of all formats
func GetFormatDescriptions() map[Format]string {
	return map[Format]string{
		FormatASCII:    "ASCII/Unicode art (edd native format)",
		FormatMermaid:  "Mermaid diagram syntax (for Markdown)",
		FormatPlantUML: "PlantUML diagram syntax",
		FormatJSON:     "JSON (edd data format)",
		FormatGraphviz: "Graphviz DOT syntax",
		FormatD2:       "D2 diagram syntax",
	}
}
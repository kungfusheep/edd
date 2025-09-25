package export

import (
	"edd/diagram"
	"encoding/json"
)

// JSONExporter exports diagrams to JSON format
type JSONExporter struct{}

// NewJSONExporter creates a new JSON exporter
func NewJSONExporter() *JSONExporter {
	return &JSONExporter{}
}

// Export converts a diagram to JSON
func (e *JSONExporter) Export(d *diagram.Diagram) (string, error) {
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetFileExtension returns the file extension for JSON
func (e *JSONExporter) GetFileExtension() string {
	return ".json"
}

// GetFormatName returns the format name
func (e *JSONExporter) GetFormatName() string {
	return "JSON"
}
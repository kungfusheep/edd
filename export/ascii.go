package export

import (
	"edd/diagram"
	"edd/render"
	"fmt"
)

// ASCIIExporter exports diagrams to ASCII/Unicode art format
type ASCIIExporter struct {
	renderer *render.Renderer
}

// NewASCIIExporter creates a new ASCII exporter
func NewASCIIExporter() *ASCIIExporter {
	return &ASCIIExporter{
		renderer: render.NewRenderer(),
	}
}

// Export converts the diagram to ASCII/Unicode art
func (e *ASCIIExporter) Export(d *diagram.Diagram) (string, error) {
	if d == nil {
		return "", fmt.Errorf("diagram is nil")
	}

	// Use the existing renderer
	output, err := e.renderer.Render(d)
	if err != nil {
		return "", fmt.Errorf("failed to render diagram: %w", err)
	}

	return output, nil
}

// GetFileExtension returns the recommended file extension
func (e *ASCIIExporter) GetFileExtension() string {
	return ".txt"
}

// GetFormatName returns the format name
func (e *ASCIIExporter) GetFormatName() string {
	return "ASCII/Unicode Art"
}
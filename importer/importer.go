package importer

import (
	"edd/diagram"
	"fmt"
	"strings"
)

// Importer interface defines methods for importing diagrams from various formats
type Importer interface {
	// CanImport checks if the given content can be imported by this importer
	CanImport(content string) bool

	// Import converts the input content into an edd diagram
	Import(content string) (*diagram.Diagram, error)

	// GetFormatName returns the human-readable name of the format
	GetFormatName() string

	// GetFileExtensions returns common file extensions for this format
	GetFileExtensions() []string
}

// ImporterRegistry manages available importers
type ImporterRegistry struct {
	importers []Importer
}

// NewImporterRegistry creates a new importer registry
func NewImporterRegistry() *ImporterRegistry {
	return &ImporterRegistry{
		importers: []Importer{
			NewMermaidImporter(),
			NewPlantUMLImporter(),
			NewGraphvizImporter(),
			NewD2Importer(),
		},
	}
}

// Register adds a new importer to the registry
func (r *ImporterRegistry) Register(importer Importer) {
	r.importers = append(r.importers, importer)
}

// DetectFormat attempts to detect the format of the given content
func (r *ImporterRegistry) DetectFormat(content string) (Importer, error) {
	for _, imp := range r.importers {
		if imp.CanImport(content) {
			return imp, nil
		}
	}
	return nil, fmt.Errorf("unable to detect format")
}

// Import attempts to import content using auto-detection
func (r *ImporterRegistry) Import(content string) (*diagram.Diagram, error) {
	importer, err := r.DetectFormat(content)
	if err != nil {
		return nil, err
	}
	return importer.Import(content)
}

// ImportWithFormat imports content using a specific format
func (r *ImporterRegistry) ImportWithFormat(content, format string) (*diagram.Diagram, error) {
	format = strings.ToLower(format)

	for _, imp := range r.importers {
		if strings.ToLower(imp.GetFormatName()) == format {
			return imp.Import(content)
		}
	}

	return nil, fmt.Errorf("unknown format: %s", format)
}

// GetAvailableFormats returns a list of available import formats
func (r *ImporterRegistry) GetAvailableFormats() []string {
	formats := make([]string, len(r.importers))
	for i, imp := range r.importers {
		formats[i] = imp.GetFormatName()
	}
	return formats
}
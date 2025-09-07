package render

import (
	"edd/diagram"
	"fmt"
)

// RendererRegistry manages diagram renderers by type.
type RendererRegistry struct {
	renderers []diagram.DiagramRenderer
	fallback  diagram.DiagramRenderer
}

// NewRendererRegistry creates a new renderer registry.
func NewRendererRegistry() *RendererRegistry {
	return &RendererRegistry{
		renderers: make([]diagram.DiagramRenderer, 0),
	}
}

// Register adds a renderer to the registry.
func (r *RendererRegistry) Register(renderer diagram.DiagramRenderer) {
	r.renderers = append(r.renderers, renderer)
}

// SetFallback sets the default renderer to use when no specific renderer matches.
func (r *RendererRegistry) SetFallback(renderer diagram.DiagramRenderer) {
	r.fallback = renderer
}

// GetRenderer returns the appropriate renderer for the given diagram type.
func (r *RendererRegistry) GetRenderer(diagramType diagram.DiagramType) (diagram.DiagramRenderer, error) {
	// Try to find a specific renderer
	for _, renderer := range r.renderers {
		if renderer.CanRender(diagramType) {
			return renderer, nil
		}
	}
	
	// Use fallback if available
	if r.fallback != nil {
		return r.fallback, nil
	}
	
	return nil, fmt.Errorf("no renderer available for diagram type: %s", diagramType)
}

// Render renders a diagram using the appropriate renderer.
func (r *RendererRegistry) Render(d *diagram.Diagram) (string, error) {
	renderer, err := r.GetRenderer(d.GetType())
	if err != nil {
		return "", err
	}
	return renderer.Render(d)
}
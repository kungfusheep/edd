package rendering

import (
	"edd/core"
	"fmt"
)

// RendererRegistry manages diagram renderers by type.
type RendererRegistry struct {
	renderers []core.DiagramRenderer
	fallback  core.DiagramRenderer
}

// NewRendererRegistry creates a new renderer registry.
func NewRendererRegistry() *RendererRegistry {
	return &RendererRegistry{
		renderers: make([]core.DiagramRenderer, 0),
	}
}

// Register adds a renderer to the registry.
func (r *RendererRegistry) Register(renderer core.DiagramRenderer) {
	r.renderers = append(r.renderers, renderer)
}

// SetFallback sets the default renderer to use when no specific renderer matches.
func (r *RendererRegistry) SetFallback(renderer core.DiagramRenderer) {
	r.fallback = renderer
}

// GetRenderer returns the appropriate renderer for the given diagram type.
func (r *RendererRegistry) GetRenderer(diagramType core.DiagramType) (core.DiagramRenderer, error) {
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
func (r *RendererRegistry) Render(diagram *core.Diagram) (string, error) {
	renderer, err := r.GetRenderer(diagram.GetType())
	if err != nil {
		return "", err
	}
	return renderer.Render(diagram)
}
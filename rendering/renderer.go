package rendering

import (
	"edd/canvas"
	"edd/core"
	"edd/validation"
	"fmt"
	"os"
)

// Renderer orchestrates the diagram rendering pipeline.
// It uses a registry to delegate to diagram-specific renderers.
type Renderer struct {
	registry      *RendererRegistry
	capabilities  canvas.TerminalCapabilities // Cached to avoid repeated detection
	validator     *validation.LineValidator // Optional output validator
	flowchartRenderer *FlowchartRenderer // Keep for backward compatibility
}

// NewRenderer creates a new renderer with sensible defaults.
func NewRenderer() *Renderer {
	// Detect terminal capabilities once
	caps := detectTerminalCapabilities()
	
	// Create the registry
	registry := NewRendererRegistry()
	
	// Create and register diagram-specific renderers
	flowchartRenderer := NewFlowchartRenderer(caps)
	sequenceRenderer := NewSequenceRenderer(caps)
	
	registry.Register(flowchartRenderer)
	registry.Register(sequenceRenderer)
	registry.SetFallback(flowchartRenderer) // Flowchart is the default
	
	return &Renderer{
		registry:      registry,
		capabilities:  caps,
		validator:     nil, // Validator is optional, enabled via SetValidator
		flowchartRenderer: flowchartRenderer, // Keep reference for backward compatibility
	}
}

// SetValidator enables output validation with the given validator.
func (r *Renderer) SetValidator(v *validation.LineValidator) {
	r.validator = v
}

// EnableValidation enables output validation with default settings.
func (r *Renderer) EnableValidation() {
	r.validator = validation.NewLineValidator()
}

// EnableDebug enables debug mode to show obstacle visualization.
func (r *Renderer) EnableDebug() {
	// Pass through to flowchart renderer for backward compatibility
	if r.flowchartRenderer != nil {
		r.flowchartRenderer.EnableDebug()
	}
}

// EnableObstacleVisualization enables showing virtual obstacles as dots in standard rendering
func (r *Renderer) EnableObstacleVisualization() {
	// Pass through to flowchart renderer for backward compatibility
	if r.flowchartRenderer != nil {
		r.flowchartRenderer.EnableObstacleVisualization()
	}
}

// GetRouter returns the router instance for external configuration
func (r *Renderer) GetRouter() interface{} {
	// Return the flowchart renderer's router for backward compatibility
	if r.flowchartRenderer != nil {
		return r.flowchartRenderer.GetRouter()
	}
	return nil
}

// SetRouterType sets the type of router to use
func (r *Renderer) SetRouterType(routerType interface{}) {
	// Pass through to flowchart renderer for backward compatibility
	// Note: This is a compatibility shim - the type system may need adjusting
	if r.flowchartRenderer != nil {
		// We'll need to fix this type issue in the integration phase
		// For now, leave it as a placeholder
	}
}

// detectTerminalCapabilities returns the current terminal's capabilities.
func detectTerminalCapabilities() canvas.TerminalCapabilities {
	// For now, return a simple default. In the future, this could
	// actually detect the terminal type and capabilities.
	return canvas.TerminalCapabilities{
		UnicodeLevel: canvas.UnicodeFull,
		SupportsColor: true,
	}
}


// Render orchestrates the complete rendering pipeline for a diagram.
// It delegates to the appropriate diagram-specific renderer.
func (r *Renderer) Render(diagram *core.Diagram) (string, error) {
	// Use the registry to get the appropriate renderer
	renderer, err := r.registry.GetRenderer(diagram.GetType())
	if err != nil {
		return "", fmt.Errorf("failed to get renderer: %w", err)
	}
	
	// Render the diagram
	output, err := renderer.Render(diagram)
	if err != nil {
		return "", fmt.Errorf("rendering failed: %w", err)
	}
	
	// Validate output if validator is enabled
	if r.validator != nil {
		errors := r.validator.Validate(output)
		if len(errors) > 0 {
			// Log validation errors but don't fail the render
			// In production, you might want to return these as warnings
			fmt.Fprintf(os.Stderr, "Warning: Output validation found %d issues:\n", len(errors))
			for i, err := range errors {
				if i >= 5 {
					fmt.Fprintf(os.Stderr, "  ... and %d more\n", len(errors)-5)
					break
				}
				fmt.Fprintf(os.Stderr, "  %s\n", err)
			}
		}
	}
	
	return output, nil
}
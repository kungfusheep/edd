package editor

import "edd/diagram"

// DiagramRenderer is the interface the TUI needs for rendering
type DiagramRenderer interface {
	Render(d *diagram.Diagram) (string, error)
}
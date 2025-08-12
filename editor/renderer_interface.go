package editor

import "edd/core"

// DiagramRenderer is the interface the TUI needs for rendering
type DiagramRenderer interface {
	Render(diagram *core.Diagram) (string, error)
}
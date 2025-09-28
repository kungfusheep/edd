package editor

import (
	"edd/diagram"
	"fmt"
	"strings"
)

// TUIState represents the complete state for rendering
type TUIState struct {
	Diagram    *diagram.Diagram
	Mode       Mode
	Selected   int            // Currently selected node ID
	JumpLabels map[int]rune   // Jump labels for nodes
	TextBuffer []rune         // Current text being edited
	CursorPos  int            // Cursor position in text buffer
	CursorLine int            // Current line in multi-line edit (0-based)
	CursorCol  int            // Current column in current line (0-based)
	EddFrame   string         // Current Ed mascot frame
	Width      int            // Terminal width
	Height     int            // Terminal height
}

// RenderTUI produces the complete rendered output from a state
// This is a pure function for easy testing
func RenderTUI(state TUIState) string {
	return RenderTUIWithRenderer(state, nil)
}

// RenderTUIWithRenderer renders with a specific renderer
func RenderTUIWithRenderer(state TUIState, renderer DiagramRenderer) string {
	if state.Diagram == nil {
		return "No diagram loaded\n"
	}

	var output string
	
	// 1. Render base diagram
	if renderer != nil && len(state.Diagram.Nodes) > 0 {
		// Use the real renderer
		rendered, err := renderer.Render(state.Diagram)
		if err != nil {
			output = fmt.Sprintf("Render error: %v\n", err)
		} else {
			output = rendered
		}
	} else if state.Diagram != nil && len(state.Diagram.Nodes) > 0 {
		// Fall back to simple representation for testing
		output = renderDiagramSimple(state.Diagram)
	} else {
		output = createEmptyCanvas(state.Width, state.Height)
	}

	// 2. Jump labels are now drawn separately in main_tui.go
	// Don't overlay them here

	// 3. Text editing is now handled by showing cursor in the node itself
	// No overlay needed

	// 4. Don't add Ed here - he's drawn separately in main_tui.go
	// output = overlayModeIndicator(output, state)

	return output
}

// renderDiagramSimple creates a simple text representation for testing
func renderDiagramSimple(d *diagram.Diagram) string {
	var lines []string
	
	// Add empty lines at top
	lines = append(lines, "")
	lines = append(lines, "")
	
	// Simple representation of nodes
	for _, node := range d.Nodes {
		nodeStr := fmt.Sprintf("  [%d] %s", node.ID, strings.Join(node.Text, " "))
		lines = append(lines, nodeStr)
	}
	
	// Add connections
	if len(d.Connections) > 0 {
		lines = append(lines, "")
		lines = append(lines, "  Connections:")
		for _, conn := range d.Connections {
			connStr := fmt.Sprintf("    %d -> %d", conn.From, conn.To)
			if conn.Label != "" {
				connStr += fmt.Sprintf(" (%s)", conn.Label)
			}
			lines = append(lines, connStr)
		}
	}
	
	return strings.Join(lines, "\n")
}

// createEmptyCanvas creates an empty canvas with borders
func createEmptyCanvas(width, height int) string {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	
	lines := make([]string, height)
	for i := range lines {
		lines[i] = strings.Repeat(" ", width)
	}
	
	// Add a simple border or message
	if height > 10 && width > 40 {
		msg := "Press 'a' to add a node, '?' for help, 'q' to quit"
		centerLine := height / 2
		centerCol := (width - len(msg)) / 2
		if centerCol > 0 && centerLine < len(lines) {
			line := lines[centerLine]
			lines[centerLine] = line[:centerCol] + msg + line[centerCol+len(msg):]
		}
	}
	
	return strings.Join(lines, "\n")
}

// CreateModeIndicatorTest is exported for testing
func CreateModeIndicatorTest(mode Mode, eddFrame string) string {
	return createModeIndicator(mode, eddFrame)
}

// createModeIndicator creates the mode indicator with Ed
func createModeIndicator(mode Mode, eddFrame string) string {
	if eddFrame == "" {
		eddFrame = "◉‿◉" // Default happy face
	}
	
	// Create colored mode indicator with Ed
	modeStr := mode.String()
	
	// Format Ed's face for display
	displayFrame := eddFrame
	if mode == ModeCommand && strings.HasPrefix(eddFrame, ":") {
		// Command mode special formatting - center the prompt
		if len(eddFrame) == 2 {
			displayFrame = fmt.Sprintf(" %s  ", eddFrame)
		} else {
			displayFrame = fmt.Sprintf(" %s ", eddFrame)
		}
	} else {
		// For normal faces, ensure they're 5 chars wide for box alignment
		// Faces are already formatted as "◉‿ ◉" (5 chars)
		// Just use as-is
		displayFrame = eddFrame
	}
	
	// Add ANSI colors based on mode
	var colorCode string
	switch mode {
	case ModeNormal:
		colorCode = "\033[36m" // Cyan
	case ModeInsert:
		colorCode = "\033[32m" // Green
	case ModeEdit:
		colorCode = "\033[33m" // Yellow
	case ModeJump:
		colorCode = "\033[35m" // Magenta
	case ModeCommand:
		colorCode = "\033[34m" // Blue
	default:
		colorCode = "\033[37m" // White
	}
	resetCode := "\033[0m"
	
	// Build the colored box with proper alignment
	// Box needs to fit the 5-char face: │◉‿ ◉│ = 7 chars total
	top := fmt.Sprintf("%s╭─────╮%s", colorCode, resetCode)
	middle := fmt.Sprintf("%s│%s│%s %s", colorCode, displayFrame, resetCode, modeStr)  
	bottom := fmt.Sprintf("%s╰─────╯%s", colorCode, resetCode)
	
	return fmt.Sprintf("%s\n%s\n%s", top, middle, bottom)
}
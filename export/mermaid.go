package export

import (
	"edd/diagram"
	"fmt"
	"strings"
)

// MermaidExporter exports diagrams to Mermaid syntax
type MermaidExporter struct{}

// NewMermaidExporter creates a new Mermaid exporter
func NewMermaidExporter() *MermaidExporter {
	return &MermaidExporter{}
}

// Export converts the diagram to Mermaid syntax
func (e *MermaidExporter) Export(d *diagram.Diagram) (string, error) {
	if d == nil {
		return "", fmt.Errorf("diagram is nil")
	}

	if len(d.Nodes) == 0 {
		return "", fmt.Errorf("diagram has no nodes")
	}

	// Determine diagram type and export accordingly
	if d.Type == "sequence" {
		return e.exportSequence(d)
	}
	return e.exportFlowchart(d)
}

// exportSequence exports a sequence diagram to Mermaid syntax
func (e *MermaidExporter) exportSequence(d *diagram.Diagram) (string, error) {
	var sb strings.Builder
	sb.WriteString("sequenceDiagram\n")

	// Create participant declarations
	// Map node IDs to participant names for easier reference
	nodeMap := make(map[int]string)
	for _, node := range d.Nodes {
		// Use the first line of text as the participant name
		name := e.getNodeLabel(node)
		// Create a valid participant ID (replace spaces with underscores)
		participantID := fmt.Sprintf("P%d", node.ID)
		nodeMap[node.ID] = participantID

		// Add participant declaration
		sb.WriteString(fmt.Sprintf("    participant %s as %s\n", participantID, name))
	}

	// Add a blank line between participants and messages
	if len(d.Connections) > 0 {
		sb.WriteString("\n")
	}

	// Add connections as messages
	for _, conn := range d.Connections {
		fromID, ok := nodeMap[conn.From]
		if !ok {
			continue // Skip invalid connections
		}
		toID, ok := nodeMap[conn.To]
		if !ok {
			continue
		}

		// Determine arrow type based on hints
		arrow := "->>" // Default to solid line with arrow
		if hints := conn.Hints; hints != nil {
			if style := hints["style"]; style == "dashed" {
				arrow = "-->>" // Dashed line with arrow
			}
		}

		// Handle self-loops
		if conn.From == conn.To {
			if conn.Label != "" {
				sb.WriteString(fmt.Sprintf("    %s%s%s: %s\n", fromID, arrow, fromID, conn.Label))
			} else {
				sb.WriteString(fmt.Sprintf("    %s%s%s: self\n", fromID, arrow, fromID))
			}
		} else {
			if conn.Label != "" {
				sb.WriteString(fmt.Sprintf("    %s%s%s: %s\n", fromID, arrow, toID, conn.Label))
			} else {
				sb.WriteString(fmt.Sprintf("    %s%s%s: \n", fromID, arrow, toID))
			}
		}
	}

	return sb.String(), nil
}

// exportFlowchart exports a flowchart/box diagram to Mermaid syntax
func (e *MermaidExporter) exportFlowchart(d *diagram.Diagram) (string, error) {
	var sb strings.Builder
	sb.WriteString("graph TD\n")

	// Create node declarations
	nodeMap := make(map[int]string)
	for _, node := range d.Nodes {
		// Create a valid node ID
		nodeID := fmt.Sprintf("N%d", node.ID)
		nodeMap[node.ID] = nodeID

		// Get node label
		label := e.getNodeLabel(node)

		// Determine node shape based on hints
		shape := e.getNodeShape(node)

		// Add node declaration with shape
		sb.WriteString(fmt.Sprintf("    %s%s\n", nodeID, e.formatNodeWithShape(label, shape)))
	}

	// Add a blank line between nodes and connections
	if len(d.Connections) > 0 {
		sb.WriteString("\n")
	}

	// Add connections
	for _, conn := range d.Connections {
		fromID, ok := nodeMap[conn.From]
		if !ok {
			continue
		}
		toID, ok := nodeMap[conn.To]
		if !ok {
			continue
		}

		// Determine connection style
		connStyle := "-->"
		if hints := conn.Hints; hints != nil {
			if style := hints["style"]; style == "dashed" {
				connStyle = "-.->"
			} else if style == "bold" {
				connStyle = "==>"
			}
		}

		// Add connection with optional label
		if conn.Label != "" {
			// For labeled connections in flowcharts, Mermaid uses: A --|text| B
			sb.WriteString(fmt.Sprintf("    %s %s|%s| %s\n", fromID, connStyle, conn.Label, toID))
		} else {
			sb.WriteString(fmt.Sprintf("    %s %s %s\n", fromID, connStyle, toID))
		}
	}

	return sb.String(), nil
}

// getNodeLabel extracts a label from a node
func (e *MermaidExporter) getNodeLabel(node diagram.Node) string {
	if len(node.Text) == 0 {
		return fmt.Sprintf("Node%d", node.ID)
	}

	// Join multiple lines with <br/> for Mermaid
	if len(node.Text) > 1 {
		// Escape special characters and join lines
		escaped := make([]string, len(node.Text))
		for i, line := range node.Text {
			escaped[i] = e.escapeLabel(line)
		}
		return strings.Join(escaped, "<br/>")
	}

	return e.escapeLabel(node.Text[0])
}

// escapeLabel escapes special characters in labels
func (e *MermaidExporter) escapeLabel(label string) string {
	// Escape quotes and other special characters
	label = strings.ReplaceAll(label, `"`, `\"`)
	label = strings.ReplaceAll(label, `|`, `\|`)
	label = strings.ReplaceAll(label, `[`, `\[`)
	label = strings.ReplaceAll(label, `]`, `\]`)
	label = strings.ReplaceAll(label, `{`, `\{`)
	label = strings.ReplaceAll(label, `}`, `\}`)
	label = strings.ReplaceAll(label, `(`, `\(`)
	label = strings.ReplaceAll(label, `)`, `\)`)
	return label
}

// getNodeShape determines the Mermaid shape based on node hints
func (e *MermaidExporter) getNodeShape(node diagram.Node) string {
	if node.Hints == nil {
		return "rectangle" // default
	}

	// Check for box-style hint
	if style := node.Hints["box-style"]; style != "" {
		switch style {
		case "rounded":
			return "rounded"
		case "double":
			return "double"
		case "hexagon":
			return "hexagon"
		case "circle":
			return "circle"
		}
	}

	// Check for shape hint
	if shape := node.Hints["shape"]; shape != "" {
		return shape
	}

	return "rectangle"
}

// formatNodeWithShape formats a node with its shape for Mermaid
func (e *MermaidExporter) formatNodeWithShape(label string, shape string) string {
	switch shape {
	case "rounded":
		return fmt.Sprintf("(%s)", label)
	case "double":
		return fmt.Sprintf("[[%s]]", label)
	case "hexagon":
		return fmt.Sprintf("{{%s}}", label)
	case "circle":
		return fmt.Sprintf("((%s))", label)
	case "rhombus", "diamond":
		return fmt.Sprintf("{%s}", label)
	default:
		return fmt.Sprintf("[%s]", label)
	}
}

// GetFileExtension returns the recommended file extension
func (e *MermaidExporter) GetFileExtension() string {
	return ".mmd"
}

// GetFormatName returns the format name
func (e *MermaidExporter) GetFormatName() string {
	return "Mermaid"
}
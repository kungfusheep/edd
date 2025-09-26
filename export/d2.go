package export

import (
	"edd/diagram"
	"fmt"
	"strings"
)

// D2Exporter exports diagrams to D2 syntax
type D2Exporter struct{}

// NewD2Exporter creates a new D2 exporter
func NewD2Exporter() *D2Exporter {
	return &D2Exporter{}
}

// Export converts the diagram to D2 syntax
func (e *D2Exporter) Export(d *diagram.Diagram) (string, error) {
	if d == nil {
		return "", fmt.Errorf("diagram is nil")
	}

	if len(d.Nodes) == 0 {
		return "", fmt.Errorf("diagram has no nodes")
	}

	var sb strings.Builder

	// Add title comment if diagram has metadata
	if d.Metadata.Name != "" {
		sb.WriteString(fmt.Sprintf("# %s\n\n", d.Metadata.Name))
	}

	// Process nodes
	nodeMap := make(map[int]string)
	for _, node := range d.Nodes {
		nodeID := e.getNodeID(node.ID)
		nodeMap[node.ID] = nodeID

		label := e.getNodeLabel(node)

		// Write node with label
		sb.WriteString(fmt.Sprintf("%s: %s\n", nodeID, label))

		// Apply node attributes from hints
		if node.Hints != nil {
			e.writeNodeAttributes(&sb, nodeID, node)
		}
	}

	// Add blank line between nodes and connections
	if len(d.Connections) > 0 {
		sb.WriteString("\n")
	}

	// Process connections
	for i, conn := range d.Connections {
		fromID := nodeMap[conn.From]
		toID := nodeMap[conn.To]

		// Determine arrow type
		arrow := e.getArrowType(conn)

		// Write connection
		if conn.Label != "" {
			sb.WriteString(fmt.Sprintf("%s %s %s: %s\n", fromID, arrow, toID, e.escapeLabel(conn.Label)))
		} else {
			sb.WriteString(fmt.Sprintf("%s %s %s\n", fromID, arrow, toID))
		}

		// Apply connection attributes from hints
		if conn.Hints != nil {
			connID := fmt.Sprintf("(%s %s %s)[%d]", fromID, arrow, toID, i)
			e.writeConnectionAttributes(&sb, connID, conn)
		}
	}

	return sb.String(), nil
}

// getNodeID returns a valid D2 node identifier
func (e *D2Exporter) getNodeID(id int) string {
	return fmt.Sprintf("node_%d", id)
}

// getNodeLabel extracts a label from a node
func (e *D2Exporter) getNodeLabel(node diagram.Node) string {
	if len(node.Text) == 0 {
		return fmt.Sprintf("Node %d", node.ID)
	}

	// Join multiple lines with \n for D2
	if len(node.Text) > 1 {
		// For multiline, join and then escape once
		joined := strings.Join(node.Text, "\\n")
		return e.escapeLabel(joined)
	}

	return e.escapeLabel(node.Text[0])
}

// escapeLabel escapes special characters in labels
func (e *D2Exporter) escapeLabel(label string) string {
	// D2 uses quotes for labels with special characters
	// But not for simple alphanumeric strings with spaces
	needsQuotes := false

	// Check if we need quotes (special chars that would confuse D2 parser)
	if strings.ContainsAny(label, ":-><|{}[]()\"") {
		needsQuotes = true
	}

	// Also check for newlines already in the string
	if strings.Contains(label, "\\n") {
		needsQuotes = false // D2 handles \n without quotes
	}

	if needsQuotes {
		// Escape quotes and backslashes
		label = strings.ReplaceAll(label, `\`, `\\`)
		label = strings.ReplaceAll(label, `"`, `\"`)
		return fmt.Sprintf("\"%s\"", label)
	}
	return label
}

// getArrowType determines the arrow syntax based on connection hints
func (e *D2Exporter) getArrowType(conn diagram.Connection) string {
	if conn.Hints == nil {
		return "->"
	}

	// Check style hint
	if style := conn.Hints["style"]; style != "" {
		switch style {
		case "dashed", "dotted":
			return "-->"
		case "thick", "double":
			return "=>"
		}
	}

	// Check bidirectional hint
	if bidir := conn.Hints["bidirectional"]; bidir == "true" {
		return "<->"
	}

	return "->"
}

// writeNodeAttributes writes D2 attributes for a node
func (e *D2Exporter) writeNodeAttributes(sb *strings.Builder, nodeID string, node diagram.Node) {
	// Handle shape
	if shape := node.Hints["shape"]; shape != "" {
		d2Shape := e.mapShapeToD2(shape)
		if d2Shape != "" {
			sb.WriteString(fmt.Sprintf("%s.shape: %s\n", nodeID, d2Shape))
		}
	}

	// Handle style (map to shape where appropriate)
	if style := node.Hints["style"]; style != "" {
		switch style {
		case "rounded":
			// D2 boxes are rounded by default
		case "double":
			sb.WriteString(fmt.Sprintf("%s.multiple: true\n", nodeID))
		case "thick":
			sb.WriteString(fmt.Sprintf("%s.style.stroke-width: 3\n", nodeID))
		case "dashed":
			sb.WriteString(fmt.Sprintf("%s.style.stroke-dash: 5\n", nodeID))
		}
	}

	// Handle box-style for compatibility
	if boxStyle := node.Hints["box-style"]; boxStyle != "" {
		switch boxStyle {
		case "double":
			sb.WriteString(fmt.Sprintf("%s.multiple: true\n", nodeID))
		case "rounded":
			// Default in D2
		}
	}

	// Handle colors
	if color := node.Hints["color"]; color != "" {
		hexColor := e.mapColorToHex(color)
		sb.WriteString(fmt.Sprintf("%s.style.fill: \"#%s\"\n", nodeID, hexColor))
	}

	// Handle bold
	if bold := node.Hints["bold"]; bold == "true" {
		sb.WriteString(fmt.Sprintf("%s.style.bold: true\n", nodeID))
	}

	// Handle italic
	if italic := node.Hints["italic"]; italic == "true" {
		sb.WriteString(fmt.Sprintf("%s.style.italic: true\n", nodeID))
	}

	// Handle shadow
	if shadow := node.Hints["shadow"]; shadow != "" {
		sb.WriteString(fmt.Sprintf("%s.style.shadow: true\n", nodeID))
	}

	// Handle text alignment
	if align := node.Hints["text-align"]; align == "center" {
		sb.WriteString(fmt.Sprintf("%s.near: center-center\n", nodeID))
	}
}

// writeConnectionAttributes writes D2 attributes for a connection
func (e *D2Exporter) writeConnectionAttributes(sb *strings.Builder, connID string, conn diagram.Connection) {
	// Style attributes that aren't handled by arrow type
	if style := conn.Hints["style"]; style != "" {
		switch style {
		case "dotted":
			sb.WriteString(fmt.Sprintf("%s.style.stroke-dash: 3\n", connID))
		case "dashed":
			sb.WriteString(fmt.Sprintf("%s.style.stroke-dash: 5\n", connID))
		case "thick":
			sb.WriteString(fmt.Sprintf("%s.style.stroke-width: 3\n", connID))
		}
	}

	// Handle colors
	if color := conn.Hints["color"]; color != "" {
		hexColor := e.mapColorToHex(color)
		sb.WriteString(fmt.Sprintf("%s.style.stroke: \"#%s\"\n", connID, hexColor))
	}

	// Handle bold
	if bold := conn.Hints["bold"]; bold == "true" {
		sb.WriteString(fmt.Sprintf("%s.style.bold: true\n", connID))
	}

	// Handle italic
	if italic := conn.Hints["italic"]; italic == "true" {
		sb.WriteString(fmt.Sprintf("%s.style.italic: true\n", connID))
	}

	// Handle flow direction for layout hints
	if flow := conn.Hints["flow"]; flow != "" {
		// D2 doesn't have direct flow control, but we can add as comment
		sb.WriteString(fmt.Sprintf("# Connection flow: %s\n", flow))
	}
}

// mapShapeToD2 maps our shape hints to D2 shape names
func (e *D2Exporter) mapShapeToD2(shape string) string {
	shapeMap := map[string]string{
		"rounded":     "rectangle", // Default is already rounded
		"circle":      "circle",
		"diamond":     "diamond",
		"rhombus":     "diamond",
		"hexagon":     "hexagon",
		"cylinder":    "cylinder",
		"parallelogram": "parallelogram",
		"oval":        "oval",
		"ellipse":     "oval",
		"cloud":       "cloud",
		"document":    "document",
		"package":     "package",
		"step":        "step",
		"callout":     "callout",
		"stored_data": "stored_data",
	}

	if d2Shape, ok := shapeMap[shape]; ok {
		return d2Shape
	}
	return "" // Use default
}

// mapColorToHex maps color names to hex codes
func (e *D2Exporter) mapColorToHex(color string) string {
	colorMap := map[string]string{
		"red":     "FF6B6B",
		"green":   "51CF66",
		"blue":    "339AF0",
		"yellow":  "FFD43B",
		"magenta": "FF6B9D",
		"cyan":    "22B8CF",
		"white":   "FFFFFF",
		"black":   "212529",
		"gray":    "868E96",
		"grey":    "868E96",
	}

	if hex, ok := colorMap[strings.ToLower(color)]; ok {
		return hex
	}

	// If it's already a hex code (with or without #), return it
	color = strings.TrimPrefix(color, "#")
	if len(color) == 6 {
		return color
	}

	return "000000" // Default to black
}

// GetFileExtension returns the recommended file extension
func (e *D2Exporter) GetFileExtension() string {
	return ".d2"
}

// GetFormatName returns the format name
func (e *D2Exporter) GetFormatName() string {
	return "D2"
}
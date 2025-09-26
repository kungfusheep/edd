package export

import (
	"edd/diagram"
	"fmt"
	"strings"
)

// GraphvizExporter exports diagrams to Graphviz DOT syntax
type GraphvizExporter struct{}

// NewGraphvizExporter creates a new Graphviz exporter
func NewGraphvizExporter() *GraphvizExporter {
	return &GraphvizExporter{}
}

// Export converts the diagram to Graphviz DOT syntax
func (e *GraphvizExporter) Export(d *diagram.Diagram) (string, error) {
	if d == nil {
		return "", fmt.Errorf("diagram is nil")
	}

	if len(d.Nodes) == 0 {
		return "", fmt.Errorf("diagram has no nodes")
	}

	var sb strings.Builder

	// Start digraph
	sb.WriteString("digraph G {\n")

	// Global attributes for better appearance
	sb.WriteString("  rankdir=TB;\n")
	sb.WriteString("  node [shape=box];\n")
	sb.WriteString("  edge [arrowhead=normal];\n\n")

	// Process nodes
	for _, node := range d.Nodes {
		nodeID := e.getNodeID(node.ID)
		label := e.getNodeLabel(node)
		attributes := e.getNodeAttributes(node)

		if attributes != "" {
			sb.WriteString(fmt.Sprintf("  %s [label=\"%s\", %s];\n", nodeID, label, attributes))
		} else {
			sb.WriteString(fmt.Sprintf("  %s [label=\"%s\"];\n", nodeID, label))
		}
	}

	// Add blank line between nodes and edges
	if len(d.Connections) > 0 {
		sb.WriteString("\n")
	}

	// Process connections
	for _, conn := range d.Connections {
		fromID := e.getNodeID(conn.From)
		toID := e.getNodeID(conn.To)
		attributes := e.getEdgeAttributes(conn)

		if attributes != "" {
			sb.WriteString(fmt.Sprintf("  %s -> %s [%s];\n", fromID, toID, attributes))
		} else {
			sb.WriteString(fmt.Sprintf("  %s -> %s;\n", fromID, toID))
		}
	}

	sb.WriteString("}\n")
	return sb.String(), nil
}

// getNodeID returns a valid DOT node identifier
func (e *GraphvizExporter) getNodeID(id int) string {
	return fmt.Sprintf("N%d", id)
}

// getNodeLabel extracts a label from a node
func (e *GraphvizExporter) getNodeLabel(node diagram.Node) string {
	if len(node.Text) == 0 {
		return fmt.Sprintf("Node%d", node.ID)
	}

	// Join multiple lines with \n for DOT
	if len(node.Text) > 1 {
		escaped := make([]string, len(node.Text))
		for i, line := range node.Text {
			escaped[i] = e.escapeLabel(line)
		}
		return strings.Join(escaped, "\\n")
	}

	return e.escapeLabel(node.Text[0])
}

// escapeLabel escapes special characters in labels
func (e *GraphvizExporter) escapeLabel(label string) string {
	// Escape quotes and backslashes
	label = strings.ReplaceAll(label, `\`, `\\`)
	label = strings.ReplaceAll(label, `"`, `\"`)
	return label
}

// getNodeAttributes builds DOT attributes from node hints
func (e *GraphvizExporter) getNodeAttributes(node diagram.Node) string {
	if node.Hints == nil {
		return ""
	}

	var attrs []string

	// Map shape hint to DOT shape
	if shape := node.Hints["shape"]; shape != "" {
		dotShape := e.mapShapeToDOT(shape)
		if dotShape != "box" { // box is default
			attrs = append(attrs, fmt.Sprintf("shape=%s", dotShape))
		}
	}

	// Map style hints to DOT style
	styles := []string{}
	if style := node.Hints["style"]; style != "" {
		switch style {
		case "rounded":
			styles = append(styles, "rounded")
		case "double":
			attrs = append(attrs, "peripheries=2")
		case "thick":
			attrs = append(attrs, "penwidth=2")
		case "dashed":
			styles = append(styles, "dashed")
		case "dotted":
			styles = append(styles, "dotted")
		}
	}

	// Handle box-style for compatibility
	if boxStyle := node.Hints["box-style"]; boxStyle != "" {
		switch boxStyle {
		case "rounded":
			styles = append(styles, "rounded")
		case "double":
			attrs = append(attrs, "peripheries=2")
		}
	}

	// Handle bold
	if bold := node.Hints["bold"]; bold == "true" {
		styles = append(styles, "bold")
	}

	// Apply accumulated styles
	if len(styles) > 0 {
		attrs = append(attrs, fmt.Sprintf("style=\"%s\"", strings.Join(styles, ",")))
	}

	// Map color hint to DOT color
	if color := node.Hints["color"]; color != "" {
		hexColor := e.mapColorToHex(color)
		attrs = append(attrs, fmt.Sprintf("fillcolor=\"#%s\"", hexColor))
		attrs = append(attrs, "style=\"filled\"")
	}

	// Handle shadow (approximate with gray border)
	if shadow := node.Hints["shadow"]; shadow != "" {
		attrs = append(attrs, "color=\"#808080\"")
		attrs = append(attrs, "penwidth=1.5")
	}

	return strings.Join(attrs, ", ")
}

// getEdgeAttributes builds DOT attributes from connection hints
func (e *GraphvizExporter) getEdgeAttributes(conn diagram.Connection) string {
	var attrs []string

	// Add label if present
	if conn.Label != "" {
		attrs = append(attrs, fmt.Sprintf("label=\"%s\"", e.escapeLabel(conn.Label)))
	}

	if conn.Hints == nil {
		return strings.Join(attrs, ", ")
	}

	// Map style hint to DOT style
	if style := conn.Hints["style"]; style != "" {
		switch style {
		case "dashed":
			attrs = append(attrs, "style=dashed")
		case "dotted":
			attrs = append(attrs, "style=dotted")
		case "thick":
			attrs = append(attrs, "penwidth=2")
		case "double":
			attrs = append(attrs, "penwidth=3")
		}
	}

	// Handle bold
	if bold := conn.Hints["bold"]; bold == "true" {
		attrs = append(attrs, "style=bold")
	}

	// Map color hint to DOT color
	if color := conn.Hints["color"]; color != "" {
		hexColor := e.mapColorToHex(color)
		attrs = append(attrs, fmt.Sprintf("color=\"#%s\"", hexColor))
	}

	// Handle bidirectional
	if bidir := conn.Hints["bidirectional"]; bidir == "true" {
		attrs = append(attrs, "dir=both")
	}

	// Handle flow direction hints (for layout)
	if flow := conn.Hints["flow"]; flow != "" {
		switch flow {
		case "down":
			attrs = append(attrs, "constraint=true")
		case "up":
			attrs = append(attrs, "constraint=true")
		case "left", "right":
			attrs = append(attrs, "constraint=false")
		}
	}

	return strings.Join(attrs, ", ")
}

// mapShapeToDOT maps our shape hints to Graphviz shape names
func (e *GraphvizExporter) mapShapeToDOT(shape string) string {
	shapeMap := map[string]string{
		"rounded":     "box",  // Will use style=rounded
		"circle":      "circle",
		"diamond":     "diamond",
		"rhombus":     "diamond",
		"hexagon":     "hexagon",
		"ellipse":     "ellipse",
		"parallelogram": "parallelogram",
		"cylinder":    "cylinder",
		"trapezoid":   "trapezium",
		"double":      "doublecircle",
	}

	if dotShape, ok := shapeMap[shape]; ok {
		return dotShape
	}
	return "box" // Default
}

// mapColorToHex maps color names to hex codes
func (e *GraphvizExporter) mapColorToHex(color string) string {
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
func (e *GraphvizExporter) GetFileExtension() string {
	return ".dot"
}

// GetFormatName returns the format name
func (e *GraphvizExporter) GetFormatName() string {
	return "Graphviz"
}
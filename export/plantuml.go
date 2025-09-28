package export

import (
	"edd/diagram"
	"fmt"
	"strings"
)

// PlantUMLExporter exports diagrams to PlantUML syntax
type PlantUMLExporter struct{}

// NewPlantUMLExporter creates a new PlantUML exporter
func NewPlantUMLExporter() *PlantUMLExporter {
	return &PlantUMLExporter{}
}

// Export converts the diagram to PlantUML syntax
func (e *PlantUMLExporter) Export(d *diagram.Diagram) (string, error) {
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
	return e.exportActivity(d)
}

// exportSequence exports a sequence diagram to PlantUML syntax
func (e *PlantUMLExporter) exportSequence(d *diagram.Diagram) (string, error) {
	var sb strings.Builder
	sb.WriteString("@startuml\n")

	// Add skinparam for better appearance
	sb.WriteString("skinparam backgroundColor white\n")
	sb.WriteString("skinparam shadowing false\n\n")

	// Create participant declarations
	nodeMap := make(map[int]string)
	for _, node := range d.Nodes {
		name := e.getNodeLabel(node)
		// Use the node ID as the participant identifier
		participantID := fmt.Sprintf("P%d", node.ID)
		nodeMap[node.ID] = participantID

		// Determine participant type based on hints
		participantType := "participant"
		if hints := node.Hints; hints != nil {
			if boxStyle := hints["box-style"]; boxStyle == "double" {
				participantType = "database"
			} else if hints["type"] == "actor" {
				participantType = "actor"
			} else if hints["type"] == "boundary" {
				participantType = "boundary"
			} else if hints["type"] == "control" {
				participantType = "control"
			} else if hints["type"] == "entity" {
				participantType = "entity"
			}
		}

		// Add color if specified
		colorDirective := ""
		if hints := node.Hints; hints != nil {
			if color := hints["color"]; color != "" {
				colorDirective = fmt.Sprintf(" #%s", e.mapColorToHex(color))
			}
		}

		// Add participant declaration
		sb.WriteString(fmt.Sprintf("%s \"%s\" as %s%s\n", participantType, name, participantID, colorDirective))
	}

	// Add a blank line between participants and messages
	if len(d.Connections) > 0 {
		sb.WriteString("\n")
	}

	// Track which participants are currently activated (stack for nested activations)
	activationStack := []string{}

	// Add connections as messages
	for _, conn := range d.Connections {
		fromID, ok := nodeMap[conn.From]
		if !ok {
			continue
		}
		toID, ok := nodeMap[conn.To]
		if !ok {
			continue
		}

		// Check for activation/deactivation hints
		activateSource := false
		activateTarget := false
		deactivateSource := false
		deactivateTarget := false

		// Determine arrow type and color based on hints
		arrowStyle := "-"
		arrowHead := ">"
		colorPart := ""

		if hints := conn.Hints; hints != nil {
			if style := hints["style"]; style == "dashed" {
				arrowStyle = "--"
			} else if style == "bold" {
				arrowHead = ">>"
			}
			if color := hints["color"]; color != "" {
				colorPart = fmt.Sprintf("[#%s]", e.mapColorToHex(color))
			}
			// Check activation hints
			// activate_source means the FROM participant gets activated
			if hints["activate_source"] == "true" {
				activateSource = true
			}
			// activate means the TO participant gets activated
			if hints["activate"] == "true" {
				activateTarget = true
			}
			// deactivate means the FROM participant gets deactivated
			if hints["deactivate"] == "true" {
				deactivateSource = true
			}
			// deactivate_target means the TO participant gets deactivated
			if hints["deactivate_target"] == "true" {
				deactivateTarget = true
			}
		}

		// Construct the full arrow with color in the middle
		arrow := fmt.Sprintf("%s%s%s", arrowStyle, colorPart, arrowHead)

		// Add activate BEFORE the message if needed (PlantUML style)
		if activateSource {
			sb.WriteString(fmt.Sprintf("activate %s\n", fromID))
			activationStack = append(activationStack, fromID)
		}
		if activateTarget {
			sb.WriteString(fmt.Sprintf("activate %s\n", toID))
			activationStack = append(activationStack, toID)
		}

		// Handle self-loops
		if conn.From == conn.To {
			// For self-calls with activation, PlantUML handles it automatically
			if conn.Label != "" {
				sb.WriteString(fmt.Sprintf("%s %s %s : %s\n", fromID, arrow, fromID, conn.Label))
			} else {
				sb.WriteString(fmt.Sprintf("%s %s %s\n", fromID, arrow, fromID))
			}
		} else {
			if conn.Label != "" {
				sb.WriteString(fmt.Sprintf("%s %s %s : %s\n", fromID, arrow, toID, conn.Label))
			} else {
				sb.WriteString(fmt.Sprintf("%s %s %s\n", fromID, arrow, toID))
			}
		}

		// Add deactivate AFTER the message that causes deactivation
		// The deactivate hint means "deactivate the most recently activated participant"
		if deactivateSource && len(activationStack) > 0 {
			// Pop from stack and deactivate
			lastActive := activationStack[len(activationStack)-1]
			activationStack = activationStack[:len(activationStack)-1]
			sb.WriteString(fmt.Sprintf("deactivate %s\n", lastActive))
		}
		if deactivateTarget {
			sb.WriteString(fmt.Sprintf("deactivate %s\n", toID))
		}
	}

	sb.WriteString("@enduml\n")
	return sb.String(), nil
}

// exportActivity exports a box/flowchart diagram
func (e *PlantUMLExporter) exportActivity(d *diagram.Diagram) (string, error) {
	var sb strings.Builder
	sb.WriteString("@startuml\n")
	sb.WriteString("!theme plain\n")
	sb.WriteString("skinparam backgroundColor white\n")
	sb.WriteString("skinparam componentStyle rectangle\n\n")

	// Create a map to store node IDs to PlantUML-safe identifiers
	nodeMap := make(map[int]string)

	// First, declare all nodes as components
	for _, node := range d.Nodes {
		// Create a safe identifier for PlantUML
		nodeID := fmt.Sprintf("N%d", node.ID)
		nodeMap[node.ID] = nodeID

		label := e.getNodeLabel(node)

		// Apply color and style hints if available
		style := ""
		if hints := node.Hints; hints != nil {
			if color := hints["color"]; color != "" {
				hexColor := e.mapColorToHex(color)
				style = fmt.Sprintf(" #%s", hexColor)
			}
		}

		// Write the component declaration
		sb.WriteString(fmt.Sprintf("component \"%s\" as %s%s\n", label, nodeID, style))
	}

	// Add a blank line between nodes and connections
	if len(d.Connections) > 0 {
		sb.WriteString("\n")
	}

	// Then add all connections
	for _, conn := range d.Connections {
		fromID, fromExists := nodeMap[conn.From]
		toID, toExists := nodeMap[conn.To]

		if !fromExists || !toExists {
			continue
		}

		// Determine arrow style based on hints
		arrowStyle := "-->"
		if hints := conn.Hints; hints != nil {
			if style := hints["style"]; style == "dashed" || style == "dotted" {
				arrowStyle = "..>"
			}
		}

		// Add the connection with label if present
		if conn.Label != "" {
			sb.WriteString(fmt.Sprintf("%s %s %s : %s\n", fromID, arrowStyle, toID, conn.Label))
		} else {
			sb.WriteString(fmt.Sprintf("%s %s %s\n", fromID, arrowStyle, toID))
		}
	}

	sb.WriteString("\n@enduml\n")
	return sb.String(), nil
}


// findStartNodes finds nodes with no incoming connections
func (e *PlantUMLExporter) findStartNodes(d *diagram.Diagram) []int {
	hasIncoming := make(map[int]bool)
	for _, conn := range d.Connections {
		hasIncoming[conn.To] = true
	}

	var startNodes []int
	for _, node := range d.Nodes {
		if !hasIncoming[node.ID] {
			startNodes = append(startNodes, node.ID)
		}
	}

	// If no start nodes found (circular graph), use the first node
	if len(startNodes) == 0 && len(d.Nodes) > 0 {
		startNodes = append(startNodes, d.Nodes[0].ID)
	}

	return startNodes
}

// findOutgoingConnections finds all connections from a node
func (e *PlantUMLExporter) findOutgoingConnections(d *diagram.Diagram, nodeID int) []diagram.Connection {
	var outgoing []diagram.Connection
	for _, conn := range d.Connections {
		if conn.From == nodeID {
			outgoing = append(outgoing, conn)
		}
	}
	return outgoing
}

// hasLabels checks if any connection has a label
func (e *PlantUMLExporter) hasLabels(connections []diagram.Connection) bool {
	for _, conn := range connections {
		if conn.Label != "" {
			return true
		}
	}
	return false
}

// getNodeLabel extracts a label from a node
func (e *PlantUMLExporter) getNodeLabel(node diagram.Node) string {
	if len(node.Text) == 0 {
		return fmt.Sprintf("Node%d", node.ID)
	}

	// Join multiple lines with \n for PlantUML
	if len(node.Text) > 1 {
		return strings.Join(node.Text, "\\n")
	}

	return node.Text[0]
}

// mapColorToHex maps color names to hex codes
func (e *PlantUMLExporter) mapColorToHex(color string) string {
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
func (e *PlantUMLExporter) GetFileExtension() string {
	return ".puml"
}

// GetFormatName returns the format name
func (e *PlantUMLExporter) GetFormatName() string {
	return "PlantUML"
}
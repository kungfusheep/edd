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
		activateTarget := false
		deactivateSource := false

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
			if hints["activate"] == "true" {
				activateTarget = true
			}
			if hints["deactivate"] == "true" {
				deactivateSource = true
			}
		}

		// Add deactivate before the message if needed
		if deactivateSource {
			sb.WriteString(fmt.Sprintf("deactivate %s\n", fromID))
		}

		// Construct the full arrow with color in the middle
		arrow := fmt.Sprintf("%s%s%s", arrowStyle, colorPart, arrowHead)

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

		// Add activate after the message if needed
		if activateTarget {
			sb.WriteString(fmt.Sprintf("activate %s\n", toID))
		}
	}

	sb.WriteString("@enduml\n")
	return sb.String(), nil
}

// exportActivity exports a flowchart as a PlantUML activity diagram
func (e *PlantUMLExporter) exportActivity(d *diagram.Diagram) (string, error) {
	var sb strings.Builder
	sb.WriteString("@startuml\n")
	sb.WriteString("!theme plain\n")
	sb.WriteString("skinparam backgroundColor white\n\n")

	// For flowcharts, we'll use activity diagram syntax (beta)
	sb.WriteString("start\n\n")

	// Create a map of node IDs to their labels
	nodeMap := make(map[int]string)
	processedNodes := make(map[int]bool)

	// Find the start node (node with no incoming connections)
	startNodes := e.findStartNodes(d)

	// Build the flow recursively
	for _, startNode := range startNodes {
		e.buildActivityFlow(&sb, d, startNode, nodeMap, processedNodes, 0)
	}

	// Add any disconnected nodes
	for _, node := range d.Nodes {
		if !processedNodes[node.ID] {
			label := e.getNodeLabel(node)
			sb.WriteString(fmt.Sprintf(":%s;\n", label))
			processedNodes[node.ID] = true
		}
	}

	sb.WriteString("\nstop\n")
	sb.WriteString("@enduml\n")
	return sb.String(), nil
}

// buildActivityFlow recursively builds the activity flow
func (e *PlantUMLExporter) buildActivityFlow(sb *strings.Builder, d *diagram.Diagram, nodeID int, nodeMap map[int]string, processed map[int]bool, depth int) {
	if processed[nodeID] {
		return // Already processed this node
	}

	// Find the node
	var currentNode *diagram.Node
	for _, node := range d.Nodes {
		if node.ID == nodeID {
			currentNode = &node
			break
		}
	}

	if currentNode == nil {
		return
	}

	processed[nodeID] = true
	label := e.getNodeLabel(*currentNode)

	// Check if this is a decision node (has multiple outgoing connections with labels)
	outgoing := e.findOutgoingConnections(d, nodeID)

	if len(outgoing) > 1 && e.hasLabels(outgoing) {
		// Decision node
		sb.WriteString(fmt.Sprintf("if (%s?) then\n", label))

		first := true
		for _, conn := range outgoing {
			if first {
				if conn.Label != "" {
					sb.WriteString(fmt.Sprintf("  (%s)\n", conn.Label))
				} else {
					sb.WriteString("  (yes)\n")
				}
				first = false
			} else {
				if conn.Label != "" {
					sb.WriteString(fmt.Sprintf("else (%s)\n", conn.Label))
				} else {
					sb.WriteString("else (no)\n")
				}
			}

			// Process the target node
			e.buildActivityFlow(sb, d, conn.To, nodeMap, processed, depth+1)
		}
		sb.WriteString("endif\n")
	} else {
		// Regular activity node
		nodeStyle := ""
		if hints := currentNode.Hints; hints != nil {
			if color := hints["color"]; color != "" {
				nodeStyle = fmt.Sprintf("#%s", e.mapColorToHex(color))
			}
		}

		if nodeStyle != "" {
			sb.WriteString(fmt.Sprintf(":%s|%s;\n", label, nodeStyle))
		} else {
			sb.WriteString(fmt.Sprintf(":%s;\n", label))
		}

		// Process outgoing connections
		for _, conn := range outgoing {
			if conn.Label != "" {
				sb.WriteString(fmt.Sprintf("note right: %s\n", conn.Label))
			}
			e.buildActivityFlow(sb, d, conn.To, nodeMap, processed, depth+1)
		}
	}
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
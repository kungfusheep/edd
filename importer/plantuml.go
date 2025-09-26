package importer

import (
	"edd/diagram"
	"fmt"
	"regexp"
	"strings"
)

// PlantUMLImporter imports PlantUML diagram format
type PlantUMLImporter struct{}

// NewPlantUMLImporter creates a new PlantUML importer
func NewPlantUMLImporter() *PlantUMLImporter {
	return &PlantUMLImporter{}
}

// CanImport checks if the content is a PlantUML diagram
func (p *PlantUMLImporter) CanImport(content string) bool {
	content = strings.TrimSpace(content)
	return strings.HasPrefix(content, "@startuml") ||
		strings.HasPrefix(content, "@startdot") ||
		strings.HasPrefix(content, "@startmindmap")
}

// Import converts PlantUML content to edd diagram
func (p *PlantUMLImporter) Import(content string) (*diagram.Diagram, error) {
	content = strings.TrimSpace(content)

	// Check for activity diagram markers
	if strings.Contains(content, ":") && (strings.Contains(content, ";") || strings.Contains(content, "if ")) {
		return p.importActivityDiagram(content)
	}

	// Check for sequence diagram markers
	if strings.Contains(content, "->") || strings.Contains(content, "-->") {
		return p.importSequenceDiagram(content)
	}

	return nil, fmt.Errorf("unsupported PlantUML diagram type")
}

// GetFormatName returns the format name
func (p *PlantUMLImporter) GetFormatName() string {
	return "PlantUML"
}

// GetFileExtensions returns common file extensions
func (p *PlantUMLImporter) GetFileExtensions() []string {
	return []string{".puml", ".plantuml", ".pu"}
}

// importSequenceDiagram imports a PlantUML sequence diagram
func (p *PlantUMLImporter) importSequenceDiagram(content string) (*diagram.Diagram, error) {
	d := &diagram.Diagram{
		Type: "sequence",
	}

	participantMap := make(map[string]int)
	nextID := 0

	lines := strings.Split(content, "\n")

	// First pass: collect participants
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "'") || line == "@startuml" || line == "@enduml" {
			continue
		}

		// Parse participant/actor declarations
		if strings.HasPrefix(line, "participant ") || strings.HasPrefix(line, "actor ") {
			// Check for alias syntax with optional color: participant "Name" as Alias #color
			aliasPattern := regexp.MustCompile(`^(participant|actor)\s+"([^"]+)"\s+as\s+(\w+)\s*(#[0-9A-Fa-f]+)?`)
			if matches := aliasPattern.FindStringSubmatch(line); len(matches) >= 4 {
				// Has alias - use the quoted name as display and alias as key
				displayName := matches[2]
				alias := matches[3]

				if _, exists := participantMap[alias]; !exists {
					node := diagram.Node{
						ID:   nextID,
						Text: []string{displayName},
					}

					// Check for color hint (matches[4] if present)
					if len(matches) > 4 && matches[4] != "" {
						if node.Hints == nil {
							node.Hints = make(map[string]string)
						}
						// Convert hex color to named color for TUI
						hexColor := strings.TrimPrefix(matches[4], "#")
						node.Hints["color"] = p.mapHexToColor(hexColor)
					}

					d.Nodes = append(d.Nodes, node)
					participantMap[alias] = nextID
					nextID++
				}
			} else {
				// No alias - simple participant declaration
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					name := strings.Join(parts[1:], " ")
					name = strings.Trim(name, `"`)
					if _, exists := participantMap[name]; !exists {
						d.Nodes = append(d.Nodes, diagram.Node{
							ID:   nextID,
							Text: []string{name},
						})
						participantMap[name] = nextID
						nextID++
					}
				}
			}
		}
	}

	// Second pass: process messages and activations
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "'") || line == "@startuml" || line == "@enduml" ||
		   strings.HasPrefix(line, "participant ") || strings.HasPrefix(line, "actor ") ||
		   strings.HasPrefix(line, "skinparam") {
			continue
		}

		if strings.HasPrefix(line, "activate ") {
			// Activate happens AFTER the previous message is added
			// Apply it to the most recent connection
			parts := strings.Fields(line)
			if len(parts) >= 2 && len(d.Connections) > 0 {
				activateAlias := parts[1]
				lastConn := &d.Connections[len(d.Connections)-1]

				// Check which participant ID this alias refers to
				if pid, exists := participantMap[activateAlias]; exists {
					if pid == lastConn.To {
						// Activating the recipient of the last message
						lastConn.Hints["activate"] = "true"
					} else if pid == lastConn.From {
						// Activating the sender of the last message
						lastConn.Hints["activate_source"] = "true"
					}
				}
			}
		} else if strings.HasPrefix(line, "deactivate ") {
			// Deactivate happens AFTER the previous message
			parts := strings.Fields(line)
			if len(parts) >= 2 && len(d.Connections) > 0 {
				// Apply deactivate to the most recent connection
				lastConn := &d.Connections[len(d.Connections)-1]
				lastConn.Hints["deactivate"] = "true"
			}
		} else {
			// Parse messages
			// Pattern: Alice -> Bob: Message or Alice --> Bob: Message
			messagePattern := regexp.MustCompile(`^([^-]+?)\s*(->|-->|-\[#[^\]]+\]>|--\[#[^\]]+\]>)\s*([^:]+)\s*:\s*(.*)$`)
			matches := messagePattern.FindStringSubmatch(line)
			if len(matches) == 5 {
				fromName := strings.TrimSpace(matches[1])
				arrow := matches[2]
				toName := strings.TrimSpace(matches[3])
				label := strings.TrimSpace(matches[4])

				// Ensure participants exist
				if _, exists := participantMap[fromName]; !exists {
					d.Nodes = append(d.Nodes, diagram.Node{
						ID:   nextID,
						Text: []string{fromName},
					})
					participantMap[fromName] = nextID
					nextID++
				}
				if _, exists := participantMap[toName]; !exists {
					d.Nodes = append(d.Nodes, diagram.Node{
						ID:   nextID,
						Text: []string{toName},
					})
					participantMap[toName] = nextID
					nextID++
				}

				conn := diagram.Connection{
					From:  participantMap[fromName],
					To:    participantMap[toName],
					Label: label,
					Hints: make(map[string]string),
				}

				// Add style hints
				if strings.Contains(arrow, "--") {
					conn.Hints["style"] = "dashed"
				}

				// Extract color if present
				colorPattern := regexp.MustCompile(`\[#([^\]]+)\]`)
				if colorMatches := colorPattern.FindStringSubmatch(arrow); len(colorMatches) > 1 {
					conn.Hints["color"] = colorMatches[1]
				}

				d.Connections = append(d.Connections, conn)
			}
		}
	}

	return d, nil
}

// mapHexToColor converts a hex color code to a named color for the TUI
func (p *PlantUMLImporter) mapHexToColor(hexColor string) string {
	// Map of known hex colors to TUI color names
	// These should match what the PlantUML exporter uses
	hexToName := map[string]string{
		"FF6B6B": "red",
		"51CF66": "green",
		"339AF0": "blue",
		"FFD43B": "yellow",
		"FF6B9D": "magenta",
		"22B8CF": "cyan",
		"FFFFFF": "white",
		"212529": "black",
		"868E96": "gray",
	}

	// Normalize hex color to uppercase
	hexColor = strings.ToUpper(strings.TrimPrefix(hexColor, "#"))

	if name, ok := hexToName[hexColor]; ok {
		return name
	}

	// If we don't recognize the hex color, just return it as is
	// The TUI might not display it, but at least we preserve it
	return hexColor
}

// importActivityDiagram imports a PlantUML activity diagram
func (p *PlantUMLImporter) importActivityDiagram(content string) (*diagram.Diagram, error) {
	d := &diagram.Diagram{
		Type: "box", // Activity diagrams are flowchart-like
	}

	nodeMap := make(map[string]int)
	nextID := 0
	var lastNodeID *int

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "'") || line == "@startuml" || line == "@enduml" {
			continue
		}
		if line == "start" || line == "stop" || line == "end" {
			continue // Skip start/stop keywords
		}

		// Parse activity nodes (format: :Activity text;)
		activityPattern := regexp.MustCompile(`^:([^;]+);?$`)
		if matches := activityPattern.FindStringSubmatch(line); len(matches) >= 2 {
			activityText := strings.TrimSpace(matches[1])

			// Create node
			node := diagram.Node{
				ID:    nextID,
				Text:  []string{activityText},
				Hints: make(map[string]string),
			}

			// Check for color hints in activity text
			if strings.Contains(activityText, "|") {
				parts := strings.Split(activityText, "|")
				if len(parts) >= 2 {
					node.Text = []string{strings.TrimSpace(parts[0])}
					colorPart := strings.TrimSpace(parts[1])
					if strings.HasPrefix(colorPart, "#") {
						node.Hints["color"] = strings.TrimPrefix(colorPart, "#")
					}
				}
			}

			d.Nodes = append(d.Nodes, node)
			nodeMap[activityText] = nextID

			// Create connection from last node if exists
			if lastNodeID != nil {
				d.Connections = append(d.Connections, diagram.Connection{
					From:  *lastNodeID,
					To:    nextID,
					Hints: make(map[string]string),
				})
			}

			lastNodeID = &nextID
			nextID++
			continue
		}

		// Parse if conditions (format: if (condition?) then)
		ifPattern := regexp.MustCompile(`^if\s*\(([^)]+)\)\s*then`)
		if matches := ifPattern.FindStringSubmatch(line); len(matches) >= 2 {
			condition := strings.TrimSpace(matches[1])

			// Create decision node
			node := diagram.Node{
				ID:    nextID,
				Text:  []string{condition},
				Hints: make(map[string]string),
			}
			node.Hints["shape"] = "diamond" // Decision nodes are diamonds

			d.Nodes = append(d.Nodes, node)
			nodeMap[condition] = nextID

			// Create connection from last node if exists
			if lastNodeID != nil {
				d.Connections = append(d.Connections, diagram.Connection{
					From:  *lastNodeID,
					To:    nextID,
					Hints: make(map[string]string),
				})
			}

			lastNodeID = &nextID
			nextID++
			continue
		}

		// Parse notes
		if strings.HasPrefix(line, "note ") {
			noteText := strings.TrimPrefix(line, "note ")
			noteText = strings.TrimSpace(noteText)

			// Add note to last node if exists
			if lastNodeID != nil && *lastNodeID < len(d.Nodes) {
				if d.Nodes[*lastNodeID].Hints == nil {
					d.Nodes[*lastNodeID].Hints = make(map[string]string)
				}
				d.Nodes[*lastNodeID].Hints["note"] = noteText
			}
			continue
		}

		// Parse direct connections (format: A -> B)
		connPattern := regexp.MustCompile(`^([^-]+?)\s*(->|-->)\s*([^:]+)(?:\s*:\s*(.*))?$`)
		if matches := connPattern.FindStringSubmatch(line); len(matches) >= 4 {
			fromName := strings.TrimSpace(matches[1])
			arrow := matches[2]
			toName := strings.TrimSpace(matches[3])
			label := ""
			if len(matches) > 4 {
				label = strings.TrimSpace(matches[4])
			}

			// Ensure nodes exist
			if _, exists := nodeMap[fromName]; !exists {
				d.Nodes = append(d.Nodes, diagram.Node{
					ID:    nextID,
					Text:  []string{fromName},
					Hints: make(map[string]string),
				})
				nodeMap[fromName] = nextID
				nextID++
			}
			if _, exists := nodeMap[toName]; !exists {
				d.Nodes = append(d.Nodes, diagram.Node{
					ID:    nextID,
					Text:  []string{toName},
					Hints: make(map[string]string),
				})
				nodeMap[toName] = nextID
				nextID++
			}

			conn := diagram.Connection{
				From:  nodeMap[fromName],
				To:    nodeMap[toName],
				Label: label,
				Hints: make(map[string]string),
			}

			// Add style hints
			if strings.Contains(arrow, "--") {
				conn.Hints["style"] = "dashed"
			}

			d.Connections = append(d.Connections, conn)
			lastNodeID = nil // Reset last node since we made explicit connection
		}
	}

	return d, nil
}
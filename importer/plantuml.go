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
			// Check for alias syntax: participant "Name" as Alias
			aliasPattern := regexp.MustCompile(`^(participant|actor)\s+"([^"]+)"\s+as\s+(\w+)`)
			if matches := aliasPattern.FindStringSubmatch(line); len(matches) == 4 {
				// Has alias - use the quoted name as display and alias as key
				displayName := matches[2]
				alias := matches[3]

				if _, exists := participantMap[alias]; !exists {
					d.Nodes = append(d.Nodes, diagram.Node{
						ID:   nextID,
						Text: []string{displayName},
					})
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
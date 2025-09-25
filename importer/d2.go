package importer

import (
	"edd/diagram"
	"fmt"
	"regexp"
	"strings"
)

// D2Importer imports D2 diagram format
type D2Importer struct{}

// NewD2Importer creates a new D2 importer
func NewD2Importer() *D2Importer {
	return &D2Importer{}
}

// CanImport checks if the content is a D2 diagram
func (d *D2Importer) CanImport(content string) bool {
	content = strings.TrimSpace(content)
	// D2 uses simple syntax like: a -> b
	// Check for D2-style connections and lack of other format markers
	hasArrows := strings.Contains(content, "->") || strings.Contains(content, "<->") || strings.Contains(content, "--")
	hasNoOtherMarkers := !strings.HasPrefix(content, "@startuml") &&
		!strings.HasPrefix(content, "graph") &&
		!strings.HasPrefix(content, "digraph") &&
		!strings.HasPrefix(content, "sequenceDiagram")

	return hasArrows && hasNoOtherMarkers
}

// Import converts D2 content to edd diagram
func (d *D2Importer) Import(content string) (*diagram.Diagram, error) {
	dia := &diagram.Diagram{
		Type: "box", // D2 typically creates flowchart-style diagrams
	}

	nodeMap := make(map[string]int)
	nextID := 0

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}

		// Parse connections
		// Patterns: a -> b, a -> b: label, a <-> b, a -- b
		connectionPattern := regexp.MustCompile(`^([^-<]+?)\s*(->|<->|--)\s*([^:]+?)(?:\s*:\s*(.*))?$`)
		if matches := connectionPattern.FindStringSubmatch(line); len(matches) >= 4 {
			fromName := strings.TrimSpace(matches[1])
			arrow := matches[2]
			toName := strings.TrimSpace(matches[3])
			label := ""
			if len(matches) > 4 {
				label = strings.TrimSpace(matches[4])
			}

			// Handle quoted names
			fromName = strings.Trim(fromName, `"'`)
			toName = strings.Trim(toName, `"'`)
			label = strings.Trim(label, `"'`)

			// Ensure nodes exist
			if _, exists := nodeMap[fromName]; !exists {
				dia.Nodes = append(dia.Nodes, diagram.Node{
					ID:   nextID,
					Text: []string{fromName},
				})
				nodeMap[fromName] = nextID
				nextID++
			}
			if _, exists := nodeMap[toName]; !exists {
				dia.Nodes = append(dia.Nodes, diagram.Node{
					ID:   nextID,
					Text: []string{toName},
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

			// Add style hints based on arrow type
			if arrow == "--" {
				conn.Hints["style"] = "dashed"
			} else if arrow == "<->" {
				conn.Hints["bidirectional"] = "true"
			}

			dia.Connections = append(dia.Connections, conn)

			// For bidirectional, add reverse connection
			if arrow == "<->" {
				dia.Connections = append(dia.Connections, diagram.Connection{
					From:  nodeMap[toName],
					To:    nodeMap[fromName],
					Label: label,
					Hints: map[string]string{"bidirectional": "true"},
				})
			}
		} else {
			// Parse node declarations with properties
			// Pattern: nodeName: label or nodeName.shape: box
			nodePattern := regexp.MustCompile(`^([^:.]+?)(?:\.shape)?\s*:\s*(.*)$`)
			if matches := nodePattern.FindStringSubmatch(line); len(matches) >= 3 {
				nodeName := strings.TrimSpace(matches[1])
				value := strings.TrimSpace(matches[2])

				nodeName = strings.Trim(nodeName, `"'`)
				value = strings.Trim(value, `"'`)

				if _, exists := nodeMap[nodeName]; !exists {
					dia.Nodes = append(dia.Nodes, diagram.Node{
						ID:   nextID,
						Text: []string{value},
						Hints: make(map[string]string),
					})
					nodeMap[nodeName] = nextID
					nextID++
				}
			}
		}
	}

	if len(dia.Nodes) == 0 {
		return nil, fmt.Errorf("no nodes found in D2 diagram")
	}

	return dia, nil
}

// GetFormatName returns the format name
func (d *D2Importer) GetFormatName() string {
	return "D2"
}

// GetFileExtensions returns common file extensions
func (d *D2Importer) GetFileExtensions() []string {
	return []string{".d2"}
}
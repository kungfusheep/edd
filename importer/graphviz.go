package importer

import (
	"edd/diagram"
	"fmt"
	"regexp"
	"strings"
)

// GraphvizImporter imports Graphviz DOT format
type GraphvizImporter struct{}

// NewGraphvizImporter creates a new Graphviz importer
func NewGraphvizImporter() *GraphvizImporter {
	return &GraphvizImporter{}
}

// CanImport checks if the content is a Graphviz DOT diagram
func (g *GraphvizImporter) CanImport(content string) bool {
	content = strings.TrimSpace(content)
	return strings.HasPrefix(content, "digraph") ||
		strings.HasPrefix(content, "graph") ||
		strings.Contains(content, "->") && strings.Contains(content, "{") && strings.Contains(content, "}")
}

// Import converts Graphviz DOT content to edd diagram
func (g *GraphvizImporter) Import(content string) (*diagram.Diagram, error) {
	d := &diagram.Diagram{
		Type: "box", // Graphviz is typically used for flowchart-style diagrams
	}

	nodeMap := make(map[string]int)
	nextID := 0

	// Simple DOT parser - this is a basic implementation
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") || line == "{" || line == "}" {
			continue
		}

		// Skip graph declaration
		if strings.HasPrefix(line, "digraph") || strings.HasPrefix(line, "graph") {
			continue
		}

		// Parse edges (connections)
		// Pattern: A -> B [label="text"];
		edgePattern := regexp.MustCompile(`^\s*"?([^"]+)"?\s*->\s*"?([^"]+)"?\s*(?:\[label="([^"]*)"\])?;?$`)
		if matches := edgePattern.FindStringSubmatch(line); len(matches) >= 3 {
			fromName := strings.TrimSpace(matches[1])
			toName := strings.TrimSpace(matches[2])
			label := ""
			if len(matches) > 3 {
				label = matches[3]
			}

			// Ensure nodes exist
			if _, exists := nodeMap[fromName]; !exists {
				d.Nodes = append(d.Nodes, diagram.Node{
					ID:   nextID,
					Text: []string{fromName},
				})
				nodeMap[fromName] = nextID
				nextID++
			}
			if _, exists := nodeMap[toName]; !exists {
				d.Nodes = append(d.Nodes, diagram.Node{
					ID:   nextID,
					Text: []string{toName},
				})
				nodeMap[toName] = nextID
				nextID++
			}

			d.Connections = append(d.Connections, diagram.Connection{
				From:  nodeMap[fromName],
				To:    nodeMap[toName],
				Label: label,
				Hints: make(map[string]string),
			})
		} else {
			// Parse node declarations
			// Pattern: A [label="text"];
			nodePattern := regexp.MustCompile(`^\s*"?([^"]+)"?\s*\[label="([^"]*)"\];?$`)
			if matches := nodePattern.FindStringSubmatch(line); len(matches) >= 3 {
				nodeName := strings.TrimSpace(matches[1])
				nodeLabel := matches[2]

				if _, exists := nodeMap[nodeName]; !exists {
					d.Nodes = append(d.Nodes, diagram.Node{
						ID:   nextID,
						Text: []string{nodeLabel},
					})
					nodeMap[nodeName] = nextID
					nextID++
				}
			}
		}
	}

	if len(d.Nodes) == 0 {
		return nil, fmt.Errorf("no nodes found in Graphviz diagram")
	}

	return d, nil
}

// GetFormatName returns the format name
func (g *GraphvizImporter) GetFormatName() string {
	return "Graphviz"
}

// GetFileExtensions returns common file extensions
func (g *GraphvizImporter) GetFileExtensions() []string {
	return []string{".dot", ".gv"}
}
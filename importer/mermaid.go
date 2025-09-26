package importer

import (
	"edd/diagram"
	"fmt"
	"regexp"
	"strings"
)

// MermaidImporter imports Mermaid diagram format
type MermaidImporter struct{}

// NewMermaidImporter creates a new Mermaid importer
func NewMermaidImporter() *MermaidImporter {
	return &MermaidImporter{}
}

// CanImport checks if the content is a Mermaid diagram
func (m *MermaidImporter) CanImport(content string) bool {
	content = strings.TrimSpace(content)
	// Check for common Mermaid diagram types
	return strings.HasPrefix(content, "graph ") ||
		strings.HasPrefix(content, "flowchart ") ||
		strings.HasPrefix(content, "sequenceDiagram") ||
		strings.Contains(content, "graph LR") ||
		strings.Contains(content, "graph TD") ||
		strings.Contains(content, "graph TB") ||
		strings.Contains(content, "graph RL") ||
		strings.Contains(content, "graph BT")
}

// Import converts Mermaid content to edd diagram
func (m *MermaidImporter) Import(content string) (*diagram.Diagram, error) {
	content = strings.TrimSpace(content)

	// Determine diagram type
	if strings.HasPrefix(content, "sequenceDiagram") {
		return m.importSequenceDiagram(content)
	} else if strings.HasPrefix(content, "graph") || strings.HasPrefix(content, "flowchart") {
		return m.importFlowchart(content)
	}

	return nil, fmt.Errorf("unsupported Mermaid diagram type")
}

// GetFormatName returns the format name
func (m *MermaidImporter) GetFormatName() string {
	return "Mermaid"
}

// GetFileExtensions returns common file extensions
func (m *MermaidImporter) GetFileExtensions() []string {
	return []string{".mmd", ".mermaid"}
}

// importSequenceDiagram imports a Mermaid sequence diagram
func (m *MermaidImporter) importSequenceDiagram(content string) (*diagram.Diagram, error) {
	d := &diagram.Diagram{
		Type: "sequence",
	}

	// Map to track participant names to IDs
	participantMap := make(map[string]int)
	nextID := 0

	lines := strings.Split(content, "\n")
	for _, line := range lines[1:] { // Skip the "sequenceDiagram" line
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "%%") {
			continue // Skip empty lines and comments
		}

		// Parse participant/actor declarations
		if strings.HasPrefix(line, "participant ") || strings.HasPrefix(line, "actor ") {
			nodeType := "participant"
			decl := line
			if strings.HasPrefix(line, "actor ") {
				nodeType = "actor"
				decl = strings.TrimPrefix(line, "actor ")
			} else {
				decl = strings.TrimPrefix(line, "participant ")
			}
			decl = strings.TrimSpace(decl)

			// Handle "participant ID as Display Name" syntax
			var id, displayName string
			if parts := strings.SplitN(decl, " as ", 2); len(parts) == 2 {
				id = strings.TrimSpace(parts[0])
				displayName = strings.TrimSpace(parts[1])
			} else {
				// No "as" clause, use the whole thing as both ID and display name
				id = decl
				displayName = decl
			}

			if _, exists := participantMap[id]; !exists {
				node := diagram.Node{
					ID:    nextID,
					Text:  []string{displayName},
					Hints: make(map[string]string),
				}
				node.Hints["node-type"] = nodeType
				d.Nodes = append(d.Nodes, node)
				participantMap[id] = nextID
				nextID++
			}
		} else {
			// Parse messages (connections)
			// Common patterns: A->>B: Message, A-->>B: Message, A-xB: Message, etc.
			messagePattern := regexp.MustCompile(`^([^-]+?)(->>?|-->>?|-x|--x|->>\+|-->\+)([^:]+):(.*)$`)
			matches := messagePattern.FindStringSubmatch(line)
			if len(matches) == 5 {
				fromName := strings.TrimSpace(matches[1])
				// arrowType := matches[2] // Could use this for hints later
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

				// Add connection
				conn := diagram.Connection{
					From:  participantMap[fromName],
					To:    participantMap[toName],
					Label: label,
					Hints: make(map[string]string),
				}

				// Add hints based on arrow type
				if strings.Contains(matches[2], "--") {
					conn.Hints["style"] = "dashed"
				}
				if strings.Contains(matches[2], "x") {
					conn.Hints["style"] = "crossed"
				}

				d.Connections = append(d.Connections, conn)
			}
		}
	}

	return d, nil
}

// importFlowchart imports a Mermaid flowchart/graph
func (m *MermaidImporter) importFlowchart(content string) (*diagram.Diagram, error) {
	d := &diagram.Diagram{
		Type: "box",
	}

	// Map to track node names to IDs
	nodeMap := make(map[string]int)
	nodeIndexMap := make(map[string]int) // Track which node in the array
	nextID := 0

	// Enhanced node pattern to capture different shapes
	// Matches: ID[text], ID(text), ID{text}, ID{{text}}, ID[[text]], ID[(text)], ID([text])
	nodePattern := regexp.MustCompile(`([A-Za-z0-9_]+)(\[("[^"]*"|[^\]]*)\]|\(("[^"]*"|[^\)]*)\)|\{("[^"]*"|[^\}]*)\}|\{\{("[^"]*"|[^\}]*)\}\}|\[\[("[^"]*"|[^\]]*)\]\]|\[\(("[^"]*"|[^\)]*)\)\]|\(\[("[^"]*"|[^\]]*)\]\)|\>("[^"]*"|[^\]]*)\]|\(\(("[^"]*"|[^\)]*)\)\))`)

	// Extract connections
	connectionPattern := regexp.MustCompile(`([A-Za-z0-9_]+)\s*(-->?|-.->|==>|--[^>]*>|<-->|o--o|x--x)\s*(?:\|([^|]*)\|)?\s*([A-Za-z0-9_]+)`)

	// Pattern for subgraphs
	subgraphPattern := regexp.MustCompile(`^\s*subgraph\s+([A-Za-z0-9_]+)\s*\[?([^\]]*)\]?`)

	// Pattern for notes
	notePattern := regexp.MustCompile(`^\s*([A-Za-z0-9_]+)\s*~~~\s*(.*)`)

	// Track current subgraph
	currentSubgraph := ""

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}

		// Skip the graph/flowchart declaration line
		if strings.HasPrefix(line, "graph") || strings.HasPrefix(line, "flowchart") {
			continue
		}

		// Check for subgraph start
		if matches := subgraphPattern.FindStringSubmatch(line); matches != nil {
			currentSubgraph = matches[1]
			continue
		}

		// Check for end of subgraph
		if line == "end" {
			currentSubgraph = ""
			continue
		}

		// Check for notes
		if matches := notePattern.FindStringSubmatch(line); matches != nil {
			nodeID := matches[1]
			noteText := matches[2]

			if idx, exists := nodeMap[nodeID]; exists {
				// Add note to existing node's hints
				nodeIndex := nodeIndexMap[nodeID]
				if d.Nodes[nodeIndex].Hints == nil {
					d.Nodes[nodeIndex].Hints = make(map[string]string)
				}
				d.Nodes[nodeIndex].Hints["note"] = noteText
				_ = idx // use idx to avoid unused variable
			}
			continue
		}

		// Check for node declarations
		if matches := nodePattern.FindAllStringSubmatch(line, -1); matches != nil {
			for _, match := range matches {
				nodeID := match[1]
				fullShape := match[2]

				// Extract text from shape
				nodeText := fullShape
				// Remove outer brackets/parens and trim quotes
				nodeText = strings.Trim(nodeText, "[](){}<>")
				nodeText = strings.Trim(nodeText, `"`)

				if _, exists := nodeMap[nodeID]; !exists {
					node := diagram.Node{
						ID:    nextID,
						Text:  []string{nodeText},
						Hints: make(map[string]string),
					}

					// Determine shape based on bracket style
					if strings.HasPrefix(fullShape, "[[") {
						node.Hints["shape"] = "double"
						node.Hints["box-style"] = "double"
					} else if strings.HasPrefix(fullShape, "((") {
						node.Hints["shape"] = "circle"
					} else if strings.HasPrefix(fullShape, "(") {
						node.Hints["shape"] = "rounded"
						node.Hints["style"] = "rounded"
					} else if strings.HasPrefix(fullShape, "{") {
						if strings.HasPrefix(fullShape, "{{") {
							node.Hints["shape"] = "hexagon"
						} else {
							node.Hints["shape"] = "diamond"
						}
					} else if strings.HasPrefix(fullShape, ">") {
						node.Hints["shape"] = "trapezoid"
					} else if strings.HasPrefix(fullShape, "[(") {
						node.Hints["shape"] = "cylinder"
					} else if strings.HasPrefix(fullShape, "([") {
						node.Hints["shape"] = "parallelogram"
					}
					// Default [text] is just a rectangle, no special hint needed

					// Add subgraph/group info if in one
					if currentSubgraph != "" {
						node.Hints["group"] = currentSubgraph
					}

					d.Nodes = append(d.Nodes, node)
					nodeMap[nodeID] = nextID
					nodeIndexMap[nodeID] = len(d.Nodes) - 1
					nextID++
				}
			}
		}

		// Check for connections
		if matches := connectionPattern.FindStringSubmatch(line); len(matches) >= 5 {
			fromID := matches[1]
			arrow := matches[2]
			label := matches[3]
			toID := matches[4]

			// Ensure nodes exist
			if _, exists := nodeMap[fromID]; !exists {
				node := diagram.Node{
					ID:    nextID,
					Text:  []string{fromID},
					Hints: make(map[string]string),
				}
				if currentSubgraph != "" {
					node.Hints["group"] = currentSubgraph
				}
				d.Nodes = append(d.Nodes, node)
				nodeMap[fromID] = nextID
				nodeIndexMap[fromID] = len(d.Nodes) - 1
				nextID++
			}
			if _, exists := nodeMap[toID]; !exists {
				node := diagram.Node{
					ID:    nextID,
					Text:  []string{toID},
					Hints: make(map[string]string),
				}
				if currentSubgraph != "" {
					node.Hints["group"] = currentSubgraph
				}
				d.Nodes = append(d.Nodes, node)
				nodeMap[toID] = nextID
				nodeIndexMap[toID] = len(d.Nodes) - 1
				nextID++
			}

			// Create connection
			conn := diagram.Connection{
				From:  nodeMap[fromID],
				To:    nodeMap[toID],
				Label: label,
				Hints: make(map[string]string),
			}

			// Enhanced arrow style detection
			if strings.Contains(arrow, "-.") {
				conn.Hints["style"] = "dashed"
			} else if strings.Contains(arrow, "==") {
				conn.Hints["style"] = "thick"
			}

			// Check for special arrow types
			if strings.Contains(arrow, "<-->") {
				conn.Hints["bidirectional"] = "true"
			} else if strings.Contains(arrow, "o--o") {
				conn.Hints["arrow-type"] = "circle"
			} else if strings.Contains(arrow, "x--x") {
				conn.Hints["arrow-type"] = "cross"
			}

			d.Connections = append(d.Connections, conn)
		}
	}

	return d, nil
}
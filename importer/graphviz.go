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
	nodeIndexMap := make(map[string]int)
	nextID := 0
	currentSubgraph := ""

	// Simple DOT parser - enhanced implementation
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// Skip graph declaration and global attributes
		if strings.HasPrefix(line, "digraph") || strings.HasPrefix(line, "graph") {
			continue
		}
		if strings.HasPrefix(line, "rankdir") || strings.HasPrefix(line, "node ") || strings.HasPrefix(line, "edge ") {
			continue
		}

		// Check for subgraph
		if strings.HasPrefix(line, "subgraph") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentSubgraph = strings.Trim(parts[1], "{")
			}
			continue
		}

		// Check for end of subgraph
		if line == "}" && currentSubgraph != "" {
			currentSubgraph = ""
			continue
		}

		// Skip single braces
		if line == "{" || line == "}" {
			continue
		}

		// Enhanced edge pattern to capture attributes
		// Pattern: A -> B [attr1=val1, attr2="val2"];
		edgePattern := regexp.MustCompile(`^\s*"?([^"\[]+)"?\s*->\s*"?([^"\[]+)"?\s*(?:\[([^\]]*)\])?;?$`)
		if matches := edgePattern.FindStringSubmatch(line); len(matches) >= 3 {
			fromName := strings.TrimSpace(matches[1])
			toName := strings.TrimSpace(matches[2])
			attributes := ""
			if len(matches) > 3 {
				attributes = matches[3]
			}

			// Ensure nodes exist
			if _, exists := nodeMap[fromName]; !exists {
				node := diagram.Node{
					ID:    nextID,
					Text:  []string{fromName},
					Hints: make(map[string]string),
				}
				if currentSubgraph != "" {
					node.Hints["group"] = currentSubgraph
				}
				d.Nodes = append(d.Nodes, node)
				nodeMap[fromName] = nextID
				nodeIndexMap[fromName] = len(d.Nodes) - 1
				nextID++
			}
			if _, exists := nodeMap[toName]; !exists {
				node := diagram.Node{
					ID:    nextID,
					Text:  []string{toName},
					Hints: make(map[string]string),
				}
				if currentSubgraph != "" {
					node.Hints["group"] = currentSubgraph
				}
				d.Nodes = append(d.Nodes, node)
				nodeMap[toName] = nextID
				nodeIndexMap[toName] = len(d.Nodes) - 1
				nextID++
			}

			conn := diagram.Connection{
				From:  nodeMap[fromName],
				To:    nodeMap[toName],
				Hints: make(map[string]string),
			}

			// Parse edge attributes
			if attributes != "" {
				attrs := g.parseAttributes(attributes)
				if label, ok := attrs["label"]; ok {
					conn.Label = label
				}
				if style, ok := attrs["style"]; ok {
					conn.Hints["style"] = style
				}
				if color, ok := attrs["color"]; ok {
					conn.Hints["color"] = g.normalizeColor(color)
				}
				if dir, ok := attrs["dir"]; ok && dir == "both" {
					conn.Hints["bidirectional"] = "true"
				}
			}

			d.Connections = append(d.Connections, conn)
		} else {
			// Enhanced node pattern to capture all attributes
			// Pattern: A [attr1=val1, attr2="val2"];
			nodePattern := regexp.MustCompile(`^\s*"?([^"\[]+)"?\s*\[([^\]]*)\];?$`)
			if matches := nodePattern.FindStringSubmatch(line); len(matches) >= 3 {
				nodeName := strings.TrimSpace(matches[1])
				attributes := matches[2]

				if _, exists := nodeMap[nodeName]; !exists {
					node := diagram.Node{
						ID:    nextID,
						Text:  []string{nodeName}, // Default to node name
						Hints: make(map[string]string),
					}

					// Parse node attributes
					attrs := g.parseAttributes(attributes)
					if label, ok := attrs["label"]; ok {
						// Handle multiline labels
						label = strings.ReplaceAll(label, "\\n", "\n")
						node.Text = strings.Split(label, "\n")
					}
					if shape, ok := attrs["shape"]; ok {
						node.Hints["shape"] = g.normalizeShape(shape)
					}
					if style, ok := attrs["style"]; ok {
						if strings.Contains(style, "rounded") {
							node.Hints["style"] = "rounded"
						} else if strings.Contains(style, "dashed") {
							node.Hints["style"] = "dashed"
						} else if strings.Contains(style, "bold") {
							node.Hints["bold"] = "true"
						} else if strings.Contains(style, "filled") {
							// Will be used with fillcolor
						}
					}
					if fillcolor, ok := attrs["fillcolor"]; ok {
						node.Hints["color"] = g.normalizeColor(fillcolor)
					} else if color, ok := attrs["color"]; ok {
						// If no fillcolor, use color
						node.Hints["color"] = g.normalizeColor(color)
					}
					if peripheries, ok := attrs["peripheries"]; ok && peripheries == "2" {
						node.Hints["box-style"] = "double"
					}

					if currentSubgraph != "" {
						node.Hints["group"] = currentSubgraph
					}

					d.Nodes = append(d.Nodes, node)
					nodeMap[nodeName] = nextID
					nodeIndexMap[nodeName] = len(d.Nodes) - 1
					nextID++
				} else {
					// Update existing node with new attributes
					nodeIndex := nodeIndexMap[nodeName]
					attrs := g.parseAttributes(attributes)

					if label, ok := attrs["label"]; ok {
						label = strings.ReplaceAll(label, "\\n", "\n")
						d.Nodes[nodeIndex].Text = strings.Split(label, "\n")
					}
					if shape, ok := attrs["shape"]; ok {
						d.Nodes[nodeIndex].Hints["shape"] = g.normalizeShape(shape)
					}
					if fillcolor, ok := attrs["fillcolor"]; ok {
						d.Nodes[nodeIndex].Hints["color"] = g.normalizeColor(fillcolor)
					}
				}
			}
		}
	}

	if len(d.Nodes) == 0 {
		return nil, fmt.Errorf("no nodes found in Graphviz diagram")
	}

	return d, nil
}

// parseAttributes parses DOT attribute string into a map
func (g *GraphvizImporter) parseAttributes(attrStr string) map[string]string {
	attrs := make(map[string]string)

	// Simple attribute parser - handles key=value and key="value"
	// This is a simplified parser that handles common cases
	attrPattern := regexp.MustCompile(`(\w+)\s*=\s*("([^"]*)"|([^,\s]+))`)
	matches := attrPattern.FindAllStringSubmatch(attrStr, -1)

	for _, match := range matches {
		key := match[1]
		value := match[3] // Quoted value
		if value == "" {
			value = match[4] // Unquoted value
		}
		attrs[key] = value
	}

	return attrs
}

// normalizeShape converts Graphviz shape names to our internal format
func (g *GraphvizImporter) normalizeShape(shape string) string {
	shapeMap := map[string]string{
		"box":           "rectangle",
		"rect":          "rectangle",
		"rectangle":     "rectangle",
		"circle":        "circle",
		"ellipse":       "ellipse",
		"diamond":       "diamond",
		"hexagon":       "hexagon",
		"parallelogram": "parallelogram",
		"cylinder":      "cylinder",
		"doublecircle":  "double",
		"doubleoctagon": "double",
		"trapezium":     "trapezoid",
	}

	if normalized, ok := shapeMap[strings.ToLower(shape)]; ok {
		return normalized
	}
	return shape
}

// normalizeColor converts Graphviz color formats to our internal format
func (g *GraphvizImporter) normalizeColor(color string) string {
	// Remove # prefix if present
	color = strings.TrimPrefix(color, "#")

	// Map common color names
	colorMap := map[string]string{
		"red":     "red",
		"green":   "green",
		"blue":    "blue",
		"yellow":  "yellow",
		"cyan":    "cyan",
		"magenta": "magenta",
		"black":   "black",
		"white":   "white",
		"gray":    "gray",
		"grey":    "gray",
	}

	if normalized, ok := colorMap[strings.ToLower(color)]; ok {
		return normalized
	}

	// If it looks like a hex code, keep it
	if len(color) == 6 {
		return color
	}

	return color
}

// GetFormatName returns the format name
func (g *GraphvizImporter) GetFormatName() string {
	return "Graphviz"
}

// GetFileExtensions returns common file extensions
func (g *GraphvizImporter) GetFileExtensions() []string {
	return []string{".dot", ".gv"}
}
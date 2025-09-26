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
	hasArrows := strings.Contains(content, "->") || strings.Contains(content, "<->") || strings.Contains(content, "--") || strings.Contains(content, "=>")
	hasNoOtherMarkers := !strings.HasPrefix(content, "@startuml") &&
		!strings.HasPrefix(content, "graph") &&
		!strings.HasPrefix(content, "digraph") &&
		!strings.HasPrefix(content, "sequenceDiagram")

	// Also check for D2-specific syntax
	hasD2Syntax := strings.Contains(content, ".shape:") || strings.Contains(content, ".style.") ||
		strings.Contains(content, ": {") || strings.Contains(content, ".near:")

	return (hasArrows || hasD2Syntax) && hasNoOtherMarkers
}

// Import converts D2 content to edd diagram
func (d *D2Importer) Import(content string) (*diagram.Diagram, error) {
	dia := &diagram.Diagram{
		Type: "box", // D2 typically creates flowchart-style diagrams
	}

	nodeMap := make(map[string]int)
	nodeIndexMap := make(map[string]int)
	nextID := 0
	currentContainer := ""
	containerStack := []string{}
	_ = 0 // indentLevel for future use

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		originalLine := line
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}

		// Calculate indent level to track nesting (for future use)
		_ = len(originalLine) - len(strings.TrimLeft(originalLine, " \t"))

		// Check for container/group start (nested structures)
		if strings.Contains(line, ": {") {
			parts := strings.Split(line, ": {")
			if len(parts) >= 1 {
				containerName := strings.TrimSpace(parts[0])
				containerStack = append(containerStack, containerName)
				currentContainer = strings.Join(containerStack, ".")
			}
			continue
		}

		// Check for container end
		if line == "}" && len(containerStack) > 0 {
			containerStack = containerStack[:len(containerStack)-1]
			if len(containerStack) > 0 {
				currentContainer = strings.Join(containerStack, ".")
			} else {
				currentContainer = ""
			}
			continue
		}

		// Parse connections
		// Enhanced patterns: a -> b, a => b, a -- b, a <-> b
		connectionPattern := regexp.MustCompile(`^([^-<=]+?)\s*(->|=>|<->|--)\s*([^:]+?)(?:\s*:\s*(.*))?$`)
		if matches := connectionPattern.FindStringSubmatch(line); len(matches) >= 4 {
			fromName := strings.TrimSpace(matches[1])
			arrow := matches[2]
			toName := strings.TrimSpace(matches[3])
			label := ""
			if len(matches) > 4 {
				label = strings.TrimSpace(matches[4])
			}

			// Handle quoted names
			fromName = d.unquote(fromName)
			toName = d.unquote(toName)
			label = d.unquote(label)

			// Add container prefix if in a container
			if currentContainer != "" {
				if !strings.Contains(fromName, ".") {
					fromName = currentContainer + "." + fromName
				}
				if !strings.Contains(toName, ".") {
					toName = currentContainer + "." + toName
				}
			}

			// Ensure nodes exist
			if _, exists := nodeMap[fromName]; !exists {
				node := diagram.Node{
					ID:    nextID,
					Text:  []string{d.getDisplayName(fromName)},
					Hints: make(map[string]string),
				}
				if currentContainer != "" {
					node.Hints["group"] = currentContainer
				}
				// Check if it's a nested node
				if strings.Contains(fromName, ".") {
					parts := strings.Split(fromName, ".")
					node.Hints["parent"] = strings.Join(parts[:len(parts)-1], ".")
				}
				dia.Nodes = append(dia.Nodes, node)
				nodeMap[fromName] = nextID
				nodeIndexMap[fromName] = len(dia.Nodes) - 1
				nextID++
			}
			if _, exists := nodeMap[toName]; !exists {
				node := diagram.Node{
					ID:    nextID,
					Text:  []string{d.getDisplayName(toName)},
					Hints: make(map[string]string),
				}
				if currentContainer != "" {
					node.Hints["group"] = currentContainer
				}
				if strings.Contains(toName, ".") {
					parts := strings.Split(toName, ".")
					node.Hints["parent"] = strings.Join(parts[:len(parts)-1], ".")
				}
				dia.Nodes = append(dia.Nodes, node)
				nodeMap[toName] = nextID
				nodeIndexMap[toName] = len(dia.Nodes) - 1
				nextID++
			}

			conn := diagram.Connection{
				From:  nodeMap[fromName],
				To:    nodeMap[toName],
				Label: label,
				Hints: make(map[string]string),
			}

			// Add style hints based on arrow type
			switch arrow {
			case "--":
				conn.Hints["style"] = "dashed"
			case "=>":
				conn.Hints["style"] = "thick"
			case "<->":
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
		} else if strings.Contains(line, ".shape:") {
			// Parse shape properties
			// Pattern: nodeName.shape: shapeName
			shapePattern := regexp.MustCompile(`^([^.]+)\.shape:\s*(.*)$`)
			if matches := shapePattern.FindStringSubmatch(line); len(matches) >= 3 {
				nodeName := strings.TrimSpace(matches[1])
				shape := d.unquote(strings.TrimSpace(matches[2]))

				if currentContainer != "" && !strings.Contains(nodeName, ".") {
					nodeName = currentContainer + "." + nodeName
				}

				if idx, exists := nodeMap[nodeName]; exists {
					nodeIndex := nodeIndexMap[nodeName]
					dia.Nodes[nodeIndex].Hints["shape"] = d.normalizeShape(shape)
					_ = idx // use idx to avoid unused variable
				}
			}
		} else if strings.Contains(line, ".style.") {
			// Parse style properties
			// Patterns: nodeName.style.fill: color, nodeName.style.stroke-width: 3
			stylePattern := regexp.MustCompile(`^([^.]+)\.style\.([^:]+):\s*(.*)$`)
			if matches := stylePattern.FindStringSubmatch(line); len(matches) >= 4 {
				nodeName := strings.TrimSpace(matches[1])
				styleProp := strings.TrimSpace(matches[2])
				styleValue := d.unquote(strings.TrimSpace(matches[3]))

				if currentContainer != "" && !strings.Contains(nodeName, ".") {
					nodeName = currentContainer + "." + nodeName
				}

				if idx, exists := nodeMap[nodeName]; exists {
					nodeIndex := nodeIndexMap[nodeName]
					switch styleProp {
					case "fill":
						dia.Nodes[nodeIndex].Hints["color"] = d.normalizeColor(styleValue)
					case "stroke":
						dia.Nodes[nodeIndex].Hints["border-color"] = d.normalizeColor(styleValue)
					case "stroke-width":
						if styleValue == "3" || styleValue == "2" {
							dia.Nodes[nodeIndex].Hints["style"] = "thick"
						}
					case "stroke-dash":
						dia.Nodes[nodeIndex].Hints["style"] = "dashed"
					case "bold":
						if styleValue == "true" {
							dia.Nodes[nodeIndex].Hints["bold"] = "true"
						}
					case "italic":
						if styleValue == "true" {
							dia.Nodes[nodeIndex].Hints["italic"] = "true"
						}
					case "shadow":
						if styleValue == "true" {
							dia.Nodes[nodeIndex].Hints["shadow"] = "southeast"
						}
					}
					_ = idx // use idx to avoid unused variable
				}
			}
		} else {
			// Parse node declarations
			// Pattern: nodeName: label
			nodePattern := regexp.MustCompile(`^([^:.]+?)(?:\.multiple)?\s*:\s*(.*)$`)
			if matches := nodePattern.FindStringSubmatch(line); len(matches) >= 3 {
				nodeName := strings.TrimSpace(matches[1])
				value := strings.TrimSpace(matches[2])

				// Skip if this looks like it's starting a nested structure
				if value == "{" || strings.HasSuffix(value, "{") {
					// containerName would be: nodeName or currentContainer + "." + nodeName
					containerStack = append(containerStack, nodeName)
					currentContainer = strings.Join(containerStack, ".")
					continue
				}

				nodeName = d.unquote(nodeName)
				value = d.unquote(value)

				// Handle multiline labels
				value = strings.ReplaceAll(value, "\\n", "\n")

				if currentContainer != "" && !strings.Contains(nodeName, ".") {
					nodeName = currentContainer + "." + nodeName
				}

				if _, exists := nodeMap[nodeName]; !exists {
					node := diagram.Node{
						ID:    nextID,
						Text:  strings.Split(value, "\n"),
						Hints: make(map[string]string),
					}
					if currentContainer != "" {
						node.Hints["group"] = currentContainer
					}
					if strings.Contains(nodeName, ".") {
						parts := strings.Split(nodeName, ".")
						node.Hints["parent"] = strings.Join(parts[:len(parts)-1], ".")
					}
					// Check for .multiple suffix
					if strings.Contains(line, ".multiple") {
						node.Hints["box-style"] = "double"
					}
					dia.Nodes = append(dia.Nodes, node)
					nodeMap[nodeName] = nextID
					nodeIndexMap[nodeName] = len(dia.Nodes) - 1
					nextID++
				} else {
					// Update existing node's text
					nodeIndex := nodeIndexMap[nodeName]
					dia.Nodes[nodeIndex].Text = strings.Split(value, "\n")
				}
			}
		}

		// Check for connection style attributes (after the connection)
		// Pattern: (node1 -> node2)[0].style.stroke: color
		if i > 0 && strings.Contains(line, ")[") && strings.Contains(line, ".style.") {
			// This is a connection style, parse and apply to last connection
			if len(dia.Connections) > 0 {
				stylePattern := regexp.MustCompile(`\.style\.([^:]+):\s*(.*)$`)
				if matches := stylePattern.FindStringSubmatch(line); len(matches) >= 3 {
					styleProp := strings.TrimSpace(matches[1])
					styleValue := d.unquote(strings.TrimSpace(matches[2]))

					lastConn := &dia.Connections[len(dia.Connections)-1]
					switch styleProp {
					case "stroke":
						lastConn.Hints["color"] = d.normalizeColor(styleValue)
					case "stroke-width":
						if styleValue == "3" || styleValue == "2" {
							lastConn.Hints["style"] = "thick"
						}
					case "stroke-dash":
						lastConn.Hints["style"] = "dashed"
					case "bold":
						if styleValue == "true" {
							lastConn.Hints["bold"] = "true"
						}
					}
				}
			}
		}
	}

	if len(dia.Nodes) == 0 {
		return nil, fmt.Errorf("no nodes found in D2 diagram")
	}

	return dia, nil
}

// unquote removes quotes from a string
func (d *D2Importer) unquote(s string) string {
	s = strings.TrimSpace(s)
	if (strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`)) ||
		(strings.HasPrefix(s, `'`) && strings.HasSuffix(s, `'`)) {
		return s[1 : len(s)-1]
	}
	return s
}

// getDisplayName extracts the last part of a dotted name for display
func (d *D2Importer) getDisplayName(fullName string) string {
	parts := strings.Split(fullName, ".")
	return parts[len(parts)-1]
}

// normalizeShape converts D2 shape names to our internal format
func (d *D2Importer) normalizeShape(shape string) string {
	shapeMap := map[string]string{
		"rectangle":     "rectangle",
		"square":        "rectangle",
		"circle":        "circle",
		"diamond":       "diamond",
		"oval":          "ellipse",
		"hexagon":       "hexagon",
		"cylinder":      "cylinder",
		"cloud":         "cloud",
		"document":      "document",
		"parallelogram": "parallelogram",
		"trapezoid":     "trapezoid",
		"package":       "package",
		"step":          "step",
		"callout":       "callout",
		"stored_data":   "cylinder",
	}

	if normalized, ok := shapeMap[strings.ToLower(shape)]; ok {
		return normalized
	}
	return shape
}

// normalizeColor converts D2 color formats to our internal format
func (d *D2Importer) normalizeColor(color string) string {
	// Remove # prefix if present
	color = strings.TrimPrefix(color, "#")
	color = strings.Trim(color, `"`)

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
	if len(color) == 6 || len(color) == 3 {
		return color
	}

	return color
}

// GetFormatName returns the format name
func (d *D2Importer) GetFormatName() string {
	return "D2"
}

// GetFileExtensions returns common file extensions
func (d *D2Importer) GetFileExtensions() []string {
	return []string{".d2"}
}
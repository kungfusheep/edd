package obstacles

import (
	"edd/core"
	"fmt"
	"strings"
)

// DebugVisualizer provides visualization of obstacles and ports
type DebugVisualizer struct {
	ShowVirtualObstacles bool
	ShowOccupiedPorts    bool
	ShowAvailablePorts   bool
	ShowPortCorridors    bool
}

// VisualizeObstacles renders obstacle zones as ASCII art
func (dv *DebugVisualizer) VisualizeObstacles(bounds core.Bounds, zones []ObstacleZone, ports []Port) string {
	// Create a 2D grid
	width := bounds.Max.X - bounds.Min.X + 1
	height := bounds.Max.Y - bounds.Min.Y + 1
	
	grid := make([][]rune, height)
	for i := range grid {
		grid[i] = make([]rune, width)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}
	
	// Mark obstacle zones
	for _, zone := range zones {
		char := dv.getObstacleChar(zone.Type)
		if !dv.shouldShowZone(zone.Type) {
			continue
		}
		
		for y := zone.MinY; y <= zone.MaxY; y++ {
			for x := zone.MinX; x <= zone.MaxX; x++ {
				if x >= bounds.Min.X && x <= bounds.Max.X &&
				   y >= bounds.Min.Y && y <= bounds.Max.Y {
					gridY := y - bounds.Min.Y
					gridX := x - bounds.Min.X
					if grid[gridY][gridX] == ' ' {
						grid[gridY][gridX] = char
					}
				}
			}
		}
	}
	
	// Mark ports
	for _, port := range ports {
		if port.ConnectionID != -1 && dv.ShowOccupiedPorts {
			dv.markPort(grid, port, bounds, '●')
		} else if port.ConnectionID == -1 && dv.ShowAvailablePorts {
			dv.markPort(grid, port, bounds, '○')
		}
	}
	
	// Convert grid to string
	var result strings.Builder
	for _, row := range grid {
		result.WriteString(string(row))
		result.WriteString("\n")
	}
	
	return result.String()
}

// GetLegend returns a legend explaining the visualization symbols
func (dv *DebugVisualizer) GetLegend() string {
	legend := []string{
		"Obstacle Visualization Legend:",
		"  █ - Physical obstacle (node body)",
		"  ░ - Virtual obstacle zone",
		"  ▒ - Port corridor obstacle",
		"  ● - Occupied port",
		"  ○ - Available port",
	}
	return strings.Join(legend, "\n")
}

// shouldShowZone determines if a zone type should be displayed
func (dv *DebugVisualizer) shouldShowZone(zoneType string) bool {
	switch zoneType {
	case "physical":
		return true
	case "virtual":
		return dv.ShowVirtualObstacles
	case "port":
		return dv.ShowPortCorridors
	default:
		return false
	}
}

// getObstacleChar returns the character to use for each obstacle type
func (dv *DebugVisualizer) getObstacleChar(zoneType string) rune {
	switch zoneType {
	case "physical":
		return '█'
	case "virtual":
		return '░'
	case "port":
		return '▒'
	default:
		return '?'
	}
}

// markPort marks a port on the grid
func (dv *DebugVisualizer) markPort(grid [][]rune, port Port, bounds core.Bounds, char rune) {
	x := port.Point.X
	y := port.Point.Y
	
	if x >= bounds.Min.X && x <= bounds.Max.X &&
	   y >= bounds.Min.Y && y <= bounds.Max.Y {
		gridY := y - bounds.Min.Y
		gridX := x - bounds.Min.X
		grid[gridY][gridX] = char
	}
}

// ExportObstacleData exports obstacle data for external visualization
func ExportObstacleData(zones []ObstacleZone, ports []Port) string {
	var result strings.Builder
	
	result.WriteString("# Obstacle Zones\n")
	for i, zone := range zones {
		result.WriteString(fmt.Sprintf("Zone %d: type=%s, node=%d, bounds=(%d,%d)-(%d,%d)\n",
			i, zone.Type, zone.NodeID, zone.MinX, zone.MinY, zone.MaxX, zone.MaxY))
	}
	
	result.WriteString("\n# Ports\n")
	for i, port := range ports {
		status := "free"
		if port.ConnectionID != -1 {
			status = fmt.Sprintf("occupied by conn %d", port.ConnectionID)
		}
		result.WriteString(fmt.Sprintf("Port %d: node=%d, edge=%s, pos=%d, %s\n",
			i, port.NodeID, edgeName(port.Edge), port.Position, status))
	}
	
	return result.String()
}
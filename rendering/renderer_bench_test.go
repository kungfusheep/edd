package rendering

import (
	"edd/core"
	"fmt"
	"testing"
)

// generateLargeDiagram creates a diagram with the specified number of nodes
func generateLargeDiagram(nodes, connectionsPerNode int) *core.Diagram {
	diagram := &core.Diagram{
		Nodes:       make([]core.Node, nodes),
		Connections: make([]core.Connection, 0, nodes*connectionsPerNode),
	}

	// Create nodes
	for i := 0; i < nodes; i++ {
		diagram.Nodes[i] = core.Node{
			ID:   i + 1,
			Text: []string{fmt.Sprintf("Node %d", i+1)},
		}
	}

	// Create connections (avoiding cycles for simple layout)
	for i := 0; i < nodes-1; i++ {
		// Connect to next node
		diagram.Connections = append(diagram.Connections, core.Connection{
			From: i + 1,
			To:   i + 2,
		})
		
		// Add some additional forward connections
		for j := 1; j < connectionsPerNode && i+j+1 < nodes; j++ {
			diagram.Connections = append(diagram.Connections, core.Connection{
				From: i + 1,
				To:   i + j + 2,
			})
		}
	}

	return diagram
}

// BenchmarkRenderer tests rendering performance
func BenchmarkRenderer(b *testing.B) {
	sizes := []struct {
		nodes       int
		connections int
	}{
		{10, 2},
		{25, 2},
		{50, 2},
		{100, 2},
	}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("%d_nodes", size.nodes), func(b *testing.B) {
			diagram := generateLargeDiagram(size.nodes, size.connections)
			renderer := NewRenderer()
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := renderer.Render(diagram)
				if err != nil {
					b.Fatal(err)
				}
			}
			
			// Report useful metrics
			b.ReportMetric(float64(size.nodes), "nodes")
			b.ReportMetric(float64(len(diagram.Connections)), "connections")
		})
	}
}

// BenchmarkRendererMemory tests memory usage
func BenchmarkRendererMemory(b *testing.B) {
	diagram := generateLargeDiagram(100, 2)
	renderer := NewRenderer()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, err := renderer.Render(diagram)
		if err != nil {
			b.Fatal(err)
		}
	}
}
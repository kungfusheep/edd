package render

import (
	"edd/diagram"
	"fmt"
	"testing"
)

// generateLargeDiagram creates a diagram with the specified number of nodes
func generateLargeDiagram(nodes, connectionsPerNode int) *diagram.Diagram {
	d := &diagram.Diagram{
		Nodes:       make([]diagram.Node, nodes),
		Connections: make([]diagram.Connection, 0, nodes*connectionsPerNode),
	}

	// Create nodes
	for i := 0; i < nodes; i++ {
		d.Nodes[i] = diagram.Node{
			ID:   i + 1,
			Text: []string{fmt.Sprintf("Node %d", i+1)},
		}
	}

	// Create connections (avoiding cycles for simple layout)
	for i := 0; i < nodes-1; i++ {
		// Connect to next node
		diagram.Connections = append(diagram.Connections, diagram.Connection{
			From: i + 1,
			To:   i + 2,
		})
		
		// Add some additional forward connections
		for j := 1; j < connectionsPerNode && i+j+1 < nodes; j++ {
			d.Connections = append(d.Connections, diagram.Connection{
				From: i + 1,
				To:   i + j + 2,
			})
		}
	}

	return d
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
			d := generateLargeDiagram(size.nodes, size.connections)
			renderer := NewRenderer()
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := renderer.Render(d)
				if err != nil {
					b.Fatal(err)
				}
			}
			
			// Report useful metrics
			b.ReportMetric(float64(size.nodes), "nodes")
			b.ReportMetric(float64(len(d.Connections)), "connections")
		})
	}
}

// BenchmarkRendererMemory tests memory usage
func BenchmarkRendererMemory(b *testing.B) {
	d := generateLargeDiagram(100, 2)
	renderer := NewRenderer()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, err := renderer.Render(d)
		if err != nil {
			b.Fatal(err)
		}
	}
}
package editor

import (
	"edd/diagram"
	"encoding/json"
	"testing"
)

// Create a test diagram with reasonable complexity
func createTestDiagram(nodes, connections int) *diagram.Diagram {
	d := &diagram.Diagram{
		Nodes:       make([]diagram.Node, nodes),
		Connections: make([]diagram.Connection, 0, connections),
	}
	
	for i := 0; i < nodes; i++ {
		d.Nodes[i] = diagram.Node{
			ID:   i + 1,
			Text: []string{"Node " + string(rune('A'+i)), "Description line", "Another line"},
		}
	}
	
	// Create connections between consecutive nodes
	for i := 0; i < connections && i < nodes-1; i++ {
		d.Connections = append(d.Connections, diagram.Connection{
			From:  i + 1,
			To:    i + 2,
			Label: "connection",
		})
	}
	
	return d
}

// Benchmark JSON serialization approach
func BenchmarkHistoryJSON(b *testing.B) {
	d := createTestDiagram(20, 15) // Reasonable size diagram
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate save
		data, _ := json.Marshal(d)
		_ = string(data)
		
		// Simulate restore
		var restored diagram.Diagram
		json.Unmarshal([]byte(data), &restored)
	}
}

// Simple deep copy implementation for benchmark
func cloneDiagram(d *diagram.Diagram) *diagram.Diagram {
	clone := &diagram.Diagram{
		Nodes:       make([]diagram.Node, len(d.Nodes)),
		Connections: make([]diagram.Connection, len(d.Connections)),
	}
	
	// Deep copy nodes
	for i, node := range d.Nodes {
		textCopy := make([]string, len(node.Text))
		copy(textCopy, node.Text)
		clone.Nodes[i] = diagram.Node{
			ID:     node.ID,
			Text:   textCopy,
			X:      node.X,
			Y:      node.Y,
			Width:  node.Width,
			Height: node.Height,
		}
	}
	
	// Copy connections (they're simple structs)
	copy(clone.Connections, d.Connections)
	
	// Copy metadata if present
	clone.Metadata = d.Metadata
	
	return clone
}

// Benchmark struct cloning approach
func BenchmarkHistoryStruct(b *testing.B) {
	d := createTestDiagram(20, 15) // Same size as JSON test
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate save (clone)
		clone := cloneDiagram(d)
		
		// Simulate restore (just return the clone)
		_ = clone
	}
}

// Benchmark memory allocations for JSON
func BenchmarkHistoryJSONAllocs(b *testing.B) {
	d := createTestDiagram(20, 15)
	
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		data, _ := json.Marshal(d)
		var restored diagram.Diagram
		json.Unmarshal(data, &restored)
	}
}

// Benchmark memory allocations for struct cloning
func BenchmarkHistoryStructAllocs(b *testing.B) {
	d := createTestDiagram(20, 15)
	
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cloneDiagram(d)
	}
}

// Test different diagram sizes
func BenchmarkHistoryScaling(b *testing.B) {
	sizes := []struct {
		name  string
		nodes int
		conns int
	}{
		{"Small", 5, 4},
		{"Medium", 20, 15},
		{"Large", 100, 80},
	}
	
	for _, size := range sizes {
		d := createTestDiagram(size.nodes, size.conns)
		
		b.Run("JSON/"+size.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				data, _ := json.Marshal(d)
				var restored diagram.Diagram
				json.Unmarshal(data, &restored)
			}
		})
		
		b.Run("Struct/"+size.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = cloneDiagram(d)
			}
		})
	}
}
package editor

import (
	"edd/diagram"
	"fmt"
	"strings"
	"testing"
)

// Test the actual scrolling implementation with real diagrams
func TestRealScrollingBehavior(t *testing.T) {
	// Create a real renderer
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	
	// Create a diagram with many nodes to force scrolling
	d := &diagram.Diagram{
		Type: "sequence",
		Nodes: []diagram.Node{},
	}
	
	// Add many actors to create a tall sequence diagram
	for i := 0; i < 20; i++ {
		d.Nodes = append(d.Nodes, diagram.Node{
			ID:   i,
			Text: []string{fmt.Sprintf("Actor%d", i)},
		})
	}
	
	// Add connections between actors
	for i := 0; i < 19; i++ {
		d.Connections = append(d.Connections, diagram.Connection{
			From:  i,
			To:    i + 1,
			Label: fmt.Sprintf("Message %d", i),
		})
	}
	
	tui.SetDiagram(d)
	tui.SetTerminalSize(80, 30) // Small terminal height
	
	// Test 1: Initial render should auto-scroll if content exceeds screen
	output1 := tui.Render()
	lines1 := strings.Split(output1, "\n")
	
	t.Logf("Initial render: %d lines, scroll offset: %d", len(lines1), tui.diagramScrollOffset)
	
	// The diagram should be tall enough to trigger scrolling
	if tui.diagramScrollOffset == 0 && len(lines1) > 25 {
		t.Error("Expected auto-scroll to bottom on overflow, but offset is still 0")
	}
	
	// Test 2: Manual scroll up
	initialOffset := tui.diagramScrollOffset
	tui.ScrollDiagram(-15) // Scroll up half page
	
	if tui.diagramScrollOffset >= initialOffset {
		t.Errorf("ScrollDiagram(-15) should decrease offset, was %d, now %d", 
			initialOffset, tui.diagramScrollOffset)
	}
	
	// Test 3: Scroll to top
	tui.ScrollDiagram(-1000) // Try to scroll way past top
	
	if tui.diagramScrollOffset != 0 {
		t.Errorf("Expected scroll to clamp at 0, got %d", tui.diagramScrollOffset)
	}
	
	// Test 4: Render and check we're actually at top
	output2 := tui.Render()
	
	// At top, there should be no "lines above" indicator
	if strings.Contains(output2, "lines above") {
		t.Errorf("At scroll offset 0, should not have 'lines above' indicator")
	}
	
	// Test 5: Scroll to bottom
	tui.ScrollDiagram(1000) // Try to scroll way past bottom
	output3 := tui.Render()
	
	// At bottom, there should be no "lines below" indicator
	if strings.Contains(output3, "lines below") {
		t.Errorf("At maximum scroll, should not have 'lines below' indicator")
	}
	
	t.Logf("Final offset: %d", tui.diagramScrollOffset)
}

// Test specific scrolling edge cases
func TestScrollingEdgeCases(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	
	// Small diagram that fits on screen
	d := &diagram.Diagram{
		Type: "box",
		Nodes: []diagram.Node{
			{ID: 0, Text: []string{"Node1"}, X: 10, Y: 5},
			{ID: 1, Text: []string{"Node2"}, X: 30, Y: 5},
		},
	}
	
	tui.SetDiagram(d)
	tui.SetTerminalSize(80, 40) // Large terminal
	
	// Render small diagram
	output := tui.Render()
	
	// Should not scroll when content fits
	if tui.diagramScrollOffset != 0 {
		t.Errorf("Content fits on screen, scroll offset should be 0, got %d", 
			tui.diagramScrollOffset)
	}
	
	// Should not have scroll indicators
	if strings.Contains(output, "more lines") {
		t.Error("Small diagram should not have scroll indicators")
	}
	
	// Try to scroll anyway
	tui.ScrollDiagram(10)
	_ = tui.Render()
	
	// Should still be at 0 since content fits
	if tui.diagramScrollOffset != 0 {
		t.Errorf("Cannot scroll when content fits, offset should stay 0, got %d",
			tui.diagramScrollOffset)
	}
}

// Test the wraparound bug specifically
func TestNoWraparound(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	
	// Create a tall diagram - sequence diagrams need connections to be tall
	d := &diagram.Diagram{
		Type: "sequence", 
		Nodes: []diagram.Node{},
	}
	for i := 0; i < 10; i++ {
		d.Nodes = append(d.Nodes, diagram.Node{
			ID:   i,
			Text: []string{fmt.Sprintf("Actor%d", i)},
		})
	}
	
	// Add many connections to make it tall
	connID := 0
	for from := 0; from < 9; from++ {
		for to := from + 1; to < 10 && to < from + 3; to++ {
			d.Connections = append(d.Connections, diagram.Connection{
				ID:    connID,
				From:  from,
				To:    to,
				Label: fmt.Sprintf("msg%d", connID),
			})
			connID++
		}
	}
	
	tui.SetDiagram(d)
	tui.SetTerminalSize(80, 20) // Small height to force scrolling
	
	// Initial render (will auto-scroll to bottom)
	tui.Render()
	maxOffset := tui.diagramScrollOffset
	t.Logf("Max offset after auto-scroll: %d", maxOffset)
	
	// Scroll to near top
	tui.ScrollDiagram(-maxOffset + 5) // Leave at offset 5
	tui.Render()
	
	if tui.diagramScrollOffset != 5 {
		t.Errorf("Expected offset 5, got %d", tui.diagramScrollOffset)
	}
	
	// Try to scroll up more - should go to 0, not wrap
	tui.ScrollDiagram(-10) 
	tui.Render()
	
	if tui.diagramScrollOffset != 0 {
		t.Errorf("Should clamp at 0, got %d", tui.diagramScrollOffset)
	}
	
	// Render again - should stay at 0, not jump to bottom
	output := tui.Render()
	
	if tui.diagramScrollOffset > 0 {
		t.Errorf("Wraparound bug: jumped from 0 to %d", tui.diagramScrollOffset)
	}
	
	// Should not show "lines above" when at top
	if strings.Contains(output, "lines above") {
		t.Error("Should not show 'lines above' when at offset 0")
	}
}

// Test that auto-scroll continues to work when returning to bottom
func TestAutoScrollAfterManualScroll(t *testing.T) {
	renderer := NewRealRenderer()
	tui := NewTUIEditor(renderer)
	
	// Create initial diagram
	d := &diagram.Diagram{
		Type: "sequence",
		Nodes: []diagram.Node{},
	}
	for i := 0; i < 10; i++ {
		d.Nodes = append(d.Nodes, diagram.Node{
			ID:   i,
			Text: []string{fmt.Sprintf("Actor%d", i)},
		})
	}
	for i := 0; i < 9; i++ {
		d.Connections = append(d.Connections, diagram.Connection{
			ID:    i,
			From:  i,
			To:    i + 1,
			Label: fmt.Sprintf("msg%d", i),
		})
	}
	
	tui.SetDiagram(d)
	tui.SetTerminalSize(80, 20)
	
	// Initial render - should auto-scroll to bottom
	tui.Render()
	initialOffset := tui.diagramScrollOffset
	t.Logf("Initial auto-scroll offset: %d", initialOffset)
	
	// Manually scroll to top
	tui.ScrollDiagram(-1000)
	tui.Render()
	if tui.diagramScrollOffset != 0 {
		t.Errorf("Should be at top after scrolling up, got %d", tui.diagramScrollOffset)
	}
	
	// Manually scroll back to bottom
	tui.ScrollDiagram(1000)
	tui.Render()
	atBottomOffset := tui.diagramScrollOffset
	t.Logf("At bottom offset after manual scroll: %d", atBottomOffset)
	
	// Add more content while at bottom
	d.Connections = append(d.Connections, diagram.Connection{
		ID:    100,
		From:  0,
		To:    9,
		Label: "New message that makes diagram taller",
	})
	tui.SetDiagram(d)
	
	// Render - should auto-scroll to new bottom since we were at bottom
	tui.Render()
	newOffset := tui.diagramScrollOffset
	t.Logf("New offset after adding content at bottom: %d", newOffset)
	
	if newOffset <= atBottomOffset {
		t.Errorf("Should have auto-scrolled to new bottom. Was at %d, now at %d", 
			atBottomOffset, newOffset)
	}
	
	// Verify we're actually at the bottom (no "lines below" indicator)
	output := tui.Render()
	if strings.Contains(output, "lines below") {
		t.Error("Should be at bottom, but found 'lines below' indicator")
	}
}
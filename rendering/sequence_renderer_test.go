package rendering

import (
	"edd/canvas"
	"edd/core"
	"strings"
	"testing"
)

func TestCanvasAndNodeRenderer(t *testing.T) {
	caps := canvas.TerminalCapabilities{UnicodeLevel: canvas.UnicodeExtended}
	nodeRenderer := canvas.NewNodeRenderer(caps)
	
	c := canvas.NewMatrixCanvas(30, 10)
	node := core.Node{
		ID:     1,
		X:      5,
		Y:      2,
		Width:  10,
		Height: 3,
		Text:   []string{"Test"},
	}
	
	err := nodeRenderer.RenderNode(c, node)
	if err != nil {
		t.Fatalf("Failed to render node: %v", err)
	}
	
	output := c.String()
	t.Logf("Direct node render:\n%s", output)
	
	if !strings.Contains(output, "Test") {
		t.Error("Should contain Test text")
	}
}

func TestSequenceRendererBasic(t *testing.T) {
	caps := canvas.TerminalCapabilities{UnicodeLevel: canvas.UnicodeExtended}
	renderer := NewSequenceRenderer(caps)
	
	diagram := &core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"User"}},
			{ID: 2, Text: []string{"Server"}},
		},
		Connections: []core.Connection{
			{From: 1, To: 2, Label: "request"},
			{From: 2, To: 1, Label: "response"},
		},
	}
	
	// Get required canvas size
	width, height := renderer.GetBounds(diagram)
	if width <= 0 || height <= 0 {
		t.Fatalf("Invalid bounds: %dx%d", width, height)
	}
	t.Logf("Canvas size: %dx%d", width, height)
	
	// Create canvas and render
	c := canvas.NewMatrixCanvas(width, height)
	err := renderer.Render(diagram, c)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}
	
	output := c.String()
	
	// Debug output
	t.Logf("Canvas output:\n%s", output)
	t.Logf("Nodes after layout: %+v", diagram.Nodes)
	
	// Check for participant boxes
	if !strings.Contains(output, "User") {
		t.Error("Should contain User participant")
	}
	if !strings.Contains(output, "Server") {
		t.Error("Should contain Server participant")
	}
	
	// Check for lifelines (vertical lines)
	if !strings.Contains(output, "│") {
		t.Error("Should contain vertical lifeline characters")
	}
	
	// Check for message arrows
	if !strings.Contains(output, "─") {
		t.Error("Should contain horizontal line characters for messages")
	}
	if !strings.Contains(output, "▶") || !strings.Contains(output, "◀") {
		t.Error("Should contain arrow characters")
	}
	
	// Check for labels
	if !strings.Contains(output, "request") {
		t.Error("Should contain request label")
	}
	if !strings.Contains(output, "response") {
		t.Error("Should contain response label")
	}
}

func TestSequenceRendererSelfMessage(t *testing.T) {
	caps := canvas.TerminalCapabilities{UnicodeLevel: canvas.UnicodeExtended}
	renderer := NewSequenceRenderer(caps)
	
	diagram := &core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"System"}},
		},
		Connections: []core.Connection{
			{From: 1, To: 1, Label: "process"},
		},
	}
	
	width, height := renderer.GetBounds(diagram)
	c := canvas.NewMatrixCanvas(width, height)
	err := renderer.Render(diagram, c)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}
	
	output := c.String()
	
	// Check for self-message loop
	if !strings.Contains(output, "┐") {
		t.Error("Should contain corner character for self-message")
	}
	if !strings.Contains(output, "process") {
		t.Error("Should contain self-message label")
	}
}

func TestSequenceRendererMultipleParticipants(t *testing.T) {
	caps := canvas.TerminalCapabilities{UnicodeLevel: canvas.UnicodeExtended}
	renderer := NewSequenceRenderer(caps)
	
	diagram := &core.Diagram{
		Nodes: []core.Node{
			{ID: 1, Text: []string{"Client"}},
			{ID: 2, Text: []string{"Server"}},
			{ID: 3, Text: []string{"Database"}},
		},
		Connections: []core.Connection{
			{From: 1, To: 2, Label: "request"},
			{From: 2, To: 3, Label: "query"},
			{From: 3, To: 2, Label: "data"},
			{From: 2, To: 1, Label: "response"},
		},
	}
	
	width, height := renderer.GetBounds(diagram)
	c := canvas.NewMatrixCanvas(width, height)
	err := renderer.Render(diagram, c)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}
	
	output := c.String()
	
	// Check all participants are present
	if !strings.Contains(output, "Client") {
		t.Error("Should contain Client participant")
	}
	if !strings.Contains(output, "Server") {
		t.Error("Should contain Server participant")
	}
	if !strings.Contains(output, "Database") {
		t.Error("Should contain Database participant")
	}
	
	// Check all message labels
	if !strings.Contains(output, "request") {
		t.Error("Should contain request message")
	}
	if !strings.Contains(output, "query") {
		t.Error("Should contain query message")
	}
	if !strings.Contains(output, "data") {
		t.Error("Should contain data message")
	}
	if !strings.Contains(output, "response") {
		t.Error("Should contain response message")
	}
}
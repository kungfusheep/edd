package editor

import (
	"edd/core"
	"fmt"
	"strings"
	"testing"
)

func TestShowEdPosition(t *testing.T) {
	state := TUIState{
		Diagram: &core.Diagram{
			Nodes: []core.Node{
				{ID: 1, Text: []string{"Node1"}},
				{ID: 2, Text: []string{"Node2"}},
			},
			Connections: []core.Connection{
				{From: 1, To: 2},
			},
		},
		Mode:     ModeNormal,
		EddFrame: "◉‿◉",
		Width:    80,
		Height:   24,
	}
	
	output := RenderTUI(state)
	lines := strings.Split(output, "\n")
	
	fmt.Println("=== Full Output (with line numbers) ===")
	for i, line := range lines {
		fmt.Printf("%2d: %-70s|\n", i, line)
	}
	
	fmt.Println("\n=== Looking for Ed ===")
	for i, line := range lines {
		if strings.Contains(line, "◉‿◉") {
			fmt.Printf("Found Ed at line %d, column %d\n", i, strings.Index(line, "◉‿◉"))
		}
		if strings.Contains(line, "NORMAL") {
			fmt.Printf("Found NORMAL at line %d\n", i)
		}
		if strings.Contains(line, "╭") {
			fmt.Printf("Found box top at line %d\n", i)
		}
	}
}
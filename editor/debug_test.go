package editor

import (
	"edd/core"
	"fmt"
	"testing"
)

func TestDebugOutput(t *testing.T) {
	state := TUIState{
		Diagram: &core.Diagram{
			Nodes: []core.Node{
				{ID: 1, Text: []string{"Server"}},
				{ID: 2, Text: []string{"Database"}},
			},
		},
		Mode:     ModeNormal,
		EddFrame: "◉‿◉",
	}
	
	output := RenderTUI(state)
	fmt.Println("=== Render Output ===")
	fmt.Println(output)
	fmt.Println("=== End Output ===")
}
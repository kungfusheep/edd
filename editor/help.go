package editor

import (
	"fmt"
	"strings"
)

// Help categories
type HelpCategory struct {
	Name     string
	Commands []HelpCommand
}

type HelpCommand struct {
	Key         string
	Description string
}

// GetHelpText returns the help text for display
func GetHelpText() string {
	categories := []HelpCategory{
		{
			Name: "Node Operations",
			Commands: []HelpCommand{
				{"a/A", "Add node (A for continuous)"},
				{"e", "Edit node/connection text"},
				{"d/D", "Delete node/connection (D for continuous)"},
			},
		},
		{
			Name: "Connection Operations",
			Commands: []HelpCommand{
				{"c/C", "Connect nodes (C for continuous)"},
				{"i/I", "Insert connection (I for continuous)"},
				{"v", "Toggle activation (sequence diagrams)"},
			},
		},
		{
			Name: "Visual Styling",
			Commands: []HelpCommand{
				{"H", "Edit hints - shows labels on nodes & connections"},
			},
		},
		{
			Name: "Navigation & View",
			Commands: []HelpCommand{
				{"J", "Toggle JSON view"},
			{"j/k", "Scroll down/up (line by line)"},
			{"Ctrl+D/U", "Scroll down/up (half page)"},
				{"t", "Toggle diagram type (sequence/box)"},
				{"E", "Edit in external editor"},
				{"?", "Show this help"},
			},
		},
		{
			Name: "Editing",
			Commands: []HelpCommand{
				{"u", "Undo"},
				{"Ctrl+R", "Redo"},
				{"ESC", "Cancel/Exit mode"},
				{":", "Command mode"},
			},
		},
		{
			Name: "System",
			Commands: []HelpCommand{
				{"q", "Quit"},
				{"Ctrl+C", "Force quit"},
			},
		},
	}

	var b strings.Builder
	b.WriteString("\n╔════════════════════════════════════════════════════╗\n")
	b.WriteString("║                  EDD HELP                         ║\n")
	b.WriteString("╠════════════════════════════════════════════════════╣\n")
	
	for i, cat := range categories {
		b.WriteString(fmt.Sprintf("║ %-50s ║\n", cat.Name+":"))
		for _, cmd := range cat.Commands {
			b.WriteString(fmt.Sprintf("║   %-8s %-40s ║\n", cmd.Key, cmd.Description))
		}
		if i < len(categories)-1 {
			b.WriteString("║                                                    ║\n")
		}
	}
	
	b.WriteString("╠════════════════════════════════════════════════════╣\n")
	b.WriteString("║ In JUMP mode: Press labeled key to select target  ║\n")
	b.WriteString("║ In EDIT mode: Ctrl+N for newline, Enter to save   ║\n")
	b.WriteString("║   Connections: Enter/Tab=next, Shift+Tab=previous ║\n")
	b.WriteString("╚════════════════════════════════════════════════════╝\n")
	
	return b.String()
}

// GetCompactHelp returns a single-line help hint
func GetCompactHelp() string {
	return "a:add c:connect e:edit d:delete H:hints J:json j/k:scroll u:undo ?:help q:quit"
}

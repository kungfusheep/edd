# edd

```
╭────╮
│◉‿ ◉│
╰────╯
```

A terminal-based diagram editor for creating sequence diagrams and flowcharts.

## Features

### Diagram Types
- **Sequence diagrams** - Show interactions between participants over time
- **Flowcharts** - Box-and-arrow diagrams for processes and flows

### Editor Modes
- **Interactive TUI** - Vim-like modal editing in the terminal
- **Command-line** - Render diagrams directly from JSON files
- **Import/Export** - Convert between multiple diagram formats

### TUI Editor Commands

#### Normal Mode
- `c` - Connect nodes
- `C` - Connect nodes (continuous mode)
- `d` - Delete node/connection
- `D` - Delete (continuous mode)
- `e` - Edit selected item
- `u` - Undo
- `Ctrl+r` - Redo
- `t` - Toggle diagram type
- `i` - Insert connection at position (sequence diagrams)
- `I` - Insert connection (continuous mode)
- `H` - Edit style hints
- `?` - Show help
- `:` - Enter command mode

#### Command Mode
- `:w [filename]` - Save diagram
- `:wq` - Save and quit
- `:q` - Quit
- `:export format [filename]` - Export to format (mermaid, plantuml, ascii)

### Import/Export Formats

#### Import Support
- **Mermaid** (.mmd, .mermaid) - Basic sequence diagrams and flowcharts
- **PlantUML** (.puml, .plantuml) - Basic sequence diagrams
- **Graphviz DOT** (.dot, .gv) - Basic directed graphs
- **D2** (.d2) - Basic diagrams
- **JSON** (.json) - Native edd format

#### Export Support
- **ASCII/Unicode** - Box-drawing characters for terminal/text files
- **Mermaid** - Markdown-compatible diagram syntax
- **PlantUML** - Text-based UML diagrams
- **JSON** - Native edd format for storage

## Installation

```bash
go install github.com/kungfusheep/edd@latest
```

## Usage

### Interactive Editing
```bash
# Start new diagram
edd

# Edit existing diagram
edd -i diagram.json

# Start with specific type
edd -type sequence
```

### Import/Export
```bash
# Render imported Mermaid to terminal
edd diagram.mmd

# Edit imported Mermaid in TUI
edd -i diagram.mmd

# Import with explicit format and render
edd -import mermaid diagram.txt

# Export to different format
edd -format plantuml diagram.json

# Convert between formats
edd -format mermaid sequence.puml
```

### Command-line Rendering
```bash
# Render to stdout
edd diagram.json

# Save to file
edd -o output.txt diagram.json

# Debug mode with layout information
edd -debug diagram.json
```

## Diagram Format

Diagrams are stored as JSON:

```json
{
  "type": "sequence",
  "nodes": [
    {"id": 0, "text": ["Client"]},
    {"id": 1, "text": ["Server"]}
  ],
  "connections": [
    {"from": 0, "to": 1, "arrow": true, "label": "Request"},
    {"from": 1, "to": 0, "arrow": true, "label": "Response"}
  ]
}
```

## Keyboard Controls

### Jump Mode
When entering commands like connect or delete, jump labels appear:
- Single letters appear on nodes/connections
- Press the letter to select that item
- ESC to cancel

### Text Editing
- Standard text input in edit/insert modes
- ESC saves and exits to normal mode (no separate confirm key)
- Backspace/Delete to remove characters

## Limitations

- Import support covers core features only (~20-30% of each format's syntax)
- No mouse support (keyboard-only)
- Terminal-based rendering (no image export)
- Sequence diagrams limited to simple message flows
- Flowcharts limited to basic box-and-arrow layouts

## Requirements

- Go 1.19+
- Terminal with Unicode support
- 80+ column width recommended

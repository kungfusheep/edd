# edd `◉‿ ◉`

Fast, keyboard-driven diagram editing in your terminal. Create sequence diagrams and flowcharts with vim-like efficiency.

## Why edd?

- **Built for speed** - Every action optimized for minimal keystrokes
- **Jump navigation** - Navigate and edit with single-key jumps (like EasyMotion for diagrams)
- **Multi-format** - Work with Mermaid, PlantUML, Graphviz, D2 using the same interface
- **Terminal-native** - No browser, no mouse, just your keyboard

## Features

### Instant Navigation with Jump Mode
Press any action key (connect, delete, edit) and jump labels appear on every element. Hit a single key to select - no arrow keys, no counting, no searching.

### Diagram Types
- **Sequence diagrams** - Show interactions between participants over time
<img width="1636" height="1009" alt="image" src="https://github.com/user-attachments/assets/0404230a-c0f6-41b0-a10a-05aa6663e10d" />

- **Flowcharts** - Box-and-arrow diagrams for processes and flows
<img width="1400" height="346" alt="image" src="https://github.com/user-attachments/assets/14feb915-c53c-46ab-8380-c1472f33e99d" />


### Multi-Format Support
Edit multiple diagram formats with the same fast interface. Import from one format, export to another.

#### Supported Formats
- **Import**: Mermaid, PlantUML, Graphviz DOT, D2, JSON
- **Export**: ASCII/Unicode, Mermaid, PlantUML, JSON
- **Convert**: Between formats in one command

### Editor Modes
- **Interactive TUI** - Vim-like modal editing with jump navigation
- **Command-line** - Render diagrams directly or convert between formats
- **Batch processing** - Convert entire directories of diagrams

### Speed-Focused Commands

Every command designed for efficiency - no mouse needed, minimal keystrokes required.

#### Core Operations (with Jump Mode)
- `c` / `C` - Connect nodes (single/continuous)
- `d` / `D` - Delete elements (single/continuous)
- `e` - Edit any element
- `i` / `I` - Insert connections (single/continuous)

#### Instant Actions
- `u` - Undo
- `Ctrl+r` - Redo
- `t` - Toggle diagram type
- `H` - Edit style hints
- `?` - Help
- `:` - Command mode

#### Vim-style Commands
- `:w [filename]` - Save
- `:wq` - Save and quit
- `:q` - Quit
- `:export format [file]` - Export to supported formats


## Installation

```bash
go install github.com/kungfusheep/edd@latest
```

## Usage

### Quick Start
```bash
# Launch interactive editor
edd

# Edit supported diagram formats
edd -i diagram.mmd
edd -i flowchart.puml
edd -i graph.dot
```

### Format Conversion - Seamless Translation Between Formats
```bash
# Mermaid to PlantUML
edd -format plantuml diagram.mmd

# PlantUML to ASCII art
edd -format ascii sequence.puml

# Graphviz to Mermaid
edd -format mermaid graph.dot

# Display various formats in terminal
edd diagram.mmd
edd flowchart.puml
edd graph.d2
```

### Real-World Workflow Example

Transform diagram formats using edd's unified editing experience:

```bash
# Your team uses PlantUML, but you need Mermaid for GitHub
$ edd -i architecture.puml
# → Opens in TUI editor with full editing capabilities
# → Make your changes using jump navigation
# → Press :export mermaid architecture.mmd
# → Now you have both formats!

# Or do it in one command
$ edd -format mermaid architecture.puml > architecture.mmd
```

### More Workflow Examples
```bash
# Edit a Mermaid diagram from your markdown docs
edd -i README.mmd

# Convert your team's PlantUML diagrams to Mermaid
for file in *.puml; do
  edd -format mermaid "$file" > "${file%.puml}.mmd"
done

# Quick ASCII diagram for documentation
edd -format ascii design.json > diagram.txt

# Import Graphviz, edit interactively, save as PlantUML
edd -i network.dot
# (edit with jump mode navigation)
# :export plantuml network.puml
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

## How Jump Mode Works

The key to edd's speed - no arrow keys, no searching, just single-key selection:

1. Press an action key (`c` for connect, `d` for delete, `e` for edit)
2. Every selectable element gets a unique letter label
3. Press that letter to instantly select the element
4. For two-target operations (like connect), repeat for the second element

Example: To connect two nodes, just type `c`, then two letters. Three keystrokes total.

### Text Input
- Direct typing in edit mode - no special insert command needed
- ESC instantly saves and returns to normal mode
- No confirmation dialogs to slow you down

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

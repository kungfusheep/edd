# EDD Command Mode Reference

Press `:` to enter command mode in the TUI editor.

## File Operations

### Save
```
:w [filename]         Save diagram to file
:write [filename]     Save diagram to file (same as :w)
```

If no filename is provided, uses the currently loaded file.

**Examples:**
```
:w                    Save to current file
:w diagram.json       Save to diagram.json
```

### Save and Quit
```
:wq [filename]        Save and quit
```

Saves the diagram and exits the editor.

### Quit
```
:q                    Quit (warns if unsaved changes)
:q!                   Force quit without saving
:quit                 Quit (same as :q)
:qq                   Force quit completely (exits markdown picker if in markdown mode)
```

**Examples:**
```
:q                    Normal quit - warns if changes
:q!                   Discard changes and quit
:qq                   Force quit to shell
```

## Export

Export diagrams to various formats:

```
:export <format> [filename]
:e <format> [filename]          Short form
```

### Supported Formats

| Format | Extensions | Description |
|--------|-----------|-------------|
| `mermaid` | `.mmd` | Mermaid diagram syntax |
| `plantuml` | `.puml` | PlantUML diagram syntax |
| `svg` | `.svg` | Scalable Vector Graphics |
| `json` | `.json` | EDD native JSON format |
| `graphviz` | `.dot`, `.gv` | Graphviz DOT syntax |
| `d2` | `.d2` | D2 diagram syntax |
| `ascii` | `.txt` | ASCII/Unicode art (terminal output) |

### Export to Clipboard

Use `clip` or `clipboard` as the filename to copy to clipboard (macOS):

```
:export svg clip              Copy SVG to clipboard
:export mermaid clipboard     Copy Mermaid to clipboard
```

### Export Examples

```
:export mermaid diagram.mmd   Export to Mermaid format
:export svg output.svg        Export to SVG
:export plantuml flow.puml    Export to PlantUML
:export json backup.json      Save as JSON
:export ascii clip            Copy ASCII art to clipboard
```

## Diagram Settings

Set diagram-level properties that affect rendering:

```
:set <property> <value>       Set a diagram property
:unset <property>             Remove a diagram property
```

### Available Properties

| Property | Values | Description | Example |
|----------|--------|-------------|---------|
| `layout` | `vertical`, `horizontal` | Layout direction | `:set layout horizontal` |
| `title` | any string | Diagram title (future) | `:set title "My Pipeline"` |

### Layout Direction

**Vertical (default):** Top-to-bottom flow, ideal for flowcharts and decision trees
```
:set layout vertical
```

**Horizontal:** Left-to-right flow, ideal for pipelines, timelines, and process flows
```
:set layout horizontal
```

**Visual comparison:**

Vertical:
```
  ╭────╮
  │ A  │
  ╰─┬──╯
    │
    ▼
  ╭────╮
  │ B  │
  ╰────╯
```

Horizontal:
```
╭────╮        ╭────╮
│ A  ├───────▶│ B  │
╰────╯        ╰────╯
```

### Settings Persistence

Settings are stored in the diagram JSON under the `hints` field:

```json
{
  "type": "box",
  "hints": {
    "layout": "horizontal"
  },
  "nodes": [...],
  "connections": [...]
}
```

## Tips

- All commands are vim-style with `:` prefix
- Press `ESC` to cancel command mode
- Command history is not currently supported
- Tab completion is not currently supported
- Multi-word values should be quoted in the future (not yet implemented)

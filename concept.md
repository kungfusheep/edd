# Terminal Diagram Editor - Specification

## Purpose
A terminal-based application for creating and editing box-and-line diagrams with vim-inspired keyboard controls, designed for efficient diagram creation without leaving the terminal.

## Core Concepts

### Modes
The editor operates in distinct modes, similar to vim:

- **Normal Mode**: Default mode for navigation and commands
  - `h/j/k/l` or arrow keys: Move cursor
  - `a`: Add new node at cursor position
  - `d`: Delete node under cursor
  - `c`: Enter Connect mode
  - `i`: Enter Insert mode to edit node text
  - `/`: Enter Jump mode
  - `:`: Enter command mode
  - `q` or `Ctrl+C`: Quit

- **Insert Mode**: Edit text within nodes
  - Type to edit node content
  - `Enter`: New line within node
  - `Esc`: Return to Normal mode

- **Connect Mode**: Create connections between nodes
  - `h/j/k/l`: Move to select target node
  - `Enter`: Create connection
  - `Esc`: Cancel and return to Normal mode

- **Jump Mode**: Quick navigation
  - Type node text to jump to matching node
  - `Esc`: Cancel

- **Command Mode**: Execute commands
  - `w [filename]`: Save diagram
  - `o [filename]`: Open diagram
  - `q`: Quit
  - `diagram [type]`: Set diagram type (boxes, sequence)
  - `layout [type]`: Set layout algorithm

### Visual Elements

#### Nodes
- Rectangular boxes with rounded corners
- Unicode box drawing: ╭─╮ │ │ ╰─╯
- Contain text labels (single or multi-line)
- Minimum size to accommodate text with padding

#### Connections
- Directed edges between nodes
- Arrow heads indicate direction: ▶ ◀ ▲ ▼
- Line segments: ─ (horizontal) │ (vertical)
- Junction characters where connections meet:
  - Box borders: ├ ┤ ┬ ┴
  - Line intersections: ┼ ╭ ╮ ╰ ╯

### Connection Routing Philosophy

#### Clarity over Efficiency
- Connections should be visually clear and easy to follow
- Avoid straight vertical drops when nodes are aligned
- Prefer routing that shows relationships explicitly

#### Routing Rules
1. **Horizontal alignment**: Simple horizontal line with arrow
2. **Vertical alignment**: Route around to show connection clearly
   - Exit horizontally from source
   - Travel vertically in a clear trunk line
   - Return horizontally to target
3. **Diagonal**: L-shaped routing
   - Horizontal first, then vertical
   - Consistent trunk positioning for multiple connections

#### Junction Character Rules
- Always show junction characters where connections attach to boxes
- ├ for connections exiting right side
- ┤ for connections exiting left side
- ┬ for connections exiting bottom
- ┴ for connections exiting top

### Layout Engine

#### Hierarchical Layout
- Organizes nodes in layers based on dependencies
- Flows left-to-right by default
- Minimizes edge crossings
- Maintains consistent spacing

#### Principles
- Connected components stay together
- Respect connection directionality
- Provide reasonable defaults while allowing manual adjustment
- Support incremental layout (adding nodes doesn't reorganize everything)

### Diagram Types

#### Box and Lines (default)
- General purpose diagrams
- Boxes connected by directed edges
- Suitable for flowcharts, architecture diagrams, concept maps

#### Sequence Diagrams (future)
- Actors with lifelines
- Messages between actors
- Time flows top-to-bottom

### User Experience Ideals

#### Keyboard-First
- All operations accessible via keyboard
- No mouse required
- Muscle memory from vim transfers naturally

#### Speed
- Quick node creation and connection
- Minimal keystrokes for common operations
- Jump mode for fast navigation in large diagrams

#### Clarity
- Clear visual feedback for current mode
- Connections easy to trace
- No ambiguous junction points

#### Flexibility
- Support various diagram styles
- Allow manual positioning when needed
- Export to common formats

## File Format
- Plain text format for version control
- Human-readable and editable
- Preserves all diagram information
- Extension: `.diagram` or `.diag`

## Design Philosophy
- Prefer clarity over compactness
- Make the common case fast
- Respect terminal constraints while maximizing capability
- Follow vim's modal philosophy consistently

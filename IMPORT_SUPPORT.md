# Import Format Support Documentation

## Mermaid Support

### Sequence Diagrams ✅
**Supported:**
- `sequenceDiagram` declaration
- `participant Name` - basic participant declaration
- `participant ID as Display Name` - participant with alias
- Arrow types:
  - `->` or `->>` - solid arrow
  - `-->` or `-->>` - dashed arrow
  - `-x` or `--x` - crossed/cancelled arrow (marked with "crossed" hint)
  - `->>+` or `-->+` - arrows with activation (not fully utilized)
- Messages with labels: `A->>B: Message text`
- Auto-creation of participants from messages

**NOT Supported Yet:**
- `actor` declarations (treated as participant)
- `activate`/`deactivate` blocks
- `loop`, `alt`, `opt`, `par` blocks
- `Note` statements
- `rect` backgrounds
- `autonumber`
- Message numbering
- Participant aliases beyond "as"

### Flowcharts/Graphs ✅ (Basic)
**Supported:**
- `graph` and `flowchart` declarations
- Directions: `LR`, `TD`, `TB`, `RL`, `BT`
- Node shapes (parsed but simplified):
  - `[text]` - rectangle
  - `(text)` - rounded
  - `{text}` - diamond
  - `[[text]]` - double border
- Connections:
  - `-->` - solid arrow
  - `-.->` - dashed arrow
  - `==>` - thick arrow
- Connection labels: `A -->|label| B`

**NOT Supported Yet:**
- Subgraphs
- Styling and classes
- Link styles
- Node shapes beyond basic (stadium, hexagon, etc.)
- Dotted/chain links
- Multi-directional arrows

## PlantUML Support

### Sequence Diagrams ✅
**Supported:**
- `@startuml`/`@enduml` blocks
- `participant` and `actor` declarations
- Arrow types:
  - `->` - solid arrow
  - `-->` - dashed arrow
- Colored arrows: `-[#color]>`
- Messages with labels: `A -> B: Message`
- Auto-creation of participants

**NOT Supported Yet:**
- `activate`/`deactivate`
- `alt`/`else`/`opt`/`loop` blocks
- `note` statements
- `ref` blocks
- `...` delays
- `|||` space
- `newpage`
- Participant creation/destruction
- Return arrows `<--`

### Other PlantUML Diagrams ❌
- Class diagrams
- Use case diagrams
- Component diagrams
- State diagrams
- Object diagrams
- Activity diagrams

## Graphviz DOT Support ✅ (Basic)

**Supported:**
- `digraph` and `graph` declarations
- Basic edges: `A -> B`
- Edge labels: `A -> B [label="text"]`
- Node labels: `A [label="text"]`
- Quoted and unquoted identifiers

**NOT Supported Yet:**
- Subgraphs/clusters
- Node shapes and styles
- Edge styles (color, weight, style)
- Graph attributes
- Rank constraints
- HTML labels
- Record/Mrecord shapes
- Ports

## D2 Support ✅ (Basic)

**Supported:**
- Simple connections: `A -> B`
- Connection labels: `A -> B: Label`
- Bidirectional: `A <-> B`
- Dashed connections: `A -- B`
- Node labels: `nodeName: Display Text`
- Quoted identifiers

**NOT Supported Yet:**
- Nested structures
- Containers
- Shapes (beyond default)
- Styles and classes
- Markdown in labels
- Icons
- Multiple connections syntax
- Imports
- Variables

## Summary

Current support is focused on the **core essentials**:
1. **Nodes/Participants** - Basic creation and naming
2. **Connections** - Directional arrows with labels
3. **Basic Styling** - Solid vs dashed lines

This covers approximately:
- **Mermaid**: ~30% of sequence diagram features, ~20% of flowchart features
- **PlantUML**: ~25% of sequence diagram features
- **Graphviz**: ~15% of DOT features
- **D2**: ~20% of features

The importers prioritize the most common use cases and can handle simple to moderately complex diagrams, but advanced features like conditional blocks, styling, and complex layouts are not yet supported.
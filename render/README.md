# Canvas Module

This module provides 2D character grid implementations for rendering diagrams in the terminal.

## MatrixCanvas

The `MatrixCanvas` is a rune matrix-based implementation that provides efficient terminal rendering with full Unicode support.

### Features

1. **Direct Matrix Access**
   - `Matrix() [][]rune` - Direct access for tools like LineValidator
   - `Get(Point) rune` - Read character at position  
   - `Set(Point, rune) error` - Write character at position

2. **High-Level Drawing Primitives**
   - `DrawBox(x, y, w, h, BoxStyle)` - Draw rectangles with various styles
   - `DrawHorizontalLine(x1, y, x2, char)` - Draw horizontal lines
   - `DrawVerticalLine(x, y1, y2, char)` - Draw vertical lines
   - `DrawLine(p1, p2, char)` - Draw diagonal lines using Bresenham's algorithm
   - `DrawSmartPath([]Point)` - Draw paths with automatic corner selection

3. **Text Operations**
   - `DrawText(x, y, string)` - Render text with proper Unicode width handling
   - `StringWidth(string) int` - Calculate display width using East Asian Width
   - `WrapText(string, width) []string` - Word wrapping with multiple modes
   - `WrapTextMode(string, width, mode)` - Wrap with Word/Char/Hyphenate modes
   - `FitText(string, width, ellipsis) string` - Text truncation
   - `TruncateToWidth(string, width)` - Truncate to exact display width

4. **Advanced Unicode Support**
   - Full East Asian Width property support
   - Handles wide characters (CJK, emoji) correctly
   - Zero-width character support (combiners, ZWJ)
   - Proper terminal cell width calculation
   - Wide character boundary protection (no partial characters)

5. **Complete Junction Resolution**
   - Automatic junction selection: ├ ┤ ┬ ┴ ┼
   - Smart corner selection: ╭ ╮ ╰ ╯
   - Preserves existing junctions when drawing
   - Compatible with LineValidator rules

### Thread Safety

MatrixCanvas is NOT thread-safe for writes. Synchronize externally:

```go
// Using mutex:
var mu sync.Mutex
mu.Lock()
canvas.DrawBox(10, 10, 20, 20, DefaultBoxStyle)
mu.Unlock()

// Using channels:
type canvasOp func(*MatrixCanvas)
ops := make(chan canvasOp)
go func() {
    for op := range ops {
        op(canvas)
    }
}()
```

### Performance

- Create 100x100 canvas: ~7.5µs
- Convert to string: ~35µs  
- DrawBox: ~6.5µs (zero allocations)
- Junction resolution: O(1) per intersection
- Thread-safe for concurrent reads

### Styles

Pre-defined box styles:
- `DefaultBoxStyle` - Unicode rounded corners (╭─╮│╰─╯)
- `SimpleBoxStyle` - ASCII style (+-|)
- `DoubleLineStyle` - Double lines (╔═╗║╚═╝)

### Text Wrapping Modes

- `WrapModeWord` - Break at word boundaries (default)
- `WrapModeChar` - Break at character boundaries
- `WrapModeHyphenate` - Add hyphens when breaking words

## Future: SmartCanvas

Planned implementation with:
- Collision detection
- Automatic rerouting
- Path optimization
- Multi-layer support
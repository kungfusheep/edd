package render

// BoxStyle defines the characters used to draw a box.
type BoxStyle struct {
	TopLeft     rune
	TopRight    rune
	BottomLeft  rune
	BottomRight rune
	Horizontal  rune
	Vertical    rune
}

// Predefined box styles
var (
	// DefaultBoxStyle uses rounded corners
	DefaultBoxStyle = BoxStyle{
		TopLeft:     '╭',
		TopRight:    '╮',
		BottomLeft:  '╰',
		BottomRight: '╯',
		Horizontal:  '─',
		Vertical:    '│',
	}

	// SimpleBoxStyle uses ASCII characters
	SimpleBoxStyle = BoxStyle{
		TopLeft:     '+',
		TopRight:    '+',
		BottomLeft:  '+',
		BottomRight: '+',
		Horizontal:  '-',
		Vertical:    '|',
	}

	// DoubleBoxStyle uses double-line characters
	DoubleBoxStyle = BoxStyle{
		TopLeft:     '╔',
		TopRight:    '╗',
		BottomLeft:  '╚',
		BottomRight: '╝',
		Horizontal:  '═',
		Vertical:    '║',
	}
)

// ArrowStyle defines the characters used for arrows in different directions.
type ArrowStyle struct {
	Right rune
	Left  rune
	Up    rune
	Down  rune
}

// Predefined arrow styles
var (
	// StandardArrows uses Unicode triangles
	StandardArrows = ArrowStyle{
		Right: '▶',
		Left:  '◀',
		Up:    '▲',
		Down:  '▼',
	}

	// SimpleArrows uses ASCII characters
	SimpleArrows = ArrowStyle{
		Right: '>',
		Left:  '<',
		Up:    '^',
		Down:  'v',
	}
)


// DefaultLineStyle uses Unicode box-drawing characters
var DefaultLineStyle = LineStyle{
	Horizontal:  '─',
	Vertical:    '│',
	TopLeft:     '╭',
	TopRight:    '╮',
	BottomLeft:  '╰',
	BottomRight: '╯',
	Cross:       '┼',
	TeeUp:       '┴',
	TeeDown:     '┬',
	TeeLeft:     '┤',
	TeeRight:    '├',
}

// TextAlign specifies text alignment within a box
type TextAlign int

const (
	AlignLeft TextAlign = iota
	AlignCenter
	AlignRight
)
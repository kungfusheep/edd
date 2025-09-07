package render

// NodeStyle defines the characters used to draw a node box
type NodeStyle struct {
	TopLeft     rune
	TopRight    rune
	BottomLeft  rune
	BottomRight rune
	Horizontal  rune
	Vertical    rune
}

// NodeStyles defines the available box styles for nodes
var NodeStyles = map[string]NodeStyle{
	"rounded": {
		TopLeft:     '╭',
		TopRight:    '╮',
		BottomLeft:  '╰',
		BottomRight: '╯',
		Horizontal:  '─',
		Vertical:    '│',
	},
	"sharp": {
		TopLeft:     '┌',
		TopRight:    '┐',
		BottomLeft:  '└',
		BottomRight: '┘',
		Horizontal:  '─',
		Vertical:    '│',
	},
	"double": {
		TopLeft:     '╔',
		TopRight:    '╗',
		BottomLeft:  '╚',
		BottomRight: '╝',
		Horizontal:  '═',
		Vertical:    '║',
	},
	"thick": {
		TopLeft:     '┏',
		TopRight:    '┓',
		BottomLeft:  '┗',
		BottomRight: '┛',
		Horizontal:  '━',
		Vertical:    '┃',
	},
	"ascii": {
		TopLeft:     '+',
		TopRight:    '+',
		BottomLeft:  '+',
		BottomRight: '+',
		Horizontal:  '-',
		Vertical:    '|',
	},
}

// GetNodeStyle returns the NodeStyle for a given style name, with fallback to default
func GetNodeStyle(styleName string, caps TerminalCapabilities) NodeStyle {
	// If no Unicode support, always use ASCII
	if caps.UnicodeLevel == UnicodeNone {
		return NodeStyles["ascii"]
	}
	
	// Try to get the requested style
	if style, ok := NodeStyles[styleName]; ok {
		return style
	}
	
	// Default to rounded for Unicode terminals
	return NodeStyles["rounded"]
}

// DefaultNodeStyle returns the default node style based on terminal capabilities
func DefaultNodeStyle(caps TerminalCapabilities) NodeStyle {
	if caps.UnicodeLevel >= UnicodeBasic {
		return NodeStyles["rounded"]
	}
	return NodeStyles["ascii"]
}
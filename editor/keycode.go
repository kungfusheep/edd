package editor

// SpecialKey represents special keys like arrows, home, end, etc.
type SpecialKey int

const (
	KeyNone SpecialKey = iota
	KeyArrowUp
	KeyArrowDown
	KeyArrowLeft
	KeyArrowRight
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeyDelete
)

// KeyEvent represents either a regular character or a special key
type KeyEvent struct {
	Rune       rune
	SpecialKey SpecialKey
}

// IsSpecial returns true if this is a special key event
func (k KeyEvent) IsSpecial() bool {
	return k.SpecialKey != KeyNone
}
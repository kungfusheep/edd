package editor

// EddCharacter represents the animated Ed mascot
type EddCharacter struct {
	frameCount int
	frames     map[Mode][]string
}

// NewEddCharacter creates a new Ed character with animations from main branch
func NewEddCharacter() *EddCharacter {
	ed := &EddCharacter{
		frameCount: 0,
		frames:     make(map[Mode][]string),
	}
	
	// Define idle animations for each mode - meet "ed"!
	ed.frames[ModeNormal] = []string{
		"◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "-‿ -", "◉‿ ◉", // Occasional blink with smile
		"◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉",
		"◉‿ ◉", "◉‿ ◉", "⊙‿ ⊙", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", "◉‿ ◉", // Occasional look around
	}
	
	ed.frames[ModeInsert] = []string{
		"○‿ ○", "○‿ ○", "○‿ ○", "○‿ ○", "○‿ ○", "○‿ ○", // Wide alert eyes
		"-‿ ○", "-‿ -", "○‿ -", "○‿ ○", "○‿ ○", "-‿ ○", // Out-of-sync blinks
		"○‿ ○", "-‿ -", "○‿ ○", "○‿ ○", "◉‿ ◉", "○‿ ○", // Full blink and focus burst
	}
	
	ed.frames[ModeEdit] = []string{
		"◉‿ ◉", "⊙‿ ⊙", "◉‿ ◉", "⊙‿ ⊙", "◉‿ ◉", "⊙‿ ⊙", // Contemplative selection
		"⊙‿ ⊙", "-‿ -", "⊙‿ ⊙", "◉‿ ◉", "⊙‿ ⊙", "◉‿ ◉", // Thoughtful scanning
	}
	
	ed.frames[ModeCommand] = []string{
		":_", ":_", ":|", ":|", ":_", ":_", ":_", ":_", // Cursor blink
		":_", ":_", ":_", ":?", ":_", ":_", ":_", ":_", // Occasional thinking
	}
	
	ed.frames[ModeJump] = []string{
		"◉‿ ◉", "◎‿ ◎", "◉‿ ◉", "◎‿ ◎", "◉‿ ◉", "◎‿ ◎", // Eyes darting between nodes
		"◎‿ ◎", "-‿ -", "◎‿ ◎", "◎‿ ◎", "◎‿ ◎", "◎‿ ◎", // Quick blink while scanning
	}
	
	return ed
}

// GetFrame returns the current animation frame for the given mode
func (e *EddCharacter) GetFrame(mode Mode) string {
	frames, ok := e.frames[mode]
	if !ok || len(frames) == 0 {
		return "◉‿◉" // Default face
	}
	
	// Return current frame
	frame := frames[e.frameCount%len(frames)]
	return frame
}

// NextFrame advances to the next animation frame
func (e *EddCharacter) NextFrame() {
	e.frameCount++
}
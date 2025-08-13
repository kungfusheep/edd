package editor

// Mode represents the current editing mode
type Mode int

const (
	ModeNormal  Mode = iota // Normal navigation mode
	ModeInsert              // Inserting new nodes
	ModeEdit                // Editing existing node text
	ModeCommand             // Command input mode
	ModeJump                // Jump selection active
	ModeJSON                // JSON view mode
)

// String returns the mode name for display
func (m Mode) String() string {
	switch m {
	case ModeNormal:
		return "NORMAL"
	case ModeInsert:
		return "INSERT"
	case ModeEdit:
		return "EDIT"
	case ModeCommand:
		return "COMMAND"
	case ModeJump:
		return "JUMP"
	case ModeJSON:
		return "JSON"
	default:
		return "UNKNOWN"
	}
}

// JumpAction represents what to do after a jump selection
type JumpAction int

const (
	JumpActionSelect     JumpAction = iota // Just select the node
	JumpActionEdit                          // Edit the selected node
	JumpActionDelete                        // Delete the selected node
	JumpActionConnectFrom                   // Start connection from this node
	JumpActionConnectTo                     // Complete connection to this node
)

// SetMode changes the editor mode
func (e *TUIEditor) SetMode(mode Mode) {
	e.mode = mode
	
	// Clear jump labels when leaving jump mode
	if mode != ModeJump {
		e.jumpLabels = make(map[int]rune)
	}
	
	// Clear text buffer when entering text modes
	if mode == ModeInsert || mode == ModeEdit {
		e.textBuffer = []rune{}
		e.cursorPos = 0
		
		// If editing existing node, load its text
		if mode == ModeEdit && e.selected >= 0 {
			for _, node := range e.diagram.Nodes {
				if node.ID == e.selected {
					if len(node.Text) > 0 {
						e.textBuffer = []rune(node.Text[0])
						e.cursorPos = len(e.textBuffer)
					}
					break
				}
			}
		}
		
		// If editing existing connection, load its label
		if mode == ModeEdit && e.selectedConnection >= 0 && e.selectedConnection < len(e.diagram.Connections) {
			e.textBuffer = []rune(e.diagram.Connections[e.selectedConnection].Label)
			e.cursorPos = len(e.textBuffer)
		}
	}
}
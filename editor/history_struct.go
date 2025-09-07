package editor

import (
	"edd/diagram"
)

// StructHistory manages undo/redo using direct struct storage (much faster than JSON)
type StructHistory struct {
	states  []*diagram.Diagram // Direct struct pointers
	current int             // Current position in history
	max     int             // Maximum number of states to keep
}

// NewStructHistory creates a new struct-based history manager
func NewStructHistory(max int) *StructHistory {
	if max <= 0 {
		max = 50
	}
	return &StructHistory{
		states:  make([]*diagram.Diagram, 0, max),
		current: -1,
		max:     max,
	}
}

// SaveState saves a new state (creates a deep copy)
func (sh *StructHistory) SaveState(d *diagram.Diagram) error {
	// Create a deep copy of the diagram
	clone := d.Clone()
	
	// If we're not at the end, truncate everything after current
	if sh.current < len(sh.states)-1 {
		sh.states = sh.states[:sh.current+1]
	}
	
	// Add new state
	sh.states = append(sh.states, clone)
	
	// If we exceed max, remove oldest
	if len(sh.states) > sh.max {
		sh.states = sh.states[1:]
	} else {
		sh.current++
	}
	
	return nil
}

// CanUndo returns true if we can undo
func (sh *StructHistory) CanUndo() bool {
	return sh.current > 0
}

// CanRedo returns true if we can redo
func (sh *StructHistory) CanRedo() bool {
	return sh.current < len(sh.states)-1
}

// Undo goes back one state
func (sh *StructHistory) Undo() (*diagram.Diagram, error) {
	if !sh.CanUndo() {
		return nil, nil
	}
	
	sh.current--
	
	// Return a clone to prevent accidental modification of history
	return sh.states[sh.current].Clone(), nil
}

// Redo goes forward one state
func (sh *StructHistory) Redo() (*diagram.Diagram, error) {
	if !sh.CanRedo() {
		return nil, nil
	}
	
	sh.current++
	
	// Return a clone to prevent accidental modification of history
	return sh.states[sh.current].Clone(), nil
}

// Clear clears all history
func (sh *StructHistory) Clear() {
	sh.states = sh.states[:0]
	sh.current = -1
}

// Stats returns current position and total states
func (sh *StructHistory) Stats() (current, total int) {
	return sh.current + 1, len(sh.states)
}
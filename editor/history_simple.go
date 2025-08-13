package editor

import (
	"edd/core"
	"encoding/json"
)

// SimpleHistory manages undo/redo using a simple slice (easier to understand and debug)
type SimpleHistory struct {
	states  []string // JSON states
	current int      // Current position in history
	max     int      // Maximum number of states to keep
}

// NewSimpleHistory creates a new simple history manager
func NewSimpleHistory(max int) *SimpleHistory {
	if max <= 0 {
		max = 50
	}
	return &SimpleHistory{
		states:  make([]string, 0, max),
		current: -1,
		max:     max,
	}
}

// SaveState saves a new state
func (sh *SimpleHistory) SaveState(diagram *core.Diagram) error {
	data, err := json.Marshal(diagram)
	if err != nil {
		return err
	}
	
	// If we're not at the end, truncate everything after current
	if sh.current < len(sh.states)-1 {
		sh.states = sh.states[:sh.current+1]
	}
	
	// Add new state
	sh.states = append(sh.states, string(data))
	
	// If we exceed max, remove oldest
	if len(sh.states) > sh.max {
		sh.states = sh.states[1:]
	} else {
		sh.current++
	}
	
	return nil
}

// CanUndo returns true if we can undo
func (sh *SimpleHistory) CanUndo() bool {
	return sh.current > 0
}

// CanRedo returns true if we can redo
func (sh *SimpleHistory) CanRedo() bool {
	return sh.current < len(sh.states)-1
}

// Undo goes back one state
func (sh *SimpleHistory) Undo() (*core.Diagram, error) {
	if !sh.CanUndo() {
		return nil, nil
	}
	
	sh.current--
	
	var diagram core.Diagram
	if err := json.Unmarshal([]byte(sh.states[sh.current]), &diagram); err != nil {
		return nil, err
	}
	
	return &diagram, nil
}

// Redo goes forward one state
func (sh *SimpleHistory) Redo() (*core.Diagram, error) {
	if !sh.CanRedo() {
		return nil, nil
	}
	
	sh.current++
	
	var diagram core.Diagram
	if err := json.Unmarshal([]byte(sh.states[sh.current]), &diagram); err != nil {
		return nil, err
	}
	
	return &diagram, nil
}

// Clear clears all history
func (sh *SimpleHistory) Clear() {
	sh.states = sh.states[:0]
	sh.current = -1
}

// Stats returns current position and total states
func (sh *SimpleHistory) Stats() (current, total int) {
	return sh.current + 1, len(sh.states)
}
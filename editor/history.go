package editor

import (
	"edd/core"
	"encoding/json"
)

// HistoryManager manages undo/redo state using a ring buffer
type HistoryManager struct {
	states   []string // Ring buffer of JSON diagram states
	head     int      // Current position in the ring buffer
	tail     int      // Oldest entry in the ring buffer
	size     int      // Current number of states stored
	capacity int      // Maximum capacity of the ring buffer
	current  int      // Current position for undo/redo
}

// NewHistoryManager creates a new history manager with a ring buffer
func NewHistoryManager(capacity int) *HistoryManager {
	if capacity <= 0 {
		capacity = 50 // Default to 50 states
	}
	return &HistoryManager{
		states:   make([]string, capacity),
		head:     0,
		tail:     0,
		size:     0,
		capacity: capacity,
		current:  -1,
	}
}

// SaveState saves the current diagram state to the ring buffer
func (h *HistoryManager) SaveState(diagram *core.Diagram) error {
	// Marshal the diagram to JSON for storage
	data, err := json.Marshal(diagram)
	if err != nil {
		return err
	}
	
	state := string(data)
	
	// If we're in the middle of history (after undo), truncate forward history
	if h.current >= 0 && h.current != h.head-1 && h.size > 0 {
		// We're not at the newest state, so we need to truncate
		nextPos := (h.current + 1) % h.capacity
		h.head = nextPos
		
		// Recalculate size
		if h.current >= h.tail {
			h.size = h.current - h.tail + 1
		} else {
			h.size = h.capacity - h.tail + h.current + 1
		}
	}
	
	// Add the new state at head position
	h.states[h.head] = state
	h.current = h.head
	
	// Move head forward for next insertion
	oldHead := h.head
	h.head = (h.head + 1) % h.capacity
	
	// Update size and tail
	if h.size < h.capacity {
		h.size++
	} else {
		// Buffer is full, move tail forward (overwriting oldest)
		if h.head == h.tail {
			h.tail = (h.tail + 1) % h.capacity
		}
	}
	
	// Current stays at the position we just wrote
	h.current = oldHead
	
	return nil
}

// CanUndo returns true if undo is possible
func (h *HistoryManager) CanUndo() bool {
	if h.size <= 1 {
		return false
	}
	
	// Check if we can move backwards from current position
	prevPos := h.current - 1
	if prevPos < 0 {
		prevPos = h.capacity - 1
	}
	
	// Can undo if previous position is valid (not before tail)
	if h.size == h.capacity {
		// Full buffer - check wraparound
		if h.tail <= h.head {
			return prevPos >= h.tail && prevPos < h.head
		} else {
			return prevPos >= h.tail || prevPos < h.head
		}
	} else {
		// Not full - simple check
		return prevPos >= 0 && prevPos < h.size-1
	}
}

// CanRedo returns true if redo is possible
func (h *HistoryManager) CanRedo() bool {
	if h.size == 0 {
		return false
	}
	
	// Can redo if current is not at the most recent state
	nextPos := (h.current + 1) % h.capacity
	
	// Check if next position has valid data
	if h.size < h.capacity {
		return nextPos < h.head && nextPos < h.size
	} else {
		// Full buffer
		return nextPos != h.head
	}
}

// Undo returns the previous state
func (h *HistoryManager) Undo() (*core.Diagram, error) {
	if !h.CanUndo() {
		return nil, nil
	}
	
	// Move current backwards
	h.current--
	if h.current < 0 {
		h.current = h.capacity - 1
	}
	
	return h.loadCurrentState()
}

// Redo returns the next state
func (h *HistoryManager) Redo() (*core.Diagram, error) {
	if !h.CanRedo() {
		return nil, nil
	}
	
	// Move current forward
	h.current = (h.current + 1) % h.capacity
	
	return h.loadCurrentState()
}

// loadCurrentState loads the diagram at the current position
func (h *HistoryManager) loadCurrentState() (*core.Diagram, error) {
	if h.current < 0 || h.size == 0 {
		return nil, nil
	}
	
	var diagram core.Diagram
	if err := json.Unmarshal([]byte(h.states[h.current]), &diagram); err != nil {
		return nil, err
	}
	
	return &diagram, nil
}

// Clear clears the history
func (h *HistoryManager) Clear() {
	h.head = 0
	h.tail = 0
	h.size = 0
	h.current = -1
	// Clear the actual data to help GC
	for i := range h.states {
		h.states[i] = ""
	}
}

// Stats returns current position and total states for display
func (h *HistoryManager) Stats() (current, total int) {
	if h.size == 0 {
		return 0, 0
	}
	
	// Calculate current position in the logical sequence
	var pos int
	if h.current >= h.tail {
		pos = h.current - h.tail + 1
	} else {
		pos = h.capacity - h.tail + h.current + 1
	}
	
	return pos, h.size
}
package editor

import (
	"testing"
)

func TestInvalidKeyInContinuousConnect(t *testing.T) {
	tui := NewTUIEditor(NewRealRenderer())
	
	// Set up diagram with 3 nodes
	tui.AddNode([]string{"A"})
	tui.AddNode([]string{"B"})
	tui.AddNode([]string{"C"})
	
	// Start continuous connect mode
	tui.HandleKey('C')
	
	// Select first node as FROM
	if label, ok := tui.jumpLabels[1]; ok {
		tui.HandleKey(label)
	}
	
	// Should be waiting for TO
	if tui.jumpAction != JumpActionConnectTo {
		t.Errorf("Expected JumpActionConnectTo, got %v", tui.jumpAction)
	}
	
	// Press an invalid key (not a jump label)
	tui.HandleKey('z') // Assuming 'z' is not in the first 3 jump labels
	
	// Should STILL be in jump mode waiting for TO
	if tui.mode != ModeJump {
		t.Errorf("Should still be in ModeJump after invalid key, got %v", tui.mode)
	}
	if tui.jumpAction != JumpActionConnectTo {
		t.Errorf("Should still be JumpActionConnectTo after invalid key, got %v", tui.jumpAction)
	}
	if !tui.continuousConnect {
		t.Error("Should still be in continuous connect mode after invalid key")
	}
	
	// Now press a valid key to complete the connection
	if label, ok := tui.jumpLabels[2]; ok {
		tui.HandleKey(label)
	}
	
	// Should have created the connection and be ready for next TO
	if len(tui.diagram.Connections) != 1 {
		t.Errorf("Expected 1 connection, got %d", len(tui.diagram.Connections))
	}
	if tui.mode != ModeJump {
		t.Errorf("Should still be in ModeJump for next connection, got %v", tui.mode)
	}
	
	// Press more invalid keys
	tui.HandleKey('x')
	tui.HandleKey('y')
	tui.HandleKey('z')
	
	// Should still be waiting for valid input
	if tui.mode != ModeJump {
		t.Errorf("Should still be in ModeJump after multiple invalid keys, got %v", tui.mode)
	}
	if !tui.continuousConnect {
		t.Error("Should still be in continuous connect mode after multiple invalid keys")
	}
	
	// Press ESC to properly exit
	tui.HandleKey(27)
	
	// Now should be in normal mode
	if tui.mode != ModeNormal {
		t.Errorf("Expected ModeNormal after ESC, got %v", tui.mode)
	}
	if tui.continuousConnect {
		t.Error("Continuous connect should be disabled after ESC")
	}
}

func TestInvalidKeyInSingleConnect(t *testing.T) {
	tui := NewTUIEditor(NewRealRenderer())
	
	// Set up diagram with 2 nodes  
	tui.AddNode([]string{"A"})
	tui.AddNode([]string{"B"})
	
	// Start single connect mode (lowercase 'c')
	tui.HandleKey('c')
	
	// Select first node as FROM
	if label, ok := tui.jumpLabels[1]; ok {
		tui.HandleKey(label)
	}
	
	// Press an invalid key
	tui.HandleKey('z')
	
	// In single connect mode, invalid key should cancel and return to normal
	if tui.mode != ModeNormal {
		t.Errorf("Expected ModeNormal after invalid key in single connect, got %v", tui.mode)
	}
	if tui.jumpAction != JumpActionSelect {
		t.Errorf("Jump action should be reset after cancel, got %v", tui.jumpAction)
	}
	
	// Verify no connections were created
	if len(tui.diagram.Connections) != 0 {
		t.Errorf("Expected no connections, got %d", len(tui.diagram.Connections))
	}
}
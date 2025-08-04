package obstacles

import (
	"edd/core"
	"testing"
)

func TestPortManager_BasicOperations(t *testing.T) {
	// Create test nodes
	nodes := []core.Node{
		{ID: 1, X: 10, Y: 10, Width: 10, Height: 5},
		{ID: 2, X: 30, Y: 10, Width: 8, Height: 6},
	}
	
	pm := NewPortManager(nodes, 2) // 2-unit wide ports
	
	t.Run("GetAvailablePorts", func(t *testing.T) {
		// Check available ports on North edge of node 1
		// Width is 10, margin is 2 on each side, so usable space is 6
		// With 2-unit ports, we should have 3 ports
		ports := pm.GetAvailablePorts(1, North)
		if len(ports) != 3 {
			t.Errorf("Expected 3 available ports on North edge, got %d", len(ports))
		}
		
		// Verify port positions
		expectedPositions := []int{2, 4, 6}
		for i, port := range ports {
			if port.Position != expectedPositions[i] {
				t.Errorf("Port %d: expected position %d, got %d", 
					i, expectedPositions[i], port.Position)
			}
			if port.ConnectionID != -1 {
				t.Errorf("Port %d should be free, but has connection %d",
					i, port.ConnectionID)
			}
		}
	})
	
	t.Run("ReservePort", func(t *testing.T) {
		// Reserve a port on East edge
		port, err := pm.ReservePort(1, East, 100)
		if err != nil {
			t.Fatalf("Failed to reserve port: %v", err)
		}
		
		if port.ConnectionID != 100 {
			t.Errorf("Reserved port should have connection ID 100, got %d", 
				port.ConnectionID)
		}
		
		// Check that the port is now occupied
		if !pm.IsPortOccupied(port) {
			t.Error("Reserved port should be occupied")
		}
		
		// Try to get available ports - should not include the reserved one
		availablePorts := pm.GetAvailablePorts(1, East)
		for _, p := range availablePorts {
			if p.Position == port.Position {
				t.Error("Reserved port should not appear in available ports")
			}
		}
	})
	
	t.Run("GetOccupiedPorts", func(t *testing.T) {
		// Should have one occupied port from previous test
		occupied := pm.GetOccupiedPorts(1)
		if len(occupied) != 1 {
			t.Errorf("Expected 1 occupied port, got %d", len(occupied))
		}
		
		if occupied[0].ConnectionID != 100 {
			t.Errorf("Occupied port should have connection ID 100, got %d",
				occupied[0].ConnectionID)
		}
	})
	
	t.Run("ReleasePort", func(t *testing.T) {
		occupied := pm.GetOccupiedPorts(1)
		if len(occupied) == 0 {
			t.Skip("No occupied ports to release")
		}
		
		port := occupied[0]
		pm.ReleasePort(port)
		
		// Port should no longer be occupied
		if pm.IsPortOccupied(port) {
			t.Error("Released port should not be occupied")
		}
		
		// Should have no occupied ports now
		occupied = pm.GetOccupiedPorts(1)
		if len(occupied) != 0 {
			t.Errorf("Expected 0 occupied ports after release, got %d", len(occupied))
		}
	})
	
	t.Run("GetPortForConnection", func(t *testing.T) {
		// Reserve a port
		port, _ := pm.ReservePort(2, South, 200)
		
		// Should be able to find it by connection ID
		foundPort, found := pm.GetPortForConnection(2, 200)
		if !found {
			t.Error("Should find port for connection 200")
		}
		
		if foundPort.Position != port.Position {
			t.Errorf("Found port position %d doesn't match reserved port position %d",
				foundPort.Position, port.Position)
		}
		
		// Should not find non-existent connection
		_, found = pm.GetPortForConnection(2, 999)
		if found {
			t.Error("Should not find port for non-existent connection")
		}
	})
}

func TestPortManager_EdgeCases(t *testing.T) {
	t.Run("SmallNode", func(t *testing.T) {
		// Node too small for any ports with margins
		nodes := []core.Node{
			{ID: 1, X: 0, Y: 0, Width: 3, Height: 3},
		}
		
		pm := NewPortManager(nodes, 2)
		
		// Should have no available ports due to margins
		ports := pm.GetAvailablePorts(1, North)
		if len(ports) != 0 {
			t.Errorf("Small node should have no available ports, got %d", len(ports))
		}
	})
	
	t.Run("NonExistentNode", func(t *testing.T) {
		nodes := []core.Node{
			{ID: 1, X: 0, Y: 0, Width: 10, Height: 10},
		}
		
		pm := NewPortManager(nodes, 2)
		
		// Should handle non-existent node gracefully
		ports := pm.GetAvailablePorts(999, North)
		if ports != nil {
			t.Error("Should return nil for non-existent node")
		}
		
		_, err := pm.ReservePort(999, North, 100)
		if err == nil {
			t.Error("Should return error for non-existent node")
		}
	})
}

func TestPortManager_PortPoints(t *testing.T) {
	nodes := []core.Node{
		{ID: 1, X: 10, Y: 20, Width: 6, Height: 8}, // Increased height to allow ports on E/W edges
	}
	
	pm := NewPortManager(nodes, 2)
	
	testCases := []struct {
		edge     EdgeSide
		expectedY int
		expectedXRange [2]int
	}{
		{North, 19, [2]int{12, 14}}, // Y should be node.Y - 1
		{South, 28, [2]int{12, 14}}, // Y should be node.Y + height (updated for height=8)
		{East, -1, [2]int{16, 16}},  // X should be node.X + width
		{West, -1, [2]int{9, 9}},    // X should be node.X - 1
	}
	
	for _, tc := range testCases {
		ports := pm.GetAvailablePorts(1, tc.edge)
		if len(ports) == 0 {
			t.Errorf("No ports available on %s edge", edgeName(tc.edge))
			continue
		}
		
		port := ports[0]
		
		if tc.expectedY != -1 && port.Point.Y != tc.expectedY {
			t.Errorf("%s edge: expected Y=%d, got Y=%d",
				edgeName(tc.edge), tc.expectedY, port.Point.Y)
		}
		
		if tc.expectedXRange[0] != -1 {
			if port.Point.X < tc.expectedXRange[0] || port.Point.X > tc.expectedXRange[1] {
				t.Errorf("%s edge: expected X in range [%d,%d], got X=%d",
					edgeName(tc.edge), tc.expectedXRange[0], tc.expectedXRange[1], port.Point.X)
			}
		}
	}
}
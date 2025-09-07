package diagram

// EnsureUniqueConnectionIDs ensures all connections in a diagram have unique IDs.
// If connections have missing IDs (all zero) or duplicate IDs, they are reassigned.
func EnsureUniqueConnectionIDs(diagram *Diagram) {
	if diagram == nil || len(diagram.Connections) == 0 {
		return
	}

	// Count occurrences of each ID
	idCount := make(map[int]int)
	allZero := true
	
	for i := range diagram.Connections {
		id := diagram.Connections[i].ID
		idCount[id]++
		if id != 0 {
			allZero = false
		}
	}
	
	// Determine if we need to reassign IDs
	needsReassignment := allZero
	for _, count := range idCount {
		if count > 1 {
			// Duplicate IDs found
			needsReassignment = true
			break
		}
	}
	
	if needsReassignment {
		// Reassign all IDs based on index to ensure uniqueness
		for i := range diagram.Connections {
			diagram.Connections[i].ID = i
		}
	} else {
		// IDs are unique but may have gaps or start from non-zero
		// Check if we need to fill in any missing IDs
		usedIDs := make(map[int]bool)
		var needsID []int
		
		for i := range diagram.Connections {
			if diagram.Connections[i].ID == 0 && len(idCount) > 1 {
				// This connection needs an ID (0 when others are non-zero)
				needsID = append(needsID, i)
			} else {
				usedIDs[diagram.Connections[i].ID] = true
			}
		}
		
		// Assign IDs to connections that need them
		nextID := 0
		for _, i := range needsID {
			// Find next available ID
			for usedIDs[nextID] {
				nextID++
			}
			diagram.Connections[i].ID = nextID
			usedIDs[nextID] = true
		}
	}
}
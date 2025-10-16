# Final Analysis: Label Positioning Bug

## The Root Cause

After extensive investigation, I've identified **TWO CRITICAL MISMATCHES** causing labels to appear on connection lines instead of box corners:

### 1. Inconsistent Sticky Header Activation

**In label_renderer.go:**
```go
if e.diagram.Type == "sequence" && e.diagramScrollOffset > 7 {
    // Sticky headers are active
```

**In tuieditor.go (actual rendering):**
```go
if e.diagram.Type == "sequence" && headerLines > 0 && startLine > headerLines {
    // We're scrolled past the headers - make them sticky
```

The label renderer uses a **hardcoded value of 7**, while the actual rendering uses **dynamically calculated `headerLines`** (which can vary based on the actual header content).

### 2. Wrong Y Coordinate Calculation for Sticky Headers

When sticky headers ARE active, the label calculation is:
```go
viewportY = pos.Y + 1 + scrollIndicatorLines
```

With `pos.Y = 2` and `scrollIndicatorLines = 1`, this gives `viewportY = 4`.

**BUT**, based on actual rendering output:
- Line 1: Scroll indicator
- Line 2: Empty space
- Line 3: Empty space
- Line 4: Box top border (╭──────╮) <- Where label SHOULD be
- Line 5: Participant text (│ Client │) <- Where label APPEARS
- Line 6: Box bottom border (╰──────╯)

The formula should be:
```go
viewportY = pos.Y + 2 + scrollIndicatorLines
```

## The Exact Problem

1. **Participants are positioned at Y=2** in the diagram coordinate system
2. **The box top border renders at Y=2** in diagram coordinates
3. **When converted to viewport coordinates with sticky headers:**
   - Current calculation: `2 + 1 + 1 = 4` (points to line 4 in 0-indexed, but ANSI uses 1-indexed)
   - Actual need: `2 + 2 + 1 = 5` to account for the additional offset in sticky header rendering

## Why This Happens

The rendering pipeline has multiple coordinate systems:
1. **Diagram coordinates**: Where nodes are positioned (Y=2 for participants)
2. **Canvas coordinates**: After applying scroll offset
3. **Viewport coordinates**: After adding UI elements (scroll indicators, headers)
4. **ANSI coordinates**: 1-indexed for terminal display

The label renderer is missing the correct transformation between these systems when sticky headers are active.

## The Fix

In `editor/label_renderer.go`, line 49:
```go
// CURRENT (WRONG):
viewportY = pos.Y + 1 + scrollIndicatorLines

// SHOULD BE:
viewportY = pos.Y + 2 + scrollIndicatorLines
```

AND check for sticky headers should match the actual rendering logic:
```go
// Instead of hardcoded 7, use the actual header size calculation
```

## Additional Issues Found

1. **Connection labels use separate logic** from node labels, potentially causing inconsistencies
2. **Multiple coordinate transformation implementations** exist across the codebase with subtle differences
3. **The header size calculation is duplicated** in multiple places rather than being centralized

## Verification

The test output clearly shows:
- Box top border at line 4 (1-indexed)
- Label calculation gives Y=4 (which in 1-indexed ANSI becomes line 4)
- But the actual box corner is at line 4, not line 5

This off-by-one error is consistent across all scrolled scenarios with sticky headers.
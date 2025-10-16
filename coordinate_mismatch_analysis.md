# Terminal UI Label Rendering Bug Analysis - Third Opinion

## Summary
After thorough investigation, I've identified the **exact coordinate mismatch** causing jump labels to appear on connection lines instead of participant box corners when scrolling in sequence diagrams. The issue is a **simple off-by-one error** in the Y coordinate calculation when sticky headers are active.

## Unicode Handling
✅ **Confirmed correct**: Box-drawing characters are properly used (╭╮╯╰│)
✅ **X positioning is correct**: Labels are placed at `pos.X - 1` for sequence diagrams

## Visual Correctness
**THE PROBLEM**: Labels appear at line 5 (participant text) instead of line 4 (box top border)

### Actual Rendering (from debug output):
```
Line 4: ╭──────────────────╮  <- Box TOP BORDER (where label should be)
Line 5: │ client           │  <- Text (where label currently appears)
Line 6: ╰──────────────────╯  <- Box bottom border
```

### Current Label Calculation (`label_renderer.go` line 49):
```go
viewportY = pos.Y + 2 + scrollIndicatorLines  // Results in Y=5
```

With:
- `pos.Y = 2` (participant's diagram Y coordinate, which is the box TOP)
- `scrollIndicatorLines = 1`
- Result: `2 + 2 + 1 = 5` (points to text line, not top border)

## Coordinate System Analysis

### Diagram Coordinates (0-based):
- Participants are positioned at `Y=2` (this is where the box TOP starts)
- Box structure spans 3 lines:
  - Y=2: Top border
  - Y=3: Text
  - Y=4: Bottom border

### Viewport Coordinates with Sticky Headers (1-based):
- Line 1: Scroll indicator
- Lines 2-3: Empty padding
- Line 4: Box top border (diagram Y=2 mapped to viewport)
- Line 5: Box text (diagram Y=3 mapped to viewport)
- Line 6: Box bottom border (diagram Y=4 mapped to viewport)

### The Mapping Formula Error:
- Current: `viewportY = pos.Y + 2 + scrollIndicatorLines`
- Should be: `viewportY = pos.Y + 1 + scrollIndicatorLines`

## Compatibility Concerns
✅ No terminal compatibility issues - this is purely a coordinate calculation error
✅ ANSI escape sequences are correctly formatted in `RenderLabelsToString`

## Performance Notes
✅ The calculation is O(1) and efficient
✅ The fix is a simple arithmetic adjustment

## Recommendations

### Immediate Fix Required
In `/Users/petegriffiths/code/go/src/edd/editor/label_renderer.go` line 49:

```go
// CURRENT (INCORRECT):
viewportY = pos.Y + 2 + scrollIndicatorLines  // Places label on text line

// SHOULD BE:
viewportY = pos.Y + 1 + scrollIndicatorLines  // Places label on box top border
```

### Why This Fix Works:
1. `pos.Y = 2` represents the box TOP in diagram coordinates
2. Adding 1 accounts for the 0-to-1 based coordinate conversion
3. Adding `scrollIndicatorLines` accounts for the scroll indicator
4. Result: `2 + 1 + 1 = 4` - correctly points to the box top border

### Test Validation:
The test in `screenshot_scenario_test.go` expects `Y=5` but this is actually testing the buggy behavior. After the fix, the test should expect `Y=4` to match the box top border position.

## Root Cause
The confusion stems from:
1. **pos.Y refers to the box TOP**, not the text inside
2. The formula incorrectly adds 2 instead of 1 for the coordinate conversion
3. This makes labels appear on the text line instead of the corner

## Verification
Running the debug program clearly shows:
- Box top border is at viewport line 4
- Labels are calculated for viewport line 5
- This one-line offset is exactly the bug users are experiencing
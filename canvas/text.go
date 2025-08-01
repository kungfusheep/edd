package canvas

import (
	"strings"
)

// MeasureText returns the display width of a string in terminal cells.
func MeasureText(text string) int {
	return StringWidth(text)
}

// WrapMode defines how text wrapping should handle long words.
type WrapMode int

const (
	// WrapModeWord wraps at word boundaries (default).
	WrapModeWord WrapMode = iota
	// WrapModeChar breaks words at character boundaries.
	WrapModeChar
	// WrapModeHyphenate adds hyphens when breaking words.
	WrapModeHyphenate
)

// WrapText wraps text to fit within maxWidth using word boundaries.
func WrapText(text string, maxWidth int) []string {
	return WrapTextMode(text, maxWidth, WrapModeWord)
}

// WrapTextMode wraps text to fit within maxWidth using the specified mode.
func WrapTextMode(text string, maxWidth int, mode WrapMode) []string {
	if maxWidth <= 0 {
		return nil
	}
	
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}
	
	var lines []string
	var currentLine strings.Builder
	currentWidth := 0
	
	for _, word := range words {
		wordWidth := StringWidth(word)
		
		// Word fits on current line
		if currentWidth == 0 || currentWidth+1+wordWidth <= maxWidth {
			if currentWidth > 0 {
				currentLine.WriteRune(' ')
				currentWidth++
			}
			currentLine.WriteString(word)
			currentWidth += wordWidth
			continue
		}
		
		// Word doesn't fit, start new line
		if currentLine.Len() > 0 {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentWidth = 0
		}
		
		// Handle word longer than line width
		if wordWidth > maxWidth {
			switch mode {
			case WrapModeChar:
				// Break at character boundaries
				remaining := word
				for StringWidth(remaining) > maxWidth {
					cutPoint := findCutPoint(remaining, maxWidth)
					if cutPoint == 0 {
						// Can't even fit one character, force it
						cutPoint = len(string([]rune(remaining)[0]))
					}
					lines = append(lines, remaining[:cutPoint])
					remaining = remaining[cutPoint:]
				}
				if len(remaining) > 0 {
					currentLine.WriteString(remaining)
					currentWidth = StringWidth(remaining)
				}
				
			case WrapModeHyphenate:
				// Break with hyphenation
				remaining := word
				hyphenWidth := 1
				for StringWidth(remaining) > maxWidth {
					cutPoint := findCutPoint(remaining, maxWidth-hyphenWidth)
					if cutPoint == 0 {
						// Can't fit with hyphen, try without
						cutPoint = findCutPoint(remaining, maxWidth)
						if cutPoint == 0 {
							cutPoint = len(string([]rune(remaining)[0]))
						}
						lines = append(lines, remaining[:cutPoint])
					} else {
						lines = append(lines, remaining[:cutPoint]+"-")
					}
					remaining = remaining[cutPoint:]
				}
				if len(remaining) > 0 {
					currentLine.WriteString(remaining)
					currentWidth = StringWidth(remaining)
				}
				
			default: // WrapModeWord
				// Force the word on its own line (may overflow)
				lines = append(lines, word)
			}
		} else {
			// Normal case: word starts new line
			currentLine.WriteString(word)
			currentWidth = wordWidth
		}
	}
	
	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}
	
	return lines
}

// findCutPoint finds where to cut a string to fit within maxWidth.
func findCutPoint(s string, maxWidth int) int {
	if maxWidth <= 0 {
		return 0
	}
	
	width := 0
	lastIndex := 0
	
	for i, r := range s {
		charWidth := UnicodeWidth(r)
		if width+charWidth > maxWidth {
			return lastIndex
		}
		width += charWidth
		lastIndex = i + len(string(r))
	}
	
	return len(s)
}

// FitText truncates text to fit within maxWidth, adding ellipsis if needed.
func FitText(text string, maxWidth int, ellipsis string) string {
	textWidth := StringWidth(text)
	if textWidth <= maxWidth {
		return text
	}
	
	ellipsisWidth := StringWidth(ellipsis)
	if maxWidth <= ellipsisWidth {
		return TruncateToWidth(text, maxWidth)
	}
	
	targetWidth := maxWidth - ellipsisWidth
	truncated := TruncateToWidth(text, targetWidth)
	return truncated + ellipsis
}
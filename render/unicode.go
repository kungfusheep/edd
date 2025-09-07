package render

// UnicodeWidth returns the display width of a rune in terminal cells.
// This implementation follows the Unicode East Asian Width property.
func UnicodeWidth(r rune) int {
	// Fast path for ASCII
	if r < 0x80 {
		if r < 0x20 || r == 0x7F {
			return 0 // Control characters
		}
		return 1
	}
	
	// Zero-width characters
	if isZeroWidth(r) {
		return 0
	}
	
	// Wide characters
	if isWideChar(r) {
		return 2
	}
	
	// Everything else is narrow
	return 1
}

// StringWidth returns the display width of a string in terminal cells.
func StringWidth(s string) int {
	width := 0
	for _, r := range s {
		width += UnicodeWidth(r)
	}
	return width
}

// isZeroWidth checks if a rune is a zero-width character.
func isZeroWidth(r rune) bool {
	return (r >= 0x0300 && r <= 0x036F) || // Combining diacritical marks
		(r >= 0x1AB0 && r <= 0x1AFF) || // Combining diacritical marks extended
		(r >= 0x1DC0 && r <= 0x1DFF) || // Combining diacritical marks supplement
		(r >= 0x20D0 && r <= 0x20FF) || // Combining diacritical marks for symbols
		(r >= 0xFE00 && r <= 0xFE0F) || // Variation selectors
		(r >= 0xFE20 && r <= 0xFE2F) || // Combining half marks
		(r >= 0xE0100 && r <= 0xE01EF) || // Variation selectors supplement
		r == 0x200B || // Zero-width space
		r == 0x200C || // Zero-width non-joiner
		r == 0x200D || // Zero-width joiner
		r == 0x2060 || // Word joiner
		r == 0xFEFF    // Zero-width no-break space
}

// isWideChar checks if a rune is a wide character (East Asian Width: Wide or Fullwidth).
func isWideChar(r rune) bool {
	// CJK characters
	if (r >= 0x1100 && r <= 0x115F) || // Hangul Jamo
		(r >= 0x11A3 && r <= 0x11A7) || // Hangul Jamo
		(r >= 0x11FA && r <= 0x11FF) || // Hangul Jamo
		(r >= 0x2329 && r <= 0x232A) || // Left/right pointing angle brackets
		(r >= 0x2E80 && r <= 0x2E99) || // CJK Radicals Supplement
		(r >= 0x2E9B && r <= 0x2EF3) || // CJK Radicals Supplement
		(r >= 0x2F00 && r <= 0x2FD5) || // Kangxi Radicals
		(r >= 0x2FF0 && r <= 0x2FFB) || // Ideographic description characters
		(r >= 0x3000 && r <= 0x303E) || // CJK symbols and punctuation
		(r >= 0x3041 && r <= 0x3096) || // Hiragana
		(r >= 0x3099 && r <= 0x30FF) || // Katakana
		(r >= 0x3105 && r <= 0x312F) || // Bopomofo
		(r >= 0x3131 && r <= 0x318E) || // Hangul Compatibility Jamo
		(r >= 0x3190 && r <= 0x31E3) || // CJK strokes and ideographic
		(r >= 0x31F0 && r <= 0x321E) || // Katakana Phonetic Extensions
		(r >= 0x3220 && r <= 0x3247) || // Enclosed CJK Letters and Months
		(r >= 0x3250 && r <= 0x4DBF) || // Enclosed CJK Letters and CJK Unified Ideographs Extension A
		(r >= 0x4E00 && r <= 0xA48C) || // CJK Unified Ideographs
		(r >= 0xA490 && r <= 0xA4C6) || // Yi Radicals
		(r >= 0xA960 && r <= 0xA97C) || // Hangul Jamo Extended-A
		(r >= 0xAC00 && r <= 0xD7A3) || // Hangul Syllables
		(r >= 0xD7B0 && r <= 0xD7C6) || // Hangul Jamo Extended-B
		(r >= 0xD7CB && r <= 0xD7FB) || // Hangul Jamo Extended-B
		(r >= 0xF900 && r <= 0xFAFF) || // CJK Compatibility Ideographs
		(r >= 0xFE10 && r <= 0xFE19) || // Vertical forms
		(r >= 0xFE30 && r <= 0xFE52) || // CJK Compatibility Forms
		(r >= 0xFE54 && r <= 0xFE66) || // Small Form Variants
		(r >= 0xFE68 && r <= 0xFE6B) || // Small Form Variants
		(r >= 0xFF01 && r <= 0xFF60) || // Fullwidth ASCII and punctuation
		(r >= 0xFFE0 && r <= 0xFFE6) || // Fullwidth symbol variants
		(r >= 0x16FE0 && r <= 0x16FE4) || // Tangut components
		(r >= 0x16FF0 && r <= 0x16FF1) || // Vietnamese alternate reading marks
		(r >= 0x17000 && r <= 0x187F7) || // Tangut
		(r >= 0x18800 && r <= 0x18CD5) || // Tangut components
		(r >= 0x18D00 && r <= 0x18D08) || // Tangut Supplement
		(r >= 0x1AFF0 && r <= 0x1B0FF) || // Kana Extended
		(r >= 0x1B150 && r <= 0x1B152) || // Small Kana Extension
		(r >= 0x1B164 && r <= 0x1B167) || // Small Kana Extension
		(r >= 0x1B170 && r <= 0x1B2FB) || // Nushu
		(r >= 0x1F004 && r == 0x1F004) || // Mahjong tile
		(r >= 0x1F0CF && r == 0x1F0CF) || // Playing card
		(r >= 0x1F18E && r == 0x1F18E) || // Negative squared AB
		(r >= 0x1F191 && r <= 0x1F19A) || // Squared CJK Unified Ideographs
		(r >= 0x1F200 && r <= 0x1F320) || // Enclosed Ideographic Supplement
		(r >= 0x1F32D && r <= 0x1F335) || // Enclosed Ideographic Supplement
		(r >= 0x1F337 && r <= 0x1F37C) || // Enclosed Ideographic Supplement
		(r >= 0x1F37E && r <= 0x1F393) || // Enclosed Ideographic Supplement
		(r >= 0x1F3A0 && r <= 0x1F3CA) || // Enclosed Ideographic Supplement
		(r >= 0x1F3CF && r <= 0x1F3D3) || // Enclosed Ideographic Supplement
		(r >= 0x1F3E0 && r <= 0x1F3F0) || // Enclosed Ideographic Supplement
		(r >= 0x1F3F4 && r == 0x1F3F4) || // Waving black flag
		(r >= 0x1F3F8 && r <= 0x1F3FA) || // Enclosed Ideographic Supplement
		(r >= 0x1F3FB && r <= 0x1F3FF) || // Emoji skin tone modifiers
		(r >= 0x1F400 && r <= 0x1F6FF) || // Emoji
		(r >= 0x1F7E0 && r <= 0x1F7EB) || // Geometric shapes extended
		(r >= 0x1F90C && r <= 0x1F9FF) || // Supplemental symbols and pictographs
		(r >= 0x1FA70 && r <= 0x1FA74) || // Symbols and Pictographs Extended-A
		(r >= 0x1FA78 && r <= 0x1FA7C) || // Symbols and Pictographs Extended-A
		(r >= 0x1FA80 && r <= 0x1FA86) || // Symbols and Pictographs Extended-A
		(r >= 0x1FA90 && r <= 0x1FAAC) || // Symbols and Pictographs Extended-A
		(r >= 0x1FAB0 && r <= 0x1FABA) || // Symbols and Pictographs Extended-A
		(r >= 0x1FAC0 && r <= 0x1FAC5) || // Symbols and Pictographs Extended-A
		(r >= 0x1FAD0 && r <= 0x1FAD9) || // Symbols and Pictographs Extended-A
		(r >= 0x1FAE0 && r <= 0x1FAE7) || // Symbols and Pictographs Extended-A
		(r >= 0x1FAF0 && r <= 0x1FAF6) || // Symbols and Pictographs Extended-A
		(r >= 0x20000 && r <= 0x2FFFD) || // CJK Unified Ideographs Extension B and others
		(r >= 0x30000 && r <= 0x3FFFD) {  // CJK Unified Ideographs Extension G
		return true
	}
	return false
}

// TruncateToWidth truncates a string to fit within the specified width.
func TruncateToWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	
	width := 0
	lastValidIndex := 0
	
	for i, r := range s {
		charWidth := UnicodeWidth(r)
		if width+charWidth > maxWidth {
			break
		}
		width += charWidth
		lastValidIndex = i + len(string(r))
	}
	
	return s[:lastValidIndex]
}
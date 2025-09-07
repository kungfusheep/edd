// Package rendering provides path rendering with terminal awareness.
package render

import (
	"os"
	"strings"
)

// UnicodeLevel represents the level of Unicode support.
type UnicodeLevel int

const (
	UnicodeNone     UnicodeLevel = iota // ASCII only
	UnicodeBasic                        // Basic box-drawing
	UnicodeExtended                     // Full box-drawing with rounded corners
	UnicodeFull                         // Including emoji, complex scripts
)

// TerminalCapabilities represents the features supported by the current terminal.
type TerminalCapabilities struct {
	Name            string
	UnicodeLevel    UnicodeLevel
	SupportsColor   bool
	ColorDepth      int  // 0, 8, 256, or 24-bit
	BoxDrawingWidth int  // 1 for normal, 2 for some CJK terminals
	IsCJK           bool // CJK environment detected
}

// DetectCapabilities detects the current terminal's capabilities.
func DetectCapabilities() TerminalCapabilities {
	// Allow override via environment variable
	if forceMode := os.Getenv("EDD_TERMINAL_MODE"); forceMode != "" {
		switch forceMode {
		case "ascii":
			return ForceASCII()
		case "unicode":
			return ForceUnicode()
		}
	}
	
	caps := TerminalCapabilities{
		Name:            "unknown",
		UnicodeLevel:    UnicodeBasic, // Default to basic Unicode
		SupportsColor:   false,
		ColorDepth:      0,
		BoxDrawingWidth: 1,
		IsCJK:           false,
	}
	
	// Detect specific terminals first
	if detectSpecificTerminal(&caps) {
		// Terminal was specifically identified
	} else {
		// Fall back to TERM environment variable
		term := os.Getenv("TERM")
		caps.Name = term
		
		// Check for color support
		if term != "" && !strings.Contains(term, "dumb") {
			// Most modern terminals support color
			if strings.Contains(term, "256color") {
				caps.SupportsColor = true
				caps.ColorDepth = 256
			} else if strings.Contains(term, "color") {
				caps.SupportsColor = true
				caps.ColorDepth = 8
			}
			// xterm and variants usually support color
			if strings.HasPrefix(term, "xterm") || strings.HasPrefix(term, "screen") {
				caps.SupportsColor = true
				if caps.ColorDepth == 0 {
					caps.ColorDepth = 256
				}
			}
		}
		
		// Check for 24-bit color
		if colorterm := os.Getenv("COLORTERM"); colorterm != "" {
			if colorterm == "truecolor" || colorterm == "24bit" {
				caps.ColorDepth = 24
			}
		}
	}
	
	// Check for NO_COLOR environment variable (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" {
		caps.SupportsColor = false
		caps.ColorDepth = 0
	}
	
	// Check for Unicode support
	hasUTF8 := detectUTF8Locale()
	
	// Detect CJK environment
	caps.IsCJK = detectCJKEnvironment()
	if caps.IsCJK {
		caps.BoxDrawingWidth = 2
	}
	
	// Determine Unicode level based on terminal and locale
	if !hasUTF8 || caps.Name == "linux" || caps.Name == "dumb" {
		caps.UnicodeLevel = UnicodeNone
	} else if caps.Name == "windows-terminal" || caps.Name == "iterm2" || caps.Name == "kitty" {
		caps.UnicodeLevel = UnicodeFull
	} else if strings.Contains(caps.Name, "xterm") || caps.Name == "alacritty" {
		caps.UnicodeLevel = UnicodeExtended
	}
	
	// Check for SSH session - might need more conservative settings
	if os.Getenv("SSH_CLIENT") != "" || os.Getenv("SSH_TTY") != "" || os.Getenv("SSH_CONNECTION") != "" {
		// Be more conservative over SSH unless we're sure about UTF-8
		if !hasUTF8 && caps.UnicodeLevel > UnicodeBasic {
			caps.UnicodeLevel = UnicodeBasic
		}
	}
	
	// Check for common CI environments that might not support Unicode
	if os.Getenv("CI") != "" || os.Getenv("CONTINUOUS_INTEGRATION") != "" {
		// Many CI environments have limited Unicode support
		if caps.Name == "" || caps.Name == "dumb" {
			caps.UnicodeLevel = UnicodeNone
		}
	}
	
	return caps
}

// detectSpecificTerminal checks for specific terminal emulators.
func detectSpecificTerminal(caps *TerminalCapabilities) bool {
	// Check Windows Terminal
	if os.Getenv("WT_SESSION") != "" {
		caps.Name = "windows-terminal"
		caps.UnicodeLevel = UnicodeFull
		caps.SupportsColor = true
		caps.ColorDepth = 24
		return true
	}
	
	// Check terminal program environment variable
	termProgram := os.Getenv("TERM_PROGRAM")
	switch termProgram {
	case "iTerm.app":
		caps.Name = "iterm2"
		caps.UnicodeLevel = UnicodeFull
		caps.SupportsColor = true
		caps.ColorDepth = 24
		return true
	case "Apple_Terminal":
		caps.Name = "terminal.app"
		caps.UnicodeLevel = UnicodeExtended
		caps.SupportsColor = true
		caps.ColorDepth = 256
		return true
	}
	
	// Check for VTE-based terminals (GNOME Terminal, etc.)
	if os.Getenv("VTE_VERSION") != "" {
		caps.Name = "vte-based"
		caps.UnicodeLevel = UnicodeExtended
		caps.SupportsColor = true
		caps.ColorDepth = 24
		return true
	}
	
	// Check for Konsole
	if os.Getenv("KONSOLE_VERSION") != "" {
		caps.Name = "konsole"
		caps.UnicodeLevel = UnicodeExtended
		caps.SupportsColor = true
		caps.ColorDepth = 24
		return true
	}
	
	// Check for Alacritty
	if term := os.Getenv("TERM"); term == "alacritty" {
		caps.Name = "alacritty"
		caps.UnicodeLevel = UnicodeExtended
		caps.SupportsColor = true
		caps.ColorDepth = 24
		return true
	}
	
	// Check for kitty
	if term := os.Getenv("TERM"); strings.HasPrefix(term, "xterm-kitty") {
		caps.Name = "kitty"
		caps.UnicodeLevel = UnicodeFull
		caps.SupportsColor = true
		caps.ColorDepth = 24
		return true
	}
	
	// Check for tmux
	if os.Getenv("TMUX") != "" {
		caps.Name = "tmux"
		caps.UnicodeLevel = UnicodeExtended
		caps.SupportsColor = true
		caps.ColorDepth = 256
		return true
	}
	
	// Check for rxvt-unicode
	if term := os.Getenv("TERM"); strings.HasPrefix(term, "rxvt-unicode") {
		caps.Name = "rxvt-unicode"
		caps.UnicodeLevel = UnicodeExtended
		caps.SupportsColor = true
		caps.ColorDepth = 256
		return true
	}
	
	// Check for WezTerm
	if os.Getenv("WEZTERM_EXECUTABLE") != "" {
		caps.Name = "wezterm"
		caps.UnicodeLevel = UnicodeFull
		caps.SupportsColor = true
		caps.ColorDepth = 24
		return true
	}
	
	return false
}

// detectUTF8Locale checks if the locale supports UTF-8.
func detectUTF8Locale() bool {
	for _, env := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		value := os.Getenv(env)
		if value == "" {
			continue
		}
		
		// Handle C.UTF-8, en_US.UTF-8, en_US.UTF-8@euro, etc.
		// Split by '.' and then handle potential @ modifier
		parts := strings.Split(value, ".")
		if len(parts) > 1 {
			charsetPart := parts[1]
			// Remove any @ modifier
			if idx := strings.Index(charsetPart, "@"); idx != -1 {
				charsetPart = charsetPart[:idx]
			}
			if strings.EqualFold(charsetPart, "UTF-8") || strings.EqualFold(charsetPart, "UTF8") {
				return true
			}
		}
		
		// Also check for UTF anywhere in the string
		upperValue := strings.ToUpper(value)
		if strings.Contains(upperValue, "UTF-8") || strings.Contains(upperValue, "UTF8") {
			return true
		}
	}
	
	return false
}

// detectCJKEnvironment checks for CJK (Chinese, Japanese, Korean) environment.
func detectCJKEnvironment() bool {
	// Check locale for CJK languages
	for _, env := range []string{"LANG", "LC_ALL", "LC_CTYPE"} {
		value := os.Getenv(env)
		if value != "" {
			// Check language prefix
			prefix := strings.Split(value, "_")[0]
			if prefix == "ja" || prefix == "ko" || prefix == "zh" {
				return true
			}
		}
	}
	
	// Check for CJK-specific environment variables
	if os.Getenv("CJK_WIDTH") == "2" {
		return true
	}
	
	// Check terminal name for CJK hints
	term := os.Getenv("TERM")
	if strings.Contains(strings.ToLower(term), "cjk") {
		return true
	}
	
	// Check for East Asian ambiguous width setting
	if os.Getenv("EAST_ASIAN_AMBIGUOUS") == "2" {
		return true
	}
	
	return false
}

// ForceASCII returns capabilities configured for ASCII-only output.
func ForceASCII() TerminalCapabilities {
	return TerminalCapabilities{
		Name:            "ascii",
		UnicodeLevel:    UnicodeNone,
		SupportsColor:   false,
		ColorDepth:      0,
		BoxDrawingWidth: 1,
		IsCJK:           false,
	}
}

// ForceUnicode returns capabilities configured for full Unicode support.
func ForceUnicode() TerminalCapabilities {
	return TerminalCapabilities{
		Name:            "unicode",
		UnicodeLevel:    UnicodeFull,
		SupportsColor:   true,
		ColorDepth:      24,
		BoxDrawingWidth: 1,
		IsCJK:           false,
	}
}
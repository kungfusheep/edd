package rendering

import (
	"os"
	"testing"
)

func TestDetectCapabilities(t *testing.T) {
	// Save original environment
	origEnv := map[string]string{
		"TERM":               os.Getenv("TERM"),
		"LANG":               os.Getenv("LANG"),
		"LC_ALL":             os.Getenv("LC_ALL"),
		"LC_CTYPE":           os.Getenv("LC_CTYPE"),
		"WT_SESSION":         os.Getenv("WT_SESSION"),
		"TERM_PROGRAM":       os.Getenv("TERM_PROGRAM"),
		"COLORTERM":          os.Getenv("COLORTERM"),
		"SSH_CLIENT":         os.Getenv("SSH_CLIENT"),
		"SSH_CONNECTION":     os.Getenv("SSH_CONNECTION"),
		"CI":                 os.Getenv("CI"),
		"NO_COLOR":           os.Getenv("NO_COLOR"),
		"TMUX":               os.Getenv("TMUX"),
		"WEZTERM_EXECUTABLE": os.Getenv("WEZTERM_EXECUTABLE"),
	}
	
	// Restore environment after tests
	defer func() {
		for k, v := range origEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()
	
	tests := []struct {
		name     string
		setup    map[string]string
		expected TerminalCapabilities
	}{
		{
			name: "Windows Terminal",
			setup: map[string]string{
				"WT_SESSION": "some-session-id",
				"LANG":       "en_US.UTF-8",
			},
			expected: TerminalCapabilities{
				Name:            "windows-terminal",
				UnicodeLevel:    UnicodeFull,
				SupportsColor:   true,
				ColorDepth:      24,
				BoxDrawingWidth: 1,
				IsCJK:           false,
			},
		},
		{
			name: "iTerm2",
			setup: map[string]string{
				"TERM_PROGRAM": "iTerm.app",
				"TERM":         "xterm-256color",
				"LANG":         "en_US.UTF-8",
			},
			expected: TerminalCapabilities{
				Name:            "iterm2",
				UnicodeLevel:    UnicodeFull,
				SupportsColor:   true,
				ColorDepth:      24,
				BoxDrawingWidth: 1,
				IsCJK:           false,
			},
		},
		{
			name: "Linux console (no Unicode)",
			setup: map[string]string{
				"TERM": "linux",
				"LANG": "C",
			},
			expected: TerminalCapabilities{
				Name:            "linux",
				UnicodeLevel:    UnicodeNone,
				SupportsColor:   false,
				ColorDepth:      0,
				BoxDrawingWidth: 1,
				IsCJK:           false,
			},
		},
		{
			name: "SSH with UTF-8",
			setup: map[string]string{
				"TERM":       "xterm-256color",
				"LANG":       "en_US.UTF-8",
				"SSH_CLIENT": "192.168.1.100 12345 22",
			},
			expected: TerminalCapabilities{
				Name:            "xterm-256color",
				UnicodeLevel:    UnicodeExtended,
				SupportsColor:   true,
				ColorDepth:      256,
				BoxDrawingWidth: 1,
				IsCJK:           false,
			},
		},
		{
			name: "CJK environment",
			setup: map[string]string{
				"TERM": "xterm",
				"LANG": "ja_JP.UTF-8",
			},
			expected: TerminalCapabilities{
				Name:            "xterm",
				UnicodeLevel:    UnicodeExtended,
				SupportsColor:   true,
				ColorDepth:      256,
				BoxDrawingWidth: 2,
				IsCJK:           true,
			},
		},
		{
			name: "CI environment",
			setup: map[string]string{
				"TERM": "dumb",
				"CI":   "true",
			},
			expected: TerminalCapabilities{
				Name:            "dumb",
				UnicodeLevel:    UnicodeNone,
				SupportsColor:   false,
				ColorDepth:      0,
				BoxDrawingWidth: 1,
				IsCJK:           false,
			},
		},
		{
			name: "24-bit color terminal",
			setup: map[string]string{
				"TERM":      "xterm-256color",
				"COLORTERM": "truecolor",
				"LANG":      "en_US.UTF-8",
			},
			expected: TerminalCapabilities{
				Name:            "xterm-256color",
				UnicodeLevel:    UnicodeExtended,
				SupportsColor:   true,
				ColorDepth:      24,
				BoxDrawingWidth: 1,
				IsCJK:           false,
			},
		},
		{
			name: "tmux session",
			setup: map[string]string{
				"TERM": "screen-256color",
				"TMUX": "/tmp/tmux-1000/default,1234,0",
				"LANG": "en_US.UTF-8",
			},
			expected: TerminalCapabilities{
				Name:            "tmux",
				UnicodeLevel:    UnicodeExtended,
				SupportsColor:   true,
				ColorDepth:      256,
				BoxDrawingWidth: 1,
				IsCJK:           false,
			},
		},
		{
			name: "WezTerm",
			setup: map[string]string{
				"TERM":              "xterm-256color",
				"WEZTERM_EXECUTABLE": "/Applications/WezTerm.app/Contents/MacOS/wezterm",
				"LANG":              "en_US.UTF-8",
			},
			expected: TerminalCapabilities{
				Name:            "wezterm",
				UnicodeLevel:    UnicodeFull,
				SupportsColor:   true,
				ColorDepth:      24,
				BoxDrawingWidth: 1,
				IsCJK:           false,
			},
		},
		{
			name: "NO_COLOR environment",
			setup: map[string]string{
				"TERM":      "xterm-256color",
				"COLORTERM": "truecolor",
				"NO_COLOR":  "1",
				"LANG":      "en_US.UTF-8",
			},
			expected: TerminalCapabilities{
				Name:            "xterm-256color",
				UnicodeLevel:    UnicodeExtended,
				SupportsColor:   false,
				ColorDepth:      0,
				BoxDrawingWidth: 1,
				IsCJK:           false,
			},
		},
		{
			name: "Locale with modifier",
			setup: map[string]string{
				"TERM": "xterm",
				"LANG": "en_US.UTF-8@euro",
			},
			expected: TerminalCapabilities{
				Name:            "xterm",
				UnicodeLevel:    UnicodeExtended,
				SupportsColor:   true,
				ColorDepth:      256,
				BoxDrawingWidth: 1,
				IsCJK:           false,
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			for k := range origEnv {
				os.Unsetenv(k)
			}
			
			// Set up test environment
			for k, v := range tt.setup {
				os.Setenv(k, v)
			}
			
			// Test detection
			caps := DetectCapabilities()
			
			// Check results
			if caps.Name != tt.expected.Name {
				t.Errorf("Name: got %q, want %q", caps.Name, tt.expected.Name)
			}
			if caps.UnicodeLevel != tt.expected.UnicodeLevel {
				t.Errorf("UnicodeLevel: got %v, want %v", caps.UnicodeLevel, tt.expected.UnicodeLevel)
			}
			if caps.SupportsColor != tt.expected.SupportsColor {
				t.Errorf("SupportsColor: got %v, want %v", caps.SupportsColor, tt.expected.SupportsColor)
			}
			if caps.ColorDepth != tt.expected.ColorDepth {
				t.Errorf("ColorDepth: got %d, want %d", caps.ColorDepth, tt.expected.ColorDepth)
			}
			if caps.BoxDrawingWidth != tt.expected.BoxDrawingWidth {
				t.Errorf("BoxDrawingWidth: got %d, want %d", caps.BoxDrawingWidth, tt.expected.BoxDrawingWidth)
			}
			if caps.IsCJK != tt.expected.IsCJK {
				t.Errorf("IsCJK: got %v, want %v", caps.IsCJK, tt.expected.IsCJK)
			}
		})
	}
}

func TestForceOverrides(t *testing.T) {
	// Test ForceASCII
	ascii := ForceASCII()
	if ascii.UnicodeLevel != UnicodeNone {
		t.Errorf("ForceASCII: UnicodeLevel should be UnicodeNone")
	}
	if ascii.SupportsColor {
		t.Errorf("ForceASCII: should not support color")
	}
	
	// Test ForceUnicode
	unicode := ForceUnicode()
	if unicode.UnicodeLevel != UnicodeFull {
		t.Errorf("ForceUnicode: UnicodeLevel should be UnicodeFull")
	}
	if !unicode.SupportsColor {
		t.Errorf("ForceUnicode: should support color")
	}
	if unicode.ColorDepth != 24 {
		t.Errorf("ForceUnicode: ColorDepth should be 24")
	}
}

func TestEnvironmentOverride(t *testing.T) {
	// Save and restore
	orig := os.Getenv("EDD_TERMINAL_MODE")
	defer func() {
		if orig == "" {
			os.Unsetenv("EDD_TERMINAL_MODE")
		} else {
			os.Setenv("EDD_TERMINAL_MODE", orig)
		}
	}()
	
	// Test ASCII override
	os.Setenv("EDD_TERMINAL_MODE", "ascii")
	caps := DetectCapabilities()
	if caps.UnicodeLevel != UnicodeNone {
		t.Error("EDD_TERMINAL_MODE=ascii should force ASCII mode")
	}
	
	// Test Unicode override
	os.Setenv("EDD_TERMINAL_MODE", "unicode")
	caps = DetectCapabilities()
	if caps.UnicodeLevel != UnicodeFull {
		t.Error("EDD_TERMINAL_MODE=unicode should force Unicode mode")
	}
}
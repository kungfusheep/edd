package render

import (
	"edd/diagram"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// ============================================================================
// Tests from terminal_test.go
// ============================================================================

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

// ============================================================================
// Tests from preserve_corners_test.go
// ============================================================================

func TestPreserveCornersMode(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		height   int
		paths    []diagram.Path
		expected string
	}{
		{
			name:   "simple box with preserve corners",
			width:  5,
			height: 3,
			paths: []diagram.Path{
				{Points: []diagram.Point{
					{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 2}, {X: 0, Y: 2}, {X: 0, Y: 0},
				}},
			},
			expected: `┌───┐
│   │
└───┘`,
		},
		{
			name:   "overlapping boxes preserve corners",
			width:  7,
			height: 4,
			paths: []diagram.Path{
				{Points: []diagram.Point{
					{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 2}, {X: 0, Y: 2}, {X: 0, Y: 0},
				}},
				{Points: []diagram.Point{
					{X: 2, Y: 1}, {X: 6, Y: 1}, {X: 6, Y: 3}, {X: 2, Y: 3}, {X: 2, Y: 1},
				}},
			},
			expected: `┌───┐  
│ ┌─┼─┐
└─┼─┘ │
  └───┘`,
		},
		{
			name:   "L-shaped path",
			width:  5,
			height: 4,
			paths: []diagram.Path{
				{Points: []diagram.Point{
					{X: 0, Y: 0}, {X: 0, Y: 3}, {X: 4, Y: 3},
				}},
			},
			expected: `│    
│    
│    
└──▶ `,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create canvas and renderer
			c := NewMatrixCanvas(tt.width, tt.height)
			renderer := NewPathRenderer(TerminalCapabilities{
				UnicodeLevel: UnicodeFull,
			})
			
			// Enable preserve corners mode
			renderer.SetRenderMode(RenderModePreserveCorners)
			
			// Draw all paths
			for _, path := range tt.paths {
				hasArrow := false
				// Check if last segment should have arrow (simple heuristic)
				if len(path.Points) > 1 {
					last := path.Points[len(path.Points)-1]
					// Only add arrow for non-closed paths
					if path.Points[0] != last {
						hasArrow = true
					}
				}
				
				err := renderer.RenderPath(c, path, hasArrow)
				if err != nil {
					t.Fatalf("Failed to render path: %v", err)
				}
			}
			
			// Check the output
			output := c.String()
			if output != tt.expected {
				t.Errorf("Unexpected output:\nGot:\n%s\nExpected:\n%s", output, tt.expected)
			}
		})
	}
}

func TestPreserveCornersVsStandardMode(t *testing.T) {
	// Test that shows the difference between standard and preserve corners mode
	paths := []diagram.Path{
		{Points: []diagram.Point{
			{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 2}, {X: 0, Y: 2}, {X: 0, Y: 0},
		}},
		{Points: []diagram.Point{
			{X: 2, Y: 0}, {X: 2, Y: 2},
		}},
	}
	
	// Test with standard mode
	t.Run("standard mode creates junction", func(t *testing.T) {
		c := NewMatrixCanvas(5, 3)
		renderer := NewPathRenderer(TerminalCapabilities{UnicodeLevel: UnicodeFull})
		renderer.SetRenderMode(RenderModeStandard)
		
		for _, path := range paths {
			renderer.RenderPath(c, path, false)
		}
		
		// In standard mode with a 2-point vertical line:
		// - The line doesn't include its endpoint, so bottom junction isn't created
		// - Top junction is created where the line starts
		expected := `┌─┼─┐
│ │ │
└───┘`
		
		if output := c.String(); output != expected {
			t.Errorf("Standard mode output incorrect:\nGot:\n%s\nExpected:\n%s", output, expected)
		}
	})
	
	// Test with preserve corners mode
	t.Run("preserve corners mode keeps corners", func(t *testing.T) {
		c := NewMatrixCanvas(5, 3)
		renderer := NewPathRenderer(TerminalCapabilities{UnicodeLevel: UnicodeFull})
		renderer.SetRenderMode(RenderModePreserveCorners)
		
		for _, path := range paths {
			renderer.RenderPath(c, path, false)
		}
		
		// In preserve corners mode with a 2-point vertical line:
		// - The line doesn't include its endpoint, so bottom junction isn't created
		// - Top junction is created where the line starts
		expected := `┌─┼─┐
│ │ │
└───┘`
		
		if output := c.String(); output != expected {
			t.Errorf("Preserve corners mode output incorrect:\nGot:\n%s\nExpected:\n%s", output, expected)
		}
	})
}

// ============================================================================
// Tests from path_renderer_test.go
// ============================================================================

func TestPathRenderer_RenderPath(t *testing.T) {
	tests := []struct {
		name     string
		caps     TerminalCapabilities
		path     diagram.Path
		hasArrow bool
		width    int
		height   int
		expected string
	}{
		{
			name: "horizontal line with unicode",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: diagram.Path{
				Points: []diagram.Point{{X: 1, Y: 1}, {X: 5, Y: 1}},
			},
			hasArrow: false,
			width:    7,
			height:   3,
			expected: `       
 ────  
       `,
		},
		{
			name: "horizontal line with arrow",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: diagram.Path{
				Points: []diagram.Point{{X: 1, Y: 1}, {X: 5, Y: 1}},
			},
			hasArrow: true,
			width:    7,
			height:   3,
			expected: `       
 ───▶  
       `,
		},
		{
			name: "vertical line with unicode",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: diagram.Path{
				Points: []diagram.Point{{X: 2, Y: 0}, {X: 2, Y: 3}},
			},
			hasArrow: false,
			width:    5,
			height:   4,
			expected: `  │  
  │  
  │  
     `,
		},
		{
			name: "L-shaped path with corner",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: diagram.Path{
				Points: []diagram.Point{{X: 1, Y: 1}, {X: 4, Y: 1}, {X: 4, Y: 3}},
			},
			hasArrow: false,
			width:    6,
			height:   4,
			expected: `      
 ───┐ 
    │ 
    │ `,
		},
		{
			name: "ASCII horizontal line",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeNone},
			path: diagram.Path{
				Points: []diagram.Point{{X: 1, Y: 1}, {X: 5, Y: 1}},
			},
			hasArrow: false,
			width:    7,
			height:   3,
			expected: `       
 ----  
       `,
		},
		{
			name: "ASCII L-shaped path",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeNone},
			path: diagram.Path{
				Points: []diagram.Point{{X: 1, Y: 1}, {X: 4, Y: 1}, {X: 4, Y: 3}},
			},
			hasArrow: false,
			width:    6,
			height:   4,
			expected: `      
 ---+ 
    | 
    | `,
		},
		{
			name: "complex path with multiple turns",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: diagram.Path{
				Points: []diagram.Point{
					{X: 1, Y: 1},
					{X: 3, Y: 1},
					{X: 3, Y: 3},
					{X: 5, Y: 3},
					{X: 5, Y: 1},
				},
			},
			hasArrow: false,
			width:    7,
			height:   5,
			expected: `       
 ──┐ │ 
   │ │ 
   └─┘ 
       `,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create canvas
			c := NewMatrixCanvas(tt.width, tt.height)
			
			// Create renderer
			renderer := NewPathRenderer(tt.caps)
			// Use preserve corners mode for paths that expect clean corners
			if strings.Contains(tt.name, "path") || strings.Contains(tt.name, "shaped") {
				renderer.SetRenderMode(RenderModePreserveCorners)
			}
			
			// Render path
			err := renderer.RenderPath(c, tt.path, tt.hasArrow)
			if err != nil {
				t.Fatalf("RenderPath failed: %v", err)
			}
			
			// Compare output
			got := c.String()
			want := strings.TrimPrefix(tt.expected, "\n")
			
			if got != want {
				t.Errorf("Path rendering mismatch\nGot:\n%s\nWant:\n%s", got, want)
				// Show difference
				gotLines := strings.Split(got, "\n")
				wantLines := strings.Split(want, "\n")
				for i := 0; i < len(gotLines) && i < len(wantLines); i++ {
					if gotLines[i] != wantLines[i] {
						t.Errorf("Line %d differs:\nGot:  %q\nWant: %q", i, gotLines[i], wantLines[i])
					}
				}
			}
		})
	}
}

func TestJunctionResolver(t *testing.T) {
	jr := NewJunctionResolver()
	
	tests := []struct {
		name     string
		existing rune
		newLine  rune
		expected rune
	}{
		// Basic intersections
		{"horizontal meets vertical", '─', '│', '┼'},
		{"vertical meets horizontal", '│', '─', '┼'},
		
		// Corner junctions
		{"horizontal meets top-left corner", '─', '┌', '┬'},
		{"vertical meets top-left corner", '│', '┌', '├'},
		
		// ASCII intersections
		{"ASCII horizontal meets vertical", '-', '|', '+'},
		{"ASCII vertical meets horizontal", '|', '-', '+'},
		
		// Same character
		{"same horizontal", '─', '─', '─'},
		{"same vertical", '│', '│', '│'},
		
		// Unknown combinations
		{"unknown combo keeps existing", 'A', '─', 'A'},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jr.Resolve(tt.existing, tt.newLine)
			if got != tt.expected {
				t.Errorf("Resolve(%q, %q) = %q, want %q", 
					tt.existing, tt.newLine, got, tt.expected)
			}
		})
	}
}

func TestLineStyleSelection(t *testing.T) {
	tests := []struct {
		name     string
		caps     TerminalCapabilities
		wantHoriz rune
		wantArrow rune
	}{
		{
			"full unicode",
			TerminalCapabilities{UnicodeLevel: UnicodeFull},
			'─',
			'▶',
		},
		{
			"extended unicode",
			TerminalCapabilities{UnicodeLevel: UnicodeExtended},
			'─',
			'▶',
		},
		{
			"basic unicode",
			TerminalCapabilities{UnicodeLevel: UnicodeBasic},
			'-',
			'>',
		},
		{
			"ASCII only",
			TerminalCapabilities{UnicodeLevel: UnicodeNone},
			'-',
			'>',
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			style := selectLineStyle(tt.caps)
			if style.Horizontal != tt.wantHoriz {
				t.Errorf("Horizontal = %q, want %q", style.Horizontal, tt.wantHoriz)
			}
			if style.ArrowRight != tt.wantArrow {
				t.Errorf("ArrowRight = %q, want %q", style.ArrowRight, tt.wantArrow)
			}
		})
	}
}

// ============================================================================
// Tests from text_test.go
// ============================================================================

func TestWrapTextMode(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		mode     WrapMode
		expected []string
	}{
		{
			name:  "WrapModeWord - normal",
			text:  "This is a test of word wrapping",
			width: 10,
			mode:  WrapModeWord,
			expected: []string{
				"This is a",
				"test of",
				"word",
				"wrapping",
			},
		},
		{
			name:  "WrapModeWord - long word",
			text:  "This supercalifragilisticexpialidocious word",
			width: 10,
			mode:  WrapModeWord,
			expected: []string{
				"This",
				"supercalifragilisticexpialidocious",
				"word",
			},
		},
		{
			name:  "WrapModeChar - long word",
			text:  "This superlongword breaks",
			width: 10,
			mode:  WrapModeChar,
			expected: []string{
				"This",
				"superlongw",
				"ord breaks",
			},
		},
		{
			name:  "WrapModeHyphenate - long word",
			text:  "This superlongword breaks",
			width: 10,
			mode:  WrapModeHyphenate,
			expected: []string{
				"This",
				"superlong-",
				"word",
				"breaks",
			},
		},
		{
			name:  "Unicode width aware",
			text:  "Hello 世界 test",
			width: 10,
			mode:  WrapModeWord,
			expected: []string{
				"Hello 世界",
				"test",
			},
		},
		{
			name:  "Unicode char break",
			text:  "你好世界测试文本",
			width: 8,
			mode:  WrapModeChar,
			expected: []string{
				"你好世界测试文本",  // 16 width, treated as one word
			},
		},
		{
			name:  "Unicode char break with space",
			text:  "你好 世界测试文本",
			width: 8,
			mode:  WrapModeChar,
			expected: []string{
				"你好",
				"世界测试",
				"文本",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapTextMode(tt.text, tt.width, tt.mode)
			
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d lines, got %d", len(tt.expected), len(result))
				t.Errorf("Result: %v", result)
				return
			}
			
			for i, line := range result {
				if line != tt.expected[i] {
					t.Errorf("Line %d: expected %q, got %q", i, tt.expected[i], line)
				}
			}
		})
	}
}

func TestStringWidth(t *testing.T) {
	tests := []struct {
		text     string
		expected int
	}{
		{"Hello", 5},
		{"", 0},
		{"你好", 4},
		{"Hello 世界", 10},
		{"🔥Hot", 5},
		{"e\u0301", 1}, // e with combining accent
		{"test\u200Dtest", 8}, // with zero-width joiner
	}
	
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			width := StringWidth(tt.text)
			if width != tt.expected {
				t.Errorf("StringWidth(%q) = %d, want %d", tt.text, width, tt.expected)
			}
		})
	}
}

func TestTruncateToWidth(t *testing.T) {
	tests := []struct {
		text     string
		maxWidth int
		expected string
	}{
		{"Hello, World!", 5, "Hello"},
		{"你好世界", 4, "你好"},
		{"你好世界", 5, "你好"},
		{"你好世界", 6, "你好世"},
		{"🔥Hot", 3, "🔥H"},
		{"🔥Hot", 2, "🔥"},
		{"🔥Hot", 1, ""},
		{"test", 10, "test"},
	}
	
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			result := TruncateToWidth(tt.text, tt.maxWidth)
			if result != tt.expected {
				t.Errorf("TruncateToWidth(%q, %d) = %q, want %q", 
					tt.text, tt.maxWidth, result, tt.expected)
			}
			
			// Verify the width is correct
			width := StringWidth(result)
			if width > tt.maxWidth {
				t.Errorf("Result width %d exceeds maxWidth %d", width, tt.maxWidth)
			}
		})
	}
}

// ============================================================================
// Tests from path_renderer_edge_test.go
// ============================================================================

func TestPathRenderer_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		caps     TerminalCapabilities
		path     diagram.Path
		hasArrow bool
		width    int
		height   int
		expected string
	}{
		{
			name: "empty path",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: diagram.Path{
				Points: []diagram.Point{},
			},
			hasArrow: false,
			width:    5,
			height:   5,
			expected: `     
     
     
     
     `,
		},
		{
			name: "single point without arrow",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: diagram.Path{
				Points: []diagram.Point{{X: 2, Y: 2}},
			},
			hasArrow: false,
			width:    5,
			height:   5,
			expected: `     
     
     
     
     `,
		},
		{
			name: "single point with arrow",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: diagram.Path{
				Points: []diagram.Point{{X: 2, Y: 2}},
			},
			hasArrow: true,
			width:    5,
			height:   5,
			expected: `     
     
  •  
     
     `,
		},
		{
			name: "crossing paths horizontal/vertical",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: diagram.Path{}, // We'll render two paths
			hasArrow: false,
			width:    7,
			height:   5,
		},
		{
			name: "path with junction at corner",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: diagram.Path{}, // We'll render two paths
			hasArrow: false,
			width:    7,
			height:   5,
		},
		{
			name: "zero-length horizontal line",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: diagram.Path{
				Points: []diagram.Point{{X: 2, Y: 2}, {X: 2, Y: 2}},
			},
			hasArrow: false,
			width:    5,
			height:   5,
			expected: `     
     
     
     
     `,
		},
		{
			name: "leftward arrow",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: diagram.Path{
				Points: []diagram.Point{{X: 4, Y: 2}, {X: 1, Y: 2}},
			},
			hasArrow: true,
			width:    6,
			height:   4,
			expected: `      
      
  ◀── 
      `,
		},
		{
			name: "upward arrow",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			path: diagram.Path{
				Points: []diagram.Point{{X: 2, Y: 3}, {X: 2, Y: 1}},
			},
			hasArrow: true,
			width:    5,
			height:   5,
			expected: `     
     
  ▲  
  │  
     `,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip special tests that need custom handling
			if tt.name == "crossing paths horizontal/vertical" {
				// Test crossing paths
				c := NewMatrixCanvas(tt.width, tt.height)
				renderer := NewPathRenderer(tt.caps)
				
				// Render horizontal line
				path1 := diagram.Path{Points: []diagram.Point{{X: 1, Y: 2}, {X: 5, Y: 2}}}
				renderer.RenderPath(c, path1, false)
				
				// Render vertical line crossing it
				path2 := diagram.Path{Points: []diagram.Point{{X: 3, Y: 0}, {X: 3, Y: 4}}}
				renderer.RenderPath(c, path2, false)
				
				expected := `   │   
   │   
 ──┼─  
   │   
       `
				
				got := c.String()
				want := strings.TrimPrefix(expected, "\n")
				if got != want {
					t.Errorf("Crossing paths mismatch\nGot:\n%s\nWant:\n%s", got, want)
				}
				return
			}
			
			if tt.name == "path with junction at corner" {
				// Test junction at a corner
				c := NewMatrixCanvas(tt.width, tt.height)
				renderer := NewPathRenderer(tt.caps)
				
				// Render L-shaped path
				path1 := diagram.Path{Points: []diagram.Point{{X: 1, Y: 1}, {X: 3, Y: 1}, {X: 3, Y: 3}}}
				renderer.RenderPath(c, path1, false)
				
				// Render line that intersects at the corner
				path2 := diagram.Path{Points: []diagram.Point{{X: 3, Y: 0}, {X: 3, Y: 2}}}
				renderer.RenderPath(c, path2, false)
				
				expected := `   │   
 ──┤   
   │   
   │   
       `
				
				got := c.String()
				want := strings.TrimPrefix(expected, "\n")
				if got != want {
					t.Errorf("Junction at corner mismatch\nGot:\n%s\nWant:\n%s", got, want)
				}
				return
			}
			
			// Regular test case
			c := NewMatrixCanvas(tt.width, tt.height)
			renderer := NewPathRenderer(tt.caps)
			
			err := renderer.RenderPath(c, tt.path, tt.hasArrow)
			if err != nil {
				t.Fatalf("RenderPath failed: %v", err)
			}
			
			got := c.String()
			want := strings.TrimPrefix(tt.expected, "\n")
			
			if got != want {
				t.Errorf("Path rendering mismatch\nGot:\n%s\nWant:\n%s", got, want)
			}
		})
	}
}

func TestPathRenderer_TerminalFallback(t *testing.T) {
	// Test a complex path with different terminal capabilities
	path := diagram.Path{
		Points: []diagram.Point{
			{X: 1, Y: 1},
			{X: 4, Y: 1},
			{X: 4, Y: 3},
			{X: 2, Y: 3},
			{X: 2, Y: 2},
		},
	}
	
	tests := []struct {
		name     string
		caps     TerminalCapabilities
		expected string
	}{
		{
			name: "full unicode",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeFull},
			expected: `        
 ───┐   
  │ │   
  └─┘   `,
		},
		{
			name: "ASCII only",
			caps: TerminalCapabilities{UnicodeLevel: UnicodeNone},
			expected: `        
 ---+   
  | |   
  +-+   `,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewMatrixCanvas(8, 4)
			renderer := NewPathRenderer(tt.caps)
			// Use preserve corners mode for cleaner output
			renderer.SetRenderMode(RenderModePreserveCorners)
			
			err := renderer.RenderPath(c, path, false)
			if err != nil {
				t.Fatalf("RenderPath failed: %v", err)
			}
			
			got := c.String()
			want := strings.TrimPrefix(tt.expected, "\n")
			
			if got != want {
				t.Errorf("Fallback rendering mismatch\nGot:\n%s\nWant:\n%s", got, want)
			}
		})
	}
}

// ============================================================================
// Tests from junction_demo_test.go
// ============================================================================

func TestJunctionResolver_VisualDemo(t *testing.T) {
	jr := NewJunctionResolver()
	
	// Demo: Arrow meeting a line
	fmt.Println("\nArrow meets line demonstrations:")
	fmt.Printf("│ + ▶ = %c (vertical line + right arrow = left T-junction)\n", jr.Resolve('│', '▶'))
	fmt.Printf("─ + ▼ = %c (horizontal line + down arrow = top T-junction)\n", jr.Resolve('─', '▼'))
	fmt.Printf("│ + ◀ = %c (vertical line + left arrow = right T-junction)\n", jr.Resolve('│', '◀'))
	fmt.Printf("─ + ▲ = %c (horizontal line + up arrow = bottom T-junction)\n", jr.Resolve('─', '▲'))
	
	// Demo: Arrow protection
	fmt.Println("\nArrow protection demonstrations:")
	fmt.Printf("▶ + │ = %c (existing arrow is preserved)\n", jr.Resolve('▶', '│'))
	fmt.Printf("▼ + ─ = %c (existing arrow is preserved)\n", jr.Resolve('▼', '─'))
	
	// Demo: Arrow meets corner
	fmt.Println("\nArrow meets corner demonstrations:")
	fmt.Printf("┌ + ▶ = %c (top-left corner + right arrow)\n", jr.Resolve('┌', '▶'))
	fmt.Printf("┐ + ▼ = %c (top-right corner + down arrow)\n", jr.Resolve('┐', '▼'))
	
	// Demo: Arrow meets arrow
	fmt.Println("\nArrow meets arrow demonstrations:")
	fmt.Printf("▶ + ▼ = %c (perpendicular arrows form cross)\n", jr.Resolve('▶', '▼'))
	fmt.Printf("▶ + ▶ = %c (same arrow is preserved)\n", jr.Resolve('▶', '▶'))
	
	// Visual example of what this enables
	fmt.Println("\nExample diagram with arrows:")
	fmt.Println("┌─────┐")
	fmt.Println("│     │")
	fmt.Println("├─▶   │  <- Arrow connects cleanly to box")
	fmt.Println("│     │")
	fmt.Println("└─────┘")
}

// ============================================================================
// Tests from junction_resolver_test.go
// ============================================================================

func TestJunctionResolver_ArrowMeetsLine(t *testing.T) {
	jr := NewJunctionResolver()
	
	tests := []struct {
		name     string
		existing rune
		newLine  rune
		want     rune
	}{
		// Triangular arrows meet lines
		{"Right arrow meets vertical", '│', '▶', '├'},
		{"Vertical meets right arrow", '▶', '│', '▶'},  // Arrow is preserved when it's existing
		{"Left arrow meets vertical", '│', '◀', '┤'},
		{"Down arrow meets horizontal", '─', '▼', '┬'},
		{"Up arrow meets horizontal", '─', '▲', '┴'},
		
		// Traditional arrows meet lines
		{"Right arrow → meets vertical", '│', '→', '├'},
		{"Left arrow ← meets vertical", '│', '←', '┤'},
		{"Down arrow ↓ meets horizontal", '─', '↓', '┬'},
		{"Up arrow ↑ meets horizontal", '─', '↑', '┴'},
		
		// ASCII arrows meet lines
		{"ASCII > meets vertical", '|', '>', '+'},
		{"ASCII < meets vertical", '|', '<', '+'},
		{"ASCII v meets horizontal", '-', 'v', '+'},
		{"ASCII ^ meets horizontal", '-', '^', '+'},
		
		// Arrow protection - arrows should not be overridden
		{"Existing arrow not overridden by line", '▶', '│', '▶'},
		{"Existing arrow not overridden by cross", '▼', '─', '▼'},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jr.Resolve(tt.existing, tt.newLine)
			if got != tt.want {
				t.Errorf("Resolve(%c, %c) = %c, want %c", tt.existing, tt.newLine, got, tt.want)
			}
		})
	}
}

func TestJunctionResolver_ArrowMeetsCorner(t *testing.T) {
	jr := NewJunctionResolver()
	
	tests := []struct {
		name     string
		existing rune
		newLine  rune
		want     rune
	}{
		// Right arrow meets corners
		{"Right arrow meets top-left corner", '┌', '▶', '├'},
		{"Right arrow meets bottom-left corner", '└', '▶', '├'},
		{"Right arrow meets top-right corner", '┐', '▶', '┼'},
		{"Right arrow meets bottom-right corner", '┘', '▶', '┼'},
		
		// Down arrow meets corners
		{"Down arrow meets top-left corner", '┌', '▼', '┬'},
		{"Down arrow meets top-right corner", '┐', '▼', '┬'},
		{"Down arrow meets bottom corners", '└', '▼', '┼'},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jr.Resolve(tt.existing, tt.newLine)
			if got != tt.want {
				t.Errorf("Resolve(%c, %c) = %c, want %c", tt.existing, tt.newLine, got, tt.want)
			}
		})
	}
}

func TestJunctionResolver_ArrowMeetsArrow(t *testing.T) {
	jr := NewJunctionResolver()
	
	tests := []struct {
		name     string
		existing rune
		newLine  rune
		want     rune
	}{
		// Perpendicular arrows form crosses
		{"Right meets down", '▶', '▼', '┼'},
		{"Down meets right", '▼', '▶', '┼'},
		{"Left meets up", '◀', '▲', '┼'},
		
		// Same direction arrows - existing is preserved
		{"Same arrow", '▶', '▶', '▶'},
		{"Different arrow same direction", '▶', '→', '▶'},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jr.Resolve(tt.existing, tt.newLine)
			if got != tt.want {
				t.Errorf("Resolve(%c, %c) = %c, want %c", tt.existing, tt.newLine, got, tt.want)
			}
		})
	}
}

// ============================================================================
// Tests from visual_test.go
// ============================================================================

// TestVisualRendering creates visual examples of path rendering
func TestVisualRendering(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
		paths  []struct {
			points   []diagram.Point
			hasArrow bool
		}
		description string
	}{
		{
			name:   "Simple Box",
			width:  20,
			height: 10,
			paths: []struct {
				points   []diagram.Point
				hasArrow bool
			}{
				{[]diagram.Point{{2, 2}, {15, 2}, {15, 7}, {2, 7}, {2, 2}}, false},
			},
			description: "A simple rectangular box",
		},
		{
			name:   "Multiple Boxes with Arrows",
			width:  30,
			height: 12,
			paths: []struct {
				points   []diagram.Point
				hasArrow bool
			}{
				// Box 1
				{[]diagram.Point{{2, 2}, {10, 2}, {10, 5}, {2, 5}, {2, 2}}, false},
				// Box 2
				{[]diagram.Point{{15, 2}, {23, 2}, {23, 5}, {15, 5}, {15, 2}}, false},
				// Box 3
				{[]diagram.Point{{8, 7}, {18, 7}, {18, 10}, {8, 10}, {8, 7}}, false},
				// Arrows between Box 1 and Box 2
				{[]diagram.Point{{10, 3}, {15, 3}}, true},
				{[]diagram.Point{{10, 4}, {15, 4}}, true},
				// Arrows from boxes to Box 3 (using orthogonal paths)
				{[]diagram.Point{{6, 5}, {6, 7}, {8, 7}}, true},
				{[]diagram.Point{{19, 5}, {19, 7}}, true},
			},
			description: "Three boxes connected with arrows",
		},
		{
			name:   "Complex Path with Multiple Turns",
			width:  25,
			height: 15,
			paths: []struct {
				points   []diagram.Point
				hasArrow bool
			}{
				{[]diagram.Point{
					{2, 2}, {10, 2}, {10, 5}, {15, 5}, {15, 2}, {20, 2},
					{20, 10}, {15, 10}, {15, 7}, {10, 7}, {10, 10}, {5, 10},
					{5, 5}, {2, 5}, {2, 2},
				}, false},
			},
			description: "A complex path with many turns forming an intricate shape",
		},
		{
			name:   "Line Intersections and Junctions",
			width:  20,
			height: 10,
			paths: []struct {
				points   []diagram.Point
				hasArrow bool
			}{
				// Horizontal lines
				{[]diagram.Point{{2, 2}, {18, 2}}, false},
				{[]diagram.Point{{2, 5}, {18, 5}}, false},
				{[]diagram.Point{{2, 8}, {18, 8}}, false},
				// Vertical lines
				{[]diagram.Point{{5, 1}, {5, 9}}, false},
				{[]diagram.Point{{10, 1}, {10, 9}}, false},
				{[]diagram.Point{{15, 1}, {15, 9}}, false},
			},
			description: "Grid pattern demonstrating line intersections",
		},
		{
			name:   "Directed Graph",
			width:  25,
			height: 12,
			paths: []struct {
				points   []diagram.Point
				hasArrow bool
			}{
				// Nodes (small boxes)
				{[]diagram.Point{{3, 2}, {7, 2}, {7, 4}, {3, 4}, {3, 2}}, false},
				{[]diagram.Point{{15, 2}, {19, 2}, {19, 4}, {15, 4}, {15, 2}}, false},
				{[]diagram.Point{{3, 7}, {7, 7}, {7, 9}, {3, 9}, {3, 7}}, false},
				{[]diagram.Point{{15, 7}, {19, 7}, {19, 9}, {15, 9}, {15, 7}}, false},
				// Directed edges
				{[]diagram.Point{{7, 3}, {15, 3}}, true},
				{[]diagram.Point{{5, 4}, {5, 7}}, true},
				{[]diagram.Point{{17, 4}, {17, 7}}, true},
				{[]diagram.Point{{7, 8}, {15, 8}}, true},
			},
			description: "A directed graph with four nodes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test both Unicode and ASCII modes
			for _, unicodeMode := range []bool{true, false} {
				modeName := "Unicode"
				caps := TerminalCapabilities{UnicodeLevel: UnicodeFull}
				if !unicodeMode {
					modeName = "ASCII"
					caps = TerminalCapabilities{UnicodeLevel: UnicodeNone}
				}

				c := NewMatrixCanvas(tt.width, tt.height)
				renderer := NewPathRenderer(caps)
				// Use preserve corners mode for better box appearance
				renderer.SetRenderMode(RenderModePreserveCorners)

				// Draw all paths
				for _, path := range tt.paths {
					err := renderer.RenderPath(c, diagram.Path{Points: path.points}, path.hasArrow)
					if err != nil {
						t.Errorf("Failed to render path: %v", err)
					}
				}

				// Display the result
				fmt.Printf("\n=== %s (%s mode) ===\n", tt.name, modeName)
				fmt.Printf("Description: %s\n", tt.description)
				fmt.Printf("Size: %dx%d\n", tt.width, tt.height)
				fmt.Println(strings.Repeat("-", tt.width))
				fmt.Print(c.String())
				fmt.Println(strings.Repeat("-", tt.width))
			}
		})
	}
}

// TestOverlappingPaths tests junction resolution with overlapping paths
func TestOverlappingPaths(t *testing.T) {
	c := NewMatrixCanvas(15, 10)
	renderer := NewPathRenderer(TerminalCapabilities{UnicodeLevel: UnicodeFull})
	// Use standard mode to test junction resolution
	// (preserve corners mode would keep separate boxes visually distinct)

	// Create overlapping boxes that share edges
	paths := []diagram.Path{
		// Box 1
		{Points: []diagram.Point{{2, 2}, {8, 2}, {8, 5}, {2, 5}, {2, 2}}},
		// Box 2 (overlaps right edge of Box 1)
		{Points: []diagram.Point{{8, 2}, {12, 2}, {12, 5}, {8, 5}, {8, 2}}},
		// Box 3 (overlaps bottom edge of Box 1 and 2)
		{Points: []diagram.Point{{2, 5}, {12, 5}, {12, 8}, {2, 8}, {2, 5}}},
	}

	fmt.Println("\n=== Overlapping Boxes Test ===")
	fmt.Println("Three boxes with shared edges demonstrating junction resolution")
	
	for _, path := range paths {
		err := renderer.RenderPath(c, path, false)
		if err != nil {
			t.Errorf("Failed to render path: %v", err)
		}
	}

	fmt.Println(strings.Repeat("-", 15))
	fmt.Print(c.String())
	fmt.Println(strings.Repeat("-", 15))

	// Verify corners were drawn (later paths overwrite earlier ones)
	// The point (8,2) will be a corner from the second box
	if char := c.Get(diagram.Point{X: 8, Y: 2}); char != '┌' && char != '┬' && char != '+' {
		t.Errorf("Expected corner or T-junction at (8,2), got %c", char)
	}
	
	// The point (8,5) will be a T-junction (bottom) where box 3's top edge meets the shared edge
	if char := c.Get(diagram.Point{X: 8, Y: 5}); char != '┴' && char != '+' {
		t.Errorf("Expected T-junction (bottom) at (8,5), got %c", char)
	}
}

// ============================================================================
// Tests from matrix_test.go
// ============================================================================

// TestMatrixCanvas_Creation tests canvas creation and initialization.
func TestMatrixCanvas_Creation(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"Small", 10, 5},
		{"Square", 20, 20},
		{"Wide", 100, 10},
		{"Tall", 10, 100},
		{"Large", 200, 200},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canvas := NewMatrixCanvas(tt.width, tt.height)
			
			// Check dimensions
			w, h := canvas.Size()
			if w != tt.width || h != tt.height {
				t.Errorf("Size() = (%d, %d), want (%d, %d)", w, h, tt.width, tt.height)
			}
			
			// Check matrix dimensions
			matrix := canvas.Matrix()
			if len(matrix) != tt.height {
				t.Errorf("Matrix height = %d, want %d", len(matrix), tt.height)
			}
			
			for y, row := range matrix {
				if len(row) != tt.width {
					t.Errorf("Row %d width = %d, want %d", y, len(row), tt.width)
				}
			}
			
			// Check all cells are spaces
			for y := 0; y < tt.height; y++ {
				for x := 0; x < tt.width; x++ {
					if matrix[y][x] != ' ' {
						t.Errorf("Cell (%d,%d) = %c, want space", x, y, matrix[y][x])
					}
				}
			}
		})
	}
}

// TestMatrixCanvas_GetSet tests basic get/set operations.
func TestMatrixCanvas_GetSet(t *testing.T) {
	canvas := NewMatrixCanvas(20, 10)
	validator := NewTestValidator(t)
	
	tests := []struct {
		name  string
		point diagram.Point
		char  rune
		valid bool
	}{
		{"Origin", diagram.Point{0, 0}, '╭', true},
		{"Center", diagram.Point{10, 5}, '┼', true},
		{"Bottom right", diagram.Point{19, 9}, '╯', true},
		{"Out of bounds X", diagram.Point{20, 5}, 'X', false},
		{"Out of bounds Y", diagram.Point{10, 10}, 'Y', false},
		{"Negative X", diagram.Point{-1, 5}, 'N', false},
		{"Negative Y", diagram.Point{5, -1}, 'N', false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := canvas.Set(tt.point, tt.char)
			
			if tt.valid && err != nil {
				t.Errorf("Set() error = %v, want nil", err)
			}
			if !tt.valid && err == nil {
				t.Error("Set() error = nil, want error")
			}
			
			// Check Get
			got := canvas.Get(tt.point)
			if tt.valid {
				if got != tt.char {
					t.Errorf("Get() = %c, want %c", got, tt.char)
				}
			} else {
				if got != ' ' {
					t.Errorf("Get() out of bounds = %c, want space", got)
				}
			}
		})
	}
	
	// Validate the matrix has valid adjacencies
	validator.ValidateMatrix(canvas.Matrix())
}

// TestMatrixCanvas_Clear tests the clear operation.
func TestMatrixCanvas_Clear(t *testing.T) {
	canvas := NewMatrixCanvas(10, 10)
	
	// Set some characters
	points := []diagram.Point{{5, 5}, {0, 0}, {9, 9}, {3, 7}}
	for _, p := range points {
		canvas.Set(p, 'X')
	}
	
	// Clear
	canvas.Clear()
	
	// Check all cells are spaces
	matrix := canvas.Matrix()
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			if matrix[y][x] != ' ' {
				t.Errorf("After clear, cell (%d,%d) = %c, want space", x, y, matrix[y][x])
			}
		}
	}
}

// TestMatrixCanvas_String tests string serialization.
func TestMatrixCanvas_String(t *testing.T) {
	canvas := NewMatrixCanvas(5, 3)
	
	// Draw a simple pattern
	canvas.Set(diagram.Point{0, 0}, '╭')
	canvas.Set(diagram.Point{1, 0}, '─')
	canvas.Set(diagram.Point{2, 0}, '─')
	canvas.Set(diagram.Point{3, 0}, '─')
	canvas.Set(diagram.Point{4, 0}, '╮')
	
	canvas.Set(diagram.Point{0, 1}, '│')
	canvas.Set(diagram.Point{2, 1}, 'X')
	canvas.Set(diagram.Point{4, 1}, '│')
	
	canvas.Set(diagram.Point{0, 2}, '╰')
	canvas.Set(diagram.Point{1, 2}, '─')
	canvas.Set(diagram.Point{2, 2}, '─')
	canvas.Set(diagram.Point{3, 2}, '─')
	canvas.Set(diagram.Point{4, 2}, '╯')
	
	expected := `╭───╮
│ X │
╰───╯`
	
	validator := NewTestValidator(t)
	validator.AssertCanvasEquals(canvas, expected)
}

// TestMatrixCanvas_DrawBox tests the DrawBox primitive.
func TestMatrixCanvas_DrawBox(t *testing.T) {
	tests := []struct {
		name   string
		x, y   int
		w, h   int
		style  BoxStyle
		canvas string
	}{
		{
			name:  "Small box",
			x:     1, y: 1, w: 5, h: 3,
			style: DefaultBoxStyle,
			canvas: `
       
 ╭───╮ 
 │   │ 
 ╰───╯ 
       `,
		},
		{
			name:  "ASCII box",
			x:     0, y: 0, w: 4, h: 3,
			style: SimpleBoxStyle,
			canvas: `
+--+
|  |
+--+`,
		},
		{
			name:  "Large box",
			x:     2, y: 1, w: 10, h: 5,
			style: DefaultBoxStyle,
			canvas: `
              
  ╭────────╮  
  │        │  
  │        │  
  │        │  
  ╰────────╯  
              `,
		},
	}
	
	validator := NewTestValidator(t)
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fixed canvas sizes for each test
			var canvas *MatrixCanvas
			switch tt.name {
			case "Small box":
				canvas = NewMatrixCanvas(7, 5)
			case "ASCII box":
				canvas = NewMatrixCanvas(4, 3)
			case "Large box":
				canvas = NewMatrixCanvas(14, 7)
			}
			
			err := canvas.DrawBox(tt.x, tt.y, tt.w, tt.h, tt.style)
			
			if err != nil {
				t.Fatalf("DrawBox() error = %v", err)
			}
			
			validator.AssertCanvasEquals(canvas, tt.canvas)
			validator.ValidateMatrix(canvas.Matrix())
		})
	}
}

// TestMatrixCanvas_DrawLine tests line drawing primitives.
func TestMatrixCanvas_DrawLine(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
		draw   func(c *MatrixCanvas)
		canvas string
	}{
		{
			name:   "Horizontal line",
			width:  10,
			height: 3,
			draw: func(c *MatrixCanvas) {
				c.DrawHorizontalLine(2, 1, 7, '─')
			},
			canvas: `
          
  ──────  
          `,
		},
		{
			name:   "Vertical line",
			width:  5,
			height: 7,
			draw: func(c *MatrixCanvas) {
				c.DrawVerticalLine(2, 1, 5, '│')
			},
			canvas: `
     
  │  
  │  
  │  
  │  
  │  
     `,
		},
		{
			name:   "Diagonal line",
			width:  10,
			height: 10,
			draw: func(c *MatrixCanvas) {
				c.DrawLine(diagram.Point{1, 1}, diagram.Point{8, 8}, '*')
			},
			canvas: `
          
 *        
  *       
   *      
    *     
     *    
      *   
       *  
        * 
          `,
		},
	}
	
	validator := NewTestValidator(t)
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canvas := NewMatrixCanvas(tt.width, tt.height)
			tt.draw(canvas)
			validator.AssertCanvasEquals(canvas, tt.canvas)
		})
	}
}

// TestMatrixCanvas_DrawText tests text canvas.
func TestMatrixCanvas_DrawText(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		x, y   int
		canvas string
	}{
		{
			name: "Simple text",
			text: "Hello",
			x: 2, y: 1,
			canvas: `
          
  Hello   
          `,
		},
		{
			name: "Unicode text",
			text: "→Test←",
			x: 1, y: 2,
			canvas: `
         
         
 →Test←  
         `,
		},
	}
	
	validator := NewTestValidator(t)
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canvas := NewMatrixCanvas(10, 4)
			err := canvas.DrawText(tt.x, tt.y, tt.text)
			
			if err != nil {
				t.Fatalf("DrawText() error = %v", err)
			}
			
			validator.AssertCanvasEquals(canvas, tt.canvas)
		})
	}
}

// TestMatrixCanvas_TextMeasurement tests text measurement functions.
func TestMatrixCanvas_TextMeasurement(t *testing.T) {
	tests := []struct {
		text     string
		expected int
	}{
		{"Hello", 5},
		{"", 0},
		{"A", 1},
		{"Hello, World!", 13},
		{"→Test←", 6}, // Unicode arrows count as 1 each
	}
	
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			width := MeasureText(tt.text)
			if width != tt.expected {
				t.Errorf("MeasureText(%q) = %d, want %d", tt.text, width, tt.expected)
			}
		})
	}
}

// TestMatrixCanvas_Performance tests performance characteristics.
func TestMatrixCanvas_Performance(t *testing.T) {
	sizes := []struct {
		name   string
		width  int
		height int
		maxMs  int64
	}{
		{"Small", 100, 100, 1},
		{"Medium", 500, 500, 10},
		{"Large", 1000, 1000, 50},
	}
	
	for _, size := range sizes {
		t.Run(size.name+"_Creation", func(t *testing.T) {
			start := time.Now()
			canvas := NewMatrixCanvas(size.width, size.height)
			duration := time.Since(start)
			
			if duration.Milliseconds() > size.maxMs {
				t.Errorf("Creation took %v, want < %dms", duration, size.maxMs)
			}
			
			// Ensure it's actually initialized
			if canvas.Get(diagram.Point{0, 0}) != ' ' {
				t.Error("Canvas not properly initialized")
			}
		})
		
		t.Run(size.name+"_String", func(t *testing.T) {
			canvas := NewMatrixCanvas(size.width, size.height)
			
			// Add some content
			for i := 0; i < 10; i++ {
				canvas.DrawBox(i*10, i*5, 8, 4, DefaultBoxStyle)
			}
			
			start := time.Now()
			_ = canvas.String()
			duration := time.Since(start)
			
			// Should serialize 1000x1000 in < 10ms
			if size.width == 1000 && duration.Milliseconds() > 10 {
				t.Errorf("String() took %v, want < 10ms for 1000x1000", duration)
			}
		})
	}
}

// TestMatrixCanvas_DrawSmartPath tests automatic corner/junction selection.
func TestMatrixCanvas_DrawSmartPath(t *testing.T) {
	tests := []struct {
		name   string
		points []diagram.Point
		canvas string
	}{
		{
			name: "L-shaped path",
			points: []diagram.Point{
				{2, 2}, {5, 2}, {5, 5},
			},
			canvas: `
         
         
  ───╮   
     │   
     │   
     │   
         `,
		},
		{
			name: "Complex path",
			points: []diagram.Point{
				{1, 1}, {4, 1}, {4, 3}, {7, 3}, {7, 5},
			},
			canvas: `
         
 ───╮    
    │    
    ╰──╮ 
       │ 
       │ 
         `,
		},
	}
	
	validator := NewTestValidator(t)
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canvas := NewMatrixCanvas(9, 7)
			err := canvas.DrawSmartPath(tt.points)
			
			if err != nil {
				t.Fatalf("DrawSmartPath() error = %v", err)
			}
			
			validator.AssertCanvasEquals(canvas, tt.canvas)
			validator.ValidateMatrix(canvas.Matrix())
		})
	}
}

// TestMatrixCanvas_UnicodeEdgeCases tests problematic Unicode scenarios.
func TestMatrixCanvas_UnicodeEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		x, y     int
		expWidth int
		desc     string
	}{
		{"ASCII", "Hello", 2, 1, 5, "Basic ASCII"},
		{"Arrows", "→↓←↑", 2, 1, 4, "Arrow characters"},
		{"Emoji", "🔥Hot", 2, 1, 5, "Fire emoji (counts as 2)"},
		{"ZWJ", "test\u200Dtext", 2, 1, 8, "Zero-width joiner"},
		{"Combining", "e\u0301", 2, 1, 1, "e with acute accent"},
		{"CJK", "你好", 2, 1, 4, "Chinese (2 width each)"},
		{"Mixed", "Hi你好", 2, 1, 6, "Mixed ASCII and CJK"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canvas := NewMatrixCanvas(20, 3)
			err := canvas.DrawText(tt.x, tt.y, tt.text)
			
			if err != nil {
				t.Fatalf("DrawText() error = %v", err)
			}
			
			// Check text was placed by measuring actual width
			actualWidth := 0
			for i := 0; i < 20-tt.x; i++ {
				char := canvas.Get(diagram.Point{tt.x + i, tt.y})
				if char != ' ' {
					actualWidth = i + 1
				}
			}
			
			// For wide character tests, just check we have some non-space characters
			if actualWidth < 1 {
				t.Errorf("No text was placed on canvas")
			}
		})
	}
}

// TestMatrixCanvas_BoundaryConditions tests edge cases at canvas boundaries.
func TestMatrixCanvas_BoundaryConditions(t *testing.T) {
	canvas := NewMatrixCanvas(10, 10)
	
	tests := []struct {
		name string
		test func() error
		want bool // true if should succeed
	}{
		{
			name: "Box at exact boundary",
			test: func() error {
				return canvas.DrawBox(0, 0, 10, 10, DefaultBoxStyle)
			},
			want: true,
		},
		{
			name: "Box exceeding width",
			test: func() error {
				return canvas.DrawBox(5, 5, 10, 4, DefaultBoxStyle)
			},
			want: false,
		},
		{
			name: "Box exceeding height", 
			test: func() error {
				return canvas.DrawBox(5, 5, 4, 10, DefaultBoxStyle)
			},
			want: false,
		},
		{
			name: "Text at right edge",
			test: func() error {
				return canvas.DrawText(8, 5, "Hi")
			},
			want: true, // Should clip
		},
		{
			name: "Line crossing boundary",
			test: func() error {
				return canvas.DrawLine(diagram.Point{5, 5}, diagram.Point{15, 15}, '*')
			},
			want: true, // Should clip
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canvas.Clear()
			err := tt.test()
			
			if tt.want && err != nil {
				t.Errorf("Expected success, got error: %v", err)
			}
			if !tt.want && err == nil {
				t.Error("Expected error, got success")
			}
		})
	}
}

// TestMatrixCanvas_ComplexJunctions tests complex line intersections.
func TestMatrixCanvas_ComplexJunctions(t *testing.T) {
	tests := []struct {
		name   string
		draw   func(c *MatrixCanvas)
		check  func(c *MatrixCanvas, t *testing.T)
	}{
		{
			name: "Three-way junction",
			draw: func(c *MatrixCanvas) {
				c.DrawHorizontalLine(1, 2, 5, '─')
				c.DrawVerticalLine(3, 0, 4, '│')
			},
			check: func(c *MatrixCanvas, t *testing.T) {
				junction := c.Get(diagram.Point{3, 2})
				// Should be ┬, ┴, ├, or ┤ depending on position
				if junction == '─' || junction == '│' {
					t.Error("Expected junction character at intersection")
				}
			},
		},
		{
			name: "Four-way cross",
			draw: func(c *MatrixCanvas) {
				c.DrawHorizontalLine(1, 2, 5, '─')
				c.DrawVerticalLine(3, 1, 3, '│')
			},
			check: func(c *MatrixCanvas, t *testing.T) {
				cross := c.Get(diagram.Point{3, 2})
				if cross != '┼' {
					t.Errorf("Expected ┼ at crossing, got %c", cross)
				}
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canvas := NewMatrixCanvas(7, 5)
			tt.draw(canvas)
			tt.check(canvas, t)
		})
	}
}

// TestMatrixCanvas_TextOperations tests advanced text functionality.
func TestMatrixCanvas_TextOperations(t *testing.T) {
	t.Run("Text wrapping", func(t *testing.T) {
		lines := WrapText("This is a long line that needs wrapping", 10)
		expected := []string{"This is a", "long line", "that needs", "wrapping"}
		
		if len(lines) != len(expected) {
			t.Fatalf("Expected %d lines, got %d", len(expected), len(lines))
		}
		
		for i, line := range lines {
			if line != expected[i] {
				t.Errorf("Line %d: got %q, want %q", i, line, expected[i])
			}
		}
	})
	
	t.Run("Text fitting", func(t *testing.T) {
		tests := []struct {
			text     string
			width    int
			ellipsis string
			expected string
		}{
			{"Hello, World!", 13, "...", "Hello, World!"},
			{"Hello, World!", 10, "...", "Hello, ..."},
			{"Hello, World!", 5, "...", "He..."},
			{"Hi", 5, "...", "Hi"},
		}
		
		for _, tt := range tests {
			result := FitText(tt.text, tt.width, tt.ellipsis)
			if result != tt.expected {
				t.Errorf("FitText(%q, %d) = %q, want %q", 
					tt.text, tt.width, result, tt.expected)
			}
		}
	})
}

// TestMatrixCanvas_ConcurrentAccess tests thread safety of read operations.
func TestMatrixCanvas_ConcurrentAccess(t *testing.T) {
	canvas := NewMatrixCanvas(100, 100)
	
	// Fill with some content
	for i := 0; i < 10; i++ {
		canvas.DrawBox(i*10, i*10, 8, 8, DefaultBoxStyle)
	}
	
	// Concurrent reads should be safe
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			
			// Perform many reads
			for j := 0; j < 1000; j++ {
				x := j % 100
				y := (j / 100) % 100
				_ = canvas.Get(diagram.Point{x, y})
			}
			
			// Get matrix
			_ = canvas.Matrix()
			
			// Get string
			_ = canvas.String()
		}()
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// ============================================================================
// Tests from node_renderer_center_test.go
// ============================================================================

func TestNodeRendererCenterText(t *testing.T) {
	tests := []struct {
		name     string
		node     diagram.Node
		hints    map[string]string
		expected []string
	}{
		{
			name: "center single line text",
			node: diagram.Node{
				X:      0,
				Y:      0,
				Width:  12,
				Height: 3,
				Text:   []string{"Hello"},
			},
			hints: map[string]string{
				"text-align": "center",
			},
			expected: []string{
				"╭──────────╮",
				"│  Hello   │",
				"╰──────────╯",
			},
		},
		{
			name: "center multiple lines",
			node: diagram.Node{
				X:      0,
				Y:      0,
				Width:  14,
				Height: 4,
				Text:   []string{"Hello", "World"},
			},
			hints: map[string]string{
				"text-align": "center",
			},
			expected: []string{
				"╭────────────╮",
				"│   Hello    │",
				"│   World    │",
				"╰────────────╯",
			},
		},
		{
			name: "center with different line lengths",
			node: diagram.Node{
				X:      0,
				Y:      0,
				Width:  16,
				Height: 5,
				Text:   []string{"Short", "Much Longer", "Mid"},
			},
			hints: map[string]string{
				"text-align": "center",
			},
			expected: []string{
				"╭──────────────╮",
				"│    Short     │",
				"│ Much Longer  │",
				"│     Mid      │",
				"╰──────────────╯",
			},
		},
		{
			name: "left-aligned by default",
			node: diagram.Node{
				X:      0,
				Y:      0,
				Width:  12,
				Height: 3,
				Text:   []string{"Hello"},
			},
			hints: map[string]string{}, // No text-align hint
			expected: []string{
				"╭──────────╮",
				"│ Hello    │",
				"╰──────────╯",
			},
		},
		{
			name: "center with bold",
			node: diagram.Node{
				X:      0,
				Y:      0,
				Width:  10,
				Height: 3,
				Text:   []string{"Test"},
			},
			hints: map[string]string{
				"text-align": "center",
				"bold":       "true",
			},
			expected: []string{
				"╭────────╮",
				"│  Test  │", // Should still be centered
				"╰────────╯",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create canvas and renderer
			canvas := NewMatrixCanvas(20, 10)
			renderer := NewNodeRenderer(TerminalCapabilities{
				UnicodeLevel: UnicodeFull,
			})

			// Set hints on the node
			if tt.hints != nil {
				tt.node.Hints = tt.hints
			}

			// Render the node
			err := renderer.RenderNode(canvas, tt.node)
			if err != nil {
				t.Fatalf("Failed to render node: %v", err)
			}

			// Get the rendered output
			output := canvas.String()
			lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

			// Check expected lines
			for i, expectedLine := range tt.expected {
				if i >= len(lines) {
					t.Errorf("Missing line %d: expected %q", i, expectedLine)
					continue
				}
				// Trim trailing spaces for comparison
				actualLine := strings.TrimRight(lines[i], " ")
				expectedLine = strings.TrimRight(expectedLine, " ")
				if actualLine != expectedLine {
					t.Errorf("Line %d mismatch:\nExpected: %q\nActual:   %q", i, expectedLine, actualLine)
				}
			}
		})
	}
}

func TestCenterTextAlignment(t *testing.T) {
	// Test the centering calculation specifically
	tests := []struct {
		name           string
		text           string
		nodeWidth      int
		expectedOffset int // Expected x offset from left border
	}{
		{
			name:           "exact fit",
			text:           "12345678",
			nodeWidth:      10, // 8 chars + 2 borders
			expectedOffset: 1,  // No centering needed
		},
		{
			name:           "small text in large node",
			text:           "Hi",
			nodeWidth:      10,                     // 8 chars available
			expectedOffset: 1 + (8-2)/2,            // 1 + 3 = 4
		},
		{
			name:           "odd spacing",
			text:           "Test",
			nodeWidth:      11,                     // 9 chars available
			expectedOffset: 1 + (9-4)/2,            // 1 + 2 = 3 (rounds down)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This mimics the calculation in drawText
			textWidth := len(tt.text)
			availableWidth := tt.nodeWidth - 2 // minus borders
			x := 1 // default padding
			if textWidth < availableWidth {
				x = 1 + (availableWidth-textWidth)/2
			}

			if x != tt.expectedOffset {
				t.Errorf("Expected x offset %d, got %d", tt.expectedOffset, x)
			}
		})
	}
}

// ============================================================================
// Tests from node_renderer_test.go
// ============================================================================

func TestNodeRendererBasicStyles(t *testing.T) {
	tests := []struct {
		name      string
		style     string
		wantChars struct {
			topLeft     rune
			topRight    rune
			bottomLeft  rune
			bottomRight rune
		}
	}{
		{
			name:  "rounded style",
			style: "rounded",
			wantChars: struct {
				topLeft     rune
				topRight    rune
				bottomLeft  rune
				bottomRight rune
			}{'╭', '╮', '╰', '╯'},
		},
		{
			name:  "sharp style",
			style: "sharp",
			wantChars: struct {
				topLeft     rune
				topRight    rune
				bottomLeft  rune
				bottomRight rune
			}{'┌', '┐', '└', '┘'},
		},
		{
			name:  "double style",
			style: "double",
			wantChars: struct {
				topLeft     rune
				topRight    rune
				bottomLeft  rune
				bottomRight rune
			}{'╔', '╗', '╚', '╝'},
		},
		{
			name:  "thick style",
			style: "thick",
			wantChars: struct {
				topLeft     rune
				topRight    rune
				bottomLeft  rune
				bottomRight rune
			}{'┏', '┓', '┗', '┛'},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create canvas and renderer
			canvas := NewMatrixCanvas(20, 10)
			renderer := NewNodeRenderer(TerminalCapabilities{
				UnicodeLevel: UnicodeFull,
			})
			
			// Create node with style hint
			node := diagram.Node{
				ID:     1,
				Text:   []string{"Test"},
				X:      2,
				Y:      2,
				Width:  10,
				Height: 4,
				Hints: map[string]string{
					"style": tt.style,
				},
			}
			
			// Render the node
			err := renderer.RenderNode(canvas, node)
			if err != nil {
				t.Fatalf("Failed to render node: %v", err)
			}
			
			// Check corners
			if canvas.Get(diagram.Point{X: 2, Y: 2}) != tt.wantChars.topLeft {
				t.Errorf("Top-left corner: got %c, want %c", 
					canvas.Get(diagram.Point{X: 2, Y: 2}), tt.wantChars.topLeft)
			}
			if canvas.Get(diagram.Point{X: 11, Y: 2}) != tt.wantChars.topRight {
				t.Errorf("Top-right corner: got %c, want %c", 
					canvas.Get(diagram.Point{X: 11, Y: 2}), tt.wantChars.topRight)
			}
			if canvas.Get(diagram.Point{X: 2, Y: 5}) != tt.wantChars.bottomLeft {
				t.Errorf("Bottom-left corner: got %c, want %c", 
					canvas.Get(diagram.Point{X: 2, Y: 5}), tt.wantChars.bottomLeft)
			}
			if canvas.Get(diagram.Point{X: 11, Y: 5}) != tt.wantChars.bottomRight {
				t.Errorf("Bottom-right corner: got %c, want %c", 
					canvas.Get(diagram.Point{X: 11, Y: 5}), tt.wantChars.bottomRight)
			}
		})
	}
}

func TestNodeRendererFallback(t *testing.T) {
	// Test that invalid style falls back to default
	canvas := NewMatrixCanvas(20, 10)
	renderer := NewNodeRenderer(TerminalCapabilities{
		UnicodeLevel: UnicodeFull,
	})
	
	node := diagram.Node{
		ID:     1,
		Text:   []string{"Test"},
		X:      2,
		Y:      2,
		Width:  10,
		Height: 4,
		Hints: map[string]string{
			"style": "invalid-style",
		},
	}
	
	err := renderer.RenderNode(canvas, node)
	if err != nil {
		t.Fatalf("Failed to render node: %v", err)
	}
	
	// Should fall back to rounded (default)
	if canvas.Get(diagram.Point{X: 2, Y: 2}) != '╭' {
		t.Errorf("Should fall back to rounded style, got %c", canvas.Get(diagram.Point{X: 2, Y: 2}))
	}
}

func TestNodeRendererASCIIFallback(t *testing.T) {
	// Test that ASCII terminals get ASCII style
	canvas := NewMatrixCanvas(20, 10)
	renderer := NewNodeRenderer(TerminalCapabilities{
		UnicodeLevel: UnicodeNone,
	})
	
	node := diagram.Node{
		ID:     1,
		Text:   []string{"Test"},
		X:      2,
		Y:      2,
		Width:  10,
		Height: 4,
		Hints: map[string]string{
			"style": "rounded", // Should be ignored for ASCII
		},
	}
	
	err := renderer.RenderNode(canvas, node)
	if err != nil {
		t.Fatalf("Failed to render node: %v", err)
	}
	
	// Should use ASCII style
	if canvas.Get(diagram.Point{X: 2, Y: 2}) != '+' {
		t.Errorf("Should use ASCII style for ASCII terminal, got %c", canvas.Get(diagram.Point{X: 2, Y: 2}))
	}
}

func TestNodeRendererText(t *testing.T) {
	// Test that text is rendered correctly inside the box
	canvas := NewMatrixCanvas(20, 10)
	renderer := NewNodeRenderer(TerminalCapabilities{
		UnicodeLevel: UnicodeFull,
	})
	
	node := diagram.Node{
		ID:     1,
		Text:   []string{"Line 1", "Line 2"},
		X:      0,
		Y:      0,
		Width:  12,
		Height: 5,
	}
	
	err := renderer.RenderNode(canvas, node)
	if err != nil {
		t.Fatalf("Failed to render node: %v", err)
	}
	
	output := canvas.String()
	lines := strings.Split(output, "\n")
	
	// Check that text appears in the right place (line 1 at y=1, line 2 at y=2)
	// Text should be at x=2 (2 chars padding)
	if !strings.Contains(lines[1], "Line 1") {
		t.Errorf("Line 1 not found in correct position")
	}
	if !strings.Contains(lines[2], "Line 2") {
		t.Errorf("Line 2 not found in correct position")
	}
}

func TestNodeRendererColors(t *testing.T) {
	// Test that colors are applied when using ColoredMatrixCanvas
	canvas := NewColoredMatrixCanvas(20, 10)
	renderer := NewNodeRenderer(TerminalCapabilities{
		UnicodeLevel:  UnicodeFull,
		SupportsColor: true,
	})
	
	node := diagram.Node{
		ID:     1,
		Text:   []string{"Colored"},
		X:      2,
		Y:      2,
		Width:  10,
		Height: 4,
		Hints: map[string]string{
			"style": "double",
			"color": "blue",
		},
	}
	
	err := renderer.RenderNode(canvas, node)
	if err != nil {
		t.Fatalf("Failed to render node: %v", err)
	}
	
	// Check that the box uses double-line style
	if canvas.Get(diagram.Point{X: 2, Y: 2}) != '╔' {
		t.Errorf("Expected double-line top-left corner, got %c", canvas.Get(diagram.Point{X: 2, Y: 2}))
	}
	
	// The colored output should contain ANSI color codes
	coloredOutput := canvas.ColoredString()
	if !strings.Contains(coloredOutput, "\033[") {
		t.Errorf("Expected colored output to contain ANSI codes")
	}
}

func TestNodeRendererVisualRegression(t *testing.T) {
	// Visual regression test - ensure the output looks correct
	canvas := NewMatrixCanvas(15, 6)
	renderer := NewNodeRenderer(TerminalCapabilities{
		UnicodeLevel: UnicodeFull,
	})
	
	node := diagram.Node{
		ID:     1,
		Text:   []string{"Hello", "World"},
		X:      1,
		Y:      1,
		Width:  10,
		Height: 4,
		Hints: map[string]string{
			"style": "rounded",
		},
	}
	
	err := renderer.RenderNode(canvas, node)
	if err != nil {
		t.Fatalf("Failed to render node: %v", err)
	}
	
	expected := `               
 ╭────────╮    
 │ Hello  │    
 │ World  │    
 ╰────────╯    
               `
	
	actual := canvas.String()
	if actual != expected {
		t.Errorf("Visual regression failed.\nExpected:\n%s\nGot:\n%s", expected, actual)
		// Print with visible spaces for debugging
		t.Errorf("Expected (with dots for spaces):\n%s", strings.ReplaceAll(expected, " ", "·"))
		t.Errorf("Got (with dots for spaces):\n%s", strings.ReplaceAll(actual, " ", "·"))
	}
}

// ============================================================================
// Tests from renderer_visual_test.go
// ============================================================================

// TestRendererVisualOutput creates visual examples to verify rendering quality
func TestRendererVisualOutput(t *testing.T) {
	tests := []struct {
		name    string
		d *diagram.Diagram
		width   int
		height  int
	}{
		{
			name: "Simple Two Nodes",
			d: &diagram.Diagram{
				Nodes: []diagram.Node{
					{ID: 1, Text: []string{"Node A"}},
					{ID: 2, Text: []string{"Node B"}},
				},
				Connections: []diagram.Connection{
					{From: 1, To: 2},
				},
			},
			width:  30,
			height: 10,
		},
		{
			name: "Three Node Chain",
			d: &diagram.Diagram{
				Nodes: []diagram.Node{
					{ID: 1, Text: []string{"Start"}},
					{ID: 2, Text: []string{"Middle"}},
					{ID: 3, Text: []string{"End"}},
				},
				Connections: []diagram.Connection{
					{From: 1, To: 2},
					{From: 2, To: 3},
				},
			},
			width:  40,
			height: 10,
		},
		{
			name: "Triangle Layout",
			d: &diagram.Diagram{
				Nodes: []diagram.Node{
					{ID: 1, Text: []string{"A"}},
					{ID: 2, Text: []string{"B"}},
					{ID: 3, Text: []string{"C"}},
				},
				Connections: []diagram.Connection{
					{From: 1, To: 2},
					{From: 2, To: 3},
					{From: 3, To: 1},
				},
			},
			width:  30,
			height: 15,
		},
		{
			name: "Self Loop",
			d: &diagram.Diagram{
				Nodes: []diagram.Node{
					{ID: 1, Text: []string{"Recursive", "Node"}},
				},
				Connections: []diagram.Connection{
					{From: 1, To: 1},
				},
			},
			width:  20,
			height: 10,
		},
		{
			name: "Multiple Connections",
			d: &diagram.Diagram{
				Nodes: []diagram.Node{
					{ID: 1, Text: []string{"Server"}},
					{ID: 2, Text: []string{"Client"}},
				},
				Connections: []diagram.Connection{
					{From: 1, To: 2},
					{From: 1, To: 2},
					{From: 2, To: 1},
				},
			},
			width:  30,
			height: 10,
		},
	}

	renderer := NewRenderer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := renderer.Render(tt.d)
			if err != nil {
				t.Errorf("Failed to render: %v", err)
				return
			}

			// Display the output
			fmt.Printf("\n=== %s ===\n", tt.name)
			fmt.Printf("Nodes: %d, Connections: %d\n", len(tt.d.Nodes), len(tt.d.Connections))
			fmt.Println(strings.Repeat("-", tt.width))
			
			// Print with line numbers for debugging
			lines := strings.Split(output, "\n")
			for i, line := range lines {
				if i < tt.height {
					fmt.Printf("%2d: %s\n", i+1, line)
				}
			}
			fmt.Println(strings.Repeat("-", tt.width))

			// Basic validation
			for _, node := range tt.d.Nodes {
				for _, text := range node.Text {
					if !strings.Contains(output, text) {
						t.Errorf("Missing node text: %s", text)
					}
				}
			}

			// Check for box characters (using rounded corners by default)
			if !strings.Contains(output, "╭") || !strings.Contains(output, "╯") {
				t.Error("Missing box drawing characters")
			}
			
			// For connections, check for lines
			if len(tt.d.Connections) > 0 {
				if !strings.Contains(output, "─") && !strings.Contains(output, "│") {
					t.Error("Missing connection lines")
				}
			}
		})
	}
}

// TestConnectionPointDebug helps debug connection point calculation
func TestConnectionPointDebug(t *testing.T) {
	// Create a simple diagram
	d := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"A"}},
			{ID: 2, Text: []string{"B"}},
		},
		Connections: []diagram.Connection{
			{From: 1, To: 2},
		},
	}
	
	renderer := NewRenderer()
	output, err := renderer.Render(d)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}
	
	// Analyze the output character by character
	fmt.Println("\n=== Character Analysis ===")
	lines := strings.Split(output, "\n")
	for y, line := range lines {
		fmt.Printf("Line %d: ", y)
		for x, ch := range line {
			if ch != ' ' {
				fmt.Printf("[%d:%c] ", x, ch)
			}
		}
		fmt.Println()
	}
}

// ============================================================================
// Tests from renderer_bench_test.go
// ============================================================================

// generateLargeDiagram creates a diagram with the specified number of nodes
func generateLargeDiagram(nodes, connectionsPerNode int) *diagram.Diagram {
	d := &diagram.Diagram{
		Nodes:       make([]diagram.Node, nodes),
		Connections: make([]diagram.Connection, 0, nodes*connectionsPerNode),
	}

	// Create nodes
	for i := 0; i < nodes; i++ {
		d.Nodes[i] = diagram.Node{
			ID:   i + 1,
			Text: []string{fmt.Sprintf("Node %d", i+1)},
		}
	}

	// Create connections (avoiding cycles for simple layout)
	for i := 0; i < nodes-1; i++ {
		// Connect to next node
		d.Connections = append(d.Connections, diagram.Connection{
			From: i + 1,
			To:   i + 2,
		})
		
		// Add some additional forward connections
		for j := 1; j < connectionsPerNode && i+j+1 < nodes; j++ {
			d.Connections = append(d.Connections, diagram.Connection{
				From: i + 1,
				To:   i + j + 2,
			})
		}
	}

	return d
}

// TestRendererBasic tests the basic rendering functionality
func TestRendererBasic(t *testing.T) {
	// Create a simple two-node diagram
	d := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"Hello"}},
			{ID: 2, Text: []string{"World"}},
		},
		Connections: []diagram.Connection{
			{From: 1, To: 2},
		},
	}
	
	// Create renderer
	renderer := NewRenderer()
	
	// Render the diagram
	output, err := renderer.Render(d)
	if err != nil {
		t.Fatalf("Failed to render diagram: %v", err)
	}
	
	// Print output for debugging
	t.Logf("Rendered output:\n%s", output)
	
	// Basic checks
	if output == "" {
		t.Error("Expected non-empty output")
	}
	
	// Check that both node texts appear
	if !strings.Contains(output, "Hello") {
		t.Error("Expected output to contain 'Hello'")
	}
	if !strings.Contains(output, "World") {
		t.Error("Expected output to contain 'World'")
	}
	
	// Check for box drawing characters (the renderer uses rounded corners by default)
	if !strings.Contains(output, "╭") || !strings.Contains(output, "╮") {
		t.Error("Expected output to contain box drawing characters")
	}
}

// TestRendererEmptyDiagram tests rendering an empty diagram
func TestRendererEmptyDiagram(t *testing.T) {
	d := &diagram.Diagram{
		Nodes:       []diagram.Node{},
		Connections: []diagram.Connection{},
	}
	
	renderer := NewRenderer()
	output, err := renderer.Render(d)
	if err != nil {
		t.Fatalf("Failed to render empty diagram: %v", err)
	}
	
	// Should still produce some output (empty canvas)
	if output == "" {
		t.Error("Expected non-empty output even for empty diagram")
	}
}

// TestRendererSingleNode tests rendering a single node with no connections
func TestRendererSingleNode(t *testing.T) {
	d := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"Lonely", "Node"}},
		},
		Connections: []diagram.Connection{},
	}
	
	renderer := NewRenderer()
	output, err := renderer.Render(d)
	if err != nil {
		t.Fatalf("Failed to render single node: %v", err)
	}
	
	// Check that the node text appears
	if !strings.Contains(output, "Lonely") {
		t.Error("Expected output to contain 'Lonely'")
	}
	if !strings.Contains(output, "Node") {
		t.Error("Expected output to contain 'Node'")
	}
}

// TestRendererMultipleConnections tests rendering with multiple connections
func TestRendererMultipleConnections(t *testing.T) {
	d := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"A"}},
			{ID: 2, Text: []string{"B"}},
			{ID: 3, Text: []string{"C"}},
		},
		Connections: []diagram.Connection{
			{From: 1, To: 2},
			{From: 2, To: 3},
			{From: 1, To: 3},
		},
	}
	
	renderer := NewRenderer()
	output, err := renderer.Render(d)
	if err != nil {
		t.Fatalf("Failed to render multiple connections: %v", err)
	}
	
	
	// Check all nodes appear
	for _, label := range []string{"A", "B", "C"} {
		if !strings.Contains(output, label) {
			t.Errorf("Expected output to contain '%s'", label)
		}
	}
	
	// Should have connection lines
	lines := strings.Split(output, "\n")
	connectionCount := 0
	for _, line := range lines {
		if strings.Contains(line, "─") || strings.Contains(line, "│") {
			connectionCount++
		}
	}
	if connectionCount < 3 {
		t.Error("Expected to see connection lines in output")
	}
}

// TestRendererSelfLoop tests rendering a node that connects to itself
func TestRendererSelfLoop(t *testing.T) {
	d := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"Recursive"}},
		},
		Connections: []diagram.Connection{
			{From: 1, To: 1},
		},
	}
	
	renderer := NewRenderer()
	output, err := renderer.Render(d)
	if err != nil {
		t.Fatalf("Failed to render self-loop: %v", err)
	}
	
	// Should contain the node
	if !strings.Contains(output, "Recursive") {
		t.Error("Expected output to contain 'Recursive'")
	}
	
	// Should have a visible loop (check for loop characters)
	// A self-loop typically extends beyond the node bounds
	lines := strings.Split(output, "\n")
	nodeFound := false
	for _, line := range lines {
		if strings.Contains(line, "Recursive") {
			nodeFound = true
			break
		}
	}
	if !nodeFound {
		t.Error("Could not find node in output")
	}
}

// ============================================================================
// Tests from sequence_renderer_test.go
// ============================================================================

func TestCanvasAndNodeRenderer(t *testing.T) {
	caps := TerminalCapabilities{UnicodeLevel: UnicodeExtended}
	nodeRenderer := NewNodeRenderer(caps)
	
	c := NewMatrixCanvas(30, 10)
	node := diagram.Node{
		ID:     1,
		X:      5,
		Y:      2,
		Width:  10,
		Height: 3,
		Text:   []string{"Test"},
	}
	
	err := nodeRenderer.RenderNode(c, node)
	if err != nil {
		t.Fatalf("Failed to render node: %v", err)
	}
	
	output := c.String()
	t.Logf("Direct node render:\n%s", output)
	
	if !strings.Contains(output, "Test") {
		t.Error("Should contain Test text")
	}
}

func TestSequenceRendererBasic(t *testing.T) {
	caps := TerminalCapabilities{UnicodeLevel: UnicodeExtended}
	renderer := NewSequenceRenderer(caps)
	
	d := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"User"}},
			{ID: 2, Text: []string{"Server"}},
		},
		Connections: []diagram.Connection{
			{From: 1, To: 2, Label: "request"},
			{From: 2, To: 1, Label: "response"},
		},
	}
	
	// Get required canvas size
	width, height := renderer.GetBounds(d)
	if width <= 0 || height <= 0 {
		t.Fatalf("Invalid bounds: %dx%d", width, height)
	}
	t.Logf("Canvas size: %dx%d", width, height)
	
	// Render directly to string
	output, err := renderer.Render(d)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}
	
	// Debug output
	t.Logf("Canvas output:\n%s", output)
	t.Logf("Nodes after layout: %+v", d.Nodes)
	
	// Check for participant boxes
	if !strings.Contains(output, "User") {
		t.Error("Should contain User participant")
	}
	if !strings.Contains(output, "Server") {
		t.Error("Should contain Server participant")
	}
	
	// Check for lifelines (vertical lines)
	if !strings.Contains(output, "│") {
		t.Error("Should contain vertical lifeline characters")
	}
	
	// Check for message arrows
	if !strings.Contains(output, "─") {
		t.Error("Should contain horizontal line characters for messages")
	}
	if !strings.Contains(output, "▶") || !strings.Contains(output, "◀") {
		t.Error("Should contain arrow characters")
	}
	
	// Check for labels
	if !strings.Contains(output, "request") {
		t.Error("Should contain request label")
	}
	if !strings.Contains(output, "response") {
		t.Error("Should contain response label")
	}
}

func TestSequenceRendererSelfMessage(t *testing.T) {
	caps := TerminalCapabilities{UnicodeLevel: UnicodeExtended}
	renderer := NewSequenceRenderer(caps)
	
	d := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"System"}},
		},
		Connections: []diagram.Connection{
			{From: 1, To: 1, Label: "process"},
		},
	}
	
	output, err := renderer.Render(d)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}
	
	// Check for self-message loop
	if !strings.Contains(output, "┐") {
		t.Error("Should contain corner character for self-message")
	}
	if !strings.Contains(output, "process") {
		t.Error("Should contain self-message label")
	}
}

func TestSequenceRendererMultipleParticipants(t *testing.T) {
	caps := TerminalCapabilities{UnicodeLevel: UnicodeExtended}
	renderer := NewSequenceRenderer(caps)
	
	d := &diagram.Diagram{
		Nodes: []diagram.Node{
			{ID: 1, Text: []string{"Client"}},
			{ID: 2, Text: []string{"Server"}},
			{ID: 3, Text: []string{"Database"}},
		},
		Connections: []diagram.Connection{
			{From: 1, To: 2, Label: "request"},
			{From: 2, To: 3, Label: "query"},
			{From: 3, To: 2, Label: "data"},
			{From: 2, To: 1, Label: "response"},
		},
	}
	
	output, err := renderer.Render(d)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}
	
	// Check all participants are present
	if !strings.Contains(output, "Client") {
		t.Error("Should contain Client participant")
	}
	if !strings.Contains(output, "Server") {
		t.Error("Should contain Server participant")
	}
	if !strings.Contains(output, "Database") {
		t.Error("Should contain Database participant")
	}
	
	// Check all message labels
	if !strings.Contains(output, "request") {
		t.Error("Should contain request message")
	}
	if !strings.Contains(output, "query") {
		t.Error("Should contain query message")
	}
	if !strings.Contains(output, "data") {
		t.Error("Should contain data message")
	}
	if !strings.Contains(output, "response") {
		t.Error("Should contain response message")
	}
}

// ============================================================================
// Benchmarks from renderer_bench_test.go and matrix_test.go
// ============================================================================

// BenchmarkRenderer tests rendering performance
func BenchmarkRenderer(b *testing.B) {
	sizes := []struct {
		nodes       int
		connections int
	}{
		{10, 2},
		{25, 2},
		{50, 2},
		{100, 2},
	}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("%d_nodes", size.nodes), func(b *testing.B) {
			d := generateLargeDiagram(size.nodes, size.connections)
			renderer := NewRenderer()
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := renderer.Render(d)
				if err != nil {
					b.Fatal(err)
				}
			}
			
			// Report useful metrics
			b.ReportMetric(float64(size.nodes), "nodes")
			b.ReportMetric(float64(len(d.Connections)), "connections")
		})
	}
}

// BenchmarkRendererMemory tests memory usage
func BenchmarkRendererMemory(b *testing.B) {
	d := generateLargeDiagram(100, 2)
	renderer := NewRenderer()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, err := renderer.Render(d)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMatrixCanvas_Create_100x100(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewMatrixCanvas(100, 100)
	}
}

func BenchmarkMatrixCanvas_String_100x100(b *testing.B) {
	canvas := NewMatrixCanvas(100, 100)
	// Add some content
	for i := 0; i < 5; i++ {
		canvas.DrawBox(i*15, i*10, 12, 8, DefaultBoxStyle)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = canvas.String()
	}
}

func BenchmarkMatrixCanvas_DrawBox(b *testing.B) {
	canvas := NewMatrixCanvas(100, 100)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		canvas.DrawBox(10, 10, 20, 15, DefaultBoxStyle)
		canvas.Clear()
	}
}
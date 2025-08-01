package canvas

import (
	"testing"
)

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
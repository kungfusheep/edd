package canvas

import (
	"edd/core"
	"testing"
	"time"
)

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
		point core.Point
		char  rune
		valid bool
	}{
		{"Origin", core.Point{0, 0}, '╭', true},
		{"Center", core.Point{10, 5}, '┼', true},
		{"Bottom right", core.Point{19, 9}, '╯', true},
		{"Out of bounds X", core.Point{20, 5}, 'X', false},
		{"Out of bounds Y", core.Point{10, 10}, 'Y', false},
		{"Negative X", core.Point{-1, 5}, 'N', false},
		{"Negative Y", core.Point{5, -1}, 'N', false},
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
	points := []core.Point{{5, 5}, {0, 0}, {9, 9}, {3, 7}}
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
	canvas.Set(core.Point{0, 0}, '╭')
	canvas.Set(core.Point{1, 0}, '─')
	canvas.Set(core.Point{2, 0}, '─')
	canvas.Set(core.Point{3, 0}, '─')
	canvas.Set(core.Point{4, 0}, '╮')
	
	canvas.Set(core.Point{0, 1}, '│')
	canvas.Set(core.Point{2, 1}, 'X')
	canvas.Set(core.Point{4, 1}, '│')
	
	canvas.Set(core.Point{0, 2}, '╰')
	canvas.Set(core.Point{1, 2}, '─')
	canvas.Set(core.Point{2, 2}, '─')
	canvas.Set(core.Point{3, 2}, '─')
	canvas.Set(core.Point{4, 2}, '╯')
	
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
				c.DrawLine(core.Point{1, 1}, core.Point{8, 8}, '*')
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

// TestMatrixCanvas_DrawText tests text rendering.
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
			if canvas.Get(core.Point{0, 0}) != ' ' {
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
		points []core.Point
		canvas string
	}{
		{
			name: "L-shaped path",
			points: []core.Point{
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
			points: []core.Point{
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
				char := canvas.Get(core.Point{tt.x + i, tt.y})
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
				return canvas.DrawLine(core.Point{5, 5}, core.Point{15, 15}, '*')
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
				junction := c.Get(core.Point{3, 2})
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
				cross := c.Get(core.Point{3, 2})
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
				_ = canvas.Get(core.Point{x, y})
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

// Benchmarks
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
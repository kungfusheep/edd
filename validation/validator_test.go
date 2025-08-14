package validation

import (
	"strings"
	"testing"
)

func TestLineValidator_BasicLines(t *testing.T) {
	tests := []struct {
		name    string
		diagram string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid horizontal line",
			diagram: `
─────
`,
			wantErr: false,
		},
		{
			name: "valid vertical line",
			diagram: `
│
│
│
`,
			wantErr: false,
		},
		{
			name: "broken horizontal line",
			diagram: `
──│──
`,
			wantErr: true,
			errMsg:  "Horizontal line cannot connect",
		},
		{
			name: "broken vertical line",
			diagram: `
│
─
│
`,
			wantErr: true,
			errMsg:  "Vertical line cannot connect",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewLineValidator()
			errors := v.Validate(strings.TrimSpace(tt.diagram))
			
			if tt.wantErr && len(errors) == 0 {
				t.Errorf("expected errors but got none")
			}
			if !tt.wantErr && len(errors) > 0 {
				t.Errorf("unexpected errors: %v", errors)
			}
			if tt.wantErr && tt.errMsg != "" {
				found := false
				for _, err := range errors {
					if strings.Contains(err.Message, tt.errMsg) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error containing %q, got %v", tt.errMsg, errors)
				}
			}
		})
	}
}

func TestLineValidator_Corners(t *testing.T) {
	tests := []struct {
		name    string
		diagram string
		wantErr bool
	}{
		{
			name: "valid box",
			diagram: `
┌───┐
│   │
└───┘
`,
			wantErr: false,
		},
		{
			name: "invalid top-left corner",
			diagram: `
┌
 │
`,
			wantErr: true,
		},
		{
			name: "invalid bottom-right corner",
			diagram: `
─┘
│ 
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewLineValidator()
			errors := v.Validate(strings.TrimSpace(tt.diagram))
			
			if tt.wantErr && len(errors) == 0 {
				t.Errorf("expected errors but got none")
			}
			if !tt.wantErr && len(errors) > 0 {
				t.Errorf("unexpected errors: %v", errors)
			}
		})
	}
}

func TestLineValidator_Junctions(t *testing.T) {
	tests := []struct {
		name    string
		diagram string
		wantErr bool
	}{
		{
			name: "valid cross junction",
			diagram: `
 │ 
─┼─
 │ 
`,
			wantErr: false,
		},
		{
			name: "valid tee-right",
			diagram: `
 │ 
 ├─
 │ 
`,
			wantErr: false,
		},
		{
			name: "invalid tee-right (missing connection)",
			diagram: `
 │ 
 ├ 
 │ 
`,
			wantErr: true,
		},
		{
			name: "valid tee-down",
			diagram: `
─┬─
 │ 
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewLineValidator()
			errors := v.Validate(strings.TrimSpace(tt.diagram))
			
			if tt.wantErr && len(errors) == 0 {
				t.Errorf("expected errors but got none")
			}
			if !tt.wantErr && len(errors) > 0 {
				t.Errorf("unexpected errors: %v", errors)
			}
		})
	}
}

func TestLineValidator_Arrows(t *testing.T) {
	tests := []struct {
		name    string
		diagram string
		wantErr bool
	}{
		{
			name: "valid right arrow",
			diagram: `
──▶
`,
			wantErr: false,
		},
		{
			name: "valid down arrow",
			diagram: `
│
▼
`,
			wantErr: false,
		},
		{
			name: "invalid right arrow (no line)",
			diagram: `
  ▶
`,
			wantErr: true,
		},
		{
			name: "invalid up arrow (wrong line)",
			diagram: `
▲
─
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewLineValidator()
			errors := v.Validate(strings.TrimSpace(tt.diagram))
			
			if tt.wantErr && len(errors) == 0 {
				t.Errorf("expected errors but got none")
			}
			if !tt.wantErr && len(errors) > 0 {
				t.Errorf("unexpected errors: %v", errors)
			}
		})
	}
}

func TestLineValidator_ComplexDiagram(t *testing.T) {
	// A more complex diagram with connections
	diagram := `
┌─────┐     ┌─────┐
│  A  │────▶│  B  │
└─────┘     └─────┘
   │           │
   │           │
   ▼           ▼
┌─────┐     ┌─────┐
│  C  │     │  D  │
└─────┘     └─────┘
`

	v := NewLineValidator()
	errors := v.Validate(strings.TrimSpace(diagram))
	
	if len(errors) > 0 {
		t.Errorf("unexpected errors in complex diagram: %v", errors)
		for _, err := range errors {
			t.Logf("  %s", err)
		}
	}
}

func TestLineValidator_StrictMode(t *testing.T) {
	// Diagram with perpendicular lines touching (not a junction)
	diagram := `
─────
  │  
  │  
`

	// Normal mode should pass
	v := NewLineValidator()
	errors := v.Validate(strings.TrimSpace(diagram))
	if len(errors) > 0 {
		t.Errorf("normal mode: unexpected errors: %v", errors)
	}

	// Strict mode should fail
	v.SetStrictMode(true)
	errors = v.Validate(strings.TrimSpace(diagram))
	if len(errors) == 0 {
		t.Errorf("strict mode: expected errors for perpendicular lines")
	}
}

func TestLineValidator_ASCIICompatibility(t *testing.T) {
	// Mixed ASCII and Unicode
	diagram := `
+---+
|   |
+---+
`

	v := NewLineValidator()
	errors := v.Validate(strings.TrimSpace(diagram))
	
	if len(errors) > 0 {
		t.Errorf("unexpected errors with ASCII characters: %v", errors)
	}
}
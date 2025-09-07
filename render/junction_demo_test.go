package render

import (
	"edd/render"
	"fmt"
	"testing"
)

func TestJunctionResolver_VisualDemo(t *testing.T) {
	jr := render.NewJunctionResolver()
	
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
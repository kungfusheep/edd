package main

import (
	"encoding/json"
	"edd/diagram"
	"edd/render"
	"edd/terminal"
	"edd/validation"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	// Define command line flags
	var (
		interactive   = flag.Bool("i", false, "Interactive TUI mode")
		edit          = flag.Bool("edit", false, "Edit diagram in TUI (same as -i)")
		validate      = flag.Bool("validate", false, "Run validation on the output")
		debug         = flag.Bool("debug", false, "Show debug visualization with obstacles and ports")
		showObstacles = flag.Bool("show-obstacles", false, "Show virtual obstacles as dots in standard rendering")
		help          = flag.Bool("help", false, "Show help")
		
		// Demo mode flags
		demo       = flag.Bool("demo", false, "Demo mode: replay stdin input with randomized timing")
		minDelay   = flag.Int("min-delay", 50, "Minimum delay between keystrokes in ms (demo mode)")
		maxDelay   = flag.Int("max-delay", 300, "Maximum delay between keystrokes in ms (demo mode)")
		lineDelay  = flag.Int("line-delay", 500, "Extra delay between lines in ms (demo mode)")
	)
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [diagram.json]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A modular diagram renderer that converts JSON diagrams to ASCII/Unicode art.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s                    # Start interactive TUI\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s diagram.json       # Render diagram to stdout\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -i diagram.json    # Edit diagram in TUI\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -debug diagram.json\n", os.Args[0])
	}
	
	flag.Parse()
	
	if *help {
		flag.Usage()
		os.Exit(0)
	}
	
	// Get filename if provided
	args := flag.Args()
	var filename string
	if len(args) > 0 {
		filename = args[0]
	}
	
	// Handle interactive mode (including demo mode)
	if *interactive || *edit || *demo || (len(args) == 0 && !*validate && !*debug && !*showObstacles) {
		// Launch TUI (with demo settings if applicable)
		var demoSettings *terminal.DemoSettings
		if *demo {
			demoSettings = &terminal.DemoSettings{
				MinDelay:  *minDelay,
				MaxDelay:  *maxDelay,
				LineDelay: *lineDelay,
			}
		}
		if err := terminal.RunInteractiveWithDemo(filename, demoSettings); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	
	// Non-interactive mode requires a file
	if filename == "" {
		fmt.Fprintf(os.Stderr, "Error: Please provide a diagram JSON file\n\n")
		flag.Usage()
		os.Exit(1)
	}
	
	// Read the diagram file
	diagram, err := loadDiagram(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading diagram: %v\n", err)
		os.Exit(1)
	}
	
	// Create renderer
	renderer := render.NewRenderer()
	
	// Enable validation if requested
	if *validate {
		renderer.EnableValidation()
	}
	
	// Enable debug mode if requested
	if *debug {
		renderer.EnableDebug()
	}
	
	// Enable obstacle visualization if requested
	if *showObstacles {
		renderer.EnableObstacleVisualization()
	}
	
	// Render the diagram
	output, err := renderer.Render(diagram)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering diagram: %v\n", err)
		os.Exit(1)
	}
	
	// Output the result
	fmt.Print(output)
	
	// Run standalone validation if requested
	if *validate {
		// The renderer already validated during render and printed warnings
		// Here we could do additional validation or exit with error code if needed
		validator := validation.NewLineValidator()
		errors := validator.Validate(output)
		if len(errors) > 0 {
			os.Exit(2) // Exit with error code to indicate validation issues
		}
	}
}

// loadDiagram loads a diagram from a JSON file
func loadDiagram(filename string) (*diagram.Diagram, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()
	
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	
	var d diagram.Diagram
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}
	
	// Basic validation
	if len(d.Nodes) == 0 {
		return nil, fmt.Errorf("diagram has no nodes")
	}
	
	// Ensure all connections have unique IDs
	diagram.EnsureUniqueConnectionIDs(&d)
	
	// Default arrows to true for all connections
	for i := range d.Connections {
		// Default arrows to true if not explicitly set to false
		if !d.Connections[i].Arrow {
			d.Connections[i].Arrow = true
		}
	}
	
	return &d, nil
}


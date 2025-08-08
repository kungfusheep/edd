package main

import (
	"encoding/json"
	"edd/core"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	// Define command line flags
	var (
		validate      = flag.Bool("validate", false, "Run validation on the output")
		debug         = flag.Bool("debug", false, "Show debug visualization with obstacles and ports")
		showObstacles = flag.Bool("show-obstacles", false, "Show virtual obstacles as dots in standard rendering")
		help          = flag.Bool("help", false, "Show help")
	)
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <diagram.json>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A modular diagram renderer that converts JSON diagrams to ASCII/Unicode art.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s diagram.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -validate diagram.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -debug diagram.json\n", os.Args[0])
	}
	
	flag.Parse()
	
	if *help {
		flag.Usage()
		os.Exit(0)
	}
	
	// Check for input file
	args := flag.Args()
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "Error: Please provide a diagram JSON file\n\n")
		flag.Usage()
		os.Exit(1)
	}
	
	// Read the diagram file
	filename := args[0]
	diagram, err := loadDiagram(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading diagram: %v\n", err)
		os.Exit(1)
	}
	
	// Create renderer
	renderer := NewRenderer()
	
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
		validator := NewLineValidator()
		errors := validator.Validate(output)
		if len(errors) > 0 {
			os.Exit(2) // Exit with error code to indicate validation issues
		}
	}
}

// loadDiagram loads a diagram from a JSON file
func loadDiagram(filename string) (*core.Diagram, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()
	
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	
	var diagram core.Diagram
	if err := json.Unmarshal(data, &diagram); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}
	
	// Basic validation
	if len(diagram.Nodes) == 0 {
		return nil, fmt.Errorf("diagram has no nodes")
	}
	
	// Assign connection IDs if not present and default arrows to true
	for i := range diagram.Connections {
		if diagram.Connections[i].ID == 0 {
			diagram.Connections[i].ID = i + 1
		}
		// Default arrows to true if not explicitly set to false
		if !diagram.Connections[i].Arrow {
			diagram.Connections[i].Arrow = true
		}
	}
	
	return &diagram, nil
}
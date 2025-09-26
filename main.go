package main

import (
	"edd/diagram"
	"edd/editor"
	"edd/export"
	"edd/importer"
	"edd/render"
	"edd/terminal"
	"edd/validation"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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

		// Diagram type flag
		diagramType = flag.String("type", "", "Initial diagram type: sequence or box (default: box)")

		// Export flags
		format     = flag.String("format", "ascii", "Export format: ascii, mermaid, plantuml")
		outputFile = flag.String("o", "", "Output file (default: stdout)")

		// Import flags
		importFormat = flag.String("import", "", "Import from format: mermaid, plantuml, graphviz, d2 (auto-detect if not specified)")

		// Demo mode flags
		demo      = flag.Bool("demo", false, "Demo mode: replay stdin input with randomized timing")
		minDelay  = flag.Int("min-delay", 50, "Minimum delay between keystrokes in ms (demo mode)")
		maxDelay  = flag.Int("max-delay", 300, "Maximum delay between keystrokes in ms (demo mode)")
		lineDelay = flag.Int("line-delay", 500, "Extra delay between lines in ms (demo mode)")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [diagram.json]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A modular diagram renderer that converts JSON diagrams to various formats.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s                    # Start interactive TUI\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s diagram.json       # Render diagram to stdout\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -i diagram.json    # Edit diagram in TUI\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -debug diagram.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -format mermaid diagram.json    # Export to Mermaid\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -format plantuml -o output.puml diagram.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -import mermaid diagram.mmd     # Import from Mermaid\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s diagram.mmd                     # Auto-detect format by extension\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nInteractive Mode Commands:\n")
		fmt.Fprintf(os.Stderr, "  :export mermaid [file]   # Export to Mermaid format\n")
		fmt.Fprintf(os.Stderr, "  :export plantuml [file]  # Export to PlantUML format\n")
		fmt.Fprintf(os.Stderr, "  :export ascii [file]     # Export to ASCII/Unicode art\n")
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

		err := runInteractiveMode(filename, *diagramType, demoSettings)
		if err != nil {
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
	diagram, err := loadDiagram(filename, *importFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading diagram: %v\n", err)
		os.Exit(1)
	}

	// Parse export format
	exportFormat, err := export.ParseFormat(*format)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Available formats: ascii, mermaid, plantuml\n")
		os.Exit(1)
	}

	// Create appropriate exporter
	exporter, err := export.NewExporter(exportFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating exporter: %v\n", err)
		os.Exit(1)
	}

	var output string

	// For ASCII format, use the renderer with debug/validation options
	if exportFormat == export.FormatASCII {
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
		output, err = renderer.Render(diagram)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error rendering diagram: %v\n", err)
			os.Exit(1)
		}
	} else {
		// For other formats, use the exporter
		output, err = exporter.Export(diagram)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting diagram: %v\n", err)
			os.Exit(1)
		}
	}

	// Output the result
	if *outputFile != "" {
		// Write to file
		err := ioutil.WriteFile(*outputFile, []byte(output), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Successfully exported to %s\n", *outputFile)
	} else {
		// Output to stdout
		fmt.Println(output)
	}

	// Run standalone validation if requested (only for ASCII format)
	if *validate && exportFormat == export.FormatASCII {
		// The renderer already validated during render and printed warnings
		// Here we could do additional validation or exit with error code if needed
		validator := validation.NewLineValidator()
		errors := validator.Validate(output)
		if len(errors) > 0 {
			os.Exit(2) // Exit with error code to indicate validation issues
		}
	}
}

// loadDiagram loads a diagram from a file, potentially importing from other formats
func loadDiagram(filename string, importFormat string) (*diagram.Diagram, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	// Check if we need to import from another format
	ext := strings.ToLower(filepath.Ext(filename))
	needImport := importFormat != "" || ext != ".json"

	// Known import extensions
	importExtensions := map[string]bool{
		".mmd":      true,
		".mermaid":  true,
		".puml":     true,
		".plantuml": true,
		".pu":       true,
		".dot":      true,
		".gv":       true,
		".d2":       true,
	}

	if needImport && importExtensions[ext] {
		// Import from another format
		registry := importer.NewImporterRegistry()

		var d *diagram.Diagram
		if importFormat != "" {
			// Use specified format
			d, err = registry.ImportWithFormat(string(data), importFormat)
		} else {
			// Auto-detect format
			d, err = registry.Import(string(data))
		}

		if err != nil {
			return nil, fmt.Errorf("importing diagram: %w", err)
		}

		// Ensure all connections have unique IDs
		diagram.EnsureUniqueConnectionIDs(d)

		// Default arrows to true for all connections
		for i := range d.Connections {
			if !d.Connections[i].Arrow {
				d.Connections[i].Arrow = true
			}
		}

		return d, nil
	}

	// Parse as JSON
	var d diagram.Diagram
	if err := json.Unmarshal(data, &d); err != nil {
		// If JSON parsing fails and it might be another format, try importing
		if importFormat != "" || importExtensions[ext] {
			registry := importer.NewImporterRegistry()

			var imported *diagram.Diagram
			if importFormat != "" {
				imported, err = registry.ImportWithFormat(string(data), importFormat)
			} else {
				imported, err = registry.Import(string(data))
			}

			if err != nil {
				return nil, fmt.Errorf("failed to parse as JSON and import failed: %w", err)
			}

			// Ensure all connections have unique IDs
			diagram.EnsureUniqueConnectionIDs(imported)

			// Default arrows to true for all connections
			for i := range imported.Connections {
				if !imported.Connections[i].Arrow {
					imported.Connections[i].Arrow = true
				}
			}

			return imported, nil
		}
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

// runInteractiveMode launches the TUI editor with optional demo mode
// This is the main entry point for interactive editing
func runInteractiveMode(filename string, diagramType string, demoSettings *terminal.DemoSettings) error {
	// Create the real renderer
	renderer := editor.NewRealRenderer()

	// Create TUI editor
	tui := editor.NewTUIEditor(renderer)

	// Load diagram if filename provided
	if filename != "" {
		// Use the main loadDiagram function which handles imports
		d, err := loadDiagram(filename, "")
		if err != nil {
			return fmt.Errorf("failed to load diagram: %w", err)
		}
		tui.SetDiagram(d)
	} else if diagramType != "" {
		// No file provided, but user specified a diagram type
		d := &diagram.Diagram{
			Type: diagramType,
			Nodes: []diagram.Node{},
			Connections: []diagram.Connection{},
		}
		// Validate the type
		if diagramType != "sequence" && diagramType != "box" {
			return fmt.Errorf("invalid diagram type: %s (must be 'sequence' or 'box')", diagramType)
		}
		tui.SetDiagram(d)
	}

	// Delegate to terminal package for the actual TUI loop
	return terminal.RunTUILoop(tui, filename, demoSettings)
}


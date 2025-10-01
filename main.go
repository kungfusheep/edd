package main

import (
	"edd/diagram"
	"edd/editor"
	"edd/export"
	"edd/importer"
	"edd/markdown"
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
		inputFormat = flag.String("input-format", "", "Input format: json, mermaid, plantuml, graphviz, d2 (auto-detect if not specified)")
		importFormat = flag.String("import", "", "[Deprecated: use -input-format] Import from format: mermaid, plantuml, graphviz, d2")

		// Markdown mode flags
		markdownMode = flag.Bool("markdown", false, "Edit diagram blocks within markdown files")
		blockIndex   = flag.Int("block", 0, "Which diagram block to edit (1-based index, 0 = show picker)")

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
		fmt.Fprintf(os.Stderr, "  %s -input-format mermaid diagram.mmd  # Import from Mermaid\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s diagram.mmd                        # Auto-detect format by extension\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -input-format mermaid <(cat file)  # Use process substitution\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -markdown README.md                 # Edit diagram block in markdown\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -markdown -block 2 README.md        # Edit 2nd diagram block\n", os.Args[0])
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

	// Handle markdown mode
	if *markdownMode && filename != "" {
		// Check if this is extraction mode (non-interactive)
		if *format != "ascii" {
			err := runMarkdownExtraction(filename, *blockIndex, *format, *outputFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		} else if *interactive || *edit || (len(args) > 0) {
			// Interactive editing mode
			err := runMarkdownMode(filename, *blockIndex)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}
		os.Exit(0)
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
	// Use inputFormat if specified, otherwise fall back to importFormat for backwards compatibility
	inFmt := *inputFormat
	if inFmt == "" {
		inFmt = *importFormat
	}
	diagram, err := loadDiagram(filename, inFmt)
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
func loadDiagram(filename string, inputFormat string) (*diagram.Diagram, error) {
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

	// If input format is explicitly specified, use it
	if inputFormat != "" && inputFormat != "json" {
		// Import from specified format
		registry := importer.NewImporterRegistry()
		d, err := registry.ImportWithFormat(string(data), inputFormat)
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

	// Auto-detect based on extension
	needImport := ext != ".json" && ext != ""

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

		// Auto-detect format
		d, err := registry.Import(string(data))

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
		if inputFormat != "" || importExtensions[ext] {
			registry := importer.NewImporterRegistry()

			var imported *diagram.Diagram
			if inputFormat != "" {
				imported, err = registry.ImportWithFormat(string(data), inputFormat)
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

// runMarkdownExtraction extracts and exports a diagram from markdown without interaction
func runMarkdownExtraction(filename string, blockIndex int, format string, outputFile string) error {
	// Read the markdown file
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("reading markdown file: %w", err)
	}

	// Scan for diagram blocks
	scanner := markdown.NewScanner(string(content))
	blocks := scanner.FindDiagramBlocks()

	if len(blocks) == 0 {
		return fmt.Errorf("no diagram blocks found in %s", filename)
	}

	// Determine which block to extract
	var selectedBlock markdown.DiagramBlock

	if blockIndex > 0 {
		// User specified a block index (1-based)
		if blockIndex > len(blocks) {
			return fmt.Errorf("block index %d is out of range (found %d blocks)", blockIndex, len(blocks))
		}
		selectedBlock = blocks[blockIndex-1]
	} else if len(blocks) == 1 {
		// Only one block, select it automatically
		selectedBlock = blocks[0]
	} else {
		return fmt.Errorf("multiple diagram blocks found, please specify which one with -block")
	}

	// Import the diagram from the block content
	registry := importer.NewImporterRegistry()
	var d *diagram.Diagram

	if selectedBlock.Type != "" {
		d, err = registry.ImportWithFormat(selectedBlock.Content, selectedBlock.Type)
	} else {
		d, err = registry.Import(selectedBlock.Content)
	}

	if err != nil {
		return fmt.Errorf("importing diagram from markdown block: %w", err)
	}

	// Parse export format
	exportFormat, err := export.ParseFormat(format)
	if err != nil {
		return fmt.Errorf("invalid format: %w", err)
	}

	// Create exporter
	exporter, err := export.NewExporter(exportFormat)
	if err != nil {
		return fmt.Errorf("creating exporter: %w", err)
	}

	// Export the diagram
	output, err := exporter.Export(d)
	if err != nil {
		return fmt.Errorf("exporting diagram: %w", err)
	}

	// Output to file or stdout
	if outputFile != "" {
		err = ioutil.WriteFile(outputFile, []byte(output), 0644)
		if err != nil {
			return fmt.Errorf("writing output file: %w", err)
		}
	} else {
		fmt.Print(output)
	}

	return nil
}

// runMarkdownMode handles editing diagram blocks within markdown files
func runMarkdownMode(filename string, blockIndex int) error {
	// Loop to allow returning to picker after editing
	for {
		// Read the markdown file
		content, err := ioutil.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("reading markdown file: %w", err)
		}

		// Scan for diagram blocks
		scanner := markdown.NewScanner(string(content))
		blocks := scanner.FindDiagramBlocks()

		if len(blocks) == 0 {
			return fmt.Errorf("no diagram blocks found in %s", filename)
		}

		// Determine which block to edit
		var selectedBlock markdown.DiagramBlock
		var selectedIndex int

		if blockIndex > 0 {
			// User specified a block index (1-based)
			if blockIndex > len(blocks) {
				return fmt.Errorf("block index %d is out of range (found %d blocks)", blockIndex, len(blocks))
			}
			selectedBlock = blocks[blockIndex-1]
			selectedIndex = blockIndex - 1
			// Reset blockIndex so we show the picker on next iteration
			blockIndex = 0
		} else if len(blocks) == 1 {
			// Only one block, select it automatically
			selectedBlock = blocks[0]
			selectedIndex = 0
			fmt.Printf("Editing %s block at line %d\n", selectedBlock.Type, selectedBlock.StartLine+1)
		} else {
			// Multiple blocks, show picker
			selectedIndex, err = showMarkdownPicker(blocks)
			if err != nil {
				return err
			}
			selectedBlock = blocks[selectedIndex]
		}

		// Import the diagram from the block content
		registry := importer.NewImporterRegistry()
		var d *diagram.Diagram

		// Try to import based on the block type
		if selectedBlock.Type != "" {
			d, err = registry.ImportWithFormat(selectedBlock.Content, selectedBlock.Type)
		} else {
			d, err = registry.Import(selectedBlock.Content)
		}

		if err != nil {
			return fmt.Errorf("importing diagram from markdown block: %w", err)
		}

		// Ensure all connections have unique IDs
		diagram.EnsureUniqueConnectionIDs(d)

		// Default arrows to true for all connections
		for i := range d.Connections {
			if !d.Connections[i].Arrow {
				d.Connections[i].Arrow = true
			}
		}

		// Create a temporary file to store the diagram JSON
		tempFile := fmt.Sprintf("/tmp/edd_markdown_%d.json", selectedIndex)
		dataJSON, err := json.MarshalIndent(d, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling diagram to JSON: %w", err)
		}
		err = ioutil.WriteFile(tempFile, dataJSON, 0644)
		if err != nil {
			return fmt.Errorf("writing temp file: %w", err)
		}

		// Store markdown context for later (using 1-based index for consistency)
		err = ioutil.WriteFile(tempFile+".ctx", []byte(fmt.Sprintf("%s\n%d\n%s", filename, selectedIndex+1, selectedBlock.Type)), 0644)
		if err != nil {
			return fmt.Errorf("writing context file: %w", err)
		}

		fmt.Printf("\n[Press :w to save back to markdown, :q to return to picker, :qq to exit]\n\n")

		// Run interactive mode with the temp file
		err = runInteractiveMode(tempFile, d.Type, nil, true) // true = markdown mode

		// Clean up temp files
		os.Remove(tempFile)
		os.Remove(tempFile + ".ctx")

		if err != nil {
			// Check if this is a special "quit to picker" signal
			if err.Error() == "return_to_picker" {
				// Continue the loop to show picker again
				continue
			}
			return err
		}

		// Normal quit - exit the loop
		return nil
	}
}

// showMarkdownPicker displays an interactive picker menu with live preview
func showMarkdownPicker(blocks []markdown.DiagramBlock) (int, error) {
	selected := 0

	// Setup terminal for raw mode
	if err := terminal.SetupTerminal(); err != nil {
		return 0, fmt.Errorf("failed to setup terminal: %w", err)
	}
	defer terminal.RestoreTerminal()

	// Enter alternate screen
	fmt.Print("\033[?1049h\033[2J\033[H\033[?25l")
	defer func() {
		fmt.Print("\033[?25h\033[?1049l")
	}()

	for {
		// Clear and render picker
		fmt.Print("\033[H\033[2J")

		// Get terminal size
		width, height := terminal.GetTerminalSize()

		// Calculate layout - split screen vertically
		listWidth := 40
		if width < 80 {
			listWidth = width / 2
		}
		previewX := listWidth + 2

		// Render list
		fmt.Print("\033[1;1H\033[1mDiagram Blocks:\033[0m")
		for i, block := range blocks {
			fmt.Printf("\033[%d;1H", i+3)
			if i == selected {
				fmt.Printf("\033[7m") // Reverse video for selection
			}
			info := markdown.FormatBlockInfo(block, i)
			if len(info) > listWidth-2 {
				info = info[:listWidth-5] + "..."
			}
			fmt.Printf(" %s ", info)
			if i == selected {
				fmt.Print("\033[0m")
			}
		}

		// Draw vertical separator
		for y := 1; y <= height; y++ {
			fmt.Printf("\033[%d;%dH│", y, listWidth+1)
		}

		// Render preview of selected diagram
		if selected >= 0 && selected < len(blocks) {
			block := blocks[selected]

			// Import the diagram
			registry := importer.NewImporterRegistry()
			var d *diagram.Diagram
			var err error

			if block.Type != "" {
				d, err = registry.ImportWithFormat(block.Content, block.Type)
			} else {
				d, err = registry.Import(block.Content)
			}

			if err == nil {
				// Ensure connections have unique IDs
				diagram.EnsureUniqueConnectionIDs(d)

				// Default arrows to true for all connections
				for i := range d.Connections {
					if !d.Connections[i].Arrow {
						d.Connections[i].Arrow = true
					}
				}

				// Render the diagram
				renderer := render.NewRenderer()
				output, err := renderer.Render(d)
				if err == nil {
					// Display preview
					lines := strings.Split(output, "\n")
					fmt.Printf("\033[1;%dH\033[1mPreview:\033[0m", previewX)

					// Calculate available space for preview
					maxLineWidth := width - previewX - 1 // Leave 1 char margin
					maxLines := height - 5 // Leave space for header and footer

					for i, line := range lines {
						if i >= maxLines {
							break
						}

						// Clear the line first to avoid artifacts
						fmt.Printf("\033[%d;%dH\033[K", i+3, previewX)

						// For preview, calculate visible width (excluding ANSI codes)
						visibleWidth := 0
						inEscape := false
						displayLine := ""

						for _, r := range line {
							if r == '\033' {
								inEscape = true
								displayLine += string(r)
							} else if inEscape {
								displayLine += string(r)
								if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
									inEscape = false
								}
							} else {
								if visibleWidth >= maxLineWidth {
									break // Stop once we've filled the preview width
								}
								displayLine += string(r)
								visibleWidth++
							}
						}

						fmt.Printf("\033[%d;%dH%s\033[0m", i+3, previewX, displayLine)
					}
				} else {
					fmt.Printf("\033[3;%dH\033[KError rendering: %v", previewX, err)
				}
			} else {
				fmt.Printf("\033[3;%dH\033[KError importing: %v", previewX, err)
			}
		}

		// Show help at bottom
		fmt.Printf("\033[%d;1H\033[1m↑/↓: Navigate | Enter: Select | q: Quit\033[0m", height)

		// Read key (with escape sequence support)
		var b [3]byte
		n, _ := os.Stdin.Read(b[:1])
		if n == 0 {
			continue
		}

		key := b[0]

		// Handle escape sequences for arrow keys
		if key == 27 { // ESC
			// Try to read more bytes for escape sequence
			n, _ = os.Stdin.Read(b[1:3])
			if n == 2 && b[1] == '[' {
				// Arrow key
				switch b[2] {
				case 'A': // Up
					if selected > 0 {
						selected--
					}
					continue
				case 'B': // Down
					if selected < len(blocks)-1 {
						selected++
					}
					continue
				}
			}
			// Just ESC
			return 0, fmt.Errorf("quit")
		}

		switch key {
		case 'q', 'Q':
			return 0, fmt.Errorf("quit")
		case 13, 10: // Enter
			return selected, nil
		case 'j': // j for down
			if selected < len(blocks)-1 {
				selected++
			}
		case 'k': // k for up
			if selected > 0 {
				selected--
			}
		}
	}
}

// runInteractiveMode launches the TUI editor with optional demo mode
// This is the main entry point for interactive editing
func runInteractiveMode(filename string, diagramType string, demoSettings *terminal.DemoSettings, markdownMode ...bool) error {
	// Create the real renderer
	renderer := editor.NewRealRenderer()

	// Create TUI editor
	tui := editor.NewTUIEditor(renderer)

	// Set markdown mode if specified
	isMarkdownMode := len(markdownMode) > 0 && markdownMode[0]
	tui.SetMarkdownMode(isMarkdownMode)

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


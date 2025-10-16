package main

import (
	"edd/importer"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	var (
		inputFile = flag.String("i", "", "Input file path")
		format    = flag.String("f", "", "Format (mermaid, plantuml, graphviz, d2) - auto-detect if not specified")
		output    = flag.String("o", "", "Output file path (default: stdout)")
	)

	flag.Parse()

	if *inputFile == "" {
		fmt.Fprintf(os.Stderr, "Error: input file required (-i)\n")
		flag.Usage()
		os.Exit(1)
	}

	// Read input file
	content, err := ioutil.ReadFile(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err)
		os.Exit(1)
	}

	// Create importer registry
	registry := importer.NewImporterRegistry()

	// Import the diagram
	var diagram interface{}
	if *format != "" {
		diagram, err = registry.ImportWithFormat(string(content), *format)
	} else {
		diagram, err = registry.Import(string(content))
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error importing diagram: %v\n", err)
		os.Exit(1)
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(diagram, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting to JSON: %v\n", err)
		os.Exit(1)
	}

	// Output result
	if *output != "" {
		err = ioutil.WriteFile(*output, jsonData, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully imported diagram to %s\n", *output)
	} else {
		fmt.Println(string(jsonData))
	}
}
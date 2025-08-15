package demo

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"
)

// Command represents a single demo command
type Command struct {
	Type     string `json:"type"`     // "key", "text", "pause"
	Value    string `json:"value"`    // the key/text to send
	Delay    int    `json:"delay"`    // base delay in milliseconds
	Variance int    `json:"variance"` // random variance in ms (±variance)
}

// Script represents a demo script
type Script struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Commands    []Command `json:"commands"`
	BaseDelay   int       `json:"base_delay"`   // default delay between commands
	BaseVariance int      `json:"base_variance"` // default variance
}

// Player plays back demo scripts
type Player struct {
	script      *Script
	onKey       func(rune)
	onText      func(string)
	stopChan    chan bool
	isPlaying   bool
}

// NewPlayer creates a new demo player
func NewPlayer(onKey func(rune), onText func(string)) *Player {
	return &Player{
		onKey:    onKey,
		onText:   onText,
		stopChan: make(chan bool, 1),
	}
}

// LoadScript loads a demo script from a file
func (p *Player) LoadScript(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read demo script: %w", err)
	}
	
	var script Script
	if err := json.Unmarshal(data, &script); err != nil {
		return fmt.Errorf("failed to parse demo script: %w", err)
	}
	
	// Set defaults
	if script.BaseDelay == 0 {
		script.BaseDelay = 300 // 300ms default
	}
	if script.BaseVariance == 0 {
		script.BaseVariance = 100 // ±100ms default
	}
	
	p.script = &script
	return nil
}

// Play starts playing the loaded script
func (p *Player) Play() error {
	if p.script == nil {
		return fmt.Errorf("no script loaded")
	}
	if p.isPlaying {
		return fmt.Errorf("already playing")
	}
	
	p.isPlaying = true
	go p.playScript()
	return nil
}

// Stop stops the current playback
func (p *Player) Stop() {
	if p.isPlaying {
		p.stopChan <- true
		p.isPlaying = false
	}
}

// IsPlaying returns whether a demo is currently playing
func (p *Player) IsPlaying() bool {
	return p.isPlaying
}

// playScript executes the script commands
func (p *Player) playScript() {
	defer func() {
		p.isPlaying = false
	}()
	
	for _, cmd := range p.script.Commands {
		select {
		case <-p.stopChan:
			return
		default:
			// Calculate delay with variance
			delay := cmd.Delay
			if delay == 0 {
				delay = p.script.BaseDelay
			}
			
			variance := cmd.Variance
			if variance == 0 {
				variance = p.script.BaseVariance
			}
			
			// Add random variance for natural feel
			if variance > 0 {
				delay += rand.Intn(variance*2) - variance // ±variance
			}
			
			// Ensure minimum delay
			if delay < 50 {
				delay = 50
			}
			
			// Execute command
			switch cmd.Type {
			case "key":
				// Send individual key presses
				for _, ch := range cmd.Value {
					p.onKey(ch)
					// Small delay between characters for natural typing
					if len(cmd.Value) > 1 {
						time.Sleep(time.Duration(30+rand.Intn(40)) * time.Millisecond)
					}
				}
				
			case "text":
				// Send text as a block
				p.onText(cmd.Value)
				
			case "pause":
				// Just pause, no action
				
			default:
				// Unknown command type, skip
			}
			
			// Wait before next command
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
	}
}

// GenerateExample creates an example demo script
func GenerateExample() string {
	script := Script{
		Name:        "Basic Diagram Demo",
		Description: "Creates a simple flow diagram",
		BaseDelay:   400,
		BaseVariance: 150,
		Commands: []Command{
			{Type: "key", Value: "a", Delay: 1000}, // Add node
			{Type: "text", Value: "Start"},
			{Type: "key", Value: "\n"}, // Enter to confirm
			
			{Type: "key", Value: "a", Delay: 600}, // Add another node
			{Type: "text", Value: "Process"},
			{Type: "key", Value: "\n"},
			
			{Type: "key", Value: "a", Delay: 600}, // Add third node
			{Type: "text", Value: "End"},
			{Type: "key", Value: "\n"},
			
			{Type: "key", Value: "c", Delay: 800}, // Connect mode
			{Type: "key", Value: "a"},             // Select first node
			{Type: "key", Value: "b"},             // Select second node
			
			{Type: "key", Value: "c", Delay: 600}, // Connect again
			{Type: "key", Value: "b"},             // Select second node
			{Type: "key", Value: "c"},             // Select third node
			
			{Type: "pause", Delay: 2000}, // Pause to show result
		},
	}
	
	data, _ := json.MarshalIndent(script, "", "  ")
	return string(data)
}
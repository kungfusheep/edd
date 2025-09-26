package markdown

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// DiagramBlock represents a diagram code block found in markdown
type DiagramBlock struct {
	Type        string // mermaid, plantuml, etc.
	Content     string // The diagram content
	StartLine   int    // Line number where block starts (0-based)
	EndLine     int    // Line number where block ends
	Indent      string // Indentation before the code fence
	ContentHash string // SHA256 hash of the original content for validation
}

// Scanner finds and extracts diagram blocks from markdown content
type Scanner struct {
	content string
	lines   []string
}

// NewScanner creates a new markdown scanner
func NewScanner(content string) *Scanner {
	return &Scanner{
		content: content,
		lines:   strings.Split(content, "\n"),
	}
}

// UpdateContent updates the scanner's internal content after a successful replacement
func (s *Scanner) UpdateContent(newContent string) {
	s.content = newContent
	s.lines = strings.Split(newContent, "\n")
}

// FindDiagramBlocks finds all diagram code blocks in the markdown
func (s *Scanner) FindDiagramBlocks() []DiagramBlock {
	var blocks []DiagramBlock
	inBlock := false
	var currentBlock *DiagramBlock

	for i, line := range s.lines {
		// Check for code fence start
		if !inBlock {
			trimmed := strings.TrimLeft(line, " \t")
			indent := line[:len(line)-len(trimmed)]

			// Check for ```mermaid, ```plantuml, etc.
			if strings.HasPrefix(trimmed, "```") {
				lang := strings.TrimPrefix(trimmed, "```")
				lang = strings.TrimSpace(lang)

				// Check if it's a diagram language we support
				if isDiagramLanguage(lang) {
					inBlock = true
					currentBlock = &DiagramBlock{
						Type:      lang,
						StartLine: i,
						Indent:    indent,
						Content:   "",
					}
				}
			}
		} else {
			// Check for code fence end
			trimmed := strings.TrimLeft(line, " \t")
			if strings.HasPrefix(trimmed, "```") {
				inBlock = false
				currentBlock.EndLine = i
				// Calculate hash of the content for later validation
				hash := sha256.Sum256([]byte(currentBlock.Content))
				currentBlock.ContentHash = hex.EncodeToString(hash[:])
				blocks = append(blocks, *currentBlock)
				currentBlock = nil
			} else {
				// Add content line
				if currentBlock.Content != "" {
					currentBlock.Content += "\n"
				}
				currentBlock.Content += line
			}
		}
	}

	return blocks
}

// ValidateBlockUnchanged checks if a block's content matches its original hash
func (s *Scanner) ValidateBlockUnchanged(block DiagramBlock) error {
	// Extract the current content of the block
	if block.StartLine < 0 || block.EndLine >= len(s.lines) || block.StartLine >= block.EndLine {
		return fmt.Errorf("invalid block boundaries")
	}

	var currentContent strings.Builder
	for i := block.StartLine + 1; i < block.EndLine; i++ {
		if i > block.StartLine+1 {
			currentContent.WriteString("\n")
		}
		// Remove indentation when extracting
		line := s.lines[i]
		if strings.HasPrefix(line, block.Indent) {
			line = line[len(block.Indent):]
		}
		currentContent.WriteString(line)
	}

	// Calculate hash and compare
	hash := sha256.Sum256([]byte(currentContent.String()))
	currentHash := hex.EncodeToString(hash[:])

	if currentHash != block.ContentHash {
		return fmt.Errorf("block content has been modified externally (hash mismatch)")
	}

	return nil
}

// ReplaceBlock replaces a diagram block's content in the markdown
// Returns the new markdown content and an error if validation fails
func (s *Scanner) ReplaceBlock(block DiagramBlock, newContent string) (string, error) {
	// Validate that the block boundaries haven't changed
	if block.StartLine < 0 || block.EndLine >= len(s.lines) || block.StartLine >= block.EndLine {
		return "", fmt.Errorf("invalid block boundaries: start=%d, end=%d, total lines=%d",
			block.StartLine, block.EndLine, len(s.lines))
	}

	// Verify the block markers are still in place
	startLine := s.lines[block.StartLine]
	endLine := s.lines[block.EndLine]

	// Check that start line still has the code fence with the right type
	trimmedStart := strings.TrimLeft(startLine, " \t")
	if !strings.HasPrefix(trimmedStart, "```"+block.Type) {
		return "", fmt.Errorf("block start marker has changed at line %d: expected '```%s', found '%s'",
			block.StartLine+1, block.Type, trimmedStart)
	}

	// Check that end line still has the closing fence
	trimmedEnd := strings.TrimLeft(endLine, " \t")
	if !strings.HasPrefix(trimmedEnd, "```") {
		return "", fmt.Errorf("block end marker has changed at line %d: expected '```', found '%s'",
			block.EndLine+1, trimmedEnd)
	}

	// Create new lines array
	newLines := make([]string, len(s.lines))
	copy(newLines, s.lines)

	// Replace the content lines (keeping the fence lines intact)
	contentLines := strings.Split(newContent, "\n")

	// Clear old content lines (between start+1 and end-1)
	deleteCount := block.EndLine - block.StartLine - 1
	if deleteCount > 0 {
		newLines = append(newLines[:block.StartLine+1], newLines[block.EndLine:]...)
	}

	// Insert new content lines with proper indentation
	insertPos := block.StartLine + 1
	for _, contentLine := range contentLines {
		// Preserve original indentation
		indentedLine := block.Indent + contentLine
		newLines = append(newLines[:insertPos], append([]string{indentedLine}, newLines[insertPos:]...)...)
		insertPos++
	}

	return strings.Join(newLines, "\n"), nil
}

// GetContent returns the current markdown content
func (s *Scanner) GetContent() string {
	return s.content
}

// isDiagramLanguage checks if a language identifier is a diagram type we support
func isDiagramLanguage(lang string) bool {
	switch strings.ToLower(lang) {
	case "mermaid", "plantuml", "puml", "graphviz", "dot", "d2":
		return true
	default:
		return false
	}
}

// FormatBlockInfo returns a human-readable description of a block
func FormatBlockInfo(block DiagramBlock, index int) string {
	// Extract first meaningful line of content for preview
	lines := strings.Split(strings.TrimSpace(block.Content), "\n")
	preview := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "@startuml") && !strings.HasPrefix(trimmed, "@enduml") {
			preview = trimmed
			if len(preview) > 50 {
				preview = preview[:47] + "..."
			}
			break
		}
	}

	return fmt.Sprintf("%d. %s (line %d): %s", index+1, block.Type, block.StartLine+1, preview)
}
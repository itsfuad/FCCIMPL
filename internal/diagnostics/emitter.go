package diagnostics

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"compiler/colors"
)

// SourceCache caches source file contents for error reporting
type SourceCache struct {
	files map[string][]string
}

func NewSourceCache() *SourceCache {
	return &SourceCache{
		files: make(map[string][]string),
	}
}

// GetLine retrieves a specific line from a source file
func (sc *SourceCache) GetLine(filepath string, line int) (string, error) {
	if lines, ok := sc.files[filepath]; ok {
		if line > 0 && line <= len(lines) {
			return lines[line-1], nil
		}
		return "", fmt.Errorf("line %d out of range", line)
	}

	// Load file
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	lines := make([]string, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	sc.files[filepath] = lines

	if line > 0 && line <= len(lines) {
		return lines[line-1], nil
	}

	return "", fmt.Errorf("line %d out of range", line)
}

// Emitter handles the rendering and output of diagnostics
type Emitter struct {
	cache *SourceCache
}

func NewEmitter() *Emitter {
	return &Emitter{
		cache: NewSourceCache(),
	}
}

// Emit renders and prints a diagnostic to stderr
func (e *Emitter) Emit(filepath string, diag *Diagnostic) {
	// Use filepath from diagnostic if available, otherwise use parameter
	if diag.FilePath != "" {
		filepath = diag.FilePath
	}

	// Print severity and message
	e.printHeader(diag)

	// Print each labeled location
	for _, label := range diag.Labels {
		e.printLabel(filepath, label)
	}

	// Print notes
	for _, note := range diag.Notes {
		e.printNote(note)
	}

	// Print help
	if diag.Help != "" {
		e.printHelp(diag.Help)
	}

	fmt.Fprintln(os.Stderr)
}

func (e *Emitter) printHeader(diag *Diagnostic) {
	var color colors.COLOR
	var severityStr string

	switch diag.Severity {
	case Error:
		color = colors.RED
		severityStr = "error"
	case Warning:
		color = colors.YELLOW
		severityStr = "warning"
	case Info:
		color = colors.BLUE
		severityStr = "info"
	case Hint:
		color = colors.CYAN
		severityStr = "hint"
	}

	color.Print(severityStr)
	if diag.Code != "" {
		fmt.Fprintf(os.Stderr, "[%s]", diag.Code)
	}
	fmt.Fprintf(os.Stderr, ": %s\n", diag.Message)
}

func (e *Emitter) printLabel(filepath string, label Label) {
	if label.Location == nil || label.Location.Start == nil {
		return
	}

	start := label.Location.Start
	end := label.Location.End
	if end == nil {
		end = start
	}

	// Print location header
	colors.BLUE.Printf("  --> %s:%d:%d\n", filepath, start.Line, start.Column)

	// Print line number gutter width
	lineNumWidth := len(fmt.Sprintf("%d", start.Line))
	if end.Line > start.Line {
		endWidth := len(fmt.Sprintf("%d", end.Line))
		if endWidth > lineNumWidth {
			lineNumWidth = endWidth
		}
	}

	// Print separator
	colors.GREY.Print(strings.Repeat(" ", lineNumWidth))
	colors.GREY.Println(" |")

	// For single-line errors
	if start.Line == end.Line {
		e.printSingleLineLabel(filepath, start.Line, start.Column, end.Column, label, lineNumWidth)
	} else {
		// For multi-line errors
		e.printMultiLineLabel(filepath, start.Line, end.Line, start.Column, end.Column, label, lineNumWidth)
	}
}

func (e *Emitter) printSingleLineLabel(filepath string, line, startCol, endCol int, label Label, lineNumWidth int) {
	// Try to get previous line for context (if not empty)
	if line > 1 {
		prevLine, err := e.cache.GetLine(filepath, line-1)
		if err == nil && strings.TrimSpace(prevLine) != "" {
			// Print previous line in grey for context
			colors.GREY.Printf("%*d | ", lineNumWidth, line-1)
			colors.GREY.Println(prevLine)
		}
	}

	// Get the error line
	sourceLine, err := e.cache.GetLine(filepath, line)
	if err != nil {
		return
	}

	// Print line number and source (line number in grey)
	colors.GREY.Printf("%*d | ", lineNumWidth, line)
	fmt.Fprintln(os.Stderr, sourceLine)

	// Print underline
	colors.GREY.Print(strings.Repeat(" ", lineNumWidth))
	colors.GREY.Print(" | ")

	// Calculate underline position and length
	padding := startCol - 1
	length := endCol - startCol
	if length <= 0 {
		length = 1
	}

	// Choose color and style based on label style
	var underlineColor colors.COLOR
	var underlineChar string

	if label.Style == Primary {
		underlineColor = colors.RED
		// Use ^ for single character, ~ for multiple
		if length == 1 {
			underlineChar = "^"
		} else {
			underlineChar = "~"
		}
	} else {
		underlineColor = colors.BLUE
		underlineChar = "-"
	}

	// Print padding and underline
	fmt.Fprint(os.Stderr, strings.Repeat(" ", padding))
	underlineColor.Print(strings.Repeat(underlineChar, length))

	// Print label message (only if not empty)
	if label.Message != "" {
		underlineColor.Printf(" %s", label.Message)
	}
	fmt.Fprintln(os.Stderr)

	// Print separator
	colors.GREY.Print(strings.Repeat(" ", lineNumWidth))
	colors.GREY.Println(" |")
}

func (e *Emitter) printMultiLineLabel(filepath string, startLine, endLine, startCol, endCol int, label Label, lineNumWidth int) {
	// Print start line
	sourceLine, err := e.cache.GetLine(filepath, startLine)
	if err != nil {
		return
	}

	colors.BLUE.Printf("%*d | ", lineNumWidth, startLine)
	fmt.Fprintln(os.Stderr, sourceLine)

	// Print underline for start
	colors.BLUE.Print(strings.Repeat(" ", lineNumWidth))
	colors.BLUE.Print(" | ")

	var underlineColor colors.COLOR
	if label.Style == Primary {
		underlineColor = colors.RED
	} else {
		underlineColor = colors.BLUE
	}

	padding := startCol - 1
	fmt.Fprint(os.Stderr, strings.Repeat(" ", padding))
	underlineColor.Println("^--- error starts here")

	// Print ellipsis for middle lines if there are many
	if endLine-startLine > 5 {
		colors.BLUE.Print(strings.Repeat(" ", lineNumWidth))
		colors.BLUE.Println(" | ...")
	} else {
		// Print middle lines
		for i := startLine + 1; i < endLine; i++ {
			line, err := e.cache.GetLine(filepath, i)
			if err != nil {
				continue
			}
			colors.BLUE.Printf("%*d | ", lineNumWidth, i)
			fmt.Fprintln(os.Stderr, line)
		}
	}

	// Print end line
	endSourceLine, err := e.cache.GetLine(filepath, endLine)
	if err == nil {
		colors.BLUE.Printf("%*d | ", lineNumWidth, endLine)
		fmt.Fprintln(os.Stderr, endSourceLine)

		// Print underline for end
		colors.BLUE.Print(strings.Repeat(" ", lineNumWidth))
		colors.BLUE.Print(" | ")
		endPadding := endCol - 1
		fmt.Fprint(os.Stderr, strings.Repeat(" ", endPadding))
		underlineColor.Print("^")

		if label.Message != "" {
			underlineColor.Printf(" %s", label.Message)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Print separator
	colors.BLUE.Print(strings.Repeat(" ", lineNumWidth))
	colors.BLUE.Println(" |")
}

func (e *Emitter) printNote(note Note) {
	colors.CYAN.Print("  = note: ")
	fmt.Fprintln(os.Stderr, note.Message)
}

func (e *Emitter) printHelp(help string) {
	colors.GREEN.Print("  = help: ")
	fmt.Fprintln(os.Stderr, help)
}

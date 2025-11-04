package diagnostics

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"compiler/colors"
)

const (
	STR_MULTIPLIER = "%*d | "
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

// labelContext groups parameters for printing labels to reduce parameter count
type labelContext struct {
	filepath     string
	line         int
	startLine    int
	endLine      int
	startCol     int
	endCol       int
	label        Label
	lineNumWidth int
	severity     Severity
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
		e.printLabel(filepath, label, diag.Severity)
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
		color = colors.BOLD_RED
		severityStr = "error"
	case Warning:
		color = colors.BOLD_YELLOW
		severityStr = "warning"
	case Info:
		color = colors.BOLD_CYAN
		severityStr = "info"
	case Hint:
		color = colors.BOLD_PURPLE
		severityStr = "hint"
	}

	color.Print(severityStr)
	if diag.Code != "" {
		fmt.Fprintf(os.Stderr, "[%s]", diag.Code)
	}
	fmt.Fprintf(os.Stderr, ": %s\n", diag.Message)
}

func (e *Emitter) printLabel(filepath string, label Label, severity Severity) {
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

	// Create context for label printing
	ctx := labelContext{
		filepath:     filepath,
		startLine:    start.Line,
		endLine:      end.Line,
		startCol:     start.Column,
		endCol:       end.Column,
		label:        label,
		lineNumWidth: lineNumWidth,
		severity:     severity,
	}

	// For single-line errors
	if start.Line == end.Line {
		ctx.line = start.Line
		e.printSingleLineLabel(ctx)
	} else {
		// For multi-line errors
		e.printMultiLineLabel(ctx)
	}
}

func (e *Emitter) printSingleLineLabel(ctx labelContext) {
	// Try to get previous line for context (if not empty)
	if ctx.line > 1 {
		prevLine, err := e.cache.GetLine(ctx.filepath, ctx.line-1)
		if err == nil && strings.TrimSpace(prevLine) != "" {
			// Print previous line in grey for context
			colors.GREY.Printf(STR_MULTIPLIER, ctx.lineNumWidth, ctx.line-1)
			colors.GREY.Println(prevLine)
		}
	}

	// Get the error line
	sourceLine, err := e.cache.GetLine(ctx.filepath, ctx.line)
	if err != nil {
		return
	}

	// Print line number and source (line number in grey)
	colors.GREY.Printf(STR_MULTIPLIER, ctx.lineNumWidth, ctx.line)
	fmt.Fprintln(os.Stderr, sourceLine)

	// Print underline
	colors.GREY.Print(strings.Repeat(" ", ctx.lineNumWidth))
	colors.GREY.Print(" | ")

	// Calculate underline position and length
	padding := ctx.startCol - 1
	length := ctx.endCol - ctx.startCol
	if length <= 0 {
		length = 1
	}

	// Choose color and style based on severity and label style
	var underlineColor colors.COLOR
	var underlineChar string

	if ctx.label.Style == Primary {
		// Primary labels use severity-based colors
		switch ctx.severity {
		case Error:
			underlineColor = colors.RED
		case Warning:
			underlineColor = colors.YELLOW
		case Info:
			underlineColor = colors.BLUE
		case Hint:
			underlineColor = colors.PURPLE
		default:
			underlineColor = colors.RED
		}
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
	if ctx.label.Message != "" {
		underlineColor.Printf(" %s", ctx.label.Message)
	}
	fmt.Fprintln(os.Stderr)

	// Print separator
	colors.GREY.Print(strings.Repeat(" ", ctx.lineNumWidth))
	colors.GREY.Println(" |")
}

func (e *Emitter) printMultiLineLabel(ctx labelContext) {
	// Print start line
	sourceLine, err := e.cache.GetLine(ctx.filepath, ctx.startLine)
	if err != nil {
		return
	}

	colors.BLUE.Printf(STR_MULTIPLIER, ctx.lineNumWidth, ctx.startLine)
	fmt.Fprintln(os.Stderr, sourceLine)

	// Print underline for start
	colors.BLUE.Print(strings.Repeat(" ", ctx.lineNumWidth))
	colors.BLUE.Print(" | ")

	var underlineColor colors.COLOR
	if ctx.label.Style == Primary {
		// Primary labels use severity-based colors
		switch ctx.severity {
		case Error:
			underlineColor = colors.BOLD_RED
		case Warning:
			underlineColor = colors.BOLD_YELLOW
		case Info:
			underlineColor = colors.BOLD_CYAN
		case Hint:
			underlineColor = colors.BOLD_PURPLE
		default:
			underlineColor = colors.RED
		}
	} else {
		underlineColor = colors.BLUE
	}

	padding := ctx.startCol - 1
	fmt.Fprint(os.Stderr, strings.Repeat(" ", padding))
	underlineColor.Println("^--- error starts here")

	// Print ellipsis for middle lines if there are many
	if ctx.endLine-ctx.startLine > 5 {
		colors.BLUE.Print(strings.Repeat(" ", ctx.lineNumWidth))
		colors.BLUE.Println(" | ...")
	} else {
		// Print middle lines
		for i := ctx.startLine + 1; i < ctx.endLine; i++ {
			line, err := e.cache.GetLine(ctx.filepath, i)
			if err != nil {
				continue
			}
			colors.BLUE.Printf(STR_MULTIPLIER, ctx.lineNumWidth, i)
			fmt.Fprintln(os.Stderr, line)
		}
	}

	// Print end line
	endSourceLine, err := e.cache.GetLine(ctx.filepath, ctx.endLine)
	if err == nil {
		colors.BLUE.Printf(STR_MULTIPLIER, ctx.lineNumWidth, ctx.endLine)
		fmt.Fprintln(os.Stderr, endSourceLine)

		// Print underline for end
		colors.BLUE.Print(strings.Repeat(" ", ctx.lineNumWidth))
		colors.BLUE.Print(" | ")
		endPadding := ctx.endCol - 1
		fmt.Fprint(os.Stderr, strings.Repeat(" ", endPadding))
		underlineColor.Print("^")

		if ctx.label.Message != "" {
			underlineColor.Printf(" %s", ctx.label.Message)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Print separator
	colors.BLUE.Print(strings.Repeat(" ", ctx.lineNumWidth))
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

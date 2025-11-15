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

	// Rust-style: If we have labels, print them intelligently
	if len(diag.Labels) > 0 {
		// Verify we have exactly one primary label
		primaryCount := 0
		var primaryLabel Label
		secondaryLabels := []Label{}

		for _, label := range diag.Labels {
			if label.Style == Primary {
				primaryCount++
				primaryLabel = label
			} else {
				secondaryLabels = append(secondaryLabels, label)
			}
		}

		if primaryCount == 0 {
			// No primary label - this shouldn't happen but handle gracefully
			for _, label := range diag.Labels {
				e.printLabel(filepath, label, diag.Severity)
			}
		} else if primaryCount > 1 {
			// Multiple primary labels - print warning
			colors.BOLD_RED.Println("INTERNAL COMPILER ERROR: Multiple primary labels in diagnostic!")
			for _, label := range diag.Labels {
				e.printLabel(filepath, label, diag.Severity)
			}
		} else {
			// Exactly one primary label - use Rust-style rendering
			if len(secondaryLabels) == 0 {
				// Only primary label
				e.printLabel(filepath, primaryLabel, diag.Severity)
			} else if len(secondaryLabels) == 1 &&
				primaryLabel.Location != nil &&
				primaryLabel.Location.Start != nil &&
				secondaryLabels[0].Location != nil &&
				secondaryLabels[0].Location.Start != nil &&
				primaryLabel.Location.Start.Line == secondaryLabels[0].Location.Start.Line {
				// Primary + one secondary on same line - use compact format
				e.printCompactDualLabel(filepath, primaryLabel, secondaryLabels[0], diag.Severity)
			} else {
				// Primary + multiple secondaries OR different lines - use routed format
				e.printRoutedLabels(filepath, primaryLabel, secondaryLabels, diag.Severity)
			}
		}
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
	fmt.Fprint(os.Stderr, ": ")
	color.Println(diag.Message)
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

// printCompactDualLabel prints primary + one secondary label on same line (Rust-style)
// Primary gets inline message, secondary gets connector line below
func (e *Emitter) printCompactDualLabel(filepath string, primary Label, secondary Label, severity Severity) {
	if primary.Location == nil || primary.Location.Start == nil {
		return
	}
	if secondary.Location == nil || secondary.Location.Start == nil {
		return
	}

	line := primary.Location.Start.Line

	primaryStart := primary.Location.Start
	primaryEnd := primary.Location.End
	if primaryEnd == nil {
		primaryEnd = primaryStart
	}

	secondaryStart := secondary.Location.Start
	secondaryEnd := secondary.Location.End
	if secondaryEnd == nil {
		secondaryEnd = secondaryStart
	}

	// Print location header (point to primary)
	colors.BLUE.Printf("  --> %s:%d:%d\n", filepath, line, primaryStart.Column)

	lineNumWidth := len(fmt.Sprintf("%d", line))

	// Print separator
	colors.GREY.Print(strings.Repeat(" ", lineNumWidth))
	colors.GREY.Println(" |")

	// Get source line
	sourceLine, err := e.cache.GetLine(filepath, line)
	if err != nil {
		return
	}

	// Print line number and source
	colors.GREY.Printf(STR_MULTIPLIER, lineNumWidth, line)
	fmt.Fprintln(os.Stderr, sourceLine)

	// Calculate positions
	primaryPadding := primaryStart.Column - 1
	primaryLength := primaryEnd.Column - primaryStart.Column
	if primaryLength <= 0 {
		primaryLength = 1
	}

	secondaryPadding := secondaryStart.Column - 1
	secondaryLength := secondaryEnd.Column - secondaryStart.Column
	if secondaryLength <= 0 {
		secondaryLength = 1
	}

	// Get colors
	primaryColor := e.getSeverityColor(severity)
	secondaryColor := colors.BLUE

	// Determine character style
	primaryChar := "^"
	if primaryLength > 1 {
		primaryChar = "~"
	}
	secondaryChar := "-"

	// Line 1: Show both underlines, primary message inline
	colors.GREY.Print(strings.Repeat(" ", lineNumWidth))
	colors.GREY.Print(" | ")

	if secondaryPadding < primaryPadding {
		// Secondary is on the left
		fmt.Fprint(os.Stderr, strings.Repeat(" ", secondaryPadding))
		secondaryColor.Print(secondaryChar)

		spaceBetween := primaryPadding - secondaryPadding - 1
		fmt.Fprint(os.Stderr, strings.Repeat(" ", spaceBetween))

		primaryColor.Print(strings.Repeat(primaryChar, primaryLength))
		if primary.Message != "" {
			primaryColor.Printf(" %s", primary.Message)
		}
	} else {
		// Primary is on the left or same position
		fmt.Fprint(os.Stderr, strings.Repeat(" ", primaryPadding))
		primaryColor.Print(strings.Repeat(primaryChar, primaryLength))

		if primaryPadding < secondaryPadding {
			spaceBetween := secondaryPadding - primaryPadding - primaryLength
			fmt.Fprint(os.Stderr, strings.Repeat(" ", spaceBetween))
			secondaryColor.Print(secondaryChar)
		}

		if primary.Message != "" {
			primaryColor.Printf(" %s", primary.Message)
		}
	}
	fmt.Fprintln(os.Stderr)

	// Line 2: Show vertical connector for secondary
	colors.GREY.Print(strings.Repeat(" ", lineNumWidth))
	colors.GREY.Print(" | ")
	fmt.Fprint(os.Stderr, strings.Repeat(" ", secondaryPadding))
	secondaryColor.Println("|")

	// Line 3: Show secondary message
	colors.GREY.Print(strings.Repeat(" ", lineNumWidth))
	colors.GREY.Print(" | ")
	fmt.Fprint(os.Stderr, strings.Repeat(" ", secondaryPadding))
	secondaryColor.Print(strings.Repeat(secondaryChar, 2))
	if secondary.Message != "" {
		secondaryColor.Printf(" %s", secondary.Message)
	}
	fmt.Fprintln(os.Stderr)

	// Print separator
	colors.GREY.Print(strings.Repeat(" ", lineNumWidth))
	colors.GREY.Println(" |")
}

// printRoutedLabels prints primary + multiple secondaries with routing (Rust-style)
// Handles cases where secondary labels are on different lines or need routing
func (e *Emitter) printRoutedLabels(filepath string, primary Label, secondaries []Label, severity Severity) {
	if primary.Location == nil || primary.Location.Start == nil {
		return
	}

	primaryLine := primary.Location.Start.Line
	primaryCol := primary.Location.Start.Column

	// Print location header
	colors.BLUE.Printf("  --> %s:%d:%d\n", filepath, primaryLine, primaryCol)

	// Collect all line numbers we need to show
	lineNumbers := []int{primaryLine}
	for _, sec := range secondaries {
		if sec.Location != nil && sec.Location.Start != nil {
			secLine := sec.Location.Start.Line
			// Add line if not already included
			found := false
			for _, ln := range lineNumbers {
				if ln == secLine {
					found = true
					break
				}
			}
			if !found {
				lineNumbers = append(lineNumbers, secLine)
			}
		}
	}

	// Sort line numbers
	for i := 0; i < len(lineNumbers); i++ {
		for j := i + 1; j < len(lineNumbers); j++ {
			if lineNumbers[i] > lineNumbers[j] {
				lineNumbers[i], lineNumbers[j] = lineNumbers[j], lineNumbers[i]
			}
		}
	}

	lineNumWidth := len(fmt.Sprintf("%d", lineNumbers[len(lineNumbers)-1]))

	// Print separator
	colors.GREY.Print(strings.Repeat(" ", lineNumWidth))
	colors.GREY.Println(" |")

	primaryColor := e.getSeverityColor(severity)
	secondaryColor := colors.BLUE

	// Print each line with appropriate labels
	for idx, lineNum := range lineNumbers {
		// Print ellipsis if there's a gap between lines
		if idx > 0 {
			prevLine := lineNumbers[idx-1]
			if lineNum-prevLine > 1 {
				colors.GREY.Print(strings.Repeat(" ", lineNumWidth))
				colors.GREY.Println(" ...")
				colors.GREY.Print(strings.Repeat(" ", lineNumWidth))
				colors.GREY.Println(" |")
			}
		}

		sourceLine, err := e.cache.GetLine(filepath, lineNum)
		if err != nil {
			continue
		}

		// Print line number and source
		colors.GREY.Printf(STR_MULTIPLIER, lineNumWidth, lineNum)
		fmt.Fprintln(os.Stderr, sourceLine)

		// Check for secondaries on this line first
		hasSecondary := false
		for _, sec := range secondaries {
			if sec.Location != nil && sec.Location.Start != nil && sec.Location.Start.Line == lineNum {
				hasSecondary = true
				break
			}
		}

		// Check if primary is on this line
		hasPrimary := lineNum == primaryLine

		// Only print underline section if there are labels on this line
		if hasPrimary || hasSecondary {
			colors.GREY.Print(strings.Repeat(" ", lineNumWidth))
			colors.GREY.Print(" | ")

			// Print primary label if on this line
			if hasPrimary {
				primaryStart := primary.Location.Start
				primaryEnd := primary.Location.End
				if primaryEnd == nil {
					primaryEnd = primaryStart
				}

				padding := primaryStart.Column - 1
				length := primaryEnd.Column - primaryStart.Column
				if length <= 0 {
					length = 1
				}

				char := "^"
				if length > 1 {
					char = "~"
				}

				fmt.Fprint(os.Stderr, strings.Repeat(" ", padding))
				primaryColor.Print(strings.Repeat(char, length))
				if primary.Message != "" {
					primaryColor.Printf(" %s", primary.Message)
				}
				fmt.Fprintln(os.Stderr)
			} else if hasSecondary {
				// Print secondary labels if on this line (and primary is not)
				for _, sec := range secondaries {
					if sec.Location != nil && sec.Location.Start != nil && sec.Location.Start.Line == lineNum {
						secStart := sec.Location.Start
						secEnd := sec.Location.End
						if secEnd == nil {
							secEnd = secStart
						}

						padding := secStart.Column - 1
						length := secEnd.Column - secStart.Column
						if length <= 0 {
							length = 1
						}

						fmt.Fprint(os.Stderr, strings.Repeat(" ", padding))
						secondaryColor.Print(strings.Repeat("-", length))
						if sec.Message != "" {
							secondaryColor.Printf(" %s", sec.Message)
						}
						fmt.Fprintln(os.Stderr)
						// Only print first secondary on this line
						break
					}
				}
			}
		}
	}

	// Print separator
	colors.GREY.Print(strings.Repeat(" ", lineNumWidth))
	colors.GREY.Println(" |")
}

// getSeverityColor returns the color for a given severity
func (e *Emitter) getSeverityColor(severity Severity) colors.COLOR {
	switch severity {
	case Error:
		return colors.RED
	case Warning:
		return colors.YELLOW
	case Info:
		return colors.BLUE
	case Hint:
		return colors.PURPLE
	default:
		return colors.RED
	}
}

// This file contains the fixed printCompactDualLabel implementation
// that respects label styles (Primary vs Secondary) instead of just position

package diagnostics

import (
	"compiler/colors"
	"compiler/internal/source"
	"fmt"
	"os"
	"strings"
)

// printCompactDualLabel prints two labels on the same line
// Primary label gets inline display, Secondary gets connector line below
func (e *Emitter) printCompactDualLabelFixed(filepath string, labels []Label, severity Severity) {
	if len(labels) != 2 {
		return
	}

	label1 := labels[0]
	label2 := labels[1]

	if label1.Location == nil || label1.Location.Start == nil {
		return
	}
	if label2.Location == nil || label2.Location.Start == nil {
		return
	}

	start1 := label1.Location.Start
	end1 := label1.Location.End
	if end1 == nil {
		end1 = start1
	}

	start2 := label2.Location.Start
	end2 := label2.Location.End
	if end2 == nil {
		end2 = start2
	}

	// Determine display order based on label STYLE, not position
	// Primary label: gets inline message (more prominent)
	// Secondary label: gets connector line below (contextual)
	var primaryLabel, secondaryLabel Label
	var primaryStart, primaryEnd, secondaryStart, secondaryEnd *source.Position

	if label1.Style == Primary {
		// label1 is primary, label2 is secondary
		primaryLabel = label1
		secondaryLabel = label2
		primaryStart, primaryEnd = start1, end1
		secondaryStart, secondaryEnd = start2, end2
	} else if label2.Style == Primary {
		// label2 is primary, label1 is secondary
		primaryLabel = label2
		secondaryLabel = label1
		primaryStart, primaryEnd = start2, end2
		secondaryStart, secondaryEnd = start1, end1
	} else {
		// Both have same style - fall back to position-based sorting
		// Rightmost gets inline display
		if start1.Column < start2.Column {
			primaryLabel = label2
			secondaryLabel = label1
			primaryStart, primaryEnd = start2, end2
			secondaryStart, secondaryEnd = start1, end1
		} else {
			primaryLabel = label1
			secondaryLabel = label2
			primaryStart, primaryEnd = start1, end1
			secondaryStart, secondaryEnd = start2, end2
		}
	}

	line := primaryStart.Line

	// Print location header (point to primary label)
	colors.BLUE.Printf("  --> %s:%d:%d\n", filepath, line, primaryStart.Column)

	// Calculate line number width
	lineNumWidth := len(fmt.Sprintf("%d", line))

	// Print separator
	colors.GREY.Print(strings.Repeat(" ", lineNumWidth))
	colors.GREY.Println(" |")

	// Get the source line
	sourceLine, err := e.cache.GetLine(filepath, line)
	if err != nil {
		return
	}

	// Print line number and source
	colors.GREY.Printf("%*d | ", lineNumWidth, line)
	fmt.Fprintln(os.Stderr, sourceLine)

	// Calculate positions and lengths
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

	// Get colors and characters for each label
	primaryColor := getLabelColorHelper(primaryLabel.Style, severity)
	secondaryColor := getLabelColorHelper(secondaryLabel.Style, severity)

	primaryChar := getLabelCharHelper(primaryLabel.Style, primaryLength)
	secondaryChar := getLabelCharHelper(secondaryLabel.Style, secondaryLength)

	// Line 1: Show both underlines, primary message inline
	colors.GREY.Print(strings.Repeat(" ", lineNumWidth))
	colors.GREY.Print(" | ")

	// Determine visual order (left to right)
	if secondaryPadding < primaryPadding {
		// Secondary is on the left
		fmt.Fprint(os.Stderr, strings.Repeat(" ", secondaryPadding))
		secondaryColor.Print(secondaryChar)

		spaceBetween := primaryPadding - secondaryPadding - 1
		fmt.Fprint(os.Stderr, strings.Repeat(" ", spaceBetween))

		primaryColor.Print(strings.Repeat(primaryChar, primaryLength))
		if primaryLabel.Message != "" {
			primaryColor.Printf(" %s", primaryLabel.Message)
		}
	} else {
		// Primary is on the left (or same position)
		fmt.Fprint(os.Stderr, strings.Repeat(" ", primaryPadding))
		primaryColor.Print(strings.Repeat(primaryChar, primaryLength))

		if primaryPadding < secondaryPadding {
			spaceBetween := secondaryPadding - primaryPadding - primaryLength
			fmt.Fprint(os.Stderr, strings.Repeat(" ", spaceBetween))
			secondaryColor.Print(secondaryChar)
		}

		if primaryLabel.Message != "" {
			primaryColor.Printf(" %s", primaryLabel.Message)
		}
	}
	fmt.Fprintln(os.Stderr)

	// Line 2: Show vertical bar for secondary label
	colors.GREY.Print(strings.Repeat(" ", lineNumWidth))
	colors.GREY.Print(" | ")
	fmt.Fprint(os.Stderr, strings.Repeat(" ", secondaryPadding))
	secondaryColor.Println("|")

	// Line 3: Show secondary label message
	colors.GREY.Print(strings.Repeat(" ", lineNumWidth))
	colors.GREY.Print(" | ")
	fmt.Fprint(os.Stderr, strings.Repeat(" ", secondaryPadding))
	secondaryColor.Print("--")
	if secondaryLabel.Message != "" {
		secondaryColor.Printf(" %s", secondaryLabel.Message)
	}
	fmt.Fprintln(os.Stderr)

	// Print separator
	colors.GREY.Print(strings.Repeat(" ", lineNumWidth))
	colors.GREY.Println(" |")
}

// Helper functions for label styling
func getLabelColorHelper(style LabelStyle, severity Severity) colors.COLOR {
	if style == Primary {
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
	return colors.BLUE
}

func getLabelCharHelper(style LabelStyle, length int) string {
	if style == Primary {
		if length == 1 {
			return "^"
		}
		return "~"
	}
	if length == 1 {
		return "^"
	}
	return "-"
}

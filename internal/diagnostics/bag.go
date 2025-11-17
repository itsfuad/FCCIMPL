package diagnostics

import (
	"bytes"
	"compiler/colors"
	"fmt"
	"io"
	"os"
	"sync"
)

// DiagnosticBag collects diagnostics during compilation
type DiagnosticBag struct {
	diagnostics []*Diagnostic
	filepath    string
	mu          sync.Mutex
	errorCount  int
	warnCount   int
}

// NewDiagnosticBag creates a new diagnostic bag for a file
func NewDiagnosticBag(filepath string) *DiagnosticBag {
	return &DiagnosticBag{
		diagnostics: make([]*Diagnostic, 0),
		filepath:    filepath,
	}
}

// Add adds a diagnostic to the bag
func (db *DiagnosticBag) Add(diag *Diagnostic) {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.diagnostics = append(db.diagnostics, diag)

	// If this is the first diagnostic with a filepath, use it as the bag's filepath
	if db.filepath == "" && diag.FilePath != "" {
		db.filepath = diag.FilePath
	}

	switch diag.Severity {
	case Error:
		db.errorCount++
	case Warning:
		db.warnCount++
	}
}

// HasErrors returns true if there are any errors
func (db *DiagnosticBag) HasErrors() bool {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.errorCount > 0
}

// ErrorCount returns the number of errors
func (db *DiagnosticBag) ErrorCount() int {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.errorCount
}

// WarningCount returns the number of warnings
func (db *DiagnosticBag) WarningCount() int {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.warnCount
}

// Diagnostics returns all diagnostics
func (db *DiagnosticBag) Diagnostics() []*Diagnostic {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.diagnostics
}

func (db *DiagnosticBag) EmitAll() {
	emitter := NewEmitter()

	db.mu.Lock()
	diagnostics := make([]*Diagnostic, len(db.diagnostics))
	copy(diagnostics, db.diagnostics)
	filepath := db.filepath
	db.mu.Unlock()

	for _, diag := range diagnostics {
		emitter.Emit(filepath, diag)
	}

	// Print summary
	if db.errorCount > 0 || db.warnCount > 0 {
		db.printSummary()
	}
}

// EmitAllToString emits all diagnostics to a string with ANSI codes
func (db *DiagnosticBag) EmitAllToString() string {
	return db.EmitAllToStringWithCache(nil)
}

// EmitAllToStringWithCache emits all diagnostics to a string with ANSI codes, using provided source cache
func (db *DiagnosticBag) EmitAllToStringWithCache(sourceLines []string) string {
	var buf bytes.Buffer
	emitter := NewEmitterWithWriter(&buf)

	// If source lines are provided, pre-populate the cache
	if sourceLines != nil {
		emitter.SetSourceLines(db.filepath, sourceLines)
	}

	db.mu.Lock()
	diagnostics := make([]*Diagnostic, len(db.diagnostics))
	copy(diagnostics, db.diagnostics)
	filepath := db.filepath
	errorCount := db.errorCount
	warnCount := db.warnCount
	db.mu.Unlock()

	for _, diag := range diagnostics {
		emitter.Emit(filepath, diag)
	}

	// Print summary to buffer
	if errorCount > 0 || warnCount > 0 {
		if errorCount > 0 {
			fmt.Fprintf(&buf, "\nCompilation failed with %d error(s)", errorCount)
			if warnCount > 0 {
				fmt.Fprintf(&buf, " and %d warning(s)", warnCount)
			}
			fmt.Fprintln(&buf)
		} else if warnCount > 0 {
			fmt.Fprintf(&buf, "\nCompilation succeeded with %d warning(s)\n", warnCount)
		}
	}

	return buf.String()
}

// EmitAllToHTML emits all diagnostics to an HTML string
func (db *DiagnosticBag) EmitAllToHTML() string {
	return db.EmitAllToHTMLWithCache(nil)
}

// EmitAllToHTMLWithCache emits all diagnostics to an HTML string, using provided source cache
func (db *DiagnosticBag) EmitAllToHTMLWithCache(sourceLines []string) string {
	ansiOutput := db.EmitAllToStringWithCache(sourceLines)
	return colors.ConvertANSIToHTML(ansiOutput)
}

// EmitAllToWriter emits all diagnostics to a specific writer
func (db *DiagnosticBag) EmitAllToWriter(w io.Writer) {
	emitter := NewEmitterWithWriter(w)

	db.mu.Lock()
	diagnostics := make([]*Diagnostic, len(db.diagnostics))
	copy(diagnostics, db.diagnostics)
	filepath := db.filepath
	db.mu.Unlock()

	for _, diag := range diagnostics {
		emitter.Emit(filepath, diag)
	}

	// Print summary
	if db.errorCount > 0 || db.warnCount > 0 {
		db.printSummaryToWriter(w)
	}
}

func (db *DiagnosticBag) printSummary() {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.errorCount > 0 {
		fmt.Fprintf(os.Stdout, "\nCompilation failed with %d error(s)", db.errorCount)
		if db.warnCount > 0 {
			fmt.Fprintf(os.Stdout, " and %d warning(s)", db.warnCount)
		}
		fmt.Println()
	} else if db.warnCount > 0 {
		fmt.Fprintf(os.Stdout, "\nCompilation succeeded with %d warning(s)\n", db.warnCount)
	}
}

func (db *DiagnosticBag) printSummaryToWriter(w io.Writer) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.errorCount > 0 {
		fmt.Fprintf(w, "\nCompilation failed with %d error(s)", db.errorCount)
		if db.warnCount > 0 {
			fmt.Fprintf(w, " and %d warning(s)", db.warnCount)
		}
		fmt.Fprintln(w)
	} else if db.warnCount > 0 {
		fmt.Fprintf(w, "\nCompilation succeeded with %d warning(s)\n", db.warnCount)
	}
}

// Clear removes all diagnostics
func (db *DiagnosticBag) Clear() {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.diagnostics = make([]*Diagnostic, 0)
	db.errorCount = 0
	db.warnCount = 0
}

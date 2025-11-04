package diagnostics

import (
	"fmt"
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

// EmitAll emits all diagnostics to stderr
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

func (db *DiagnosticBag) printSummary() {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.errorCount > 0 {
		fmt.Fprintf(os.Stderr, "\nCompilation failed with %d error(s)", db.errorCount)
		if db.warnCount > 0 {
			fmt.Fprintf(os.Stderr, " and %d warning(s)", db.warnCount)
		}
		fmt.Fprintln(os.Stderr)
	} else if db.warnCount > 0 {
		fmt.Fprintf(os.Stderr, "\nCompilation succeeded with %d warning(s)\n", db.warnCount)
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

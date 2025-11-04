package diagnostics

import (
	"compiler/internal/source"
)

// Severity represents the severity level of a diagnostic
type Severity int

const (
	Error Severity = iota
	Warning
	Info
	Hint
)

func (s Severity) String() string {
	switch s {
	case Error:
		return "error"
	case Warning:
		return "warning"
	case Info:
		return "info"
	case Hint:
		return "hint"
	default:
		return "unknown"
	}
}

// Label represents a labeled section of code in a diagnostic
type Label struct {
	Location *source.Location
	Message  string
	Style    LabelStyle
}

type LabelStyle int

const (
	Primary   LabelStyle = iota // The main error location (uses ^^^)
	Secondary                   // Additional context (uses ---)
)

// Note represents additional information attached to a diagnostic
type Note struct {
	Message string
}

// Diagnostic represents a compiler diagnostic (error, warning, etc.)
type Diagnostic struct {
	Severity Severity
	Message  string
	Code     string // Error code like "E0001"
	FilePath string // Source file for this diagnostic
	Labels   []Label
	Notes    []Note
	Help     string // Suggestion for fixing the error
}

// NewError creates a new error diagnostic
func NewError(message string) *Diagnostic {
	return &Diagnostic{
		Severity: Error,
		Message:  message,
		Labels:   make([]Label, 0),
		Notes:    make([]Note, 0),
	}
}

// NewWarning creates a new warning diagnostic
func NewWarning(message string) *Diagnostic {
	return &Diagnostic{
		Severity: Warning,
		Message:  message,
		Labels:   make([]Label, 0),
		Notes:    make([]Note, 0),
	}
}

// NewInfo creates a new info diagnostic
func NewInfo(message string) *Diagnostic {
	return &Diagnostic{
		Severity: Info,
		Message:  message,
		Labels:   make([]Label, 0),
		Notes:    make([]Note, 0),
	}
}

// WithCode sets the error code
func (d *Diagnostic) WithCode(code string) *Diagnostic {
	d.Code = code
	return d
}

// WithLabel adds a labeled location to the diagnostic
func (d *Diagnostic) WithLabel(filepath string, loc *source.Location, message string, style LabelStyle) *Diagnostic {
	if d.FilePath == "" {
		d.FilePath = filepath
	}
	d.Labels = append(d.Labels, Label{
		Location: loc,
		Message:  message,
		Style:    style,
	})
	return d
}

// WithPrimaryLabel adds a primary labeled location
func (d *Diagnostic) WithPrimaryLabel(filepath string, loc *source.Location, message string) *Diagnostic {
	return d.WithLabel(filepath, loc, message, Primary)
}

// WithSecondaryLabel adds a secondary labeled location
func (d *Diagnostic) WithSecondaryLabel(filepath string, loc *source.Location, message string) *Diagnostic {
	return d.WithLabel(filepath, loc, message, Secondary)
}

// WithNote adds a note to the diagnostic
func (d *Diagnostic) WithNote(message string) *Diagnostic {
	d.Notes = append(d.Notes, Note{Message: message})
	return d
}

// WithHelp sets helpful suggestion for fixing the error
func (d *Diagnostic) WithHelp(help string) *Diagnostic {
	d.Help = help
	return d
}

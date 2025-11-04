package diagnostics

import (
	"compiler/internal/source"
)

// Common diagnostic builders for the lexer

// UnexpectedCharacter creates a diagnostic for an unexpected character
func UnexpectedCharacter(filepath string, loc *source.Location, char rune) *Diagnostic {
	return NewError("unexpected character").
		WithCode(ErrUnexpectedCharacter).
		WithPrimaryLabel(filepath, loc, "unexpected character").
		WithHelp("remove this character or check if it's a typo")
}

// UnterminatedString creates a diagnostic for an unterminated string literal
func UnterminatedString(filepath string, loc *source.Location) *Diagnostic {
	return NewError("unterminated string literal").
		WithCode(ErrUnterminatedString).
		WithPrimaryLabel(filepath, loc, "string starts here").
		WithHelp("add a closing quote (\") to terminate the string")
}

// InvalidNumberLiteral creates a diagnostic for an invalid number
func InvalidNumberLiteral(filepath string, loc *source.Location, reason string) *Diagnostic {
	return NewError("invalid number literal").
		WithCode(ErrInvalidNumber).
		WithPrimaryLabel(filepath, loc, reason).
		WithHelp("check the number format")
}

// InvalidEscapeSequence creates a diagnostic for an invalid escape sequence
func InvalidEscapeSequence(filepath string, loc *source.Location, sequence string) *Diagnostic {
	return NewError("invalid escape sequence").
		WithCode(ErrInvalidEscape).
		WithPrimaryLabel(filepath, loc, "unknown escape sequence").
		WithNote("valid escape sequences are: \\n, \\t, \\r, \\\\, \\\", \\'").
		WithHelp("use a valid escape sequence or remove the backslash")
}

// Common diagnostic builders for the parser

// UnexpectedToken creates a diagnostic for an unexpected token
func UnexpectedToken(filepath string, loc *source.Location, found, expected string) *Diagnostic {
	msg := "unexpected token"
	if expected != "" {
		msg = "expected " + expected + ", found " + found
	}

	return NewError(msg).
		WithCode(ErrUnexpectedToken).
		WithPrimaryLabel(filepath, loc, "unexpected token here")
}

// ExpectedToken creates a diagnostic for a missing expected token
func ExpectedToken(filepath string, loc *source.Location, expected string) *Diagnostic {
	return NewError("expected "+expected).
		WithCode(ErrExpectedToken).
		WithPrimaryLabel(filepath, loc, "expected "+expected+" here")
}

// MissingIdentifier creates a diagnostic for a missing identifier
func MissingIdentifier(filepath string, loc *source.Location) *Diagnostic {
	return NewError("expected identifier").
		WithCode(ErrMissingIdentifier).
		WithPrimaryLabel(filepath, loc, "expected identifier here")
}

// Common diagnostic builders for type checker

// TypeMismatch creates a diagnostic for type mismatch
func TypeMismatch(filepath string, loc *source.Location, expected, found string) *Diagnostic {
	return NewError("type mismatch").
		WithCode(ErrTypeMismatch).
		WithPrimaryLabel(filepath, loc, "expected "+expected+", found "+found)
}

// UndefinedSymbol creates a diagnostic for undefined symbol
func UndefinedSymbol(filepath string, loc *source.Location, name string) *Diagnostic {
	return NewError("undefined symbol: "+name).
		WithCode(ErrUndefinedSymbol).
		WithPrimaryLabel(filepath, loc, "not found in this scope").
		WithHelp("check if the symbol is declared and imported correctly")
}

// RedeclaredSymbol creates a diagnostic for redeclared symbol
func RedeclaredSymbol(filepath string, newLoc, prevLoc *source.Location, name string) *Diagnostic {
	return NewError(name+" is already declared").
		WithCode(ErrRedeclaredSymbol).
		WithPrimaryLabel(filepath, newLoc, "redeclared here").
		WithSecondaryLabel(filepath, prevLoc, "previously declared here").
		WithHelp("use a different name or remove one of the declarations")
}

// WrongArgumentCount creates a diagnostic for wrong number of arguments
func WrongArgumentCount(filepath string, loc *source.Location, expected, found int) *Diagnostic {
	return NewError("wrong number of arguments").
		WithCode(ErrWrongArgumentCount).
		WithPrimaryLabel(filepath, loc, "expected "+string(rune(expected))+" arguments, found "+string(rune(found)))
}

// FieldNotFound creates a diagnostic for field not found
func FieldNotFound(filepath string, loc *source.Location, fieldName, typeName string) *Diagnostic {
	return NewError("field "+fieldName+" not found").
		WithCode(ErrFieldNotFound).
		WithPrimaryLabel(filepath, loc, typeName+" has no field "+fieldName).
		WithHelp("check the field name spelling")
}

// InvalidCast creates a diagnostic for invalid type cast
func InvalidCast(filepath string, loc *source.Location, from, to string) *Diagnostic {
	return NewError("cannot cast "+from+" to "+to).
		WithCode(ErrInvalidCast).
		WithPrimaryLabel(filepath, loc, "invalid cast")
}

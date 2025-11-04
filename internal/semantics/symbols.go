package semantics

import (
	"compiler/internal/frontend/ast"
)

// SymbolKind represents the kind of symbol (variable, constant, function, type, etc.)
type SymbolKind int

const (
	SymbolVar SymbolKind = iota
	SymbolConst
	SymbolType      // For type aliases and named types
	SymbolFunc      // For functions
	SymbolStruct    // For struct definitions
	SymbolEnum      // For enum definitions
	SymbolInterface // For interface definitions
	SymbolMethod    // For methods on types
	SymbolField     // For struct/enum fields
	SymbolParam     // For function parameters
)

// String returns a string representation of the SymbolKind
func (k SymbolKind) String() string {
	switch k {
	case SymbolVar:
		return "variable"
	case SymbolConst:
		return "constant"
	case SymbolType:
		return "type"
	case SymbolFunc:
		return "function"
	case SymbolStruct:
		return "struct"
	case SymbolEnum:
		return "enum"
	case SymbolInterface:
		return "interface"
	case SymbolMethod:
		return "method"
	case SymbolField:
		return "field"
	case SymbolParam:
		return "parameter"
	default:
		return "unknown"
	}
}

// Symbol represents a named entity in the program (variable, constant, type, etc.)
//
// AUTHORITATIVE DATA:
// - This is the single source of truth for all semantic information about a declaration
// - Type field holds type information (populated during type checking)
// - Decl field links back to the AST node that declared this symbol
// - No duplication with AST or side maps
type Symbol struct {
	Name string
	Kind SymbolKind
	Type Type     // Type information (set by type checker, Phase 5)
	Decl ast.Node // Back-reference to the declaration AST node
	// For functions/methods: scope containing local variables
	SelfScope *SymbolTable
}

// NewSymbol creates a new symbol with the given properties
func NewSymbol(name string, kind SymbolKind, semanticType Type) *Symbol {
	return &Symbol{
		Name: name,
		Kind: kind,
		Type: semanticType,
	}
}

// NewSymbolWithDecl creates a new symbol with declaration node and location
func NewSymbolWithDecl(name string, kind SymbolKind, decl ast.Node) *Symbol {
	return &Symbol{
		Name: name,
		Kind: kind,
		Decl: decl,
	}
}

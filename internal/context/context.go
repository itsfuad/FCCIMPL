// Package context provides a shared compilation context for all compiler phases
//
// ARCHITECTURE DESIGN:
// This package implements the central "context" pattern used in production compilers
// (Rustc, Zig, LLVM). All compiler phases are stateless workers that receive a
// CompilerContext and operate on SourceFile objects within it.
//
// Key principles:
// 1. Single source of truth: All global state lives in CompilerContext
// 2. Phases are workers: Lexer, Parser, etc. don't own state
// 3. Incremental compilation ready: Each SourceFile tracks its own compilation state
// 4. Thread-safe design: Context can be protected with locks for parallel compilation
// 5. No duplication: SymbolTable is authoritative for symbols, Symbol.Type for types
//
// AUTHORITATIVE DATA SOURCES:
// - SymbolTable (in Scope): The SINGLE source of truth for all symbols in a module
// - Symbol.Type: The SINGLE source of truth for type information
// - Symbol.Decl: Back-reference to AST node that declared the symbol
// - Symbol.Exported: Determines visibility across module boundaries
//
// NO side maps (e.g., ExprTypes, NamedTypes) should be maintained. All semantic
// information flows through the SymbolTable hierarchy and Symbol structs.
package context

import (
	"compiler/internal/diagnostics"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/semantics"
	"compiler/internal/types"
	"sync"
)

// CompilerContext is the central hub for all compilation state.
// This is the ONLY place where global compiler state should live.
//
// Thread safety: All mutations should go through methods that can add locking.
// Future: Add sync.RWMutex for parallel compilation support.
type CompilerContext struct {
	// Diagnostics - centralized error and warning collection
	// All phases report here instead of storing their own errors
	Diagnostics *diagnostics.DiagnosticBag

	// Files - maps absolute file path -> SourceFile
	// This is the single registry of all files in the compilation
	Files map[string]*SourceFile

	// SemanticModels - maps file path -> semantic analysis results
	// Populated during Collector/Resolver/TypeChecker phases
	SemanticModels map[string]*SemanticModel

	// UniverseScope - root scope containing built-in types and functions
	// This is the parent of all module scopes
	// Contains: i32, f64, bool, string, print, etc.
	UniverseScope *semantics.SymbolTable

	// Options - compiler configuration
	Options *CompilerOptions

	// FileOrder - tracks order files were added (for deterministic builds)
	FileOrder []string

	// Mutex for thread-safe operations (future: parallel compilation)
	mu sync.RWMutex
}

// SourceFile represents ONLY the lexical and syntactic state of a source file.
// This is the output of Lexer (Phase 1) and Parser (Phase 2).
//
// SEPARATION OF CONCERNS:
// - SourceFile = Lexical + Syntactic (source, tokens, AST)
// - SemanticModel = Semantic analysis (symbols, scopes, types)
//
// This separation keeps phase boundaries clean and enables:
// - Easy serialization of AST without semantic data
// - Incremental re-parsing without losing semantic info
// - Thread-safe parallel parsing
type SourceFile struct {
	// Source metadata
	Path    string // Absolute file path
	Content string // Raw source code

	// Phase 1: Lexer output
	Tokens []lexer.Token

	// Phase 2: Parser output
	AST *ast.Module // Pure syntax tree (no semantic annotations yet)
}

// SemanticModel holds all semantic analysis results for a single file/module.
// This is populated by Collector (Phase 3), Resolver (Phase 4), and TypeChecker (Phase 5).
//
// IMPORTANT: This is the semantic layer - completely separate from syntax.
// The AST in SourceFile remains pure and immutable after parsing.
//
// AUTHORITATIVE DATA SOURCES:
// - Scope (SymbolTable) is the SINGLE source of truth for all symbols in this module
// - Symbol.Type is the SINGLE source of truth for type information
// - No duplication between Scope, AST, or side maps
type SemanticModel struct {
	// File reference
	SourceFile *SourceFile

	// Phase 3 & 4: Collector + Resolver output
	// AUTHORITATIVE symbol table for this module
	// Contains all declarations: variables, functions, types, structs, etc.
	// Parent: UniverseScope (built-ins)
	Scope *semantics.SymbolTable

	// Phase 4: Resolver output
	// Resolved import statements
	Imports []*ImportInfo

	// Module metadata
	ModuleName string // Derived from file path or module declaration

	// Phase completion flags
	Collected bool // Has symbol collection completed?
	Resolved  bool // Has symbol resolution completed?
	Checked   bool // Has type checking completed?
}

// ImportInfo tracks a single import statement and its resolution.
type ImportInfo struct {
	Path         string          // Import path string (e.g., "std/io")
	Alias        string          // Optional alias (e.g., "import io as myio")
	ResolvedFile *SourceFile     // Resolved source file (set during Resolver phase)
	Symbols      []string        // Specific symbols imported (empty = wildcard import)
	ASTNode      *ast.ImportStmt // Back-reference to the AST node
}

// CompilerOptions holds compiler configuration.
// Passed to the context at creation time and remains immutable.
type CompilerOptions struct {
	Debug        bool     // Enable debug output during compilation
	NoStdlib     bool     // Don't automatically import standard library
	OutputPath   string   // Output file path for compiled binary
	IncludePaths []string // Additional paths to search for imports
	Optimization int      // Optimization level (0-3)
	Target       string   // Target platform (future: cross-compilation)
}

// ============================================================================
// CONSTRUCTOR FUNCTIONS
// ============================================================================

// New creates a new compiler context with the given options.
// This is the entry point for starting a new compilation session.
func New(options *CompilerOptions) *CompilerContext {
	if options == nil {
		options = &CompilerOptions{}
	}

	// Create universe scope with built-in types and functions
	universeScope := semantics.NewSymbolTable(nil)
	registerBuiltins(universeScope)

	return &CompilerContext{
		Diagnostics:    diagnostics.NewDiagnosticBag(""),
		Files:          make(map[string]*SourceFile),
		SemanticModels: make(map[string]*SemanticModel),
		UniverseScope:  universeScope,
		Options:        options,
		FileOrder:      make([]string, 0),
	}
}

// registerBuiltins registers built-in types and functions in the universe scope
func registerBuiltins(scope *semantics.SymbolTable) {
	// Built-in types
	builtinTypes := []struct {
		name     string
		typeName types.TYPE_NAME
	}{
		{"i8", types.TYPE_I8},
		{"i16", types.TYPE_I16},
		{"i32", types.TYPE_I32},
		{"i64", types.TYPE_I64},
		{"u8", types.TYPE_U8},
		{"u16", types.TYPE_U16},
		{"u32", types.TYPE_U32},
		{"u64", types.TYPE_U64},
		{"f32", types.TYPE_F32},
		{"f64", types.TYPE_F64},
		{"bool", types.TYPE_BOOL},
		{"string", types.TYPE_STRING},
		{"void", types.TYPE_VOID},
	}

	for _, bt := range builtinTypes {
		sym := &semantics.Symbol{
			Name:     bt.name,
			Kind:     semantics.SymbolType,
			Type:     bt.typeName,
		}
		scope.Declare(bt.name, sym)
	}
}

// ============================================================================
// FILE MANAGEMENT
// ============================================================================

// AddFile registers a new source file in the context.
// This should be called during the initial file discovery phase.
//
// Thread-safe for future parallel compilation.
func (ctx *CompilerContext) AddFile(path string, content string) *SourceFile {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	file := &SourceFile{
		Path:    path,
		Content: content,
	}

	ctx.Files[path] = file
	ctx.FileOrder = append(ctx.FileOrder, path)

	return file
}

// GetFile retrieves a source file by path.
// Returns nil if the file hasn't been registered.
func (ctx *CompilerContext) GetFile(path string) *SourceFile {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.Files[path]
}

// GetAllFiles returns all registered files in the order they were added.
// This ensures deterministic compilation order.
func (ctx *CompilerContext) GetAllFiles() []*SourceFile {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	files := make([]*SourceFile, 0, len(ctx.FileOrder))
	for _, path := range ctx.FileOrder {
		files = append(files, ctx.Files[path])
	}
	return files
}

// ============================================================================
// SEMANTIC MODEL MANAGEMENT
// ============================================================================

// GetOrCreateSemanticModel retrieves or creates a semantic model for a file.
// Called by Collector phase to initialize semantic analysis.
func (ctx *CompilerContext) GetOrCreateSemanticModel(file *SourceFile) *SemanticModel {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	if model, exists := ctx.SemanticModels[file.Path]; exists {
		return model
	}

	model := &SemanticModel{
		SourceFile: file,
		// Scope will be created by Resolver with UniverseScope as parent
	}

	ctx.SemanticModels[file.Path] = model
	return model
}

// GetSemanticModel retrieves a semantic model by file path.
// Returns nil if no semantic analysis has been done for this file yet.
func (ctx *CompilerContext) GetSemanticModel(filePath string) *SemanticModel {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.SemanticModels[filePath]
}

// ============================================================================
// DIAGNOSTICS
// ============================================================================

// HasErrors returns true if any errors have been reported during compilation.
func (ctx *CompilerContext) HasErrors() bool {
	return ctx.Diagnostics.HasErrors()
}

// EmitDiagnostics outputs all collected diagnostics to the console.
// Typically called at the end of compilation.
func (ctx *CompilerContext) EmitDiagnostics() {
	ctx.Diagnostics.EmitAll()
}

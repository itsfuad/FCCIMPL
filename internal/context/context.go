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
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"compiler/internal/diagnostics"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/frontend/parser"
	"compiler/internal/semantics"
	"compiler/internal/types"
)

// CompilationPhase tracks the current phase of compilation
// This is global state - not per-file. All files move through phases together.
type CompilationPhase int

const (
	PhaseInitial      CompilationPhase = iota // Not started
	PhaseLexing                               // Tokenizing source files
	PhaseParsing                              // Building ASTs
	PhaseCollecting                           // Building symbol tables
	PhaseResolving                            // Resolving symbols and imports
	PhaseTypeChecking                         // Validating types
	PhaseCodeGen                              // Generating output code
	PhaseComplete                             // Compilation finished
)

// Phase runner function types - these will be set by the semantic packages
var (
	CollectorRun func(*CompilerContext)
	ResolverRun  func(*CompilerContext)
	CheckerRun   func(*CompilerContext)
)

// DependencyGraph tracks the import relationships between files
// This enables parallel compilation and incremental builds
type DependencyGraph struct {
	// Adjacency list: file path -> list of files it depends on
	Dependencies map[string][]string

	// Reverse dependencies: file path -> list of files that depend on it
	Dependents map[string][]string

	// Track which files have been processed
	Processed map[string]bool

	// Mutex for thread-safe graph operations during parallel discovery
	mu sync.RWMutex
}

// CompilerContext is the central hub for all compilation state.
// This is the ONLY place where global compiler state should live.
//
// Thread safety: All mutations should go through methods that can add locking.
type CompilerContext struct {
	// Diagnostics - centralized error and warning collection
	// All phases report here instead of storing their own errors
	Diagnostics *diagnostics.DiagnosticBag

	// Files - maps absolute file path -> SourceFile
	// This is the single registry of all files in the compilation
	// Each SourceFile contains both syntax (AST) and semantics (Scope, Imports)
	Files map[string]*SourceFile

	// BlockScopes - maps Block AST nodes to their symbol tables
	// Blocks (if/else/for/while/etc) create child scopes for their declarations
	BlockScopes map[interface{}]*semantics.SymbolTable

	// DependencyGraph - tracks import relationships for parallel compilation
	Graph *DependencyGraph

	// UniverseScope - root scope containing built-in types and functions
	// This is the parent of all module scopes
	// Contains: i32, f64, bool, string, print, etc.
	UniverseScope *semantics.SymbolTable

	// CurrentPhase - global compilation phase tracker
	// All files move through phases together for consistency
	CurrentPhase CompilationPhase

	// Options - compiler configuration
	Options *CompilerOptions

	// FileOrder - tracks order files were added (for deterministic builds)
	FileOrder []string

	// Mutex for thread-safe operations during parallel compilation
	mu sync.RWMutex
}

// SourceFile represents a complete source file through all compilation phases.
// This is the SINGLE structure per file - no separate semantic model.
//
// PHASE PROGRESSION:
//
//	Phase 0: Discovery - Path, Content populated
//	Phase 1: Lexer    - Tokens populated
//	Phase 2: Parser   - AST populated
//	Phase 3: Collector- Scope populated (TODO)
//	Phase 4: Resolver - Imports, ModuleName populated (TODO)
//	Phase 5: TypeChecker - Types stored in Scope.Symbols (TODO)
//
// This matches the design of production compilers (Rustc, Zig, Go, TypeScript)
// where semantic data is attached directly to the file, not stored separately.
type SourceFile struct {
	// Phase 0: Discovery
	Path    string // Absolute file path
	Content string // Raw source code

	// Phase 1: Lexer output
	Tokens []lexer.Token

	// Phase 2: Parser output
	AST *ast.Module // Pure syntax tree (no semantic annotations)

	// Phase 3: Collector output (TODO: implement)
	// Symbol table for this module's declarations
	// Parent scope is CompilerContext.UniverseScope
	Scope *semantics.SymbolTable

	// Phase 4: Resolver output (TODO: implement)
	// Resolved import statements
	Imports []*ImportInfo
	// Module name derived from file path or explicit module declaration
	ModuleName string

	// Phase 5: Type Checker output (TODO: implement)
	// Types are stored directly in Scope.Symbols[name].Type
	// No separate type map needed
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
		Diagnostics:   diagnostics.NewDiagnosticBag(""),
		Files:         make(map[string]*SourceFile),
		BlockScopes:   make(map[interface{}]*semantics.SymbolTable),
		UniverseScope: universeScope,
		Graph: &DependencyGraph{
			Dependencies: make(map[string][]string),
			Dependents:   make(map[string][]string),
			Processed:    make(map[string]bool),
		},
		Options:      options,
		FileOrder:    make([]string, 0),
		CurrentPhase: PhaseInitial,
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
			Name: bt.name,
			Kind: semantics.SymbolType,
			Type: bt.typeName,
		}
		scope.Declare(bt.name, sym)
	}

	// Built-in constants
	builtinConstants := []struct {
		name string
		typ  types.TYPE_NAME
	}{
		{"true", types.TYPE_BOOL},
		{"false", types.TYPE_BOOL},
		{"none", types.TYPE_NONE}, // or could be a special option type
	}

	for _, bc := range builtinConstants {
		sym := &semantics.Symbol{
			Name: bc.name,
			Kind: semantics.SymbolConst,
			Type: bc.typ,
		}
		scope.Declare(bc.name, sym)
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
// SEMANTIC INITIALIZATION
// ============================================================================

// InitializeSemantics initializes semantic fields for a SourceFile.
// Call this when transitioning to Phase 3 (Collector).
func (ctx *CompilerContext) InitializeSemantics(file *SourceFile) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	if file.Scope == nil {
		file.Scope = semantics.NewSymbolTable(ctx.UniverseScope)
	}
	if file.Imports == nil {
		file.Imports = make([]*ImportInfo, 0)
	}
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

// ============================================================================
// COMPILATION PIPELINE
// ============================================================================

// Compile runs the complete compilation pipeline on the entry point file.
//
// CURRENT IMPLEMENTATION (Phases 0-2):
//
//	Phase 0: File Discovery - Build dependency graph (parallel)
//	Phase 1: Lexer - Tokenize all files (parallel)
//	Phase 2: Parser - Build ASTs (parallel)
//
// FUTURE PHASES (to be implemented):
//
//	Phase 3: Collector - Build symbol tables
//	Phase 4: Resolver - Resolve all symbols and imports
//	Phase 5: Type Checker - Validate types and expressions
//	Phase 6: Code Generator - Emit target code
//
// All phases operate on the CompilerContext state and report diagnostics
// through ctx.Diagnostics. Returns error only for fatal compilation failures.
func (ctx *CompilerContext) Compile(entryPoint string) error {
	if ctx.Options.Debug {
		fmt.Println("╔════════════════════════════════════════╗")
		fmt.Println("║    FERRET COMPILER - BUILD PIPELINE    ║")
		fmt.Println("╚════════════════════════════════════════╝")
	}

	// Phase 0: File Discovery
	// Recursively discover all source files by following import statements
	// Builds dependency graph for parallel compilation
	if err := ctx.BuildDependencyGraph(entryPoint); err != nil {
		return fmt.Errorf("file discovery failed: %w", err)
	}

	// Phase 1 & 2: Lex + Parse (Combined for efficiency)
	// Tokenize and parse all discovered files in parallel
	if err := ctx.RunLexAndParsePhase(); err != nil {
		return fmt.Errorf("lex/parse failed: %w", err)
	}

	// Phase 3: Collector
	// Walks ASTs and builds symbol tables for each module
	if ctx.Options.Debug {
		fmt.Printf("\n[Phase 3] Declaration Collection\n")
	}
	ctx.RunCollectorPhase()

	// Phase 4: Resolver
	// Resolves all symbol references and import dependencies
	if ctx.Options.Debug {
		fmt.Printf("\n[Phase 4] Type Resolution\n")
	}
	ctx.RunResolverPhase()

	// Phase 5: Type Checker
	// Validates type correctness of all expressions
	if ctx.Options.Debug {
		fmt.Printf("\n[Phase 5] Type Checking\n")
	}
	ctx.RunCheckerPhase()

	// TODO: Phase 6 - Code Generator
	// codegen.Run(ctx)
	// Emits target code (bytecode, LLVM IR, etc.)

	// Check for compilation errors
	if ctx.HasErrors() {
		return fmt.Errorf("compilation failed with errors")
	}

	return nil
}

// BuildDependencyGraph discovers all source files starting from entry point
// Recursively follows import statements and builds the dependency graph
// Uses breadth-first traversal with concurrent file processing
func (ctx *CompilerContext) BuildDependencyGraph(entryPoint string) error {
	if ctx.Options.Debug {
		fmt.Printf("\n[Phase 0] File Discovery (Parallel)\n")
	}

	absPath, err := filepath.Abs(entryPoint)
	if err != nil {
		return fmt.Errorf("failed to resolve path %s: %w", entryPoint, err)
	}

	// Queue for BFS traversal
	toProcess := []string{absPath}
	ctx.Graph.Processed[absPath] = true

	for len(toProcess) > 0 {
		// Process current batch in parallel
		currentBatch := toProcess
		toProcess = nil

		var mu sync.Mutex // Protects toProcess slice
		var wg sync.WaitGroup
		errorChan := make(chan error, len(currentBatch))

		for _, filePath := range currentBatch {
			wg.Add(1)
			go func(path string) {
				defer wg.Done()

				// Discover file and get its imports
				imports, err := ctx.discoverFile(path)
				if err != nil {
					errorChan <- err
					return
				}

				// Add unprocessed imports to next batch
				mu.Lock()
				for _, imp := range imports {
					ctx.Graph.mu.Lock()
					if !ctx.Graph.Processed[imp] {
						ctx.Graph.Processed[imp] = true
						toProcess = append(toProcess, imp)
						ctx.Graph.mu.Unlock()
					} else {
						ctx.Graph.mu.Unlock()
					}
				}
				mu.Unlock()
			}(filePath)
		}

		wg.Wait()
		close(errorChan)

		// Check for errors
		for err := range errorChan {
			if err != nil {
				return err
			}
		}
	}

	if ctx.Options.Debug {
		fmt.Printf("  Discovered %d file(s)\n", len(ctx.Files))
		if len(ctx.Graph.Dependencies) > 0 {
			fmt.Printf("  Dependency edges: %d\n", ctx.countDependencyEdges())
		}
	}

	return nil
}

// countDependencyEdges counts total import edges in the graph
func (ctx *CompilerContext) countDependencyEdges() int {
	count := 0
	for _, deps := range ctx.Graph.Dependencies {
		count += len(deps)
	}
	return count
}

// discoverFile reads a file, registers it, and returns its import dependencies
// Returns list of absolute paths that this file imports
func (ctx *CompilerContext) discoverFile(filePath string) ([]string, error) {
	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Register file in context (thread-safe)
	// Use separate lock to avoid deadlock
	ctx.mu.Lock()
	alreadyExists := false
	if _, exists := ctx.Files[filePath]; !exists {
		// Create file entry without using AddFile (to avoid mutex recursion)
		ctx.Files[filePath] = &SourceFile{
			Path:    filePath,
			Content: string(content),
		}
		ctx.FileOrder = append(ctx.FileOrder, filePath)
		if ctx.Options.Debug {
			fmt.Printf("  %s\n", filepath.Base(filePath))
		}
	} else {
		alreadyExists = true
	}
	ctx.mu.Unlock()

	// If file already processed, return early
	if alreadyExists {
		return nil, nil
	}

	// Quick scan for import statements (lightweight parse)
	imports := ctx.extractImports(string(content), filePath)

	// Register dependencies in graph
	if len(imports) > 0 {
		ctx.Graph.mu.Lock()
		ctx.Graph.Dependencies[filePath] = imports
		for _, importPath := range imports {
			ctx.Graph.Dependents[importPath] = append(ctx.Graph.Dependents[importPath], filePath)
		}
		ctx.Graph.mu.Unlock()
	}

	return imports, nil
}

// extractImports does a lightweight scan for import statements
// Returns list of absolute file paths for imported modules
func (ctx *CompilerContext) extractImports(content, currentFile string) []string {
	// Tokenize just enough to find import statements
	tokenizer := lexer.New(currentFile, content)
	tokens := tokenizer.Tokenize(false) // Don't print tokens

	var imports []string
	i := 0
	for i < len(tokens) {
		// Look for: import "path/to/module"
		if tokens[i].Kind == lexer.IMPORT_TOKEN {
			i++
			// Skip to next token
			if i >= len(tokens) {
				break
			}
			// Expect string literal
			if tokens[i].Kind == lexer.STRING_TOKEN {
				importPath := tokens[i].Value
				// Resolve relative to current file or search paths
				resolvedPath := ctx.resolveImportPath(importPath, currentFile)
				if resolvedPath != "" {
					imports = append(imports, resolvedPath)
				}
			}
		}
		i++
	}

	return imports
}

// resolveImportPath converts an import path to an absolute file path
func (ctx *CompilerContext) resolveImportPath(importPath, currentFile string) string {
	// Remove quotes from import path (lexer includes quotes in string token value)
	if len(importPath) >= 2 && importPath[0] == '"' && importPath[len(importPath)-1] == '"' {
		importPath = importPath[1 : len(importPath)-1]
	}

	// Try relative to current file first
	currentDir := filepath.Dir(currentFile)
	candidatePath := filepath.Join(currentDir, importPath)

	// Add .fer extension if not present
	if filepath.Ext(candidatePath) != ".fer" {
		candidatePath += ".fer"
	}

	// Check if file exists
	if _, err := os.Stat(candidatePath); err == nil {
		absPath, _ := filepath.Abs(candidatePath)
		return absPath
	}

	// Try include paths
	for _, includePath := range ctx.Options.IncludePaths {
		candidatePath = filepath.Join(includePath, importPath)
		if filepath.Ext(candidatePath) != ".fer" {
			candidatePath += ".fer"
		}
		if _, err := os.Stat(candidatePath); err == nil {
			absPath, _ := filepath.Abs(candidatePath)
			return absPath
		}
	}

	// Import not found - will be reported as error later
	return ""
}

// RunLexAndParsePhase tokenizes and parses all files in parallel
func (ctx *CompilerContext) RunLexAndParsePhase() error {
	if ctx.Options.Debug {
		fmt.Printf("\n[Phase 1 & 2] Lex + Parse (Parallel)\n")
	}

	files := ctx.GetAllFiles()
	errorChan := make(chan error, len(files))
	var wg sync.WaitGroup

	// Process each file in parallel
	for _, file := range files {
		wg.Add(1)
		go func(f *SourceFile) {
			defer wg.Done()

			// Lex
			if err := ctx.lexFile(f); err != nil {
				errorChan <- fmt.Errorf("lexer failed on %s: %w", f.Path, err)
				return
			}

			// Parse
			if err := ctx.parseFile(f); err != nil {
				errorChan <- fmt.Errorf("parser failed on %s: %w", f.Path, err)
				return
			}
		}(file)
	}

	// Wait for all to complete
	wg.Wait()
	close(errorChan)

	// Check for errors
	for err := range errorChan {
		if err != nil {
			return err
		}
	}

	if ctx.Options.Debug {
		fmt.Printf("  ✓ Processed %d file(s)\n", len(files))
	}

	return nil
}

// RunCollectorPhase walks ASTs and builds symbol tables (Phase 3)
func (ctx *CompilerContext) RunCollectorPhase() {
	// Initialize semantics for all files
	for _, file := range ctx.GetAllFiles() {
		ctx.InitializeSemantics(file)
	}

	// Run the collector if it's been registered
	if CollectorRun != nil {
		CollectorRun(ctx)
	}

	if ctx.Options.Debug {
		fmt.Printf("  ✓ Collected symbols from %d file(s)\n", len(ctx.GetAllFiles()))
	}
}

// RunResolverPhase resolves all types (Phase 4)
func (ctx *CompilerContext) RunResolverPhase() {
	// Run the type resolver if it's been registered
	if ResolverRun != nil {
		ResolverRun(ctx)
	}

	if ctx.Options.Debug {
		fmt.Printf("  ✓ Resolved types for %d file(s)\n", len(ctx.GetAllFiles()))
	}
}

// RunCheckerPhase validates type correctness (Phase 5)
func (ctx *CompilerContext) RunCheckerPhase() {
	// Run the type checker if it's been registered
	if CheckerRun != nil {
		CheckerRun(ctx)
	}

	if ctx.Options.Debug {
		fmt.Printf("  ✓ Type checked %d file(s)\n", len(ctx.GetAllFiles()))
	}
}

// RunLexerPhase tokenizes all registered source files (sequential fallback)
func (ctx *CompilerContext) RunLexerPhase() error {
	if ctx.Options.Debug {
		fmt.Printf("\n[Phase 1] Lexer\n")
	}

	// Process each file in order
	for _, file := range ctx.GetAllFiles() {
		if err := ctx.lexFile(file); err != nil {
			return fmt.Errorf("lexer failed on %s: %w", file.Path, err)
		}
	}

	if ctx.Options.Debug {
		fmt.Printf("  ✓ Tokenization complete\n")
	}

	return nil
} // lexFile tokenizes a single source file
// This is the lexer phase worker - it's stateless and operates on the context
func (ctx *CompilerContext) lexFile(file *SourceFile) error {
	if ctx.Options.Debug {
		fmt.Printf("  Tokenizing %s (%d bytes)\n", file.Path, len(file.Content))
	}

	// Use the existing tokenizer implementation
	tokenizer := lexer.New(file.Path, file.Content)
	tokens := tokenizer.Tokenize(ctx.Options.Debug)

	// Store tokens in the file
	file.Tokens = tokens

	// Transfer any errors from the tokenizer to the context
	for _, err := range tokenizer.Errors {
		// Create minimal error diagnostic
		ctx.Diagnostics.Add(
			diagnostics.NewError(fmt.Sprintf("Lexer error in %s: %s", file.Path, err.Error())).
				WithCode("L0001"),
		)
	}

	if ctx.Options.Debug {
		fmt.Printf("    Generated %d tokens\n", len(tokens))
	}

	return nil
}

// RunParserPhase parses all tokenized files into ASTs
func (ctx *CompilerContext) RunParserPhase() error {
	if ctx.Options.Debug {
		fmt.Printf("\n[Phase 2] Parser\n")
	}

	// Process each file in order
	for _, file := range ctx.GetAllFiles() {
		if err := ctx.parseFile(file); err != nil {
			return fmt.Errorf("parser failed on %s: %w", file.Path, err)
		}
	}

	if ctx.Options.Debug {
		fmt.Printf("  ✓ Parsing complete\n")
	}

	return nil
}

// parseFile parses a single tokenized file into an AST
// This is the parser phase worker - it's stateless and operates on the context
func (ctx *CompilerContext) parseFile(file *SourceFile) error {
	if ctx.Options.Debug {
		fmt.Printf("  Parsing %s (%d tokens)\n", file.Path, len(file.Tokens))
	}

	// Call the parser with tokens and diagnostics
	file.AST = parser.Parse(file.Tokens, file.Path, ctx.Diagnostics)

	file.AST.SaveAST()

	if ctx.Options.Debug {
		if file.AST != nil {
			fmt.Printf("    Generated %d top-level declarations\n", len(file.AST.Nodes))
		}
	}

	return nil
}

// SetBlockScope associates a scope with a block AST node
func (ctx *CompilerContext) SetBlockScope(block interface{}, scope *semantics.SymbolTable) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.BlockScopes[block] = scope
}

// GetBlockScope retrieves the scope associated with a block AST node
func (ctx *CompilerContext) GetBlockScope(block interface{}) *semantics.SymbolTable {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.BlockScopes[block]
}

// GetAllBlockScopes returns all block scopes
func (ctx *CompilerContext) GetAllBlockScopes() []*semantics.SymbolTable {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	scopes := make([]*semantics.SymbolTable, 0, len(ctx.BlockScopes))
	for _, scope := range ctx.BlockScopes {
		scopes = append(scopes, scope)
	}
	return scopes
}

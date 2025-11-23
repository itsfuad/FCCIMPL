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
// This matches the design of production compilers (Rustc, Zig, Go, TypeScript)
// where semantic data is attached directly to the file, not stored separately.
type SourceFile struct {
	Path    string // Absolute file path
	Content string // Raw source code

	Tokens []lexer.Token
	AST    *ast.Module // Pure syntax tree (no semantic annotations)
	// Parent scope is CompilerContext.UniverseScope
	Scope *semantics.SymbolTable
	// Resolved import statements
	Imports []*ImportInfo
	// Module name derived from file path or explicit module declaration
	ModuleName string
}

// ImportInfo tracks a single import statement and its resolution.
type ImportInfo struct {
	Path         string          // Import path string (e.g., "std/io")
	Alias        string          // Optional alias (e.g., "import "std/io" as myio")
	ResolvedFile *SourceFile     // Resolved source file (set during Resolver phase)
	Symbols      []string        // Specific symbols imported (empty = wildcard import)
	ASTNode      *ast.ImportStmt // Back-reference to the AST node for diagnostics
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

// HasErrors returns true if any errors have been reported during compilation.
func (ctx *CompilerContext) HasErrors() bool {
	return ctx.Diagnostics.HasErrors()
}

// EmitDiagnostics outputs all collected diagnostics to the console.
// Typically called at the end of compilation.
func (ctx *CompilerContext) EmitDiagnostics() {
	ctx.Diagnostics.EmitAll()
}



// BuildDependencyGraph discovers all source files starting from entry point
// Recursively follows import statements and builds the dependency graph
// Uses breadth-first traversal with concurrent file processing
func (ctx *CompilerContext) BuildDependencyGraph(entryPoint string) error {
	if ctx.Options.Debug {
		fmt.Fprintf(os.Stderr, "\n[Phase 0] File Discovery (Parallel)\n")
	}

	absPath, err := filepath.Abs(entryPoint)
	if err != nil {
		return fmt.Errorf("failed to resolve path %s: %w", entryPoint, err)
	}

	// Initialize BFS queue with entry point
	toProcess := []string{absPath}
	ctx.Graph.Processed[absPath] = true

	// Process files level-by-level (breadth-first)
	for len(toProcess) > 0 {
		nextBatch, err := ctx.processBatch(toProcess)
		if err != nil {
			return err
		}
		toProcess = nextBatch
	}

	if ctx.Options.Debug {
		ctx.logDiscoveryStats()
	}

	return nil
}

// processBatch processes a batch of files in parallel and returns newly discovered files
func (ctx *CompilerContext) processBatch(batch []string) ([]string, error) {
	var nextBatch []string
	var mu sync.Mutex
	var wg sync.WaitGroup
	errorChan := make(chan error, len(batch))

	for _, filePath := range batch {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			ctx.processFileInBatch(path, &nextBatch, &mu, errorChan)
		}(filePath)
	}

	wg.Wait()
	close(errorChan)

	// Collect any errors
	for err := range errorChan {
		if err != nil {
			return nil, err
		}
	}

	return nextBatch, nil
}

// processFileInBatch discovers imports for a single file and queues new ones
func (ctx *CompilerContext) processFileInBatch(filePath string, nextBatch *[]string, mu *sync.Mutex, errorChan chan<- error) {
	imports, err := ctx.discoverFile(filePath)
	if err != nil {
		errorChan <- err
		return
	}

	// Queue new imports for processing
	ctx.queueNewImports(imports, nextBatch, mu)
}

// queueNewImports adds unprocessed imports to the next batch
func (ctx *CompilerContext) queueNewImports(imports []string, nextBatch *[]string, mu *sync.Mutex) {
	for _, imp := range imports {
		if ctx.shouldProcessImport(imp) {
			mu.Lock()
			*nextBatch = append(*nextBatch, imp)
			mu.Unlock()
		}
	}
}

// shouldProcessImport checks if an import needs processing
func (ctx *CompilerContext) shouldProcessImport(importPath string) bool {
	ctx.Graph.mu.Lock()
	defer ctx.Graph.mu.Unlock()

	if ctx.Graph.Processed[importPath] {
		return false
	}
	ctx.Graph.Processed[importPath] = true
	return true
}

// logDiscoveryStats prints discovery statistics
func (ctx *CompilerContext) logDiscoveryStats() {
	fmt.Fprintf(os.Stderr, "  Discovered %d file(s)\n", len(ctx.Files))
	if len(ctx.Graph.Dependencies) > 0 {
		fmt.Fprintf(os.Stderr, "  Dependency edges: %d\n", ctx.countDependencyEdges())
	}
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
			fmt.Fprintf(os.Stderr, "  %s\n", filepath.Base(filePath))
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
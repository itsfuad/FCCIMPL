// Package context - compilation pipeline
//
// PIPELINE ARCHITECTURE:
// The pipeline orchestrates compilation phases as a series of transformations.
// Each phase is a stateless worker function that:
//  1. Receives CompilerContext and SourceFile(s)
//  2. Reads from previous phase's output
//  3. Writes to the next phase's input
//  4. Reports errors to ctx.Diagnostics
//
// Phase progression:
//
//	Entry -> [File Discovery] -> Lexer -> Parser -> Collector -> Resolver -> TypeChecker -> CodeGen -> Exit
//
// This design:
//   - Makes each phase independently testable
//   - Enables incremental compilation (skip unchanged files)
//   - Supports parallel processing (future: goroutines)
//   - Provides clear separation of concerns
package context

import (
	"compiler/internal/diagnostics"
	"compiler/internal/frontend/lexer"
	"compiler/internal/frontend/parser"
	"fmt"
	"os"
	"path/filepath"
)

// CompilationPhase represents a phase in the compilation pipeline
type CompilationPhase int

const (
	PhaseLexer CompilationPhase = iota
	PhaseParser
	PhaseCollector
	PhaseResolver
	PhaseTypeCheck
	PhaseCodeGen
)

// Pipeline manages the compilation pipeline
type Pipeline struct {
	Context *CompilerContext
}

// NewPipeline creates a new compilation pipeline with the given options
func NewPipeline(options *CompilerOptions) *Pipeline {
	return &Pipeline{
		Context: New(options),
	}
}

// Compile runs the compilation pipeline on the entry point file.
// Implements: Lexer -> Parser
// Collector, Resolver, TypeChecker are called externally to avoid import cycles
//
// Returns: error if compilation failed
func (p *Pipeline) Compile(entryPoint string) error {
	if p.Context.Options.Debug {
		fmt.Println("╔════════════════════════════════════════╗")
		fmt.Println("║    FERRET COMPILER - BUILD PIPELINE    ║")
		fmt.Println("╚════════════════════════════════════════╝")
	}

	// Phase 0: File Discovery - Register entry point
	if err := p.addNewFile(entryPoint); err != nil {
		return err
	}

	// Phase 1: Lexer - Tokenize all files
	if err := p.runLexerPhase(); err != nil {
		return err
	}

	// Phase 2: Parser - Build ASTs
	if err := p.runParserPhase(); err != nil {
		return err
	}

	// NOTE: Phases 3-5 (Collector, Resolver, TypeChecker) are called
	// from main.go to avoid import cycles

	if p.Context.HasErrors() {
		return fmt.Errorf("compilation failed with errors")
	}

	return nil
}

// addNewFile registers the entry point file in the context
// Future: This will recursively discover imported files
func (p *Pipeline) addNewFile(filePath string) error {
	if p.Context.Options.Debug {
		fmt.Printf("\n[Phase 0] File Discovery\n")
	}

	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Register in context
	absPath, _ := filepath.Abs(filePath)
	p.Context.AddFile(absPath, string(content))

	if p.Context.Options.Debug {
		fmt.Printf("  Registered: %s (%d bytes)\n", absPath, len(content))
	}

	return nil
}

// runLexerPhase tokenizes all registered source files
func (p *Pipeline) runLexerPhase() error {
	if p.Context.Options.Debug {
		fmt.Printf("\n[Phase 1] Lexer\n")
	}

	// Process each file in order
	for _, file := range p.Context.GetAllFiles() {
		if err := p.lexFile(file); err != nil {
			return fmt.Errorf("lexer failed on %s: %w", file.Path, err)
		}
	}

	if p.Context.Options.Debug {
		fmt.Printf("  ✓ Tokenization complete\n")
	}

	return nil
}

// lexFile tokenizes a single source file
// This is the lexer phase worker - it's stateless and operates on the context
func (p *Pipeline) lexFile(file *SourceFile) error {
	if p.Context.Options.Debug {
		fmt.Printf("  Tokenizing %s (%d bytes)\n", file.Path, len(file.Content))
	}

	// Use the existing tokenizer implementation
	tokenizer := lexer.New(file.Path, file.Content)
	tokens := tokenizer.Tokenize(p.Context.Options.Debug)

	// Store tokens in the file
	file.Tokens = tokens

	// Transfer any errors from the tokenizer to the context
	for _, err := range tokenizer.Errors {
		// Create minimal error diagnostic
		p.Context.Diagnostics.Add(
			diagnostics.NewError(fmt.Sprintf("Lexer error in %s: %s", file.Path, err.Error())).
				WithCode("L0001"),
		)
	}

	if p.Context.Options.Debug {
		fmt.Printf("    Generated %d tokens\n", len(tokens))
	}

	return nil
}

// runParserPhase parses all tokenized files into ASTs
func (p *Pipeline) runParserPhase() error {
	if p.Context.Options.Debug {
		fmt.Printf("\n[Phase 2] Parser\n")
	}

	// Process each file in order
	for _, file := range p.Context.GetAllFiles() {
		if err := p.parseFile(file); err != nil {
			return fmt.Errorf("parser failed on %s: %w", file.Path, err)
		}
	}

	if p.Context.Options.Debug {
		fmt.Printf("  ✓ Parsing complete\n")
	}

	return nil
}

// parseFile parses a single tokenized file into an AST
// This is the parser phase worker - it's stateless and operates on the context
func (p *Pipeline) parseFile(file *SourceFile) error {
	if p.Context.Options.Debug {
		fmt.Printf("  Parsing %s (%d tokens)\n", file.Path, len(file.Tokens))
	}

	// Call the parser with tokens and diagnostics
	file.AST = parser.Parse(file.Tokens, file.Path, p.Context.Diagnostics)

	file.AST.SaveAST()

	if p.Context.Options.Debug {
		if file.AST != nil {
			fmt.Printf("    Generated %d top-level declarations\n", len(file.AST.Nodes))
		}
	}

	return nil
}

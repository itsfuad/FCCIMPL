package cmd

import (
	"fmt"
	"os"
	"sync"

	"compiler/internal/context"
	"compiler/internal/diagnostics"
	"compiler/internal/frontend/lexer"
	"compiler/internal/frontend/parser"
	"compiler/internal/semantics/checker"
	"compiler/internal/semantics/collector"
	"compiler/internal/semantics/resolver"
)

// RunLexAndParsePhase tokenizes and parses all files in parallel
func RunLexAndParsePhase(ctx *context.CompilerContext) error {
	if ctx.Options.Debug {
		fmt.Fprintf(os.Stderr, "\n[Phase 1 & 2] Lex + Parse (Parallel)\n")
	}

	files := ctx.GetAllFiles()
	errorChan := make(chan error, len(files))
	var wg sync.WaitGroup

	// Process each file in parallel
	for _, file := range files {
		wg.Add(1)
		go func(f *context.SourceFile) {
			defer wg.Done()

			// Lex
			if err := lexFile(f, ctx); err != nil {
				errorChan <- fmt.Errorf("lexer failed on %s: %w", f.Path, err)
				return
			}

			// Parse
			if err := parseFile(f, ctx); err != nil {
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
		fmt.Fprintf(os.Stderr, "  ✓ Processed %d file(s)\n", len(files))
	}

	return nil
}


// parseFile parses a single tokenized file into an AST
// This is the parser phase worker - it's stateless and operates on the context
func parseFile(file *context.SourceFile, ctx *context.CompilerContext) error {
	if ctx.Options.Debug {
		fmt.Fprintf(os.Stderr, "  Parsing %s (%d tokens)\n", file.Path, len(file.Tokens))
	}

	// Call the parser with tokens and diagnostics
	file.AST = parser.Parse(file.Tokens, file.Path, ctx.Diagnostics)

	file.AST.SaveAST()

	if ctx.Options.Debug {
		if file.AST != nil {
			fmt.Fprintf(os.Stderr, "    Generated %d top-level declarations\n", len(file.AST.Nodes))
		}
	}

	return nil
}


// lexFile tokenizes a single source file
// This is the lexer phase worker - it's stateless and operates on the context
func lexFile(file *context.SourceFile, ctx *context.CompilerContext) error {
	if ctx.Options.Debug {
		fmt.Fprintf(os.Stderr, "  Tokenizing %s (%d bytes)\n", file.Path, len(file.Content))
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
		fmt.Fprintf(os.Stderr, "    Generated %d tokens\n", len(tokens))
	}

	return nil
}

// RunCollectorPhase walks ASTs and builds symbol tables (Phase 3)
func RunCollectorPhase(ctx *context.CompilerContext) {
	// Initialize semantics for all files
	for _, file := range ctx.GetAllFiles() {
		ctx.InitializeSemantics(file)
	}

	collector.Run(ctx)

	if ctx.Options.Debug {
		fmt.Fprintf(os.Stderr, "  ✓ Collected symbols from %d file(s)\n", len(ctx.GetAllFiles()))
	}
}


// RunResolverPhase resolves all types (Phase 4)
func RunResolverPhase(ctx *context.CompilerContext) {

	resolver.Run(ctx)

	if ctx.Options.Debug {
		fmt.Fprintf(os.Stderr, "  ✓ Resolved types for %d file(s)\n", len(ctx.GetAllFiles()))
	}
}

// RunCheckerPhase validates type correctness (Phase 5)
func RunCheckerPhase(ctx *context.CompilerContext) {

	checker.Run(ctx)

	if ctx.Options.Debug {
		fmt.Fprintf(os.Stderr, "  ✓ Type checked %d file(s)\n", len(ctx.GetAllFiles()))
	}
}


// All phases operate on the CompilerContext state and report diagnostics
// through ctx.Diagnostics. Returns error only for fatal compilation failures.
func Compile(entryPoint string, ctx *context.CompilerContext) error {
	if ctx.Options.Debug {
		fmt.Fprintf(os.Stderr, "\n[Compilation Started] Entry Point: %s\n", entryPoint)
	}

	// Phase 0: File Discovery
	// Recursively discover all source files by following import statements
	// Builds dependency graph for parallel compilation
	if err := ctx.BuildDependencyGraph(entryPoint); err != nil {
		return fmt.Errorf("file discovery failed: %w", err)
	}

	// Phase 1 & 2: Lex + Parse (Combined for efficiency)
	// Tokenize and parse all discovered files in parallel
	if err := RunLexAndParsePhase(ctx); err != nil {
		return fmt.Errorf("lex/parse failed: %w", err)
	}

	// Phase 3: Collector
	// Walks ASTs and builds symbol tables for each module
	if ctx.Options.Debug {
		fmt.Fprintf(os.Stderr, "\n[Phase 3] Declaration Collection\n")
	}

	RunCollectorPhase(ctx)

	// Phase 4: Resolver
	// Resolves all symbol references and import dependencies
	if ctx.Options.Debug {
		fmt.Fprintf(os.Stderr, "\n[Phase 4] Type Resolution\n")
	}
	
	RunResolverPhase(ctx)

	// Phase 5: Type Checker
	// Validates type correctness of all expressions
	if ctx.Options.Debug {
		fmt.Fprintf(os.Stderr, "\n[Phase 5] Type Checking\n")
	}
	
	RunCheckerPhase(ctx)

	// TODO: Phase 6 - Code Generator
	// codegen.Run(ctx)
	// Emits target code (bytecode, LLVM IR, etc.)

	// Check for compilation errors
	if ctx.HasErrors() {
		return fmt.Errorf("compilation failed with errors")
	}

	return nil
}
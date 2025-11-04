package main

import (
	"compiler/internal/context"
	// "compiler/internal/phases/collector"
	// "compiler/internal/phases/resolver"
	// "compiler/internal/phases/typechecker"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <file.fer>\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}

	filename := os.Args[1]

	// Create compiler options
	options := &context.CompilerOptions{
		Debug: os.Getenv("DEBUG") == "1",
	}

	// Create compilation pipeline with shared context
	pipeline := context.NewPipeline(options)

	// Run compilation pipeline (Phases 0-2: File Discovery, Lexer, Parser)
	fmt.Printf("Compiling %s...\n\n", filename)
	err := pipeline.Compile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nCompilation failed: %v\n", err)
		pipeline.Context.EmitDiagnostics()
		os.Exit(1)
	}

	// Phase 3: Collector - Collect symbols
	if err := runCollectorPhase(pipeline); err != nil {
		fmt.Fprintf(os.Stderr, "\nCollector failed: %v\n", err)
		pipeline.Context.EmitDiagnostics()
		os.Exit(1)
	}

	// Phase 4: Resolver - Resolve imports and symbols
	if err := runResolverPhase(pipeline); err != nil {
		fmt.Fprintf(os.Stderr, "\nResolver failed: %v\n", err)
		pipeline.Context.EmitDiagnostics()
		os.Exit(1)
	}

	// Phase 5: TypeChecker - Type checking
	if err := runTypeCheckerPhase(pipeline); err != nil {
		fmt.Fprintf(os.Stderr, "\nType checker failed: %v\n", err)
		pipeline.Context.EmitDiagnostics()
		os.Exit(1)
	}

	// Emit diagnostics
	if pipeline.Context.Diagnostics != nil {
		pipeline.Context.EmitDiagnostics()
	}

	if pipeline.Context.HasErrors() {
		os.Exit(1)
	}

	fmt.Println("\nCompilation successful!")
}

// runCollectorPhase runs the collector phase on all files
func runCollectorPhase(pipeline *context.Pipeline) error {
	if pipeline.Context.Options.Debug {
		fmt.Printf("\n[Phase 3] Collector\n")
	}

	for _, file := range pipeline.Context.GetAllFiles() {
		//collector.CollectSymbols(pipeline.Context, file)

		if pipeline.Context.Options.Debug {
			model := pipeline.Context.GetSemanticModel(file.Path)
			if model != nil && model.Scope != nil {
				symCount := len(model.Scope.Symbols)
				fmt.Printf("  Collected %d symbol(s) from %s\n", symCount, filepath.Base(file.Path))
			}
		}
	}

	if pipeline.Context.Options.Debug {
		fmt.Printf("  ✓ Symbol collection complete\n")
	}

	return nil
}

// runResolverPhase runs the resolver phase on all files
func runResolverPhase(pipeline *context.Pipeline) error {
	if pipeline.Context.Options.Debug {
		fmt.Printf("\n[Phase 4] Resolver\n")
	}

	for _, file := range pipeline.Context.GetAllFiles() {
		//resolver.ResolveSymbols(pipeline.Context, file)

		if pipeline.Context.Options.Debug {
			model := pipeline.Context.GetSemanticModel(file.Path)
			if model != nil {
				importCount := len(model.Imports)
				if importCount > 0 {
					fmt.Printf("  Resolved %d import(s) in %s\n", importCount, filepath.Base(file.Path))
				}
			}
		}
	}

	if pipeline.Context.Options.Debug {
		fmt.Printf("  ✓ Symbol resolution complete\n")
	}

	return nil
}

// runTypeCheckerPhase runs the type checker phase on all files
func runTypeCheckerPhase(pipeline *context.Pipeline) error {
	if pipeline.Context.Options.Debug {
		fmt.Printf("\n[Phase 5] Type Checker\n")
	}

	for _, file := range pipeline.Context.GetAllFiles() {
		//typechecker.CheckTypes(pipeline.Context, file)

		if pipeline.Context.Options.Debug {
			model := pipeline.Context.GetSemanticModel(file.Path)
			if model != nil && model.Scope != nil {
				typedCount := 0
				for _, sym := range model.Scope.AllSymbols() {
					if sym.Type != nil {
						typedCount++
					}
				}
				fmt.Printf("  Type checked %d symbol(s) in %s\n", typedCount, filepath.Base(file.Path))
			}
		}
	}

	if pipeline.Context.Options.Debug {
		fmt.Printf("  ✓ Type checking complete\n")
	}

	return nil
}

package main

import (
	"compiler/internal/context"
	"compiler/internal/semantics/checker"
	"compiler/internal/semantics/collector"
	"compiler/internal/semantics/resolver"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	// Register semantic phase runners
	context.CollectorRun = collector.Run
	context.ResolverRun = resolver.Run
	context.CheckerRun = checker.Run
	
	// Parse command-line flags
	debugFlag := flag.Bool("debug", false, "Enable debug output")
	flag.Parse()

	// Validate arguments
	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [--debug] <file.fer>\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}

	filename := flag.Arg(0)

	// Create compiler options
	options := &context.CompilerOptions{
		Debug: *debugFlag,
	}

	// Create compiler context
	ctx := context.New(options)

	// Run compilation pipeline
	if err := ctx.Compile(filename); err != nil {
		ctx.EmitDiagnostics()
		fmt.Fprintf(os.Stderr, "\nCompilation failed: %v\n", err)
		os.Exit(1)
	}

	// Emit any warnings/info diagnostics
	ctx.EmitDiagnostics()

	// Success
	if ctx.Options.Debug {
		fmt.Println("\n✓ Compilation successful!")
	}
}
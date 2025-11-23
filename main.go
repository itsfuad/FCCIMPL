//go:build !js && !wasm

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"compiler/internal/cmd"
	"compiler/internal/context"
)

func main() {

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
	if err := cmd.Compile(filename, ctx); err != nil {
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

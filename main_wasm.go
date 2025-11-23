//go:build js && wasm

package main

import (
	"compiler/internal/context"
	"compiler/internal/semantics/checker"
	"compiler/internal/semantics/collector"
	"compiler/internal/semantics/resolver"
	"fmt"
	"strings"
	"syscall/js"
)

// compileCode compiles Ferret code and returns the result
func compileCode(code string, debug bool) (string, error) {
	// Log entry to browser console FIRST
	jsConsole := js.Global().Get("console")

	// Defer panic recovery
	defer func() {
		if r := recover(); r != nil {
			jsConsole.Call("error", "💥 PANIC in compileCode:", r)
		}
	}()

	// Register semantic phase runners
	context.CollectorRun = collector.Run
	context.ResolverRun = resolver.Run
	context.CheckerRun = checker.Run

	// Create compiler options
	options := &context.CompilerOptions{
		Debug: debug,
	}

	// Create compiler context
	ctx := context.New(options)

	// WASM WORKAROUND: Directly add the code as a "virtual file"
	// instead of using the file system
	virtualFilePath := "main.fer"
	file := ctx.AddFile(virtualFilePath, code)

	// Manually run the compilation phases without file system I/O
	// Phase 1: Lex
	if err := ctx.LexFile(file); err != nil {
		return "", fmt.Errorf("lexer failed: %v", err)
	}

	// Phase 2: Parse
	if err := ctx.ParseFile(file); err != nil {
		return "", fmt.Errorf("parser failed: %v", err)
	}

	// Phase 3: Collector
	ctx.InitializeSemantics(file)
	if context.CollectorRun != nil {
		context.CollectorRun(ctx)
	}

	// Phase 4: Resolver
	if context.ResolverRun != nil {
		context.ResolverRun(ctx)
	}

	// Phase 5: Checker
	if context.CheckerRun != nil {
		context.CheckerRun(ctx)
	}

	// Check for errors
	var err error
	if ctx.HasErrors() {
		err = fmt.Errorf("compilation failed with errors")
	}

	// Split code into lines for source cache
	sourceLines := strings.Split(code, "\n")

	// Get diagnostics output as HTML string
	output := ctx.Diagnostics.EmitAllToHTMLWithCache(sourceLines)

	// Return errors if any
	if err != nil {
		return output, err
	}

	// Success case
	if output == "" {
		if debug {
			output = "✓ Compilation successful! No diagnostics."
		} else {
			output = "✓ Compilation successful!"
		}
	} else if debug {
		output += "\n✓ Compilation completed with diagnostics shown above."
	}

	return output, nil
}

// ferretCompileJS is the JavaScript-callable function
func ferretCompileJS(this js.Value, args []js.Value) interface{} {
	// Defer panic recovery
	defer func() {
		if r := recover(); r != nil {
			jsConsole := js.Global().Get("console")
			jsConsole.Call("error", "💥 PANIC in compiler:", r)
		}
	}()

	// Check arguments
	if len(args) < 1 {
		return map[string]interface{}{
			"success": false,
			"error":   "Expected at least 1 argument (code string)",
		}
	}

	code := args[0].String()
	debug := false
	if len(args) > 1 {
		debug = args[1].Bool()
	}

	// Compile the code
	output, err := compileCode(code, debug)

	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   output,
		}
	}

	return map[string]interface{}{
		"success": true,
		"output":  output,
	}
}

func main() {
	// Prevent the program from exiting
	c := make(chan struct{})

	// Register JavaScript function
	js.Global().Set("ferretCompile", js.FuncOf(ferretCompileJS))

	// Set version info that JavaScript can check
	js.Global().Set("ferretWasmVersion", "v0.0.2-production")

	// Log ready message
	fmt.Println("✅ Ferret WASM Compiler Ready")

	// Keep the program running
	<-c
}

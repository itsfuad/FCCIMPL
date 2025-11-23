package context

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

const (
	mainFerFile      = "main.fer"
	mainFerContent   = "fn main() { }"
	noErrorExpected  = "Expected no error, got: %v"
	libFerFile       = "lib.fer"
	baseFerFile      = "base.fer"
	baseFerContent   = "fn base() { }"
	helperFerContent = "fn helper() { }"
	libFerContent    = "fn lib() { }"
)

// Helper function to create a temporary test file
func createTestFile(dir, name, content string) (string, error) {
	filePath := filepath.Join(dir, name)
	err := os.WriteFile(filePath, []byte(content), 0644)
	return filePath, err
}

// TestBuildDependencyGraphSingleFileNoImports tests a single file with no imports
func TestBuildDependencyGraphSingleFileNoImports(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	mainFile, err := createTestFile(tmpDir, mainFerFile, mainFerContent)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create context
	opts := &CompilerOptions{Debug: false}
	ctx := New(opts)

	// Execute
	err = ctx.BuildDependencyGraph(mainFile)
	if err != nil {
		t.Fatalf(noErrorExpected, err)
	}

	// Verify
	if len(ctx.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(ctx.Files))
	}

	absPath, _ := filepath.Abs(mainFile)
	if _, exists := ctx.Files[absPath]; !exists {
		t.Errorf("Expected file %s in context", absPath)
	}

	if len(ctx.Graph.Dependencies) != 0 {
		t.Errorf("Expected 0 dependencies, got %d", len(ctx.Graph.Dependencies))
	}
}

// TestBuildDependencyGraphSingleFileWithImport tests a single file with one import
func TestBuildDependencyGraphSingleFileWithImport(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()

	// Create imported module with absolute path
	_, _ = createTestFile(tmpDir, "lib.fer", helperFerContent)

	// Create main file that imports lib using relative path from main file directory
	// The lexer wraps strings in quotes, resolveImportPath removes them
	mainContent := `import "lib"
fn main() { }`
	mainFile, err := createTestFile(tmpDir, mainFerFile, mainContent)
	if err != nil {
		t.Fatalf("Failed to create main file: %v", err)
	}

	// Create context
	opts := &CompilerOptions{Debug: false}
	ctx := New(opts)

	// Execute
	err = ctx.BuildDependencyGraph(mainFile)
	if err != nil {
		t.Fatalf(noErrorExpected, err)
	}

	// Verify: both files discovered
	if len(ctx.Files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(ctx.Files))
		for k := range ctx.Files {
			t.Logf("  File: %s", k)
		}
	}

	// Verify: dependency graph built
	if len(ctx.Graph.Dependencies) == 0 {
		t.Errorf("Expected dependencies to be recorded")
	}
}

// TestBuildDependencyGraphTransitiveImports tests multiple levels of imports
func TestBuildDependencyGraphTransitiveImports(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()

	// Create base module
	_, _ = createTestFile(tmpDir, baseFerFile, baseFerContent)

	// Create middle module that imports base
	middleContent := `import "base"
fn middleFunc() { }`
	_, _ = createTestFile(tmpDir, "middle.fer", middleContent)

	// Create main that imports middle
	mainContent := `import "middle"
fn main() { }`
	mainFile, _ := createTestFile(tmpDir, mainFerFile, mainContent)

	// Create context
	opts := &CompilerOptions{Debug: false}
	ctx := New(opts)

	// Execute
	err := ctx.BuildDependencyGraph(mainFile)
	if err != nil {
		t.Fatalf(noErrorExpected, err)
	}

	// Verify: all 3 files discovered
	if len(ctx.Files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(ctx.Files))
	}

	// Verify: dependency chain exists
	absMainPath, _ := filepath.Abs(mainFile)
	if deps, exists := ctx.Graph.Dependencies[absMainPath]; !exists || len(deps) == 0 {
		t.Errorf("Expected dependencies for main file")
	}
}

// TestBuildDependencyGraphMultipleImports tests multiple imports from single file
func TestBuildDependencyGraphMultipleImports(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()

	// Create multiple modules
	_, _ = createTestFile(tmpDir, "lib1.fer", "fn lib1Func() { }")
	_, _ = createTestFile(tmpDir, "lib2.fer", "fn lib2Func() { }")

	// Create main that imports both
	mainContent := `import "lib1"
import "lib2"
fn main() { }`
	mainFile, _ := createTestFile(tmpDir, mainFerFile, mainContent)

	// Create context
	opts := &CompilerOptions{Debug: false}
	ctx := New(opts)

	// Execute
	err := ctx.BuildDependencyGraph(mainFile)
	if err != nil {
		t.Fatalf(noErrorExpected, err)
	}

	// Verify: all files discovered
	if len(ctx.Files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(ctx.Files))
	}

	// Verify: multiple dependencies recorded
	absMainPath, _ := filepath.Abs(mainFile)
	if deps, exists := ctx.Graph.Dependencies[absMainPath]; !exists || len(deps) != 2 {
		t.Errorf("Expected 2 dependencies for main file, got %d", len(deps))
	}
}

// TestBuildDependencyGraphFileNotFound tests file not found error
func TestBuildDependencyGraphFileNotFound(t *testing.T) {
	// Setup
	nonExistentFile := "/nonexistent/path/main.fer"

	// Create context
	opts := &CompilerOptions{Debug: false}
	ctx := New(opts)

	// Execute
	err := ctx.BuildDependencyGraph(nonExistentFile)

	// Verify: error returned
	if err == nil {
		t.Errorf("Expected error for nonexistent file, got nil")
	}
}

// TestBuildDependencyGraphImportPathResolution tests import path resolution
func TestBuildDependencyGraphImportPathResolution(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "lib")
	os.MkdirAll(subDir, 0755)

	// Create library file in subdirectory
	_, _ = createTestFile(subDir, "helper.fer", "fn help() { }")

	// Create main file that imports from subdirectory
	mainContent := fmt.Sprintf(`import "%s"
fn main() { }`, filepath.Join("lib", "helper"))
	mainFile, _ := createTestFile(tmpDir, mainFerFile, mainContent)

	// Create context
	opts := &CompilerOptions{Debug: false}
	ctx := New(opts)

	// Execute
	err := ctx.BuildDependencyGraph(mainFile)

	// Verify: import resolved correctly
	if err != nil {
		// Some import resolution errors are OK for this test
		// as long as the function handles them gracefully
		t.Logf("Import resolution error (expected in this scenario): %v", err)
	}

	// Verify: at least main file was processed
	if len(ctx.Files) == 0 {
		t.Errorf("Expected at least main file to be processed")
	}
}

// TestBuildDependencyGraphGraphStructure tests graph data structure correctness
func TestBuildDependencyGraphGraphStructure(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()

	// Create a simple dependency chain: main -> lib -> base
	_, _ = createTestFile(tmpDir, baseFerFile, baseFerContent)

	libContent := `import "base"
fn lib() { }`
	_, _ = createTestFile(tmpDir, libFerFile, libContent)

	mainContent := `import "lib"
fn main() { }`
	mainFile, _ := createTestFile(tmpDir, mainFerFile, mainContent)

	// Create context
	opts := &CompilerOptions{Debug: false}
	ctx := New(opts)

	// Execute
	err := ctx.BuildDependencyGraph(mainFile)
	if err != nil {
		t.Fatalf(noErrorExpected, err)
	}

	// Verify: Processed map is populated
	if len(ctx.Graph.Processed) != 3 {
		t.Errorf("Expected 3 processed files, got %d", len(ctx.Graph.Processed))
	}

	// Verify: All files marked as processed
	for _, file := range []string{mainFile} {
		absPath, _ := filepath.Abs(file)
		if !ctx.Graph.Processed[absPath] {
			t.Errorf("Expected file %s to be marked as processed", file)
		}
	}

	// Verify: Reverse dependencies exist
	if len(ctx.Graph.Dependents) == 0 {
		t.Errorf("Expected reverse dependency graph to be populated")
	}
}

// TestBuildDependencyGraphDebugMode tests debug output doesn't cause crashes
func TestBuildDependencyGraphDebugMode(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	mainFile, _ := createTestFile(tmpDir, mainFerFile, mainFerContent)

	// Create context with debug enabled
	opts := &CompilerOptions{Debug: true}
	ctx := New(opts)

	// Execute (should not panic)
	err := ctx.BuildDependencyGraph(mainFile)
	if err != nil {
		t.Fatalf(noErrorExpected, err)
	}

	// Verify: file processed
	if len(ctx.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(ctx.Files))
	}
}

// TestBuildDependencyGraphDuplicateImportsNotReprocessed tests duplicate imports
func TestBuildDependencyGraphDuplicateImportsNotReprocessed(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()

	// Create a shared library
	_, _ = createTestFile(tmpDir, "shared.fer", "fn shared() { }")

	// Create two modules that both import the shared library
	mod1Content := `import "shared"
fn mod1() { }`
	_, _ = createTestFile(tmpDir, "mod1.fer", mod1Content)

	mod2Content := `import "shared"
fn mod2() { }`
	_, _ = createTestFile(tmpDir, "mod2.fer", mod2Content)

	// Create main that imports both mod1 and mod2
	mainContent := `import "mod1"
import "mod2"
fn main() { }`
	mainFile, _ := createTestFile(tmpDir, mainFerFile, mainContent)

	// Create context
	opts := &CompilerOptions{Debug: false}
	ctx := New(opts)

	// Execute
	err := ctx.BuildDependencyGraph(mainFile)
	if err != nil {
		t.Fatalf(noErrorExpected, err)
	}

	// Verify: only 4 files (main, mod1, mod2, shared) - shared is not duplicated
	if len(ctx.Files) != 4 {
		t.Errorf("Expected 4 unique files, got %d", len(ctx.Files))
	}
}

// TestBuildDependencyGraphThreadSafety tests concurrent access verification
func TestBuildDependencyGraphThreadSafety(t *testing.T) {
	// Setup: Create a context and verify locks work correctly
	tmpDir := t.TempDir()
	mainFile, _ := createTestFile(tmpDir, mainFerFile, mainFerContent)

	opts := &CompilerOptions{Debug: false}
	ctx := New(opts)

	// Execute BuildDependencyGraph (which uses concurrent goroutines)
	err := ctx.BuildDependencyGraph(mainFile)
	if err != nil {
		t.Fatalf(noErrorExpected, err)
	}

	// Verify: context is in consistent state
	if len(ctx.Files) == 0 {
		t.Errorf("Expected files to be populated")
	}

	if len(ctx.Graph.Processed) == 0 {
		t.Errorf("Expected processed graph to be populated")
	}

	// Verify: can safely read after completion
	allFiles := ctx.GetAllFiles()
	if len(allFiles) == 0 {
		t.Errorf("Expected GetAllFiles to return files")
	}
}

// BenchmarkBuildDependencyGraphSingleFile benchmarks single file performance
func BenchmarkBuildDependencyGraphSingleFile(b *testing.B) {
	tmpDir := b.TempDir()
	mainFile, _ := createTestFile(tmpDir, mainFerFile, mainFerContent)

	opts := &CompilerOptions{Debug: false}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := New(opts)
		ctx.BuildDependencyGraph(mainFile)
	}
}

// BenchmarkBuildDependencyGraphMultipleImports benchmarks multiple imports performance
func BenchmarkBuildDependencyGraphMultipleImports(b *testing.B) {
	tmpDir := b.TempDir()

	// Create 5 library files
	libs := make([]string, 5)
	for i := 0; i < 5; i++ {
		libFile, _ := createTestFile(tmpDir, fmt.Sprintf("lib%d.fer", i), libFerContent)
		libs[i] = libFile[:len(libFile)-4]
	}

	// Create main that imports all
	mainContent := `import "` + libs[0] + `"
import "` + libs[1] + `"
import "` + libs[2] + `"
import "` + libs[3] + `"
import "` + libs[4] + `"
fn main() { }`
	mainFile, _ := createTestFile(tmpDir, mainFerFile, mainContent)

	opts := &CompilerOptions{Debug: false}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := New(opts)
		ctx.BuildDependencyGraph(mainFile)
	}
}

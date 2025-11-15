# Compiler Development Report

Date: 2025-11-15

## Part 1: Parser Refactor

### Summary
Refactored `internal/frontend/parser/decl.go` to extract `parseDeclItem(isConst bool) ast.DeclItem` from `parseVariableDeclaration`, reducing function complexity and addressing a SonarQube lint concern about cognitive complexity.

### What I checked
- Duplicate/conflicting definitions:
  - Searched for `parseDeclItem`, `parseVariableDeclaration`, `parseVarDecl`, `parseConstDecl` across the repo.
  - Results: All symbols are defined only in `internal/frontend/parser/decl.go` and referenced from `internal/frontend/parser/parser.go`. No duplicates or conflicting definitions were found.

- Build status:
  - `go build ./...` completed with no errors.

- Tests:
  - Ran `go test ./...` — packages with tests passed. Several packages have no test files, which is expected.

### Files changed (Part 1)
- `internal/frontend/parser/decl.go`
  - Added `parseDeclItem(isConst bool) ast.DeclItem` helper.
  - Simplified `parseVariableDeclaration` to call the helper for each item.

---

## Part 2: Semantic Analysis Implementation (3-Pass Architecture)

### Architecture Decision: 3-Pass System

**Rationale:**
- **Pass 1 (Collector)**: Gathers all declarations into symbol tables without resolving types. Enables forward references.
- **Pass 2 (Resolver)**: Resolves AST type nodes to semantic types. Handles user-defined types and circular dependencies.
- **Pass 3 (Checker)**: Validates type compatibility for assignments, function calls, and operations.

This approach matches proven compiler architectures (Rust, TypeScript, Go) and cleanly separates concerns.

### Implementation Details

#### Pass 1: Declaration Collector (`internal/semantics/collector/`)
- Walks AST nodes and creates `Symbol` entries in `SymbolTable`
- Handles:
  - Variable declarations (`let`)
  - Constant declarations (`const`)
  - Type declarations (`type`)
  - Function declarations (`fn`)
  - Method declarations
  - Parameters and local scopes
- Detects redeclaration errors
- Creates scope hierarchy (global → function → block)

#### Pass 2: Type Resolver (`internal/semantics/resolver/`)
- Converts AST type nodes to semantic types
- Resolves:
  - Primitive types (i32, f64, string, bool, etc.)
  - User-defined types and aliases
  - Array types
  - Struct types
  - Function types
  - Interface types
  - Optional types (`T?`)
  - Error types (`T ! E`)
- Handles type inference for `:=` declarations

#### Pass 3: Type Checker (`internal/semantics/checker/`)
- Validates type compatibility
- Checks:
  - Variable/constant initializers
  - Assignment statements
  - Constant reassignment (generates error)
  - Function calls and return types
  - Field access on structs
  - Binary/unary operations
- **Implements requested error**: "cannot assign value of type X to variable/constant of type Y"

### Files created (Part 2)
- `internal/semantics/collector/collector.go` - Declaration collection
- `internal/semantics/resolver/resolver.go` - Type resolution
- `internal/semantics/checker/checker.go` - Type checking and validation
- `internal/diagnostics/codes.go` - Added `ErrConstantReassignment` error code

### Files modified (Part 2)
- `internal/context/context.go`
  - Added `CollectorRun`, `ResolverRun`, `CheckerRun` function variables
  - Added `RunCollectorPhase()`, `RunResolverPhase()`, `RunCheckerPhase()` methods
  - Integrated semantic passes into `Compile()` pipeline
- `main.go`
  - Registered semantic phase runners on startup

### Test files created
- `tests/test_type_check_valid.fer` - Valid type assignments (passes)
- `tests/test_type_check_mismatch.fer` - Type mismatches (errors correctly reported)
- `tests/test_const_reassign.fer` - Constant reassignment (error correctly reported)

### Test Results

**Valid assignments (should pass):**
```bash
.\compiler.exe --debug tests\test_type_check_valid.fer
# Result: ✓ Compilation successful!
```

**Type mismatches (should error):**
```bash
.\compiler.exe tests\test_type_check_mismatch.fer
# Result: error[T0001]: cannot assign value of type i32 to variable of type string
#         error[T0001]: cannot assign value of type f64 to constant of type i32
```

**Constant reassignment (should error):**
```bash
.\compiler.exe tests\test_const_reassign.fer
# Result: error[T0017]: cannot assign to constant 'PI'
```

**Go tests:**
```bash
go test ./...
# Result: All tests PASS
```

### Progress / Todo
- [x] Add helper for single decl parsing (`parseDeclItem`) — implemented
- [x] Refactor `parseVariableDeclaration` to use helper — implemented
- [x] Build the project (`go build ./...`) — OK
- [x] Create Pass 1: Collector — implemented
- [x] Create Pass 2: Resolver — implemented
- [x] Create Pass 3: Checker — implemented
- [x] Wire semantic passes into compilation pipeline — done
- [x] Test type checking with .fer files — all tests pass
- [x] Create this report — done

### Notes & Recommendations

#### Current Capabilities
The type checker now supports:
- ✅ Type mismatch detection for variable/constant assignments
- ✅ Constant reassignment prevention
- ✅ Type inference for `:=` syntax
- ✅ Scope-aware symbol resolution
- ✅ Function parameter type checking
- ✅ Struct field access validation
- ✅ Clear, actionable error messages with source locations

#### Future Enhancements
- Add numeric type coercion rules (e.g., i32 → i64 widening)
- Implement full operator type checking (arithmetic, comparison, logical)
- Add interface implementation validation
- Implement generic/template type support
- Add exhaustiveness checking for pattern matching

#### Architecture Benefits
- **Clean separation**: Each pass has a single responsibility
- **Extensible**: Easy to add new semantic checks in Pass 3
- **Debuggable**: Can inspect symbol tables and types after each pass
- **Performant**: Linear passes over AST, no backtracking
- **Proven design**: Matches industry-standard multi-pass compilers

### Next steps (Optional)
- Re-run SonarQube to confirm both refactors cleared linting issues
- Add unit tests for individual semantic pass components
- Extend type compatibility rules for numeric conversions
- Implement full expression type inference

---

**Generated by:** Automated development session  
**Compiler:** Ferret Language Compiler  
**Version:** 0.1.0-dev


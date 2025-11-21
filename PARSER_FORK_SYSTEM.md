# Parser Fork System

## Overview

This implementation adds a "parser fork" system to safely handle ambiguous grammar in the language, specifically the ambiguity between composite literals and blocks.

## The Problem

In expressions like `if T{} {}`, the parser cannot immediately determine whether `{}` after `T` is:
1. A composite literal completing the condition: `T{}`
2. The start of the if-statement's block body

Other languages face similar issues:
- Go uses lookahead to distinguish struct literals from blocks
- Rust uses "parser forks" for ambiguous rules
- C++ has the "most vexing parse" problem

## The Solution

The fork system allows the parser to:
1. **Fork**: Save current state (token position, diagnostics count)
2. **Try Parse**: Attempt to parse one interpretation
3. **Commit**: If successful, accept the result
4. **Restore**: If failed, rollback to saved state and try alternative

## Implementation Details

### Core Components

1. **Savepoint struct**: Captures parser state
   - `current`: Token position
   - `diagnosticsSize`: Number of diagnostics

2. **fork()**: Creates a savepoint

3. **restore(sp)**: Rolls back to savepoint

4. **commit(sp)**: Accepts current state (no-op, for symmetry)

5. **tryParse(fn)**: Wrapper that forks, tries parsing, commits or restores

### Safety Guarantees

- **No infinite loops**: tryParse attempts only once, never retries
- **Clean state**: Diagnostics are rolled back on failure
- **Minimal overhead**: Only stores token position and count

### Usage Example

```go
// In parseIfStmt, after parsing condition
if p.match(lexer.OPEN_CURLY) {
    // Try to extend condition with composite literal
    node, ok := p.tryParse(func() (ast.Node, error) {
        return p.tryParseCompositeLiteral(cond)
    })
    if ok {
        cond = node.(ast.Expression)
    }
    // Otherwise, { starts the body block
}
```

## Test Coverage

Comprehensive tests validate:
- ✅ Simple composite literals: `T{}`
- ✅ Composite with fields: `T{.x = 1, .y = 2}`
- ✅ Package-qualified types: `pkg::Type{}`
- ✅ Ambiguous if statements: `if T{} {}`
- ✅ Unknown types: `if MaybeType{} {}`
- ✅ Nested cases: `if T{} { if S{} {} }`
- ✅ Else clauses: `if T{} {} else if S{} {}`
- ✅ Regular conditions unchanged: `if flag {}`

All tests pass without regressions to existing functionality.

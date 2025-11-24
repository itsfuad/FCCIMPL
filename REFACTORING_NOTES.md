# Type Checker Refactoring: Control Flow Analysis Improvements

## Summary

This implementation addresses the design issues outlined in `issue.md` by:

1. **Removing fragile global state**: Eliminated `currentReturnType` from the Checker struct
2. **Using scope metadata**: Attached return type information directly to function scopes
3. **Improving diagnostics**: Implemented detailed flow path tracking with branch-level information

## Changes Made

### 1. SymbolTable Enhanced with Scope Metadata

**File**: `internal/semantics/s_table.go`

Added new types and fields:
```go
type ScopeKind int

const (
    ScopeModule ScopeKind = iota
    ScopeFunction
    ScopeBlock
)

type SymbolTable struct {
    // ... existing fields ...
    ScopeKind  ScopeKind
    ReturnType Type  // only meaningful for ScopeFunction
}
```

**Benefit**: The scope tree now encodes all information needed for control flow analysis. No need to maintain parallel global state.

### 2. Function Scope Marking in Resolver

**File**: `internal/semantics/resolver/resolver.go`

When resolving function types, we now also mark the function scope:
```go
// Mark function scope and attach return type for control flow analysis
if sym.SelfScope != nil {
    sym.SelfScope.ScopeKind = semantics.ScopeFunction
    sym.SelfScope.ReturnType = funcType.ReturnType
}
```

**Benefit**: Return type information is available throughout type checking without needing to be stored separately.

### 3. Removed Global State from Checker

**File**: `internal/semantics/checker/checker.go`

- Removed `currentReturnType` field from Checker struct
- Added `enclosingReturnType()` helper that walks the scope chain:

```go
func (c *Checker) enclosingReturnType() semantics.Type {
    for sc := c.currentScope; sc != nil; sc = sc.Parent {
        if sc.ScopeKind == semantics.ScopeFunction {
            return sc.ReturnType
        }
    }
    return nil // not inside a function
}
```

- Updated `checkReturnStmt` to use the helper instead of global state

**Benefit**: 
- Single source of truth (scope tree)
- Nested functions "just work" automatically
- No need for save/restore patterns
- More robust and maintainable

### 4. Flow Path Tracking System

**File**: `internal/semantics/checker/controlflow.go`

Implemented comprehensive control flow analysis with:

#### New Types
```go
type FlowStepKind int
const (
    StepFunctionBody FlowStepKind = iota
    StepIfThen
    StepIfElse
    StepLoopBody
)

type FlowStep struct {
    Kind FlowStepKind
    Loc  *source.Location
}

type FlowPath struct {
    Steps []FlowStep       // branch trace
    Loc   *source.Location // where path falls through
}

type FlowResult struct {
    AlwaysReturns bool       // true if ALL paths return
    MissingPaths  []FlowPath // paths that can fail to return
}
```

#### Analysis Functions
- `analyzeBlock()` - Analyzes blocks, detects unreachable code
- `analyzeStmt()` - Dispatches to specific statement analyzers
- `analyzeIfStmt()` - Analyzes if/else branches, requires both to return
- `analyzeForStmt()` - Conservative: loops don't guarantee execution
- `analyzeWhileStmt()` - Conservative: while loops don't guarantee execution
- `describeFlowPath()` - Converts flow paths to human-readable descriptions

**Result**: Much better diagnostics that show exactly which branch path fails to return

## Improvements in Error Messages

### Before
```
error[T0016]: missing return in function with return type i32
```

### After
```
error[T0016]: not all code paths in function 'incomplete' return a value of type i32
  --> file.fer:1:4
  |
1 | fn incomplete() -> i32 {
  |    ~~~~~~~~~~ function may exit without returning a value
  |
- this path (function body → else-branch at line 2) can reach end of function without returning
```

### Example: Nested Branches
```
error[T0016]: not all code paths in function 'nested' return a value of type i32
  --> file.fer:1:4
  |
1 | fn nested(x: i32, y: i32) -> i32 {
  |    ~~~~~~ function may exit without returning a value
  |
- this path (function body → if-branch → else-branch at line 4) can reach end of function without returning
```

## Architecture Benefits

1. **Maintainability**: Return type is data in the scope, not mutable checker state
2. **Correctness**: Nested functions automatically use correct return type
3. **Extensibility**: Easy to add more scope metadata for future features
4. **Testability**: Scope metadata can be verified independently
5. **Performance**: No expensive scope chain walks needed (happens once at analysis)

## Backward Compatibility

All changes are internal to the type checker. The public API and diagnostics codes remain the same. Existing code that uses the checker continues to work without modification.

## Testing

The implementation has been tested with:
- Missing returns in simple functions
- Incomplete returns in if-without-else branches
- Nested conditions with multiple missing paths
- Functions with all paths returning (no errors)
- Return type validation continues to work
- Unreachable code detection works correctly

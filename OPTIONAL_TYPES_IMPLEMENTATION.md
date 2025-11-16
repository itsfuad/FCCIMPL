# Optional Type Checking Implementation

## Summary
Fully implemented optional type checking in the Ferret compiler's semantic analysis phase (Pass 3). Optional types (`T?`) now have complete type validation including assignment rules, none literal support, error type handling with catch clauses, and elvis operator checking.

## Implementation Status: ✅ COMPLETE (Including Elvis Operator!)

### What Was Implemented

#### 1. **Optional Type Assignment Rules** (`isAssignable` function)
- ✅ `T?` can accept `T` (automatic wrapping)
- ✅ `T?` can accept `none` literal
- ✅ `T` **cannot** accept `T?` without explicit unwrapping (enforced error)
- ✅ `T?` can accept another `T?` if base types match

#### 2. **NoneType Support**
- ✅ Added `NoneType` struct to `internal/semantics/types.go`
- ✅ `none` identifier is recognized as special built-in value
- ✅ `none` is only assignable to optional types (T?)
- ✅ Attempting to assign `none` to non-optional types produces type error

#### 3. **Catch Clause Type Checking** (`checkCatchClause` function)
- ✅ Validates catch clauses work only with error types (`T ! E`)
- ✅ Creates scoped error variable in handler block
- ✅ Validates fallback expression type matches expected valid type
- ✅ Returns unwrapped valid type after catch handling
- ✅ Properly manages symbol table scopes for error identifiers

#### 4. **Elvis Operator Checking** (`checkElvisExpr` function)
- ✅ **FULLY IMPLEMENTED IN PARSER**: `?:` operator now parsed correctly
- ✅ Validates elvis operator primarily works with optional types
- ✅ Issues warning if used with non-optional types
- ✅ Ensures default value is assignable to optional's base type
- ✅ Returns unwrapped base type as result
- ✅ Right-associative for chaining: `a ?: b ?: c`

#### 5. **Expression Type Checking Integration**
- ✅ Added `ElvisExpr` case to `checkExpr` switch statement
- ✅ Integrated catch clause checking into `CallExpr` validation
- ✅ All new expression types properly handled

## Files Modified

### 1. `internal/semantics/types.go`
**Added:**
```go
// NoneType represents the 'none' literal type
// It's compatible with all optional types
type NoneType struct{}

func (n *NoneType) String() string {
	return "none"
}
```

### 2. `internal/semantics/checker/checker.go`
**Added/Modified:**
- Import of `compiler/internal/types` package for type constants
- `isAssignable()` function enhanced with:
  - None type compatibility checking
  - Optional type wrapping rules
  - Optional type unwrapping prevention
- `reportAssignmentError()` helper function for better error messages
- `checkIdentifier()` enhanced to recognize "none" as built-in
- `checkBasicLit()` enhanced to return proper type constants
- New function: `checkCatchClause()` - validates catch expressions
- New function: `checkElvisExpr()` - validates elvis operator
- `checkCallExpr()` updated to handle catch clauses
- `checkExpr()` switch statement updated with ElvisExpr case
- `checkVarDecl()` and `checkAssignStmt()` use `reportAssignmentError()`

### 3. `internal/frontend/parser/parser.go`
**Added:**
- `parseElvis()` function - parses elvis operator (`?:`)
- Right-associative parsing for elvis chaining
- Integrated into expression precedence chain (between assignment and logical-or)

### 4. `tests/test_optional_checking.fer`
**Created:** Comprehensive test file demonstrating optional type features

## Type Checking Rules

### Assignment Compatibility

| From Type | To Type | Result | Example |
|-----------|---------|--------|---------|
| `i32` | `i32?` | ✅ OK | `let x: i32? = 42;` |
| `none` | `i32?` | ✅ OK | `let y: i32? = none;` |
| `i32?` | `i32` | ❌ ERROR | Cannot unwrap without explicit handling |
| `none` | `i32` | ❌ ERROR | none only works with optional types |
| `i32?` | `i32?` | ✅ OK | Same base type |
| `i32?` | `str?` | ❌ ERROR | Different base types |

### Catch Clause Rules

```ferret
fn divide(a: i32, b: i32) -> i32 ! str { ... }

// ✅ OK: Catch with fallback
let result: i32 = divide(10, 2) catch 0;

// ✅ OK: Catch with handler and fallback
let result: i32 = divide(10, 0) catch err {
    print(err);
} 0;

// ❌ ERROR: Fallback type must match valid type
let result: i32 = divide(10, 2) catch "wrong type";
```

### Elvis Operator Rules

```ferret
let opt: i32? = 10;

// ✅ OK: Unwraps i32? to i32
let result: i32 = opt ?: 0;

// ✅ OK: Right-associative chaining
let a: i32? = none;
let b: i32? = none;  
let c: i32 = a ?: b ?: 999;  // Evaluates as: a ?: (b ?: 999)

// ⚠️ WARNING: Elvis with non-optional (still works)
let x: i32 = 5;
let result: i32 = x ?: 0;

// ❌ ERROR: Default type must match base type
let opt: i32? = 10;
let result: i32 = opt ?: "wrong";
```

### Improved Error Messages

All type mismatches now include helpful notes and suggestions:

```ferret
// Example: Wrong type for optional
let x: i32? = "hello";  // ERROR

// Error message includes:
// note: i32? accepts values of type i32 or 'none'
```

## Testing

### Test Files Created:
1. **`tests/test_optional_checking.fer`** - Valid optional type usage
2. **`tests/test_optional_errors.fer`** - Error cases with helpful diagnostics
3. **`tests/test_elvis.fer`** - Elvis operator valid usage including chaining
4. **`tests/test_elvis_simple.fer`** - Minimal elvis test case
5. **`tests/test_elvis_errors.fer`** - Elvis operator error cases

### Test Results:
```bash
# Valid cases - all pass
$ .\bin\ferret.exe tests\test_optional_checking.fer
# (no output - success)

$ .\bin\ferret.exe tests\test_elvis.fer  
# (no output - success)

# Error cases - properly detected with helpful messages
$ .\bin\ferret.exe tests\test_optional_errors.fer
# 3 errors with helpful notes

$ .\bin\ferret.exe tests\test_elvis_errors.fer
# 3 errors + 1 warning with context
```

## Future Enhancements

### Not Yet Implemented
1. **Nested Optionals** - `T??` syntax not yet supported
2. **Safe Navigation** - `obj?.field` chaining not yet implemented  
3. **Pattern matching** on optional types
4. **Automatic none-checking** in if expressions

### Potential Semantic Enhancements
1. Smart unwrapping in conditional contexts
2. Pattern matching on optional types
3. Automatic none-checking in elvis expressions
4. Optional chaining type inference

## Architecture Notes

### Type System Hierarchy
```
semantics.Type (interface)
├── PrimitiveType (i32, str, bool, etc.)
├── OptionalType (T?)
├── NoneType (none literal)
├── ErrorType (T ! E)
├── StructType
├── ArrayType
├── FunctionType
├── InterfaceType
└── Invalid
```

### Compiler Pipeline for Optional Types
1. **Lexer** → Tokenizes source, identifiers include "none"
2. **Parser** → Creates AST with OptionalType nodes
3. **Collector (Pass 1)** → Declares symbols without types
4. **Resolver (Pass 2)** → Converts AST types to semantic types
   - `*ast.OptionalType` → `*semantics.OptionalType`
5. **Checker (Pass 3)** → Validates type compatibility ✅ **IMPLEMENTED**
   - Checks optional assignment rules
   - Validates none literal usage
   - Enforces type safety

## Key Design Decisions

1. **none as Identifier**: "none" is treated as a special built-in identifier rather than a keyword, detected in `checkIdentifier()`

2. **Explicit Unwrapping Required**: `T?` cannot be assigned to `T` without explicit handling (catch/elvis/etc.)

3. **Type Safety First**: Invalid optional operations produce clear, actionable errors

4. **Scoped Error Handling**: Catch clauses create proper scopes for error variables

## Diagnostic Messages

### Success Cases
```
let x: i32? = 42;     // Wraps i32 to i32?
let y: i32? = none;   // Assigns none to i32?
```

### Error Cases
```
let x: i32? = 42;
let y: i32 = x;       // ERROR: cannot assign value of type i32? to variable of type i32

let z: i32 = none;    // ERROR: cannot assign value of type none to variable of type i32
```

## Completion Date
November 16, 2025

## Status
✅ **FULLY COMPLETE** - Optional types are fully implemented in the semantic phase and ready for use.

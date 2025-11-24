# Control Flow Analysis Implementation

## Overview

The Ferret compiler now includes comprehensive control flow analysis that validates function return paths and detects unreachable code. This system ensures type safety and code correctness at compile time.

## Features

### 1. Return Path Analysis
The compiler tracks all possible execution paths through a function and ensures that functions with return types actually return values on all code paths.

**Example:**
```ferret
fn maybeReturn(x: i32) -> i32 {
    if x > 0 {
        return x;
    }
    // Error: missing return - what if x <= 0?
}
```

### 2. Return Type Validation
Every return statement is checked to ensure the returned value matches the function's declared return type.

**Example:**
```ferret
fn getName() -> str {
    return 42;  // Error: cannot return i32 in function with return type str
}
```

### 3. Branch Analysis
The system analyzes conditional statements (if/else) to determine if all branches provide return values.

**Example:**
```ferret
fn abs(x: i32) -> i32 {
    if x < 0 {
        return -x;
    } else {
        return x;
    }
    // OK: both branches return
}
```

### 4. Unreachable Code Detection
Code that appears after terminal statements (return, break, continue) is flagged with a warning.

**Example:**
```ferret
fn example() -> i32 {
    return 42;
    let x := 10;  // Warning: unreachable code
}
```

### 5. Loop Statement Validation
Break and continue statements are validated to ensure they only appear inside loops.

**Example:**
```ferret
fn invalid() -> i32 {
    break;  // Error: break statement outside loop
    return 0;
}
```

### 6. Void Function Checks
Functions without return types cannot return values, but can use empty return statements.

**Example:**
```ferret
fn valid() {
    return;  // OK: empty return
}

fn invalid() {
    return 42;  // Error: cannot return a value in function with no return type
}
```

## Implementation Details

### File Structure
- `internal/semantics/checker/controlflow.go` - Core control flow analysis logic
- `internal/semantics/checker/checker.go` - Integration with type checker
- `internal/diagnostics/codes.go` - Diagnostic error codes

### Key Components

#### FlowResult Struct
```go
type FlowResult struct {
    Returns    bool              // Whether this path definitely returns
    Breaks     bool              // Whether this path breaks
    Continues  bool              // Whether this path continues
    ReturnType semantics.Type    // Type of returned value
    ReturnLoc  interface{}       // Location of return statement
}
```

#### Analysis Functions
- `checkFunctionReturns()` - Entry point for function analysis
- `analyzeFlow()` - Dispatches to specific node type analyzers
- `analyzeBlockFlow()` - Analyzes blocks, detects unreachable code
- `analyzeIfFlow()` - Analyzes if/else branches
- `analyzeForFlow()` - Analyzes for loops
- `analyzeWhileFlow()` - Analyzes while loops

### Diagnostic Codes
- `T0015` - Return value in void function
- `T0016` - Missing return in function
- `T0017` - Invalid return type
- `T0018` - Invalid break statement
- `T0019` - Invalid continue statement
- `W0001` - Unreachable code warning

## Testing

Comprehensive test cases are available in `tests/test_control_flow.fer`, covering:
- Missing returns
- Incomplete return paths (if without else)
- Complete return paths (if with else)
- Unreachable code detection
- Nested conditionals
- Return type mismatches
- Void function validation

## Usage

The control flow analysis runs automatically during the type checking phase. No special flags or configuration are required.

```bash
go run main.go yourfile.fer
```

If there are control flow errors, they will be reported with clear diagnostic messages:

```
error[T0016]: missing return in function with return type i32
  --> yourfile.fer:5:4
  |
5 | fn example(x: i32) -> i32 {
  |    ~~~~~~~ function must return a value
  |
```

## Future Enhancements

Potential improvements for future versions:
- Exhaustiveness checking for match/switch statements
- Detection of infinite loops
- Analysis of panic/exit paths
- More sophisticated reachability analysis
- Dead code elimination hints

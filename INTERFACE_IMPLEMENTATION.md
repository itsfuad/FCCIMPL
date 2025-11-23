# Interface Type Checking Implementation

## Overview

The Ferret compiler now has complete interface support with Go-style semantics:

1. **Empty interfaces** - Accept all types (similar to `interface{}` in Go or `any` in TypeScript)
2. **Method interfaces** - Define contracts that types must implement
3. **Implicit implementation** - Types implement interfaces automatically by having matching methods
4. **Compile-time checking** - Interface implementation is verified during type checking

## Syntax

### Defining Interfaces

```ferret
type InterfaceName interface {
    method1() -> ReturnType,
    method2(param: ParamType) -> ReturnType
};
```

**Important**: Interface method declarations do NOT use the `fn` keyword.

### Empty Interface

```ferret
type Any interface {};
```

Empty interfaces have no methods and are implemented by all types.

### Implementing Interfaces

Types implement interfaces implicitly by defining methods with matching signatures:

```ferret
type Rectangle struct {
    .width: f64,
    .height: f64
};

fn (r: Rectangle) area() -> f64 {
    return r.width * r.height;
}
```

If an interface `Shape` requires an `area() -> f64` method, then `Rectangle` implements `Shape`.

## Type Checking Rules

### 1. Interface Assignment

A value can be assigned to an interface variable if:
- The interface is empty (accepts all types), OR
- The value's type implements all methods required by the interface

```ferret
type Shape interface {
    area() -> f64
};

type Circle struct { .radius: f64 };
fn (c: Circle) area() -> f64 { return 3.14 * c.radius * c.radius; }

let c := {.radius = 5.0} as Circle;
let s: Shape = c;  // OK - Circle implements Shape
```

### 2. Method Signature Matching

For a type to implement an interface method, the signatures must match:
- Parameter count must be the same
- Parameter types must match exactly (by type string representation)
- Return types must match exactly
- Variadic status must match

**Note**: The receiver parameter is NOT part of the method signature for matching purposes.

### 3. Multiple Interface Implementation

A type can implement multiple interfaces if it has all required methods:

```ferret
type Drawable interface {
    draw() -> str
};

type Shape interface {
    area() -> f64
};

type Circle struct { .radius: f64 };
fn (c: Circle) draw() -> str { return "circle"; }
fn (c: Circle) area() -> f64 { return 3.14 * c.radius * c.radius; }

let c := {.radius = 5.0} as Circle;
let d: Drawable = c;  // OK
let s: Shape = c;      // Also OK
```

## Implementation Details

### Key Functions

1. **`implementsInterface(typ, iface)`** - Checks if a type implements an interface
   - Returns true for empty interfaces
   - Verifies all interface methods exist on the type
   - Calls `methodSignaturesMatch` for each method

2. **`methodSignaturesMatch(impl, iface)`** - Compares method signatures
   - Checks parameter count
   - Compares parameter types using string representation
   - Compares return types using string representation
   - Verifies variadic status matches

3. **`getInterfaceType(typ)`** - Unwraps UserType to get InterfaceType
   - Handles both direct InterfaceType and UserType wrapping InterfaceType
   - Returns nil if type is not an interface

### isAssignable Enhancement

The `isAssignable` function was enhanced to check for interface targets BEFORE structural comparison:

```go
// Check if 'to' is an interface type - if so, check implementation
if toIsUser {
    if toInterface := c.getInterfaceType(toUser); toInterface != nil {
        return c.implementsInterface(from, toInterface)
    }
}
```

This is crucial because:
- Interfaces are wrapped in UserType (named types)
- Without this check, the code would try to do structural comparison between interface and struct definitions
- Methods are stored on the UserType, not on the underlying struct definition

## Error Messages

When a type doesn't implement an interface, you get a clear error:

```
error[T0001]: cannot assign value of type Point to symbol of type Printable
  --> test.fer:20:28
   |
19 | let pt := {.x = 1.0, .y = 2.0} as Point;
20 | let printable: Printable = pt;
   |     ---------              ~~ expression has type Point
   |     |
   |     -- symbol has type Printable
```

## Examples

### Example 1: Basic Interface

```ferret
type Stringer interface {
    toString() -> str
};

type Point struct {
    .x: f64,
    .y: f64
};

fn (p: Point) toString() -> str {
    return "Point";
}

let pt := {.x = 1.0, .y = 2.0} as Point;
let s: Stringer = pt;  // OK
```

### Example 2: Empty Interface

```ferret
type Any interface {};

let anyInt: Any = 42;
let anyStr: Any = "hello";
let anyStruct: Any = {.x = 1.0} as Point;
```

### Example 3: Method Signature Mismatch (Error)

```ferret
type Calculator interface {
    add(a: i32, b: i32) -> i32
};

type SimpleCalc struct { .dummy: i32 };

// Wrong! Takes f64 instead of i32
fn (c: SimpleCalc) add(a: f64, b: f64) -> i32 {
    return 0;
}

let calc := {.dummy = 1} as SimpleCalc;
let calculator: Calculator = calc;  // ERROR - signature mismatch
```

### Example 4: Missing Method (Error)

```ferret
type Printable interface {
    print() -> str,
    format() -> str
};

type Point struct { .x: f64, .y: f64 };

// Only implements print(), missing format()
fn (p: Point) print() -> str {
    return "Point";
}

let pt := {.x = 1.0, .y = 2.0} as Point;
let printable: Printable = pt;  // ERROR - missing format() method
```

## Testing

Run the provided test files to verify interface functionality:

- `test_empty_interface.fer` - Empty interface accepts all types
- `test_interface_complete.fer` - Comprehensive interface tests
- `test_interface_error.fer` - Missing method detection
- `test_interface_signature_error.fer` - Signature mismatch detection
- `test_rectangle_shape.fer` - Classic shape example

All tests should pass (or fail with expected errors for error tests).

## Future Enhancements

Potential future improvements:
1. **Method calls through interface variables** - Currently only assignment is supported
2. **Covariance/Contravariance** - Allow compatible return/parameter types
3. **Interface embedding** - Interfaces that extend other interfaces
4. **Type assertions** - Runtime type checking for interface values
5. **Better error messages** - Show which methods are missing or have wrong signatures

---
title: Syntax Basics
description: Learn the fundamental syntax of Ferret
---

# Ferret Syntax

This page covers the basic syntax and structure of Ferret programs.

## Comments

```ferret
// Single-line comment

/* Multi-line
   comment */
```

## Variables and Constants

### Variable Declaration

```ferret
// With type annotation
let name: str = "Ferret";
let age: i32 = 1;

// Type inference with walrus operator
let count := 42;  // i32 inferred
let message := "Hello";  // str inferred
```

### Constants

```ferret
const PI: f64 = 3.14159;
const MAX_SIZE := 1000;
```

## Basic Types

| Type | Description | Example |
|------|-------------|---------|
| `i32` | 32-bit integer | `42` |
| `i64` | 64-bit integer | `1000` |
| `f32` | 32-bit float | `3.14` |
| `f64` | 64-bit float | `2.718` |
| `str` | String | `"hello"` |
| `bool` | Boolean | `true`, `false` |
| `byte` | Character | `'a'` |

## Optional Types

Optional types can be `T` or `none`:

```ferret
let maybeNumber: i32? = 42;
let nothing: str? = none;

// Check for none
if maybeNumber != none {
    // maybeNumber is i32 here (type narrowing)
    let doubled: i32 = maybeNumber * 2;
}

// Elvis operator for default values
let value: i32 = maybeNumber ?: 0;
```

## Control Flow

### If Statements

```ferret
if condition {
    // code
} else if another_condition {
    // code
} else {
    // code
}
```

### Loops

```ferret
// While loop
while x < 10 {
    x = x + 1;
}

// For loop
for i in 0..10 {
    print(i);
}
```

### When Expression

```ferret
when value {
    1 => print("one"),
    2 => print("two"),
    _ => print("other"),
}
```

## Functions

```ferret
// Basic function
fn greet(name: str) -> str {
    return "Hello, " + name;
}

// Function with optional parameter
fn divide(a: i32, b: i32) -> i32? {
    if b == 0 {
        return none;
    }
    return a / b;
}

// Function with error type
fn readFile(path: str) -> str ! Error {
    // Can return str or Error
}
```

## Operators

### Arithmetic
- `+` Addition
- `-` Subtraction
- `*` Multiplication
- `/` Division
- `%` Modulo

### Comparison
- `==` Equal
- `!=` Not equal
- `<` Less than
- `>` Greater than
- `<=` Less or equal
- `>=` Greater or equal

### Logical
- `and` Logical AND
- `or` Logical OR
- `not` Logical NOT

### Special
- `?:` Elvis operator (optional unwrap with default)
- `:=` Walrus operator (declaration with type inference)
- `::` Static access (modules, enums)

## Next Steps

- [Learn about the type system in detail](/language/types)
- [Explore optional types](/language/optionals)
- [See error handling](/language/errors)



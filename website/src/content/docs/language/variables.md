---
title: "Variables & Constants"
description: "Learn about variables and constants in Ferret"
---

Ferret provides two ways to declare named values: mutable variables with `let` and immutable constants with `const`.

## Variable Declaration

Variables are declared using the `let` keyword and can be reassigned after declaration.

### With Type Annotation

```ferret
let name: str = "Ferret";
let age: i32 = 1;
let price: f64 = 99.99;
```

### Type Inference with Walrus Operator

The walrus operator `:=` allows you to declare variables with type inference:

```ferret
let count := 42;        // i32 inferred
let message := "Hello"; // str inferred
let active := true;     // bool inferred
```

### Variable Reassignment

Variables can be reassigned to new values of the same type:

```ferret
let score: i32 = 0;
score = 10;  // ✅ OK
score = 20;  // ✅ OK
```

## Constants

Constants are declared using the `const` keyword and cannot be reassigned after initialization.

```ferret
const PI: f64 = 3.14159;
const MAX_SIZE := 1000;
const APP_NAME := "Ferret Compiler";
```

### Attempting to Reassign

```ferret
const VERSION := 1;
VERSION = 2;  // ❌ ERROR: Cannot assign to constant
```

## Naming Conventions

### Variables and Constants
- Use `snake_case` for variables: `user_name`, `total_count`
- Use `SCREAMING_SNAKE_CASE` for constants: `MAX_VALUE`, `DEFAULT_TIMEOUT`

```ferret
let user_name := "Alice";
let total_count := 42;

const MAX_RETRIES := 3;
const DEFAULT_PORT := 8080;
```

## Scope

Variables and constants are scoped to the block in which they are declared:

```ferret
let x := 10;

if true {
    let y := 20;
    // Both x and y are accessible here
    let sum := x + y;  // ✅ OK
}

// y is not accessible here
let result := y;  // ❌ ERROR: y not found
```

## Shadowing

You can declare a new variable with the same name in a nested scope:

```ferret
let x := 5;

if true {
    let x := 10;  // Shadows outer x
    print(x);     // Prints: 10
}

print(x);  // Prints: 5
```

## Next Steps

- [Learn about Data Types](/language/types)
- [Explore Operators](/language/operators)
- [Understand Control Flow](/language/if-statements)

---
title: Error Handling
description: Working with errors and result types in Ferret
---

Ferret uses explicit error handling with result types.

## Result Types

Functions that can fail return `T ! E` (Result type):

```ferret
fn divide(a: i32, b: i32) -> i32 ! str {
    if b == 0 {
        return error("Division by zero");
    }
    return a / b;
}
```

## Handling Errors

Use match to handle success and error cases:

```ferret
let result := divide(10, 2);

match result {
    ok(value) => print("Result: " + value),
    error(msg) => print("Error: " + msg),
}
```

## Error Propagation

Use `?` operator to propagate errors:

```ferret
fn calculate() -> i32 ! str {
    let x := divide(10, 2)?;  // Propagate error if any
    let y := divide(x, 3)?;
    return y;
}
```

## Try-Catch Alternative

Ferret doesn't use exceptions, preferring explicit error handling:

```ferret
// No try-catch - use match instead
let result := risky_operation();
match result {
    ok(value) => {
        // Handle success
    },
    error(e) => {
        // Handle error
    },
}
```

## Custom Error Types

Define your own error types:

```ferret
enum FileError {
    NotFound,
    PermissionDenied,
    IOError(str),
}

fn read_file(path: str) -> str ! FileError {
    // Implementation
}
```

## Next Steps

- [Learn about Generics](/language/generics)
- [Explore Modules](/language/modules)

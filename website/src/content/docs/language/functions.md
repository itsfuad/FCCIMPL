---
title: Functions
description: Defining and calling functions in Ferret
---

Functions are reusable blocks of code that perform specific tasks.

## Function Declaration

```ferret
fn greet(name: str) -> str {
    return "Hello, " + name;
}
```

## Calling Functions

```ferret
let message := greet("World");
print(message);  // Hello, World
```

## Return Types

Functions can return values:

```ferret
fn add(a: i32, b: i32) -> i32 {
    return a + b;
}

let sum := add(5, 3);  // 8
```

## Void Functions

Functions that don't return a value:

```ferret
fn log_message(msg: str) {
    print("[INFO] " + msg);
}
```

## Expression Body

Single-expression functions:

```ferret
fn square(x: i32) -> i32 = x * x;
```

## Next Steps

- [Learn about Parameters](/language/parameters)
- [Explore Error Handling](/language/errors)

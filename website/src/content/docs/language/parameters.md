---
title: Parameters
description: Function parameters in Ferret
---

Learn how to define and use function parameters effectively.

## Basic Parameters

```ferret
fn add(a: i32, b: i32) -> i32 {
    return a + b;
}
```

## Optional Parameters

Parameters can be optional:

```ferret
fn greet(name: str, title: str?) -> str {
    if title != none {
        return "Hello, " + title + " " + name;
    }
    return "Hello, " + name;
}
```

## Default Values

Parameters with default values:

```ferret
fn power(base: i32, exponent: i32 = 2) -> i32 {
    // Implementation
}

let squared := power(5);      // Uses default exponent=2
let cubed := power(5, 3);     // Explicit exponent=3
```

## Variadic Parameters

Accept multiple arguments:

```ferret
fn sum(numbers: ...i32) -> i32 {
    let total := 0;
    for n in numbers {
        total = total + n;
    }
    return total;
}

let result := sum(1, 2, 3, 4, 5);  // 15
```

## Next Steps

- [Learn about Structs](/language/structs)
- [Explore Methods](/language/structs)

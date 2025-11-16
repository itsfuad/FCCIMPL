---
title: Match Expressions
description: Pattern matching in Ferret
---

Match expressions provide powerful pattern matching capabilities.

## Basic Match

```ferret
let status := 200;

match status {
    200 => print("OK"),
    404 => print("Not Found"),
    500 => print("Server Error"),
    _ => print("Unknown Status"),
}
```

## Match with Values

Match expressions return values:

```ferret
let message := match status {
    200 => "Success",
    404 => "Not Found",
    _ => "Error",
};
```

## Pattern Matching

Match on complex patterns:

```ferret
match value {
    0 => "zero",
    1..=10 => "small",
    11..=100 => "medium",
    _ => "large",
}
```

## Next Steps

- [Learn about Functions](/language/functions)
- [Explore Enums](/language/enums)

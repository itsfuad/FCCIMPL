---
title: Comments
description: Learn how to write comments in Ferret
---

Comments are used to document your code and are ignored by the compiler.

## Single-Line Comments

Use `//` for single-line comments:

```ferret
// This is a single-line comment
let x := 42;  // Comments can also go at the end of a line
```

## Multi-Line Comments

Use `/* */` for multi-line comments:

```ferret
/* This is a multi-line comment
   that spans multiple lines.
   Useful for longer explanations. */

let y := 10;
```

## Documentation Comments

Use `///` for documentation comments that can be extracted by documentation tools:

```ferret
/// Calculates the sum of two numbers.
/// 
/// # Parameters
/// - `a`: The first number
/// - `b`: The second number
/// 
/// # Returns
/// The sum of `a` and `b`
fn add(a: i32, b: i32) -> i32 {
    return a + b;
}
```

## Best Practices

### ✅ Do
- Write comments to explain **why**, not **what**
- Use comments for complex logic
- Keep comments up-to-date with code changes

```ferret
// Calculate discount based on loyalty tier
// Premium members get 20% off
let discount := user.is_premium ? 0.20 : 0.10;
```

### ❌ Don't
- State the obvious
- Leave commented-out code in production
- Use comments as a substitute for clear code

```ferret
// Bad: Stating the obvious
let x := 5;  // Set x to 5

// Better: No comment needed, code is clear
let retry_count := 5;
```

## Next Steps

- [Learn about Variables](/language/variables)
- [Explore Functions](/language/functions)

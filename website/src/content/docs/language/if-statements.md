---
title: If Statements
description: Learn about conditional logic in Ferret
---

If statements allow you to execute code conditionally based on boolean expressions.

## Basic If Statement

```ferret
let age := 18;

if age >= 18 {
    print("You are an adult");
}
```

## If-Else

```ferret
let score := 75;

if score >= 60 {
    print("You passed!");
} else {
    print("You failed.");
}
```

## If-Else If-Else

```ferret
let grade := 85;

if grade >= 90 {
    print("A");
} else if grade >= 80 {
    print("B");
} else if grade >= 70 {
    print("C");
} else {
    print("F");
}
```

## Type Narrowing with Optionals

Ferret automatically narrows types in conditional branches:

```ferret
let maybe_value: i32? = 42;

if maybe_value != none {
    // maybe_value is i32 here, not i32?
    let doubled: i32 = maybe_value * 2;
}
```

Learn more about [Optional Types](/language/optionals).

## Next Steps

- [Learn about Loops](/language/loops)
- [Explore Match Expressions](/language/match)
- [Understand Optional Types](/language/optionals)

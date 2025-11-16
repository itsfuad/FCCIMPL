---
title: Generics
description: Generic types and functions in Ferret
---

We use fixed types in our programs, but sometimes we want to write code that can operate on different types without duplicating it. Generics allow you to write flexible, reusable code that works with multiple types.

## Generic Functions

To define a generic function, we use angle brackets `<T>` to specify type parameters.

```ferret
fn identity<T>(value: T) -> T {
    return value;
}

let num := identity(42);         // T = i32
let text := identity("hello");   // T = str
```

## Generic Structs

```ferret
type Box struct<T> {
    .value: T,
};

let int_box := Box<i32>{.value = 42};
let str_box := Box<str>{.value = "hello"};
```

## Generic Interfaces

```ferret
type Container interface<T> {
    get() -> T;
    set(value: T);
};
```

## Multiple Type Parameters

```ferret
type Pair struct<K, V> {
    .key: K,
    .value: V,
};

let entry := Pair<str, i32>{.key = "age", .value = 30};
```

## Generic Methods

```ferret
type Box struct<T> {
    .value: T,
};

fn (b: Box<T>) get() -> T {
    return b.value;
}
```

## Next Steps

- [Learn about Modules](/language/modules)
- [Explore Advanced Topics](/language/syntax)

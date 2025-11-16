---
title: Generics
description: Generic types and functions in Ferret
---

Generics allow you to write flexible, reusable code that works with multiple types.

## Generic Functions

```ferret
fn identity<T>(value: T) -> T {
    return value;
}

let num := identity(42);         // T = i32
let text := identity("hello");   // T = str
```

## Generic Structs

```ferret
type Box<T> struct {
    .value: T,
};

let int_box := Box{.value = 42};
let str_box := Box{.value = "hello"};
```

## Generic Interfaces

```ferret
type Container<T> interface {
    get() -> T;
    set(value: T);
};
```

## Multiple Type Parameters

```ferret
type Pair<K, V> struct {
    .key: K,
    .value: V,
};

let entry := Pair{.key = "age", .value = 30};
```

## Generic Methods

```ferret
type Box<T> struct {
    .value: T,
};

fn (b: Box<T>) get() -> T {
    return b.value;
}
```

## Next Steps

- [Learn about Modules](/language/modules)
- [Explore Advanced Topics](/language/syntax)

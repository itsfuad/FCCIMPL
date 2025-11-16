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
struct Box<T> {
    value: T,
}

let int_box := Box{ value: 42 };
let str_box := Box{ value: "hello" };
```

## Generic Interfaces

```ferret
interface Container<T> {
    fn get(self) -> T;
    fn set(self, value: T);
}
```

## Type Constraints

Constrain generic types:

```ferret
fn print_if_displayable<T: Display>(value: T) {
    print(value.to_string());
}
```

## Multiple Type Parameters

```ferret
struct Pair<K, V> {
    key: K,
    value: V,
}

let entry := Pair{ key: "age", value: 30 };
```

## Generic Methods

```ferret
impl<T> Box<T> {
    fn new(value: T) -> Box<T> {
        return Box{ value: value };
    }
    
    fn get(self) -> T {
        return self.value;
    }
}
```

## Next Steps

- [Learn about Modules](/language/modules)
- [Explore Advanced Topics](/language/syntax)

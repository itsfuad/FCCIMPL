---
title: Data Types
description: Learn about Ferret's built-in data types
---

Ferret provides a rich set of built-in types for different kinds of data.

## Primitive Types

### Integer Types

| Type | Size | Range | Description |
|------|------|-------|-------------|
| `i32` | 32-bit | -2³¹ to 2³¹-1 | Default integer type |
| `i64` | 64-bit | -2⁶³ to 2⁶³-1 | Large integers |
| `u32` | 32-bit | 0 to 2³²-1 | Unsigned integer |
| `u64` | 64-bit | 0 to 2⁶⁴-1 | Large unsigned integer |

```ferret
let count: i32 = 42;
let big_number: i64 = 9223372036854775807;
let positive: u32 = 4294967295;
```

### Floating-Point Types

| Type | Size | Precision | Description |
|------|------|-----------|-------------|
| `f32` | 32-bit | ~7 digits | Single precision |
| `f64` | 64-bit | ~15 digits | Double precision (default) |

```ferret
let pi: f32 = 3.14159;
let e: f64 = 2.718281828459045;
```

### String Type

Strings are UTF-8 encoded text:

```ferret
let name: str = "Ferret";
let emoji: str = "🦦";
let multiline: str = "Hello
World";
```

### Boolean Type

```ferret
let is_active: bool = true;
let is_complete: bool = false;
```

### Character Type

Single Unicode characters:

```ferret
let letter: byte = 'A';
let symbol: byte = '♠';
let emoji: byte = '🎉';
```

## Compound Types

### Arrays

Arrays are collections of elements of the same type. The elements can be fixed or dynamic in size.

```ferret
let numbers: []i32 = [1, 2, 3, 4, 5];
```
We created an array of integers. No size specified means dynamic size. But you can also define fixed-size arrays:

```ferret
let numbers: [5]i32 = [1, 2, 3, 4, 5];
```
Now the array can only hold 5 integers. You cannot add or remove elements.

## Optional Types

Types that can be `none`:

```ferret
let maybe_number: i32? = 42;
let no_value: str? = none;
```

Learn more about [Optional Types](/language/optionals).

## Custom Types

### Type Aliases

Create new names for existing types:

```ferret
type UserId = i64;
type Email = str;

let user_id: UserId = 12345;
let email: Email = "user@example.com";
```

## Type Inference

Ferret can infer types automatically:

```ferret
let number := 42;        // i32
let text := "Hello";     // str
let flag := true;        // bool
let decimal := 3.14;     // f64
```

## Type Conversion

### Explicit Casting

```ferret
let x: i32 = 42;
let y: i64 = x as i64;  // Convert i32 to i64

let pi: f64 = 3.14;
let rounded: i32 = pi as i32;  // 3
```

### String Conversion

```ferret
let num: i32 = 42;
let text: str = num.to_string();

let parsed: i32? = "123".parse_int();
```

## Next Steps

- [Learn about Operators](/language/operators)
- [Explore Structs](/language/structs)
- [Understand Optional Types](/language/optionals)

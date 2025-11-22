---

title: "Data Types"
description: "Learn about Ferret's built-in data types"
---

Now that you know how to create variables and constants, it's time to learn what kinds of values they can store. These are called **data types**.

Ferret comes with a set of built‑in types that let you work with numbers, text, true/false values, and more.

## Primitive Types

Primitive types are the simplest kinds of data. Internally they are just numbers. 

### Integer Types

These types store whole numbers.

| Type  | Size   | Range         | Description                 |
| ----- | ------ | ------------- | --------------------------- |
| `i32` | 32‑bit | -2³¹ to 2³¹‑1 | Standard integer            |
| `i64` | 64‑bit | -2⁶³ to 2⁶³‑1 | Bigger integer              |
| `u32` | 32‑bit | 0 to 2³²‑1    | Non‑negative integer        |
| `u64` | 64‑bit | 0 to 2⁶⁴‑1    | Bigger non‑negative integer |

Now if you are confused about the `i` and `u` prefixes, `i` stands for signed integers (can be negative) and `u` stands for unsigned integers (non-negative only). And the numbers `32` and `64` stand for the number of bits used to store the value. Other languages may use different names for these types, but the concepts are the same. So when you see `i32`, think of it as a 32-bit signed integer.

Now remember the `:=` operator you learned about in the Variables & Constants section? It is used for declaring variables and constants with type inference. Type inference means Ferret can automatically figure out the type based on the value you provide. But if you want to explicitly specify the type, you can do so using a colon `:` followed by the type name.

```ferret
let count: i32 = 42;
let big_number: i64 = 9223372036854775807;
let positive: u32 = 4294967295;
```

### Floating‑Point Types

These types store numbers with decimals.

| Type  | Size   | Precision  | Description                |
| ----- | ------ | ---------- | -------------------------- |
| `f32` | 32‑bit | ~7 digits  | Single precision           |
| `f64` | 64‑bit | ~15 digits | Double precision (default) |

```ferret
let pi: f32 = 3.14159;
let e: f64 = 2.718281828459045;
```

### String Type

Strings store text.

```ferret
let name: str = "Ferret";
let emoji: str = "🦦";
let multiline: str = "Hello
World";
```

### Boolean Type

Booleans store true or false.

```ferret
let is_active: bool = true;
let is_complete: bool = false;
```

### Character Type

Single Unicode characters.

```ferret
let letter: byte = 'A';
let symbol: byte = '♠';
let emoji: byte = '🎉';
```

## Compound Types

Types made from other types.

### Arrays

Arrays store a list of values that all have the same type.

Dynamic array:

```ferret
let numbers: []i32 = [1, 2, 3, 4, 5];
```

Fixed‑size array:

```ferret
let numbers: [5]i32 = [1, 2, 3, 4, 5];
```

## Optional Types

A type that may or may not contain a value.

```ferret
let maybe_number: i32? = 42;
let no_value: str? = none;
```

Learn more: [Optional Types](/language/optionals)

## Custom Types

### Type Aliases

A type alias gives a new name to an existing type.

```ferret
type UserId = i64;
type Email = str;

let user_id: UserId = 12345;
let email: Email = "user@example.com";
```

## Type Inference

Ferret can figure out types automatically when you use `:=`.

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
let y: i64 = x as i64;

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

* [Learn about Operators](/language/operators)
* [Explore Structs](/language/structs)
* [Understand Optional Types](/language/optionals)

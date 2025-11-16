---
title: Operators
description: Learn about operators in Ferret
---

Ferret provides a comprehensive set of operators for arithmetic, comparison, logical operations, and more.

## Arithmetic Operators

| Operator | Description | Example | Result |
|----------|-------------|---------|--------|
| `+` | Addition | `5 + 3` | `8` |
| `-` | Subtraction | `5 - 3` | `2` |
| `*` | Multiplication | `5 * 3` | `15` |
| `/` | Division | `15 / 3` | `5` |
| `%` | Modulo | `17 % 5` | `2` |

```ferret
let sum := 10 + 5;       // 15
let difference := 10 - 5; // 5
let product := 10 * 5;    // 50
let quotient := 10 / 5;   // 2
let remainder := 10 % 3;  // 1
```

## Comparison Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `==` | Equal to | `5 == 5` → `true` |
| `!=` | Not equal to | `5 != 3` → `true` |
| `<` | Less than | `3 < 5` → `true` |
| `>` | Greater than | `5 > 3` → `true` |
| `<=` | Less than or equal | `5 <= 5` → `true` |
| `>=` | Greater than or equal | `5 >= 3` → `true` |

```ferret
let is_equal := 10 == 10;     // true
let is_different := 10 != 5;  // true
let is_greater := 10 > 5;     // true
let is_less := 5 < 10;        // true
```

## Logical Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `and` | Logical AND | `true and false` → `false` |
| `or` | Logical OR | `true or false` → `true` |
| `not` | Logical NOT | `not true` → `false` |

```ferret
let result := true and false;   // false
let either := true or false;    // true
let negated := not true;        // false

// Short-circuit evaluation
let x := 5;
let safe := x != 0 and 10 / x > 1;  // Evaluates left to right
```

## Assignment Operators

| Operator | Description | Example | Equivalent |
|----------|-------------|---------|------------|
| `=` | Simple assignment | `x = 5` | - |
| `+=` | Add and assign | `x += 3` | `x = x + 3` |
| `-=` | Subtract and assign | `x -= 3` | `x = x - 3` |
| `*=` | Multiply and assign | `x *= 3` | `x = x * 3` |
| `/=` | Divide and assign | `x /= 3` | `x = x / 3` |
| `%=` | Modulo and assign | `x %= 3` | `x = x % 3` |

```ferret
let x := 10;
x += 5;   // x is now 15
x -= 3;   // x is now 12
x *= 2;   // x is now 24
x /= 4;   // x is now 6
```

## Special Operators

### Walrus Operator (`:=`)

Type inference declaration:

```ferret
let count := 42;        // Type inferred as i32
let name := "Ferret";   // Type inferred as str
```

### Elvis Operator (`?:`)

Provide default values for optionals:

```ferret
let maybe_value: i32? = none;
let value := maybe_value ?: 0;  // value is 0

let some_value: i32? = 42;
let result := some_value ?: 0;  // result is 42
```

Learn more about [Optional Types](/language/optionals).

### Range Operator (`..`)

Create ranges:

```ferret
let range := 1..5;       // [1, 2, 3, 4]
let inclusive := 1..=5;  // [1, 2, 3, 4, 5]
```

### Null Coalescing (`??`)

Chain multiple optional values:

```ferret
let a: i32? = none;
let b: i32? = none;
let c: i32? = 42;

let result := a ?? b ?? c ?? 0;  // result is 42
```

## Member Access Operators

### Dot Operator (`.`)

Access struct fields and methods:

```ferret
struct Point {
    .x: i32,
    .y: i32,
}

let p := Point{ .x = 10, .y = 20 };
let x := p.x;  // 10
```

### Optional Chaining (`?.`)

Safely access optional values:

```ferret
struct User {
    .name: str,
    .email: str?,
}

let user := User{ .name = "Alice", .email = none };
let email_len := user.email?.length;  // none (safe access)
```

## Bitwise Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `&` | Bitwise AND | `5 & 3` → `1` |
| `\|` | Bitwise OR | `5 \| 3` → `7` |
| `^` | Bitwise XOR | `5 ^ 3` → `6` |
| `<<` | Left shift | `5 << 1` → `10` |
| `>>` | Right shift | `5 >> 1` → `2` |

```ferret
let and_result := 0b1010 & 0b1100;  // 0b1000 (8)
let or_result := 0b1010 | 0b1100;   // 0b1110 (14)
let xor_result := 0b1010 ^ 0b1100;  // 0b0110 (6)
```

## Operator Precedence

From highest to lowest:

1. Member access (`.`, `?.`)
2. Unary (`not`, `-`, `+`)
3. Multiplication, Division, Modulo (`*`, `/`, `%`)
4. Addition, Subtraction (`+`, `-`)
5. Bit shifts (`<<`, `>>`)
6. Comparison (`<`, `>`, `<=`, `>=`)
7. Equality (`==`, `!=`)
8. Bitwise AND (`&`)
9. Bitwise XOR (`^`)
10. Bitwise OR (`|`)
11. Logical AND (`and`)
12. Logical OR (`or`)
13. Elvis operator (`?:`)
14. Assignment (`=`, `+=`, etc.)

Use parentheses to override precedence:

```ferret
let result := 2 + 3 * 4;      // 14 (multiplication first)
let explicit := (2 + 3) * 4;  // 20 (addition first)
```

## Next Steps

- [Learn about Control Flow](/language/if-statements)
- [Explore Functions](/language/functions)
- [Understand Optional Types](/language/optionals)

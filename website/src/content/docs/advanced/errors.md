---
title: Error Handling
description: Working with errors and result types in Ferret
---

Ferret uses explicit error handling with Error Types to manage failures safely.

## Result Types

Functions that can fail return `T ! E` (Error type). Here `T` is the normal return type, and `E` is the error type.

```ferret
fn divide(a: i32, b: i32) -> i32 ! str {
    if b == 0 {
        return "Division by zero"!; // Look at the `!` operator here. It constructs an error value.
    }
    return a / b;
}
```

You can skip the error type if you don't care about the specific error. Ferret will use `str` as the default error type.

```ferret
fn divide(a: i32, b: i32) -> i32 ! {
    if b == 0 {
        return "Division by zero"!;
    }
    return a / b;
}

## Handling Errors

Use match to handle success and error cases:

```ferret
let result := divide(10, 2); // You cannot call the function without handling the error
```
So you must handle it using `catch` clause:

```ferret
let result := divide(10, 0) catch e { // e holds the error value
    // Handle error case
    print("Error occurred: " + e);
    return; // Early return
};
```
If you want to provide a default value on error, you can use `catch` with a value:

```ferret
let result := divide(10, 0) catch e {
    // handle error case and provide default
} -1 ; // default value if error occurs, result will be -1
```
You either have to return so you don't continue with an invalid state, or provide a default value.

## Shorthand
You can just provide the default value directly:

```ferret
let result := divide(10, 0) catch -1; // result will be -1 on error
```

## Next Steps

- [Learn about Generics](/generics)
- [Explore Modules](/modules)

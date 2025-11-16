---
title: Enums
description: Enumeration types in Ferret
---

Enums (enumerations) define a type with a fixed set of possible values.

## Basic Enum

```ferret
enum Status {
    Pending,
    Active,
    Completed,
    Cancelled,
}
```

## Using Enums

```ferret
let status := Status::Active;

match status {
    Status::Pending => print("Waiting"),
    Status::Active => print("Running"),
    Status::Completed => print("Done"),
    Status::Cancelled => print("Aborted"),
}
```

## Enums with Values

```ferret
enum HttpStatus {
    Ok = 200,
    NotFound = 404,
    ServerError = 500,
}
```

## Tagged Unions

Enums can hold associated data:

```ferret
enum Result<T, E> {
    Ok(T),
    Err(E),
}

let result := Result::Ok(42);
```

## Pattern Matching

Extract data from enum variants:

```ferret
match result {
    Result::Ok(value) => print("Success: " + value),
    Result::Err(error) => print("Error: " + error),
}
```

## Next Steps

- [Learn about Interfaces](/language/interfaces)
- [Explore Match Expressions](/language/match)

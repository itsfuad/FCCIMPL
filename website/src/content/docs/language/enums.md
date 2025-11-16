---
title: Enums
description: Enumeration types in Ferret
---

Enums (enumerations) define a type with a fixed set of possible values.

## Basic Enum

```ferret
type Status enum {
    Pending,
    Active,
    Completed,
    Cancelled,
};
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
type HttpStatus enum {
    Ok = 200,
    NotFound = 404,
    ServerError = 500,
}
```

## Next Steps

- [Learn about Interfaces](/language/interfaces)
- [Explore Match Expressions](/language/match)

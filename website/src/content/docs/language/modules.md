---
title: Modules
description: Module system and code organization in Ferret
---

Modules help organize code into logical units.

## Module Declaration

```ferret
// In file: math.fer
module math;

pub fn add(a: i32, b: i32) -> i32 {
    return a + b;
}

fn internal_helper() {
    // Private to this module
}
```

## Importing Modules

```ferret
import math;

let sum := math::add(5, 3);
```

## Selective Imports

```ferret
import math::{add, subtract};

let result := add(10, 5);
```

## Nested Modules

```ferret
module utils {
    pub module string {
        pub fn capitalize(s: str) -> str {
            // Implementation
        }
    }
}

import utils::string;
let text := string::capitalize("hello");
```

## Visibility

Control what's exported:

```ferret
pub fn public_function() { }    // Exported
fn private_function() { }       // Internal only

pub struct PublicStruct { }
struct PrivateStruct { }
```

## Module Aliases

```ferret
import very::long::module::path as short;

short::function();
```

## Re-exports

```ferret
// In lib.fer
pub import math::add;     // Re-export from another module
pub import string::*;     // Re-export everything
```

## Next Steps

- [Explore the API Reference](/api/overview)
- [Read Best Practices](/guides/best-practices)

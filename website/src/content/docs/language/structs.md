---
title: Structs
description: Defining and using structs in Ferret
---

Structs are custom data types that group related data together.

## Struct Definition

```ferret
struct Person {
    name: str,
    age: i32,
    email: str,
}
```

## Creating Instances

```ferret
let person := Person{
    name: "Alice",
    age: 30,
    email: "alice@example.com",
};
```

## Accessing Fields

```ferret
print(person.name);   // Alice
print(person.age);    // 30
```

## Methods

Define methods on structs:

```ferret
impl Person {
    fn greet(self) -> str {
        return "Hello, I'm " + self.name;
    }
    
    fn is_adult(self) -> bool {
        return self.age >= 18;
    }
}

let greeting := person.greet();
```

## Mutable Fields

```ferret
let mut user := Person{
    name: "Bob",
    age: 25,
    email: "bob@example.com",
};

user.age = 26;  // OK with mut
```

## Next Steps

- [Learn about Enums](/language/enums)
- [Explore Interfaces](/language/interfaces)

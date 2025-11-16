---
title: Interfaces
description: Interface types and polymorphism in Ferret
---

Interfaces define contracts that types can implement.

## Interface Definition

```ferret
interface Drawable {
    fn draw(self);
    fn get_bounds(self) -> (i32, i32, i32, i32);
}
```

## Implementing Interfaces

```ferret
struct Circle {
    x: i32,
    y: i32,
    radius: i32,
}

impl Drawable for Circle {
    fn draw(self) {
        print("Drawing circle at (" + self.x + ", " + self.y + ")");
    }
    
    fn get_bounds(self) -> (i32, i32, i32, i32) {
        return (self.x - self.radius, self.y - self.radius,
                self.x + self.radius, self.y + self.radius);
    }
}
```

## Using Interfaces

```ferret
fn render(shape: Drawable) {
    shape.draw();
}

let circle := Circle{ x: 10, y: 20, radius: 5 };
render(circle);
```

## Multiple Interfaces

A type can implement multiple interfaces:

```ferret
impl Drawable for Circle { /* ... */ }
impl Serializable for Circle { /* ... */ }
```

## Next Steps

- [Learn about Generics](/language/generics)
- [Explore Error Handling](/language/errors)

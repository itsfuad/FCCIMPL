---
title: Interfaces
description: Interface types and polymorphism in Ferret
---

Interfaces define contracts that types can implement.

## Interface Definition

```ferret
type Drawable interface {
    draw();
    get_bounds() -> (i32, i32, i32, i32);
};
```

## Implementing Interfaces

```ferret
type Circle struct {
    .x: i32,
    .y: i32,
    .radius: i32,
};

fn (c: Circle) draw() {
    print("Drawing circle at (" + c.x + ", " + c.y + ")");
}

fn (c: Circle) get_bounds() -> (i32, i32, i32, i32) {
    return (c.x - c.radius, c.y - c.radius,
            c.x + c.radius, c.y + c.radius);
}
```

## Using Interfaces

```ferret
fn render(shape: Drawable) {
    shape.draw();
}

let circle := Circle{.x = 10, .y = 20, .radius = 5};
render(circle);
```

## Next Steps

- [Learn about Generics](/language/generics)
- [Explore Error Handling](/language/errors)

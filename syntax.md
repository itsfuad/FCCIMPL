# Ferret Keywords

## Core Declarations
```rs
let       // variable declaration
const     // immutable binding
type      // define a type
struct    // define a struct
fn        // define a function
interface // define an interface
enum{}      // define an enum
map[K]V       // define a map

//-----

// struct type
struct{
    .field1: Type1,
    .field2: Type2,
}    // define a struct

// struct instantiation
StructType{
    .field1 = Value1,
    .field2 = Value2,
}

// interface type
interface{
    fn method1(param: Type) -> ReturnType,
    fn method2() -> ReturnType,
}

// interface cannot be a literal, it must be implemented by a struct or type

// enum type
enum{
    Variant1,
    Variant2,
}    // define an enum

// enum usage
EnumType::Variant1 // all static symbols are accessed with ::, no dot notation (ex: module, enum)

//map type
map[KeyType]ValueType   // define a map
// map literal
map[KeyType]ValueType {
    Key1 => Value1,
    Key2 => Value2,
}

```

## Control Flow
```rs
if
else
for
while
break
continue
when      // replaces match
```

## Special Values
```ts
true
false
none
```

## Memory / Scope
```go
defer     // run when scope exits
```

## Modules / Imports
```ts
import
```

## Error Handling
```zig
catch     // catch block for T! functions
```

## Miscellaneous
```ts
as        // type cast
return
fork     // start coroutine
```

## Special Operators
```rs
?: // elvis operator. checks if the left side is none, then returns the right side. Only works with Optional types.
:= // walrus operator. used with variable or constant declarations to assign values without repeating the type.
```

**Notes:**
- Total keywords: 20

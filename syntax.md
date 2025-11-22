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
```

```rs

// struct type
struct{
    .field1: Type1,
    .field2: Type2,
}    // define a struct

// Anonymous struct literal (type inferred from fields)
{
    .field1 = Value1,
    .field2 = Value2,
}

// Anonymous struct with explicit type cast
{
    .field1 = Value1,
    .field2 = Value2,
} as StructType

// struct literal with type annotation
let x: StructType = {
    .field1 = Value1,
    .field2 = Value2,
};

// Shorthand with walrus operator and type cast
let x := {
    .field1 = Value1,
    .field2 = Value2,
} as StructType;

// named struct type
type StructType struct{
    .field1: Type1,
    .field2: Type2,
};

// interface type
interface{
    fn method1(param: Type) -> ReturnType,
    fn method2() -> ReturnType,
}

// named interface type
type InterfaceType interface{
    fn method1(param: Type) -> ReturnType,
    fn method2() -> ReturnType,
};

// interface cannot be a literal, it must be implemented by a struct or type

// enum type
enum{
    Variant1,
    Variant2,
}    // define an enum

// enum instantiation
enum{
    Variant1,
    Variant2,
}::Variant1 // access enum variant

// named enum type
type EnumType enum{
    Variant1,
    Variant2,
};

// enum usage
EnumType::Variant1 // all static symbols are accessed with ::, no dot notation (ex: module, enum)

// map type
map[KeyType]ValueType   // define a map

// Anonymous map literal (type inferred from key/value types)
{
    Key1 => Value1,
    Key2 => Value2,
}

// Anonymous map with explicit type cast
{
    "key1" => 10,
    "key2" => 20,
} as map[str]i32

// Map literal with type annotation
let m: map[str]i32 = {
    "key1" => 10,
    "key2" => 20,
};

// Shorthand with walrus operator and type cast
let m := {
    "a" => 1,
    "b" => 2,
} as map[str]i32;

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

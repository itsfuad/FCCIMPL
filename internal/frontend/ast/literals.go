package ast

import "compiler/internal/source"

type LiteralKind int

const (
	INT LiteralKind = iota
	FLOAT
	STRING
	BYTE
	BOOL
	NONE // represents the 'none' keyword
)

// BasicLit represents a literal of basic type (int, float, string, bool, none)
type BasicLit struct {
	Kind  LiteralKind
	Value string // the literal value as a string
	source.Location
}

func (b *BasicLit) INode()                {} // Implements Node interface
func (b *BasicLit) Expr()                 {} // Expr is a marker interface for all expressions
func (b *BasicLit) Loc() *source.Location { return &b.Location }

// CompositeLit represents a composite literal (array, struct, map, enum)
// Examples: []i32{1, 2, 3}, Point{.x = 1, .y = 2}, map[str]i32{"a" => 1}
type CompositeLit struct {
	Type TypeNode     // type of the composite literal (can be nil for inferred types)
	Elts []Expression // list of composite elements
	source.Location
}

func (c *CompositeLit) INode()                {} // Implements Node interface
func (c *CompositeLit) Expr()                 {} // Expr is a marker interface for all expressions
func (c *CompositeLit) Loc() *source.Location { return &c.Location }

// KeyValueExpr represents a key-value pair in a composite literal
// Used for struct fields (.field = value) and map entries (key => value)
type KeyValueExpr struct {
	Key   Expression // field name or map key
	Value Expression // field value or map value
	source.Location
}

func (k *KeyValueExpr) INode()                {} // Implements Node interface
func (k *KeyValueExpr) Expr()                 {} // Expr is a marker interface for all expressions
func (k *KeyValueExpr) Loc() *source.Location { return &k.Location }

// FuncLit represents a function literal (anonymous function/lambda)
type FuncLit struct {
	Type *FuncType // function signature
	Body *Block    // function body
	source.Location
}

func (f *FuncLit) INode()                {} // Implements Node interface
func (f *FuncLit) Expr()                 {} // Expr is a marker interface for all expressions
func (f *FuncLit) Loc() *source.Location { return &f.Location }

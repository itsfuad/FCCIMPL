package checker

import (
	"compiler/internal/context"
	"compiler/internal/diagnostics"
	"compiler/internal/frontend/ast"
	"compiler/internal/semantics"
	"compiler/internal/source"
	"fmt"
)

// Checker performs type checking and validation (Pass 3 / Phase 5)
// Validates type compatibility, infers types, checks assignments
type Checker struct {
	ctx          *context.CompilerContext
	currentScope *semantics.SymbolTable
	currentFile  string
}

// New creates a new type checker
func New(ctx *context.CompilerContext) *Checker {
	return &Checker{
		ctx: ctx,
	}
}

// Run executes Pass 3: Type Checking for all files
func Run(ctx *context.CompilerContext) {
	checker := New(ctx)

	for _, file := range ctx.GetAllFiles() {
		checker.CheckFile(file)
	}
}

// CheckFile performs type checking for a single source file
func (c *Checker) CheckFile(file *context.SourceFile) {
	c.currentFile = file.Path
	c.currentScope = file.Scope

	if file.AST == nil || file.Scope == nil {
		return
	}

	// Walk the AST and check types
	c.checkModule(file.AST)
}

// checkModule checks all declarations in a module
func (c *Checker) checkModule(module *ast.Module) {
	for _, node := range module.Nodes {
		c.checkNode(node)
	}
}

// checkNode type-checks a single AST node
func (c *Checker) checkNode(node ast.Node) semantics.Type {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ast.VarDecl:
		return c.checkVarDecl(n)
	case *ast.ConstDecl:
		return c.checkConstDecl(n)
	case *ast.AssignStmt:
		return c.checkAssignStmt(n)
	case *ast.FuncDecl:
		return c.checkFuncDecl(n)
	case *ast.Block:
		return c.checkBlock(n)
	case *ast.IfStmt:
		return c.checkIfStmt(n)
	case *ast.ReturnStmt:
		return c.checkReturnStmt(n)
	case *ast.ExprStmt:
		return c.checkExpr(n.X)
	// Add more node types as needed
	default:
		return nil
	}
}

// checkVarDecl checks variable declarations and initializers
func (c *Checker) checkVarDecl(decl *ast.VarDecl) semantics.Type {
	for _, item := range decl.Decls {
		// Look up the symbol
		sym, ok := c.currentScope.Lookup(item.Name.Name)
		if !ok {
			continue // Symbol not found (error already reported in collector)
		}

		// If there's an initializer, check it
		if item.Value != nil {
			valueType := c.checkExpr(item.Value)

			// If type was explicitly specified, check compatibility
			if sym.Type != nil {
				if !c.isAssignable(sym.Type, valueType) {
					c.ctx.Diagnostics.Add(
						diagnostics.NewError(
							fmt.Sprintf("cannot assign value of type %s to variable of type %s",
								c.typeString(valueType),
								c.typeString(sym.Type)),
						).
							WithCode(diagnostics.ErrTypeMismatch).
							WithPrimaryLabel(c.currentFile, item.Value.Loc(),
								fmt.Sprintf("type %s", c.typeString(valueType))).
							WithSecondaryLabel(c.currentFile, item.Name.Loc(),
								fmt.Sprintf("variable has type %s", c.typeString(sym.Type))),
					)
				}
			} else {
				// Type inference: set symbol type from initializer
				sym.Type = valueType
			}
		}
	}

	return nil
}

// checkConstDecl checks constant declarations and initializers
func (c *Checker) checkConstDecl(decl *ast.ConstDecl) semantics.Type {
	for _, item := range decl.Decls {
		// Look up the symbol
		sym, ok := c.currentScope.Lookup(item.Name.Name)
		if !ok {
			continue // Symbol not found (error already reported in collector)
		}

		// Constants must have an initializer (checked in parser)
		if item.Value != nil {
			valueType := c.checkExpr(item.Value)

			// If type was explicitly specified, check compatibility
			if sym.Type != nil {
				if !c.isAssignable(sym.Type, valueType) {
					c.ctx.Diagnostics.Add(
						diagnostics.NewError(
							fmt.Sprintf("cannot assign value of type %s to constant of type %s",
								c.typeString(valueType),
								c.typeString(sym.Type)),
						).
							WithCode(diagnostics.ErrTypeMismatch).
							WithPrimaryLabel(c.currentFile, item.Value.Loc(),
								fmt.Sprintf("type %s", c.typeString(valueType))).
							WithSecondaryLabel(c.currentFile, item.Name.Loc(),
								fmt.Sprintf("constant has type %s", c.typeString(sym.Type))),
					)
				}
			} else {
				// Type inference: set symbol type from initializer
				sym.Type = valueType
			}
		}
	}

	return nil
}

// checkAssignStmt checks assignment statements
func (c *Checker) checkAssignStmt(stmt *ast.AssignStmt) semantics.Type {
	// Check LHS and RHS types
	lhsType := c.checkExpr(stmt.Lhs)
	rhsType := c.checkExpr(stmt.Rhs)

	// Check if assignment is valid
	if lhsType != nil && rhsType != nil {
		if !c.isAssignable(lhsType, rhsType) {
			c.ctx.Diagnostics.Add(
				diagnostics.NewError(
					fmt.Sprintf("cannot assign value of type %s to variable of type %s",
						c.typeString(rhsType),
						c.typeString(lhsType)),
				).
					WithCode(diagnostics.ErrTypeMismatch).
					WithPrimaryLabel(c.currentFile, stmt.Rhs.Loc(),
						fmt.Sprintf("type %s", c.typeString(rhsType))).
					WithSecondaryLabel(c.currentFile, stmt.Lhs.Loc(),
						fmt.Sprintf("target has type %s", c.typeString(lhsType))),
			)
		}
	}

	// Check if LHS is a constant (not allowed to reassign)
	if ident, ok := stmt.Lhs.(*ast.IdentifierExpr); ok {
		if sym, found := c.currentScope.Lookup(ident.Name); found {
			if sym.Kind == semantics.SymbolConst {
				diag := diagnostics.NewError(
					fmt.Sprintf("cannot assign to constant '%s'", ident.Name),
				).
					WithCode(diagnostics.ErrConstantReassignment).
					WithPrimaryLabel(c.currentFile, stmt.Lhs.Loc(), "cannot assign to constant").
					WithHelp("constants cannot be reassigned after declaration")

				// Add secondary label pointing to where the constant was declared
				if sym.Decl != nil {
					if constDecl, ok := sym.Decl.(*ast.ConstDecl); ok {
						// Find the specific declaration item for this constant
						for _, item := range constDecl.Decls {
							if item.Name.Name == ident.Name {
								// Create a location that spans the full identifier name
								nameLoc := item.Name.Loc()
								fullLoc := &source.Location{
									Start: nameLoc.Start,
									End: &source.Position{
										Line:   nameLoc.Start.Line,
										Column: nameLoc.Start.Column + len(ident.Name),
										Index:  nameLoc.Start.Index + len(ident.Name),
									},
								}
								diag.WithSecondaryLabel(c.currentFile, fullLoc,
									fmt.Sprintf("constant '%s' declared here", ident.Name))
								break
							}
						}
					}
				}

				c.ctx.Diagnostics.Add(diag)
			}
		}
	}

	return nil
}

// checkFuncDecl checks a function declaration
func (c *Checker) checkFuncDecl(decl *ast.FuncDecl) semantics.Type {
	// Get function symbol
	sym, ok := c.currentScope.Lookup(decl.Name.Name)
	if !ok || sym.SelfScope == nil {
		return nil
	}

	// Check function body in function scope
	prevScope := c.currentScope
	c.currentScope = sym.SelfScope

	if decl.Body != nil {
		c.checkBlock(decl.Body)
	}

	c.currentScope = prevScope

	return sym.Type
}

// checkBlock checks a block of statements
func (c *Checker) checkBlock(block *ast.Block) semantics.Type {
	// Switch to the block's scope if it exists
	prevScope := c.currentScope
	if blockScope := c.ctx.GetBlockScope(block); blockScope != nil {
		c.currentScope = blockScope
	}

	var lastType semantics.Type
	for _, node := range block.Nodes {
		lastType = c.checkNode(node)
	}

	c.currentScope = prevScope
	return lastType
}

// checkIfStmt checks an if statement
func (c *Checker) checkIfStmt(stmt *ast.IfStmt) semantics.Type {
	// Check condition
	condType := c.checkExpr(stmt.Cond)

	// Condition should be boolean (we can add this check later)
	_ = condType

	// Check then branch
	if stmt.Body != nil {
		c.checkBlock(stmt.Body)
	}

	// Check else branch
	if stmt.Else != nil {
		switch e := stmt.Else.(type) {
		case *ast.Block:
			c.checkBlock(e)
		case *ast.IfStmt:
			c.checkIfStmt(e)
		}
	}

	return nil
}

// checkReturnStmt checks a return statement
func (c *Checker) checkReturnStmt(stmt *ast.ReturnStmt) semantics.Type {
	if stmt.Result != nil {
		return c.checkExpr(stmt.Result)
	}
	return nil
}

// checkExpr checks an expression and returns its type
func (c *Checker) checkExpr(expr ast.Expression) semantics.Type {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *ast.BasicLit:
		return c.checkBasicLit(e)
	case *ast.IdentifierExpr:
		return c.checkIdentifier(e)
	case *ast.BinaryExpr:
		return c.checkBinaryExpr(e)
	case *ast.UnaryExpr:
		return c.checkUnaryExpr(e)
	case *ast.CallExpr:
		return c.checkCallExpr(e)
	case *ast.SelectorExpr:
		return c.checkSelectorExpr(e)
	case *ast.CompositeLit:
		return c.checkCompositeLit(e)
	// Add more expression types as needed
	default:
		return nil
	}
}

// checkBasicLit infers the type of a literal
func (c *Checker) checkBasicLit(lit *ast.BasicLit) semantics.Type {
	switch lit.Kind {
	case ast.INT:
		// Default integer type is i32
		return &semantics.PrimitiveType{TypeName: "i32"}
	case ast.FLOAT:
		// Default float type is f64
		return &semantics.PrimitiveType{TypeName: "f64"}
	case ast.STRING:
		return &semantics.PrimitiveType{TypeName: "string"}
	case ast.BOOL:
		return &semantics.PrimitiveType{TypeName: "bool"}
	default:
		return nil
	}
}

// checkIdentifier checks an identifier expression
func (c *Checker) checkIdentifier(ident *ast.IdentifierExpr) semantics.Type {
	sym, ok := c.currentScope.Lookup(ident.Name)
	if !ok {
		c.ctx.Diagnostics.Add(
			diagnostics.UndefinedSymbol(c.currentFile, ident.Loc(), ident.Name),
		)
		return &semantics.Invalid{}
	}

	return sym.Type
}

// checkBinaryExpr checks a binary expression
func (c *Checker) checkBinaryExpr(expr *ast.BinaryExpr) semantics.Type {
	leftType := c.checkExpr(expr.X)
	rightType := c.checkExpr(expr.Y)

	// For now, return left type (simple approximation)
	// TODO: Add proper operator type checking
	_ = rightType

	return leftType
}

// checkUnaryExpr checks a unary expression
func (c *Checker) checkUnaryExpr(expr *ast.UnaryExpr) semantics.Type {
	return c.checkExpr(expr.X)
}

// checkCallExpr checks a function call expression
func (c *Checker) checkCallExpr(expr *ast.CallExpr) semantics.Type {
	funcType := c.checkExpr(expr.Fun)

	// Check if it's a function type
	if ft, ok := funcType.(*semantics.FunctionType); ok {
		// Check argument count and types
		// TODO: Implement argument checking
		return ft.ReturnType
	}

	return nil
}

// checkSelectorExpr checks a field access expression
func (c *Checker) checkSelectorExpr(expr *ast.SelectorExpr) semantics.Type {
	baseType := c.checkExpr(expr.X)

	// Check if base type has the field
	if structType, ok := baseType.(*semantics.StructType); ok {
		fieldType := structType.GetFieldType(expr.Sel.Name)
		if fieldType == nil {
			c.ctx.Diagnostics.Add(
				diagnostics.FieldNotFound(
					c.currentFile,
					expr.Sel.Loc(),
					expr.Sel.Name,
					c.typeString(baseType),
				),
			)
			return &semantics.Invalid{}
		}
		return fieldType
	}

	return nil
}

// checkCompositeLit checks a composite literal
func (c *Checker) checkCompositeLit(lit *ast.CompositeLit) semantics.Type {
	// If type is specified, resolve it
	if lit.Type != nil {
		// TODO: Convert AST type to semantic type
		// For now, return a placeholder
		return nil
	}
	return nil
}

// isAssignable checks if a value of type 'from' can be assigned to type 'to'
func (c *Checker) isAssignable(to, from semantics.Type) bool {
	if to == nil || from == nil {
		return false
	}

	// Check for invalid types
	if _, ok := to.(*semantics.Invalid); ok {
		return false
	}
	if _, ok := from.(*semantics.Invalid); ok {
		return false
	}

	// Exact type match
	if to.String() == from.String() {
		return true
	}

	// Check primitive type compatibility
	toPrim, toIsPrim := to.(*semantics.PrimitiveType)
	fromPrim, fromIsPrim := from.(*semantics.PrimitiveType)

	if toIsPrim && fromIsPrim {
		// Allow compatible primitive types
		// For now, require exact match
		// TODO: Add numeric type conversions (i32 -> i64, etc.)
		return toPrim.TypeName == fromPrim.TypeName
	}

	// TODO: Add more type compatibility rules
	// - Interface implementation
	// - Struct compatibility
	// - Array element type matching

	return false
}

// typeString returns a human-readable string for a type
func (c *Checker) typeString(t semantics.Type) string {
	if t == nil {
		return "unknown"
	}
	return t.String()
}

package checker

import (
	"compiler/internal/context"
	"compiler/internal/diagnostics"
	"compiler/internal/frontend/ast"
	"compiler/internal/semantics"
	"compiler/internal/source"
	"compiler/internal/types"
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
					c.reportAssignmentError(sym.Type, valueType, item.Value.Loc(), item.Name.Loc())
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
			c.reportAssignmentError(lhsType, rhsType, stmt.Rhs.Loc(), stmt.Lhs.Loc())
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

// checkIfStmt checks an if statement with type narrowing
func (c *Checker) checkIfStmt(stmt *ast.IfStmt) semantics.Type {
	// Check condition
	condType := c.checkExpr(stmt.Cond)

	// Condition should be boolean (we can add this check later)
	_ = condType

	// Analyze condition for type narrowing
	thenNarrowings, elseNarrowings := c.analyzeTypeNarrowing(stmt.Cond)

	// Check then branch with narrowed types
	if stmt.Body != nil {
		// Get the block's scope to apply narrowings there
		blockScope := c.ctx.GetBlockScope(stmt.Body)
		if blockScope != nil {
			c.applyNarrowingsInScope(blockScope, thenNarrowings, func() {
				c.checkBlock(stmt.Body)
			})
		} else {
			c.checkBlock(stmt.Body)
		}
	}

	// Check else branch with narrowed types
	if stmt.Else != nil {
		switch e := stmt.Else.(type) {
		case *ast.Block:
			// Get the block's scope to apply narrowings there
			blockScope := c.ctx.GetBlockScope(e)
			if blockScope != nil {
				c.applyNarrowingsInScope(blockScope, elseNarrowings, func() {
					c.checkBlock(e)
				})
			} else {
				c.checkBlock(e)
			}
		case *ast.IfStmt:
			// For else-if, apply narrowings in current scope
			c.applyNarrowings(elseNarrowings, func() {
				c.checkIfStmt(e)
			})
		}
	}

	return nil
}

// TypeNarrowing represents a type refinement for a variable
type TypeNarrowing struct {
	SymbolName   string
	NarrowedType semantics.Type
}

// analyzeTypeNarrowing analyzes a condition and returns type narrowings for then/else branches
// For example: if (x != none) narrows x to T in then branch, and to none in else branch
func (c *Checker) analyzeTypeNarrowing(cond ast.Expression) (thenNarrowings, elseNarrowings []TypeNarrowing) {
	// Check if condition is a binary expression
	binExpr, ok := cond.(*ast.BinaryExpr)
	if !ok {
		return nil, nil
	}

	// Check if one side is an identifier and the other is 'none'
	var identExpr *ast.IdentifierExpr
	var isNoneCheck bool
	var isEqualityCheck bool
	var isNotEqualCheck bool

	// Determine operator type
	switch binExpr.Op.Kind {
	case "==":
		isEqualityCheck = true
	case "!=":
		isNotEqualCheck = true
	default:
		return nil, nil // Not a comparison we handle
	}

	// Check if comparing with 'none'
	if ident, ok := binExpr.X.(*ast.IdentifierExpr); ok {
		if noneIdent, ok := binExpr.Y.(*ast.IdentifierExpr); ok && noneIdent.Name == "none" {
			identExpr = ident
			isNoneCheck = true
		}
	} else if ident, ok := binExpr.Y.(*ast.IdentifierExpr); ok {
		if noneIdent, ok := binExpr.X.(*ast.IdentifierExpr); ok && noneIdent.Name == "none" {
			identExpr = ident
			isNoneCheck = true
		}
	}

	if !isNoneCheck || identExpr == nil {
		return nil, nil
	}

	// Look up the symbol
	sym, ok := c.currentScope.Lookup(identExpr.Name)
	if !ok || sym.Type == nil {
		return nil, nil
	}

	// Check if symbol has optional type
	optType, isOptional := sym.Type.(*semantics.OptionalType)
	if !isOptional {
		return nil, nil
	}

	// Create narrowings based on the operator
	if isNotEqualCheck {
		// x != none: then branch has T (not none), else branch has none
		thenNarrowings = []TypeNarrowing{
			{SymbolName: identExpr.Name, NarrowedType: optType.Base},
		}
		elseNarrowings = []TypeNarrowing{
			{SymbolName: identExpr.Name, NarrowedType: &semantics.NoneType{}},
		}
	} else if isEqualityCheck {
		// x == none: then branch has none, else branch has T (not none)
		thenNarrowings = []TypeNarrowing{
			{SymbolName: identExpr.Name, NarrowedType: &semantics.NoneType{}},
		}
		elseNarrowings = []TypeNarrowing{
			{SymbolName: identExpr.Name, NarrowedType: optType.Base},
		}
	}

	return thenNarrowings, elseNarrowings
}

// applyNarrowings temporarily narrows types in the current scope and executes the function
func (c *Checker) applyNarrowings(narrowings []TypeNarrowing, fn func()) {
	if len(narrowings) == 0 {
		fn()
		return
	}

	// Save original types
	originalTypes := make(map[string]semantics.Type)
	for _, narrowing := range narrowings {
		if sym, ok := c.currentScope.Lookup(narrowing.SymbolName); ok {
			originalTypes[narrowing.SymbolName] = sym.Type
			// Temporarily narrow the type
			sym.Type = narrowing.NarrowedType
		}
	}

	// Execute the function with narrowed types
	fn()

	// Restore original types
	for name, originalType := range originalTypes {
		if sym, ok := c.currentScope.Lookup(name); ok {
			sym.Type = originalType
		}
	}
}

// applyNarrowingsInScope temporarily narrows types in a specific scope and executes the function
func (c *Checker) applyNarrowingsInScope(scope *semantics.SymbolTable, narrowings []TypeNarrowing, fn func()) {
	if len(narrowings) == 0 {
		fn()
		return
	}

	// Save original types
	originalTypes := make(map[string]semantics.Type)
	for _, narrowing := range narrowings {
		if sym, ok := scope.Lookup(narrowing.SymbolName); ok {
			originalTypes[narrowing.SymbolName] = sym.Type
			// Temporarily narrow the type
			sym.Type = narrowing.NarrowedType
		}
	}

	// Execute the function with narrowed types
	fn()

	// Restore original types
	for name, originalType := range originalTypes {
		if sym, ok := scope.Lookup(name); ok {
			sym.Type = originalType
		}
	}
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
	case *ast.ElvisExpr:
		return c.checkElvisExpr(e)
	case *ast.CastExpr:
		return c.checkCastExpr(e)
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
		return &semantics.PrimitiveType{TypeName: types.TYPE_I32}
	case ast.FLOAT:
		// Default float type is f64
		return &semantics.PrimitiveType{TypeName: types.TYPE_F64}
	case ast.STRING:
		return &semantics.PrimitiveType{TypeName: types.TYPE_STRING}
	case ast.BOOL:
		return &semantics.PrimitiveType{TypeName: types.TYPE_BOOL}
	case ast.NONE:
		// 'none' is a special type that can be assigned to any optional type
		// We represent it as a special marker type
		return &semantics.NoneType{}
	default:
		return nil
	}
}

// checkIdentifier checks an identifier expression
func (c *Checker) checkIdentifier(ident *ast.IdentifierExpr) semantics.Type {
	// Handle 'none' as a special built-in value
	if ident.Name == "none" {
		return &semantics.NoneType{}
	}

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

		returnType := ft.ReturnType

		// Handle catch clause if present
		if expr.Catch != nil {
			returnType = c.checkCatchClause(expr.Catch, returnType)
		}

		return returnType
	}

	return nil
}

// checkCatchClause validates a catch clause and returns the effective return type
func (c *Checker) checkCatchClause(catch *ast.CatchClause, funcReturnType semantics.Type) semantics.Type {
	if catch == nil {
		return funcReturnType
	}

	// The function must return an error type (T ! E) to use catch
	errorType, isErrorType := funcReturnType.(*semantics.ErrorType)
	if !isErrorType {
		// Report error: catch can only be used with functions that return error types
		c.ctx.Diagnostics.Add(
			diagnostics.NewError(
				"catch can only be used with functions that return error types (T ! E)",
			).
				WithCode(diagnostics.ErrTypeMismatch).
				WithPrimaryLabel(c.currentFile, catch.Loc(), "catch clause used here"),
		)
		return &semantics.Invalid{}
	}

	// If there's an error identifier, add it to the scope within the handler
	if catch.ErrIdent != nil && catch.Handler != nil {
		// Create a new scope for the handler block
		handlerScope := semantics.NewSymbolTable(c.currentScope)
		prevScope := c.currentScope
		c.currentScope = handlerScope

		// Add the error variable to the handler scope
		errorSym := &semantics.Symbol{
			Name: catch.ErrIdent.Name,
			Type: errorType.Error,
			Kind: semantics.SymbolVar,
		}
		handlerScope.Declare(catch.ErrIdent.Name, errorSym)

		// Check the handler block
		c.checkBlock(catch.Handler)

		// Restore the previous scope
		c.currentScope = prevScope
	} else if catch.Handler != nil {
		// Handler without error identifier - just check the block
		c.checkBlock(catch.Handler)
	}

	// Check the fallback expression if present
	if catch.Fallback != nil {
		fallbackType := c.checkExpr(catch.Fallback)

		// The fallback must be assignable to the valid type
		if !c.isAssignable(errorType.Valid, fallbackType) {
			c.ctx.Diagnostics.Add(
				diagnostics.NewError(
					fmt.Sprintf("catch fallback type '%s' is not assignable to expected type '%s'",
						c.typeString(fallbackType),
						c.typeString(errorType.Valid)),
				).
					WithCode(diagnostics.ErrTypeMismatch).
					WithPrimaryLabel(c.currentFile, catch.Fallback.Loc(),
						fmt.Sprintf("expression has type %s", c.typeString(fallbackType))).
					WithSecondaryLabel(c.currentFile, catch.Loc(),
						fmt.Sprintf("expected type %s", c.typeString(errorType.Valid))),
			)
		}
	}

	// After catch, the effective return type is the valid type (error is handled)
	return errorType.Valid
}

// checkElvisExpr checks an elvis operator expression (a ?: b)
func (c *Checker) checkElvisExpr(expr *ast.ElvisExpr) semantics.Type {
	condType := c.checkExpr(expr.Cond)
	defaultType := c.checkExpr(expr.Default)

	// The condition should be an optional type
	optionalType, isOptional := condType.(*semantics.OptionalType)
	if !isOptional {
		c.ctx.Diagnostics.Add(
			diagnostics.NewError(
				"elvis operator (?:) is only used with optional types",
			).
				WithPrimaryLabel(c.currentFile, expr.Cond.Loc(),
					fmt.Sprintf("expression has type %s", c.typeString(condType))),
		)
		return condType
	}

	// The default value must be assignable to the base type of the optional
	if !c.isAssignable(optionalType.Base, defaultType) {
		c.ctx.Diagnostics.Add(
			diagnostics.NewError(
				fmt.Sprintf("elvis default value type '%s' is not compatible with optional base type '%s'",
					c.typeString(defaultType),
					c.typeString(optionalType.Base)),
			).
				WithCode(diagnostics.ErrTypeMismatch).
				WithPrimaryLabel(c.currentFile, expr.Default.Loc(),
					fmt.Sprintf("expression has type %s", c.typeString(defaultType))).
				WithSecondaryLabel(c.currentFile, expr.Cond.Loc(),
					fmt.Sprintf("expression has type %s", c.typeString(condType))),
		)
		return &semantics.Invalid{}
	}

	// The result type is the base type (unwrapped) since we provide a default
	return optionalType.Base
}

// checkSelectorExpr checks a field access expression
func (c *Checker) checkSelectorExpr(expr *ast.SelectorExpr) semantics.Type {
	baseType := c.checkExpr(expr.X)

	// Handle struct field access
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

	// Handle method access on user-defined types
	if userType, ok := baseType.(*semantics.UserType); ok {
		// First check if the underlying definition has this field
		if userType.Definition != nil {
			if structType, ok := userType.Definition.(*semantics.StructType); ok {
				fieldType := structType.GetFieldType(expr.Sel.Name)
				if fieldType != nil {
					return fieldType
				}
			}
		}

		// Check if it's a method
		if methodType, ok := userType.Methods[expr.Sel.Name]; ok {
			return methodType
		}

		// Neither field nor method found
		c.ctx.Diagnostics.Add(
			diagnostics.NewError(
				fmt.Sprintf("type '%s' has no field or method '%s'",
					c.typeString(baseType),
					expr.Sel.Name)).
				WithCode(diagnostics.ErrFieldNotFound).
				WithPrimaryLabel(c.currentFile, expr.Sel.Loc(),
					fmt.Sprintf("'%s' not found", expr.Sel.Name)),
		)
		return &semantics.Invalid{}
	}

	return nil
}

// checkCompositeLit checks a composite literal
func (c *Checker) checkCompositeLit(lit *ast.CompositeLit) semantics.Type {
	// If type is specified, resolve it
	if lit.Type != nil {
		targetType := c.astTypeToSemanticType(lit.Type)
		c.checkCompositeLitElements(lit, targetType)
		return targetType
	}

	// Type inference based on elements
	if len(lit.Elts) == 0 {
		c.ctx.Diagnostics.Add(
			diagnostics.NewError("cannot infer type of empty composite literal").
				WithCode(diagnostics.ErrTypeMismatch).
				WithPrimaryLabel(c.currentFile, lit.Loc(), "add type annotation or elements").
				WithHelp("use {} as Type to specify the type"),
		)
		return &semantics.Invalid{}
	}

	// Check first element to determine if it's a struct, map, or array
	firstElem := lit.Elts[0]

	if kvExpr, ok := firstElem.(*ast.KeyValueExpr); ok {
		// Key-value syntax - could be struct or map
		// If all elements are KeyValueExpr, check the first key type
		// Structs will have identifiers as keys (from .field syntax)
		// Maps will have expression keys

		// Since parser creates identifiers for .field syntax,
		// we need another way to distinguish. We'll check if this is being
		// assigned to a known type or look at the syntax context.
		// For now, we'll infer as struct if keys are simple identifiers
		// and as map otherwise.

		if ident, ok := kvExpr.Key.(*ast.IdentifierExpr); ok {
			// Simple identifier - likely a struct field from .field syntax
			_ = ident
			return c.inferStructType(lit)
		}

		// Complex expression key - it's a map
		return c.inferMapType(lit)
	}

	// Array literal: {val1, val2, val3}
	return c.inferArrayType(lit)
}

// inferStructType infers an anonymous struct type from composite literal
func (c *Checker) inferStructType(lit *ast.CompositeLit) semantics.Type {
	fields := make(map[string]semantics.Type)

	for _, elem := range lit.Elts {
		kvExpr, ok := elem.(*ast.KeyValueExpr)
		if !ok {
			c.ctx.Diagnostics.Add(
				diagnostics.NewError("struct literal must use field syntax: .field = value").
					WithCode(diagnostics.ErrTypeMismatch).
					WithPrimaryLabel(c.currentFile, elem.Loc(), "expected .field = value"),
			)
			continue
		}

		// Key should be an identifier (field name) for struct literals
		ident, ok := kvExpr.Key.(*ast.IdentifierExpr)
		if !ok {
			c.ctx.Diagnostics.Add(
				diagnostics.NewError("struct field must be an identifier").
					WithCode(diagnostics.ErrTypeMismatch).
					WithPrimaryLabel(c.currentFile, kvExpr.Key.Loc(), "expected field name"),
			)
			continue
		}

		fieldName := ident.Name

		// Check for duplicate fields
		if _, exists := fields[fieldName]; exists {
			c.ctx.Diagnostics.Add(
				diagnostics.NewError(fmt.Sprintf("duplicate field '%s' in struct literal", fieldName)).
					WithCode(diagnostics.ErrTypeMismatch).
					WithPrimaryLabel(c.currentFile, kvExpr.Key.Loc(), "duplicate field"),
			)
			continue
		}

		// Infer type from value
		valueType := c.checkExpr(kvExpr.Value)
		if valueType != nil {
			fields[fieldName] = valueType
		}
	}

	return &semantics.StructType{Fields: fields}
}

// inferMapType infers a map type from composite literal
func (c *Checker) inferMapType(lit *ast.CompositeLit) semantics.Type {
	if len(lit.Elts) == 0 {
		return &semantics.Invalid{}
	}

	var keyType, valueType semantics.Type

	for i, elem := range lit.Elts {
		kvExpr, ok := elem.(*ast.KeyValueExpr)
		if !ok {
			c.ctx.Diagnostics.Add(
				diagnostics.NewError("map literal must use key => value syntax").
					WithCode(diagnostics.ErrTypeMismatch).
					WithPrimaryLabel(c.currentFile, elem.Loc(), "expected key => value"),
			)
			continue
		}

		elemKeyType := c.checkExpr(kvExpr.Key)
		elemValueType := c.checkExpr(kvExpr.Value)

		if i == 0 {
			// First element determines the map type
			keyType = elemKeyType
			valueType = elemValueType
		} else {
			// Subsequent elements must match the inferred types
			if !c.isAssignable(keyType, elemKeyType) {
				c.ctx.Diagnostics.Add(
					diagnostics.NewError(
						fmt.Sprintf("map key type mismatch: expected %s, got %s",
							c.typeString(keyType),
							c.typeString(elemKeyType))).
						WithCode(diagnostics.ErrTypeMismatch).
						WithPrimaryLabel(c.currentFile, kvExpr.Key.Loc(),
							fmt.Sprintf("type %s", c.typeString(elemKeyType))).
						WithSecondaryLabel(c.currentFile, lit.Elts[0].Loc(),
							fmt.Sprintf("first element has key type %s", c.typeString(keyType))),
				)
			}

			if !c.isAssignable(valueType, elemValueType) {
				c.ctx.Diagnostics.Add(
					diagnostics.NewError(
						fmt.Sprintf("map value type mismatch: expected %s, got %s",
							c.typeString(valueType),
							c.typeString(elemValueType))).
						WithCode(diagnostics.ErrTypeMismatch).
						WithPrimaryLabel(c.currentFile, kvExpr.Value.Loc(),
							fmt.Sprintf("type %s", c.typeString(elemValueType))).
						WithSecondaryLabel(c.currentFile, lit.Elts[0].Loc(),
							fmt.Sprintf("first element has value type %s", c.typeString(valueType))),
				)
			}
		}
	}

	return &semantics.MapType{
		KeyType:   keyType,
		ValueType: valueType,
	}
}

// inferArrayType infers an array type from composite literal
func (c *Checker) inferArrayType(lit *ast.CompositeLit) semantics.Type {
	if len(lit.Elts) == 0 {
		return &semantics.Invalid{}
	}

	var elemType semantics.Type

	for i, elem := range lit.Elts {
		currentType := c.checkExpr(elem)

		if i == 0 {
			elemType = currentType
		} else {
			// All elements must have the same type
			if !c.isAssignable(elemType, currentType) {
				c.ctx.Diagnostics.Add(
					diagnostics.NewError(
						fmt.Sprintf("array element type mismatch: expected %s, got %s",
							c.typeString(elemType),
							c.typeString(currentType))).
						WithCode(diagnostics.ErrTypeMismatch).
						WithPrimaryLabel(c.currentFile, elem.Loc(),
							fmt.Sprintf("type %s", c.typeString(currentType))).
						WithSecondaryLabel(c.currentFile, lit.Elts[0].Loc(),
							fmt.Sprintf("first element has type %s", c.typeString(elemType))),
				)
			}
		}
	}

	return &semantics.ArrayType{
		IsFixed:     false,
		ElementType: elemType,
	}
}

// checkCompositeLitElements validates elements against target type
func (c *Checker) checkCompositeLitElements(lit *ast.CompositeLit, targetType semantics.Type) {
	switch t := targetType.(type) {
	case *semantics.StructType:
		c.checkStructLitElements(lit, t)
	case *semantics.MapType:
		c.checkMapLitElements(lit, t)
	case *semantics.ArrayType:
		c.checkArrayLitElements(lit, t)
	case *semantics.UserType:
		// Named type - check the underlying definition
		if t.Definition != nil {
			c.checkCompositeLitElements(lit, t.Definition)
		}
	}
}

// checkStructLitElements validates struct literal elements
func (c *Checker) checkStructLitElements(lit *ast.CompositeLit, structType *semantics.StructType) {
	providedFields := make(map[string]bool)

	for _, elem := range lit.Elts {
		kvExpr, ok := elem.(*ast.KeyValueExpr)
		if !ok {
			c.ctx.Diagnostics.Add(
				diagnostics.NewError("struct literal must use field syntax: .field = value").
					WithCode(diagnostics.ErrTypeMismatch).
					WithPrimaryLabel(c.currentFile, elem.Loc(), "expected .field = value"),
			)
			continue
		}

		ident, ok := kvExpr.Key.(*ast.IdentifierExpr)
		if !ok {
			c.ctx.Diagnostics.Add(
				diagnostics.NewError("struct field must be an identifier").
					WithCode(diagnostics.ErrTypeMismatch).
					WithPrimaryLabel(c.currentFile, kvExpr.Key.Loc(), "expected field name"),
			)
			continue
		}

		fieldName := ident.Name

		// Check if field exists in struct type
		fieldType, exists := structType.Fields[fieldName]
		if !exists {
			c.ctx.Diagnostics.Add(
				diagnostics.FieldNotFound(
					c.currentFile,
					kvExpr.Key.Loc(),
					fieldName,
					structType.String(),
				),
			)
			continue
		}

		// Check for duplicate field assignments
		if providedFields[fieldName] {
			c.ctx.Diagnostics.Add(
				diagnostics.NewError(fmt.Sprintf("duplicate field '%s' in struct literal", fieldName)).
					WithCode(diagnostics.ErrTypeMismatch).
					WithPrimaryLabel(c.currentFile, kvExpr.Key.Loc(), "duplicate field"),
			)
			continue
		}

		providedFields[fieldName] = true

		// Check value type matches field type
		valueType := c.checkExpr(kvExpr.Value)
		if !c.isAssignable(fieldType, valueType) {
			c.ctx.Diagnostics.Add(
				diagnostics.NewError(
					fmt.Sprintf("cannot assign value of type %s to field '%s' of type %s",
						c.typeString(valueType),
						fieldName,
						c.typeString(fieldType))).
					WithCode(diagnostics.ErrTypeMismatch).
					WithPrimaryLabel(c.currentFile, kvExpr.Value.Loc(),
						fmt.Sprintf("type %s", c.typeString(valueType))).
					WithSecondaryLabel(c.currentFile, kvExpr.Key.Loc(),
						fmt.Sprintf("field has type %s", c.typeString(fieldType))),
			)
		}
	}

	// Check if all required fields are provided
	for fieldName := range structType.Fields {
		if !providedFields[fieldName] {
			c.ctx.Diagnostics.Add(
				diagnostics.NewError(
					fmt.Sprintf("missing field '%s' in struct literal", fieldName)).
					WithCode(diagnostics.ErrTypeMismatch).
					WithPrimaryLabel(c.currentFile, lit.Loc(), fmt.Sprintf("missing field .%s", fieldName)),
			)
		}
	}
}

// checkMapLitElements validates map literal elements
func (c *Checker) checkMapLitElements(lit *ast.CompositeLit, mapType *semantics.MapType) {
	for _, elem := range lit.Elts {
		kvExpr, ok := elem.(*ast.KeyValueExpr)
		if !ok {
			c.ctx.Diagnostics.Add(
				diagnostics.NewError("map literal must use key => value syntax").
					WithCode(diagnostics.ErrTypeMismatch).
					WithPrimaryLabel(c.currentFile, elem.Loc(), "expected key => value"),
			)
			continue
		}

		keyType := c.checkExpr(kvExpr.Key)
		valueType := c.checkExpr(kvExpr.Value)

		// Check key type
		if !c.isAssignable(mapType.KeyType, keyType) {
			c.ctx.Diagnostics.Add(
				diagnostics.NewError(
					fmt.Sprintf("map key type mismatch: expected %s, got %s",
						c.typeString(mapType.KeyType),
						c.typeString(keyType))).
					WithCode(diagnostics.ErrTypeMismatch).
					WithPrimaryLabel(c.currentFile, kvExpr.Key.Loc(),
						fmt.Sprintf("type %s", c.typeString(keyType))),
			)
		}

		// Check value type
		if !c.isAssignable(mapType.ValueType, valueType) {
			c.ctx.Diagnostics.Add(
				diagnostics.NewError(
					fmt.Sprintf("map value type mismatch: expected %s, got %s",
						c.typeString(mapType.ValueType),
						c.typeString(valueType))).
					WithCode(diagnostics.ErrTypeMismatch).
					WithPrimaryLabel(c.currentFile, kvExpr.Value.Loc(),
						fmt.Sprintf("type %s", c.typeString(valueType))),
			)
		}
	}
}

// checkArrayLitElements validates array literal elements
func (c *Checker) checkArrayLitElements(lit *ast.CompositeLit, arrayType *semantics.ArrayType) {
	for _, elem := range lit.Elts {
		elemType := c.checkExpr(elem)

		if !c.isAssignable(arrayType.ElementType, elemType) {
			c.ctx.Diagnostics.Add(
				diagnostics.NewError(
					fmt.Sprintf("array element type mismatch: expected %s, got %s",
						c.typeString(arrayType.ElementType),
						c.typeString(elemType))).
					WithCode(diagnostics.ErrTypeMismatch).
					WithPrimaryLabel(c.currentFile, elem.Loc(),
						fmt.Sprintf("type %s", c.typeString(elemType))),
			)
		}
	}

	// Check fixed array size
	if arrayType.IsFixed && len(lit.Elts) != arrayType.Size {
		c.ctx.Diagnostics.Add(
			diagnostics.NewError(
				fmt.Sprintf("fixed array literal has %d elements, expected %d",
					len(lit.Elts),
					arrayType.Size)).
				WithCode(diagnostics.ErrTypeMismatch).
				WithPrimaryLabel(c.currentFile, lit.Loc(),
					fmt.Sprintf("%d elements provided", len(lit.Elts))),
		)
	}
}

// checkCastExpr checks a type cast expression (expr as Type)
func (c *Checker) checkCastExpr(expr *ast.CastExpr) semantics.Type {
	// Check the expression being cast
	sourceType := c.checkExpr(expr.X)

	// Convert AST type to semantic type
	targetType := c.astTypeToSemanticType(expr.Type)

	// Validate that the cast is legal
	if !c.isCastable(sourceType, targetType) {
		c.ctx.Diagnostics.Add(
			diagnostics.NewError(
				fmt.Sprintf("cannot cast value of type %s to type %s",
					c.typeString(sourceType),
					c.typeString(targetType))).
				WithCode(diagnostics.ErrInvalidCast).
				WithPrimaryLabel(c.currentFile, expr.X.Loc(),
					fmt.Sprintf("expression has type %s", c.typeString(sourceType))).
				WithSecondaryLabel(c.currentFile, expr.Type.Loc(),
					fmt.Sprintf("cannot cast to %s", c.typeString(targetType))),
		)
		return &semantics.Invalid{}
	}

	return targetType
}

// isCastable checks if a value of type 'from' can be cast to type 'to'
func (c *Checker) isCastable(from, to semantics.Type) bool {
	if from == nil || to == nil {
		return false
	}

	// Invalid types cannot be cast
	if _, ok := from.(*semantics.Invalid); ok {
		return false
	}
	if _, ok := to.(*semantics.Invalid); ok {
		return false
	}

	// Check if types are already assignable (safe cast)
	if c.isAssignable(to, from) {
		return true
	}

	// Allow casting between primitive numeric types
	fromPrim, fromIsPrim := from.(*semantics.PrimitiveType)
	toPrim, toIsPrim := to.(*semantics.PrimitiveType)

	if fromIsPrim && toIsPrim {
		// Allow numeric conversions
		if c.isNumericType(fromPrim.TypeName) && c.isNumericType(toPrim.TypeName) {
			return true
		}
	}

	// Allow casting from anonymous struct to named type if structurally compatible
	fromStruct, fromIsStruct := from.(*semantics.StructType)
	toUser, toIsUser := to.(*semantics.UserType)

	if fromIsStruct && toIsUser {
		if toUser.Definition != nil {
			if toStruct, ok := toUser.Definition.(*semantics.StructType); ok {
				return c.areStructsCompatible(toStruct, fromStruct)
			}
		}
	}

	// Allow casting between named types with same underlying structure
	fromUser, fromIsUser := from.(*semantics.UserType)

	if fromIsUser && toIsUser {
		if fromUser.Definition != nil && toUser.Definition != nil {
			return c.isStructurallyCompatible(toUser.Definition, fromUser.Definition)
		}
	}

	// Allow casting from named type to its underlying structure
	if fromIsUser {
		if fromUser.Definition != nil {
			return c.isCastable(fromUser.Definition, to)
		}
	}

	// Allow casting from anonymous map to named map type if compatible
	fromMap, fromIsMap := from.(*semantics.MapType)
	toMap, toIsMap := to.(*semantics.MapType)

	if fromIsMap && toIsMap {
		return c.isAssignable(toMap.KeyType, fromMap.KeyType) &&
			c.isAssignable(toMap.ValueType, fromMap.ValueType)
	}

	// Allow casting from anonymous array to named array type if compatible
	fromArray, fromIsArray := from.(*semantics.ArrayType)
	toArray, toIsArray := to.(*semantics.ArrayType)

	if fromIsArray && toIsArray {
		return c.isAssignable(toArray.ElementType, fromArray.ElementType)
	}

	return false
}

// isNumericType checks if a type name represents a numeric type
func (c *Checker) isNumericType(typeName types.TYPE_NAME) bool {
	switch typeName {
	case types.TYPE_I8, types.TYPE_I16, types.TYPE_I32, types.TYPE_I64,
		types.TYPE_U8, types.TYPE_U16, types.TYPE_U32, types.TYPE_U64,
		types.TYPE_F32, types.TYPE_F64:
		return true
	}
	return false
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

	// 'none' can be assigned to any optional type
	if _, isNone := from.(*semantics.NoneType); isNone {
		_, toIsOptional := to.(*semantics.OptionalType)
		return toIsOptional
	}

	// Handle optional type assignments
	toOptional, toIsOptional := to.(*semantics.OptionalType)
	fromOptional, fromIsOptional := from.(*semantics.OptionalType)

	if toIsOptional {
		// T? can accept T (wrapping)
		if !fromIsOptional {
			// Check if the base type matches
			return c.isAssignable(toOptional.Base, from)
		}
		// T? can accept T? if base types match
		return c.isAssignable(toOptional.Base, fromOptional.Base)
	}

	if fromIsOptional {
		// T cannot accept T? without unwrapping (this is an error)
		return false
	}

	// Exact type match
	if to.String() == from.String() {
		return true
	}

	// Handle user-defined types and structural compatibility
	toUser, toIsUser := to.(*semantics.UserType)
	fromUser, fromIsUser := from.(*semantics.UserType)

	// Named type to named type - must be exact match or underlying compatible
	if toIsUser && fromIsUser {
		// Same named type
		if toUser.Name == fromUser.Name {
			return true
		}
		// Check underlying structural compatibility
		if toUser.Definition != nil && fromUser.Definition != nil {
			return c.isStructurallyCompatible(toUser.Definition, fromUser.Definition)
		}
		return false
	}

	// Named type to structural type - check underlying definition
	if toIsUser && !fromIsUser {
		if toUser.Definition != nil {
			return c.isStructurallyCompatible(toUser.Definition, from)
		}
		return false
	}

	// Structural type to named type - check underlying definition
	if !toIsUser && fromIsUser {
		if fromUser.Definition != nil {
			return c.isStructurallyCompatible(to, fromUser.Definition)
		}
		return false
	}

	// Struct type compatibility - structural typing
	toStruct, toIsStruct := to.(*semantics.StructType)
	fromStruct, fromIsStruct := from.(*semantics.StructType)

	if toIsStruct && fromIsStruct {
		return c.areStructsCompatible(toStruct, fromStruct)
	}

	// Map type compatibility
	toMap, toIsMap := to.(*semantics.MapType)
	fromMap, fromIsMap := from.(*semantics.MapType)

	if toIsMap && fromIsMap {
		return c.isAssignable(toMap.KeyType, fromMap.KeyType) &&
			c.isAssignable(toMap.ValueType, fromMap.ValueType)
	}

	// Array type compatibility
	toArray, toIsArray := to.(*semantics.ArrayType)
	fromArray, fromIsArray := from.(*semantics.ArrayType)

	if toIsArray && fromIsArray {
		// Fixed arrays must have same size
		if toArray.IsFixed && fromArray.IsFixed && toArray.Size != fromArray.Size {
			return false
		}
		// Element types must be compatible
		return c.isAssignable(toArray.ElementType, fromArray.ElementType)
	}

	// Check primitive type compatibility
	toPrim, toIsPrim := to.(*semantics.PrimitiveType)
	fromPrim, fromIsPrim := from.(*semantics.PrimitiveType)

	if toIsPrim && fromIsPrim {
		// Allow compatible primitive types
		// For now, require exact match
		return toPrim.TypeName == fromPrim.TypeName
	}

	return false
}

// isStructurallyCompatible checks if two types are structurally compatible
func (c *Checker) isStructurallyCompatible(to, from semantics.Type) bool {
	// Unwrap user types
	if toUser, ok := to.(*semantics.UserType); ok {
		if toUser.Definition != nil {
			to = toUser.Definition
		}
	}
	if fromUser, ok := from.(*semantics.UserType); ok {
		if fromUser.Definition != nil {
			from = fromUser.Definition
		}
	}

	// Now check structural compatibility
	toStruct, toIsStruct := to.(*semantics.StructType)
	fromStruct, fromIsStruct := from.(*semantics.StructType)

	if toIsStruct && fromIsStruct {
		return c.areStructsCompatible(toStruct, fromStruct)
	}

	// For other types, use standard assignment check
	return c.isAssignable(to, from)
}

// areStructsCompatible checks if two struct types are compatible
// Structs are compatible if they have the same fields with compatible types
func (c *Checker) areStructsCompatible(to, from *semantics.StructType) bool {
	// Must have same number of fields
	if len(to.Fields) != len(from.Fields) {
		return false
	}

	// All fields must match
	for fieldName, toFieldType := range to.Fields {
		fromFieldType, exists := from.Fields[fieldName]
		if !exists {
			return false
		}
		if !c.isAssignable(toFieldType, fromFieldType) {
			return false
		}
	}

	return true
}

// typeString returns a human-readable string for a type
func (c *Checker) typeString(t semantics.Type) string {
	if t == nil {
		return "unknown"
	}
	return t.String()
}

// reportAssignmentError reports a type mismatch error with helpful context
func (c *Checker) reportAssignmentError(expectedType, actualType semantics.Type, valueLoc, declLoc *source.Location) {
	mainMessage := fmt.Sprintf("cannot assign value of type %s to symbol of type %s",
		c.typeString(actualType),
		c.typeString(expectedType))

	// Check if it's an optional type mismatch and provide helpful note
	var note string
	var help string
	if optType, isOptional := expectedType.(*semantics.OptionalType); isOptional {
		// Expected is optional, actual is not compatible
		note = fmt.Sprintf("%s accepts values of type %s or 'none'",
			c.typeString(expectedType),
			c.typeString(optType.Base))
	} else if _, actualIsOptional := actualType.(*semantics.OptionalType); actualIsOptional {
		// Trying to assign optional to non-optional
		note = "optional types must be unwrapped before assigning to non-optional types"
		help = "use the elvis operator (?:) or a catch clause to unwrap"
	} else if _, actualIsNone := actualType.(*semantics.NoneType); actualIsNone {
		// Trying to assign none to non-optional
		note = "'none' can only be assigned to optional types"
		help = fmt.Sprintf("change symbol type to %s?", c.typeString(expectedType))
	}

	diag := diagnostics.NewError(mainMessage).
		WithCode(diagnostics.ErrTypeMismatch).
		WithPrimaryLabel(c.currentFile, valueLoc,
			fmt.Sprintf("expression has type %s", c.typeString(actualType))).
		WithSecondaryLabel(c.currentFile, declLoc,
			fmt.Sprintf("symbol has type %s", c.typeString(expectedType)))

	if note != "" {
		diag = diag.WithNote(note)
	}

	if help != "" {
		diag = diag.WithHelp(help)
	}

	c.ctx.Diagnostics.Add(diag)
}

// astTypeToSemanticType converts an AST type node to a semantic type
func (c *Checker) astTypeToSemanticType(astType ast.TypeNode) semantics.Type {
	if astType == nil {
		return nil
	}

	switch t := astType.(type) {
	case *ast.IdentifierExpr:
		// Handle primitive type names
		switch t.Name {
		case "i8", "i16", "i32", "i64":
			return &semantics.PrimitiveType{TypeName: types.TYPE_NAME(t.Name)}
		case "u8", "u16", "u32", "u64":
			return &semantics.PrimitiveType{TypeName: types.TYPE_NAME(t.Name)}
		case "f32", "f64":
			return &semantics.PrimitiveType{TypeName: types.TYPE_NAME(t.Name)}
		case "bool":
			return &semantics.PrimitiveType{TypeName: types.TYPE_BOOL}
		case "str":
			return &semantics.PrimitiveType{TypeName: types.TYPE_STRING}
		default:
			// Look up named type in scope
			if sym, ok := c.currentScope.Lookup(t.Name); ok {
				return sym.Type
			}
			return &semantics.Invalid{}
		}

	case *ast.StructType:
		// Convert struct type
		fields := make(map[string]semantics.Type)
		for _, field := range t.Fields {
			fieldType := c.astTypeToSemanticType(field.Type)
			fields[field.Name.Name] = fieldType
		}
		return &semantics.StructType{Fields: fields}

	case *ast.ArrayType:
		// Convert array type
		elemType := c.astTypeToSemanticType(t.ElType)
		arrayType := &semantics.ArrayType{
			IsFixed:     t.Len != nil,
			ElementType: elemType,
		}
		return arrayType

	case *ast.MapType:
		// Convert map type
		keyType := c.astTypeToSemanticType(t.Key)
		valueType := c.astTypeToSemanticType(t.Value)
		return &semantics.MapType{
			KeyType:   keyType,
			ValueType: valueType,
		}

	case *ast.OptionalType:
		// Convert optional type
		baseType := c.astTypeToSemanticType(t.Base)
		return &semantics.OptionalType{Base: baseType}

	case *ast.FuncType:
		// Convert function type
		params := []semantics.ParamsType{}
		for _, param := range t.Params {
			paramType := c.astTypeToSemanticType(param.Type)
			params = append(params, semantics.ParamsType{
				Name:       param.Name.Name,
				Type:       paramType,
				IsVariadic: param.IsVariadic,
			})
		}
		returnType := c.astTypeToSemanticType(t.Result)
		return &semantics.FunctionType{
			Parameters: params,
			ReturnType: returnType,
		}

	default:
		return &semantics.Invalid{}
	}
}

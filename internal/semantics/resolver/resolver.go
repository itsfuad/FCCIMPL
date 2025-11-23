package resolver

import (
	"compiler/internal/context"
	"compiler/internal/diagnostics"
	"compiler/internal/frontend/ast"
	"compiler/internal/semantics"
	"compiler/internal/types"
	"fmt"
)

// Resolver converts AST type expressions to semantic types (Pass 2 / Phase 4)
// Resolves all type information and stores it in symbols
type Resolver struct {
	ctx          *context.CompilerContext
	currentScope *semantics.SymbolTable
	currentFile  string
}

// New creates a new type resolver
func New(ctx *context.CompilerContext) *Resolver {
	return &Resolver{
		ctx: ctx,
	}
}

// Run executes Pass 2: Type Resolution for all files
func Run(ctx *context.CompilerContext) {
	resolver := New(ctx)

	for _, file := range ctx.GetAllFiles() {
		resolver.ResolveFile(file)
	}

	// Second pass: register methods on types
	for _, file := range ctx.GetAllFiles() {
		resolver.RegisterMethods(file)
	}
}

// ResolveFile resolves types for a single source file
func (r *Resolver) ResolveFile(file *context.SourceFile) {
	r.currentFile = file.Path
	r.currentScope = file.Scope

	if file.Scope == nil {
		return // No symbols to resolve
	}

	// Walk all symbols in file scope and resolve their types
	for _, sym := range file.Scope.AllSymbols() {
		r.resolveSymbolType(sym)
	}

	// Also resolve symbols in all block scopes
	r.resolveBlockScopes()
}

// resolveBlockScopes resolves types for all block-level symbols
func (r *Resolver) resolveBlockScopes() {
	for _, scope := range r.ctx.GetAllBlockScopes() {
		r.currentScope = scope
		for _, sym := range scope.AllSymbols() {
			r.resolveSymbolType(sym)
		}
	}
}

// RegisterMethods registers methods on their associated types
func (r *Resolver) RegisterMethods(file *context.SourceFile) {
	r.currentFile = file.Path
	r.currentScope = file.Scope

	if file.AST == nil || file.Scope == nil {
		return
	}

	// Walk all declarations looking for methods
	for _, node := range file.AST.Nodes {
		if methodDecl, ok := node.(*ast.MethodDecl); ok {
			r.registerMethod(methodDecl)
		}
	}
}

// registerMethod registers a method on its receiver type
func (r *Resolver) registerMethod(methodDecl *ast.MethodDecl) {
	if methodDecl.Receiver == nil || methodDecl.Receiver.Type == nil {
		return
	}

	// Get the receiver type name
	var receiverTypeName string
	if ident, ok := methodDecl.Receiver.Type.(*ast.IdentifierExpr); ok {
		receiverTypeName = ident.Name
	} else {
		// Methods can only be defined on named types
		r.ctx.Diagnostics.Add(
			diagnostics.NewError("methods can only be defined on named types").
				WithCode(diagnostics.ErrTypeMismatch).
				WithPrimaryLabel(r.currentFile, methodDecl.Receiver.Type.Loc(), "receiver must be a named type"),
		)
		return
	}

	// Look up the receiver type in the symbol table
	typeSym, ok := r.currentScope.Lookup(receiverTypeName)
	if !ok {
		r.ctx.Diagnostics.Add(
			diagnostics.UndefinedSymbol(r.currentFile, methodDecl.Receiver.Type.Loc(), receiverTypeName),
		)
		return
	}

	// Ensure it's a type symbol
	if typeSym.Kind != semantics.SymbolType {
		r.ctx.Diagnostics.Add(
			diagnostics.NewError(fmt.Sprintf("'%s' is not a type", receiverTypeName)).
				WithCode(diagnostics.ErrTypeMismatch).
				WithPrimaryLabel(r.currentFile, methodDecl.Receiver.Type.Loc(), "expected a type"),
		)
		return
	}

	// Get the UserType
	userType, ok := typeSym.Type.(*semantics.UserType)
	if !ok {
		r.ctx.Diagnostics.Add(
			diagnostics.NewError(fmt.Sprintf("methods can only be defined on user-defined types, not %s", typeSym.Type.String())).
				WithCode(diagnostics.ErrTypeMismatch).
				WithPrimaryLabel(r.currentFile, methodDecl.Receiver.Type.Loc(), "not a user-defined type"),
		)
		return
	}

	// Build the method signature
	methodType := &semantics.FunctionType{
		Parameters: []semantics.ParamsType{},
	}

	// Add method parameters
	if methodDecl.Type != nil {
		for _, param := range methodDecl.Type.Params {
			paramType := r.resolveTypeNode(param.Type)
			methodType.Parameters = append(methodType.Parameters, semantics.ParamsType{
				Name:       param.Name.Name,
				Type:       paramType,
				IsVariadic: param.IsVariadic,
			})
		}

		// Add return type
		if methodDecl.Type.Result != nil {
			methodType.ReturnType = r.resolveTypeNode(methodDecl.Type.Result)
		}
	}

	// Check for duplicate methods
	methodName := methodDecl.Name.Name
	if _, exists := userType.Methods[methodName]; exists {
		r.ctx.Diagnostics.Add(
			diagnostics.NewError(fmt.Sprintf("method '%s' already defined on type '%s'", methodName, receiverTypeName)).
				WithCode(diagnostics.ErrRedeclaredSymbol).
				WithPrimaryLabel(r.currentFile, methodDecl.Name.Loc(), "duplicate method"),
		)
		return
	}

	// Register the method
	userType.Methods[methodName] = methodType
}

// resolveSymbolType resolves the type of a symbol based on its declaration
func (r *Resolver) resolveSymbolType(sym *semantics.Symbol) {
	if sym.Type != nil {
		return // Already resolved
	}

	switch sym.Kind {
	case semantics.SymbolVar:
		r.resolveVarType(sym)
	case semantics.SymbolConst:
		r.resolveConstType(sym)
	case semantics.SymbolType:
		r.resolveUserType(sym)
	case semantics.SymbolFunc:
		r.resolveFuncType(sym)
	case semantics.SymbolParam:
		r.resolveParamType(sym)
	}
}

// resolveVarType resolves the type of a variable
func (r *Resolver) resolveVarType(sym *semantics.Symbol) {
	varDecl, ok := sym.Decl.(*ast.VarDecl)
	if !ok {
		return
	}

	// Find this variable's DeclItem
	for _, item := range varDecl.Decls {
		if item.Name.Name == sym.Name {
			if item.Type != nil {
				// Explicit type provided
				sym.Type = r.resolveTypeNode(item.Type)
			} else if item.Value != nil {
				// Type inference from initializer (will be done in Pass 3)
				// For now, mark as unresolved - checker will infer it
				sym.Type = nil
			} else {
				// Variable with no type and no initializer - error
				r.ctx.Diagnostics.Add(
					diagnostics.NewError("variable must have either a type or an initializer").
						WithCode(diagnostics.ErrMissingType).
						WithPrimaryLabel(r.currentFile, item.Name.Loc(), "add type annotation or initializer"),
				)
			}
			return
		}
	}
}

// resolveConstType resolves the type of a constant
func (r *Resolver) resolveConstType(sym *semantics.Symbol) {
	constDecl, ok := sym.Decl.(*ast.ConstDecl)
	if !ok {
		return
	}

	// Find this constant's DeclItem
	for _, item := range constDecl.Decls {
		if item.Name.Name == sym.Name {
			if item.Type != nil {
				// Explicit type provided
				sym.Type = r.resolveTypeNode(item.Type)
			}
			// If no explicit type, it will be inferred in Pass 3
			return
		}
	}
}

// resolveParamType resolves the type of a function parameter
func (r *Resolver) resolveParamType(sym *semantics.Symbol) {
	field, ok := sym.Decl.(*ast.Field)
	if !ok || field.Type == nil {
		return
	}

	sym.Type = r.resolveTypeNode(field.Type)
}

// resolveUserType resolves a user-defined type
func (r *Resolver) resolveUserType(sym *semantics.Symbol) {
	typeDecl, ok := sym.Decl.(*ast.TypeDecl)
	if !ok {
		return
	}

	// Create a UserType
	userType := &semantics.UserType{
		Name:    sym.Name,
		Methods: make(map[string]*semantics.FunctionType),
		State:   semantics.TypeNotStarted,
	}

	sym.Type = userType

	// Resolve the underlying type definition
	if typeDecl.Type != nil {
		userType.State = semantics.TypeResolving
		userType.Definition = r.resolveTypeNode(typeDecl.Type)
		userType.State = semantics.TypeComplete
	}
}

// resolveFuncType resolves the type of a function
func (r *Resolver) resolveFuncType(sym *semantics.Symbol) {
	funcDecl, ok := sym.Decl.(*ast.FuncDecl)
	if !ok || funcDecl.Type == nil {
		return
	}

	funcType := &semantics.FunctionType{
		Parameters: []semantics.ParamsType{},
	}

	// Resolve parameters
	for _, param := range funcDecl.Type.Params {
		paramType := r.resolveTypeNode(param.Type)
		funcType.Parameters = append(funcType.Parameters, semantics.ParamsType{
			Name:       param.Name.Name,
			Type:       paramType,
			IsVariadic: param.IsVariadic,
		})
	}

	// Resolve return type
	if funcDecl.Type.Result != nil {
		funcType.ReturnType = r.resolveTypeNode(funcDecl.Type.Result)
	}

	sym.Type = funcType

	// Resolve types for parameters in function scope
	if sym.SelfScope != nil {
		prevScope := r.currentScope
		r.currentScope = sym.SelfScope

		for _, paramSym := range sym.SelfScope.AllSymbols() {
			if paramSym.Kind == semantics.SymbolParam {
				r.resolveParamType(paramSym)
			}
		}

		r.currentScope = prevScope
	}
}

// resolveTypeNode converts an AST type node to a semantic type
func (r *Resolver) resolveTypeNode(node ast.TypeNode) semantics.Type {
	if node == nil {
		return nil
	}

	switch t := node.(type) {
	case *ast.IdentifierExpr:
		// Primitive type or user-defined type
		return r.resolveNamedType(t.Name)

	case *ast.ArrayType:
		elemType := r.resolveTypeNode(t.ElType)
		arrayType := &semantics.ArrayType{
			ElementType: elemType,
			IsFixed:     t.Len != nil,
		}
		// TODO: Evaluate t.Len expression for size
		return arrayType

	case *ast.StructType:
		structType := &semantics.StructType{
			Fields: make(map[string]semantics.Type),
		}
		for _, field := range t.Fields {
			if field.Name != nil {
				fieldType := r.resolveTypeNode(field.Type)
				structType.Fields[field.Name.Name] = fieldType
			}
		}
		return structType

	case *ast.FuncType:
		funcType := &semantics.FunctionType{
			Parameters: []semantics.ParamsType{},
		}
		for _, param := range t.Params {
			paramType := r.resolveTypeNode(param.Type)
			funcType.Parameters = append(funcType.Parameters, semantics.ParamsType{
				Name:       param.Name.Name,
				Type:       paramType,
				IsVariadic: param.IsVariadic,
			})
		}
		if t.Result != nil {
			funcType.ReturnType = r.resolveTypeNode(t.Result)
		}
		return funcType

	case *ast.MapType:
		keyType := r.resolveTypeNode(t.Key)
		valueType := r.resolveTypeNode(t.Value)
		return &semantics.MapType{
			KeyType:   keyType,
			ValueType: valueType,
		}

	case *ast.InterfaceType:
		interfaceType := &semantics.InterfaceType{
			Methods: make(map[string]*semantics.FunctionType),
		}
		for _, method := range t.Methods {
			if method.Name != nil && method.Type != nil {
				methodType := r.resolveTypeNode(method.Type)
				if funcType, ok := methodType.(*semantics.FunctionType); ok {
					interfaceType.Methods[method.Name.Name] = funcType
				}
			}
		}
		return interfaceType

	case *ast.OptionalType:
		baseType := r.resolveTypeNode(t.Base)
		return &semantics.OptionalType{Base: baseType}

	case *ast.ErrorType:
		valueType := r.resolveTypeNode(t.Value)
		errorType := r.resolveTypeNode(t.Error)
		return &semantics.ErrorType{
			Valid: valueType,
			Error: errorType,
		}

	default:
		// Unknown type - return invalid
		return &semantics.Invalid{}
	}
}

// resolveNamedType resolves a type by name (primitive or user-defined)
func (r *Resolver) resolveNamedType(name string) semantics.Type {
	// Check if it's a primitive type
	switch name {
	case string(types.TYPE_I8):
		return &semantics.PrimitiveType{TypeName: types.TYPE_I8}
	case string(types.TYPE_I16):
		return &semantics.PrimitiveType{TypeName: types.TYPE_I16}
	case string(types.TYPE_I32):
		return &semantics.PrimitiveType{TypeName: types.TYPE_I32}
	case string(types.TYPE_I64):
		return &semantics.PrimitiveType{TypeName: types.TYPE_I64}
	case string(types.TYPE_U8):
		return &semantics.PrimitiveType{TypeName: types.TYPE_U8}
	case string(types.TYPE_U16):
		return &semantics.PrimitiveType{TypeName: types.TYPE_U16}
	case string(types.TYPE_U32):
		return &semantics.PrimitiveType{TypeName: types.TYPE_U32}
	case string(types.TYPE_U64):
		return &semantics.PrimitiveType{TypeName: types.TYPE_U64}
	case string(types.TYPE_F32):
		return &semantics.PrimitiveType{TypeName: types.TYPE_F32}
	case string(types.TYPE_F64):
		return &semantics.PrimitiveType{TypeName: types.TYPE_F64}
	case string(types.TYPE_BOOL):
		return &semantics.PrimitiveType{TypeName: types.TYPE_BOOL}
	case string(types.TYPE_STRING):
		return &semantics.PrimitiveType{TypeName: types.TYPE_STRING}
	case string(types.TYPE_VOID):
		return &semantics.PrimitiveType{TypeName: types.TYPE_VOID}
	case string(types.TYPE_BYTE):
		return &semantics.PrimitiveType{TypeName: types.TYPE_BYTE}
	}

	// Look up user-defined type in symbol table
	if sym, ok := r.currentScope.Lookup(name); ok {
		if sym.Kind == semantics.SymbolType {
			// Resolve the type if not already done
			if sym.Type == nil {
				r.resolveUserType(sym)
			}
			return sym.Type
		}
	}

	// Type not found - report error and return invalid
	r.ctx.Diagnostics.Add(
		diagnostics.NewError(fmt.Sprintf("undefined type: %s", name)).
			WithCode(diagnostics.ErrUndefinedSymbol),
	)

	return &semantics.Invalid{}
}

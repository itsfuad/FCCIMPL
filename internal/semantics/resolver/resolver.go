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
}

// ResolveFile resolves types for a single source file
func (r *Resolver) ResolveFile(file *context.SourceFile) {
	r.currentFile = file.Path
	r.currentScope = file.Scope

	if file.Scope == nil {
		return // No symbols to resolve
	}

	// Walk all symbols and resolve their types
	for _, sym := range file.Scope.AllSymbols() {
		r.resolveSymbolType(sym)
	}
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
		// Map types not yet implemented in semantic type system
		// For now, return a simple representation
		return &semantics.PrimitiveType{TypeName: types.TYPE_MAP}

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
	case "i8":
		return &semantics.PrimitiveType{TypeName: types.TYPE_I8}
	case "i16":
		return &semantics.PrimitiveType{TypeName: types.TYPE_I16}
	case "i32":
		return &semantics.PrimitiveType{TypeName: types.TYPE_I32}
	case "i64":
		return &semantics.PrimitiveType{TypeName: types.TYPE_I64}
	case "u8":
		return &semantics.PrimitiveType{TypeName: types.TYPE_U8}
	case "u16":
		return &semantics.PrimitiveType{TypeName: types.TYPE_U16}
	case "u32":
		return &semantics.PrimitiveType{TypeName: types.TYPE_U32}
	case "u64":
		return &semantics.PrimitiveType{TypeName: types.TYPE_U64}
	case "f32":
		return &semantics.PrimitiveType{TypeName: types.TYPE_F32}
	case "f64":
		return &semantics.PrimitiveType{TypeName: types.TYPE_F64}
	case "bool":
		return &semantics.PrimitiveType{TypeName: types.TYPE_BOOL}
	case "string", "str":
		return &semantics.PrimitiveType{TypeName: types.TYPE_STRING}
	case "void":
		return &semantics.PrimitiveType{TypeName: types.TYPE_VOID}
	case "byte":
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

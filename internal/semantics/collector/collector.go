package collector

import (
	"compiler/internal/context"
	"compiler/internal/diagnostics"
	"compiler/internal/frontend/ast"
	"compiler/internal/semantics"
	"compiler/internal/source"
)

// Collector walks the AST and builds symbol tables (Pass 1 / Phase 3)
// Collects declarations without resolving types
type Collector struct {
	ctx          *context.CompilerContext
	currentScope *semantics.SymbolTable
	currentFile  string
}

// New creates a new declaration collector
func New(ctx *context.CompilerContext) *Collector {
	return &Collector{
		ctx: ctx,
	}
}

// Run executes Pass 1: Declaration Collection for all files
func Run(ctx *context.CompilerContext) {
	collector := New(ctx)
	
	for _, file := range ctx.GetAllFiles() {
		collector.CollectFile(file)
	}
}

// CollectFile collects declarations from a single source file
func (c *Collector) CollectFile(file *context.SourceFile) {
	c.currentFile = file.Path
	
	// Initialize symbol table if not already done
	if file.Scope == nil {
		ctx := c.ctx
		ctx.InitializeSemantics(file)
	}
	
	c.currentScope = file.Scope
	
	// Walk the AST module
	if file.AST != nil {
		c.collectModule(file.AST)
	}
}

// collectModule walks the module's declarations
func (c *Collector) collectModule(module *ast.Module) {
	for _, node := range module.Nodes {
		c.collectDecl(node)
	}
}

// collectDecl collects a single declaration
func (c *Collector) collectDecl(node ast.Node) {
	switch n := node.(type) {
	case *ast.VarDecl:
		c.collectVarDecl(n)
	case *ast.ConstDecl:
		c.collectConstDecl(n)
	case *ast.TypeDecl:
		c.collectTypeDecl(n)
	case *ast.FuncDecl:
		c.collectFuncDecl(n)
	case *ast.MethodDecl:
		c.collectMethodDecl(n)
	// Block statements can contain declarations (local variables)
	case *ast.Block:
		c.collectBlock(n)
	case *ast.IfStmt:
		c.collectIfStmt(n)
	// Ignore other statement types for declaration collection
	default:
		// Not a declaration, skip
	}
}

// collectVarDecl collects variable declarations: let x := 10;
func (c *Collector) collectVarDecl(decl *ast.VarDecl) {
	for _, item := range decl.Decls {
		name := item.Name.Name
		
		// Create symbol (type will be resolved in Pass 2)
		sym := semantics.NewSymbolWithDecl(name, semantics.SymbolVar, decl)
		
		// Try to declare in current scope
		if err := c.currentScope.Declare(name, sym); err != nil {
			// Symbol already declared - find previous declaration
			if prevSym, ok := c.currentScope.Lookup(name); ok {
				var prevLoc *source.Location
				if prevSym.Decl != nil {
					prevLoc = prevSym.Decl.Loc()
				}
				
				c.ctx.Diagnostics.Add(
					diagnostics.RedeclaredSymbol(
						c.currentFile,
						item.Name.Loc(),
						prevLoc,
						name,
					),
				)
			}
		}
	}
}

// collectConstDecl collects constant declarations: const pi := 3.14;
func (c *Collector) collectConstDecl(decl *ast.ConstDecl) {
	for _, item := range decl.Decls {
		name := item.Name.Name
		
		// Create symbol (type will be resolved in Pass 2)
		sym := semantics.NewSymbolWithDecl(name, semantics.SymbolConst, decl)
		
		// Try to declare in current scope
		if err := c.currentScope.Declare(name, sym); err != nil {
			// Symbol already declared
			if prevSym, ok := c.currentScope.Lookup(name); ok {
				var prevLoc *source.Location
				if prevSym.Decl != nil {
					prevLoc = prevSym.Decl.Loc()
				}
				
				c.ctx.Diagnostics.Add(
					diagnostics.RedeclaredSymbol(
						c.currentFile,
						item.Name.Loc(),
						prevLoc,
						name,
					),
				)
			}
		}
	}
}

// collectTypeDecl collects type declarations: type Point struct { ... };
func (c *Collector) collectTypeDecl(decl *ast.TypeDecl) {
	if decl.Name == nil {
		// Anonymous type declaration at top level - skip
		return
	}
	
	name := decl.Name.Name
	
	// Create symbol (type definition will be resolved in Pass 2)
	sym := semantics.NewSymbolWithDecl(name, semantics.SymbolType, decl)
	
	// Try to declare in current scope
	if err := c.currentScope.Declare(name, sym); err != nil {
		// Symbol already declared
		if prevSym, ok := c.currentScope.Lookup(name); ok {
			var prevLoc *source.Location
			if prevSym.Decl != nil {
				prevLoc = prevSym.Decl.Loc()
			}
			
			c.ctx.Diagnostics.Add(
				diagnostics.RedeclaredSymbol(
					c.currentFile,
					decl.Name.Loc(),
					prevLoc,
					name,
				),
			)
		}
	}
}

// collectFuncDecl collects function declarations: fn add(a: i32) -> i32 { ... }
func (c *Collector) collectFuncDecl(decl *ast.FuncDecl) {
	name := decl.Name.Name
	
	// Create symbol (function type will be resolved in Pass 2)
	sym := semantics.NewSymbolWithDecl(name, semantics.SymbolFunc, decl)
	
	// Try to declare in current scope
	if err := c.currentScope.Declare(name, sym); err != nil {
		// Symbol already declared
		if prevSym, ok := c.currentScope.Lookup(name); ok {
			var prevLoc *source.Location
			if prevSym.Decl != nil {
				prevLoc = prevSym.Decl.Loc()
			}
			
			c.ctx.Diagnostics.Add(
				diagnostics.RedeclaredSymbol(
					c.currentFile,
					decl.Name.Loc(),
					prevLoc,
					name,
				),
			)
		}
	}
	
	// Create a new scope for the function body
	// This will contain parameters and local variables
	if sym.SelfScope == nil {
		sym.SelfScope = semantics.NewSymbolTable(c.currentScope)
		sym.SelfScope.ScopeName = semantics.SYMBOL_TABLE_FUNCTION
	}
	
	// Collect parameters in function scope
	if decl.Type != nil {
		prevScope := c.currentScope
		c.currentScope = sym.SelfScope
		
		for i := range decl.Type.Params {
			c.collectParam(&decl.Type.Params[i])
		}
		
		c.currentScope = prevScope
	}
	
	// Collect declarations in function body
	if decl.Body != nil {
		prevScope := c.currentScope
		c.currentScope = sym.SelfScope
		c.collectBlock(decl.Body)
		c.currentScope = prevScope
	}
}

// collectMethodDecl collects method declarations: fn (r Rect) area() -> f64 { ... }
func (c *Collector) collectMethodDecl(decl *ast.MethodDecl) {
	// Methods are collected differently - they're associated with types
	// For now, we'll create a symbol for the method
	name := decl.Name.Name
	
	// Create symbol (method type will be resolved in Pass 2)
	sym := semantics.NewSymbolWithDecl(name, semantics.SymbolMethod, decl)
	
	// Create a new scope for the method body
	if sym.SelfScope == nil {
		sym.SelfScope = semantics.NewSymbolTable(c.currentScope)
		sym.SelfScope.ScopeName = semantics.SYMBOL_TABLE_FUNCTION
	}
	
	// Collect receiver as a parameter
	if decl.Receiver != nil {
		prevScope := c.currentScope
		c.currentScope = sym.SelfScope
		c.collectParam(decl.Receiver)
		c.currentScope = prevScope
	}
	
	// Collect parameters
	if decl.Type != nil {
		prevScope := c.currentScope
		c.currentScope = sym.SelfScope
		
		for i := range decl.Type.Params {
			c.collectParam(&decl.Type.Params[i])
		}
		
		c.currentScope = prevScope
	}
	
	// Collect declarations in method body
	if decl.Body != nil {
		prevScope := c.currentScope
		c.currentScope = sym.SelfScope
		c.collectBlock(decl.Body)
		c.currentScope = prevScope
	}
}

// collectParam collects a function/method parameter
func (c *Collector) collectParam(param *ast.Field) {
	if param.Name == nil {
		return
	}
	
	name := param.Name.Name
	
	// Create symbol for parameter
	sym := semantics.NewSymbolWithDecl(name, semantics.SymbolParam, param)
	
	// Declare in function scope
	if err := c.currentScope.Declare(name, sym); err != nil {
		// Parameter already declared
		if prevSym, ok := c.currentScope.Lookup(name); ok {
			var prevLoc *source.Location
			if prevSym.Decl != nil {
				prevLoc = prevSym.Decl.Loc()
			}
			
			c.ctx.Diagnostics.Add(
				diagnostics.RedeclaredSymbol(
					c.currentFile,
					param.Name.Loc(),
					prevLoc,
					name,
				),
			)
		}
	}
}

// collectBlock collects declarations within a block statement
func (c *Collector) collectBlock(block *ast.Block) {
	for _, stmt := range block.Nodes {
		c.collectDecl(stmt)
	}
}

// collectIfStmt collects declarations in if statement branches
func (c *Collector) collectIfStmt(stmt *ast.IfStmt) {
	if stmt.Body != nil {
		c.collectBlock(stmt.Body)
	}
	if stmt.Else != nil {
		// Else can be a block or another if statement
		switch e := stmt.Else.(type) {
		case *ast.Block:
			c.collectBlock(e)
		case *ast.IfStmt:
			c.collectIfStmt(e)
		}
	}
}

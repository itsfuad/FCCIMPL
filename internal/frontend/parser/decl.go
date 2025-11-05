package parser

import (
	"compiler/internal/diagnostics"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/source"
)

// parseVarDecl: let x := 10; or let x: i32 = 10;
func (p *Parser) parseVarDecl() *ast.VarDecl {
	start := p.advance().Start
	decls, location := p.parseVariableDeclaration(start, false)
	return &ast.VarDecl{
		Decls:    decls,
		Location: location,
	}
}

// parseConstDecl: const pi := 3.14;
func (p *Parser) parseConstDecl() *ast.ConstDecl {
	start := p.peek().Start
	p.expect(lexer.CONST_TOKEN)

	decls, location := p.parseVariableDeclaration(start, true)

	return &ast.ConstDecl{
		Decls:    decls,
		Location: location,
	}
}

// parseVariableDeclaration: unified parser for let/const declarations
// Returns the declarations and location
func (p *Parser) parseVariableDeclaration(start source.Position, isConst bool) ([]ast.DeclLists, source.Location) {
	var decls []ast.DeclLists

	// can be let a : <type> = <expr>; or let a := <expr>; or let a: <type>;
	// or list of variables, let a : i32, b := 10, c: str;
	// for const, initializer is mandatory

	for !p.isAtEnd() {

		var decl ast.DeclLists

		decl.Name = p.parseIdentifier()

		if p.match(lexer.COLON_TOKEN) {
			p.advance()
			decl.Type = p.parseType()

			// Check for initializer
			if p.match(lexer.EQUALS_TOKEN) {
				p.advance()
				decl.Value = p.parseExpr()
			} else if isConst {
				// Constants MUST have an initializer
				peek := p.peek()
				p.diagnostics.Add(
					diagnostics.NewError("expected = after constant type").
						WithCode(diagnostics.ErrUnexpectedToken).
						WithHelp("Constants must have an initializer").
						WithPrimaryLabel(p.filepath, source.NewLocation(&peek.Start, &peek.End), "missing initializer for constant"),
				)
				// Continue parsing to gather more errors
			}
			// For variables, no initializer is OK (decl.Value remains nil)
		} else if p.match(lexer.WALRUS_TOKEN) {
			p.advance()
			decl.Value = p.parseExpr()
		} else if p.match(lexer.EQUALS_TOKEN) {
			// Allow the parsing but report an error
			loc := p.makeLocation(p.advance().Start)
			decl.Value = p.parseExpr()
			p.diagnostics.Add(
				diagnostics.NewError("expected ':=' for inferred symbols with values"). 
				WithCode(diagnostics.ErrUnexpectedToken). 
				WithPrimaryLabel(p.filepath, &loc, "add a `:` before `=`"),
			)
		} else {
			if isConst {
				p.error("expected ':' or ':=' in constant declaration")
			} else {
				p.error("expected ':' or ':=' in variable declaration")
			}
		}

		decls = append(decls, decl)

		if !p.match(lexer.COMMA_TOKEN) {
			break
		}
		p.advance()
	}

	p.expect(lexer.SEMICOLON_TOKEN)

	return decls, p.makeLocation(start)
}

// parseTypeDecl: type Point struct { .x: i32 };
func (p *Parser) parseTypeDecl() *ast.TypeDecl {
	start := p.peek().Start
	p.expect(lexer.TYPE_TOKEN)

	name := p.parseIdentifier()
	typ := p.parseType()

	p.expect(lexer.SEMICOLON_TOKEN)

	return &ast.TypeDecl{
		Name:     name,
		Type:     typ,
		Location: p.makeLocation(start),
	}
}

// parseFuncDecl: fn add(a: i32) -> i32 { return a; }
// This is the entry point for all function-like declarations:
// - Named functions: fn add(a: i32) -> i32 { ... }
// - Anonymous functions: fn (a: i32) -> i32 { ... }
// - Methods: fn (r Rect) area() -> f64 { ... }
func (p *Parser) parseFuncDecl() ast.Node {
	start := p.peek().Start
	p.expect(lexer.FUNCTION_TOKEN)

	// Lookahead to determine what kind of function this is
	if p.match(lexer.OPEN_PAREN) {
		// Could be: fn (params) { body } -- anonymous function
		// Or:       fn (receiver) methodName(params) { body } -- method

		params := p.parseFunctionParams()

		p.expect(lexer.CLOSE_PAREN)

		// If next token is identifier, it's a method
		if p.match(lexer.IDENTIFIER_TOKEN) {
			return p.parseMethodDecl(start, params)
		}

		// Otherwise, it's an anonymous function (function literal)
		return p.parseFuncLit(start, params)
	}

	// Named function: fn name(params) -> return { body }
	return p.parseNamedFuncDecl(start)
}

// parseFunctionParams parses function parameters: name: type, name: type, ...
// Used for both regular parameters and method receivers
func (p *Parser) parseFunctionParams() []*ast.Field {

	p.advance() // consume '('

	params := []*ast.Field{}

	if p.match(lexer.CLOSE_PAREN) {
		return params
	}

	for !(p.match(lexer.CLOSE_PAREN) || p.isAtEnd()) {

		// Parse parameter name
		name := p.parseIdentifier()

		// Expect colon
		if !p.match(lexer.COLON_TOKEN) {
			p.error("expected ':' after parameter name")
			break
		}

		p.expect(lexer.COLON_TOKEN)

		// Parse parameter type
		paramType := p.parseType()

		param := &ast.Field{
			Name:     name,
			Type:     paramType,
			Location: p.makeLocation(*name.Start),
		}

		params = append(params, param)

		if p.match(lexer.CLOSE_PAREN) {
			break
		}

		if p.checkTrailing(lexer.COMMA_TOKEN, lexer.CLOSE_PAREN, "function parameters") {
			p.advance() // skip the token
			break
		}

		p.expect(lexer.COMMA_TOKEN)
	}

	p.expect(lexer.CLOSE_PAREN)

	return params
}

// parseNamedFuncDecl parses a named function declaration: fn name(params) -> return { body }
func (p *Parser) parseNamedFuncDecl(start source.Position) *ast.FuncDecl {
	name := p.parseIdentifier()

	// Parse function type (parameters and return type)
	funcType := p.parseFuncType()

	// Parse body
	var body *ast.Block
	if p.match(lexer.OPEN_CURLY) {
		body = p.parseBlock()
	}

	return &ast.FuncDecl{
		Name:     name,
		Type:     funcType,
		Body:     body,
		Location: p.makeLocation(start),
	}
}

// parseFuncLit parses an anonymous function: fn (params) -> return { body }
func (p *Parser) parseFuncLit(start source.Position, params []*ast.Field) *ast.FuncLit {
	// Create FieldList from params
	paramList := &ast.FieldList{
		List:     params,
		Location: p.makeLocation(start),
	}

	// Parse return type if present
	var result ast.TypeNode
	if p.match(lexer.ARROW_TOKEN) {
		p.advance()
		typenode := p.parseType()
		result = typenode
	}

	funcType := &ast.FuncType{
		Params:   paramList,
		Result:   result,
		Location: p.makeLocation(start),
	}

	// Parse body
	body := p.parseBlock()

	return &ast.FuncLit{
		Type:     funcType,
		Body:     body,
		Location: p.makeLocation(start),
	}
}

// parseMethodDecl parses a method declaration: fn (receiver) methodName(params) -> return { body }
func (p *Parser) parseMethodDecl(start source.Position, receivers []*ast.Field) *ast.MethodDecl {
	// Method name
	methodName := p.parseIdentifier()

	// Validate receiver count
	if len(receivers) == 0 {
		p.diagnostics.Add(
			diagnostics.NewError("method must have a receiver").
				WithCode(diagnostics.ErrUnexpectedToken).
				WithPrimaryLabel(p.filepath, methodName.Loc(), "method declared without receiver"),
		)
		return nil
	}

	if len(receivers) > 1 {
		secondReceiver := receivers[1]
		p.diagnostics.Add(
			diagnostics.NewError("method can only have one receiver").
				WithCode(diagnostics.ErrUnexpectedToken).
				WithPrimaryLabel(p.filepath, secondReceiver.Loc(), "extra receiver"),
		)
	}

	receiver := receivers[0]

	// Parse method signature (parameters and return type)
	funcType := p.parseFuncType()

	// Parse body
	body := p.parseBlock()

	return &ast.MethodDecl{
		Receiver: receiver,
		Name:     methodName,
		Type:     funcType,
		Body:     body,
		Location: p.makeLocation(start),
	}
}

// parseBlock: { ... }
func (p *Parser) parseBlock() *ast.Block {

	start := p.expect(lexer.OPEN_CURLY).Start

	nodes := []ast.Node{}
	for !(p.match(lexer.CLOSE_CURLY) || p.isAtEnd()) {
		node := p.parseStmt()
		if node == nil {
			// parseStmt() returned nil - this is an error case
			// Advance to prevent infinite loop and try to recover
			p.error("unexpected token in block")
			p.advance()
			continue
		}
		nodes = append(nodes, node)
	}

	p.expect(lexer.CLOSE_CURLY)

	return &ast.Block{
		Nodes:    nodes,
		Location: p.makeLocation(start),
	}
}

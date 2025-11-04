package parser

import (
	"compiler/internal/diagnostics"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/source"
)

// parseVarDecl: let x := 10; or let x: i32 = 10;
func (p *Parser) parseVarDecl() *ast.VarDecl {
	start := p.peek().Start
	p.expect(lexer.LET_TOKEN)

	var decls []ast.DeclLists

	// can be let a : <type> = <expr>; or let a := <expr>; or let a: <type>;
	// or list of variables, let a : i32, b := 10, c: str;

	for !p.isAtEnd() {
		var decl ast.DeclLists

		decl.Name = p.parseIdentifier()

		if p.match(lexer.COLON_TOKEN) {
			decl.Type = p.parseType()
			p.expect(lexer.EQUALS_TOKEN)
			decl.Value = p.parseExpr()
		} else if p.match(lexer.WALRUS_TOKEN) {
			decl.Value = p.parseExpr()
		} else {
			p.error("expected ':' or ':=' in variable declaration")
		}

		decls = append(decls, decl)

		if !p.match(lexer.COMMA_TOKEN) {
			break
		}
	}

	p.expect(lexer.SEMICOLON_TOKEN)

	return &ast.VarDecl{
		Decls:    decls,
		Location: p.makeLocation(start),
	}
}

// parseConstDecl: const pi := 3.14;
func (p *Parser) parseConstDecl() *ast.ConstDecl {
	start := p.peek().Start
	p.expect(lexer.CONST_TOKEN)

	// can be const a := <expr>; or const a: <type> = <expr>; or list of constants, const a := 10, b: str = "hello";
	// must have initializers
	var decls []ast.DeclLists
	for !p.isAtEnd() {
		var decl ast.DeclLists

		decl.Name = p.parseIdentifier()

		if p.match(lexer.COLON_TOKEN) {
			decl.Type = p.parseType()
			if !p.check(lexer.EQUALS_TOKEN) {
				//p.error("expected = after constant type")
				peek := p.peek()
				p.diagnostics.Add(
					diagnostics.NewError("expected = after constant type").
						WithCode(diagnostics.ErrUnexpectedToken).
						WithHelp("Constants must have an initializer").
						WithPrimaryLabel(p.filepath, source.NewLocation(&peek.Start, &peek.End), "missing initializer for constant"),
				)
				return nil
			}
			p.expect(lexer.EQUALS_TOKEN)
			decl.Value = p.parseExpr()
		} else if p.match(lexer.WALRUS_TOKEN) {
			decl.Value = p.parseExpr()
		} else {
			p.error("expected ':' or ':=' in constant declaration")
		}

		decls = append(decls, decl)

		if !p.match(lexer.COMMA_TOKEN) {
			break
		}
	}

	p.expect(lexer.SEMICOLON_TOKEN)

	return &ast.ConstDecl{
		Decls:    decls,
		Location: p.makeLocation(start),
	}
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
	if p.check(lexer.OPEN_PAREN) {
		// Could be: fn (params) { body } -- anonymous function
		// Or:       fn (receiver) methodName(params) { body } -- method
		p.advance() // consume '('

		params := p.parseFunctionParams()

		p.expect(lexer.CLOSE_PAREN)

		// If next token is identifier, it's a method
		if p.check(lexer.IDENTIFIER_TOKEN) {
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
	params := []*ast.Field{}

	if p.check(lexer.CLOSE_PAREN) {
		return params
	}

	for {
		paramStart := p.peek().Start

		// Safety check: if we're at closing paren or end of file, break
		if p.check(lexer.CLOSE_PAREN) || p.isAtEnd() {
			break
		}

		// Verify we have an identifier before parsing parameter
		if !p.check(lexer.IDENTIFIER_TOKEN) {
			p.error("expected parameter name")
			p.advance() // Prevent infinite loop
			// Try to recover by skipping to next comma or closing paren
			for !p.check(lexer.COMMA_TOKEN) && !p.check(lexer.CLOSE_PAREN) && !p.isAtEnd() {
				p.advance()
			}
			if p.match(lexer.COMMA_TOKEN) {
				continue
			}
			break
		}

		// Parse parameter name
		name := p.parseIdentifier()

		// Expect colon
		if !p.check(lexer.COLON_TOKEN) {
			p.error("expected ':' after parameter name")
			// Try to recover
			for !p.check(lexer.COMMA_TOKEN) && !p.check(lexer.CLOSE_PAREN) && !p.isAtEnd() {
				p.advance()
			}
			if p.match(lexer.COMMA_TOKEN) {
				continue
			}
			break
		}
		p.expect(lexer.COLON_TOKEN)

		// Parse parameter type
		paramType := p.parseType()

		param := &ast.Field{
			Name:     name,
			Type:     paramType,
			Location: p.makeLocation(paramStart),
		}

		params = append(params, param)

		// Check for comma
		if !p.match(lexer.COMMA_TOKEN) {
			break
		}

		// Check for trailing comma
		if p.checkTrailingComma(lexer.CLOSE_PAREN, "function parameters") {
			break
		}
	}

	return params
}

// parseNamedFuncDecl parses a named function declaration: fn name(params) -> return { body }
func (p *Parser) parseNamedFuncDecl(start source.Position) *ast.FuncDecl {
	name := p.parseIdentifier()

	// Parse function type (parameters and return type)
	funcType := p.parseFuncType()

	// Parse body
	var body *ast.Block
	if p.check(lexer.OPEN_CURLY) {
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
	var results *ast.FieldList
	if p.match(lexer.ARROW_TOKEN) {
		returnType := p.parseType()
		results = &ast.FieldList{
			List: []*ast.Field{
				{
					Type:     returnType,
					Location: *returnType.Loc(),
				},
			},
			Location: *returnType.Loc(),
		}
	}

	funcType := &ast.FuncType{
		Params:   paramList,
		Results:  results,
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
	start := p.peek().Start
	p.expect(lexer.OPEN_CURLY)

	nodes := []ast.Node{}
	for !p.check(lexer.CLOSE_CURLY) && !p.isAtEnd() {
		node := p.parseStmt()
		if node == nil {
			// parseStmt() returned nil - this is an error case
			// Advance to prevent infinite loop and try to recover
			p.error("unexpected token in block, skipping")
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

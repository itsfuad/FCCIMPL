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
func (p *Parser) parseFuncDecl() *ast.FuncDecl {
	start := p.peek().Start
	p.expect(lexer.FUNCTION_TOKEN)

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

// parseBlock: { ... }
func (p *Parser) parseBlock() *ast.Block {
	start := p.peek().Start
	p.expect(lexer.OPEN_CURLY)

	nodes := []ast.Node{}
	for !p.check(lexer.CLOSE_CURLY) && !p.isAtEnd() {
		node := p.parseStmt()
		nodes = append(nodes, node)
	}

	p.expect(lexer.CLOSE_CURLY)

	return &ast.Block{
		Nodes:    nodes,
		Location: p.makeLocation(start),
	}
}

package parser

import (
	"compiler/internal/diagnostics"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"fmt"
)

// parseType parses a type expression
func (p *Parser) parseType() ast.TypeNode {
	tok := p.peek()

	var t ast.TypeNode

	switch tok.Kind {
	case lexer.IDENTIFIER_TOKEN:
		// Type identifier - convert IdentifierExpr to support both Expr() and TypeExpr()
		t = p.parseIdentifier()

	case lexer.OPEN_BRACKET:
		t = p.parseArrayType()

	case lexer.STRUCT_TOKEN:
		t = p.parseStructType()

	case lexer.INTERFACE_TOKEN:
		t = p.parseInterfaceType()

	case lexer.ENUM_TOKEN:
		t = p.parseEnumType()

	case lexer.MAP_TOKEN:
		t = p.parseMapType()

	case lexer.FUNCTION_TOKEN:
		t = p.parseFuncType()

	default:
		p.error(fmt.Sprintf("expected type, got %s", tok.Value))
		return nil
	}

	// Check for optional type T?
	if p.match(lexer.QUESTION_TOKEN) {
		t = &ast.OptionalType{
			Base:     t,
			Location: *t.Loc(),
		}
	}

	return t
}

func (p *Parser) parseArrayType() *ast.ArrayType {

	tok := p.expect(lexer.OPEN_BRACKET)

	var size *ast.BasicLit
	if !p.check(lexer.CLOSE_BRACKET) {
		sizeExpr := p.parseExpr()
		if lit, ok := sizeExpr.(*ast.BasicLit); ok {
			size = lit
		} else {
			p.error("array size must be a constant integer literal")
			return nil
		}
	}

	p.expect(lexer.CLOSE_BRACKET)

	elem := p.parseType()

	return &ast.ArrayType{
		Len:      size, // nil for dynamic arrays []T
		ElType:   elem,
		Location: p.makeLocation(tok.Start),
	}
}

func (p *Parser) parseStructType() *ast.StructType {

	tok := p.expect(lexer.STRUCT_TOKEN)
	p.expect(lexer.OPEN_CURLY)

	fields := &ast.FieldList{
		List: []*ast.Field{},
	}

	for !p.check(lexer.CLOSE_CURLY) && !p.isAtEnd() {
		// Error recovery: Check if we have a dot token
		if !p.check(lexer.DOT_TOKEN) {
			p.error(fmt.Sprintf("expected . for struct field, got %s", p.peek().Value))
			p.advance() // Advance to prevent infinite loop
			continue
		}

		p.expect(lexer.DOT_TOKEN)
		name := p.parseIdentifier()
		p.expect(lexer.COLON_TOKEN)
		typ := p.parseType()

		fields.List = append(fields.List, &ast.Field{
			Name: name,
			Type: typ,
		})

		if !p.match(lexer.COMMA_TOKEN) {
			break
		}

		// Check for trailing comma before closing brace
		if p.checkTrailingComma(lexer.CLOSE_CURLY, "struct type") {
			break
		}
	}

	p.expect(lexer.CLOSE_CURLY)

	return &ast.StructType{
		Fields:   fields,
		Location: p.makeLocation(tok.Start),
	}
}

func (p *Parser) parseFuncType() *ast.FuncType {

	tok := p.expect(lexer.OPEN_PAREN)

	params := &ast.FieldList{
		List: []*ast.Field{},
	}

	if !p.check(lexer.CLOSE_PAREN) {
		name := p.parseIdentifier()
		p.expect(lexer.COLON_TOKEN)
		typ := p.parseType()

		params.List = append(params.List, &ast.Field{
			Name: name,
			Type: typ,
		})

		for p.match(lexer.COMMA_TOKEN) {
			// Check for trailing comma before closing paren
			if p.checkTrailingComma(lexer.CLOSE_PAREN, "function parameters") {
				break
			}

			name := p.parseIdentifier()
			p.expect(lexer.COLON_TOKEN)
			typ := p.parseType()

			params.List = append(params.List, &ast.Field{
				Name: name,
				Type: typ,
			})
		}
	}

	p.expect(lexer.CLOSE_PAREN)

	var results *ast.FieldList
	if p.match(lexer.ARROW_TOKEN) {
		resultType := p.parseType()
		// Wrap single return type in a FieldList
		results = &ast.FieldList{
			List: []*ast.Field{
				{
					Name: nil, // Anonymous return
					Type: resultType,
				},
			},
		}
	}

	return &ast.FuncType{
		Params:   params,
		Results:  results,
		Location: p.makeLocation(tok.Start),
	}
}

func (p *Parser) parseInterfaceType() *ast.InterfaceType {

	tok := p.expect(lexer.INTERFACE_TOKEN)

	// Interface MUST have braces - reject standalone "interface"
	if !p.check(lexer.OPEN_CURLY) {
		//p.error(fmt.Sprintf("expected { after interface, got %s", p.peek().Value))
		loc := p.makeLocation(tok.Start)
		p.diagnostics.Add(
			diagnostics.NewError(fmt.Sprintf("expected { after interface, got %s", p.peek().Value)).
			WithCode(diagnostics.ErrUnexpectedToken).
			WithPrimaryLabel(p.filepath, &loc, "here").
			WithHelp("Interfaces must have a body"),
		)
		// Return empty interface to allow parsing to continue
		return &ast.InterfaceType{
			Methods:  &ast.FieldList{List: []*ast.Field{}},
			Location: loc,
		}
	}

	p.expect(lexer.OPEN_CURLY)

	methods := &ast.FieldList{
		List: []*ast.Field{},
	}

	for !p.check(lexer.CLOSE_CURLY) && !p.isAtEnd() {
		methodStart := p.peek().Start

		// Check if we have an identifier (method name)
		if !p.check(lexer.IDENTIFIER_TOKEN) {
			// Error recovery: skip unexpected token and try to continue
			p.error(fmt.Sprintf("expected method name, got %s", p.peek().Value))
			p.advance() // CRITICAL: advance to prevent infinite loop
			continue
		}

		// Parse method name
		name := p.parseIdentifier()

		// Parse method signature (function type)
		funcType := p.parseFuncType()

		// Expect semicolon after method signature
		if !p.match(lexer.SEMICOLON_TOKEN) {
			p.error("expected ';' after method signature")
			// Try to recover by skipping to next semicolon or closing brace
			for !p.check(lexer.SEMICOLON_TOKEN) && !p.check(lexer.CLOSE_CURLY) && !p.isAtEnd() {
				p.advance()
			}
			if p.check(lexer.SEMICOLON_TOKEN) {
				p.advance()
			}
		}

		methods.List = append(methods.List, &ast.Field{
			Name:     name,
			Type:     funcType,
			Location: p.makeLocation(methodStart),
		})

		// Check if we're at the closing brace
		if p.check(lexer.CLOSE_CURLY) {
			break
		}
	}

	p.expect(lexer.CLOSE_CURLY)

	return &ast.InterfaceType{
		Methods:  methods,
		Location: p.makeLocation(tok.Start),
	}
}

func (p *Parser) parseEnumType() *ast.EnumType {
	tok := p.expect(lexer.ENUM_TOKEN)
	p.expect(lexer.OPEN_CURLY)

	fields := &ast.FieldList{
		List: []*ast.Field{},
	}

	for !p.check(lexer.CLOSE_CURLY) && !p.isAtEnd() {
		// Error recovery: Check if we have an identifier
		if !p.check(lexer.IDENTIFIER_TOKEN) {
			p.error(fmt.Sprintf("expected enum variant name, got %s", p.peek().Value))
			p.advance() // Advance to prevent infinite loop
			continue
		}

		name := p.parseIdentifier()
		fields.List = append(fields.List, &ast.Field{
			Name: name,
		})

		if !p.match(lexer.COMMA_TOKEN) {
			break
		}

		// Check for trailing comma before closing brace
		if p.checkTrailingComma(lexer.CLOSE_CURLY, "enum type") {
			break
		}
	}

	p.expect(lexer.CLOSE_CURLY)

	return &ast.EnumType{
		Variants: fields,
		Location: p.makeLocation(tok.Start),
	}
}

func (p *Parser) parseMapType() *ast.MapType {
	tok := p.expect(lexer.MAP_TOKEN)
	p.expect(lexer.OPEN_BRACKET)

	keyType := p.parseType()

	p.expect(lexer.CLOSE_BRACKET)

	valueType := p.parseType()

	return &ast.MapType{
		Key:      keyType,
		Value:    valueType,
		Location: p.makeLocation(tok.Start),
	}
}

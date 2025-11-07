package parser

import (
	"compiler/internal/diagnostics"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/source"
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
		//p.error(fmt.Sprintf("expected type, got %s", tok.Value))
		p.diagnostics.Add(
			diagnostics.NewError(fmt.Sprintf("expected type, got %s", tok.Value)).
				WithCode(diagnostics.ErrMissingType).
				WithPrimaryLabel(p.filepath, source.NewLocation(&tok.Start, &tok.End), fmt.Sprintf("remove `%s` from here", tok.Value)),
		)
		// skip the token
		p.advance()
		// return &ast.Invalid{
		// 	Location: p.makeLocation(tok.Start),
		// }
		return p.parseType()
	}

	// Check for optional type T?
	if p.match(lexer.QUESTION_TOKEN) {
		p.advance()
		t = &ast.OptionalType{
			Base:     t,
			Location: *t.Loc(),
		}
	}

	return t
}

func (p *Parser) parseArrayType() *ast.ArrayType {

	tok := p.expect(lexer.OPEN_BRACKET)

	var size ast.Expression
	if !p.match(lexer.CLOSE_BRACKET) {
		size = p.parseExpr()
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

	for !(p.match(lexer.CLOSE_CURLY) || p.isAtEnd()) {
		// Error recovery: Check if we have a dot token
		if !p.match(lexer.DOT_TOKEN) {
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

		if p.match(lexer.CLOSE_CURLY) {
			break
		}

		if p.checkTrailing(lexer.COMMA_TOKEN, lexer.CLOSE_CURLY, "struct type") {
			p.advance() // skip the token
			break
		}

		p.expect(lexer.COMMA_TOKEN)
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

	if !p.match(lexer.CLOSE_PAREN) {

		for !(p.match(lexer.CLOSE_PAREN) || p.isAtEnd()) {

			name := p.parseIdentifier()
			p.expect(lexer.COLON_TOKEN)
			typ := p.parseType()

			params.List = append(params.List, &ast.Field{
				Name:     name,
				Type:     typ,
				Location: *source.NewLocation(name.Start, typ.Loc().End),
			})

			if p.match(lexer.CLOSE_PAREN) {
				break
			}

			if p.checkTrailing(lexer.COMMA_TOKEN, lexer.CLOSE_PAREN, "function parameters") {
				p.advance() // skip the token
				break
			}

			p.expect(lexer.COMMA_TOKEN)
		}
	}

	p.expect(lexer.CLOSE_PAREN)

	var result ast.TypeNode
	if p.match(lexer.ARROW_TOKEN) {
		p.advance()
		resultType := p.parseType()
		// Wrap single return type in a FieldList
		result = resultType
	}

	return &ast.FuncType{
		Params:   params,
		Result:   result,
		Location: p.makeLocation(tok.Start),
	}
}

func (p *Parser) parseInterfaceType() *ast.InterfaceType {

	tok := p.advance()

	p.expect(lexer.OPEN_CURLY)

	methods := &ast.FieldList{
		List: []*ast.Field{},
	}

	for !(p.match(lexer.CLOSE_CURLY) || p.isAtEnd()) {

		name := p.parseIdentifier()

		functype := p.parseFuncType()

		methods.List = append(methods.List, &ast.Field{
			Name:     name,
			Type:     functype,
			Location: *source.NewLocation(name.Start, functype.End),
		})

		if p.match(lexer.CLOSE_CURLY) {
			break
		}

		if p.checkTrailing(lexer.SEMICOLON_TOKEN, lexer.CLOSE_CURLY, "interface type") {
			p.advance() // skip the token
			break
		}

		p.expect(lexer.COMMA_TOKEN)
	}

	end := p.expectError(lexer.CLOSE_CURLY, fmt.Sprintf("expected '}' to close interface type, found %s", p.peek().Kind)).End

	return &ast.InterfaceType{
		Methods:  methods,
		Location: *source.NewLocation(&tok.Start, &end),
	}
}

func (p *Parser) parseEnumType() *ast.EnumType {

	tok := p.expect(lexer.ENUM_TOKEN)

	p.expect(lexer.OPEN_CURLY)

	fields := &ast.FieldList{
		List: []*ast.Field{},
	}

	for !(p.match(lexer.CLOSE_CURLY) || p.isAtEnd()) {
		// Error recovery: Check if we have an identifier
		if !p.match(lexer.IDENTIFIER_TOKEN) {
			p.error(fmt.Sprintf("expected enum variant name, got %s", p.peek().Value))
			p.advance() // Advance to prevent infinite loop
			continue
		}

		name := p.parseIdentifier()
		fields.List = append(fields.List, &ast.Field{
			Name: name,
		})

		if p.match(lexer.CLOSE_CURLY) {
			break
		}

		// Check for trailing comma before closing brace
		if p.checkTrailing(lexer.COMMA_TOKEN, lexer.CLOSE_CURLY, "enum type") {
			p.advance() // skip the token
			break
		}

		p.expect(lexer.COMMA_TOKEN)
	}

	p.expectError(lexer.CLOSE_CURLY, fmt.Sprintf("expected '}' to close enum type, found %s", p.peek().Kind))

	return &ast.EnumType{
		Variants: fields,
		Location: p.makeLocation(tok.Start),
	}
}

func (p *Parser) parseMapType() *ast.MapType {

	tok := p.advance()

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

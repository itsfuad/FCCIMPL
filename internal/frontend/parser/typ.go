
package parser

import (
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
	p.expect(lexer.OPEN_BRACKET)

	var size *ast.BasicLit
	if !p.check(lexer.CLOSE_BRACKET) {
		sizeExpr := p.parseExpr()
		if lit, ok := sizeExpr.(*ast.BasicLit); ok {
			size = lit
		}
	}

	p.expect(lexer.CLOSE_BRACKET)

	elem := p.parseType()

	return &ast.ArrayType{
		Len:    size, // nil for dynamic arrays []T
		ElType: elem,
	}
}

func (p *Parser) parseStructType() *ast.StructType {
	p.expect(lexer.STRUCT_TOKEN)
	p.expect(lexer.OPEN_CURLY)

	fields := &ast.FieldList{
		List: []*ast.Field{},
	}

	for !p.check(lexer.CLOSE_CURLY) && !p.isAtEnd() {
		p.expect(lexer.DOT_TOKEN)
		name := p.parseIdentifier()
		p.expect(lexer.COLON_TOKEN)
		typ := p.parseType()

		fields.List = append(fields.List, &ast.Field{
			Names: []*ast.IdentifierExpr{name},
			Type:  typ,
		})

		if !p.match(lexer.COMMA_TOKEN) {
			break
		}
	}

	p.expect(lexer.CLOSE_CURLY)

	return &ast.StructType{
		Fields: fields,
	}
}

func (p *Parser) parseFuncType() *ast.FuncType {
	p.expect(lexer.OPEN_PAREN)

	params := &ast.FieldList{
		List: []*ast.Field{},
	}

	if !p.check(lexer.CLOSE_PAREN) {
		name := p.parseIdentifier()
		p.expect(lexer.COLON_TOKEN)
		typ := p.parseType()

		params.List = append(params.List, &ast.Field{
			Names: []*ast.IdentifierExpr{name},
			Type:  typ,
		})

		for p.match(lexer.COMMA_TOKEN) {
			name := p.parseIdentifier()
			p.expect(lexer.COLON_TOKEN)
			typ := p.parseType()

			params.List = append(params.List, &ast.Field{
				Names: []*ast.IdentifierExpr{name},
				Type:  typ,
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
					Names: nil, // Anonymous return
					Type:  resultType,
				},
			},
		}
	}

	return &ast.FuncType{
		Params:  params,
		Results: results,
	}
}

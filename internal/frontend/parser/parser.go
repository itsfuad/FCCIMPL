package parser

import (
	"compiler/internal/diagnostics"
	"compiler/internal/frontend/ast"
	"compiler/internal/frontend/lexer"
	"compiler/internal/source"
	"fmt"
)

// ============================================================================
// PARSER - Token to AST Conversion
// ============================================================================
//
// The Parser builds an AST from a token stream.
// The ParseFile function is in the pipeline package to avoid import cycles.

// Parser holds temporary state during parsing of a single file.
// This is created on-the-fly, not stored persistently.
type Parser struct {
	tokens      []lexer.Token
	current     int
	diagnostics *diagnostics.DiagnosticBag
	filepath    string
}

// Parse is the internal parsing function called by the pipeline.
// It takes tokens and diagnostics as parameters to avoid import cycles.
func Parse(tokens []lexer.Token, filepath string, diag *diagnostics.DiagnosticBag) *ast.Module {
	state := &Parser{
		tokens:      tokens,
		current:     0,
		diagnostics: diag,
		filepath:    filepath,
	}

	return state.parseModule()
}

// parseModule parses the entire module (all top-level declarations)
func (p *Parser) parseModule() *ast.Module {
	module := &ast.Module{
		FullPath: p.filepath,
		Nodes:    []ast.Node{},
	}

	for !p.isAtEnd() {
		node := p.parseTopLevel()
		if node != nil {
			module.Nodes = append(module.Nodes, node)
		}
	}

	return module
}

// parseTopLevel parses a single top-level declaration
func (p *Parser) parseTopLevel() ast.Node {
	tok := p.peek()
	switch tok.Kind {
	case lexer.IMPORT_TOKEN:
		return p.parseImport()
	case lexer.LET_TOKEN:
		return p.parseVarDecl()
	case lexer.CONST_TOKEN:
		return p.parseConstDecl()
	case lexer.TYPE_TOKEN:
		return p.parseTypeDecl()
	case lexer.FUNCTION_TOKEN:
		return p.parseFuncDecl()
	default:
		p.error(fmt.Sprintf("unexpected token at top level: %s", tok.Value))
		p.advance()
		return nil
	}
}

// parseImport: import "path";
func (p *Parser) parseImport() *ast.ImportStmt {
	start := p.peek().Start
	p.expect(lexer.IMPORT_TOKEN)

	if !p.check(lexer.STRING_TOKEN) {
		p.error("expected string literal after 'import'")
		return nil
	}

	pathTok := p.advance()
	path := &ast.BasicLit{
		Kind:     ast.STRING,
		Value:    pathTok.Value,
		Location: *source.NewLocation(&pathTok.Start, &pathTok.End),
	}

	p.expect(lexer.SEMICOLON_TOKEN)

	return &ast.ImportStmt{
		Path:     path,
		Location: p.makeLocation(start),
	}
}

// parseStmt parses a statement
func (p *Parser) parseStmt() ast.Node {
	tok := p.peek()
	switch tok.Kind {
	case lexer.LET_TOKEN:
		return p.parseVarDecl()
	case lexer.CONST_TOKEN:
		return p.parseConstDecl()
	case lexer.RETURN_TOKEN:
		return p.parseReturnStmt()
	case lexer.IF_TOKEN:
		return p.parseIfStmt()
	case lexer.FOR_TOKEN:
		return p.parseForStmt()
	case lexer.WHILE_TOKEN:
		return p.parseWhileStmt()
	case lexer.OPEN_CURLY:
		return p.parseBlock()
	default:
		return p.parseExprOrAssign()
	}
}

// parseReturnStmt: return expr;
func (p *Parser) parseReturnStmt() *ast.ReturnStmt {
	start := p.peek().Start
	p.expect(lexer.RETURN_TOKEN)

	var result ast.Expression
	var isError bool

	if !p.check(lexer.SEMICOLON_TOKEN) {
		result = p.parseExpr()
		if p.match(lexer.NOT_TOKEN) {
			isError = true
		}
	}

	p.expect(lexer.SEMICOLON_TOKEN)

	return &ast.ReturnStmt{
		Result:   result,
		IsError:  isError,
		Location: p.makeLocation(start),
	}
}

// parseIfStmt: if cond { } else { }
func (p *Parser) parseIfStmt() *ast.IfStmt {
	start := p.peek().Start
	p.expect(lexer.IF_TOKEN)

	cond := p.parseExpr()
	body := p.parseBlock()

	var elseNode ast.Node
	if p.match(lexer.ELSE_TOKEN) {
		if p.check(lexer.IF_TOKEN) {
			elseNode = p.parseIfStmt()
		} else {
			elseNode = p.parseBlock()
		}
	}

	return &ast.IfStmt{
		Cond:     cond,
		Body:     body,
		Else:     elseNode,
		Location: p.makeLocation(start),
	}
}

// parseForStmt: for init; cond; post { }
func (p *Parser) parseForStmt() *ast.ForStmt {
	start := p.peek().Start
	p.expect(lexer.FOR_TOKEN)

	var init, cond, post ast.Expression

	// init
	if !p.check(lexer.SEMICOLON_TOKEN) {
		init = p.parseExpr()
	}
	p.expect(lexer.SEMICOLON_TOKEN)

	// cond
	if !p.check(lexer.SEMICOLON_TOKEN) {
		cond = p.parseExpr()
	}
	p.expect(lexer.SEMICOLON_TOKEN)

	// post
	if !p.check(lexer.OPEN_CURLY) {
		post = p.parseExpr()
	}

	body := p.parseBlock()

	return &ast.ForStmt{
		Init:     init,
		Cond:     cond,
		Post:     post,
		Body:     body,
		Location: p.makeLocation(start),
	}
}

// parseWhileStmt: while cond { }
func (p *Parser) parseWhileStmt() *ast.WhileStmt {
	start := p.peek().Start
	p.expect(lexer.WHILE_TOKEN)

	cond := p.parseExpr()
	body := p.parseBlock()

	return &ast.WhileStmt{
		Cond:     cond,
		Body:     body,
		Location: p.makeLocation(start),
	}
}

// parseExprOrAssign: expr; or expr = expr;
func (p *Parser) parseExprOrAssign() ast.Node {
	start := p.peek().Start
	lhs := p.parseExpr()

	// Check for assignment
	if p.match(lexer.EQUALS_TOKEN) {
		rhs := p.parseExpr()
		p.expect(lexer.SEMICOLON_TOKEN)

		return &ast.AssignStmt{
			Lhs:      lhs,
			Rhs:      rhs,
			Location: p.makeLocation(start),
		}
	}

	p.expect(lexer.SEMICOLON_TOKEN)

	return &ast.ExprStmt{
		X:        lhs,
		Location: p.makeLocation(start),
	}
}

// parseExpr parses an expression
func (p *Parser) parseExpr() ast.Expression {
	return p.parseLogicalOr()
}

func (p *Parser) parseLogicalOr() ast.Expression {
	left := p.parseLogicalAnd()

	for p.match(lexer.OR_TOKEN) {
		op := p.previous()
		right := p.parseLogicalAnd()
		left = &ast.BinaryExpr{
			X:  left,
			Op: op,
			Y:  right,
		}
	}

	return left
}

func (p *Parser) parseLogicalAnd() ast.Expression {
	left := p.parseEquality()

	for p.match(lexer.AND_TOKEN) {
		op := p.previous()
		right := p.parseEquality()
		left = &ast.BinaryExpr{
			X:  left,
			Op: op,
			Y:  right,
		}
	}

	return left
}

func (p *Parser) parseEquality() ast.Expression {
	left := p.parseComparison()

	for p.match(lexer.DOUBLE_EQUAL_TOKEN, lexer.NOT_EQUAL_TOKEN) {
		op := p.previous()
		right := p.parseComparison()
		left = &ast.BinaryExpr{
			X:  left,
			Op: op,
			Y:  right,
		}
	}

	return left
}

func (p *Parser) parseComparison() ast.Expression {
	left := p.parseAdditive()

	for p.match(lexer.LESS_TOKEN, lexer.LESS_EQUAL_TOKEN, lexer.GREATER_TOKEN, lexer.GREATER_EQUAL_TOKEN) {
		op := p.previous()
		right := p.parseAdditive()
		left = &ast.BinaryExpr{
			X:  left,
			Op: op,
			Y:  right,
		}
	}

	return left
}

func (p *Parser) parseAdditive() ast.Expression {
	left := p.parseMultiplicative()

	for p.match(lexer.PLUS_TOKEN, lexer.MINUS_TOKEN) {
		op := p.previous()
		right := p.parseMultiplicative()
		left = &ast.BinaryExpr{
			X:  left,
			Op: op,
			Y:  right,
		}
	}

	return left
}

func (p *Parser) parseMultiplicative() ast.Expression {
	left := p.parseUnary()

	for p.match(lexer.MUL_TOKEN, lexer.DIV_TOKEN, lexer.MOD_TOKEN) {
		op := p.previous()
		right := p.parseUnary()
		left = &ast.BinaryExpr{
			X:  left,
			Op: op,
			Y:  right,
		}
	}

	return left
}

func (p *Parser) parseUnary() ast.Expression {
	if p.match(lexer.NOT_TOKEN, lexer.MINUS_TOKEN) {
		op := p.previous()
		expr := p.parseUnary()
		return &ast.UnaryExpr{
			Op: op,
			X:  expr,
		}
	}

	return p.parsePostfix()
}

func (p *Parser) parsePostfix() ast.Expression {
	expr := p.parsePrimary()

	for {
		if p.match(lexer.OPEN_PAREN) {
			// Call
			args := []ast.Expression{}
			if !p.check(lexer.CLOSE_PAREN) {
				args = append(args, p.parseExpr())
				for p.match(lexer.COMMA_TOKEN) {
					args = append(args, p.parseExpr())
				}
			}
			p.expect(lexer.CLOSE_PAREN)

			expr = &ast.CallExpr{
				Fun:  expr,
				Args: args,
			}
		} else if p.match(lexer.OPEN_BRACKET) {
			// Index
			index := p.parseExpr()
			p.expect(lexer.CLOSE_BRACKET)

			expr = &ast.IndexExpr{
				X:     expr,
				Index: index,
			}
		} else if p.match(lexer.DOT_TOKEN) {
			// Selector
			sel := p.parseIdentifier()
			expr = &ast.SelectorExpr{
				X:   expr,
				Sel: sel,
			}
		} else if p.match(lexer.SCOPE_TOKEN) {
			// Scope resolution
			member := p.parseIdentifier()
			expr = &ast.ScopeResolutionExpr{
				X:   expr,
				Sel: member,
			}
		} else {
			break
		}
	}

	return expr
}

func (p *Parser) parsePrimary() ast.Expression {
	tok := p.peek()

	switch tok.Kind {
	case lexer.NUMBER_TOKEN:
		p.advance()
		// Determine if int or float
		kind := ast.INT
		for _, ch := range tok.Value {
			if ch == '.' {
				kind = ast.FLOAT
				break
			}
		}
		return &ast.BasicLit{
			Kind:     kind,
			Value:    tok.Value,
			Location: *source.NewLocation(&tok.Start, &tok.End),
		}

	case lexer.STRING_TOKEN:
		p.advance()
		return &ast.BasicLit{
			Kind:     ast.STRING,
			Value:    tok.Value,
			Location: *source.NewLocation(&tok.Start, &tok.End),
		}

	case lexer.BYTE_TOKEN:
		p.advance()
		return &ast.BasicLit{
			Kind:     ast.BYTE,
			Value:    tok.Value,
			Location: *source.NewLocation(&tok.Start, &tok.End),
		}

	case lexer.IDENTIFIER_TOKEN:
		return p.parseIdentifier()

	case lexer.OPEN_PAREN:
		p.advance()
		expr := p.parseExpr()
		p.expect(lexer.CLOSE_PAREN)
		return expr

	case lexer.OPEN_BRACKET:
		return p.parseArrayLiteral()

	default:
		p.error(fmt.Sprintf("unexpected token in expression: %s", tok.Value))
		p.advance()
		return nil
	}
}

func (p *Parser) parseArrayLiteral() *ast.CompositeLit {
	start := p.peek().Start
	p.expect(lexer.OPEN_BRACKET)

	elems := []ast.Expression{}
	if !p.check(lexer.CLOSE_BRACKET) {
		elems = append(elems, p.parseExpr())
		for p.match(lexer.COMMA_TOKEN) {
			if p.check(lexer.CLOSE_BRACKET) {
				break
			}
			elems = append(elems, p.parseExpr())
		}
	}

	p.expect(lexer.CLOSE_BRACKET)

	return &ast.CompositeLit{
		Elts:     elems,
		Location: p.makeLocation(start),
	}
}

func (p *Parser) parseIdentifier() *ast.IdentifierExpr {
	if !p.check(lexer.IDENTIFIER_TOKEN) {
		p.error(fmt.Sprintf("expected identifier, got %s", p.peek().Value))
		return &ast.IdentifierExpr{Name: "<error>"}
	}

	tok := p.advance()
	return &ast.IdentifierExpr{
		Name:     tok.Value,
		Location: *source.NewLocation(&tok.Start, &tok.End),
	}
}

// Helper methods

func (p *Parser) isAtEnd() bool {
	if p.current >= len(p.tokens) {
		return true
	}
	return p.tokens[p.current].Kind == lexer.EOF_TOKEN
}

func (p *Parser) peek() lexer.Token {
	if p.current >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[p.current]
}

func (p *Parser) previous() lexer.Token {
	return p.tokens[p.current-1]
}

func (p *Parser) advance() lexer.Token {
	if !p.isAtEnd() {
		p.current++
	}
	return p.previous()
}

func (p *Parser) check(kind lexer.TOKEN) bool {
	if p.isAtEnd() {
		return false
	}
	return p.peek().Kind == kind
}

func (p *Parser) match(kinds ...lexer.TOKEN) bool {
	for _, kind := range kinds {
		if p.check(kind) {
			p.advance()
			return true
		}
	}
	return false
}

func (p *Parser) expect(kind lexer.TOKEN) lexer.Token {
	if p.check(kind) {
		return p.advance()
	}

	p.error(fmt.Sprintf("expected %v but got %s", kind, p.peek().Value))
	return p.peek()
}

// error reports a parsing error to the diagnostics
func (p *Parser) error(msg string) {
	tok := p.peek()
	loc := source.NewLocation(&tok.Start, &tok.End)
	p.diagnostics.Add(
		diagnostics.NewError(msg).
			WithCode("P0001").
			WithPrimaryLabel(p.filepath, loc, ""),
	)
}

// makeLocation creates a source location from start to current position
func (p *Parser) makeLocation(start source.Position) source.Location {
	end := p.previous().End
	return *source.NewLocation(&start, &end)
}

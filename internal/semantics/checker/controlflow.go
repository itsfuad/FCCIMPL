package checker

import (
	"compiler/internal/diagnostics"
	"compiler/internal/frontend/ast"
	"compiler/internal/semantics"
	"fmt"
)

type FlowResult struct {
	Returns    bool
	Breaks     bool
	Continues  bool
	ReturnType semantics.Type
	ReturnLoc  interface{}
}

func do() string {
    s := 10;
    //s = "2";

    if s > 10 {
        return "large shape";
    } else {
        //return "small shape";
        if s == 10 {
            return "medium shape";
        } else {
            //return "small shape";
        }
        return "small shape";
    }

    //return "s";
}

func (c *Checker) checkFunctionReturns(decl *ast.FuncDecl, funcType *semantics.FunctionType) {
	if decl.Body == nil {
		return
	}

	result := c.analyzeFlow(decl.Body, false)

	if funcType.ReturnType != nil {
		if !result.Returns {
			c.ctx.Diagnostics.Add(
				diagnostics.NewError(
					fmt.Sprintf("missing return in function with return type %s", c.typeString(funcType.ReturnType)),
				).
					WithCode(diagnostics.ErrMissingReturn).
					WithPrimaryLabel(c.currentFile, decl.Name.Loc(), "function must return a value"),
			)
		}
	}
}

func (c *Checker) analyzeFlow(node ast.Node, inLoop bool) FlowResult {
	if node == nil {
		return FlowResult{}
	}

	switch n := node.(type) {
	case *ast.Block:
		return c.analyzeBlockFlow(n, inLoop)
	case *ast.ReturnStmt:
		returnType := c.checkReturnStmt(n)
		return FlowResult{Returns: true, ReturnType: returnType, ReturnLoc: n}
	case *ast.IfStmt:
		return c.analyzeIfFlow(n, inLoop)
	case *ast.ForStmt:
		return c.analyzeForFlow(n)
	case *ast.WhileStmt:
		return c.analyzeWhileFlow(n)
	case *ast.BreakStmt:
		if !inLoop {
			c.ctx.Diagnostics.Add(
				diagnostics.NewError("break statement outside loop").
					WithCode(diagnostics.ErrInvalidBreak).
					WithPrimaryLabel(c.currentFile, n.Loc(), "not in a loop"),
			)
		}
		return FlowResult{Breaks: true}
	case *ast.ContinueStmt:
		if !inLoop {
			c.ctx.Diagnostics.Add(
				diagnostics.NewError("continue statement outside loop").
					WithCode(diagnostics.ErrInvalidContinue).
					WithPrimaryLabel(c.currentFile, n.Loc(), "not in a loop"),
			)
		}
		return FlowResult{Continues: true}
	default:
		c.checkNode(n)
		return FlowResult{}
	}
}

func (c *Checker) analyzeBlockFlow(block *ast.Block, inLoop bool) FlowResult {
	prevScope := c.currentScope
	if blockScope := c.ctx.GetBlockScope(block); blockScope != nil {
		c.currentScope = blockScope
	}

	var result FlowResult
	for i, node := range block.Nodes {
		// Check if previous statements already had terminal control flow
		if i > 0 && (result.Returns || result.Breaks || result.Continues) {
			c.ctx.Diagnostics.Add(
				diagnostics.NewWarning("unreachable code").
					WithCode(diagnostics.WarnUnreachableCode).
					WithPrimaryLabel(c.currentFile, node.Loc(), "this code will never execute").
					WithHelp("remove this code"),
			)
			break // Don't process unreachable statements
		}

		nodeResult := c.analyzeFlow(node, inLoop)

		if nodeResult.Returns {
			result.Returns = true
			result.ReturnType = nodeResult.ReturnType
			result.ReturnLoc = nodeResult.ReturnLoc
		}
		if nodeResult.Breaks {
			result.Breaks = true
		}
		if nodeResult.Continues {
			result.Continues = true
		}
	}

	c.currentScope = prevScope
	return result
}

func (c *Checker) analyzeIfFlow(stmt *ast.IfStmt, inLoop bool) FlowResult {
	c.checkExpr(stmt.Cond)

	thenNarrowings, elseNarrowings := c.analyzeTypeNarrowing(stmt.Cond)

	var thenResult FlowResult
	if stmt.Body != nil {
		blockScope := c.ctx.GetBlockScope(stmt.Body)
		if blockScope != nil {
			c.applyNarrowingsInScope(blockScope, thenNarrowings, func() {
				thenResult = c.analyzeFlow(stmt.Body, inLoop)
			})
		} else {
			thenResult = c.analyzeFlow(stmt.Body, inLoop)
		}
	}

	var elseResult FlowResult
	if stmt.Else != nil {
		switch e := stmt.Else.(type) {
		case *ast.Block:
			blockScope := c.ctx.GetBlockScope(e)
			if blockScope != nil {
				c.applyNarrowingsInScope(blockScope, elseNarrowings, func() {
					elseResult = c.analyzeFlow(e, inLoop)
				})
			} else {
				elseResult = c.analyzeFlow(e, inLoop)
			}
		case *ast.IfStmt:
			c.applyNarrowings(elseNarrowings, func() {
				elseResult = c.analyzeFlow(e, inLoop)
			})
		}
	}

	result := FlowResult{}

	if thenResult.Returns && elseResult.Returns {
		result.Returns = true
		result.ReturnType = thenResult.ReturnType
		result.ReturnLoc = thenResult.ReturnLoc
	}

	if stmt.Else != nil && thenResult.Breaks && elseResult.Breaks {
		result.Breaks = true
	}

	if stmt.Else != nil && thenResult.Continues && elseResult.Continues {
		result.Continues = true
	}

	return result
}

func (c *Checker) analyzeForFlow(stmt *ast.ForStmt) FlowResult {
	if stmt.Init != nil {
		c.checkNode(stmt.Init)
	}
	if stmt.Cond != nil {
		c.checkExpr(stmt.Cond)
	}
	if stmt.Post != nil {
		c.checkNode(stmt.Post)
	}
	if stmt.Body != nil {
		c.analyzeFlow(stmt.Body, true)
	}
	return FlowResult{}
}

func (c *Checker) analyzeWhileFlow(stmt *ast.WhileStmt) FlowResult {
	if stmt.Cond != nil {
		c.checkExpr(stmt.Cond)
	}
	if stmt.Body != nil {
		c.analyzeFlow(stmt.Body, true)
	}
	return FlowResult{}
}

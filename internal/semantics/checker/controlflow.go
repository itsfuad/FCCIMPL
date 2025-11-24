package checker

import (
	"compiler/internal/diagnostics"
	"compiler/internal/frontend/ast"
	"compiler/internal/semantics"
	"compiler/internal/source"
	"fmt"
)

// FlowStepKind identifies what kind of branch decision a step represents
type FlowStepKind int

const (
	StepFunctionBody FlowStepKind = iota
	StepIfThen
	StepIfElse
	StepLoopBody
)

// FlowStep represents a single branch in a control flow path
type FlowStep struct {
	Kind FlowStepKind
	Loc  *source.Location
}

// FlowPath represents a complete control flow path that can reach the end without returning
type FlowPath struct {
	Steps []FlowStep       // branch trace (function body → else → then etc.)
	Loc   *source.Location // where this path falls through
}

// FlowResult represents the control flow analysis of a statement or block
type FlowResult struct {
	AlwaysReturns bool       // true if ALL paths through this node return
	MissingPaths  []FlowPath // paths that can reach end without returning
}

// checkFunctionReturns analyzes whether a function properly returns on all paths
func (c *Checker) checkFunctionReturns(decl *ast.FuncDecl, funcType *semantics.FunctionType) {
	// No return type = nothing to enforce
	if funcType.ReturnType == nil {
		return
	}

	if decl.Body == nil {
		return
	}

	// Analyze body
	basePath := []FlowStep{{
		Kind: StepFunctionBody,
		Loc:  decl.Name.Loc(),
	}}
	flow := c.analyzeBlock(decl.Body, basePath)

	if flow.AlwaysReturns {
		return
	}

	// At least one path fails to return → build diagnostic
	diag := diagnostics.NewError(
		fmt.Sprintf("not all code paths in function '%s' return a value of type %s",
			decl.Name.Name, c.typeString(funcType.ReturnType)),
	).
		WithCode(diagnostics.ErrMissingReturn).
		WithPrimaryLabel(c.currentFile, decl.Name.Loc(),
			"function may exit without returning a value")

	// Add detailed notes for failing paths (limit to 3 for readability)
	for i, p := range flow.MissingPaths {
		if i >= 3 {
			diag = diag.WithNote("additional non-returning paths omitted")
			break
		}

		desc := c.describeFlowPath(p.Steps)
		diag = diag.WithSecondaryLabel(
			c.currentFile,
			p.Loc,
			fmt.Sprintf("this path (%s) can reach end of function without returning", desc),
		)
	}

	c.ctx.Diagnostics.Add(diag)
}

// analyzeBlock analyzes a block for return paths
func (c *Checker) analyzeBlock(block *ast.Block, basePath []FlowStep) FlowResult {
	prevScope := c.currentScope
	if blockScope := c.ctx.GetBlockScope(block); blockScope != nil {
		c.currentScope = blockScope
	}

	var res FlowResult
	var lastWasIfWithoutElse bool

	for i, node := range block.Nodes {
		// Check if previous statement was terminal (return, break, continue, or if/else always exits)
		if i > 0 && res.AlwaysReturns {
			c.ctx.Diagnostics.Add(
				diagnostics.NewWarning("unreachable code").
					WithCode(diagnostics.WarnUnreachableCode).
					WithPrimaryLabel(c.currentFile, node.Loc(), "this code will never execute").
					WithHelp("remove this code"),
			)
			// Continue scanning for more unreachable code
			continue
		}

		stmtRes := c.analyzeStmt(node, basePath)

		// Track if last statement was an if without else
		if ifStmt, ok := node.(*ast.IfStmt); ok {
			lastWasIfWithoutElse = ifStmt.Else == nil
		} else {
			lastWasIfWithoutElse = false
		}

		// propagate inner missing paths
		res.MissingPaths = append(res.MissingPaths, stmtRes.MissingPaths...)

		if stmtRes.AlwaysReturns {
			res.AlwaysReturns = true
			// Don't return yet - continue to detect unreachable code
		}
	}

	c.currentScope = prevScope

	// If block doesn't always return, add a fallthrough path for the block itself
	// UNLESS the last statement was an if without else (implicit else already creates fallthrough path)
	if !res.AlwaysReturns && len(basePath) > 0 && !lastWasIfWithoutElse {
		res.MissingPaths = append(res.MissingPaths, FlowPath{
			Steps: basePath,
			Loc:   block.Loc(),
		})
	}

	return res
}

// analyzeStmt analyzes a single statement for return paths
func (c *Checker) analyzeStmt(node ast.Node, basePath []FlowStep) FlowResult {
	if node == nil {
		return FlowResult{}
	}

	switch n := node.(type) {
	case *ast.Block:
		return c.analyzeBlock(n, basePath)
	case *ast.ReturnStmt:
		c.checkReturnStmt(n)
		return FlowResult{AlwaysReturns: true}
	case *ast.IfStmt:
		return c.analyzeIfStmt(n, basePath)
	case *ast.ForStmt:
		return c.analyzeForStmt(n, basePath)
	case *ast.WhileStmt:
		return c.analyzeWhileStmt(n, basePath)
	case *ast.BreakStmt:
		c.ctx.Diagnostics.Add(
			diagnostics.NewError("break statement outside loop").
				WithCode(diagnostics.ErrInvalidBreak).
				WithPrimaryLabel(c.currentFile, n.Loc(), "not in a loop"),
		)
		return FlowResult{}
	case *ast.ContinueStmt:
		c.ctx.Diagnostics.Add(
			diagnostics.NewError("continue statement outside loop").
				WithCode(diagnostics.ErrInvalidContinue).
				WithPrimaryLabel(c.currentFile, n.Loc(), "not in a loop"),
		)
		return FlowResult{}
	default:
		c.checkNode(n)
		return FlowResult{}
	}
}

// analyzeIfStmt analyzes an if statement for return paths
func (c *Checker) analyzeIfStmt(stmt *ast.IfStmt, basePath []FlowStep) FlowResult {
	c.checkExpr(stmt.Cond)

	// then branch
	thenPath := make([]FlowStep, len(basePath))
	copy(thenPath, basePath)
	thenPath = append(thenPath, FlowStep{
		Kind: StepIfThen,
		Loc:  stmt.Cond.Loc(),
	})

	var thenRes FlowResult
	if stmt.Body != nil {
		thenRes = c.analyzeBlock(stmt.Body, thenPath)
	}

	// else branch
	var elseRes FlowResult
	if stmt.Else != nil {
		elsePath := make([]FlowStep, len(basePath))
		copy(elsePath, basePath)
		elsePath = append(elsePath, FlowStep{
			Kind: StepIfElse,
			Loc:  stmt.Loc(),
		})

		switch e := stmt.Else.(type) {
		case *ast.Block:
			elseRes = c.analyzeBlock(e, elsePath)
		case *ast.IfStmt:
			elseRes = c.analyzeIfStmt(e, elsePath)
		}
	} else {
		// implicit else that does nothing → fallthrough path
		elseRes.AlwaysReturns = false
		elsePath := make([]FlowStep, len(basePath))
		copy(elsePath, basePath)
		elsePath = append(elsePath, FlowStep{
			Kind: StepIfElse,
			Loc:  stmt.Loc(),
		})
		elseRes.MissingPaths = []FlowPath{{
			Steps: elsePath,
			Loc:   stmt.Loc(),
		}}
	}

	result := FlowResult{
		AlwaysReturns: thenRes.AlwaysReturns && elseRes.AlwaysReturns,
		MissingPaths:  append(thenRes.MissingPaths, elseRes.MissingPaths...),
	}

	return result
}

// analyzeForStmt analyzes a for loop
func (c *Checker) analyzeForStmt(stmt *ast.ForStmt, basePath []FlowStep) FlowResult {
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
		loopPath := make([]FlowStep, len(basePath))
		copy(loopPath, basePath)
		loopPath = append(loopPath, FlowStep{
			Kind: StepLoopBody,
			Loc:  stmt.Loc(),
		})
		c.analyzeBlockInLoop(stmt.Body, loopPath)
	}

	// We can't assume loop runs; so it doesn't guarantee a return.
	return FlowResult{}
}

// analyzeWhileStmt analyzes a while loop
func (c *Checker) analyzeWhileStmt(stmt *ast.WhileStmt, basePath []FlowStep) FlowResult {
	if stmt.Cond != nil {
		c.checkExpr(stmt.Cond)
	}

	if stmt.Body != nil {
		loopPath := make([]FlowStep, len(basePath))
		copy(loopPath, basePath)
		loopPath = append(loopPath, FlowStep{
			Kind: StepLoopBody,
			Loc:  stmt.Loc(),
		})
		c.analyzeBlockInLoop(stmt.Body, loopPath)
	}

	// We can't assume loop runs; so it doesn't guarantee a return.
	return FlowResult{}
}

// analyzeBlockInLoop analyzes a block inside a loop (ignoring break/continue for return purposes)
func (c *Checker) analyzeBlockInLoop(block *ast.Block, basePath []FlowStep) FlowResult {
	prevScope := c.currentScope
	if blockScope := c.ctx.GetBlockScope(block); blockScope != nil {
		c.currentScope = blockScope
	}

	var res FlowResult

	for _, node := range block.Nodes {
		switch n := node.(type) {
		case *ast.BreakStmt:
			// break is allowed in loops
		case *ast.ContinueStmt:
			// continue is allowed in loops
		default:
			stmtRes := c.analyzeStmt(n, basePath)
			res.MissingPaths = append(res.MissingPaths, stmtRes.MissingPaths...)
			if stmtRes.AlwaysReturns {
				res.AlwaysReturns = true
				c.currentScope = prevScope
				return res
			}
		}
	}

	c.currentScope = prevScope
	return res
}

// isTerminalStatement checks if a statement always exits its block
// (either via return, break, or continue)
func (c *Checker) isTerminalStatement(stmt ast.Node) bool {
	if stmt == nil {
		return false
	}

	switch s := stmt.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.BreakStmt:
		return true
	case *ast.ContinueStmt:
		return true
	case *ast.IfStmt:
		// if/else is terminal only if BOTH branches are terminal
		if s.Else == nil {
			return false // no else means execution can fall through
		}

		thenTerminal := false
		if s.Body != nil {
			thenTerminal = c.blockIsTerminal(s.Body)
		}

		elseTerminal := false
		switch e := s.Else.(type) {
		case *ast.Block:
			elseTerminal = c.blockIsTerminal(e)
		case *ast.IfStmt:
			elseTerminal = c.isTerminalStatement(e)
		}

		return thenTerminal && elseTerminal

	case *ast.Block:
		return c.blockIsTerminal(s)

	default:
		return false
	}
}

// blockIsTerminal checks if a block always exits (all paths are terminal)
func (c *Checker) blockIsTerminal(block *ast.Block) bool {
	if block == nil || len(block.Nodes) == 0 {
		return false
	}

	for _, node := range block.Nodes {
		if c.isTerminalStatement(node) {
			return true // found terminal statement, rest is unreachable
		}
	}

	return false
}

// describeFlowPath converts a flow path to a human-readable description
func (c *Checker) describeFlowPath(steps []FlowStep) string {
	if len(steps) == 0 {
		return "function body"
	}

	parts := []string{}
	for _, step := range steps {
		switch step.Kind {
		case StepFunctionBody:
			parts = append(parts, "function body")
		case StepIfThen:
			if step.Loc != nil && step.Loc.Start != nil {
				parts = append(parts, fmt.Sprintf("if-branch at line %d", step.Loc.Start.Line))
			} else {
				parts = append(parts, "if-branch")
			}
		case StepIfElse:
			if step.Loc != nil && step.Loc.Start != nil {
				parts = append(parts, fmt.Sprintf("else-branch at line %d", step.Loc.Start.Line))
			} else {
				parts = append(parts, "else-branch")
			}
		case StepLoopBody:
			if step.Loc != nil && step.Loc.Start != nil {
				parts = append(parts, fmt.Sprintf("loop-body at line %d", step.Loc.Start.Line))
			} else {
				parts = append(parts, "loop-body")
			}
		}
	}

	// Build chain description
	result := ""
	for i, part := range parts {
		if i == 0 {
			result = part
		} else {
			result += " → " + part
		}
	}
	return result
}

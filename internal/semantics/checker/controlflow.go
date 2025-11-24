//controlflow.go
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

	// Prune redundant parent paths if deeper child paths already explain the issue
	originalCount := len(flow.MissingPaths)
	missing := pruneMissingPaths(flow.MissingPaths)
	if len(missing) == 0 {
		// Defensive fallback (shouldn't happen)
		missing = flow.MissingPaths
	}

	// Build diagnostic
	diag := diagnostics.NewError(
		fmt.Sprintf("not all code paths in function '%s' return a value of type %s",
			decl.Name.Name, c.typeString(funcType.ReturnType)),
	).
		WithCode(diagnostics.ErrMissingReturn).
		WithPrimaryLabel(c.currentFile, decl.Name.Loc(),
			"missing return on some paths").
		WithHelp("make sure every branch returns, or add a final return at the end of the function")

	// If pruning hid parent paths, add a simple beginner-friendly note
	if len(missing) < originalCount {
		diag = diag.WithNote("fix these branches or add a final return at the end")
	}

	// Add detailed notes for failing paths (limit to 3 for readability)
	for i, p := range missing {
		if i >= 3 {
			diag = diag.WithNote("additional non-returning paths omitted")
			break
		}

		last := p.Steps[len(p.Steps)-1]
		desc := detectPath(last)

		diag = diag.WithSecondaryLabel(
			c.currentFile,
			p.Loc,
			fmt.Sprintf("missing return in %s", desc),
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

	thenLoc := stmt.Body.Loc() // use THEN block's own loc
	thenPath = append(thenPath, FlowStep{
		Kind: StepIfThen,
		Loc:  thenLoc,
	})

	thenRes := FlowResult{}
	if stmt.Body != nil {
		thenRes = c.analyzeBlock(stmt.Body, thenPath)
	}

	// else branch
	elseRes := FlowResult{}
	if stmt.Else != nil {
		elsePath := make([]FlowStep, len(basePath))
		copy(elsePath, basePath)

		elseLoc := stmt.Else.Loc() // use ELSE node's own loc
		elsePath = append(elsePath, FlowStep{
			Kind: StepIfElse,
			Loc:  elseLoc,
		})

		switch e := stmt.Else.(type) {
		case *ast.Block:
			elseRes = c.analyzeBlock(e, elsePath)
		case *ast.IfStmt:
			elseRes = c.analyzeIfStmt(e, elsePath)
		}
	} else {
		// implicit else (no node to own a location)
		elsePath := make([]FlowStep, len(basePath))
		copy(elsePath, basePath)
		elsePath = append(elsePath, FlowStep{
			Kind: StepIfElse,
			Loc:  stmt.Loc(), // fallback only because implicit else has no loc
		})

		elseRes.AlwaysReturns = false
		elseRes.MissingPaths = []FlowPath{{
			Steps: elsePath,
			Loc:   stmt.Loc(),
		}}
	}

	return FlowResult{
		AlwaysReturns: thenRes.AlwaysReturns && elseRes.AlwaysReturns,
		MissingPaths:  append(thenRes.MissingPaths, elseRes.MissingPaths...),
	}
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

func detectPath(step FlowStep) string {
	var part string
	switch step.Kind {
	case StepFunctionBody:
		part = "function"
	case StepIfThen:
		if step.Loc != nil && step.Loc.Start != nil {
			part = fmt.Sprintf("if at line %d", step.Loc.Start.Line)
		} else {
			part = "if"
		}
	case StepIfElse:
		if step.Loc != nil && step.Loc.Start != nil {
			part = fmt.Sprintf("else at line %d", step.Loc.Start.Line)
		} else {
			part = "else"
		}
	case StepLoopBody:
		if step.Loc != nil && step.Loc.Start != nil {
			part = fmt.Sprintf("loop at line %d", step.Loc.Start.Line)
		} else {
			part = "loop"
		}
	}
	return part
}

//
// --------- Path pruning helpers ---------
//
// Goal: If a parent path is just a prefix of a deeper child path,
// show only the deeper child path.
//

func pruneMissingPaths(paths []FlowPath) []FlowPath {
	if len(paths) <= 1 {
		return dedupeMissingPaths(paths)
	}

	paths = dedupeMissingPaths(paths)

	out := make([]FlowPath, 0, len(paths))
	for i, p := range paths {
		if isPrefixOfAnyLonger(p, paths, i) {
			// child paths already explain the fallthrough more precisely
			continue
		}
		out = append(out, p)
	}
	return out
}

func isPrefixOfAnyLonger(p FlowPath, all []FlowPath, selfIdx int) bool {
	for j, q := range all {
		if j == selfIdx {
			continue
		}
		if len(q.Steps) <= len(p.Steps) {
			continue
		}
		if stepsIsPrefix(p.Steps, q.Steps) {
			return true
		}
	}
	return false
}

func stepsIsPrefix(a, b []FlowStep) bool {
	if len(a) > len(b) {
		return false
	}
	for i := range a {
		if !flowStepEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func flowStepEqual(x, y FlowStep) bool {
	if x.Kind != y.Kind {
		return false
	}
	// Compare by line number if available; otherwise both must be nil-line.
	return stepLine(x) == stepLine(y)
}

func stepLine(s FlowStep) int {
	if s.Loc != nil && s.Loc.Start != nil {
		return s.Loc.Start.Line
	}
	return -1
}

func dedupeMissingPaths(paths []FlowPath) []FlowPath {
	if len(paths) <= 1 {
		return paths
	}

	seen := map[string]bool{}
	out := make([]FlowPath, 0, len(paths))

	for _, p := range paths {
		key := serializePath(p)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, p)
	}
	return out
}

func serializePath(p FlowPath) string {
	// Stable key: kinds + line numbers + fallthrough loc line.
	key := ""
	for _, s := range p.Steps {
		key += fmt.Sprintf("%d@%d|", s.Kind, stepLine(s))
	}
	locLine := -1
	if p.Loc != nil && p.Loc.Start != nil {
		locLine = p.Loc.Start.Line
	}
	key += fmt.Sprintf("loc@%d", locLine)
	return key
}

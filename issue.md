Type Checker Design Notes: Return Types & Missing-Return Diagnostics
1. Tracking the Current Function’s Return Type
1.1 Problem

Right now the checker keeps:

type Checker struct {
    ctx          *context.CompilerContext
    currentScope *semantics.SymbolTable
    currentFile  string

    currentReturnType semantics.Type
}


checkFuncDecl sets currentReturnType before walking the body and resets it afterward. With nested functions, this becomes:

fragile (you must always save/restore correctly),

duplicated state (the scope tree already encodes where we are),

error‑prone as the language grows (lambdas, async, generics, etc.).

Conceptually:

The expected return type at any return is determined by the nearest enclosing function scope, not by global checker state.

1.2 Design Goal

Make the model match the language:

Return type is a property of the function scope.

At a return, we derive the expected type by walking the scope chain, not by reading a mutable field in Checker.

1.3 Solution: Attach Return Type to Function Scope

Step 1 — Extend SymbolTable with scope metadata

type ScopeKind int

const (
    ScopeModule ScopeKind = iota
    ScopeFunction
    ScopeBlock
)

type SymbolTable struct {
    Parent     *SymbolTable
    ScopeKind  ScopeKind
    ReturnType semantics.Type // only meaningful for ScopeFunction
    // ... existing fields
}


Step 2 — Mark function scopes when building them

Wherever we create sym.SelfScope for a function:

funcScope := semantics.NewSymbolTable(parent)
funcScope.ScopeKind  = ScopeFunction
funcScope.ReturnType = funcType.ReturnType
sym.SelfScope = funcScope


Step 3 — Remove currentReturnType from Checker

Delete the field and stop assigning to it in checkFuncDecl.

checkFuncDecl just manages currentScope:

func (c *Checker) checkFuncDecl(decl *ast.FuncDecl) semantics.Type {
    sym, ok := c.currentScope.Lookup(decl.Name.Name)
    if !ok || sym.SelfScope == nil {
        return nil
    }

    funcType, ok := sym.Type.(*semantics.FunctionType)
    if !ok {
        return sym.Type
    }

    prevScope := c.currentScope
    c.currentScope = sym.SelfScope

    if decl.Body != nil {
        c.checkFunctionReturns(decl, funcType)
    }

    c.currentScope = prevScope
    return sym.Type
}


Step 4 — Resolve expected return type from scope

Add a helper:

func (c *Checker) enclosingReturnType() semantics.Type {
    for sc := c.currentScope; sc != nil; sc = sc.Parent {
        if sc.ScopeKind == ScopeFunction {
            return sc.ReturnType
        }
    }
    return nil // not inside a function
}


Use it in checkReturnStmt:

func (c *Checker) checkReturnStmt(stmt *ast.ReturnStmt) semantics.Type {
    var valueType semantics.Type
    if stmt.Result != nil {
        valueType = c.checkExpr(stmt.Result)
    }

    expected := c.enclosingReturnType()

    if expected != nil {
        if valueType == nil {
            // function expects a value, but this return has none
            // (emit ErrInvalidReturn)
        } else if !c.isAssignable(expected, valueType) {
            // (emit ErrTypeMismatch)
        }
    } else if valueType != nil {
        // return with a value outside any function
        // (emit ErrInvalidReturn)
    }

    return valueType
}


Result

Single source of truth: function scope has the return type.

Nested functions “just work”: nearest ScopeFunction wins.

No fragile global currentReturnType state.

2. “Not All Paths Return” – Pinpointing the Problematic Branch
2.1 Problem

Example:

fn printShapeInfo(s: Shape) -> str {
    let s := 10;

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


The function promises -> str. The rule:

Every possible control-flow path must return a str.

Currently the compiler can say “missing return” or “type mismatch”, but:

it doesn’t say which branch / path can fall off the end,

this gets painful with nested if/else/loops.

We need diagnostics like:

“The else-branch of if s > 10, specifically the else-branch of if s == 10, may exit the function without returning a value.”

2.2 Design Goal

For each function with a non-none return type:

Decide: Does the body guarantee a return on all paths?

If not, record which paths can fall through, with locations and a readable trace of the branch decisions.

We can do this structurally over the AST (no need for a full CFG).

2.3 Solution: A Small Return-Flow Analysis

Define a result for statements/blocks:

type FlowStepKind int

const (
    StepFunctionBody FlowStepKind = iota
    StepIfThen
    StepIfElse
    StepLoopBody
)

type FlowStep struct {
    Kind FlowStepKind
    Loc  *source.Location
}

type FlowPath struct {
    Steps []FlowStep       // branch trace (function body → else → then etc.)
    Loc   *source.Location // where this path falls through
}

type FlowResult struct {
    AlwaysReturns bool       // true if ALL paths through this node return
    MissingPaths  []FlowPath // paths that can reach end without returning
}

2.3.1 Block { ... }
func (c *Checker) analyzeBlock(block *ast.Block, basePath []FlowStep) FlowResult {
    var res FlowResult

    for _, stmt := range block.Nodes {
        stmtRes := c.analyzeStmt(stmt, basePath)

        // propagate inner missing paths
        res.MissingPaths = append(res.MissingPaths, stmtRes.MissingPaths...)

        if stmtRes.AlwaysReturns {
            res.AlwaysReturns = true
            return res // later stmts are unreachable for "must return" purposes
        }
    }

    res.AlwaysReturns = false
    return res
}


(Optionally, you can add a final “falls through here” FlowPath, but often you only need that at function level.)

2.3.2 If statement
func (c *Checker) analyzeIf(stmt *ast.IfStmt, basePath []FlowStep) FlowResult {
    // then branch
    thenPath := append(append([]FlowStep(nil), basePath...), FlowStep{
        Kind: StepIfThen,
        Loc:  stmt.Cond.Loc(),
    })
    thenRes := c.analyzeBlock(stmt.Body, thenPath)

    var elseRes FlowResult
    if stmt.Else != nil {
        elsePath := append(append([]FlowStep(nil), basePath...), FlowStep{
            Kind: StepIfElse,
            Loc:  stmt.Loc(),
        })

        switch e := stmt.Else.(type) {
        case *ast.Block:
            elseRes = c.analyzeBlock(e, elsePath)
        case *ast.IfStmt:
            elseRes = c.analyzeIf(e, elsePath)
        }
    } else {
        // implicit else that does nothing → fallthrough path
        elseRes.AlwaysReturns = false
        elseRes.MissingPaths = []FlowPath{{
            Steps: append(append([]FlowStep(nil), basePath...), FlowStep{
                Kind: StepIfElse,
                Loc:  stmt.Loc(),
            }),
            Loc: stmt.Loc(),
        }}
    }

    return FlowResult{
        AlwaysReturns: thenRes.AlwaysReturns && elseRes.AlwaysReturns,
        MissingPaths:  append(thenRes.MissingPaths, elseRes.MissingPaths...),
    }
}

2.3.3 Return statement
func (c *Checker) analyzeReturn(stmt *ast.ReturnStmt, basePath []FlowStep) FlowResult {
    // type checking happens separately in checkReturnStmt
    return FlowResult{
        AlwaysReturns: true,
        MissingPaths:  nil,
    }
}

2.3.4 Loops (conservative)
func (c *Checker) analyzeWhile(stmt *ast.WhileStmt, basePath []FlowStep) FlowResult {
    loopPath := append(append([]FlowStep(nil), basePath...), FlowStep{
        Kind: StepLoopBody,
        Loc:  stmt.Loc(),
    })
    bodyRes := c.analyzeBlock(stmt.Body, loopPath)

    // We can't assume loop runs; so it doesn't guarantee a return.
    return FlowResult{
        AlwaysReturns: false,
        MissingPaths:  bodyRes.MissingPaths,
    }
}


(Same idea for for.)

2.3.5 Hook into checkFunctionReturns

checkFuncDecl already calls checkFunctionReturns(decl, funcType); we make that function authoritative for “does this function actually return?”.

func (c *Checker) checkFunctionReturns(decl *ast.FuncDecl, ft *semantics.FunctionType) {
    // No return type = nothing to enforce
    if ft.ReturnType == nil {
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
            decl.Name.Name, c.typeString(ft.ReturnType)),
    ).
        WithCode(diagnostics.ErrInvalidReturn).
        WithPrimaryLabel(c.currentFile, decl.Name.Loc(),
            "function may exit without returning a value")

    // Add detailed notes for failing paths (optionally limit the number)
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


A simple describeFlowPath can turn a list of steps into something like:

function body → else-branch of if at line 5 → else-branch of if at line 9.

2.4 What the user sees

For a function like printShapeInfo where some nested else path doesn’t return, the user gets:

A primary error on the function declaration:
not all code paths in function 'printShapeInfo' return a value of type str.

One or more secondary labels on specific branches, e.g.:
this path (else-branch of if at line 5 → else-branch of if at line 9) can reach end of function without returning.

This makes it obvious which branch is missing the return instead of a vague “missing return somewhere”.
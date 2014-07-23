package peggen

import (
	"github.com/neelance/jetpeg"
	"go/ast"
	"go/token"
	"strconv"
)

type Context struct {
}

func (c *Context) compileExpr(expr ParsingExpression, onFailure func() []ast.Stmt) []ast.Stmt {
	switch e := expr.(type) {
	case *StringTerminal:
		str, err := strconv.Unquote(`"` + e.Chars.String() + `"`)
		if err != nil {
			panic(err)
		}
		hasPrefixFun := "HasPrefix"
		if e.Fold {
			hasPrefixFun = "HasPrefixFold"
		}
		return []ast.Stmt{
			&ast.IfStmt{
				Cond: not(peglibCall(hasPrefixFun, input, stringConst(string(str)))),
				Body: &ast.BlockStmt{List: onFailure()},
			},
			consumeInput(intConst(len(str))),
		}

	case *CharacterClassTerminal:
		if len(e.Selections) == 0 {
			return []ast.Stmt{
				consumeInput(intConst(1)),
			}
		}

		unquoteChar := func(s jetpeg.Stringer) rune {
			str := s.String()
			switch str {
			case `\-`:
				return '-'
			case `\0`:
				return 0
			}
			char, _, _, err := strconv.UnquoteChar(str, 0)
			if err != nil {
				panic(err)
			}
			return char
		}
		var selections []rune
		for _, sel := range e.Selections {
			switch s := sel.(type) {
			case *CharacterClassSingleCharacter:
				selections = append(selections, unquoteChar(s.Char))
			case *CharacterClassRange:
				for i := unquoteChar(s.BeginChar); i <= unquoteChar(s.EndChar); i++ {
					selections = append(selections, i)
				}
			}
		}

		cond := peglibCall("ContainsByte", stringConst(string(selections)), &ast.IndexExpr{X: input, Index: intConst(0)})
		if !e.Inverted {
			cond = not(cond)
		}
		return []ast.Stmt{
			&ast.IfStmt{
				Cond: cond,
				Body: &ast.BlockStmt{List: onFailure()},
			},
			consumeInput(intConst(1)),
		}

	case *Sequence:
		var stmts []ast.Stmt
		for _, child := range e.Children {
			stmts = append(stmts, c.compileExpr(child.(ParsingExpression), onFailure)...)
		}
		return stmts

	case *Choice:
		if len(e.Children) == 1 {
			return c.compileExpr(e.Children[0].(ParsingExpression), onFailure)
		}

		choiceSuccessful := newDynamicLabel("choiceSuccessful")
		beforeChoice := newIdent("beforeChoice")
		stmts := []ast.Stmt{simpleDefine(beforeChoice, input)}
		for i, child := range e.Children {
			if i == len(e.Children)-1 {
				stmts = append(stmts, c.compileExpr(child.(ParsingExpression), onFailure)...)
				break
			}
			nextChoice := newDynamicLabel("nextChoice")
			stmts = append(stmts,
				&ast.BlockStmt{List: c.compileExpr(child.(ParsingExpression), nextChoice.GotoSlice)},
				choiceSuccessful.Goto(),
				nextChoice.WithLabel(nil),
				simpleAssign(input, beforeChoice),
			)
		}
		stmts = append(stmts, choiceSuccessful.WithLabel(nil))
		return stmts

	case *Repetition:
		repetitionLabel := newDynamicLabel("repetition")
		beforeRepetition := newIdent("beforeRepetition")
		var first *ast.Ident
		var forInit, forPost ast.Stmt
		if e.AtLeastOnce || e.GlueExpression != nil {
			first = newIdent("first")
			forInit = simpleDefine(first, ast.NewIdent("true"))
			forPost = simpleAssign(first, ast.NewIdent("false"))
		}
		breakLoop := func() []ast.Stmt {
			if e.AtLeastOnce {
				return []ast.Stmt{
					&ast.IfStmt{
						Cond: first,
						Body: &ast.BlockStmt{List: onFailure()},
					},
					simpleAssign(input, beforeRepetition),
					repetitionLabel.Break(),
				}
			}
			return []ast.Stmt{
				simpleAssign(input, beforeRepetition),
				repetitionLabel.Break(),
			}
		}
		var body []ast.Stmt
		if e.GlueExpression != nil {
			body = append(body, &ast.IfStmt{
				Cond: not(first),
				Body: &ast.BlockStmt{List: c.compileExpr(e.GlueExpression, breakLoop)},
			})
		}
		body = append(body, c.compileExpr(e.Child, breakLoop)...)
		if repetitionLabel.Used {
			body = append([]ast.Stmt{simpleDefine(beforeRepetition, input)}, body...)
		}
		return []ast.Stmt{
			repetitionLabel.WithLabel(&ast.ForStmt{
				Init: forInit,
				Post: forPost,
				Body: &ast.BlockStmt{List: body},
			}),
		}

	case *Until:
		untilLabel := newDynamicLabel("until")
		checkFailed := newDynamicLabel("checkFailed")
		beforeCheck := newIdent("beforeCheck")
		body := []ast.Stmt{simpleDefine(beforeCheck, input)}
		body = append(body, c.compileExpr(e.UntilExpression, checkFailed.GotoSlice)...)
		body = append(body, untilLabel.Break(), checkFailed.WithLabel(simpleAssign(input, beforeCheck)))
		body = append(body, c.compileExpr(e.Child, onFailure)...)
		return []ast.Stmt{
			untilLabel.WithLabel(&ast.ForStmt{
				Body: &ast.BlockStmt{List: body},
			}),
		}

	case *PositiveLookahead:
		beforeLookahead := newIdent("beforeLookahead")
		var stmts []ast.Stmt
		stmts = append(stmts, simpleDefine(beforeLookahead, input))
		stmts = append(stmts, c.compileExpr(e.Child, onFailure)...)
		stmts = append(stmts, simpleAssign(input, beforeLookahead))
		return stmts

	case *NegativeLookahead:
		lookaheadSuccessful := newDynamicLabel("lookaheadSuccessful")
		beforeLookahead := newIdent("beforeLookahead")
		var stmts []ast.Stmt
		stmts = append(stmts, simpleDefine(beforeLookahead, input))
		stmts = append(stmts, c.compileExpr(e.Child, lookaheadSuccessful.GotoSlice)...)
		stmts = append(stmts, onFailure()...)
		stmts = append(stmts, lookaheadSuccessful.WithLabel(simpleAssign(input, beforeLookahead)))
		return stmts

	case *RuleCall:
		return []ast.Stmt{
			simpleAssign(input, &ast.CallExpr{
				Fun:  ast.NewIdent(e.Name.String()),
				Args: []ast.Expr{input},
			}),
			&ast.IfStmt{
				Cond: &ast.BinaryExpr{X: input, Op: token.EQL, Y: ast.NewIdent("nil")},
				Body: &ast.BlockStmt{List: onFailure()},
			},
		}

	case *ParenthesizedExpression:
		return c.compileExpr(e.Child, onFailure)

	case *EmptyParsingExpression:
		return nil

	case *Label:
		stmts := c.compileExpr(e.Child, onFailure)
		nameIsAt := e.Name.String() == "@"
		childHasReturnValue := hasReturnValue(e.Child)

		if childHasReturnValue && nameIsAt {
			stmts = append(stmts, exprStmt(peglibCall("Pop")))
			childHasReturnValue = false
		}
		if !childHasReturnValue {
			labelStart := newIdent("labelStart")
			stmts = append([]ast.Stmt{simpleDefine(labelStart, input)}, stmts...)
			stmts = append(stmts, exprStmt(peglibCall("PushInputRange", labelStart, input)))
		}

		switch {
		case e.IsLocal:
			stmts = append(stmts, exprStmt(peglibCall("LocalsPush", intConst(1))))
		case nameIsAt:
			// don't add label
		default:
			stmts = append(stmts, exprStmt(peglibCall("MakeLabel", stringConst(e.Name.String()))))
		}

		return stmts

	default:
		panic("c.compileExpr not implemented for given type")
	}
}

func hasReturnValue(expr ParsingExpression) bool {
	switch e := expr.(type) {
	case *Sequence:
		for _, child := range e.Children {
			if hasReturnValue(child.(ParsingExpression)) {
				return true
			}
		}
		return false

	case *Choice:
		for _, child := range e.Children {
			if hasReturnValue(child.(ParsingExpression)) {
				return true
			}
		}
		return false

	case *ParenthesizedExpression:
		return hasReturnValue(e.Child)

	case *Label:
		return !e.IsLocal

	default:
		return false
	}
}

var input = ast.NewIdent("input")

func simpleAssign(lhs, rhs ast.Expr) ast.Stmt {
	return &ast.AssignStmt{Lhs: []ast.Expr{lhs}, Tok: token.ASSIGN, Rhs: []ast.Expr{rhs}}
}

func simpleDefine(lhs, rhs ast.Expr) ast.Stmt {
	return &ast.AssignStmt{Lhs: []ast.Expr{lhs}, Tok: token.DEFINE, Rhs: []ast.Expr{rhs}}
}

func consumeInput(length ast.Expr) ast.Stmt {
	return simpleAssign(input, &ast.SliceExpr{X: input, Low: length})
}

func intConst(i int) ast.Expr {
	return &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(i)}
}

func stringConst(s string) ast.Expr {
	return &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(s)}
}

func not(x ast.Expr) ast.Expr {
	return &ast.UnaryExpr{Op: token.NOT, X: x}
}

func exprStmt(x ast.Expr) ast.Stmt {
	return &ast.ExprStmt{X: x}
}

var nameCounters = make(map[string]int)

func newIdent(prefix string) *ast.Ident {
	nameCounters[prefix]++
	return &ast.Ident{Name: prefix + strconv.Itoa(nameCounters[prefix]), Obj: &ast.Object{}}
}

func peglibCall(fun string, args ...ast.Expr) ast.Expr {
	return &ast.CallExpr{
		Fun:  &ast.SelectorExpr{X: ast.NewIdent("peglib"), Sel: ast.NewIdent(fun)},
		Args: args,
	}
}

type dynamicLabel struct {
	Ident *ast.Ident
	Used  bool
}

func newDynamicLabel(prefix string) *dynamicLabel {
	return &dynamicLabel{Ident: newIdent(prefix)}
}

func (l *dynamicLabel) Goto() ast.Stmt {
	l.Used = true
	return &ast.BranchStmt{Tok: token.GOTO, Label: l.Ident}
}

func (l *dynamicLabel) GotoSlice() []ast.Stmt {
	return []ast.Stmt{l.Goto()}
}

func (l *dynamicLabel) Break() ast.Stmt {
	l.Used = true
	return &ast.BranchStmt{Tok: token.BREAK, Label: l.Ident}
}

func (l *dynamicLabel) WithLabel(stmt ast.Stmt) ast.Stmt {
	if stmt == nil {
		stmt = &ast.EmptyStmt{}
	}
	if !l.Used {
		return stmt
	}
	return &ast.LabeledStmt{Label: l.Ident, Stmt: stmt}
}

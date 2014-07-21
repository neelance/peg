package peg

import (
	"github.com/neelance/jetpeg"
	"go/ast"
)

type Sequence struct {
	Children []interface{}
}

func (e *Sequence) Compile(onFailure func() []ast.Stmt) (stmts []ast.Stmt) {
	for _, c := range e.Children {
		stmts = append(stmts, c.(ParsingExpression).Compile(onFailure)...)
	}
	return
}

type Choice struct {
	Children []interface{}
}

func (e *Choice) Compile(onFailure func() []ast.Stmt) (stmts []ast.Stmt) {
	if len(e.Children) == 1 {
		return e.Children[0].(ParsingExpression).Compile(onFailure)
	}

	choiceSuccessful := newDynamicLabel("choiceSuccessful")
	beforeChoice := newIdent("beforeChoice")
	stmts = append(stmts, simpleDefine(beforeChoice, input))
	for i, c := range e.Children {
		if i == len(e.Children)-1 {
			stmts = append(stmts, c.(ParsingExpression).Compile(onFailure)...)
			break
		}
		nextChoice := newDynamicLabel("nextChoice")
		stmts = append(stmts, c.(ParsingExpression).Compile(nextChoice.GotoSlice)...)
		stmts = append(stmts,
			choiceSuccessful.Goto(),
			nextChoice.WithLabel(nil),
			simpleAssign(input, beforeChoice),
		)
	}
	stmts = append(stmts, choiceSuccessful.WithLabel(nil))
	return
}

type Repetition struct {
	Child          ParsingExpression
	GlueExpression ParsingExpression
	AtLeastOnce    bool
}

func (e *Repetition) Compile(onFailure func() []ast.Stmt) []ast.Stmt {
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
			Body: &ast.BlockStmt{List: e.GlueExpression.Compile(breakLoop)},
		})
	}
	body = append(body, e.Child.Compile(breakLoop)...)
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
}

type Until struct {
	Child           ParsingExpression
	UntilExpression ParsingExpression
}

func (e *Until) Compile(onFailure func() []ast.Stmt) []ast.Stmt {
	untilLabel := newDynamicLabel("until")
	checkFailed := newDynamicLabel("checkFailed")
	beforeCheck := newIdent("beforeCheck")
	body := []ast.Stmt{simpleDefine(beforeCheck, input)}
	body = append(body, e.UntilExpression.Compile(checkFailed.GotoSlice)...)
	body = append(body, untilLabel.Break(), checkFailed.WithLabel(simpleAssign(input, beforeCheck)))
	body = append(body, e.Child.Compile(onFailure)...)
	return []ast.Stmt{
		untilLabel.WithLabel(&ast.ForStmt{
			Body: &ast.BlockStmt{List: body},
		}),
	}
}

type PositiveLookahead struct {
	Child ParsingExpression
}

func (e *PositiveLookahead) Compile(onFailure func() []ast.Stmt) (stmts []ast.Stmt) {
	beforeLookahead := newIdent("beforeLookahead")
	stmts = append(stmts, simpleDefine(beforeLookahead, input))
	stmts = append(stmts, e.Child.Compile(onFailure)...)
	stmts = append(stmts, simpleAssign(input, beforeLookahead))
	return
}

type NegativeLookahead struct {
	Child ParsingExpression
}

func (e *NegativeLookahead) Compile(onFailure func() []ast.Stmt) (stmts []ast.Stmt) {
	lookaheadSuccessful := newDynamicLabel("lookaheadSuccessful")
	beforeLookahead := newIdent("beforeLookahead")
	stmts = append(stmts, simpleDefine(beforeLookahead, input))
	stmts = append(stmts, e.Child.Compile(lookaheadSuccessful.GotoSlice)...)
	stmts = append(stmts, onFailure()...)
	stmts = append(stmts, lookaheadSuccessful.WithLabel(simpleAssign(input, beforeLookahead)))
	return
}

type RuleCall struct {
	Name      jetpeg.Stringer
	Arguments []interface{}
}

type ParenthesizedExpression struct {
	Child ParsingExpression
}

func (e *ParenthesizedExpression) Compile(onFailure func() []ast.Stmt) []ast.Stmt {
	return e.Child.Compile(onFailure)
}

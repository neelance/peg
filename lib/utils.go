package peg

import (
	"go/ast"
	"go/token"
	"strconv"
)

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

func not(x ast.Expr) ast.Expr {
	return &ast.UnaryExpr{Op: token.NOT, X: x}
}

var nameCounters = make(map[string]int)

func newIdent(prefix string) *ast.Ident {
	nameCounters[prefix]++
	return &ast.Ident{Name: prefix + strconv.Itoa(nameCounters[prefix]), Obj: &ast.Object{}}
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

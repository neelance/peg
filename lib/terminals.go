package peg

import (
	"github.com/neelance/jetpeg"
	"go/ast"
	"go/token"
	"strconv"
)

type StringTerminal struct {
	Chars jetpeg.Stringer
	Fold  bool
}

func (e *StringTerminal) Compile(onFailure func() []ast.Stmt) []ast.Stmt {
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
			Cond: not(&ast.CallExpr{
				Fun: ast.NewIdent(hasPrefixFun),
				Args: []ast.Expr{
					input,
					&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(string(str))},
				},
			}),
			Body: &ast.BlockStmt{List: onFailure()},
		},
		consumeInput(intConst(len(str))),
	}
}

type CharacterClassTerminal struct {
	Selections []interface{}
	Inverted   bool
}

type CharacterClassSingleCharacter struct {
	Char jetpeg.Stringer
}

type CharacterClassRange struct {
	BeginChar jetpeg.Stringer
	EndChar   jetpeg.Stringer
}

func (e *CharacterClassTerminal) Compile(onFailure func() []ast.Stmt) []ast.Stmt {
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
		c, _, _, err := strconv.UnquoteChar(str, 0)
		if err != nil {
			panic(err)
		}
		return c
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

	op := token.EQL
	if e.Inverted {
		op = token.NEQ
	}
	return []ast.Stmt{
		&ast.IfStmt{
			Cond: &ast.BinaryExpr{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{X: ast.NewIdent("strings"), Sel: ast.NewIdent("IndexByte")},
					Args: []ast.Expr{
						&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(string(selections))},
						&ast.IndexExpr{X: input, Index: intConst(0)},
					},
				},
				Op: op,
				Y:  intConst(-1),
			},
			Body: &ast.BlockStmt{List: onFailure()},
		},
		consumeInput(intConst(1)),
	}
}

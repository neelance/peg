package peglib

import (
	"github.com/neelance/jetpeg"
	"go/ast"
	"go/token"
	"reflect"
	"strings"
)

var metagrammarParser *jetpeg.Parser

func init() {
	typeMap := map[string]reflect.Type{}
	addType := func(i interface{}) {
		t := reflect.TypeOf(i)
		typeMap[t.Name()] = t
	}
	addType(Rule{})
	addType(EmptyParsingExpression{})
	addType(Sequence{})
	addType(Choice{})
	addType(Repetition{})
	addType(Until{})
	addType(PositiveLookahead{})
	addType(NegativeLookahead{})
	addType(RuleCall{})
	addType(ParenthesizedExpression{})
	addType(StringData{})
	addType(BooleanData{})
	addType(HashData{})
	addType(HashDataEntry{})
	addType(ArrayData{})
	addType(ArrayDataEntry{})
	addType(ObjectData{})
	addType(LabelData{})
	addType(TrueFunction{})
	addType(FalseFunction{})
	addType(MatchFunction{})
	addType(ErrorFunction{})
	addType(EnterModeFunction{})
	addType(LeaveModeFunction{})
	addType(InModeFunction{})
	addType(StringValue{})
	addType(Label{})
	addType(LocalValue{})
	addType(ObjectCreator{})
	addType(ValueCreator{})
	addType(StringTerminal{})
	addType(CharacterClassTerminal{})
	addType(CharacterClassSingleCharacter{})
	addType(CharacterClassRange{})

	jetpeg.Factory = func(name string, value interface{}) interface{} {
		inst := reflect.New(typeMap[name])
		for k, v := range value.(map[string]interface{}) {
			if v == nil {
				continue
			}
			k = strings.ToUpper(k[:1]) + k[1:]
			f := inst.Elem().FieldByName(k)
			if !f.IsValid() {
				panic("no such field: " + k + " of " + name)
			}
			f.Set(reflect.ValueOf(v))
		}
		return inst.Interface()
	}

	var err error
	metagrammarParser, err = jetpeg.Load("/Users/richard/gopath/src/github.com/neelance/peg/metagrammar.bc")
	if err != nil {
		panic(err)
	}
}

var byteSlice = &ast.ArrayType{Elt: ast.NewIdent("byte")}

func Compile(grammar string, mainRule string, fset *token.FileSet) *ast.File {
	g, err := metagrammarParser.Parse("Grammar", []byte(grammar))
	if err != nil {
		panic(err)
	}

	file := &ast.File{
		Name: ast.NewIdent("main"),
		Decls: []ast.Decl{
			&ast.GenDecl{
				Tok: token.IMPORT,
				Specs: []ast.Spec{
					&ast.ImportSpec{
						Path: &ast.BasicLit{Kind: token.STRING, Value: `"github.com/neelance/peg/runtime"`},
					},
				},
			},
			&ast.FuncDecl{
				Name: ast.NewIdent("main"),
				Type: &ast.FuncType{},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{&ast.ExprStmt{X: pegruntimeCall("Main", ast.NewIdent(mainRule))}},
				},
			},
		},
	}

	for _, rule := range g.(map[string]interface{})["Rules"].([]interface{}) {
		r := rule.(map[string]interface{})

		onFailure := func() []ast.Stmt {
			return []ast.Stmt{
				&ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("nil")}},
			}
		}
		file.Decls = append(file.Decls, &ast.FuncDecl{
			Name: ast.NewIdent(r["Name"].(jetpeg.Stringer).String()),
			Type: &ast.FuncType{
				Params:  &ast.FieldList{List: []*ast.Field{&ast.Field{Names: []*ast.Ident{input}, Type: byteSlice}}},
				Results: &ast.FieldList{List: []*ast.Field{&ast.Field{Type: byteSlice}}},
			},
			Body: &ast.BlockStmt{
				List: append(compileExpr(r["Child"].(*Rule).Child, onFailure),
					&ast.ReturnStmt{Results: []ast.Expr{input}},
				),
			},
		})
	}

	return file
}

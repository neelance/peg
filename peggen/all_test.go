package peggen

import (
	"go/ast"
	"go/printer"
	"go/token"
	"os"
	"os/exec"
	"testing"
)

func TestStringTerminal(t *testing.T) {
	testRule(t, `'abc'`, map[string]string{
		"abc":  "true",
		"ab":   "false",
		"Abc":  "false",
		"abC":  "false",
		"abcX": "false",
	})

	testRule(t, `"abc"`, map[string]string{
		"abc":  "true",
		"Abc":  "true",
		"abC":  "true",
		"ab":   "false",
		"Xbc":  "false",
		"abX":  "false",
		"abcX": "false",
	})
}

func TestCharacterClassTerminal(t *testing.T) {
	testRule(t, `[b-df\-h]`, map[string]string{
		"b": "true",
		"c": "true",
		"d": "true",
		"f": "true",
		"-": "true",
		"h": "true",
		"a": "false",
		"e": "false",
		"g": "false",
	})

	testRule(t, `[^a]`, map[string]string{
		"b": "true",
		"a": "false",
	})

	testRule(t, `[\n]`, map[string]string{
		"\n": "true",
		"n":  "false",
	})
}

func TestAnyCharacterTerminal(t *testing.T) {
	testRule(t, `.`, map[string]string{
		"a":  "true",
		"B":  "true",
		"5":  "true",
		"":   "false",
		"99": "false",
	})

	testRule(t, `.*`, map[string]string{
		"aaa": "true",
	})
}

func TestSequence(t *testing.T) {
	testRule(t, `'abc' 'def'`, map[string]string{
		"abcdef":  "true",
		"abcde":   "false",
		"aXcdef":  "false",
		"abcdXf":  "false",
		"abcdefX": "false",
	})
}

func TestChoice(t *testing.T) {
	testRule(t, `/ 'abc' / 'def'`, map[string]string{
		"abc":  "true",
		"def":  "true",
		"ab":   "false",
		"aXc":  "false",
		"defX": "false",
	})
}

func TestOptional(t *testing.T) {
	testRule(t, `'abc'? 'def'`, map[string]string{
		"abcdef": "true",
		"def":    "true",
		"abc":    "false",
		"aXcdef": "false",
		"abdef":  "false",
	})
}

func TestZeroOrMore(t *testing.T) {
	testRule(t, `'a'*`, map[string]string{
		"":      "true",
		"a":     "true",
		"aaaaa": "true",
		"X":     "false",
		"aaaX":  "false",
	})
}

func TestOneOrMore(t *testing.T) {
	testRule(t, `'a'+`, map[string]string{
		"a":     "true",
		"aaaaa": "true",
		"":      "false",
		"X":     "false",
		"aaaX":  "false",
	})
}

func TestRepetitionGlue(t *testing.T) {
	testRule(t, `'a'*[ ',' ]`, map[string]string{
		"":      "true",
		"a":     "true",
		"a,a,a": "true",
		"aa":    "false",
		",":     "false",
		"a,a,":  "false",
		",a,a":  "false",
		"a,,a":  "false",
	})

	testRule(t, `'a'+[ ',' ]`, map[string]string{
		"a":     "true",
		"a,a,a": "true",
		"aa":    "false",
		"":      "false",
		",":     "false",
		"a,a,":  "false",
		",a,a":  "false",
		"a,,a":  "false",
	})
}

func TestUntil(t *testing.T) {
	testRule(t, `( 'a' . )*->'ac'`, map[string]string{
		"ac":       "true",
		"ababac":   "true",
		"":         "false",
		"ab":       "false",
		"abXbac":   "false",
		"ababacX":  "false",
		"ababacab": "false",
		"ababacac": "false",
	})
}

func TestParenthesizedExpression(t *testing.T) {
	testRule(t, `( 'a' ( ) 'b' )? 'c'`, map[string]string{
		"abc": "true",
		"c":   "true",
		"ac":  "false",
		"bc":  "false",
	})
}

func TestPositiveLookahead(t *testing.T) {
	testRule(t, `&'a' .`, map[string]string{
		"a":  "true",
		"":   "false",
		"X":  "false",
		"aX": "false",
	})
}

func TestNegativeLookahead(t *testing.T) {
	testRule(t, `!'a' .`, map[string]string{
		"X":  "true",
		"":   "false",
		"a":  "false",
		"XX": "false",
	})
}

func TestRuleDefinition(t *testing.T) {
	testGrammar(t, `
		rule SomeName
			'a'
		end
	`, "SomeName", map[string]string{
		"a": "true",
		"X": "false",
	})
}

func TestRuleReference(t *testing.T) {
	testGrammar(t, `
    rule Test
      a
    end
    rule a
      'b'
    end
	`, "Test", map[string]string{
		"b": "true",
		"X": "false",
		"a": "false",
	})
}

func TestRecursiveRule(t *testing.T) {
	testGrammar(t, `
    rule Test
      '(' Test ')' / ( )
    end
	`, "Test", map[string]string{
		"":       "true",
		"()":     "true",
		"((()))": "true",
		"()))":   "false",
		"((()":   "false",
	})
}

func TestLabel(t *testing.T) {
	testRule(t, `'a' char:. 'c' / 'def'`, map[string]string{
		"abc": `{"char":"b"}`,
	})

	testRule(t, `word:( 'a' 'b' 'c' )`, map[string]string{
		"abc": `{"word":"abc"}`,
	})

	testRule(t, `( word:[abc]+ )?`, map[string]string{
		"abc": `{"word":"abc"}`,
		"":    "true",
	})

	testRule(t, `'a' outer:( inner:. ) 'c' / 'def'`, map[string]string{
		"abc": `{"outer":{"inner":"b"}}`,
	})
}

// func TestNestedLabel(t *testing.T) {
// 	testRule(t, `word:( 'a' char:. 'c' )`, map[string]string {
// 	"abc") == { word: { char: "b" } : "true",
// }

// func TestAtLabel(t *testing.T) {
// 	testRule(t, `'a' @:. 'c'`, map[string]string {
// 	"abc") == "b: "true",

// 	testGrammar(t, `
// 	//       rule test
// 	//         char:a
// 	//       end
// 	//       rule a
// 	//         'a' @:a 'c' / @:'b'
// 	//       end
// 	//     "
// 	//     assert grammar.parse_rule(:test, "abc") == { char: "b" }
// }

// func TestLabelMerge(t *testing.T) {
// 	testRule(t, `( char:'a' x:'x' / 'b' x:'x' / char:( inner:'c' ) x:'x' ) / 'y'`, map[string]string {
// 	"ax") == { char: "a", x: "x" : "true",
// 	"bx") == { x: "x" : "true",
// 	"cx") == { char: { inner: "c" }, x: "x" : "true",
// }

// func TestRuleWithLabel(t *testing.T) {
// 	testGrammar(t, `
// 	//       rule test
// 	//         a word:( 'b' a ) :a
// 	//       end
// 	//       rule a
// 	//         d:'d' / char:.
// 	//       end
// 	//     "
// 	//     assert grammar.parse_rule(:test, "abcd") == { char: "a", word: { char: "c" }, a: { d: "d" } }
// }

// func TestRecursiveRuleWithLabel(t *testing.T) {
// 	testGrammar(t, `
// 	//       rule test
// 	//         '(' inner:( test ( other:'b' )? ) ')' / char:'a'
// 	//       end
// 	//     "
// 	//     assert grammar.parse_rule(:test, "((a)b)") == { inner: { inner: { char: "a" }, other: "b"} }

// 	testGrammar(t, `
// 	//       rule test
// 	//         '(' test ')' / char:'a'
// 	//       end
// 	//     "
// 	//     assert grammar.parse_rule(:test, "((a))") == { char: "a" }

// 	testGrammar(t, `
// 	//       rule test
// 	//         '(' test2 ')' / char:'a'
// 	//       end
// 	//       rule test2
// 	//         a:test b:test
// 	//       end
// 	//     "
// 	//     assert grammar.parse_rule(:test, "((aa)(aa))") == { a: { a: { char: "a" }, b: { char: "a" }}, b: { a: { char: "a" }, b: { char: "a" } } }
// }

// func TestRepetitionWithLabel(t *testing.T) {
// 	testRule(t, `list:( char:( 'a' / 'b' / 'c' ) )*`, map[string]string {
// 	"abc") == { list: [{ char: "a" }, { char: "b" }, { char: "c" }] : "true",

// 	testRule(t, `list:( char:'a' / char:'b' / 'c' )+`, map[string]string {
// 	"abc") == { list: [{ char: "a" }, { char: "b" }, {}] : "true",

// 	testRule(t, `( 'a' / 'b' / 'c' )+`, map[string]string {
// 	"abc") == {: "true",

// 	testRule(t, `list:( 'a' char:. )*->( 'ada' final:. )`, map[string]string {
// 	"abacadae") == { list: [{ char: "b" }, { char: "c" }, { final: "e" }] : "true",

// 	testGrammar(t, `
// 	//       rule test
// 	//         ( char1:'a' inner:test / 'b' )*
// 	//       end
// 	//     "
// 	//     assert grammar.parse_rule(:test, "ab")
// }

// func TestObjectCreator(t *testing.T) {
// 	testRule(t, `'a' char:. 'c' <TestClassA> / 'd' char:. 'f' <TestClassB>", class_scope: self.clas`, map[string]string {
// 	"abc") == TestClassA.new({ char: "b" }: "true",
// 	"def") == TestClassB.new({ char: "e" }: "true",

// 	testRule(t, `'a' char:. 'c' <TestClassA { a: 'test1', b: [ <TestClassB "true">, <TestClassB { r: @char }> ] }>", class_scope: self.clas`, map[string]string {
// 	"abc") == TestClassA.new({ a: "test1", b: [ TestClassB.new("true"), TestClassB.new({ r: "b" }) ] }: "true",
// }

// func TestValueCreator(t *testing.T) {
// 	testRule(t, `, map[string]string {
// 	//       'a' char:. 'c' { @char.upcase } /
// 	//       word:'def' { @word.chars.map { |c| c.ord } } /
// 	//       'ghi' { [__FILE__, __LINE__] }
// 	//     ", filename: "test.jetpeg"
// 	"abc") == "B: "true",
// 	"def") == ["d".ord, "e".ord, "f".ord: "true",
// 	"ghi") == ["test.jetpeg", 4: "true",
// }

// func TestLocalLabel(t *testing.T) {
// 	testRule(t, `'a' %temp:( char:'b' )* 'c' ( result:%temp )`, map[string]string {
// 	"abc") == { result: [{ char: "b" }] : "true",
// 	"abX") == ni: "true",

// 	testRule(t, `'a' %temp:( char:'b' )* 'c' result1:%temp result2:%temp`, map[string]string {
// 	"abc") == { result1: [{ char: "b" }], result2: [{ char: "b" }] : "true",
// }

// func TestParameters(t *testing.T) {
// 	testGrammar(t, `
// 	//       rule test
// 	//         %a:. %b:. test2[%a, %b, $"true"]
// 	//       end
// 	//       rule test2[%v, %w, %x]
// 	//         result1:%v result2:%w result3:%x
// 	//       end
// 	//     "
// 	//     assert grammar.parse_rule(:test, "ab") == { result1: "a", result2: "b", result3: "true" }
// }

// func TestUndefinedLocalLabelError(t *testing.T) {
// 	//     assert_raise JetPEG::CompilationError do
// 	testRule(t, `char:%missing`, map[string]string {
// 	//       rule.parse "abc"
// 	//     end
// }

// func TestLeftRecursionHandling(t *testing.T) {
// 	testGrammar(t, `
// 	//       rule expr
// 	//         add:( l:expr '+' r:num ) /
// 	//         sub:( l:expr '-' r:num ) /
// 	//         expr /
// 	//         @:num
// 	//       end

// 	//       rule num
// 	//         [0-9]+
// 	//       end
// 	//     "
// 	//     assert grammar.parse_rule(:expr, "1-2-3") == { sub: { l: { sub: { l: "1", r: "2" } }, r: "3" } }
// }

// func TestBooleanFunctions(t *testing.T) {
// 	testRule(t, `'a' v:$"true" 'bc' / 'd' v:$"false 'ef'`, map[string]string "{
// 	"abc") == { v: "true" : "true",
// 	"def") == { v: "false : "true"",
//   })

// 	testRule(t, `'a' ( 'b' v:$"true" )? 'c'`, map[string]string {
// 	"abc") == { v: "true" : "true",
// 	"ac") == {: "true",
//   })
// }

// func TestErrorFunction(t *testing.T) {
// 	testRule(t, `'a' $error['test'] 'bc'`, map[string]string {
// 	"abc": "false",
// 	//     assert rule.parser.failure_reason.is_a? JetPEG::ParsingError
// 	//     assert rule.parser.failure_reason.position == 1
// 	//     assert rule.parser.failure_reason.other_reasons == ["test"]
// }

// func TestMatchFunction(t *testing.T) {
// 	testRule(t, `%a:( . . ) $match[%a]`, map[string]string {
// 	"abab": "true",
// 	"cdcd": "true",
// 	"a": "false",
// 	"ab": "false",
// 	"aba": "false",
// 	"abaX": "false",
// }

// func TestModes(t *testing.T) {
// 	testGrammar(t, `
// 	//       rule test
// 	//         test2 $enter_mode['somemode', test2 $enter_mode['othermode', $leave_mode['somemode', test2]]]
// 	//       end
// 	//       rule test2
// 	//         !$in_mode['somemode'] 'a' / $in_mode['somemode'] 'b'
// 	//       end
// 	//     "
// 	//     assert grammar.parse_rule(:test, "aba")
// 	//     assert !grammar.parse_rule(:test, "aaa")
// 	//     assert !grammar.parse_rule(:test, "bba")
// 	//     assert !grammar.parse_rule(:test, "abb")
// }

func testRule(t *testing.T, rule string, inputs map[string]string) {
	testGrammar(t, "rule Test\n"+rule+"\nend\n", "Test", inputs)
}

func testGrammar(t *testing.T, grammar, mainRule string, inputs map[string]string) {
	file := &ast.File{
		Name: ast.NewIdent("main"),
		Decls: append(
			[]ast.Decl{
				&ast.GenDecl{
					Tok: token.IMPORT,
					Specs: []ast.Spec{
						&ast.ImportSpec{
							Path: &ast.BasicLit{Kind: token.STRING, Value: `"github.com/neelance/peg/peglib"`},
						},
					},
				},
				&ast.FuncDecl{
					Name: ast.NewIdent("main"),
					Type: &ast.FuncType{},
					Body: &ast.BlockStmt{
						List: []ast.Stmt{&ast.ExprStmt{X: peglibCall("Test", ast.NewIdent(mainRule))}},
					},
				},
			},
			Compile(grammar)...,
		),
	}

	os.Mkdir("tmp", 0777)
	testfile, err := os.Create("tmp/test.go")
	if err != nil {
		t.Fatal(err)
	}

	printer.Fprint(testfile, token.NewFileSet(), file)
	testfile.Close()

	for input, expected := range inputs {
		output, err := exec.Command("go", "run", testfile.Name(), input).CombinedOutput()
		if err != nil {
			t.Log(string(output))
			t.Fatal(err)
		}

		expected += "\n"
		got := string(output)
		if expected != got {
			t.Errorf("grammar %q gave wrong result on %q: expected %q, got %q", grammar, input, expected, got)
		}
	}

	if !t.Failed() {
		os.Remove(testfile.Name())
		os.Remove("tmt")
	}
}

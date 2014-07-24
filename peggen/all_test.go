package peggen

import (
	"encoding/json"
	"go/ast"
	"go/printer"
	"go/token"
	"os"
	"os/exec"
	"reflect"
	"testing"
)

func TestStringTerminal(t *testing.T) {
	testRule(t, `'abc'`, map[string]string{
		"abc":  "{}",
		"ab":   "null",
		"Abc":  "null",
		"abC":  "null",
		"abcX": "null",
	})

	testRule(t, `"abc"`, map[string]string{
		"abc":  "{}",
		"Abc":  "{}",
		"abC":  "{}",
		"ab":   "null",
		"Xbc":  "null",
		"abX":  "null",
		"abcX": "null",
	})
}

func TestCharacterClassTerminal(t *testing.T) {
	testRule(t, `[b-df\-h]`, map[string]string{
		"b": "{}",
		"c": "{}",
		"d": "{}",
		"f": "{}",
		"-": "{}",
		"h": "{}",
		"a": "null",
		"e": "null",
		"g": "null",
	})

	testRule(t, `[^a]`, map[string]string{
		"b": "{}",
		"a": "null",
	})

	testRule(t, `[\n]`, map[string]string{
		"\n": "{}",
		"n":  "null",
	})
}

func TestAnyCharacterTerminal(t *testing.T) {
	testRule(t, `.`, map[string]string{
		"a":  "{}",
		"B":  "{}",
		"5":  "{}",
		"":   "null",
		"99": "null",
	})

	testRule(t, `.*`, map[string]string{
		"aaa": "{}",
	})
}

func TestSequence(t *testing.T) {
	testRule(t, `'abc' 'def'`, map[string]string{
		"abcdef":  "{}",
		"abcde":   "null",
		"aXcdef":  "null",
		"abcdXf":  "null",
		"abcdefX": "null",
	})
}

func TestChoice(t *testing.T) {
	testRule(t, `/ 'abc' / 'def'`, map[string]string{
		"abc":  "{}",
		"def":  "{}",
		"ab":   "null",
		"aXc":  "null",
		"defX": "null",
	})
}

func TestOptional(t *testing.T) {
	testRule(t, `'abc'? 'def'`, map[string]string{
		"abcdef": "{}",
		"def":    "{}",
		"abc":    "null",
		"aXcdef": "null",
		"abdef":  "null",
	})
}

func TestZeroOrMore(t *testing.T) {
	testRule(t, `'a'*`, map[string]string{
		"":      "{}",
		"a":     "{}",
		"aaaaa": "{}",
		"X":     "null",
		"aaaX":  "null",
	})
}

func TestOneOrMore(t *testing.T) {
	testRule(t, `'a'+`, map[string]string{
		"a":     "{}",
		"aaaaa": "{}",
		"":      "null",
		"X":     "null",
		"aaaX":  "null",
	})
}

func TestRepetitionGlue(t *testing.T) {
	testRule(t, `'a'*[ ',' ]`, map[string]string{
		"":      "{}",
		"a":     "{}",
		"a,a,a": "{}",
		"aa":    "null",
		",":     "null",
		"a,a,":  "null",
		",a,a":  "null",
		"a,,a":  "null",
	})

	testRule(t, `'a'+[ ',' ]`, map[string]string{
		"a":     "{}",
		"a,a,a": "{}",
		"aa":    "null",
		"":      "null",
		",":     "null",
		"a,a,":  "null",
		",a,a":  "null",
		"a,,a":  "null",
	})
}

func TestUntil(t *testing.T) {
	testRule(t, `( 'a' . )*->'ac'`, map[string]string{
		"ac":       "{}",
		"ababac":   "{}",
		"":         "null",
		"ab":       "null",
		"abXbac":   "null",
		"ababacX":  "null",
		"ababacab": "null",
		"ababacac": "null",
	})
}

func TestParenthesizedExpression(t *testing.T) {
	testRule(t, `( 'a' ( ) 'b' )? 'c'`, map[string]string{
		"abc": "{}",
		"c":   "{}",
		"ac":  "null",
		"bc":  "null",
	})
}

func TestPositiveLookahead(t *testing.T) {
	testRule(t, `&'a' .`, map[string]string{
		"a":  "{}",
		"":   "null",
		"X":  "null",
		"aX": "null",
	})
}

func TestNegativeLookahead(t *testing.T) {
	testRule(t, `!'a' .`, map[string]string{
		"X":  "{}",
		"":   "null",
		"a":  "null",
		"XX": "null",
	})
}

func TestRuleDefinition(t *testing.T) {
	testGrammar(t, `
		rule SomeName
			'a'
		end
	`, "SomeName", map[string]string{
		"a": "{}",
		"X": "null",
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
		"b": "{}",
		"X": "null",
		"a": "null",
	})
}

func TestRecursiveRule(t *testing.T) {
	testGrammar(t, `
		rule Test
			'(' Test ')' / ( )
		end
	`, "Test", map[string]string{
		"":       "{}",
		"()":     "{}",
		"((()))": "{}",
		"()))":   "null",
		"((()":   "null",
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
		"":    "{}",
	})
}

func TestNestedLabel(t *testing.T) {
	testRule(t, `word:( 'a' char:. 'c' )`, map[string]string{
		"abc": `{"word":{"char":"b"}}`,
	})

	testRule(t, `'a' outer:( inner:. ) 'c' / 'def'`, map[string]string{
		"abc": `{"outer":{"inner":"b"}}`,
	})
}

func TestAtLabel(t *testing.T) {
	testRule(t, `'a' @:. 'c'`, map[string]string{
		"abc": `"b"`,
	})

	testGrammar(t, `
		rule Test
			char:a
		end
		rule a
			'a' @:a 'c' / @:'b'
		end
	`, "Test", map[string]string{
		"abc": `{"char":"b"}`,
	})
}

func TestLabelMerge(t *testing.T) {
	testRule(t, `( char:'a' x:'x' / 'b' x:'x' / char:( inner:'c' ) x:'x' ) / 'y'`, map[string]string{
		"ax": `{"char":"a","x":"x"}`,
		"bx": `{"x":"x"}`,
		"cx": `{"char":{"inner":"c"},"x":"x"}`,
	})
}

func TestRuleWithLabel(t *testing.T) {
	testGrammar(t, `
		rule Test
			a word:( 'b' a ) :a
		end
		rule a
			d:'d' / char:.
		end
	`, "Test", map[string]string{
		"abcd": `{"char":"a","word":{"char":"c"},"a":{"d":"d"}}`,
	})
}

func TestRecursiveRuleWithLabel(t *testing.T) {
	testGrammar(t, `
		rule Test
			'(' inner:( Test ( other:'b' )? ) ')' / char:'a'
		end
	`, "Test", map[string]string{
		"((a)b)": `{"inner":{"inner":{"char":"a"},"other":"b"}}`,
	})

	testGrammar(t, `
		rule Test
			'(' Test ')' / char:'a'
		end
	`, "Test", map[string]string{
		"((a))": `{"char":"a"}`,
	})

	testGrammar(t, `
		rule Test
			'(' test2 ')' / char:'a'
		end
		rule test2
			a:Test b:Test
		end
	`, "Test", map[string]string{
		"((aa)(aa))": `{"a":{"a":{"char":"a"},"b":{"char":"a"}},"b":{"a":{"char":"a"},"b":{"char":"a"}}}`,
	})
}

func TestRepetitionWithLabel(t *testing.T) {
	testRule(t, `list:( char:( 'a' / 'b' / 'c' ) )*`, map[string]string{
		"abc": `{"list":[{"char":"a"},{"char":"b"},{"char":"c"}]}`,
	})

	testRule(t, `list:( char:'a' / char:'b' / 'c' )+`, map[string]string{
		"abc": `{"list":[{"char":"a"},{"char":"b"},{}]}`,
	})

	testRule(t, `list:( 'a' char:. )*->( 'ada' final:. )`, map[string]string{
		"abacadae": `{"list":[{"char":"b"},{"char":"c"},{"final":"e"}]}`,
	})

	testGrammar(t, `
		rule Test
			( char:'a' inner:Test / 'b' )*
		end
	`, "Test", map[string]string{
		"ab": `[{"char":"a","inner":[{}]}]`,
	})
}

// func TestObjectCreator(t *testing.T) {
//  testRule(t, `'a' char:. 'c' <TestClassA> / 'd' char:. 'f' <TestClassB>", class_scope: self.clas`, map[string]string {
//  "abc") == TestClassA.new({ char: "b" }: "{}",
//  "def") == TestClassB.new({ char: "e" }: "{}",

//  testRule(t, `'a' char:. 'c' <TestClassA { a: 'test1', b: [ <TestClassB "{}">, <TestClassB { r: @char }> ] }>", class_scope: self.clas`, map[string]string {
//  "abc") == TestClassA.new({ a: "test1", b: [ TestClassB.new("{}"), TestClassB.new({ r: "b" }) ] }: "{}",
// }

// func TestValueCreator(t *testing.T) {
//  testRule(t, `, map[string]string {
//  //       'a' char:. 'c' { @char.upcase } /
//  //       word:'def' { @word.chars.map { |c| c.ord } } /
//  //       'ghi' { [__FILE__, __LINE__] }
//  //     ", filename: "test.jetpeg"
//  "abc") == "B: "{}",
//  "def") == ["d".ord, "e".ord, "f".ord: "{}",
//  "ghi") == ["test.jetpeg", 4: "{}",
// }

// func TestLocalLabel(t *testing.T) {
//  testRule(t, `'a' %temp:( char:'b' )* 'c' ( result:%temp )`, map[string]string {
//  "abc") == { result: [{ char: "b" }] : "{}",
//  "abX") == ni: "{}",

//  testRule(t, `'a' %temp:( char:'b' )* 'c' result1:%temp result2:%temp`, map[string]string {
//  "abc") == { result1: [{ char: "b" }], result2: [{ char: "b" }] : "{}",
// }

// func TestParameters(t *testing.T) {
//  testGrammar(t, `
//  //       rule test
//  //         %a:. %b:. test2[%a, %b, $"{}"]
//  //       end
//  //       rule test2[%v, %w, %x]
//  //         result1:%v result2:%w result3:%x
//  //       end
//  //     "
//  //     assert grammar.parse_rule(:test, "ab") == { result1: "a", result2: "b", result3: "{}" }
// }

// func TestUndefinedLocalLabelError(t *testing.T) {
//  //     assert_raise JetPEG::CompilationError do
//  testRule(t, `char:%missing`, map[string]string {
//  //       rule.parse "abc"
//  //     end
// }

// func TestLeftRecursionHandling(t *testing.T) {
//  testGrammar(t, `
//  //       rule expr
//  //         add:( l:expr '+' r:num ) /
//  //         sub:( l:expr '-' r:num ) /
//  //         expr /
//  //         @:num
//  //       end

//  //       rule num
//  //         [0-9]+
//  //       end
//  //     "
//  //     assert grammar.parse_rule(:expr, "1-2-3") == { sub: { l: { sub: { l: "1", r: "2" } }, r: "3" } }
// }

// func TestBooleanFunctions(t *testing.T) {
//  testRule(t, `'a' v:$"{}" 'bc' / 'd' v:$"false 'ef'`, map[string]string "{
//  "abc") == { v: "{}" : "{}",
//  "def") == { v: "false : "{}"",
//   })

//  testRule(t, `'a' ( 'b' v:$"{}" )? 'c'`, map[string]string {
//  "abc") == { v: "{}" : "{}",
//  "ac") == {: "{}",
//   })
// }

// func TestErrorFunction(t *testing.T) {
//  testRule(t, `'a' $error['test'] 'bc'`, map[string]string {
//  "abc": "null",
//  //     assert rule.parser.failure_reason.is_a? JetPEG::ParsingError
//  //     assert rule.parser.failure_reason.position == 1
//  //     assert rule.parser.failure_reason.other_reasons == ["test"]
// }

// func TestMatchFunction(t *testing.T) {
//  testRule(t, `%a:( . . ) $match[%a]`, map[string]string {
//  "abab": "{}",
//  "cdcd": "{}",
//  "a": "null",
//  "ab": "null",
//  "aba": "null",
//  "abaX": "null",
// }

// func TestModes(t *testing.T) {
//  testGrammar(t, `
//  //       rule test
//  //         test2 $enter_mode['somemode', test2 $enter_mode['othermode', $leave_mode['somemode', test2]]]
//  //       end
//  //       rule test2
//  //         !$in_mode['somemode'] 'a' / $in_mode['somemode'] 'b'
//  //       end
//  //     "
//  //     assert grammar.parse_rule(:test, "aba")
//  //     assert !grammar.parse_rule(:test, "aaa")
//  //     assert !grammar.parse_rule(:test, "bba")
//  //     assert !grammar.parse_rule(:test, "abb")
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

	for input, expectedJson := range inputs {
		output, err := exec.Command("go", "run", testfile.Name(), input).CombinedOutput()
		if err != nil {
			t.Log(string(output))
			t.Fatal(err)
		}

		var expected, got interface{}
		if err := json.Unmarshal([]byte(expectedJson), &expected); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(output, &got); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(expected, got) {
			t.Errorf("grammar %q gave wrong result on %q:\nexpected %#v\ngot      %#v", grammar, input, expected, got)
		}
	}

	if !t.Failed() {
		os.Remove(testfile.Name())
		os.Remove("tmt")
	}
}

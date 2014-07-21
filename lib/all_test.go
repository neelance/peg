package peg

import (
	"code.google.com/p/go.tools/go/loader"
	"code.google.com/p/go.tools/go/ssa"
	"code.google.com/p/go.tools/go/ssa/interp"
	"code.google.com/p/go.tools/go/types"
	"go/build"
	"go/printer"
	"go/token"
	"os"
	"testing"
)

func TestStringTerminal(t *testing.T) {
	testRule(t, `'abc'`, map[string]bool{
		"abc":  true,
		"ab":   false,
		"Abc":  false,
		"abC":  false,
		"abcX": false,
	})

	testRule(t, `"abc"`, map[string]bool{
		"abc":  true,
		"Abc":  true,
		"abC":  true,
		"ab":   false,
		"Xbc":  false,
		"abX":  false,
		"abcX": false,
	})
}

func TestCharacterClassTerminal(t *testing.T) {
	testRule(t, `[b-df\-h]`, map[string]bool{
		"b": true,
		"c": true,
		"d": true,
		"f": true,
		"-": true,
		"h": true,
		"a": false,
		"e": false,
		"g": false,
	})

	testRule(t, `[^a]`, map[string]bool{
		"b": true,
		"a": false,
	})

	testRule(t, `[\n]`, map[string]bool{
		"\n": true,
		"n":  false,
	})
}

func TestAnyCharacterTerminal(t *testing.T) {
	testRule(t, `.`, map[string]bool{
		"a":  true,
		"B":  true,
		"5":  true,
		"":   false,
		"99": false,
	})

	testRule(t, `.*`, map[string]bool{
		"aaa": true,
	})
}

func TestSequence(t *testing.T) {
	testRule(t, `'abc' 'def'`, map[string]bool{
		"abcdef":  true,
		"abcde":   false,
		"aXcdef":  false,
		"abcdXf":  false,
		"abcdefX": false,
	})
}

func TestChoice(t *testing.T) {
	testRule(t, `/ 'abc' / 'def'`, map[string]bool{
		"abc":  true,
		"def":  true,
		"ab":   false,
		"aXc":  false,
		"defX": false,
	})
}

func TestOptional(t *testing.T) {
	testRule(t, `'abc'? 'def'`, map[string]bool{
		"abcdef": true,
		"def":    true,
		"abc":    false,
		"aXcdef": false,
		"abdef":  false,
	})
}

func TestZeroOrMore(t *testing.T) {
	testRule(t, `'a'*`, map[string]bool{
		"":      true,
		"a":     true,
		"aaaaa": true,
		"X":     false,
		"aaaX":  false,
	})
}

func TestOneOrMore(t *testing.T) {
	testRule(t, `'a'+`, map[string]bool{
		"a":     true,
		"aaaaa": true,
		"":      false,
		"X":     false,
		"aaaX":  false,
	})
}

func TestRepetitionGlue(t *testing.T) {
	testRule(t, `'a'*[ ',' ]`, map[string]bool{
		"":      true,
		"a":     true,
		"a,a,a": true,
		"aa":    false,
		",":     false,
		"a,a,":  false,
		",a,a":  false,
		"a,,a":  false,
	})

	testRule(t, `'a'+[ ',' ]`, map[string]bool{
		"a":     true,
		"a,a,a": true,
		"aa":    false,
		"":      false,
		",":     false,
		"a,a,":  false,
		",a,a":  false,
		"a,,a":  false,
	})
}

func TestUntil(t *testing.T) {
	testRule(t, `( 'a' . )*->'ac'`, map[string]bool{
		"ac":       true,
		"ababac":   true,
		"":         false,
		"ab":       false,
		"abXbac":   false,
		"ababacX":  false,
		"ababacab": false,
		"ababacac": false,
	})
}

func TestParenthesizedExpression(t *testing.T) {
	testRule(t, `( 'a' ( ) 'b' )? 'c'`, map[string]bool{
		"abc": true,
		"c":   true,
		"ac":  false,
		"bc":  false,
	})
}

func TestPositiveLookahead(t *testing.T) {
	testRule(t, `&'a' .`, map[string]bool{
		"a":  true,
		"":   false,
		"X":  false,
		"aX": false,
	})
}

func TestNegativeLookahead(t *testing.T) {
	testRule(t, `!'a' .`, map[string]bool{
		"X":  true,
		"":   false,
		"a":  false,
		"XX": false,
	})
}

// func TestRuleDefinition(t *testing.T) {
// 	//     grammar = JetPEG::Compiler.compile_grammar "
// 	//       rule test
// 	//         'a'
// 	//       end
// 	//     "
// 	//     assert grammar.parse_rule(:test, "a")
// 	//     assert !grammar.parse_rule(:test, "X")
// }

// func TestRuleReference(t *testing.T) {
// 	//     grammar = JetPEG::Compiler.compile_grammar "
// 	//       rule test
// 	//         a
// 	//       end
// 	//       rule a
// 	//         'b'
// 	//       end
// 	//     "
// 	//     assert grammar.parse_rule(:test, "b")
// 	//     assert !grammar.parse_rule(:test, "X")
// 	//     assert !grammar.parse_rule(:test, "a")
// }

// func TestRecursiveRule(t *testing.T) {
// 	//     grammar = JetPEG::Compiler.compile_grammar "
// 	//       rule test
// 	//         '(' test ')' / ( )
// 	//       end
// 	//     "
// 	//     assert grammar.parse_rule(:test, "")
// 	//     assert grammar.parse_rule(:test, "()")
// 	//     assert grammar.parse_rule(:test, "((()))")
// 	//     assert !grammar.parse_rule(:test, "()))")
// 	//     assert !grammar.parse_rule(:test, "((()")
// }

// func TestBooleanFunctions(t *testing.T) {
// 	testRule(t, `'a' v:$true 'bc' / 'd' v:$false 'ef'`, map[string]bool {
// 	"abc") == { v: true : true,
// 	"def") == { v: false : true,
//   })

// 	testRule(t, `'a' ( 'b' v:$true )? 'c'`, map[string]bool {
// 	"abc") == { v: true : true,
// 	"ac") == {: true,
//   })
// }

// func TestErrorFunction(t *testing.T) {
// 	testRule(t, `'a' $error['test'] 'bc'`, map[string]bool {
// 	"abc": false,
// 	//     assert rule.parser.failure_reason.is_a? JetPEG::ParsingError
// 	//     assert rule.parser.failure_reason.position == 1
// 	//     assert rule.parser.failure_reason.other_reasons == ["test"]
// }

// func TestMatchFunction(t *testing.T) {
// 	testRule(t, `%a:( . . ) $match[%a]`, map[string]bool {
// 	"abab": true,
// 	"cdcd": true,
// 	"a": false,
// 	"ab": false,
// 	"aba": false,
// 	"abaX": false,
// }

// func TestModes(t *testing.T) {
// 	//     grammar = JetPEG::Compiler.compile_grammar "
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

// func TestLabel(t *testing.T) {
// 	testRule(t, `'a' char:. 'c' / 'def'`, map[string]bool {
// 	//     result = rule.parse "abc"
// 	//     assert result == { char: "b" }
// 	//     assert result[:char] == "b"
// 	//     assert result[:char] === "b"
// 	//     assert "b" == result[:char]
// 	//     assert "b" === result[:char]

// 	testRule(t, `word:( 'a' 'b' 'c' )`, map[string]bool {
// 	"abc") == { word: "abc" : true,

// 	testRule(t, `( word:[abc]+ )?`, map[string]bool {
// 	"abc") == { word: "abc" : true,
// 	"") == {: true,

// 	testRule(t, `'a' outer:( inner:. ) 'c' / 'def'`, map[string]bool {
// 	"abc") == { outer: { inner: "b" } : true,
// }

// func TestNestedLabel(t *testing.T) {
// 	testRule(t, `word:( 'a' char:. 'c' )`, map[string]bool {
// 	"abc") == { word: { char: "b" } : true,
// }

// func TestAtLabel(t *testing.T) {
// 	testRule(t, `'a' @:. 'c'`, map[string]bool {
// 	"abc") == "b: true,

// 	//     grammar = JetPEG::Compiler.compile_grammar "
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
// 	testRule(t, `( char:'a' x:'x' / 'b' x:'x' / char:( inner:'c' ) x:'x' ) / 'y'`, map[string]bool {
// 	"ax") == { char: "a", x: "x" : true,
// 	"bx") == { x: "x" : true,
// 	"cx") == { char: { inner: "c" }, x: "x" : true,
// }

// func TestRuleWithLabel(t *testing.T) {
// 	//     grammar = JetPEG::Compiler.compile_grammar "
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
// 	//     grammar = JetPEG::Compiler.compile_grammar "
// 	//       rule test
// 	//         '(' inner:( test ( other:'b' )? ) ')' / char:'a'
// 	//       end
// 	//     "
// 	//     assert grammar.parse_rule(:test, "((a)b)") == { inner: { inner: { char: "a" }, other: "b"} }

// 	//     grammar = JetPEG::Compiler.compile_grammar "
// 	//       rule test
// 	//         '(' test ')' / char:'a'
// 	//       end
// 	//     "
// 	//     assert grammar.parse_rule(:test, "((a))") == { char: "a" }

// 	//     grammar = JetPEG::Compiler.compile_grammar "
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
// 	testRule(t, `list:( char:( 'a' / 'b' / 'c' ) )*`, map[string]bool {
// 	"abc") == { list: [{ char: "a" }, { char: "b" }, { char: "c" }] : true,

// 	testRule(t, `list:( char:'a' / char:'b' / 'c' )+`, map[string]bool {
// 	"abc") == { list: [{ char: "a" }, { char: "b" }, {}] : true,

// 	testRule(t, `( 'a' / 'b' / 'c' )+`, map[string]bool {
// 	"abc") == {: true,

// 	testRule(t, `list:( 'a' char:. )*->( 'ada' final:. )`, map[string]bool {
// 	"abacadae") == { list: [{ char: "b" }, { char: "c" }, { final: "e" }] : true,

// 	//     grammar = JetPEG::Compiler.compile_grammar "
// 	//       rule test
// 	//         ( char1:'a' inner:test / 'b' )*
// 	//       end
// 	//     "
// 	//     assert grammar.parse_rule(:test, "ab")
// }

// func TestObjectCreator(t *testing.T) {
// 	testRule(t, `'a' char:. 'c' <TestClassA> / 'd' char:. 'f' <TestClassB>", class_scope: self.clas`, map[string]bool {
// 	"abc") == TestClassA.new({ char: "b" }: true,
// 	"def") == TestClassB.new({ char: "e" }: true,

// 	testRule(t, `'a' char:. 'c' <TestClassA { a: 'test1', b: [ <TestClassB true>, <TestClassB { r: @char }> ] }>", class_scope: self.clas`, map[string]bool {
// 	"abc") == TestClassA.new({ a: "test1", b: [ TestClassB.new(true), TestClassB.new({ r: "b" }) ] }: true,
// }

// func TestValueCreator(t *testing.T) {
// 	testRule(t, `, map[string]bool {
// 	//       'a' char:. 'c' { @char.upcase } /
// 	//       word:'def' { @word.chars.map { |c| c.ord } } /
// 	//       'ghi' { [__FILE__, __LINE__] }
// 	//     ", filename: "test.jetpeg"
// 	"abc") == "B: true,
// 	"def") == ["d".ord, "e".ord, "f".ord: true,
// 	"ghi") == ["test.jetpeg", 4: true,
// }

// func TestLocalLabel(t *testing.T) {
// 	testRule(t, `'a' %temp:( char:'b' )* 'c' ( result:%temp )`, map[string]bool {
// 	"abc") == { result: [{ char: "b" }] : true,
// 	"abX") == ni: true,

// 	testRule(t, `'a' %temp:( char:'b' )* 'c' result1:%temp result2:%temp`, map[string]bool {
// 	"abc") == { result1: [{ char: "b" }], result2: [{ char: "b" }] : true,
// }

// func TestParameters(t *testing.T) {
// 	//     grammar = JetPEG::Compiler.compile_grammar "
// 	//       rule test
// 	//         %a:. %b:. test2[%a, %b, $true]
// 	//       end
// 	//       rule test2[%v, %w, %x]
// 	//         result1:%v result2:%w result3:%x
// 	//       end
// 	//     "
// 	//     assert grammar.parse_rule(:test, "ab") == { result1: "a", result2: "b", result3: true }
// }

// func TestUndefinedLocalLabelError(t *testing.T) {
// 	//     assert_raise JetPEG::CompilationError do
// 	testRule(t, `char:%missing`, map[string]bool {
// 	//       rule.parse "abc"
// 	//     end
// }

// func TestLeftRecursionHandling(t *testing.T) {
// 	//     grammar = JetPEG::Compiler.compile_grammar "
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

func testRule(t *testing.T, rule string, inputs map[string]bool) {
	fset := token.NewFileSet()
	file := CompileRule(rule, fset)

	if false {
		printer.Fprint(os.Stdout, fset, file)
	}

	config := loader.Config{
		Fset:  fset,
		Build: &build.Default,
		TypeChecker: types.Config{
			Packages: make(map[string]*types.Package),
			Sizes:    &types.StdSizes{8, 8},
			Error: func(err error) {
				t.Error(err)
			},
		},
	}
	config.CreateFromFiles("main", file)
	config.SourceImports = true
	iprog, err := config.Load()
	if err != nil {
		t.Error(err)
	}

	prog := ssa.Create(iprog, ssa.SanityCheckFunctions)
	prog.BuildAll()

	for input, matchExpected := range inputs {
		exitCode := interp.Interpret(prog.Package(iprog.Created[0].Pkg), 0, config.TypeChecker.Sizes, "main.go", []string{input})
		switch exitCode {
		case 100:
			if !matchExpected {
				t.Errorf("rule %q did match %q", rule, input)
			}
		case 101:
			if matchExpected {
				t.Errorf("rule %q did not match %q", rule, input)
			}
		default:
			t.Errorf("unexpected exit code: %d", exitCode)
		}
	}
}

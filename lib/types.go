package peg

import (
	"github.com/neelance/jetpeg"
)

type Rule struct {
	// RuleName   jetpeg.Stringer
	// Parameters []interface{}
	Child ParsingExpression
}

type ParsingExpression interface{}

type EmptyParsingExpression struct{}

type StringTerminal struct {
	Chars jetpeg.Stringer
	Fold  bool
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

type Sequence struct {
	Children []interface{}
}

type Choice struct {
	Children []interface{}
}

type Repetition struct {
	Child          ParsingExpression
	GlueExpression ParsingExpression
	AtLeastOnce    bool
}

type Until struct {
	Child           ParsingExpression
	UntilExpression ParsingExpression
}

type PositiveLookahead struct {
	Child ParsingExpression
}

type NegativeLookahead struct {
	Child ParsingExpression
}

type RuleCall struct {
	Name      jetpeg.Stringer
	Arguments []interface{}
}

type ParenthesizedExpression struct {
	Child ParsingExpression
}

type Label struct {
	Name    jetpeg.Stringer
	IsLocal bool
	Child   ParsingExpression
}

type LocalValue struct {
	Name jetpeg.Stringer
}

type ObjectCreator struct {
	Child     ParsingExpression
	ClassName jetpeg.Stringer
	Data      interface{}
}

type ValueCreator struct {
	Child ParsingExpression
	Code  jetpeg.Stringer
}

type TrueFunction struct {
}

type FalseFunction struct {
}

type MatchFunction struct {
	Value interface{}
}

type ErrorFunction struct {
	Msg jetpeg.Stringer
}

type EnterModeFunction struct {
	Name  jetpeg.Stringer
	Child ParsingExpression
}

type LeaveModeFunction struct {
	Name  jetpeg.Stringer
	Child ParsingExpression
}

type InModeFunction struct {
	Name jetpeg.Stringer
}

type StringValue struct {
	String jetpeg.Stringer
}

type StringData struct {
	String jetpeg.Stringer
}

type BooleanData struct {
	Value bool
}

type HashData struct {
	Entries []interface{}
}

type HashDataEntry struct {
	Label jetpeg.Stringer
	Data  interface{}
}

type ArrayData struct {
	Entries []interface{}
}

type ArrayDataEntry struct {
	Data interface{}
}

type ObjectData struct {
	ClassName jetpeg.Stringer
	Data      interface{}
}

type LabelData struct {
	Name jetpeg.Stringer
}

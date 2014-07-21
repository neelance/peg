package peg

import (
	"github.com/neelance/jetpeg"
)

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

package peg

import (
	"github.com/neelance/jetpeg"
)

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

package peg

import (
	"github.com/neelance/jetpeg"
)

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

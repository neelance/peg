package pegruntime

import (
	"bytes"
	"fmt"
	"os"
	"strings"
)

type Stringer interface {
	String() string
}

type InputRange struct {
	Input []byte
	Start int
	End   int
}

func (r *InputRange) Bytes() []byte {
	return r.Input[r.Start:r.End]
}

func (r *InputRange) String() string {
	return string(r.Bytes())
}

type StringData string

func (s StringData) String() string {
	return string(s)
}

type ParsingError struct {
	Input        []byte
	Position     int
	Expectations []string
	OtherReasons []string
}

func (e *ParsingError) Error() string {
	before := e.Input[:e.Position]
	line := bytes.Count(before, []byte{'\n'}) + 1
	column := len(before) - bytes.LastIndex(before, []byte{'\n'})
	if column == len(before)+1 {
		column = len(before)
	}

	reasons := e.OtherReasons
	if len(e.Expectations) != 0 {
		reasons = append(reasons, "expected one of "+strings.Join(e.Expectations, ", "))
	}
	prefixOffset := len(before) - 20
	if prefixOffset < 0 {
		prefixOffset = 0
	}
	return fmt.Sprintf("at line %d, column %d (byte %d, after %q): %s", line, column, e.Position, string(before[prefixOffset:]), strings.Join(reasons, " / "))
}

// callbacks rely on global variables since Go has no easy way of passing closures to C and the JetPEG backend has no support for passing the parser object yet
var Debug = false
var Factory = func(class string, value interface{}) interface{} { return value }
var input []byte
var inputOffset uintptr
var outputStack []interface{}
var localsStack []interface{}
var tempSource map[string]interface{}
var failurePosition int
var failureExpectations []string
var failureOtherReasons []string

func Test(rule func([]byte) []byte) {
	inputAtEnd := rule(append([]byte(os.Args[1]), 0))
	if len(inputAtEnd) != 1 || inputAtEnd[0] != 0 {
		fmt.Println("false")
		return
	}
	fmt.Println("true")
}

func HasPrefix(input []byte, prefix string) bool {
	return len(input) >= len(prefix) && bytes.Equal(input[:len(prefix)], []byte(prefix))
}

func HasPrefixFold(input []byte, prefix string) bool {
	return len(input) >= len(prefix) && bytes.EqualFold(input[:len(prefix)], []byte(prefix))
}

func ContainsByte(s string, b byte) bool {
	return strings.IndexByte(s, b) != -1
}

func pushOutput(v interface{}) {
	outputStack = append(outputStack, v)
}

func popOutput() interface{} {
	v := outputStack[len(outputStack)-1]
	outputStack = outputStack[:len(outputStack)-1]
	return v
}

func PushEmpty() {
	if Debug {
		fmt.Printf("pushEmpty()\n")
	}
	outputStack = append(outputStack, make(map[string]interface{}))
}

func PushInputRange(from uintptr, to uintptr) {
	if Debug {
		fmt.Printf("pushInputRange(%d, %d)\n", from-inputOffset, to-inputOffset)
	}
	pushOutput(&InputRange{input, int(from - inputOffset), int(to - inputOffset)})
}

func PushBoolean(value bool) {
	if Debug {
		fmt.Printf("pushBoolean(%t)\n", value)
	}
	pushOutput(value)
}

func PushString(value string) {
	if Debug {
		fmt.Printf("pushString(%q)\n", value)
	}
	pushOutput(StringData(value))
}

func PushArray(appendCurrent bool) {
	if Debug {
		fmt.Printf("pushArray(%t)\n", appendCurrent)
	}
	if appendCurrent {
		pushOutput([]interface{}{popOutput()})
		return
	}
	pushOutput([]interface{}{})
}

func AppendToArray() {
	if Debug {
		fmt.Printf("appendToArray()\n")
	}
	v := popOutput()
	pushOutput(append(popOutput().([]interface{}), v))
}

func MakeLabel(name string) {
	if Debug {
		fmt.Printf("makeLabel(%q)\n", name)
	}
	pushOutput(map[string]interface{}{name: popOutput()})
}

func MergeLabels(count int) {
	if Debug {
		fmt.Printf("mergeLabels(%d)\n", count)
	}
	merged := make(map[string]interface{})
	for i := 0; i < int(count); i++ {
		if m, ok := popOutput().(map[string]interface{}); ok {
			for k, v := range m {
				merged[k] = v
			}
		}
	}
	pushOutput(merged)
}

func MakeValue(code string, filename string, line int) {
	if Debug {
		fmt.Printf("makeValue(%q, %q, %d)\n", code, filename, line)
	}
	panic("makeValue not supported")
}

func MakeObject(class string) {
	if Debug {
		fmt.Printf("makeObject(%q)\n", class)
	}
	pushOutput(Factory(class, popOutput()))
}

func Pop() {
	if Debug {
		fmt.Printf("pop()\n")
	}
	popOutput()
}

func LocalsPush(count int) {
	if Debug {
		fmt.Printf("localsPush(%d)\n", count)
	}
	for i := 0; i < int(count); i++ {
		localsStack = append(localsStack, popOutput())
	}
}

func LocalsLoad(index int) {
	if Debug {
		fmt.Printf("localsLoad(%d)\n", index)
	}
	pushOutput(localsStack[len(localsStack)-1-int(index)])
}

func LocalsPop(count int) {
	if Debug {
		fmt.Printf("localsPop(%d)\n", count)
	}
	localsStack = localsStack[:len(localsStack)-int(count)]
}

func Match(absPos uintptr) uintptr {
	pos := absPos - inputOffset
	if Debug {
		fmt.Printf("match(%d)\n", pos)
	}
	var expected []byte
	switch e := popOutput().(type) {
	case *InputRange:
		expected = e.Bytes()
	case StringData:
		expected = []byte(e)
	default:
		panic("invalid type for match")
	}
	if bytes.HasPrefix(input[pos:], expected) {
		return absPos + uintptr(len(expected))
	}
	return 0
}

func SetAsSource() {
	if Debug {
		fmt.Printf("setAsSource()\n")
	}
	tempSource, _ = popOutput().(map[string]interface{})
}

func ReadFromSource(name string) {
	if Debug {
		fmt.Printf("readFromSource(%q)\n", name)
	}
	pushOutput(tempSource[name])
}

func TraceEnter(name string) {
	if Debug {
		fmt.Printf("traceEnter(%q)\n", name)
	}
}

func TraceLeave(name string, successful bool) {
	if Debug {
		fmt.Printf("traceLeave(%q, %t)\n", name, successful)
	}
}

func TraceFailure(absPos uintptr, reason string, isExpectation bool) {
	pos := int(absPos - inputOffset)
	if Debug {
		fmt.Printf("traceFailure(%d, %q, %t  )\n", pos, reason, isExpectation)
	}
	if pos > failurePosition {
		failurePosition = pos
		failureExpectations = nil
		failureOtherReasons = nil
	}
	if pos == failurePosition {
		switch isExpectation {
		case true:
			failureExpectations = append(failureExpectations, reason)
		case false:
			failureOtherReasons = append(failureOtherReasons, reason)
		}
	}
}

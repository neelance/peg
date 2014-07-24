package peglib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Stringer interface {
	String() string
}

type InputRange []byte

func (r InputRange) String() string {
	return string(r)
}

func (r InputRange) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(r))
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

var Debug = false
var Factory = func(class string, value interface{}) interface{} { return value }
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
		fmt.Println("null")
		return
	}
	if len(outputStack) == 0 {
		PushEmpty()
	}
	if len(outputStack) != 1 {
		panic("len(outputStack) != 1")
	}
	if err := json.NewEncoder(os.Stdout).Encode(outputStack[0]); err != nil {
		panic(err)
	}
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
		fmt.Printf("PushEmpty()\n")
	}
	pushOutput(make(map[string]interface{}))
}

func PushInputRange(startInput, endInput []byte) {
	if Debug {
		fmt.Printf("PushInputRange(...)\n")
	}
	pushOutput(InputRange(startInput[:len(startInput)-len(endInput)]))
}

func PushTrue() {
	if Debug {
		fmt.Printf("PushTrue()\n")
	}
	pushOutput(true)
}

func PushFalse() {
	if Debug {
		fmt.Printf("PushFalse()\n")
	}
	pushOutput(false)
}

func PushString(value string) {
	if Debug {
		fmt.Printf("PushString(%q)\n", value)
	}
	pushOutput(StringData(value))
}

func PushArray() {
	if Debug {
		fmt.Printf("PushArray()\n")
	}
	pushOutput([]interface{}{})
}

func AppendToArray() {
	if Debug {
		fmt.Printf("AppendToArray()\n")
	}
	v := popOutput()
	pushOutput(append(popOutput().([]interface{}), v))
}

func MakeLabel(name string) {
	if Debug {
		fmt.Printf("MakeLabel(%q)\n", name)
	}
	pushOutput(map[string]interface{}{name: popOutput()})
}

func MergeLabels(count int) {
	if Debug {
		fmt.Printf("MergeLabels(%d)\n", count)
	}
	merged := make(map[string]interface{})
	for i := 0; i < count; i++ {
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
		fmt.Printf("MakeValue(%q, %q, %d)\n", code, filename, line)
	}
	panic("makeValue not supported")
}

func MakeObject(class string) {
	if Debug {
		fmt.Printf("MakeObject(%q)\n", class)
	}
	pushOutput(Factory(class, popOutput()))
}

func Pop(count int) {
	if Debug {
		fmt.Printf("Pop(%d)\n", count)
	}
	for i := 0; i < count; i++ {
		popOutput()
	}
}

func LocalsPush(count int) {
	if Debug {
		fmt.Printf("LocalsPush(%d)\n", count)
	}
	for i := 0; i < count; i++ {
		localsStack = append(localsStack, popOutput())
	}
}

func LocalsLoad(index int) {
	if Debug {
		fmt.Printf("LocalsLoad(%d)\n", index)
	}
	pushOutput(localsStack[len(localsStack)-1-int(index)])
}

func LocalsPop(count int) {
	if Debug {
		fmt.Printf("LocalsPop(%d)\n", count)
	}
	localsStack = localsStack[:len(localsStack)-count]
}

// func Match(absPos uintptr) uintptr {
// 	pos := absPos - inputOffset
// 	if Debug {
// 		fmt.Printf("Match(%d)\n", pos)
// 	}
// 	var expected []byte
// 	switch e := popOutput().(type) {
// 	case InputRange:
// 		expected = e.Bytes()
// 	case StringData:
// 		expected = []byte(e)
// 	default:
// 		panic("invalid type for match")
// 	}
// 	if bytes.HasPrefix(input[pos:], expected) {
// 		return absPos + uintptr(len(expected))
// 	}
// 	return 0
// }

func SetAsSource() {
	if Debug {
		fmt.Printf("SetAsSource()\n")
	}
	tempSource, _ = popOutput().(map[string]interface{})
}

func ReadFromSource(name string) {
	if Debug {
		fmt.Printf("ReadFromSource(%q)\n", name)
	}
	pushOutput(tempSource[name])
}

func TraceEnter(name string) {
	if Debug {
		fmt.Printf("TraceEnter(%q)\n", name)
	}
}

func TraceLeave(name string, successful bool) {
	if Debug {
		fmt.Printf("TraceLeave(%q, %t)\n", name, successful)
	}
}

func TraceFailure(absPos uintptr, reason string, isExpectation bool) {
	pos := int(absPos - inputOffset)
	if Debug {
		fmt.Printf("TraceFailure(%d, %q, %t  )\n", pos, reason, isExpectation)
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

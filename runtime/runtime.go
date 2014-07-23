package pegruntime

import (
	"bytes"
	"os"
	"strings"
)

func Main(rule func([]byte) []byte) {
	inputAtEnd := rule(append([]byte(os.Args[1]), 0))
	if len(inputAtEnd) != 1 || inputAtEnd[0] != 0 {
		os.Exit(101)
	}
	os.Exit(100)
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

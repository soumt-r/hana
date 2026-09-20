// Package conv holds value conversions the tree-walker and the bytecode VM
// must agree on exactly (the browser engines mirror them by hand).
package conv

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/soumt-r/hana/console"
	"github.com/soumt-r/hana/errs"
	"github.com/soumt-r/hana/typecheck"
)

var plainNumber = regexp.MustCompile(`^[+-]?(\d+(\.\d*)?|\.\d+)$`)

// ParseInput converts a line read for `'x'를 [타입]으로 입력받자` into the value it
// binds. An empty typeName means the default, text. The number and boolean
// forms are strict (surrounding spaces are ignored, nothing else is
// accepted) so JS-style leniencies like "" -> 0 or "0x10" -> 16 never
// differ between engines.
func ParseInput(typeName string, names typecheck.Names, text, trueWord, falseWord string) (interface{}, error) {
	switch typeName {
	case "", names.String:
		return text, nil
	case names.Number:
		trimmed := strings.TrimSpace(text)
		if !plainNumber.MatchString(trimmed) {
			return nil, errs.New(errs.InputToNumberFailed, text)
		}
		n, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return nil, errs.New(errs.InputToNumberFailed, text)
		}
		return n, nil
	case names.Boolean:
		switch strings.TrimSpace(text) {
		case trueWord:
			return true, nil
		case falseWord:
			return false, nil
		}
		return nil, errs.New(errs.InputToBooleanFailed, text)
	}
	return nil, errs.New(errs.InputTypeUnsupported, typeName)
}

// NewLineReader returns a function that yields the next line of r without its
// line ending, and "" once r is exhausted (a closed stdin never blocks).
func NewLineReader(r io.Reader) func() string {
	br := bufio.NewReader(r)
	return func() string {
		console.Flush() // a prompt printed just before must be visible before we wait
		line, _ := br.ReadString('\n')
		return strings.TrimRight(line, "\r\n")
	}
}

// RuneAt is the idx-th character (0-based) of s, counted in code points like the
// language's `1번째`; ok is false when s has no such character. It walks to the
// character instead of converting the whole string.
func RuneAt(s string, idx int) (char string, ok bool) {
	if idx < 0 {
		return "", false
	}
	i := 0
	for _, r := range s {
		if i == idx {
			return string(r), true
		}
		i++
	}
	return "", false
}

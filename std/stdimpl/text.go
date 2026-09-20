package stdimpl

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/soumt-r/hana/errs"
)

// maxResultRunes bounds what pad/repeat/range may build.
const maxResultRunes = 1_000_000

func textFunc1(f func(string) string) Func {
	return func(args []interface{}) (interface{}, error) {
		if err := exactly(args, 1); err != nil {
			return nil, err
		}
		s, err := stringArg(args, 0)
		if err != nil {
			return nil, err
		}
		return f(s), nil
	}
}

var textUpper = textFunc1(func(s string) string { return strings.Map(unicode.ToUpper, s) })
var textLower = textFunc1(func(s string) string { return strings.Map(unicode.ToLower, s) })

// textStrip removes spaces, tabs, line breaks, no-break spaces and the
// ideographic space (U+3000) from both ends.
var textStrip = textFunc1(func(s string) string { return strings.Trim(s, " \t\n\v\f\r 　") })

var textReverse = textFunc1(func(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
})

// padArgs reads (text, width, fill) and returns the filler needed to reach width.
func padArgs(args []interface{}) (text string, filler string, err error) {
	if err = exactly(args, 3); err != nil {
		return
	}
	if text, err = stringArg(args, 0); err != nil {
		return
	}
	width, err := integerArg(args, 1)
	if err != nil {
		return
	}
	fill, err := stringArg(args, 2)
	if err != nil {
		return
	}
	if utf8.RuneCountInString(fill) != 1 {
		return "", "", errs.New(errs.PadFillLength)
	}
	if width < 0 || width > maxResultRunes {
		return "", "", errs.New(errs.ResultTooLarge)
	}
	missing := int(width) - utf8.RuneCountInString(text)
	if missing < 0 {
		missing = 0
	}
	return text, strings.Repeat(fill, missing), nil
}

func textPadLeft(args []interface{}) (interface{}, error) {
	text, filler, err := padArgs(args)
	if err != nil {
		return nil, err
	}
	return filler + text, nil
}

func textPadRight(args []interface{}) (interface{}, error) {
	text, filler, err := padArgs(args)
	if err != nil {
		return nil, err
	}
	return text + filler, nil
}

func textRepeat(args []interface{}) (interface{}, error) {
	if err := exactly(args, 2); err != nil {
		return nil, err
	}
	text, err := stringArg(args, 0)
	if err != nil {
		return nil, err
	}
	n, err := integerArg(args, 1)
	if err != nil {
		return nil, err
	}
	if n < 0 {
		return nil, errs.New(errs.NativeArgInteger, 2)
	}
	if int64(utf8.RuneCountInString(text))*n > maxResultRunes {
		return nil, errs.New(errs.ResultTooLarge)
	}
	return strings.Repeat(text, int(n)), nil
}

func twoStrings(args []interface{}) (string, string, error) {
	if err := exactly(args, 2); err != nil {
		return "", "", err
	}
	a, err := stringArg(args, 0)
	if err != nil {
		return "", "", err
	}
	b, err := stringArg(args, 1)
	return a, b, err
}

func textStartsWith(args []interface{}) (interface{}, error) {
	text, part, err := twoStrings(args)
	if err != nil {
		return nil, err
	}
	return strings.HasPrefix(text, part), nil
}

func textEndsWith(args []interface{}) (interface{}, error) {
	text, part, err := twoStrings(args)
	if err != nil {
		return nil, err
	}
	return strings.HasSuffix(text, part), nil
}

// textJoin glues a list of strings together with a separator between them.
func textJoin(args []interface{}) (interface{}, error) {
	if err := exactly(args, 2); err != nil {
		return nil, err
	}
	list, err := listArg(args, 0)
	if err != nil {
		return nil, err
	}
	sep, err := stringArg(args, 1)
	if err != nil {
		return nil, err
	}
	parts := make([]string, len(list))
	for i, v := range list {
		s, ok := v.(string)
		if !ok {
			return nil, errs.New(errs.NativeListStrings, 1)
		}
		parts[i] = s
	}
	return strings.Join(parts, sep), nil
}

// textCount is how many times a part occurs, without overlapping.
func textCount(args []interface{}) (interface{}, error) {
	text, part, err := twoStrings(args)
	if err != nil {
		return nil, err
	}
	if part == "" {
		return nil, errs.New(errs.TextEmptyPart)
	}
	return float64(strings.Count(text, part)), nil
}

// textFind is the 1-based position of the first occurrence, or 비어있음.
func textFind(args []interface{}) (interface{}, error) {
	text, part, err := twoStrings(args)
	if err != nil {
		return nil, err
	}
	if part == "" {
		return nil, errs.New(errs.TextEmptyPart)
	}
	idx := strings.Index(text, part)
	if idx < 0 {
		return nil, nil
	}
	return float64(utf8.RuneCountInString(text[:idx]) + 1), nil
}
